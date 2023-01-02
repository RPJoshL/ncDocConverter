package ncworker

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/internal/nextcloud"
	"rpjosh.de/ncDocConverter/pkg/logger"
	"rpjosh.de/ncDocConverter/pkg/utils"
)

type convertJob struct {
	job    *models.NcConvertJob
	ncUser *models.NextcloudUser
}

type ncFiles struct {
	extension    string
	path         string
	lastModified time.Time
	contentType  string
	size         int
	fileid       int
}

func NewNcJob(job *models.NcConvertJob, ncUser *models.NextcloudUser) *convertJob {
	convJob := &convertJob{
		job:    job,
		ncUser: ncUser,
	}

	return convJob
}

func (job *convertJob) ExecuteJob() {

	// Get existing directory contents
	source, err := nextcloud.SearchInDirectory(
		job.ncUser,
		job.job.SourceDir,
		[]string{
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/msword",
		},
	)
	if err != nil {
		logger.Error("Failed to get files in source directory '%s': %s", job.job.SourceDir, err)
		return
	}

	destination, err := nextcloud.SearchInDirectory(
		job.ncUser,
		job.job.DestinationDir,
		[]string{
			"application/pdf",
		},
	)
	if err != nil {
		logger.Error("Failed to get files in destination directory '%s': %s", job.job.DestinationDir, err)
		return
	}

	preCount := len("/remote.php/dav/files/" + job.ncUser.Username + "/")
	// Store the files in a map
	sourceMap := make(map[string]ncFiles)
	destinationMap := make(map[string]ncFiles)

	for _, file := range source.Response {
		href, _ := url.QueryUnescape(file.Href)
		path := href[preCount:]
		var extension = filepath.Ext(path)
		var name = path[0 : len(path)-len(extension)][len(job.job.SourceDir):]
		time := file.GetLastModified()
		size, err := strconv.Atoi(file.Propstat.Prop.Size)
		if err != nil {
			logger.Error("%s", err)
		}
		sourceMap[name] = ncFiles{
			extension:    extension,
			path:         path,
			lastModified: time,
			size:         size,
			contentType:  file.Propstat.Prop.Getcontenttype,
			fileid:       file.Propstat.Prop.Fileid,
		}
	}

	for _, file := range destination.Response {
		href, _ := url.QueryUnescape(file.Href)
		path := href[preCount:]
		var extension = filepath.Ext(path)
		var name = path[0 : len(path)-len(extension)][len(job.job.DestinationDir):]

		time, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", file.Propstat.Prop.Getlastmodified)
		if err != nil {
			logger.Error("%s", err)
		}
		size, err := strconv.Atoi(file.Propstat.Prop.Size)
		if err != nil {
			logger.Error("%s", err)
		}
		destinationMap[name] = ncFiles{
			extension:    extension,
			path:         path,
			lastModified: time,
			size:         size,
			contentType:  file.Propstat.Prop.Getcontenttype,
			fileid:       file.Propstat.Prop.Fileid,
		}
	}

	convertCount := 0
	for index, source := range sourceMap {
		// check if the file exists in the destination map
		if dest, exists := destinationMap[index]; exists {
			// compare timestamp and size
			if dest.lastModified.Before(source.lastModified) {
				job.convertFile(source.path, source.fileid, dest.path)
				convertCount++
			}
			delete(destinationMap, index)
		} else {
			job.convertFile(
				source.path, source.fileid, job.getDestinationDir(source.path),
			)
			convertCount++
			delete(destinationMap, index)
		}
	}

	// Delete the files which are not available anymore
	for _, dest := range destinationMap {
		err := nextcloud.DeleteFile(job.ncUser, dest.path)
		if err != nil {
			logger.Error(utils.FirstCharToUppercase(err.Error()))
		}
	}

	logger.Info("Finished Nextcloud job \"%s\": %d documents converted", job.job.JobName, convertCount)
}

func (job *convertJob) getDestinationDir(sourceFile string) string {
	sourceFile = sourceFile[len(job.job.SourceDir):]
	var extension = filepath.Ext(sourceFile)
	var name = sourceFile[0 : len(sourceFile)-len(extension)]

	return job.job.DestinationDir + name + ".pdf"
}

// Converts the source file to the destination file utilizing the onlyoffice convert api
func (job *convertJob) convertFile(sourceFile string, sourceid int, destinationFile string) {
	logger.Debug("Trying to convert %s (%d) to %s", sourceFile, sourceid, destinationFile)

	nextcloud.CreateFoldersRecursively(job.ncUser, destinationFile)

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, job.ncUser.NextcloudBaseUrl+"/apps/onlyoffice/downloadas", nil)
	if err != nil {
		logger.Error("%s", err)
	}
	req.SetBasicAuth(job.ncUser.Username, job.ncUser.Password)

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
	uploadReq, err := http.NewRequest(http.MethodPut, job.ncUser.NextcloudBaseUrl+"/remote.php/dav/files/"+job.ncUser.Username+"/"+destinationFile, res.Body)

	if err != nil {
		logger.Error("%s", err)
	}
	uploadReq.SetBasicAuth(job.ncUser.Username, job.ncUser.Password)
	uploadReq.Header.Set("Content-Type", "application/binary")

	res, err = uploadClient.Do(uploadReq)
	if err != nil {
		logger.Error("%s", err)
	}

	if res.StatusCode != 204 && res.StatusCode != 201 {
		logger.Error("Failed to create file %s (#%d)", destinationFile, res.StatusCode)
	}
	// Status Code 201
	res.Body.Close()
}
