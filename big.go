// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"math/big"

	"github.com/pkg/errors"
)

// BigInt wraps a golang math/big.Int.
// All functions on BigInt have their equivalent in the documentation below.
// See https://golang.org/pkg/math/big/#Int
// If we do this as embedding, the functions are skipped because of the wrong return type.
type BigInt struct {
	i *big.Int
}

// NewBigIntFromBytes creates a BigInt from a byte slice.
func NewBigIntFromBytes(data []byte) *BigInt {
	return &BigInt{new(big.Int).SetBytes(data)}
}

// NewBigIntFromInt64 creates a BigInt from an int64.
func NewBigIntFromInt64(v int64) *BigInt {
	return &BigInt{new(big.Int).SetInt64(v)}
}

// NewBigIntFromString creates a BigInt by parsing a string.
// A prefix of "0b" or "0B" selects base 2, "0", "0o" or "0O" selects base 8,
// and "0x" or "0X" selects base 16. Otherwise, the selected base is 10 and no prefix is accepted.
// Read documentation of https://pkg.go.dev/math/big?tab=doc#Int.SetString for more details.
func NewBigIntFromString(data string) (*BigInt, error) {
	b, success := new(big.Int).SetString(data, 0)
	if !success {
		return nil, errors.New("invalid number string")
	}
	return &BigInt{b}, nil
}

// NewBigIntFromStringBase creates a BigInt by parsing a string containing a number of given base.
func NewBigIntFromStringBase(data string, base int) (*BigInt, error) {
	b, success := new(big.Int).SetString(data, base)
	if !success {
		return nil, errors.New("invalid number string")
	}
	return &BigInt{b}, nil
}

// Add returns the result of the receiver + x. Does not change the reveiver.
func (b *BigInt) Add(x *BigInt) *BigInt {
	return &BigInt{new(big.Int).Add(b.i, x.i)}
}

// Sub returns the result of the receiver - x. Does not change the reveiver.
func (b *BigInt) Sub(x *BigInt) *BigInt {
	return &BigInt{new(big.Int).Sub(b.i, x.i)}
}

// ToInt64 wraps math/big.Int.Int64
func (b *BigInt) ToInt64() int64 {
	return b.i.Int64()
}

// String wraps math/big.Int.String
func (b *BigInt) String() string {
	return b.i.String()
}

// StringBase wraps math/big.Int.Text
func (b *BigInt) StringBase(base int) string {
	return b.i.Text(base)
}

// ToBytesArray wraps math/big.Int.Bytes
func (b *BigInt) ToBytesArray() []byte {
	return b.i.Bytes()
}

// BigInt can not be called from Java, only here to improve reusability.
func (b *BigInt) BigInt() *big.Int {
	return b.i
}

// BigInts is a slice of BigInt's
type BigInts struct {
	values []*big.Int
}

// NewBigInts creates a new BitInts with the given length.
func NewBigInts(length int) *BigInts {
	return &BigInts{values: make([]*big.Int, length)}
}

// NewBalances creates a new BigInts of length two with the given values.
func NewBalances(first, second *BigInt) *BigInts {
	return &BigInts{values: []*big.Int{first.i, second.i}}
}

// Length returns the length of the BigInts slice.
func (bs *BigInts) Length() int {
	return len(bs.values)
}

// Get returns the element at the given index.
func (bs *BigInts) Get(index int) (*BigInt, error) {
	if index < 0 || index >= len(bs.values) {
		return nil, errors.New("get: index out of range")
	}
	return &BigInt{bs.values[index]}, nil
}

// Set sets the element at the given index.
func (bs *BigInts) Set(index int, value *BigInt) error {
	if index < 0 || index >= len(bs.values) {
		return errors.New("set: index out of range")
	}
	bs.values[index] = value.i
	return nil
}

// Data can not be called from Java, only here to improve reusability.
func (bs *BigInts) Data() []*big.Int {
	return bs.values
}
