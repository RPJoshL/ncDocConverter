package ncworker

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/pkg/logger"
)

type NcConvertScheduler struct {
	users  *models.NcConvertUsers
	config *models.WebConfig

	scheduler *gocron.Scheduler
}

func NewScheduler(users *models.NcConvertUsers, config *models.WebConfig) *NcConvertScheduler {
	scheduler := NcConvertScheduler{
		users:     users,
		config:    config,
		scheduler: gocron.NewScheduler(time.Local),
	}
	// Don't reschedule a task if it's still running
	scheduler.scheduler.SingletonMode()
	scheduler.scheduler.StartAsync()

	if config.Server.OneShot {
		scheduler.ScheduleExecutionsOneShot()
	} else {
		scheduler.ScheduleExecutions()

		fmt.Println("Started in schedule mode.\nType \"exit\" to leave or \"execute\" to execute all jobs")
		// Endless loop
		for {
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString('\n')
			if err != nil {
				// No console input
				var wg sync.WaitGroup
				logger.Debug("No console available")
				wg.Add(1)
				wg.Wait()
			}

			input := strings.Trim(strings.ToLower(text), "\n")
			if input == "exit" {
				break
			} else if input == "execute" {
				scheduler.scheduler.RunAll()
			}
		}
	}

	return &scheduler
}

// Executes all jobs and exits the program afterwards
func (scheduler NcConvertScheduler) ScheduleExecutionsOneShot() {
	for _, user := range scheduler.users.Users {

		// Schedule Nextcloud jobs
		for _, job := range user.ConvertJobs {
			convJob := NewNcJob(&job, &user)
			convJob.ExecuteJob()
		}

		// Schedule boockstack jobs
		if user.BookStack.URL != "" {
			for _, job := range user.BookStack.Jobs {
				bsJob := NewBsJob(&job, &user)
				bsJob.ExecuteJob()
			}
		}

	}
}

// Schedules all jobs with gocron
func (s NcConvertScheduler) ScheduleExecutions() {
	for ui, user := range s.users.Users {

		// Schedule Nextcloud jobs
		for i, job := range user.ConvertJobs {
			convJob := NewNcJob(&s.users.Users[ui].ConvertJobs[i], &s.users.Users[i])

			_, err := s.scheduler.Cron(job.Execution).DoWithJobDetails(s.executeJob, convJob)
			if err != nil {
				logger.Fatal("Failed to schedule office job '%s': %s", job.JobName, err)
			}
		}

		// Schedule boockstack jobs
		if user.BookStack.URL != "" {
			for i, job := range user.BookStack.Jobs {
				bsJob := NewBsJob(&s.users.Users[ui].BookStack.Jobs[i], &s.users.Users[i])

				_, err := s.scheduler.Cron(job.Execution).DoWithJobDetails(s.executeJob, bsJob)
				if err != nil {
					logger.Fatal("Failed to schedule BookStack job '%s': %s", job.JobName, err)
				}
			}
		}
	}
}

func (s NcConvertScheduler) executeJob(job Job, scheduledJob gocron.Job) {
	job.ExecuteJob()
}
