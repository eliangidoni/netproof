package netproof

import "context"

func withContext(f func(ctx context.Context)) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	f(ctx)
}
