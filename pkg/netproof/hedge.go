package netproof

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"
)

var ErrStopHedging = errors.New("stop hedging")

type Hedge struct {
	ctx         context.Context
	maxRequests int
	getDelay    func() time.Duration
	next        *requestHeap
	nextLock    sync.Mutex
	t           timer
}

const tickerResolution = 50 * time.Millisecond

// NewHedgeFixed creates a new Hedge with fixed reqDelay.
func NewHedgeFixed(ctx context.Context, maxRequests int, reqDelay time.Duration) *Hedge {
	return NewHedge(ctx, maxRequests,
		func() time.Duration {
			return reqDelay
		},
	)
}

// NewHedge creates a new Hedge with minimum delay tickerResolution.
// When ctx is canceled, it stops any ongoing requests.
func NewHedge(ctx context.Context, maxRequests int, getDelay func() time.Duration) *Hedge {
	return newHedge(ctx, maxRequests, getDelay, realTimer{})
}

func newHedge(ctx context.Context, maxRequests int, getDelay func() time.Duration, t timer) *Hedge {
	next := &requestHeap{}
	heap.Init(next)
	h := &Hedge{
		ctx:         ctx,
		maxRequests: maxRequests,
		getDelay:    getDelay,
		next:        next,
		t:           t,
	}
	go h.ticker()
	return h
}

func (h *Hedge) ticker() {
	nextTick := h.t.Now().Add(tickerResolution)
	for {
		err := h.ctx.Err()
		if err != nil {
			for {
				req, ok := h.getAnyRequest()
				if !ok {
					break
				}
				req.setError(err)
			}
			return
		}
		h.t.Sleep(h.t.Until(nextTick))
		nextTick = h.t.Now().Add(tickerResolution)
		for {
			req, ok := h.getExpiredRequest()
			if !ok {
				// wait until the next earliest request
				break
			}
			go func(req *hedgeRequest) {
				if req.stopped() {
					// prevent running a stopped request
					return
				}
				req.setError(req.f())
			}(req)
		}

	}
}

// Hedge runs f() and returns the first successful request.
// When f() fails, it'll retry up to maxRequests times, with reqDelay between each request.
// When f() returns ErrStopHedging it stops all requests.
func (h *Hedge) Hedge(f func() error) error {
	requests := newRequests(h.t, h.maxRequests, h.getDelay(), f)
	for _, req := range requests {
		if err := h.addRequest(req); err != nil {
			return err
		}
	}
	var err error
	var ok bool
	for _, req := range requests {
		err, ok = req.getError()
		if !ok {
			// ignore unfinished requests
			continue
		}
		if err == nil || errors.Is(err, ErrStopHedging) {
			break
		}
	}
	return err
}

// WaitingRequests returns the number of requests waiting to be sent.
func (h *Hedge) WaitingRequests() int {
	h.nextLock.Lock()
	defer h.nextLock.Unlock()
	return h.next.Len()
}

func (h *Hedge) getExpiredRequest() (*hedgeRequest, bool) {
	h.nextLock.Lock()
	defer h.nextLock.Unlock()
	if h.next.Len() == 0 {
		return nil, false
	}
	if h.t.Now().Before((*h.next)[0].deadline) {
		return nil, false
	}
	req := heap.Pop(h.next).(*hedgeRequest)
	return req, true
}

func (h *Hedge) getAnyRequest() (*hedgeRequest, bool) {
	h.nextLock.Lock()
	defer h.nextLock.Unlock()
	if h.next.Len() == 0 {
		return nil, false
	}
	req := heap.Pop(h.next).(*hedgeRequest)
	return req, true
}

func (h *Hedge) addRequest(r *hedgeRequest) error {
	h.nextLock.Lock()
	defer h.nextLock.Unlock()
	if err := h.ctx.Err(); err != nil {
		// prevent new requests if ctx is canceled
		return err
	}
	heap.Push(h.next, r)
	return nil
}

type hedgeRequest struct {
	deadline  time.Time    // immutable
	f         func() error // immutable
	condErr   error
	condReady bool
	condStop  *bool      // per request batch
	cond      *sync.Cond // per request batch
}

func newRequests(t timer, total int, delay time.Duration, f func() error) []*hedgeRequest {
	cond := sync.NewCond(&sync.Mutex{})
	stop := false
	now := t.Now()
	var requests []*hedgeRequest
	for i := 0; i < total; i++ {
		requests = append(requests, &hedgeRequest{
			deadline: now.Add(delay * time.Duration(i)),
			f:        f,
			cond:     cond,
			condStop: &stop,
		})
	}
	return requests
}

func (r *hedgeRequest) getError() (err error, ok bool) {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()
	for !(r.condReady || *r.condStop) {
		r.cond.Wait()
	}
	return r.condErr, r.condReady
}

func (r *hedgeRequest) setError(err error) {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()
	r.condErr = err
	r.condReady = true
	*r.condStop = err == nil || errors.Is(err, ErrStopHedging)
	r.cond.Signal()
}

func (r *hedgeRequest) stopped() bool {
	r.cond.L.Lock()
	defer r.cond.L.Unlock()
	return *r.condStop
}

type requestHeap []*hedgeRequest

func (h requestHeap) Len() int           { return len(h) }
func (h requestHeap) Less(i, j int) bool { return h[i].deadline.Before(h[j].deadline) }
func (h requestHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *requestHeap) Push(x any) {
	*h = append(*h, x.(*hedgeRequest))
}

func (h *requestHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil // invalidate the item to allow any GC
	*h = old[0 : n-1]
	return x
}
