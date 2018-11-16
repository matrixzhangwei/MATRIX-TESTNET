//1542342259.7689953
//1542342248.6244626
//1542342236.6080987
//1542342225.5118246
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

package matrixwork

import (
	"encoding/json"
	"errors"
	"github.com/matrix/go-matrix/params"
	"math/big"
	"time"

	"github.com/matrix/go-matrix/ca"
	"github.com/matrix/go-matrix/depoistInfo"
	"github.com/matrix/go-matrix/mc"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/core/state"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/core/vm"
	"github.com/matrix/go-matrix/event"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/params/man"
)

var packagename = "matrixwork"

// Work is the workers current environment and holds
// all of the current state information
type Work struct {
	config *params.ChainConfig
	signer types.Signer

	State *state.StateDB // apply state changes here
	//ancestors *set.Set       // ancestor set (used for checking uncle parent validity)
	//family    *set.Set       // family set (used for checking uncle invalidity)
	//uncles    *set.Set       // uncle set
	tcount  int           // tx count in cycle
	gasPool *core.GasPool // available gas used to pack transactions

	Block *types.Block // the new block

	header   *types.Header
	txs      []*types.Transaction
	Receipts []*types.Receipt

	createdAt time.Time
}

func NewWork(config *params.ChainConfig, bc *core.BlockChain, gasPool *core.GasPool, header *types.Header) (*Work, error) {

	Work := &Work{
		config:  config,
		signer:  types.NewEIP155Signer(config.ChainId),
		gasPool: gasPool,
		header:  header,
	}
	var err error

	Work.State, err = bc.StateAt(bc.GetBlockByHash(header.ParentHash).Root())

	if err != nil {
		return nil, err
	}
	return Work, nil
}

