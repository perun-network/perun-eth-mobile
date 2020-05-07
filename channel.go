// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-demo. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/pkg/errors"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

type State struct {
	s *channel.State
}

func (s *State) GetID() []byte {
	return s.s.ID[:]
}

func (s *State) GetVersion() int64 {
	return int64(s.s.Version)
}

func (s *State) GetBalances() *BigInts {
	return &BigInts{values: s.s.Balances[0]}
}

func (s *State) IsFinal() bool {
	return s.s.IsFinal
}

type Params struct {
	params *channel.Params
}

func (p *Params) GetID() []byte {
	id := p.params.ID()
	return id[:]
}

func (p *Params) GetChallengeDuration() int64 {
	return int64(p.params.ChallengeDuration)
}

func (p *Params) GetParts() *Addresses {
	addrs := make([]ethwallet.Address, len(p.params.Parts))
	for i := range addrs {
		addrs[i] = *p.params.Parts[i].(*ethwallet.Address)
	}
	return &Addresses{values: addrs}
}

type PaymentChannel struct {
	ch *client.Channel
}

func (c *PaymentChannel) HandleUpdates(handler client.UpdateHandler) {
	c.ch.ListenUpdates(handler)
}

func (c *PaymentChannel) Watch() error {
	return c.ch.Watch()
}

func (c *PaymentChannel) Send(ctx *Context, amount *BigInt) error {
	if amount.i.Sign() <= 1 {
		return errors.New("Only positive amounts supported in send")
	}

	state := c.ch.State().Clone()
	my := c.ch.Idx()
	other := 1 - my
	state.Allocation.Balances[0][my].Sub(state.Allocation.Balances[0][my], amount.i)
	state.Allocation.Balances[0][other].Add(state.Allocation.Balances[0][other], amount.i)
	return c.ch.Update(ctx.ctx, client.ChannelUpdate{
		State:    state,
		ActorIdx: c.ch.Idx(),
	})
}

func (c *PaymentChannel) GetIdx() int {
	return int(c.ch.Idx())
}

func (c *PaymentChannel) Finalize(ctx *Context) error {
	state := c.ch.State().Clone()
	state.IsFinal = true
	return c.ch.Update(ctx.ctx, client.ChannelUpdate{
		State:    state,
		ActorIdx: c.ch.Idx(),
	})
}

func (c *PaymentChannel) Settle(ctx *Context) error {
	return c.ch.Settle(ctx.ctx)
}

func (c *PaymentChannel) Close() error {
	return c.ch.Close()
}

func (c *PaymentChannel) GetState() *State {
	return &State{c.ch.State()}
}

func (c *PaymentChannel) GetParams() *Params {
	return &Params{c.ch.Params()}
}
