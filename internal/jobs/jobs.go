package jobs

import "context"

type BackgroundJob struct{ fn func(context.Context) error }

func NewBackgroundJob(fn func(context.Context) error) BackgroundJob { return BackgroundJob{fn: fn} }
func (j BackgroundJob) Run(ctx context.Context) error               { return j.fn(ctx) }