func (env *Work) commitTransactions(mux *event.TypeMux, txs *types.TransactionsByPriceAndNonce, bc *core.BlockChain, coinbase common.Address) (listN []uint32, retTxs []*types.Transaction) {
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(env.header.GasLimit)
	}

	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}

		if tx.N == nil {
			log.Info("===========tx.N is nil")
			txs.Pop()
			continue
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !env.config.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", env.config.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.State.Prepare(tx.Hash(), common.Hash{}, env.tcount)
		err, logs := env.commitTransaction(tx, bc, coinbase, env.gasPool)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			//==========hezi===================
			if tx.N != nil {
				listN = append(listN, tx.N[0])
				retTxs = append(retTxs, tx)
				//log.INFO("=======", "commitTransactions:len(listN)", len(listN))
			}
			//==================================
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	if len(coalescedLogs) > 0 || env.tcount > 0 {
		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		go func(logs []*types.Log, tcount int) {
			if len(logs) > 0 {
				mux.Post(core.PendingLogsEvent{Logs: logs})
			}
			if tcount > 0 {
				mux.Post(core.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}
	return listN, retTxs
}

func (env *Work) commitTransaction(tx *types.Transaction, bc *core.BlockChain, coinbase common.Address, gp *core.GasPool) (error, []*types.Log) {
	snap := env.State.Snapshot()

	receipt, _, err := core.ApplyTransaction(env.config, bc, &coinbase, gp, env.State, env.header, tx, &env.header.GasUsed, vm.Config{})
	if err != nil {
		log.Info("*************", "ApplyTransaction:err", err)
		env.State.RevertToSnapshot(snap)
		return err, nil
	}
	env.txs = append(env.txs, tx)
	env.Receipts = append(env.Receipts, receipt)

	return nil, receipt.Logs
}

//Leader
var lostCnt int = 0

type retStruct struct {
	no  []uint32
	txs []*types.Transaction
}

func (self *Work) ProcessTransactions(mux *event.TypeMux, tp *core.TxPool, bc *core.BlockChain) ([]uint32, []*types.Transaction) {

	//ret := make(chan *retStruct, 1)
	//tm := time.NewTimer(time.Second * 5)
	//go func(ret1 chan *retStruct) {
	//	log.ERROR("Tx", "LostCnt", lostCnt)
	//	log.Info("===========", "ProcessTransactions:befor", 0)
	//	pending, err := tp.Pending()
	//	log.Info("===========", "ProcessTransactions:after", 1)
	//	if err != nil {
	//		log.Error("Failed to fetch pending transactions", "err", err)
	//		ret1 <- &retStruct{nil, nil}
	//	}
	//	log.INFO("===========", "ProcessTransactions:pending:", len(pending))
	//	txs := types.NewTransactionsByPriceAndNonce(self.signer, pending)
	//	log.INFO("===========", "ProcessTransactions:txs:", txs)
	//	a, b := self.commitTransactions(mux, txs, bc, common.Address{})
	//	ret1 <- &retStruct{a, b}
	//}(ret)
	//select {
	//case val := <-ret:
	//	return val.no, val.txs
	//case <-tm.C:
	//	log.ERROR("Tx", "Tx Proc TimeOut", lostCnt)
	//	lostCnt++
	//	return nil, nil
	//}

	pending, err := tp.Pending()
	if err != nil {
		log.Error("Failed to fetch pending transactions", "err", err)
		return nil, nil
	}
	log.INFO("===========", "ProcessTransactions:pending:", len(pending))
	txs := types.NewTransactionsByPriceAndNonce(self.signer, pending)
	//log.INFO("===========", "ProcessTransactions:txs:", txs)
	return self.commitTransactions(mux, txs, bc, common.Address{})
}

/*//==============================================================================//
//Leader
func (self *Work) ProcessTransactions(mux *event.TypeMux, tp *core.TxPool, bc *core.BlockChain) ([]uint32, []*types.Transaction) {
	pending, err := tp.Pending()
	if err != nil {
		log.Error("Failed to fetch pending transactions", "err", err)
		return nil, nil
	}
	log.INFO("===========", "ProcessTransactions:pending:", len(pending))
	txs := types.NewTransactionsByPriceAndNonce(self.signer, pending)
	log.INFO("===========", "ProcessTransactions:txs:", txs)
	return self.commitTransactions(mux, txs, bc, common.Address{})

}*/

//Broadcast
func (self *Work) ProcessBroadcastTransactions(mux *event.TypeMux, txs []*types.Transaction, bc *core.BlockChain) {

	for _, tx := range txs {
		//log.INFO("========","ProcessBroadcastTransactions:tx",tx)
		//log.INFO("========","ProcessBroadcastTransactions:tx.price",tx.GasPrice())
		self.commitTransaction(tx, bc, common.Address{}, nil)
	}

	return
}

func (env *Work) ConsensusTransactions(mux *event.TypeMux, txs []*types.Transaction, bc *core.BlockChain) error {
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(env.header.GasLimit)
	}
	var coalescedLogs []*types.Log

	for _, tx := range txs {
		// If we don't have enough gas for any further transactions then we're done
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			return errors.New("Not enough gas for further transactions")
		}

		// Start executing the transaction
		env.State.Prepare(tx.Hash(), common.Hash{}, env.tcount)
		err, logs := env.commitTransaction(tx, bc, common.Address{}, env.gasPool)
		if err == nil {
			//log.Info("=========","ConsensusTransactions:tx.N",tx.N)
			env.tcount++
			coalescedLogs = append(coalescedLogs, logs...)
		} else {
			return err
		}
	}

	if len(coalescedLogs) > 0 || env.tcount > 0 {
		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		go func(logs []*types.Log, tcount int) {
			if len(logs) > 0 {
				mux.Post(core.PendingLogsEvent{Logs: logs})
			}
			if tcount > 0 {
				mux.Post(core.PendingStateEvent{})
			}
		}(cpy, env.tcount)
	}

	return nil
}

func (env *Work) GetUpTimeAccounts(num uint64) ([]common.Address, error) {

	log.INFO(packagename, "获取所有参与uptime点名高度", num)

	upTimeAccounts := make([]common.Address, 0)

	minerNum := num - (num % common.GetBroadcastInterval()) - man.MinerTopologyGenerateUpTime
	log.INFO(packagename, "candidate miners' uptime height", minerNum)
	ans, err := ca.GetElectedByHeightAndRole(big.NewInt(int64(minerNum)), common.RoleMiner)
	if err != nil {
		return nil, err
	}

	log.INFO("getUpTimeAccounts", "ans", ans)
	for _, v := range ans {
		upTimeAccounts = append(upTimeAccounts, v.Address)
		log.INFO("v.Address", "v.Address", v.Address)
	}
	validatorNum := num - (num % common.GetBroadcastInterval()) - man.VerifyTopologyGenerateUpTime
	log.INFO(packagename, "参选验证节点uptime高度", validatorNum)
	ans1, err := ca.GetElectedByHeightAndRole(big.NewInt(int64(validatorNum)), common.RoleValidator)
	if err != nil {
		return upTimeAccounts, err
	}
	for _, v := range ans1 {
		upTimeAccounts = append(upTimeAccounts, v.Address)
		log.INFO("v.Address", "v.Address", v.Address)
	}
	log.INFO(packagename, "obtain all uptime accounts", upTimeAccounts)
	return upTimeAccounts, nil
}

func (env *Work) GetUpTimeData(num uint64) (map[common.Address]uint32, map[common.Address][]byte, error) {

	log.INFO(packagename, "obtain all heartbeat transactions", "")
	heatBeatUnmarshallMMap, error := core.GetBroadcastTxs(new(big.Int).SetUint64(num), mc.Heartbeat)
	if nil != error {
		log.ERROR(packagename, "获取主动心跳交易错误", error)
		return nil, nil, error
	}

	calltherollUnmarshall, error := core.GetBroadcastTxs(new(big.Int).SetUint64(num), mc.CallTheRoll)
	if nil != error {
		log.ERROR(packagename, "获取点名心跳交易错误", error)
		return nil, nil, error
	}
	calltherollMap := make(map[common.Address]uint32, 0)
	for _, v := range calltherollUnmarshall {
		error := json.Unmarshal(v, &calltherollMap)
		if nil != error {
			log.ERROR(packagename, "序列化主动心跳交易错误", error)
			return nil, nil, error
		}
	}
	return calltherollMap, heatBeatUnmarshallMMap, nil
}

func (env *Work) HandleUpTime(state *state.StateDB, accounts []common.Address, calltherollRspAccounts map[common.Address]uint32, heatBeatAccounts map[common.Address][]byte, blockNum uint64, bc *core.BlockChain) error {
	var blockHash common.Hash
	HeatBeatReqAccounts := make([]common.Address, 0)
	HeartBeatMap := make(map[common.Address]bool, 0)
	blockNumRem := blockNum % common.GetBroadcastInterval()

	//subVal latest broadcast block, eg, if the current block height is 198 or 101, then subVal is 100
	subVal := blockNum - blockNumRem
	//subVal latest broadcast block, eg, if the current block height is 198 or 101, then subVal is 100
	subVal = subVal
	if blockNum < common.GetBroadcastInterval() { //当前区块小于100说明是100区块内 (下面的if else是为了应对中途加入的参选节点)
		blockHash = bc.GetBlockByNumber(0).Hash() //hash of genesis block
	} else {
		blockHash = bc.GetBlockByNumber(subVal).Hash() //获取最近的广播区块的hash
	}
	// todo: remove
	//blockHash = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3e4")
	broadcastBlock := blockHash.Big()
	val := broadcastBlock.Uint64() % ((common.GetBroadcastInterval()) - 1)

	for _, v := range accounts {
		currentAcc := v.Big() //YY TODO 这里应该是广播账户。后期需要修改
		ret := currentAcc.Uint64() % (common.GetBroadcastInterval() - 1)
		if ret == val {
			HeatBeatReqAccounts = append(HeatBeatReqAccounts, v)
			if _, ok := heatBeatAccounts[v]; ok {
				HeartBeatMap[v] = true
			} else {
				HeartBeatMap[v] = false

			}
			log.Info(packagename, "计算主动心跳的账户", v, "心跳状态", HeartBeatMap[v])
		}
	}

	var upTime uint64
	for _, account := range accounts {
		onlineBlockNum, ok := calltherollRspAccounts[account]
		if ok { //被点名,使用点名的uptime
			upTime = uint64(onlineBlockNum)
			log.INFO(packagename, "点名账号", account, "uptime", upTime)

		} else { //没被点名，没有主动上报，则为最大值，
			if v, ok := HeartBeatMap[account]; ok { //有主动上报
				if v {
					upTime = common.GetBroadcastInterval() - 2
					log.INFO(packagename, "没被点名，有主动上报有响应", account, "uptime", upTime)
				} else {
					upTime = 0
					log.INFO(packagename, "没被点名，有主动上报无响应", account, "uptime", upTime)
				}
			} else { //没被点名和主动上报
				upTime = common.GetBroadcastInterval() - 2
				log.INFO(packagename, "没被点名，没要求主动上报", account, "uptime", upTime)

			}
		}
		// todo: add
		depoistInfo.AddOnlineTime(state, account, new(big.Int).SetUint64(upTime))
		if read, err := depoistInfo.GetOnlineTime(state, account); nil == err {
			log.INFO(packagename, "read the state tree", account, "uptime", read)
		}

	}

	return nil
}
