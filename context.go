// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"context"
	"time"
)

// Context wraps a golang context/Context with a cancel function when available.
// See https://golang.org/pkg/context/#Context
type Context struct {
	ctx context.Context
	// used to cancel the context. Does not work on ContextBackground().
	cancel context.CancelFunc
}

// ContextBackground returns the Background context that is never cancelled.
// ref https://golang.org/pkg/context/#Background
func ContextBackground() *Context {
	return &Context{ctx: context.Background()}
}

// WithCancel returns a new Context that can cancelled with Context.Cancel() from the given Context.
// ref https://golang.org/pkg/context/#WithCancel
func (c *Context) WithCancel() *Context {
	newCtx, cancel := context.WithCancel(c.ctx)
	return &Context{ctx: newCtx, cancel: cancel}
}

// WithTimeout returns a new Context that is automatically cancelled after `seconds` pass or
// when its parent context is cancelled.
// It can still be cancelled beforehand with Cancel().
// ref https://golang.org/pkg/context/#WithTimeout
func (c *Context) WithTimeout(seconds int) *Context {
	newCtx, cancel := context.WithTimeout(c.ctx, time.Duration(seconds)*time.Second)
	return &Context{ctx: newCtx, cancel: cancel}
}

// ContextWithCancel returns a new Context that can be cancelled with Context.Cancel() from the Background Context.
// ref https://golang.org/pkg/context/#WithCancel
func ContextWithCancel() *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{ctx: ctx, cancel: cancel}
}

// ContextWithTimeout returns a new Context that is automatically cancelled after `seconds` pass.
// It can still be cancelled beforehand with Cancel().
// ref https://golang.org/pkg/context/#WithTimeout
func ContextWithTimeout(seconds int) *Context {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	return &Context{ctx: ctx, cancel: cancel}
}

// Cancel cancels the context. Does not work on ContextBackground().
// It is safe to call Cancel() more than once, even from different threads.
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Context can not be called from Java, only here to improve reusability.
func (c *Context) Context() (context.Context, context.CancelFunc) {
	return c.ctx, c.cancel
}
