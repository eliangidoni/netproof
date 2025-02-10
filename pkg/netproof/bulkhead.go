package netproof

import (
	"context"
	"errors"
)

type Bulkhead struct {
	available chan any
	ctx       context.Context
}

var ErrExhausted = errors.New("bulkhead exhausted")

// NewBulkhead creates a new Bulkhead with the given resources.
// When ctx is canceled, it stops any ongoing requests.
func NewBulkhead(ctx context.Context, resources []any) *Bulkhead {
	available := make(chan any, len(resources))
	for _, i := range resources {
		available <- i
	}
	return &Bulkhead{ctx: ctx, available: available}
}

func (b *Bulkhead) TotalUsed() int {
	return cap(b.available) - len(b.available)
}

// BulkheadWait runs the given function f() up to len(resources) times, otherwise blocks.
// When either context is canceled returns the ctx error.
func (b *Bulkhead) BulkheadWait(ctx context.Context, f func(resource any) error) error {
	select {
	case resource := <-b.available:
		err := f(resource)
		b.available <- resource
		return err
	case <-b.ctx.Done():
		return b.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Bulkhead runs the given function f() up to len(resources) times, otherwise returns ErrExhausted.
// When either context is canceled returns the ctx error.
func (b *Bulkhead) Bulkhead(ctx context.Context, f func(resource any) error) error {
	select {
	case resource := <-b.available:
		err := f(resource)
		b.available <- resource
		return err
	case <-b.ctx.Done():
		return b.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrExhausted
	}
}
