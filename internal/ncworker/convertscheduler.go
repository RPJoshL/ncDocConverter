package ncworker

import (

	"rpjosh.de/ncDocConverter/internal/models"
)

type NcConvertScheduler struct {
	users		*models.NcConvertUsers
}

func NewScheduler(users *models.NcConvertUsers) *NcConvertScheduler {
	scheduler := NcConvertScheduler {
		users: users,
	}

	scheduler.ScheduleExecutions()
	
	return &scheduler
}

func (scheduler NcConvertScheduler) ScheduleExecutions() {
	for _, user := range scheduler.users.Users {
		for _, job := range user.ConvertJobs {
			convJob := NewJob(&job, &user)
			convJob.ExecuteJob()
		}

	}
}