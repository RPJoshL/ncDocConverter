package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// The root nextcloud user where the files are stored
// and the files for onlyoffice jobs are defined
type NextcloudUser struct {
	NextcloudBaseUrl string `json:"nextcloudUrl"`
	Username         string `json:"username"`
	Password         string `json:"password"`

	// OnlyOffice
	ConvertJobs []NcConvertJob `json:"jobs"`

	// BookStack
	BookStack BookStack `json:"bookStack"`
}

// A OnlyOffice docs convert job
type NcConvertJob struct {
	JobName        string `json:"jobName"`
	SourceDir      string `json:"sourceDir"`
	DestinationDir string `json:"destinationDir"`
	KeepFolders    string `json:"keepFolders"`
	Recursive      string `json:"recursive"`
	Execution      string `json:"execution"`
}

type NcConvertUsers struct {
	Users []NextcloudUser `json:"nextcloudUsers"`
}

// Parses the given file to the in memory struct
func ParseConvertUsers(filePath string) (*NcConvertUsers, error) {

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the file '%s': %s", filePath, err)
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'ncConverter.json': %s", err)
	}

	var conv NcConvertUsers

	json.Unmarshal(byteValue, &conv)

	return &conv, nil
}
