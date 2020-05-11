// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/client"
	"perun.network/go-perun/peer/net"
	"perun.network/go-perun/wallet"
)

type Client struct {
	cfg *Config

	client *client.Client

	w       *ethwallet.Wallet
	onChain wallet.Account

	dialer *net.Dialer
}

func NewClient(cfg *Config) (*Client, error) {
	w, acc, err := importAccount(cfg.KeyStorePath, "0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9")
	if err != nil {
		return nil, errors.WithMessage(err, "importing account")
	}
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
	cb := ethchannel.NewContractBackend(node, w.Ks, &acc.Account)
	adjudicator := ethchannel.NewAdjudicator(cb, adjudicatorAdr, acc.Account.Address)
	funder := ethchannel.NewETHFunder(cb, assetAddr)

	client := client.New(acc, dialer, funder, adjudicator, w)
	go client.Listen(listener)
	return &Client{cfg: cfg, client: client, w: w, onChain: acc, dialer: dialer}, nil
}

func importAccount(walletPath, secret string) (*ethwallet.Wallet, *ethwallet.Account, error) {
	ks := keystore.NewKeyStore(walletPath, 2, 1)
	sk, err := crypto.HexToECDSA(secret[2:])
	if err != nil {
		return nil, nil, errors.WithMessage(err, "decoding secret key")
	}
	var ethAcc accounts.Account
	addr := crypto.PubkeyToAddress(sk.PublicKey)
	if ethAcc, err = ks.Find(accounts.Account{Address: addr}); err != nil {
		ethAcc, err = ks.ImportECDSA(sk, "")
		if err != nil && errors.Cause(err).Error() != "account already exists" {
			return nil, nil, errors.WithMessage(err, "importing secret key")
		}
	}

	w, err := ethwallet.NewWallet(ks, "")
	if err != nil {
		return nil, nil, errors.WithMessage(err, "creating wallet")
	}

	wAcc := ethwallet.NewAccountFromEth(w, &ethAcc)
	acc, err := w.Unlock(wAcc.Address())
	return w, acc.(*ethwallet.Account), err
}

func (c *Client) AddPeer(perunID *Address, host string, port int) {
	c.dialer.Register((*ethwallet.Address)(&perunID.addr), fmt.Sprintf("%s:%d", host, port))
}
