package models

// BookStack details to fetch books from
type BookStack struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Token    string `json:"apiToken"`

	Jobs []BookStackJob `json:"jobs"`
}

// A concrete BookStacksJob
type BookStackJob struct {
	JobName        string `json:"jobName"`
	DestinationDir string `json:"destinationDir"`

	Shelves      []string `json:"shelves"`
	ShelvesRegex string   `json:"shelveRegex"`

	Books      []string `json:"books"`
	BooksRegex string   `json:"booksRegex"`

	IncludeBooksWithoutShelve bool   `json:"includeBooksWithoutShelve"`
	Format                    Format `json:"format"`
	KeepStructure             bool   `json:"keepStructure"`

	Recursive string `json:"recursive"`
	Execution string `json:"execution"`

	CacheCount int `json:"cache"`
}

type Format string

const (
	HTML Format = "html"
	PDF  Format = "pdf"
)
