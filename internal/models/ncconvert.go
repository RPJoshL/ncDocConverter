package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type User struct {
	AuthUser			string`json:"authUser"`
	NextcloudBaseUrl	string`json:"nextcloudUrl"`
	Username			string`json:"username"`
	Password			string`json:"password"`
}

type ConvertJob struct {
	JobName				string`json:"jobName"`
	SourceDir			string`json:"sourceDir"`
	DestinationDir		string`json:"destinationDir"`
	KeepFolders			string`json:"keepFolders"`
	Recursive			string`json:"recursive"`
	Executions			[]string`json:"execution"`
}

type NcConverter struct {
	Users				[]User`json:"users"`
}

// Parses the given file to the in memory struct
func ParseUsers(filePath string) (*NcConverter, error) {

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the file '%s': %s", filePath, err)
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'ncConverter.json': %s", err)
	}

	var conv NcConverter

	json.Unmarshal(byteValue, &conv)

	return &conv, nil
}