// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import "perun.network/go-perun/client"

type (
	// An UpdateHandler decides how to handle incoming channel update requests
	// from other channel participants.
	UpdateHandler interface {
		// HandleUpdate is the user callback called by the channel controller
		// on an incoming update request.
		HandleUpdate(*ChannelUpdate, *UpdateResponder)
	}

	// updateHandler implements a client.UpdateHandler wrapping a prnm
	// UpdateHandler
	updateHandler struct {
		h UpdateHandler
	}

	// ChannelUpdate is a channel update proposal.
	// If the ActorIdx is the own channel index, this is a payment request.
	// If State.IsFinal() is true, this is a request to finalize the channel.
	ChannelUpdate struct {
		State    *State // Proposed new state.
		ActorIdx int    // Who is transferring funds.
	}

	// An UpdateResponder lets the user respond to a channel update. If the
	// user wants to accept the update, they should call Accept(), otherwise
	// Reject(). Only a single function must be called and every further call
	// causes a panic.
	UpdateResponder struct {
		r *client.UpdateResponder
	}
)

// HandleUpdate implements the client.UpdateHandler interface by converting the
// passed types from the go-perun/client package into their local counterparts
// and then calling the prnm.UpdateHandler.
func (h *updateHandler) HandleUpdate(_update client.ChannelUpdate, _resp *client.UpdateResponder) {
	update := &ChannelUpdate{
		State:    &State{_update.State},
		ActorIdx: int(_update.ActorIdx),
	}
	resp := &UpdateResponder{r: _resp}
	h.h.HandleUpdate(update, resp)
}

// Accept lets the user signal that they want to accept the channel update.
func (r *UpdateResponder) Accept(ctx *Context) error {
	return r.r.Accept(ctx.ctx)
}

// Reject lets the user signal that they reject the channel update.
func (r *UpdateResponder) Reject(ctx *Context, reason string) error {
	return r.r.Reject(ctx.ctx, reason)
}
