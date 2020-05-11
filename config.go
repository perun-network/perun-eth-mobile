// Copyright (c) 2020 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package prnm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"

	"perun.network/go-perun/apps/payment"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/log"
	plogrus "perun.network/go-perun/log/logrus"
)

// Config complete configuration needed to operate the Client.
type Config struct {
	// Name to be used in state channels.
	Alias string
	// Address will be unlocked in the KeyStore with password
	KeyStorePath, Password string
	Address                *Address // OnChain address also calles PerunID.
	DatabasePath           string   // Path to the database file.
	ETHNodeURL             string   // URL of the ETH node. Example: ws://127.0.0.1:8545
	IP                     string   // Ip to listen on.
	Port                   uint16   // Port to listen on.
}

// assetAddr Address of the Asset to be used.
var assetAddr = common.HexToAddress("0xb051EAD0C6CC2f568166F8fEC4f07511B88678bA")

// adjudicatorAdr Address of the Adjudicator to be used.
var adjudicatorAdr = common.HexToAddress("0xDc4A7e107aD6dBDA1870df34d70B51796BBd1335")

// Random address
var appDef = common.HexToAddress("0x0583849a3C5F37aEfAb8cCcA303f9229AdF5A32a")

// NewConfig creates a new configuration
func NewConfig(alias, keyStorePath, password string, address *Address, databasePath, ETHNodeURL, ip string, port int) *Config {
	return &Config{
		Alias:        alias,
		KeyStorePath: keyStorePath,
		Password:     password,
		Address:      address,
		DatabasePath: databasePath,
		ETHNodeURL:   ETHNodeURL,
		IP:           ip,
		Port:         uint16(port),
	}
}

var logger *logrus.Logger

func init() {
	payment.SetAppDef((*ethwallet.Address)(&appDef))

	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	log.Set(plogrus.FromLogrus(logger))
}

// SetLogLevel takes a logrus.Level argument.
// See https://godoc.org/github.com/sirupsen/logrus#Level
func SetLogLevel(level int) {
	logger.SetLevel(logrus.Level(level))
}
