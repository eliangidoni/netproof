package netproof

import (
	"sync"
	"time"
)

type timer interface {
	Now() time.Time
	Sleep(time.Duration)
	Since(time.Time) time.Duration
	Until(time.Time) time.Duration
}

type realTimer struct{}

func (r realTimer) Now() time.Time {
	return time.Now()
}

func (r realTimer) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (r realTimer) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (r realTimer) Sleep(d time.Duration) {
	time.Sleep(d)
}

var _ timer = &realTimer{}

type fakeTimer struct {
	now  time.Time
	cond *sync.Cond
}

func newFakeTimer() *fakeTimer {
	t, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		panic(err)
	}
	return &fakeTimer{
		now:  t,
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (f *fakeTimer) Now() time.Time {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	return f.now
}

func (f *fakeTimer) Since(t time.Time) time.Duration {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	return f.now.Sub(t)
}

func (f *fakeTimer) Until(t time.Time) time.Duration {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	return t.Sub(f.now)
}

func (f *fakeTimer) Advance(d time.Duration) {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	f.now = f.now.Add(d)
	f.cond.Broadcast()
}

func (f *fakeTimer) Sleep(d time.Duration) {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	next := f.now.Add(d)
	for {
		if next.Before(f.now) || next.Equal(f.now) {
			break
		}
		f.cond.Wait()
	}
}

var _ timer = &fakeTimer{}
