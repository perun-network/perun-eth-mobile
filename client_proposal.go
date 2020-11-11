// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/pkg/errors"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
	"perun.network/go-perun/wire"
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
//
// The remote peer must have been added to the Client via AddPeer prior
// to the call to ProposeChannel. Should the connected peer have a different
// `perunID` than the one given in AddPeer, an error in the form of
// "Dialed impersonator" will be thrown.
func (c *Client) ProposeChannel(
	ctx *Context,
	perunID *Address,
	challengeDuration int64,
	initialBals *BigInts,
) (*PaymentChannel, error) {
	alloc := &channel.Allocation{
		Assets:   []channel.Asset{(*ethwallet.Address)(&c.cfg.AssetHolder.addr)},
		Balances: [][]channel.Bal{initialBals.values},
	}
	prop, err := client.NewLedgerChannelProposal(
		uint64(challengeDuration),
		c.wallet.NewAccount().Address(),
		alloc,
		[]wire.Address{c.onChain.Address(), (*ethwallet.Address)(&perunID.addr)},
		client.WithoutApp())
	if err != nil {
		return nil, err
	}
	_ch, err := c.client.ProposeChannel(ctx.ctx, prop)
	return &PaymentChannel{_ch}, err
}

type (
	// A ProposalHandler decides how to handle incoming channel proposals from
	// other channel network peers.
	ProposalHandler interface {
		// HandleProposal is the user callback called by the Client on an
		// incoming channel proposal.
		HandleProposal(*ChannelProposal, *ProposalResponder)
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
		Peer              *Address // The peer proposing the channel.
		ChallengeDuration int64    // Proposed challenge duration in case of disputes, in seconds.
		InitBals          *BigInts // Initial channel balances.
	}

	// A ProposalResponder lets the user respond to a channel proposal. If the
	// user wants to accept the proposal, they should call Accept(), otherwise
	// Reject(). Only a single function must be called and every further call
	// causes a panic.
	ProposalResponder struct {
		c *Client // back-reference for account generation in Accept
		p client.LedgerChannelProposal
		r *client.ProposalResponder // wrapped ProposalResponder
	}
)

// HandleProposal implements the client.ProposalHandler interface by converting the
// passed types from the go-perun/client package into their local conterparts
// and then calling the prnm.ProposalHandler.
func (h *proposalHandler) HandleProposal(_prop client.ChannelProposal, _resp *client.ProposalResponder) {
	ledgerProp, ok := _prop.(*client.LedgerChannelProposal)
	if !ok {
		// We can not reject here since there is no context available.
		log.Warn("Ignored sub-channel proposal")
		return
	}
	if err := checkProp(*ledgerProp); err != nil {
		log.Warn("Ignored proposal: ", err)
		return
	}
	// Security Note: we don't check the remote nonce or channel participant. If
	// this code were to evolve to production grade, this needs to be taken care
	// of. In this case, at least the Nonce should be part of the ChannelProposal
	// struct, as is the case for the client.ChannelProposal.
	prop := &ChannelProposal{
		Peer:              &Address{*(ledgerProp.Participant).(*ethwallet.Address)},
		ChallengeDuration: int64(ledgerProp.ChallengeDuration),
		InitBals:          &BigInts{ledgerProp.InitBals.Balances[0]},
	}
	resp := &ProposalResponder{c: h.c, p: *ledgerProp, r: _resp}
	h.h.HandleProposal(prop, resp)
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
	// Generate new account as channel participant.
	account := r.c.wallet.NewAccount().Address()
	acceptor := r.p.Accept(account, client.WithRandomNonce())
	ch, err := r.r.Accept(ctx.ctx, acceptor)
	return &PaymentChannel{ch}, err
}

// Reject lets the user signal that they reject the channel proposal.
// Returns whether the rejection message was successfully sent. Panics if the
// proposal was already accepted or rejected.
func (r *ProposalResponder) Reject(ctx *Context, reason string) error {
	return r.r.Reject(ctx.ctx, reason)
}

func checkProp(prop client.LedgerChannelProposal) error {
	switch {
	case len(prop.InitBals.Assets) != 1:
		return errors.New("only single-asset channels are supported")
	case !channel.IsNoApp(prop.App):
		return errors.New("only payment channels are supported")
	}
	return nil
}
