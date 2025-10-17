package workflow

import (
	"context"
)

type Job struct {
	events  <-chan Event
	resultC chan *RunResult
	errC    chan error
	cancel  context.CancelFunc
}

func (j *Job) Events() <-chan Event { return j.events }

func (j *Job) Wait() (*RunResult, error) {
	select {
	case res := <-j.resultC:
		return res, nil
	case err := <-j.errC:
		return nil, err
	}
}

func (j *Job) Cancel() {
	if j.cancel != nil {
		j.cancel()
	}
}

func (r *Runner) RunAsync(ctx context.Context, params RunParams) *Job {
	events := make(chan Event, 256)
	resC := make(chan *RunResult, 1)
	errC := make(chan error, 1)
	runCtx, cancel := context.WithCancel(ctx)
	rr := &Runner{
		api:            r.api,
		reporter:       NewChannelReporter(events),
		pollInterval:   r.pollInterval,
		clean:          r.clean,
		tokenExp:       r.tokenExp,
		clock:          r.clock,
		checkInBuilder: r.checkInBuilder,
	}

	go func() {
		defer close(events)
		res, err := rr.Run(runCtx, params)
		if err != nil {
			errC <- err
			return
		}
		resC <- res
	}()

	return &Job{events: events, resultC: resC, errC: errC, cancel: cancel}
}
