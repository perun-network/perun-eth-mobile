// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm_test

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkgtest "perun.network/go-perun/pkg/test"

	prnm "github.com/perun-network/perun-eth-mobile"
)

func TestBigInt(t *testing.T) {
	rng := pkgtest.Prng(t)
	max := new(big.Int).Lsh(big.NewInt(1), 4)

	t.Run("add", func(t *testing.T) {
		for i := int64(0); i < 10; i++ {
			for j := int64(0); j < 10; j++ {
				x, err := rand.Int(rng, max)
				require.NoError(t, err)
				y, err := rand.Int(rng, max)
				require.NoError(t, err)

				a := prnm.NewBigIntFromBytes(x.Bytes())
				b := prnm.NewBigIntFromBytes(y.Bytes())
				c := new(big.Int).Add(x, y)

				assert.Equal(t, c, a.Add(b).BigInt())
			}
		}
	})
	t.Run("sub", func(t *testing.T) {
		for i := int64(0); i < 10; i++ {
			for j := int64(0); j < 10; j++ {
				x, err := rand.Int(rng, max)
				require.NoError(t, err)
				y, err := rand.Int(rng, max)
				require.NoError(t, err)

				a := prnm.NewBigIntFromBytes(x.Bytes())
				b := prnm.NewBigIntFromBytes(y.Bytes())
				c := new(big.Int).Sub(x, y)

				assert.Equal(t, c, a.Sub(b).BigInt())
			}
		}
	})
	t.Run("is-within", func(t *testing.T) {
		for i := int64(0); i < 10; i++ {
			for j := int64(0); j < 10; j++ {
				x, err := rand.Int(rng, max)
				require.NoError(t, err)
				y, err := rand.Int(rng, max)
				require.NoError(t, err)
				d, err := rand.Int(rng, max)
				require.NoError(t, err)

				a := prnm.NewBigIntFromBytes(x.Bytes())
				b := prnm.NewBigIntFromBytes(y.Bytes())
				c := prnm.NewBigIntFromBytes(d.Bytes())

				require.Equal(t, new(big.Int).Sub(x, y).CmpAbs(d) <= 0, a.IsWithin(b, c))
				require.Equal(t, new(big.Int).Sub(x, y).CmpAbs(d) <= 0, b.IsWithin(a, c))
			}
		}
	})
}
