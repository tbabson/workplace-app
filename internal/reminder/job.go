package reminder

import (
	"context"
	"fmt"
	"log"
	"time"

	"workplace/internal/attendance"
	"workplace/internal/notification"
	"workplace/internal/project"
	"workplace/internal/task"

	"github.com/redis/go-redis/v9"
)

type Job struct {
	projectRepo    *project.Repository
	taskRepo       *task.Repository
	attendanceRepo *attendance.Repository
	notifSvc       *notification.Service
	rdb            *redis.Client
	loc            *time.Location
}

// New creates the reminder job. Times are interpreted in loc. rdb is used to make
// sure each job runs once per day even when several app replicas are running.
func New(projectRepo *project.Repository, taskRepo *task.Repository, attendanceRepo *attendance.Repository, notifSvc *notification.Service, rdb *redis.Client, loc *time.Location) *Job {
	return &Job{
		projectRepo:    projectRepo,
		taskRepo:       taskRepo,
		attendanceRepo: attendanceRepo,
		notifSvc:       notifSvc,
		rdb:            rdb,
		loc:            loc,
	}
}

// Run blocks forever, running three daily jobs:
// 08:00 — deadline reminders, 16:30 — sign-out reminder, 18:00 — auto-close open records.
func (j *Job) Run() {
	go j.runDaily("deadlines", 8, 0, j.fireDeadlineReminders)
	go j.runDaily("signout", 16, 30, j.fireSignOutReminder)
	j.runDaily("autoclose", 18, 0, j.fireAutoClose)
}

func (j *Job) runDaily(name string, hour, minute int, fn func()) {
	for {
		now := time.Now().In(j.loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, j.loc)
		if !now.Before(next) {
			// AddDate (not +24h) keeps the wall-clock time across DST changes.
			next = next.AddDate(0, 0, 1)
		}
		time.Sleep(time.Until(next))
		if j.claim(name, next) {
			fn()
		}
	}
}

// claim takes a per-job, per-day lock in Redis so only one replica runs the job.
func (j *Job) claim(name string, day time.Time) bool {
	key := "reminder:" + name + ":" + day.Format("2006-01-02")
	ok, err := j.rdb.SetNX(context.Background(), key, "1", 23*time.Hour).Result()
	if err != nil {
		log.Printf("reminder: lock %s: %v (running anyway)", key, err)
		return true
	}
	return ok
}

func (j *Job) fireDeadlineReminders() {
	ctx := context.Background()

	// Projects due tomorrow
	projects, err := j.projectRepo.ListDueTomorrow(ctx)
	if err != nil {
		log.Printf("reminder: list projects due tomorrow: %v", err)
	} else {
		for _, p := range projects {
			assignees, err := j.projectRepo.GetAssignees(ctx, p.ID)
			if err != nil {
				continue
			}
			refID := p.ID
			var inputs []notification.NotifyInput
			for _, uid := range assignees {
				inputs = append(inputs, notification.NotifyInput{
					UserID: uid,
					Type:   notification.TypeProjectDueReminder,
					Title:  "Project Due Tomorrow: " + p.Title,
					Body:   fmt.Sprintf(`Project "%s" is due tomorrow. Make sure all tasks are completed.`, p.Title),
					RefID:  &refID,
				})
			}
			j.notifSvc.NotifyMany(ctx, inputs)
		}
	}

	// Tasks due tomorrow
	tasks, err := j.taskRepo.ListDueTomorrow(ctx)
	if err != nil {
		log.Printf("reminder: list tasks due tomorrow: %v", err)
	} else {
		for _, t := range tasks {
			if t.AssignedTo == nil {
				continue
			}
			refID := t.ID
			j.notifSvc.Notify(ctx, notification.NotifyInput{
				UserID: *t.AssignedTo,
				Type:   notification.TypeTaskDueReminder,
				Title:  "Task Due Tomorrow: " + t.Title,
				Body:   fmt.Sprintf(`Task "%s" is due tomorrow. Please ensure it's completed on time.`, t.Title),
				RefID:  &refID,
			})
		}
	}
}

func (j *Job) fireAutoClose() {
	ctx := context.Background()

	closedAt := time.Now().In(j.loc)
	userIDs, err := j.attendanceRepo.AutoCloseOpenRecords(ctx, closedAt)
	if err != nil {
		log.Printf("reminder: auto-close open records: %v", err)
		return
	}

	if len(userIDs) == 0 {
		return
	}

	var inputs []notification.NotifyInput
	for _, uid := range userIDs {
		inputs = append(inputs, notification.NotifyInput{
			UserID: uid,
			Type:   notification.TypeAttendanceAutoClosed,
			Title:  "Attendance Auto-Closed",
			Body:   fmt.Sprintf("You did not sign out today. Your attendance has been automatically closed at %s.", closedAt.Format("3:04 PM")),
		})
	}
	j.notifSvc.NotifyMany(ctx, inputs)
}

func (j *Job) fireSignOutReminder() {
	ctx := context.Background()

	userIDs, err := j.attendanceRepo.ListSignedInToday(ctx)
	if err != nil {
		log.Printf("reminder: list signed-in users: %v", err)
		return
	}

	var inputs []notification.NotifyInput
	for _, uid := range userIDs {
		inputs = append(inputs, notification.NotifyInput{
			UserID: uid,
			Type:   notification.TypeSignOutReminder,
			Title:  "Don't Forget to Sign Out",
			Body:   "Closing time is 5:00 PM. Please remember to sign out before you leave.",
		})
	}
	j.notifSvc.NotifyMany(ctx, inputs)
}
