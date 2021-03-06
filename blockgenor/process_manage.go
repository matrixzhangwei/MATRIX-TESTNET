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
package blockgenor

import (
	"sync"

	"github.com/matrix/go-matrix/accounts/signhelper"
	"github.com/matrix/go-matrix/consensus"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/hd"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/reelection"
	"github.com/pkg/errors"
)

type ProcessManage struct {
	mu         sync.Mutex
	curNumber  uint64
	processMap map[uint64]*Process
	matrix     Backend
	hd         *hd.HD
	signHelper *signhelper.SignHelper
	bc         *core.BlockChain
	txPool     *core.TxPool
	reElection *reelection.ReElection
	engine     consensus.Engine
	dposEngine consensus.DPOSEngine
}

func NewProcessManage(matrix Backend) *ProcessManage {
	return &ProcessManage{
		curNumber:  0,
		processMap: make(map[uint64]*Process),
		matrix:     matrix,
		hd:         matrix.HD(),
		signHelper: matrix.SignHelper(),
		bc:         matrix.BlockChain(),
		txPool:     matrix.TxPool(),
		reElection: matrix.ReElection(),
		engine:     matrix.BlockChain().Engine(),
		dposEngine: matrix.BlockChain().DPOSEngine(),
	}
}

func (pm *ProcessManage) SetCurNumber(number uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.curNumber = number
	pm.fixProcessMap()
}

func (pm *ProcessManage) GetCurNumber() uint64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.curNumber
}

func (pm *ProcessManage) GetCurrentProcess() *Process {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.getProcess(pm.curNumber)
}

func (pm *ProcessManage) GetProcess(number uint64) (*Process, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if err := pm.isLegalNumber(number); err != nil {
		return nil, err
	}
	return pm.getProcess(number), nil
}

func (pm *ProcessManage) GetProcessAndPreProcess(number uint64) (*Process, *Process, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if err := pm.isLegalNumber(number); err != nil {
		return nil, nil, err
	}

	if number == 0 {
		return pm.getProcess(number), nil, nil
	} else {
		return pm.getProcess(number), pm.getProcess(number - 1), nil
	}
}

func (pm *ProcessManage) fixProcessMap() {
	if pm.curNumber == 0 {
		return
	}

	if len(pm.processMap) == 0 {
		return
	}

	log.INFO(pm.logExtraInfo(), "PM 开始修正map, process数量", len(pm.processMap), "修复高度", pm.curNumber)

	delKeys := make([]uint64, 0)
	for key, process := range pm.processMap {
		if key < pm.curNumber-1 {
			process.Close()
			delKeys = append(delKeys, key)
		}
	}

	for _, delKey := range delKeys {
		delete(pm.processMap, delKey)
	}

	log.INFO(pm.logExtraInfo(), "PM 结束修正map, process数量", len(pm.processMap))
}

func (pm *ProcessManage) isLegalNumber(number uint64) error {
	var minNumber uint64
	if pm.curNumber < 1 {
		minNumber = 0
	} else {
		minNumber = pm.curNumber - 1
	}

	if number < minNumber {
		return errors.Errorf("number(%d) is less than current number(%d)", number, pm.curNumber)
	}

	if number > pm.curNumber+2 {
		return errors.Errorf("number(%d) is too big than current number(%d)", number, pm.curNumber)
	}

	return nil
}

func (pm *ProcessManage) getProcess(number uint64) *Process {
	process, OK := pm.processMap[number]
	if OK == false {
		log.INFO(pm.logExtraInfo(), "PM 创建process，高度", number)
		process = newProcess(number, pm)
		pm.processMap[number] = process
	}

	return process
}

func (pm *ProcessManage) logExtraInfo() string {
	return "区块生成"
}
