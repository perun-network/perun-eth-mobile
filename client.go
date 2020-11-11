// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/backend/ethereum/wallet/keystore"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/go-perun/client"
	"perun.network/go-perun/log"
	"perun.network/go-perun/pkg/sortedkv/leveldb"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire/net"
	"perun.network/go-perun/wire/net/simple"
)

type (
	// Client is a state channel client. It is the central controller to interact
	// with a state channel network. It can be used to propose channels to other
	// channel network peers.
	// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client
	Client struct {
		cfg *Config

		ethClient *ethclient.Client
		client    *client.Client
		persister *keyvalue.PersistRestorer

		wallet  *keystore.Wallet
		onChain wallet.Account

		dialer *simple.Dialer
		bus    *net.Bus
	}

	// NewChannelCallback wraps a `func(*PaymentChannel)`
	// function pointer for the `Client.OnNewChannel` callback.
	NewChannelCallback interface {
		OnNew(*PaymentChannel)
	}
)

// NewClient sets up a new Client with configuration `cfg`.
// The Client:
//  - imports the keystore and unlocks the account
//  - listens on IP:port
//  - connects to the eth node
//  - in case either the Adjudicator and AssetHolder of the `cfg` are nil, it
//    deploys needed contract. There is currently no check that the
//    correct bytecode is deployed to the given addresses if they are
//    not nil.
//  - sets the `cfg`s Adjudicator and AssetHolder to the deployed contracts
//    addresses in case they were deployed.
func NewClient(ctx *Context, cfg *Config, w *Wallet) (*Client, error) {
	endpoint := fmt.Sprintf("%s:%d", cfg.IP, cfg.Port)
	listener, err := simple.NewTCPListener(endpoint)
	if err != nil {
		return nil, errors.WithMessagef(err, "listening on %s", endpoint)
	}
	dialer := simple.NewTCPDialer(time.Second * 15)
	ethClient, err := ethclient.Dial(cfg.ETHNodeURL)
	if err != nil {
		return nil, errors.WithMessage(err, "connecting to ethereum node")
	}

	acc, err := w.unlock(*cfg.Address)
	if err != nil {
		return nil, errors.WithMessage(err, "finding account")
	}
	cb := ethchannel.NewContractBackend(ethClient, keystore.NewTransactor(*w.w))
	if err := setupContracts(ctx.ctx, cb, acc.Account, cfg); err != nil {
		return nil, errors.WithMessage(err, "setting up contracts")
	}

	bus := net.NewBus(acc, dialer)
	adjudicator := ethchannel.NewAdjudicator(cb, common.Address(cfg.Adjudicator.addr), acc.Account.Address, acc.Account)
	accs := map[ethchannel.Asset]accounts.Account{cfg.AssetHolder.addr: acc.Account}
	depositor := new(ethchannel.ETHDepositor)
	deps := map[ethchannel.Asset]ethchannel.Depositor{cfg.AssetHolder.addr: depositor}

	funder := ethchannel.NewFunder(cb, accs, deps)
	c, err := client.New(acc.Address(), bus, funder, adjudicator, w.w)
	if err != nil {
		return nil, errors.WithMessage(err, "creating client")
	}
	go bus.Listen(listener)

	return &Client{cfg: cfg, ethClient: ethClient,
		client:    c,
		persister: nil,
		wallet:    w.w,
		onChain:   acc,
		dialer:    dialer,
		bus:       bus}, nil
}

// Close closes the client and its PersistRestorer to synchronize the database.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Channel.Close
// ref https://pkg.go.dev/perun.network/go-perun/channel/persistence/keyvalue?tab=doc#PersistRestorer.Close
func (c *Client) Close() error {
	if err := c.client.Close(); err != nil {
		return errors.WithMessage(err, "closing client")
	}
	if err := c.bus.Close(); err != nil {
		return errors.WithMessage(err, "closing bus")
	}
	if c.persister != nil {
		return errors.WithMessage(c.persister.Close(), "closing persister")
	}
	return nil
}

