package nextcloud

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/pkg/logger"
	"rpjosh.de/ncDocConverter/web"
)

// The internal representation of a nextcloud file
type NcFile struct {
	// File extension: txt
	Extension string
	// Relative path of the file to the nextcloud root: /folder/file.txt
	Path         string
	LastModified time.Time
	ContentType  string
	// Size in Bytes
	Size int
	// The unique file ID of the nextcloud server
	Fileid int
	// The Webdav URL for file reference
	WebdavURL string
}

type searchTemplateData struct {
	Username    string
	Directory   string
	ContentType []string
}

type searchResult struct {
	XMLName  xml.Name               `xml:"multistatus"`
	Text     string                 `xml:",chardata"`
	D        string                 `xml:"d,attr"`
	S        string                 `xml:"s,attr"`
	Oc       string                 `xml:"oc,attr"`
	Nc       string                 `xml:"nc,attr"`
	Response []searchResultResponse `xml:"response"`
}
type searchResultResponse struct {
	Text     string `xml:",chardata"`
	Href     string `xml:"href"`
	Propstat struct {
		Text string `xml:",chardata"`
		Prop struct {
			Text            string `xml:",chardata"`
			Getcontenttype  string `xml:"getcontenttype"`
			Getlastmodified string `xml:"getlastmodified"`
			Size            string `xml:"size"`
			Fileid          int    `xml:"fileid"`
		} `xml:"prop"`
		Status string `xml:"status"`
	} `xml:"propstat"`
}

func (r *searchResultResponse) GetLastModified() time.Time {
	// Time format: Fri, 23 Sep 2022 05:46:31 GMT
	rtc, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", r.Propstat.Prop.Getlastmodified)
	if err != nil {
		logger.Warning("%s", err)
		rtc = time.Unix(0, 1)
	}

	return rtc
}

// Returns a new request to the Nexcloud API.
// The path beginning AFTER /dav/ should be given (e.g.: myUser/folder/file.txt)
func getRequest(method string, path string, body io.Reader, ncUser *models.NextcloudUser) *http.Request {
	req, err := http.NewRequest(method, ncUser.NextcloudBaseUrl+"/remote.php/dav/"+path, body)
	if err != nil {
		logger.Error("%s", err)
	}
	req.SetBasicAuth(ncUser.Username, ncUser.Password)

	return req
}

// Searches for all files of the given content type starting in the given directory.
func SearchInDirectory(ncUser *models.NextcloudUser, directory string, contentType []string) (*searchResult, error) {
	client := http.Client{Timeout: 5 * time.Second}

	template, err := template.ParseFS(web.ApiTemplateFiles, "apitemplate/ncsearch.tmpl.xml")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	templateData := searchTemplateData{
		Username:    ncUser.Username,
		Directory:   directory,
		ContentType: contentType,
	}
	if err = template.Execute(&buf, templateData); err != nil {
		return nil, err
	}

	// Status code 207
	req := getRequest("SEARCH", "", &buf, ncUser)
	req.Header.Set("Content-Type", "application/xml")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Decody body first before checking status code to print in error message
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Create folder if not existing
	if res.StatusCode == 404 {
		logger.Info("Creating directory '%s' because it does not exist", "/"+directory)
		CreateFoldersRecursively(ncUser, "/"+directory+"notExisting.pdf")
		return &searchResult{}, nil
	}

	if res.StatusCode != 207 {
		return nil, fmt.Errorf("status code %d: %s", res.StatusCode, resBody)
	}

	var result searchResult
	if err = xml.Unmarshal(resBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Parses the response from the given search format to an NcFile.
// A map with the relative path based on the source Directory ("someFolder/file.txt")
// and the mathing NcFile will be returned. Therefore, also the source Directory has to be given.
//
// To determine the path without the prefix "/remote.php/dav/user/" it has to be given.
func ParseSearchResult(result *searchResult, prefix string, sourceDir string) map[string]NcFile {
	preCount := len(prefix)
	rtc := make(map[string]NcFile)

	for _, file := range result.Response {
		href, _ := url.QueryUnescape(file.Href)
		path := href[preCount:]
		var extension = filepath.Ext(path)
		var name = path[0 : len(path)-len(extension)][len(sourceDir):]
		time := file.GetLastModified()
		size, err := strconv.Atoi(file.Propstat.Prop.Size)
		if err != nil {
			logger.Error("Failed to parse the file size '%s' to an integer: %s", file.Propstat.Prop.Size, err)
			continue
		}
		rtc[name] = NcFile{
			Extension:    extension,
			Path:         path,
			LastModified: time,
			Size:         size,
			ContentType:  file.Propstat.Prop.Getcontenttype,
			Fileid:       file.Propstat.Prop.Fileid,
			WebdavURL:    file.Href,
		}
	}

	return rtc
}

// Delets a file with the given path.
// The path has to start at the root level: Ebook/myFolder/file.txt
func DeleteFile(ncUser *models.NextcloudUser, filePath string) error {
	client := http.Client{Timeout: 5 * time.Second}

	req := getRequest(http.MethodDelete, "files/"+ncUser.Username+"/"+filePath, nil, ncUser)

	res, err := client.Do(req)
	if err != nil {
		logger.Error("%s", err)
	}

	if res.StatusCode != 204 {
		return fmt.Errorf("failed to delete file %s (%d)", filePath, res.StatusCode)
	}

	return nil
}

// Creates all required directorys to create the destination file recursively.
// The path should be relative to the root: ebook/folder1/folder2/file.txt
func CreateFoldersRecursively(ncUser *models.NextcloudUser, destinationFile string) {
	s := strings.Split(destinationFile, "/")
	folderTree := ""

	// Webdav doesn't have a function to create directories recursively → iterate
	for _, folder := range s[:len(s)-1] {
		folderTree += folder + "/"

		client := http.Client{Timeout: 5 * time.Second}
		req := getRequest("MKCOL", "files/"+ncUser.Username+"/"+folderTree, nil, ncUser)

		res, err := client.Do(req)
		if err != nil {
			logger.Error("%s", err)
		}

		if res.StatusCode != 201 && res.StatusCode != 405 {
			logger.Error("Failed to create directory '%s'", folderTree)
		}
	}
}

// Uploads a file to the nextcloud server.
// It will be saved to the destination as a relative path to the nextcloud root (ebook/file.txt).
func UploadFile(ncUser *models.NextcloudUser, destination string, content io.ReadCloser) error {
	client := http.Client{Timeout: 5 * time.Second}
	req := getRequest(http.MethodPut, "files/"+ncUser.Username+"/"+destination, content, ncUser)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 201 && res.StatusCode != 204 {
		return fmt.Errorf("expected status code 201 or 204 but got %d", res.StatusCode)
	}

	return nil
}
