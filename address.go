// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"perun.network/go-perun/backend/ethereum/wallet"
)

// Address wraps a go-perun ethereum Address.
type Address struct {
	addr wallet.Address
}

// NewAddressFromHex creates an Address from the given string. String must be in the form
// 0x32be343b94f860124dc5fee278fdcbd38c102d88
func NewAddressFromHex(str string) (*Address, error) {
	if len(str) != 42 || str[:2] != "0x" {
		return nil, errors.New("Address must be chars 40 hex strings prefixed with 0x")
	}
	bytes, err := hex.DecodeString(str[2:])
	if err != nil {
		return nil, errors.WithMessage(err, "parsing address as hexadecimal")
	}
	var a common.Address
	copy(a[:], bytes)
	return &Address{wallet.Address(a)}, nil
}

// ToHex returns the hexadecimal representation of the Address as string.
// Example: 0x32be343b94f860124dc5fee278fdcbd38c102d88
func (a *Address) ToHex() string {
	return "0x" + hex.EncodeToString(a.addr.Bytes())
}

// Addresses is a slice of Address'es
type Addresses struct {
	values []wallet.Address
}

// NewAddresses creates a new Addresses with the given length.
func NewAddresses(length int) *Addresses {
	return &Addresses{values: make([]wallet.Address, length)}
}

// Length returns the length of the Addresses slice.
func (as *Addresses) Length() int {
	return len(as.values)
}

// Get returns the element at the given index.
func (as *Addresses) Get(index int) (*Address, error) {
	if index < 0 || index >= len(as.values) {
		return nil, errors.New("get: index out of range")
	}
	return &Address{as.values[index]}, nil
}

// Set sets the element at the given index.
func (as *Addresses) Set(index int, value *Address) error {
	if index < 0 || index >= len(as.values) {
		return errors.New("set: index out of range")
	}
	as.values[index] = value.addr
	return nil
}
