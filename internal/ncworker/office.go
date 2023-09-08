package ncworker

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
	"git.rpjosh.de/ncDocConverter/internal/models"
	"git.rpjosh.de/ncDocConverter/internal/nextcloud"
	"git.rpjosh.de/ncDocConverter/pkg/utils"
)

type convertJob struct {
	job    *models.NcConvertJob
	ncUser *models.NextcloudUser
}

type convertQueu struct {
	source      nextcloud.NcFile
	destination string
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
	sourceFolder, err := nextcloud.SearchInDirectory(
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

	destinationFolder, err := nextcloud.SearchInDirectory(
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

	// Store all files in a map
	prefix := "/remote.php/dav/files/" + job.ncUser.Username + "/"
	sourceMap := nextcloud.ParseSearchResult(sourceFolder, prefix, job.job.SourceDir)
	destinationMap := nextcloud.ParseSearchResult(destinationFolder, prefix, job.job.DestinationDir)

	// check which files should be converted
	var filesToConvert []convertQueu
	var directorys []string

	for index, source := range sourceMap {
		// Check if the file exists in the destination map
		if dest, exists := destinationMap[index]; exists {
			// Compare timestamp and size
			if dest.LastModified.Before(source.LastModified) {
				filesToConvert = append(filesToConvert, convertQueu{source: source, destination: dest.Path})
			}
			delete(destinationMap, index)
		} else {
			// the directory could not be existing -> check for existance
			destinationDir := job.getDestinationDir(source.Path)
			appendIfNotExists(&directorys, destinationDir[0:strings.LastIndex(destinationDir, "/")+1])

			filesToConvert = append(filesToConvert, convertQueu{source: source, destination: destinationDir})

			delete(destinationMap, index)
		}
	}

	var wg sync.WaitGroup

	// Delete the files which are not available anymore
	wg.Add(len(destinationMap))
	for _, dest := range destinationMap {
		go func(file *nextcloud.NcFile) {
			err := nextcloud.DeleteFile(job.ncUser, file.Path)
			if err != nil {
				logger.Error(utils.FirstCharToUppercase(err.Error()))
			}
			wg.Done()
		}(&dest)
	}
	wg.Wait()

	// Create required directorys
	wg.Add(len(directorys))
	for _, dest := range directorys {
		go func(path string) {
			nextcloud.CreateFoldersRecursively(job.ncUser, path)
			wg.Done()
		}(dest)
	}
	wg.Wait()

	// Convert the files
	wg.Add(len(filesToConvert))
	for _, file := range filesToConvert {
		go func(cvt convertQueu) {
			job.convertFile(cvt.source.Path, cvt.source.Fileid, cvt.destination)
			wg.Done()
		}(file)
	}
	wg.Wait()

	logger.Info("Finished Nextcloud job \"%s\": %d documents converted", job.job.JobName, len(filesToConvert))
}

// Appends the directory to the array if it isn't contained
// by another element already
func appendIfNotExists(dirs *[]string, directory string) {
	directoryLength := len(directory)
	for i, currentDir := range *dirs {
		currentLength := len(currentDir)

		// the existing directory is already referenced in the current
		if directoryLength > currentLength && directory[0:currentLength] == currentDir {
			(*dirs)[i] = directory
			continue
		} else if directoryLength <= currentLength && currentDir[0:directoryLength] == directory {
			continue
		}
	}
	*dirs = append(*dirs, directory)
}

func (job *convertJob) getDestinationDir(sourceFile string) string {
	sourceFile = sourceFile[len(job.job.SourceDir):]
	var extension = filepath.Ext(sourceFile)
	var name = sourceFile[0 : len(sourceFile)-len(extension)]

	return job.job.DestinationDir + name + ".pdf"
}

// Converts the source file to the destination file utilizing the onlyoffice convert api
func (job *convertJob) convertFile(sourceFile string, sourceid int, destinationFile string) {
	logger.Debug("Converting %s (%d) to %s", sourceFile, sourceid, destinationFile)

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
		logger.Error("Failed to access the convert api: %s", err)
		return
	}

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		logger.Error("Failed to access the convert api (#%d). Do you have OnlyOffice installed?: %s", res.StatusCode, body)
		return
	}

	if err := nextcloud.UploadFile(job.ncUser, destinationFile, res.Body); err != nil {
		logger.Error("Failed to upload file %q to nextcloud: %s", destinationFile, err)
	}

	res.Body.Close()
}
