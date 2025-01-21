// Package errgroup wraps the standard errgroup capturing panics in the go routines as errors instead.
package errgroup

import (
	"context"

	"github.com/zircuit-labs/zkr-go-common/calm"
	"golang.org/x/sync/errgroup"
)

type Group struct {
	group *errgroup.Group
}

func WithContext(ctx context.Context) (*Group, context.Context) {
	group, ctx := errgroup.WithContext(ctx)
	return &Group{group: group}, ctx
}

func New() *Group {
	return &Group{group: new(errgroup.Group)}
}

func (g *Group) Go(f func() error) {
	g.group.Go(func() error {
		return calm.Unpanic(f)
	})
}

func (g *Group) SetLimit(n int) {
	g.group.SetLimit(n)
}

func (g *Group) TryGo(f func() error) bool {
	return g.group.TryGo(func() error {
		return calm.Unpanic(f)
	})
}

func (g *Group) Wait() error {
	return g.group.Wait()
}
