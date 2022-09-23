package ncworker

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/pkg/logger"
	"rpjosh.de/ncDocConverter/web"
) 

type convertJob struct {
	job		*models.ConvertJob
	user	*models.User
}

type searchResult struct {
	XMLName  xml.Name `xml:"multistatus"`
	Text     string   `xml:",chardata"`
	D        string   `xml:"d,attr"`
	S        string   `xml:"s,attr"`
	Oc       string   `xml:"oc,attr"`
	Nc       string   `xml:"nc,attr"`
	Response []struct {
		Text     string `xml:",chardata"`
		Href     string `xml:"href"`
		Propstat struct {
			Text string `xml:",chardata"`
			Prop struct {
				Text            string `xml:",chardata"`
				Getcontenttype  string `xml:"getcontenttype"`
				Getlastmodified string `xml:"getlastmodified"`
				Size            string `xml:"size"`
				Fileid          int `xml:"fileid"`
			} `xml:"prop"`
			Status string `xml:"status"`
		} `xml:"propstat"`
	} `xml:"response"`
} 

type ncFiles struct {
	extension		string
	path			string
	lastModified	time.Time
	contentType		string
	size			int
	fileid			int
}

type searchTemplateData struct {
	Username	string
	Directory	string
	ContentType []string
}

func NewJob(job *models.ConvertJob, user *models.User) *convertJob {
	convJob := &convertJob{
		job: job,
		user: user,

	}

	return convJob
}

func (job *convertJob) ExecuteJob() {
	source := job.searchInDirectory(
		job.job.SourceDir, 
		[]string { 
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/msword",
		},
	)
	destination := job.searchInDirectory(
		job.job.DestinationDir,
		[]string {
			"application/pdf",
		},
	)


	preCount := len("/remote.php/dav/files/" + job.user.Username + "/")
	// store the files in a map
	sourceMap := make(map[string]ncFiles)
	destinationMap := make(map[string]ncFiles)

	for _, file := range source.Response {
		path := file.Href[preCount:]
		var extension = filepath.Ext(path)
		var name = path[0:len(path)-len(extension)][len(job.job.SourceDir):]
		// Time format: Fri, 23 Sep 2022 05:46:31 GMT
		time, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", file.Propstat.Prop.Getlastmodified)
		if err != nil {
			logger.Error("%s", err)
		}
		size, err := strconv.Atoi(file.Propstat.Prop.Size)
		if err != nil {
			logger.Error("%s", err)
		}
		sourceMap[name] = ncFiles{
			extension: extension,
			path: path,
			lastModified: time,
			size: size,
			contentType: file.Propstat.Prop.Getcontenttype,
			fileid: file.Propstat.Prop.Fileid,
		}
	}

	for _, file := range destination.Response {
		path := file.Href[preCount:]
		var extension = filepath.Ext(path)
		var name = path[0:len(path)-len(extension)][len(job.job.DestinationDir):]

		time, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", file.Propstat.Prop.Getlastmodified)
		if err != nil {
			logger.Error("%s", err)
		}
		size, err := strconv.Atoi(file.Propstat.Prop.Size)
		if err != nil {
			logger.Error("%s", err)
		}
		destinationMap[name] = ncFiles{
			extension: extension,
			path: path,
			lastModified: time,
			size: size,
			contentType: file.Propstat.Prop.Getcontenttype,
			fileid: file.Propstat.Prop.Fileid,
		}
	}

	for index, source := range sourceMap {
		// check if the file exists in the destination map
		if dest, exists := destinationMap[index]; exists {
			// compare timestamp and size
			if dest.lastModified.Before(source.lastModified) {
				job.convertFile(source.path, source.fileid, dest.path)
			}
			delete(destinationMap, index)
		} else {
			job.convertFile(
				source.path, source.fileid, job.getDestinationDir(source.path),
			)
			delete(destinationMap, index)
		}
	}

	// delete the files which are not available anymore
	for _, dest := range destinationMap {
		job.deleteFile(dest.path)
	}
}

func (job *convertJob) getDestinationDir(sourceFile string) string {
	sourceFile = sourceFile[len(job.job.SourceDir):]
	var extension = filepath.Ext(sourceFile)
	var name = sourceFile[0:len(sourceFile)-len(extension)]

	return job.job.DestinationDir + name + ".pdf"
}

