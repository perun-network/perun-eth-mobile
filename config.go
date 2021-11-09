// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/sirupsen/logrus"

	"perun.network/go-perun/log"
	plogrus "perun.network/go-perun/log/logrus"
)

// Config complete configuration needed to operate the Client.
type Config struct {
	// Name to be used in state channels.
	Alias   string
	Address *Address // OnChain address and PerunID.
	// On-chain addresses of the Adjudicator and AssetHolder Contract.
	// In case any of them is nil, the Client will deploy the contract in its
	// NewClient constructor.
	Adjudicator, AssetHolder *Address
	ETHNodeURL               string // URL of the ETH node. Example: ws://127.0.0.1:8545
	IP                       string // Ip to listen on.
	Port                     uint16 // Port to listen on.
	// TxFinalityDepth how many blocks a Transaction needs to be included
	// in to be considered final.
	TxFinalityDepth uint64
}

// NewConfig creates a new configuration.
func NewConfig(alias string, address, adjudicator, assetHolder *Address, ETHNodeURL, ip string, port int, txFinalityDepth int) *Config {
	return &Config{
		Alias:           alias,
		Address:         address,
		Adjudicator:     adjudicator,
		AssetHolder:     assetHolder,
		ETHNodeURL:      ETHNodeURL,
		IP:              ip,
		Port:            uint16(port),
		TxFinalityDepth: uint64(txFinalityDepth),
	}
}

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	log.Set(plogrus.FromLogrus(logger))
}

// SetLogLevel takes a logrus.Level argument.
// See https://godoc.org/github.com/sirupsen/logrus#Level
func SetLogLevel(level int) {
	logger.SetLevel(logrus.Level(level))
}
