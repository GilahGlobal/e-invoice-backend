package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

type Scheduler struct {
	scheduler *asynq.Scheduler
}

func NewScheduler(redisConnection asynq.RedisClientOpt) *Scheduler {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		panic(err)
	}
	scheduler := asynq.NewScheduler(
		redisConnection,
		&asynq.SchedulerOpts{
			Location: loc,
		},
	)
	return &Scheduler{scheduler: scheduler}
}

func (p *Scheduler) Start() error {
	// err := p.ScheduleTask("bulkUpload", "0 0 0 0")
	// if err != nil {
	// 	return err
	// }

	go func() {
		if err := p.scheduler.Run(); err != nil {
			log.Printf("❌ Scheduler stopped: %v", err)
		}
	}()

	log.Println("🟢 Scheduler started successfully.")
	return nil
}

func (p *Scheduler) ScheduleTask(taskName string, cronExpr string) error {
	task := asynq.NewTask(taskName, nil)
	_, err := p.scheduler.Register(cronExpr, task)
	if err != nil {
		return fmt.Errorf("failed to schedule task: %v", err)
	}
	return nil
}
