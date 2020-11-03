// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/ethereum/go-ethereum/accounts"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/backend/ethereum/wallet/keystore"
)

// Wallet represents an ethereum wallet. It uses the go-ethereum keystore to
// store keys. Accessing the wallet is threadsafe, however you should not
// create two wallets from the same key directory.
// ref https://pkg.go.dev/perun.network/go-perun/backend/ethereum/wallet?tab=doc#Wallet
type Wallet struct {
	w        *keystore.Wallet
	password string
}

// NewWallet returns a new wallet with the given path and password.
func NewWallet(path, password string) (*Wallet, error) {
	// We use 2,1 as scrypt parameters here for development because on an Android phone
	// it is quite slow to use the standard parameters. Do not to this in production.
	ks := ethkeystore.NewKeyStore(path, 2, 1)
	w, err := keystore.NewWallet(ks, password)
	return &Wallet{w: w, password: password}, errors.WithMessage(err, "creating wallet")
}

// ImportAccount imports an Ethereum secret key into the Wallet and
// returns the corresponding Address of it. Secret key example:
// 0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9
// Accounts can safely be imported more than once.
func (w *Wallet) ImportAccount(secretKey string) (*Address, error) {
	if len(secretKey) != 66 || secretKey[:2] != "0x" {
		return nil, errors.New("Secret key must start with 0x and be 66 characters long")
	}
	sk, err := crypto.HexToECDSA(secretKey[2:])
	if err != nil {
		return nil, errors.WithMessage(err, "decoding secret key")
	}
	var ethAcc accounts.Account
	addr := crypto.PubkeyToAddress(sk.PublicKey)
	if ethAcc, err = w.w.Ks.Find(accounts.Account{Address: addr}); err != nil {
		ethAcc, err = w.w.Ks.ImportECDSA(sk, w.password)
		if err != nil && errors.Cause(err).Error() != "account already exists" {
			return nil, errors.WithMessage(err, "importing secret key")
		}
	}

	wAcc := keystore.NewAccountFromEth(w.w, &ethAcc)
	acc, err := w.w.Unlock(wAcc.Address())
	if err != nil {
		return nil, errors.WithMessage(err, "unlocking account")
	}
	return &Address{*(acc.Address()).(*ethwallet.Address)}, err
}

// CreateAccount returns the Address of a new randomly created Account.
// ref https://pkg.go.dev/perun.network/go-perun/backend/ethereum/wallet?tab=doc#Wallet.NewAccount
func (w *Wallet) CreateAccount() *Address {
	return &Address{ethwallet.Address(w.w.NewAccount().Account.Address)}
}

func (w *Wallet) unlock(a Address) (*keystore.Account, error) {
	acc, err := w.w.Unlock(&a.addr)
	return acc.(*keystore.Account), err
}
