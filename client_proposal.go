// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"crypto/rand"
	"math/big"

	"perun.network/go-perun/apps/payment"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
	"perun.network/go-perun/wallet"
)

// ProposeChannel proposes a new channel to the given peer (perunID) with
// challengeDuration seconds as the challenge duration in case of disputes and
// initialBals as the initial channel balances. Returns the newly created
// channel controller if the channel was successfully created and funded.
//
// After the channel got successfully created, the user is required to start the
// update handler with PaymentChannel.HandleUpdates(UpdateHandler) and to start
// the channel watcher with PaymentChannel.Watch() on the returned channel
// controller.
//
// It is important that the passed context does not cancel before twice the
// ChallengeDuration has passed (at least for real blockchain backends with wall
// time), or the channel cannot be settled if a peer times out funding.
func (c *Client) ProposeChannel(
	ctx *Context,
	perunID *Address,
	challengeDuration int64,
	initialBals *BigInts,
) (*PaymentChannel, error) {
	alloc := &channel.Allocation{
		Assets:   []channel.Asset{(*ethwallet.Address)(&assetAddr)},
		Balances: [][]channel.Bal{initialBals.values},
	}
	prop := &client.ChannelProposal{
		ChallengeDuration: uint64(challengeDuration),
		Nonce:             nonce(),
		ParticipantAddr:   c.w.NewAccount().Address(),
		AppDef:            payment.AppDef(),
		InitData:          &payment.NoData{},
		InitBals:          alloc,
		PeerAddrs:         []wallet.Address{c.onChain.Address(), (*ethwallet.Address)(&perunID.addr)},
	}
	_ch, err := c.client.ProposeChannel(ctx.ctx, prop)
	return &PaymentChannel{_ch}, err
}

// HandleChannelProposals is the incoming channel proposal handler routine. It
// must only be started at most once by the user. Incoming channel proposals are
// handled using the passed handler.
func (c *Client) HandleChannelProposals(h ProposalHandler) {
	c.client.HandleChannelProposals(&proposalHandler{c: c, h: h})
}

type (
	// A ProposalHandler decides how to handle incoming channel proposals from
	// other channel network peers.
	ProposalHandler interface {
		// Handle is the user callback called by the Client on an incoming channel
		// proposal.
		Handle(*ChannelProposal, *ProposalResponder)
	}

	// proposalHandler implements a client.ProposalHandler wrapping a prnm
	// ProposalHandler
	proposalHandler struct {
		c *Client         // back-reference for ProposalResponder
		h ProposalHandler // wrapped ProposalHandler
	}

	// A ChannelProposal describes a proposal to open a new channel.
	//
	// The proposer has index 0 and proposee index 1.
	ChannelProposal struct {
		PeerPerunID       *Address // The peer proposing the channel.
		ChallengeDuration int64    // Proposed challenge duration in case of disputes, in seconds.
		InitBals          *BigInts // Initial channel balances.
	}

	// A ProposalResponder lets the user respond to a channel proposal. If the
	// user wants to accept the proposal, they should call Accept(), otherwise
	// Reject(). Only a single function must be called and every further call
	// causes a panic.
	ProposalResponder struct {
		c *Client                   // back-reference for account generation in Accept
		r *client.ProposalResponder // wrapped ProposalResponder
	}
)

// Handle implements the client.ProposalHandler interface by converting the
// passed types from the go-perun/client package into their local conterparts
// and then calling the prnm.ProposalHandler.
func (h *proposalHandler) Handle(_prop *client.ChannelProposal, _resp *client.ProposalResponder) {
	// Security Note: we don't check the remote nonce or channel participant. If
	// this code were to evolve to production grade, this needs to be taken care
	// of. In this case, at least the Nonce should be part of the ChannelProposal
	// struct, as is the case for the client.ChannelProposal.
	prop := &ChannelProposal{
		PeerPerunID:       &Address{*(_prop.PeerAddrs[0]).(*ethwallet.Address)},
		ChallengeDuration: int64(_prop.ChallengeDuration),
		InitBals:          &BigInts{_prop.InitBals.Balances[0]},
	}
	resp := &ProposalResponder{c: h.c, r: _resp}
	h.h.Handle(prop, resp)
}

// Accept lets the user signal that they want to accept the channel proposal.
// Returns the newly created channel controller if the channel was successfully
// created and funded. Panics if the proposal was already accepted or rejected.
//
// After the channel got successfully created, the user is required to start the
// update handler with PaymentChannel.HandleUpdates(UpdateHandler) and to start
// the channel watcher with PaymentChannel.Watch() on the returned channel
// controller.
//
// It is important that the passed context does not cancel before twice the
// ChallengeDuration has passed (at least for real blockchain backends with wall
// time), or the channel cannot be settled if a peer times out funding.
func (r *ProposalResponder) Accept(ctx *Context) (*PaymentChannel, error) {
	ch, err := r.r.Accept(ctx.ctx, client.ProposalAcc{
		// Generate new account as channel participant.
		Participant: r.c.w.NewAccount().Address(),
	})
	return &PaymentChannel{ch}, err
}

// Reject lets the user signal that they reject the channel proposal.
// Returns whether the rejection message was successfully sent. Panics if the
// proposal was already accepted or rejected.
func (r *ProposalResponder) Reject(ctx *Context, reason string) error {
	return r.r.Reject(ctx.ctx, reason)
}

// Used as upper (exclusive) limit when generating a random uint256. Equals 2^256.
var limitUint256 = new(big.Int).Lsh(big.NewInt(1), 256)

// nonce generates a cryptographically secure random value in the range [0, 2^256)
func nonce() *big.Int {
	val, err := rand.Int(rand.Reader, limitUint256)
	if err != nil {
		log.Panic("Could not create nonce")
	}
	return val
}