// Handle is the handler routine for channel proposals and channel updates. It
// must only be started at most once by the user.
// Incoming proposals and updates are forwarded to the passed handlers.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.Handle
func (c *Client) Handle(ph ProposalHandler, uh UpdateHandler) {
	c.client.Handle(&proposalHandler{c: c, h: ph}, &updateHandler{h: uh})
}

// OnNewChannel sets a handler to be called whenever a new channel is created
// or restored. Only one such handler can be set at a time, and repeated calls
// to this function will overwrite the currently existing handler. This
// function may be safely called at any time.
// Start the watcher routine here, if needed.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.OnNewChannel
func (c *Client) OnNewChannel(callback NewChannelCallback) {
	c.client.OnNewChannel(func(ch *client.Channel) {
		callback.OnNew(&PaymentChannel{ch})
	})
}

// EnablePersistence loads or creates a levelDB database at the given `dbPath`
// and tries to restore all channels from it.
// After this function was successfully called, all changes to the Client are
// saved to the database.
// This function is not thread safe.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.EnablePersistence
func (c *Client) EnablePersistence(dbPath string) (err error) {
	var db *leveldb.Database

	db, err = leveldb.LoadDatabase(dbPath)
	if err != nil {
		return errors.WithMessage(err, "creating/loading database")
	}
	c.persister = keyvalue.NewPersistRestorer(db)
	c.client.EnablePersistence(c.persister)
	return nil
}

// Restore restores all channels from persistence. Channels are restored in
// parallel. Newly restored channels should be acquired through the
// OnNewChannel callback.
// Note that connections are currently established serially, so allow for
// enough time in the passed context.
// ref https://pkg.go.dev/perun.network/go-perun/client?tab=doc#Client.Restore
func (c *Client) Restore(ctx *Context) error {
	if c.persister == nil {
		return errors.New("persistence not enabled")
	}
	return c.client.Restore(ctx.ctx)
}

// AddPeer adds a new peer to the client. Must be called before proposing
// a new channel with said peer. Wraps go-perun/peer/net/Dialer.Register.
// ref https://pkg.go.dev/perun.network/go-perun/peer/net?tab=doc#Dialer.Register
func (c *Client) AddPeer(perunID *Address, host string, port int) {
	c.dialer.Register((*ethwallet.Address)(&perunID.addr), fmt.Sprintf("%s:%d", host, port))
}

// setupContracts checks which contracts of the `cfg` are nil and deploys them
// to the blockchain. Writes the addresses of the deployed contracts back to
// the `cfg` struct.
func setupContracts(ctx context.Context, cb ethchannel.ContractBackend, deployer accounts.Account, cfg *Config) error {
	if cfg.Adjudicator == nil {
		adjudicator, err := ethchannel.DeployAdjudicator(ctx, cb, deployer)
		if err != nil {
			return errors.WithMessage(err, "deploying adjudicator")
		}
		cfg.Adjudicator = &Address{ethwallet.Address(adjudicator)}
	}
	if cfg.AssetHolder == nil {
		assetHolder, err := ethchannel.DeployETHAssetholder(ctx, cb, common.Address(cfg.Adjudicator.addr), deployer)
		if err != nil {
			return errors.WithMessage(err, "deploying eth assetHolder")
		}
		cfg.AssetHolder = &Address{ethwallet.Address(assetHolder)}
	}
	// The deployment itself is already logged in the `DeployX` methods
	log.WithFields(log.Fields{"adjudicator": cfg.Adjudicator.ToHex(), "assetHolder": cfg.AssetHolder.ToHex()}).Debugf("Set contracts")
	return nil
}

// OnChainBalance returns the on-chain balance for `address` in Wei.
func (c *Client) OnChainBalance(ctx *Context, address *Address) (*BigInt, error) {
	bal, err := c.ethClient.BalanceAt(ctx.ctx, common.Address(address.addr), nil)
	return &BigInt{bal}, err
}
