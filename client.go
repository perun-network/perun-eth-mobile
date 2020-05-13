// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/client"
	"perun.network/go-perun/peer/net"
	"perun.network/go-perun/wallet"
)

// Client is a state channel client. It is the central controller to interact
// with a state channel network. It can be used to propose channels to other
// channel network peers.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client
type Client struct {
	cfg *Config

	client *client.Client

	w       *ethwallet.Wallet
	onChain wallet.Account

	dialer *net.Dialer
}

// NewClient sets up a new Client with configuration `cfg`.
// The Client:
//  - imports the keystore and unlocks the account
//  - listens on IP:port
//  - connects to the eth node
//  - sets up the connection to the contracts (asset holder/adjudicator)
func NewClient(cfg *Config, w *Wallet) (*Client, error) {
	endpoint := fmt.Sprintf("%s:%d", cfg.IP, cfg.Port)
	listener, err := net.NewTCPListener(endpoint)
	if err != nil {
		return nil, errors.WithMessagef(err, "listening on %s", endpoint)
	}
	dialer := net.NewTCPDialer(time.Second * 15)
	node, err := ethclient.Dial(cfg.ETHNodeURL)
	if err != nil {
		return nil, errors.WithMessage(err, "connecting to ethereum node")
	}
	acc, err := w.findAccount(*cfg.Address)
	if err != nil {
		return nil, errors.WithMessage(err, "finding account")
	}
	cb := ethchannel.NewContractBackend(node, w.w.Ks, &acc.Account)
	adjudicator := ethchannel.NewAdjudicator(cb, adjudicatorAdr, acc.Account.Address)
	funder := ethchannel.NewETHFunder(cb, assetAddr)

	client := client.New(acc, dialer, funder, adjudicator, w.w)
	go client.Listen(listener)
	return &Client{cfg: cfg, client: client, w: w.w, onChain: acc, dialer: dialer}, nil
}

// AddPeer adds a new peer to the client. Must be called before proposing
// a new channel with said peer. Wraps go-perun/peer/net/Dialer.Register.
// ref https://pkg.go.dev/perun.network/go-perun/peer/net?tab=doc#Dialer.Register
func (c *Client) AddPeer(perunID *Address, host string, port int) {
	c.dialer.Register((*ethwallet.Address)(&perunID.addr), fmt.Sprintf("%s:%d", host, port))
}
