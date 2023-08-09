package ncworker

// @TODO delete folders for shelves that doesn't exist anyore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
	"git.rpjosh.de/ncDocConverter/internal/models"
	"git.rpjosh.de/ncDocConverter/internal/nextcloud"
	"git.rpjosh.de/ncDocConverter/pkg/utils"
)

type BsJob struct {
	job    *models.BookStackJob
	ncUser *models.NextcloudUser

	cacheCount   int
	cacheBooks   map[int]book
	cacheShelves []shelf
	// If the cache should be usedi n the current execution
	useCache bool
}

type shelf struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	// This has to be fetched extra
	books []int
}
type shelfDetails struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	Books []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"books"`
}
type shelves struct {
	Data []shelf `json:"data"`
}

type book struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	// This has to be calculated of the latest modify page of a page
	lastModified time.Time
	// If the book should be ignored to convert
	ignore bool

	// If the book has been already converted
	converted bool
}
type books struct {
	Data []book `json:"data"`
}
type bookDetails struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Contents []struct {
		ID        int       `json:"id"`
		Name      string    `json:"name"`
		Slug      string    `json:"slug"`
		BookID    int       `json:"book_id"`
		ChapterID int       `json:"chapter_id"`
		Draft     bool      `json:"draft"`
		Template  bool      `json:"template"`
		UpdatedAt time.Time `json:"updated_at"`
		URL       string    `json:"url"`
		Type      string    `json:"type"`
	} `json:"contents"`
	Tags []string `json:"tags"`
}

func NewBsJob(job *models.BookStackJob, ncUser *models.NextcloudUser) *BsJob {
	bsJob := BsJob{
		job:    job,
		ncUser: ncUser,
	}

	return &bsJob
}

func (job *BsJob) ExecuteJob() {
	// Get all existing files in the destination folder
	destination, err := nextcloud.SearchInDirectory(
		job.ncUser, job.job.DestinationDir,
		[]string{
			"text/html",
			"application/pdf",
		},
	)
	if err != nil {
		logger.Error("Failed to get files in destination directory '%s': %s", job.job.DestinationDir, err)
		return
	}

	// Make a map with path as index
	prefix := "/remote.php/dav/files/" + job.ncUser.Username + "/"
	destinationMap := nextcloud.ParseSearchResult(destination, prefix, job.job.DestinationDir)

	// Check for cache
	job.cache()

	// Get all shelves
	shelves, err := job.getShelves()
	if err != nil {
		logger.Error("Failed to get shelves: %s", err)
		return
	}

	// Get all books
	books, err := job.getBooks()
	if err != nil {
		logger.Error("Failed to get books: %s", err)
		return
	}

	// Index books by path
	indexedBooks := job.getIndexedBooks(shelves, books)

	// Cache data
	if job.job.CacheCount > 0 && !job.useCache {
		job.cacheCount = job.job.CacheCount

		job.cacheShelves = *shelves
		job.cacheBooks = utils.CopyMap(*books)
	}

	// Now finally convert the books :)
	convertCount := 0
	var wg sync.WaitGroup
	for i, b := range indexedBooks {
		// mark as converted
		indexedBooks[i].converted = true
		(*books)[b.ID] = *indexedBooks[i]

		// check if it has to be converted again (updated) or for the first time
		des, exists := destinationMap[i]

		if (!exists || b.lastModified.After(des.LastModified)) && !b.ignore {
			wg.Add(1)
			convertCount++
			go func(book book, path string) {
				defer wg.Done()
				job.convertBook(book, path)
			}(*b, i)
		} else if b.ignore {
			logger.Debug("Duplicate book name: %s", b.Name)
		}

		// Ignore states that a book with a duplicate name exists → delete the orig also
		if !b.ignore {
			delete(destinationMap, i)
		}
	}
	wg.Wait()

	// Convert remaining books
	if job.job.IncludeBooksWithoutShelve {
		for _, b := range *books {
			// check if it has to be converted again (updated) or for the first time
			des, exists := destinationMap[b.Name]

			if !b.converted && !b.ignore && (!exists || b.lastModified.After(des.LastModified)) {
				wg.Add(1)
				convertCount++
				go func(book book, path string) {
					defer wg.Done()
					job.convertBook(book, path)
				}(b, b.Name)
			}
			delete(destinationMap, b.Name)
		}
		wg.Wait()
	}

	// Delete the files which are not available anymore
	for _, dest := range destinationMap {
		err := nextcloud.DeleteFile(job.ncUser, dest.Path)
		if err != nil {
			logger.Error(utils.FirstCharToUppercase(err.Error()))
		}
	}

	logger.Info("Finished BookStack job \"%s\": %d books converted", job.job.JobName, convertCount)
}

// Checks and initializes the cache
func (job *BsJob) cache() {
	if job.job.CacheCount > 0 {
		job.cacheCount--
		if job.cacheCount < 0 {
			job.useCache = false
		} else {
			job.useCache = true
		}
	}
}

// Return the relative path of the book to save in nextcloud
func (job *BsJob) getPath(bookName string, shelfName string) string {
	if job.job.KeepStructure {
		return shelfName + "/" + bookName
	} else {
		return bookName
	}
}

// Gets all shelves
func (job *BsJob) getShelves() (*[]shelf, error) {
	if job.useCache {
		return &job.cacheShelves, nil
	}

	client := http.Client{Timeout: 10 * time.Second}

	req := job.getRequest(http.MethodGet, "shelves", nil)

	// Add shelve filter
	q := req.URL.Query()
	for _, j := range job.job.Shelves {
		q.Add("filter[name:eq]", j)
	}
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected status code 200, got %d", res.StatusCode)
	}

	rtc := shelves{}
	if err = json.NewDecoder(res.Body).Decode(&rtc); err != nil {
		return nil, fmt.Errorf("failed to decode response: %s", err)
	}

	if job.job.ShelvesRegex != "" {
		reg, err := regexp.Compile(job.job.ShelvesRegex)
		// This is fatal
		logger.Fatal("Failed to parse the regex '%s': %s", job.job.ShelvesRegex, err)

		rtc2 := shelves{}

		for i, shelve := range rtc.Data {
			if reg.Match([]byte(shelve.Name)) {
				rtc2.Data = append(rtc2.Data, rtc.Data[i])
			} else {
				logger.Debug("Ignoring shelve %s", shelve.Name)
			}
		}

		rtc = rtc2
	}

	return &rtc.Data, nil
}

// Returns the IDs of books which belongs to the shelf
func (job *BsJob) getBooksInShelve(id int) ([]int, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req := job.getRequest(http.MethodGet, "shelves/"+fmt.Sprintf("%d", id), nil)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected status code 200, got %d", res.StatusCode)
	}

	shelfDetails := shelfDetails{}
	if err = json.NewDecoder(res.Body).Decode(&shelfDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %s", err)
	}

	rtc := make([]int, len(shelfDetails.Books))
	for i, details := range shelfDetails.Books {
		rtc[i] = details.ID
	}

	return rtc, nil
}

// Indexes the books by the relative path
func (job *BsJob) getIndexedBooks(shelves *[]shelf, books *map[int]book) map[string]*book {
	// Now it has to be checked which book belongs to which shelve.
	// When cached this was already done
	if !job.useCache {
		var wg sync.WaitGroup
		for i, shelv := range *shelves {
			wg.Add(1)

			go func(shelf shelf, index int) {
				defer wg.Done()

				ids, err := job.getBooksInShelve(shelf.ID)
				if err != nil {
					logger.Error("Failed to get shelf details: %s", err)
				} else {
					b := make([]int, 0)

					for _, id := range ids {
						// Check if book should be excluded → it is not contained in the book map
						book, exists := (*books)[id]
						if exists {
							b = append(b, book.ID)
						}
					}

					(*shelves)[index].books = b
				}
			}(shelv, i)
		}
		wg.Wait()
	}

	// A book can have the same name. This would lead to conflicts
	// if they are in the same shelve / folder.
	// In such a case the ID of the book will be appended to the name "bookName_123".
	// Because of that a map indexed by the path will be created and AFTERWARDS the file is converted
	indexedBooks := make(map[string]*book)
	for _, shelf := range *shelves {
		for _, bookId := range shelf.books {
			b := (*books)[bookId]
			bookPath := job.getPath(b.Name, shelf.Name)
			existingBook, doesExists := indexedBooks[bookPath]

			if doesExists {
				// The book path will be appended
				newBookPath := fmt.Sprintf("%s_%d", bookPath, b.ID)
				indexedBooks[newBookPath] = &b

				// Also add the other book with the ID
				otherNewBookPath := fmt.Sprintf("%s_%d", bookPath, existingBook.ID)
				indexedBooks[otherNewBookPath] = existingBook

				// The original book won't be removed because otherwise a third book with the same
				// name will be inserted using its real name.
				// But because this is a pointer, a copy is needed
				var existingBookCopy book
				utils.Copy(existingBook, &existingBookCopy)
				existingBookCopy.ignore = true
				indexedBooks[bookPath] = &existingBookCopy
			} else {
				indexedBooks[bookPath] = &b
			}
		}

		// If the structure should be keept, a folder for every shelve has to be created
		if job.job.KeepStructure && !job.useCache {
			nextcloud.CreateFoldersRecursively(job.ncUser, job.job.DestinationDir+shelf.Name+"/")
		}
	}

	return indexedBooks
}

// Gets all books and returns a map indexed by the ID of the book
func (job *BsJob) getBooks() (*map[int]book, error) {
	if job.useCache {
		books := utils.CopyMap(job.cacheBooks)

		// The last Change date has to be updated even in cache
		var wg sync.WaitGroup
		var mut = &sync.Mutex{}
		for i, b := range books {
			wg.Add(1)

			go func(book book, index int) {
				defer wg.Done()
				lastModified, err := job.getLastModifiedOfBook(book.ID)
				if err != nil {
					logger.Warning("Failed to get last modified date of book %s (%d) - using old date: %s", book.Name, book.ID, err)
					return
				}

				book.lastModified = *lastModified

				mut.Lock()
				books[index] = book
				mut.Unlock()
			}(b, i)
		}
		wg.Wait()

		return &books, nil
	}

	client := http.Client{Timeout: 10 * time.Second}
	req := job.getRequest(http.MethodGet, "books", nil)

	// Add shelve filter
	q := req.URL.Query()
	for _, j := range job.job.Books {
		q.Add("filter[name:eq]", j)
	}
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected status code 200, got %d", res.StatusCode)
	}

	booksArray := books{}
	if err = json.NewDecoder(res.Body).Decode(&booksArray); err != nil {
		return nil, fmt.Errorf("failed to decode response: %s", err)
	}

	if job.job.BooksRegex != "" {
		reg, err := regexp.Compile(job.job.BooksRegex)
		// This is fatal
		logger.Fatal("Failed to parse the regex '%s': %s", job.job.BooksRegex, err)

		booksArray2 := books{}

		for i, book := range booksArray.Data {
			if reg.Match([]byte(book.Name)) {
				booksArray2.Data = append(booksArray2.Data, booksArray.Data[i])
			} else {
				logger.Debug("Ignoring shelve %s", book.Name)
			}
		}

		booksArray = booksArray2
	}

	// Create indexed map
	rtc := make(map[int]book)
	var wg sync.WaitGroup
	var mut = &sync.Mutex{}
	for _, b := range booksArray.Data {
		wg.Add(1)

		go func(b book) {
			defer wg.Done()
			lastModified, err := job.getLastModifiedOfBook(b.ID)
			if err != nil {
				logger.Warning("Failed to get last modified date of book %s (%d) - skipping: %s", b.Name, b.ID, err)
				return
			}

			if lastModified.Unix() == 0 {
				logger.Info("Skipping book %s (%d) because of no content", b.Name, b.ID)
				return
			}

			mut.Lock()
			rtc[b.ID] = book{
				ID:           b.ID,
				Name:         b.Name,
				lastModified: *lastModified,
			}
			mut.Unlock()
		}(b)
	}
	wg.Wait()

	return &rtc, nil
}

// Returns the last modified time of a book
func (job *BsJob) getLastModifiedOfBook(id int) (*time.Time, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req := job.getRequest(http.MethodGet, "books/"+fmt.Sprintf("%d", id), nil)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("expected status code 200, got %d", res.StatusCode)
	}

	bd := bookDetails{}
	if err = json.NewDecoder(res.Body).Decode(&bd); err != nil {
		return nil, fmt.Errorf("failed to decode response: %s", err)
	}

	lastMod := time.Unix(0, 0)
	for i, content := range bd.Contents {
		if content.Template || content.Draft {
			continue
		}

		if content.UpdatedAt.After(lastMod) {
			lastMod = bd.Contents[i].UpdatedAt
		}
	}

	return &lastMod, nil
}

// Returns a new request to the bookStack API.
// The path beginning AFTER /api/ should be given (e.g.: shelves)
func (job *BsJob) getRequest(method string, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, job.ncUser.BookStack.URL+"/api/"+path, body)
	if err != nil {
		logger.Error("%s", err)
	}
	req.Header.Set("Authorization", "Token "+job.ncUser.BookStack.Token)

	return req
}

// Converts the given book and uploads it to nextcloud.
// The path is being expected relative to the root dir of the jobs directory and does
// not contain a file extension
func (job *BsJob) convertBook(book book, path string) {
	fileExtension, url := job.getFileExtension()

	client := http.Client{Timeout: 10 * time.Second}
	req := job.getRequest(http.MethodGet, fmt.Sprintf("books/%d/export/%s", book.ID, url), nil)

	res, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to convert book: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		logger.Error("Failed to convert book: expected status code 200, got %d", res.StatusCode)
		return
	}

	err = nextcloud.UploadFile(job.ncUser, job.job.DestinationDir+path+fileExtension, res.Body)
	if err != nil {
		logger.Error("Failed to upload book to nextcloud: %s", err)
	}
}

func (job *BsJob) getFileExtension() (fileExtension string, url string) {
	switch strings.ToLower(string(job.job.Format)) {
	case "html":
		{
			fileExtension = ".html"
			url = "html"
		}
	case "pdf":
		{
			fileExtension = ".pdf"
			url = "pdf"
		}
	default:
		{
			logger.Fatal("Invalid format given: '%s'. Expected 'html' or 'pdf'", job.job.Format)
		}
	}

	return
}
