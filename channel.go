// Copyright (c) 2021 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/pkg/errors"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

// State wraps a go-perun/channel.State
// ref https://pkg.go.dev/perun.network/go-perun/channel?tab=doc#State
type State struct {
	s *channel.State
}

// GetID returns the immutable id of the channel this state belongs to.
func (s *State) GetID() []byte {
	return s.s.ID[:]
}

// GetVersion returns the version counter.
func (s *State) GetVersion() int64 {
	return int64(s.s.Version)
}

// GetBalances returns a BigInts with length two containing the current
// balances.
func (s *State) GetBalances() *BigInts {
	return &BigInts{values: s.s.Balances[0]}
}

// IsFinal indicates that the channel is in its final state.
// Such a state can immediately be settled on the blockchain.
// A final state cannot be further progressed.
func (s *State) IsFinal() bool {
	return s.s.IsFinal
}

// Params wraps a go-perun/channel.Params
// ref https://pkg.go.dev/perun.network/go-perun/channel?tab=doc#Params
type Params struct {
	params *channel.Params
}

// GetID returns the channelID of this channel.
// ref https://pkg.go.dev/perun.network/go-perun/channel?tab=doc#Params.ID
func (p *Params) GetID() []byte {
	id := p.params.ID()
	return id[:]
}

// GetChallengeDuration how many seconds an on-chain dispute of a
// non-final channel can be refuted.
func (p *Params) GetChallengeDuration() int64 {
	return int64(p.params.ChallengeDuration)
}

// GetParts returns the channel participants.
func (p *Params) GetParts() *Addresses {
	addrs := make([]ethwallet.Address, len(p.params.Parts))
	for i := range addrs {
		addrs[i] = *p.params.Parts[i].(*ethwallet.Address)
	}
	return &Addresses{values: addrs}
}

type (
	// PaymentChannel is a convenience wrapper for go-perun/client.Channel
	// which provides all necessary functionality of a two-party payment channel.
	// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel
	PaymentChannel struct {
		ch *client.Channel
	}

	// ConcludedEventHandler handles channel conclusions.
	ConcludedEventHandler interface {
		HandleConcluded(id []byte)
	}

	// ConcludedWatcher implements the AdjudicatorEventHandler and notifies the
	// ConcludedEventHandler if an channel is concluded.
	ConcludedWatcher struct {
		h ConcludedEventHandler
	}
)

// HandleAdjudicatorEvent handles channel events emitted by the Adjudicator.
func (w *ConcludedWatcher) HandleAdjudicatorEvent(e channel.AdjudicatorEvent) {
	if _, ok := e.(*channel.ConcludedEvent); ok {
		id := e.ID()
		w.h.HandleConcluded(id[:])
	}
}

// Watch starts the channel watcher routine. It subscribes to RegisteredEvents
// on the adjudicator. If an event is registered, it is handled by making sure
// the latest state is registered and then all funds withdrawn to the receiver
// specified in the adjudicator that was passed to the channel.
// In case of a channel conclusion event, the given handler `h` is called.
//
// If handling failed, the watcher routine returns the respective error. It is
// the user's job to restart the watcher after the cause of the error got fixed.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Watch
func (c *PaymentChannel) Watch(h ConcludedEventHandler) error {
	w := &ConcludedWatcher{h: h}
	return c.ch.Watch(w)
}

// Send pays `amount` to the counterparty. Only positive amounts are supported.
func (c *PaymentChannel) Send(ctx *Context, amount *BigInt) error {
	if amount.i.Sign() < 1 {
		return errors.New("Only positive amounts supported in send")
	}

	return c.ch.UpdateBy(ctx.ctx, func(state *channel.State) error {
		my := c.ch.Idx()
		other := 1 - my
		bals := state.Allocation.Balances[0]
		bals[my].Sub(bals[my], amount.i)
		bals[other].Add(bals[other], amount.i)
		return nil
	})
}

// GetIdx returns our index in the channel.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Idx
func (c *PaymentChannel) GetIdx() int {
	return int(c.ch.Idx())
}

// Finalize finalizes the channel with the current state.
func (c *PaymentChannel) Finalize(ctx *Context) error {
	return c.ch.UpdateBy(ctx.ctx, func(state *channel.State) error {
		state.IsFinal = true
		return nil
	})
}

// Settle settles the channel: it is made sure that the current state is
// registered and the final balance withdrawn. This call blocks until the
// channel has been successfully withdrawn.
// Call Finalize before settling a channel to avoid waiting a full
// challenge duration.
// If the `secondary` flag is set to true, the Adjudicator runs an optimized
// protocol, where it is assumed that the other peer also settles the channel.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Settle
func (c *PaymentChannel) Settle(ctx *Context, secondary bool) error {
	if err := c.ch.Register(ctx.ctx); err != nil {
		return errors.WithMessage(err, "registering")
	}
	return c.ch.Settle(ctx.ctx, secondary)
}

// GetState returns the current state. Do not modify it.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.State
func (c *PaymentChannel) GetState() *State {
	return &State{c.ch.State()}
}

// GetParams returns the channel parameters.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Params
func (c *PaymentChannel) GetParams() *Params {
	return &Params{c.ch.Params()}
}
