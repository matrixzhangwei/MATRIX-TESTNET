// Copyright 2018 The MATRIX Authors as well as Copyright 2014-2017 The go-ethereum Authors
// This file is consisted of the MATRIX library and part of the go-ethereum library.
//
// The MATRIX-ethereum library is free software: you can redistribute it and/or modify it under the terms of the MIT License.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, 
//and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject tothe following conditions:
//
//The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, 
//WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISINGFROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
//OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package miner implements Matrix block creation and mining.
package miner

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/consensus"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/core/state"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/event"
	"github.com/matrix/go-matrix/hd"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/params"
)

const (
	ModuleMiner = "Miner Miner"
)

var (
	invalidParameter          = errors.New("Parameter is invalid!")
	currentNotMiner           = errors.New("Current Not Miner")
	smallThanCurrentHeightt   = errors.New("Small than current height")
	LimitBroadcastRole        = errors.New("change Broadcast to others")
	currentRoleIsNotBroadcast = errors.New("current Role Is Not Broadcast")
	difficultyIsZero          = errors.New("difficulty Is Zero")
)

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mux *event.TypeMux

	worker *worker

	coinbase common.Address
	bc       *core.BlockChain
	engine   consensus.Engine

	canStart    int32 // can start indicates whether we can start the mining operation
	shouldStart int32 // should start indicates whether we should start after sync
}

func (s *Miner) Getworker() *worker { return s.worker }

func New(bc *core.BlockChain, config *params.ChainConfig, mux *event.TypeMux, engine consensus.Engine, dposEngine consensus.DPOSEngine, hd *hd.HD) (*Miner, error) {
	miner := &Miner{
		mux:    mux,
		engine: engine,

		canStart: 1,
	}
	var err error
	miner.worker, err = newWorker(config, engine, bc, dposEngine, mux, hd)
	if err != nil {
		log.ERROR(ModuleMiner, "创建work", "失败")
		return miner, err
	}
	miner.Register(NewCpuAgent(bc, engine))
	//go miner.update()
	log.INFO(ModuleMiner, "创建miner", "成功")
	return miner, nil
}

//Start
func (self *Miner) Start() {
	atomic.StoreInt32(&self.shouldStart, 1)

	if atomic.LoadInt32(&self.canStart) == 0 {
		log.Info("Network syncing, will start miner afterwards")
		return
	}
}

func (self *Miner) Stop() {
	// todo:

	//self.worker.Stop()
	atomic.StoreInt32(&self.shouldStart, 0)

}

func (self *Miner) Register(agent Agent) {
	if self.Mining() {
		agent.Start()
	}
	self.worker.Register(agent)
}

func (self *Miner) Unregister(agent Agent) {
	self.worker.Unregister(agent)
}

func (self *Miner) Mining() bool {
	return atomic.LoadInt32(&self.Getworker().mining) > 0
}

func (self *Miner) HashRate() (tot int64) {
	if pow, ok := self.engine.(consensus.PoW); ok {
		tot += int64(pow.Hashrate())
	}
	// do we care this might race? is it worth we're rewriting some
	// aspects of the worker/locking up agents so we can get an accurate
	// hashrate?
	for agent := range self.worker.agents {
		if _, ok := agent.(*CpuAgent); !ok {
			tot += agent.GetHashRate()
		}
	}
	return
}

func (self *Miner) SetExtra(extra []byte) error {
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("Extra exceeds max length. %d > %v", len(extra), params.MaximumExtraDataSize)
	}
	self.worker.setExtra(extra)
	return nil
}

// Pending returns the currently pending block and associated state.
func (self *Miner) Pending() (*types.Block, *state.StateDB) {
	return self.worker.pending()
}

/*
// PendingBlock returns the currently pending block.
//
// Note, to access both the pending block and the pending state
// simultaneously, please use Pending(), as the pending state can
// change between multiple method calls
*/
func (self *Miner) PendingBlock() *types.Block {
	return self.worker.pendingBlock()
}
