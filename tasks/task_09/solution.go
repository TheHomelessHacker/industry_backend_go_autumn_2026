package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](
	ctx context.Context, 
	workers int, 
	in []T, 
	fn func(context.Context, T) (R, error),
) ([]Result[R], error) {

results := make([]Result[R], len(in))
jobs := make(chan int)
var wg sync.WaitGroup
var zero R

if workers <= 0 {
	return nil, ErrInvalidWorkers
}

if ctx.Err() != nil {
	return nil, ctx.Err()
}

if len(in) == 0 {
	return make([]Result[R], 0), nil
}

for range workers {
	wg.Go(func() {
		for i := range jobs {
			value, err := fn(ctx, in[i])
			if err != nil {
				results[i].Value = zero
				results[i].Err = err
			} else {
				results[i].Value = value
				results[i].Err = nil
			}
		}
	})
}

feed:
	for i := range in {
		
		select {
		case jobs <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() == nil {
		return results, ctx.Err()
	}
	return nil, ctx.Err()
}