func (job *convertJob) createFoldersRecursively(destinationFile string) {
	s := strings.Split(destinationFile, "/")
	folderTree := ""

	logger.Debug("Creating directory for file '%s'", destinationFile)

	// webdav doesn't have an function to create directories recursively
	for _, folder := range s[:len(s) - 1] {
		folderTree += folder + "/"

		client := http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("MKCOL", job.user.NextcloudBaseUrl + "/remote.php/dav/files/" + job.user.Username + "/" + folderTree, nil)
		if err != nil {
			logger.Error("%s", err)
		}
		req.SetBasicAuth(job.user.Username, job.user.Password)

		res, err := client.Do(req)
		if err != nil {
			logger.Error("%s", err)
		}
		if (res.StatusCode != 201 && res.StatusCode != 405) {

		}
		// status code 201 or 405 (already existing)
	}
}

func (job *convertJob) convertFile(sourceFile string, sourceid int, destinationFile string) {
	logger.Debug("Trying to convert %s (%d) to %s", sourceFile, sourceid, destinationFile)

	job.createFoldersRecursively(destinationFile)

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, job.user.NextcloudBaseUrl + "/apps/onlyoffice/downloadas", nil)
    if err != nil {
        logger.Error("%s", err)
    }
    req.SetBasicAuth(job.user.Username, job.user.Password)

	q := req.URL.Query()
    q.Add("fileId", fmt.Sprint(sourceid))
    q.Add("toExtension", "pdf")
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
    if err != nil {
        logger.Error("%s", err)
    }
	// Status Code 200
    defer res.Body.Close()

	uploadClient := http.Client{Timeout: 10 * time.Second}
	uploadReq, err := http.NewRequest(http.MethodPut, job.user.NextcloudBaseUrl + "/remote.php/dav/files/" + job.user.Username + "/" + destinationFile, res.Body)

	if err != nil {
        logger.Error("%s", err)
    }
    uploadReq.SetBasicAuth(job.user.Username, job.user.Password)
	uploadReq.Header.Set("Content-Type", "application/binary")

	res, err = uploadClient.Do(uploadReq)
    if err != nil {
        logger.Error("%s", err)
    }

	if (res.StatusCode != 204 && res.StatusCode != 201) {
		logger.Error("Failed to create file %s (#%d)", destinationFile, res.StatusCode)
	}
	// Status Code 201
    res.Body.Close()
}

func (job *convertJob) deleteFile(filePath string) {
	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodDelete, job.user.NextcloudBaseUrl + "/remote.php/dav/files/" + job.user.Username + "/" + filePath, nil)
    if err != nil {
        logger.Error("%s", err)
    }
    req.SetBasicAuth(job.user.Username, job.user.Password)

    res, err := client.Do(req)
    if err != nil {
        logger.Error("%s", err)
    }
	
	if (res.StatusCode != 204) {
		logger.Error("Failed to delete file %s (%d)", filePath, res.StatusCode)
	}
}

// Searches all doc files in the source directory
func (job *convertJob) searchInDirectory(directory string, contentType []string) *searchResult {
	client := http.Client{Timeout: 5 * time.Second}

	template, err := template.ParseFS(web.ApiTemplateFiles, "apitemplate/ncsearch.tmpl.xml")
	if err != nil {
		logger.Error("%s", err)
	}
	var buf bytes.Buffer
	templateData := searchTemplateData{ 
		Username: job.user.Username, 
		Directory: directory,
		ContentType: contentType,
	}
	if err = template.Execute(&buf, templateData); err != nil {
		logger.Error("%s", err)
	}
	// Status code 207
    req, err := http.NewRequest("SEARCH", job.user.NextcloudBaseUrl + "/remote.php/dav/", &buf)
    if err != nil {
        logger.Error("%s", err)
    }
    req.SetBasicAuth(job.user.Username, job.user.Password)
	req.Header.Set("Content-Type", "application/xml")

    res, err := client.Do(req)
    if err != nil {
        logger.Error("%s", err)
    }

    defer res.Body.Close()

    resBody, err := io.ReadAll(res.Body)
    if err != nil {
        logger.Error("%s", err)
    }

	fmt.Print(res.StatusCode)
	var result searchResult
	if err = xml.Unmarshal(resBody, &result); err != nil {
		logger.Error("%s", err)
	}

	return &result
}