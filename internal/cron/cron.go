// Package cron provides a small wrapper around gocron for running named jobs
// on a cron schedule. Callers build a Scheduler, Register one or more jobs,
// then Start it; Shutdown stops the scheduler and waits for in-flight runs.
package cron

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

// Scheduler runs registered jobs on their cron schedules.
type Scheduler struct {
	scheduler gocron.Scheduler
	ctx       context.Context
}

// New creates a scheduler. ctx is passed to each job on every run, so jobs are
// cancelled when ctx is done.
func New(ctx context.Context) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}
	return &Scheduler{scheduler: s, ctx: ctx}, nil
}

// Register schedules task to run on the given crontab (5-field cron syntax).
// name identifies the job in logs and registration errors.
func (s *Scheduler) Register(name, crontab string, task func(context.Context)) error {
	if _, err := s.scheduler.NewJob(
		gocron.CronJob(crontab, false),
		gocron.NewTask(task, s.ctx),
	); err != nil {
		return fmt.Errorf("schedule job %q: %w", name, err)
	}
	return nil
}

// Start begins executing scheduled jobs.
func (s *Scheduler) Start() {
	s.scheduler.Start()
	slog.Info("cron scheduler started")
}

// Shutdown stops the scheduler, waiting for any in-flight run to finish or the
// context to be cancelled.
func (s *Scheduler) Shutdown(ctx context.Context) {
	if err := s.scheduler.ShutdownWithContext(ctx); err != nil {
		slog.Error("cron scheduler shutdown error", "error", err)
	}
}
