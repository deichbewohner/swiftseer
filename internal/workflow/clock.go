package workflow

import "time"

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
	NewTicker(d time.Duration) Ticker
}

type Ticker interface {
	Chan() <-chan time.Time
	Stop()
}

type realClock struct{}

func (realClock) Now() time.Time                   { return time.Now() }
func (realClock) Since(t time.Time) time.Duration  { return time.Since(t) }
func (realClock) Until(t time.Time) time.Duration  { return time.Until(t) }
func (realClock) NewTicker(d time.Duration) Ticker { return realTicker{t: time.NewTicker(d)} }

type realTicker struct{ t *time.Ticker }

func (r realTicker) Chan() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()                  { r.t.Stop() }
