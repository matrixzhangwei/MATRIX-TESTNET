//1542343146.5298045
//1542343135.0943298
//1542343123.8914564
//1542343112.6709805
//1542343101.4204564
//1542343090.0399055
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
	"math/big"

	"github.com/matrix/go-matrix/ca"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/core/state"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/matrixwork"
	"github.com/matrix/go-matrix/mc"
	"github.com/pkg/errors"
	"time"
)

func (p *Process) ProcessRecoveryMsg(msg *mc.RecoveryStateMsg) {
	log.INFO(p.logExtraInfo(), "状态恢复消息处理", "开始", "类型", msg.Type, "高度", p.number, "leader", msg.Header.Leader.Hex())
	defer log.INFO(p.logExtraInfo(), "状态恢复消息处理", "结束", "类型", msg.Type, "高度", p.number, "leader", msg.Header.Leader.Hex())

	p.mu.Lock()
	defer p.mu.Unlock()

	header := msg.Header
	headerHash := header.HashNoSignsAndNonce()
	minerResult := &mc.HD_MiningRspMsg{
		From:       header.Coinbase,
		Number:     header.Number.Uint64(),
		BlockHash:  headerHash,
		Difficulty: header.Difficulty,
		Nonce:      header.Nonce,
		Coinbase:   header.Coinbase,
		MixDigest:  header.MixDigest,
		Signatures: header.Signatures,
	}
	log.INFO(p.logExtraInfo(), "状态恢复消息处理", "开始补全挖矿结果消息")
	if err := p.powPool.AddMinerResult(minerResult.BlockHash, minerResult.Difficulty, minerResult); err != nil {
		log.WARN(p.logExtraInfo(), "状态恢复消息处理", "挖矿结果入池失败", "err", err, "高度", p.number)
	}
	p.processMinerResultVerify(header.Leader, false)

	if p.state != StateEnd {
		//处理完成后，状态不是完成状态，说明缺少数据
		log.INFO(p.logExtraInfo(), "状态恢复消息处理", "处理完毕后，本地状态不是end", "本地状态", p.state, "header hash", headerHash.TerminalString())
		if p.FullBlockReqCache.IsExistMsg(headerHash) {
			data, err := p.FullBlockReqCache.ReUseMsg(headerHash)
			if err != nil {
				return
			}
			reqMsg, _ := data.(*mc.HD_FullBlockReqMsg)
			log.INFO(p.logExtraInfo(), "状态恢复消息处理", "发送完整区块获取请求消息", "to", msg.From.Hex(), "高度", reqMsg.Number, "header hash", reqMsg.HeaderHash.TerminalString())
			p.pm.hd.SendNodeMsg(mc.HD_FullBlockReq, reqMsg, common.RoleNil, []common.Address{msg.From})
		} else {
			reqMsg := &mc.HD_FullBlockReqMsg{
				HeaderHash: headerHash,
				Number:     header.Number.Uint64(),
			}
			p.FullBlockReqCache.AddMsg(headerHash, reqMsg, time.Now().Unix())
			log.INFO(p.logExtraInfo(), "状态恢复消息处理", "发送完整区块获取请求消息", "to", msg.From.Hex(), "高度", reqMsg.Number, "header hash", reqMsg.HeaderHash.TerminalString())
			p.pm.hd.SendNodeMsg(mc.HD_FullBlockReq, reqMsg, common.RoleNil, []common.Address{msg.From})
		}
	}
}

func (p *Process) ProcessFullBlockReq(req *mc.HD_FullBlockReqMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	blockData := p.blockCache.GetBlockDataByBlockHash(req.HeaderHash)
	if nil == blockData {
		log.ERROR(p.logExtraInfo(), "处理完整区块请求", "区块信息未找到", "高度", p.number, "hash", req.HeaderHash.TerminalString())
		return
	}

	if blockData.state != blockStateReady {
		log.ERROR(p.logExtraInfo(), "处理完整区块请求", "区块未生成", "高度", p.number, "hash", req.HeaderHash.TerminalString())
		return
	}

	rspMsg := &mc.HD_FullBlockRspMsg{
		Header: blockData.block.Header,
		Txs:    blockData.block.Txs,
	}
	log.INFO(p.logExtraInfo(), "处理完整区块请求", "发送响应消息", "to", req.From, "full hash", rspMsg.Header.Hash(), "交易数量", rspMsg.Txs.Len())
	p.pm.hd.SendNodeMsg(mc.HD_FullBlockRsp, rspMsg, common.RoleNil, []common.Address{req.From})
}

func (p *Process) ProcessFullBlockRsp(rsp *mc.HD_FullBlockRspMsg) {
	fullHash := rsp.Header.Hash()
	headerHash := rsp.Header.HashNoSignsAndNonce()
	log.INFO(p.logExtraInfo(), "处理完整区块响应", "开始", "full hash", fullHash.TerminalString(), "交易数量", rsp.Txs.Len(), "高度", p.number)
	p.mu.Lock()
	defer p.mu.Unlock()

	if blockData := p.blockCache.GetBlockDataByBlockHash(headerHash); blockData != nil {
		log.ERROR(p.logExtraInfo(), "处理完整区块响应", "已存在的区块信息", "header Hash", headerHash.TerminalString(), "高度", p.number)
		return
	}

	if err := p.pm.engine.VerifyHeader(p.pm.bc, rsp.Header, true); err != nil {
		log.ERROR(p.logExtraInfo(), "处理完整区块响应", "POW验证未通过", "err", err, "高度", p.number)
		return
	}

	if err := p.pm.dposEngine.VerifyBlock(p.pm.bc, rsp.Header); err != nil {
		log.ERROR(p.logExtraInfo(), "处理完整区块响应", "POS验证未通过", "err", err, "高度", p.number)
		return
	}

	//运行交易
	receipts, stateDB, err := p.runTxs(rsp.Header, headerHash, rsp.Txs)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "处理完整区块响应", "执行交易错误", "err", err, "高度", p.number)
		return
	}

	p.blockCache.SaveReadyBlock(&mc.BlockLocalVerifyOK{
		Header:    rsp.Header,
		BlockHash: common.Hash{},
		Txs:       rsp.Txs,
		Receipts:  receipts,
		State:     stateDB,
	})

	readyMsg := &mc.NewBlockReadyMsg{
		Header: rsp.Header,
	}
	mc.PublishEvent(mc.BlockGenor_NewBlockReady, readyMsg)

	p.state = StateBlockInsert
	p.processBlockInsert()
}

func (p *Process) runTxs(header *types.Header, headerHash common.Hash, Txs types.Transactions) ([]*types.Receipt, *state.StateDB, error) {
	parent := p.blockChain().GetBlockByHash(header.ParentHash)
	if parent == nil {
		return nil, nil, errors.Errorf("父区块(%s)获取失败!", header.ParentHash.TerminalString())
	}

	localHeader := types.CopyHeader(header)
	localHeader.GasUsed = 0

	work, err := matrixwork.NewWork(p.blockChain().Config(), p.blockChain(), nil, localHeader)
	if err != nil {
		return nil, nil, errors.Errorf("创建worker错误(%v)", err)
	}
	err = work.ConsensusTransactions(p.pm.matrix.EventMux(), Txs, p.pm.bc)
	if err != nil {
		return nil, nil, errors.Errorf("执行交易错误(%v)", err)
	}
	_, err = p.blockChain().Engine().Finalize(p.blockChain(), localHeader, work.State,
		Txs, nil, work.Receipts)
	if err != nil {
		return nil, nil, errors.Errorf("Finalize error(%v)", err)
	}
	//localBlock check
	localHash := localHeader.HashNoSignsAndNonce()

	if localHash != headerHash {
		log.ERROR(p.logExtraInfo(), "交易验证，错误", "block hash不匹配",
			"local hash", localHash.TerminalString(), "remote hash", headerHash.TerminalString(),
			"local root", localHeader.Root.TerminalString(), "remote root", header.Root.TerminalString(),
			"local txHash", localHeader.TxHash.TerminalString(), "remote txHash", header.TxHash.TerminalString(),
			"local ReceiptHash", localHeader.ReceiptHash.TerminalString(), "remote ReceiptHash", header.ReceiptHash.TerminalString(),
			"local Bloom", localHeader.Bloom.Big(), "remote Bloom", header.Bloom.Big(),
			"local GasLimit", localHeader.GasLimit, "remote GasLimit", header.GasLimit,
			"local GasUsed", localHeader.GasUsed, "remote GasUsed", header.GasUsed)
		return nil, nil, errors.Errorf("block hash不匹配.LocalHash(%s) != remoteHash(%s)", localHash.TerminalString(), headerHash.TerminalString())
	}
	return work.Receipts, work.State, nil
}

func (p *Process) AddMinerResult(minerResult *mc.HD_MiningRspMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.powPool.AddMinerResult(minerResult.BlockHash, minerResult.Difficulty, minerResult); err != nil {
		log.ERROR(p.logExtraInfo(), "矿工挖矿结果入池失败", err, "高度", p.number)
		return
	}
	p.processMinerResultVerify(p.curLeader, true)
}

func (p *Process) AddConsensusBlock(block *mc.BlockLocalVerifyOK) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.blockCache.SaveVerifiedBlock(block)
	p.processMinerResultVerify(p.curLeader, true)
}

func (p *Process) processMinerResultVerify(leader common.Address, checkState bool) {
	if checkState {
		if p.checkState(StateMinerResultVerify) == false {
			log.WARN(p.logExtraInfo(), "准备进行挖矿结果验证，状态错误", p.state.String())
			return
		}
	}

	if common.IsBroadcastNumber(p.number) {
		log.INFO(p.logExtraInfo(), "当前高度为广播区块, 进行广播挖矿结果验证, 高度", p.number)
		p.dealMinerResultVerifyBroadcast()
	} else {
		log.INFO(p.logExtraInfo(), "当前高度为普通区块, 进行普通挖矿结果验证, 高度", p.number)
		p.dealMinerResultVerifyCommon(leader)
	}
}

func (p *Process) dealMinerResultVerifyCommon(leader common.Address) {
	var blockData *blockCacheData = nil
	if p.role == common.RoleBroadcast {
		blockData = p.blockCache.GetLastBlockData()
	} else {
		blockData = p.blockCache.GetBlockData(leader)
	}

	if nil == blockData {
		log.WARN(p.logExtraInfo(), "准备进行挖矿结果验证", "验证区块还未收到！等待验证区块", "高度", p.number, "身份", p.role, "leader", leader.Hex())
		return
	}

	if blockData.state == blockStateLocalVerified {
		diff := big.NewInt(blockData.block.Header.Difficulty.Int64())
		results, err := p.powPool.GetMinerResults(blockData.block.BlockHash, diff)
		if err != nil {
			log.WARN(p.logExtraInfo(), "挖矿结果验证，挖矿结果获取失败", err, "高度", p.number, "难度", diff, "block hash", blockData.block.BlockHash.TerminalString())
			return
		}

		satisfyResult, err := p.pickSatisfyMinerResults(blockData.block.Header, results)
		if err != nil {
			log.WARN(p.logExtraInfo(), "挖矿结果验证，获取合适挖矿结果错误", err, "高度", p.number)
			return
		}
		blockData.block.Header = p.copyHeader(blockData.block.Header, satisfyResult)
		blockData.state = blockStateReady
	}

	readyMsg := &mc.NewBlockReadyMsg{
		Header: blockData.block.Header,
	}
	log.INFO(p.logExtraInfo(), "普通区块验证完成", "发送新区块准备完毕消息", "高度", p.number, "leader", readyMsg.Header.Leader.Hex())
	mc.PublishEvent(mc.BlockGenor_NewBlockReady, readyMsg)

	p.state = StateBlockInsert
	p.processBlockInsert()
}

func (p *Process) processBlockInsert() {
	if p.state < StateBlockInsert {
		log.WARN(p.logExtraInfo(), "准备进行区块插入，状态错误", p.state.String(), "高度", p.number)
		return
	}

	if common.IsBroadcastNumber(p.number + 1) {
		if p.role != common.RoleBroadcast {
			log.WARN(p.logExtraInfo(), "准备进行区块插入，广播区块前一个区块，由广播节点插入", p.role.String(), "高度", p.number)
			return
		}
	} else {
		if p.role != common.RoleValidator {
			log.WARN(p.logExtraInfo(), "准备进行区块插入，身份错误", "当前身份不是验证者", "高度", p.number, "身份", p.role.String())
			return
		}

		if (p.nextLeader == common.Address{}) {
			log.WARN(p.logExtraInfo(), "准备进行区块插入", "下个区块leader为空", "需要等待leader的高度", p.number+1)
			return
		}

		if p.nextLeader != ca.GetAddress() {
			log.INFO(p.logExtraInfo(), "准备进行区块广播,自己不是下个区块leader,高度", p.number, "next leader", p.nextLeader.Hex(), "self", ca.GetAddress())
			return
		}
	}

	log.INFO(p.logExtraInfo(), "~~~~区块插入~~~~", "开始", "高度", p.number)
	hash, err := p.insertAndBcBlock(true, nil)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "区块插入，错误", err)
		return
	}

	log.INFO(p.logExtraInfo(), "~~~~区块插入~~~~", "完成", "高度", p.number, "插入区块hash", hash.TerminalString())
	p.state = StateEnd
}

func (p *Process) pickSatisfyMinerResults(header *types.Header, results []*mc.HD_MiningRspMsg) (*mc.HD_MiningRspMsg, error) {
	//todo 应该加入备选矿工滞后选择的流程
	for _, result := range results {
		if err := p.verifyOneResult(header, result); err != nil {
			log.WARN(p.logExtraInfo(), "验证挖矿结果失败，删除该挖矿结果, from", result.From, "diff", result.Difficulty,
				"高度", p.number, "block hash", result.BlockHash.TerminalString())
			p.powPool.DelOneResult(result.BlockHash, result.Difficulty, result.From)
			continue
		}
		return result, nil
	}
	return nil, HaveNotOKResultError
}

func (p *Process) verifyOneResult(rawHeader *types.Header, result *mc.HD_MiningRspMsg) error {
	header := p.copyHeader(rawHeader, result)
	headerHash := header.HashNoSignsAndNonce()
	if headerHash != result.BlockHash {
		log.ERROR(p.logExtraInfo(), "挖矿结果不匹配, header hash", headerHash.TerminalString(), "挖矿结果hash", result.BlockHash.TerminalString())
		return MinerResultError
	}

	if err := p.dposEngine().VerifyBlock(p.blockChain(), header); err != nil {
		log.ERROR(p.logExtraInfo(), "挖矿结果DPOS共识失败", err)
		return err
	}

	//todo 不是原始难度的结果，需要修改POW seal验证过程
	if err := p.engine().VerifySeal(p.blockChain(), header); err != nil {
		log.ERROR(p.logExtraInfo(), "挖矿结果POW验证失败", err)
		return err
	}

	return nil
}

func (p *Process) copyHeader(header *types.Header, minerResult *mc.HD_MiningRspMsg) *types.Header {
	newHeader := types.CopyHeader(header)
	newHeader.Nonce = minerResult.Nonce
	newHeader.Coinbase = minerResult.Coinbase
	newHeader.MixDigest = minerResult.MixDigest
	newHeader.Signatures = make([]common.Signature, 0)
	newHeader.Signatures = append(newHeader.Signatures, minerResult.Signatures...)
	return newHeader
}

func (p *Process) insertAndBcBlock(isSelf bool, header *types.Header) (common.Hash, error) {
	var blockData *blockCacheData = nil
	if p.role == common.RoleBroadcast {
		blockData = p.blockCache.GetLastBlockData()
	} else {
		blockData = p.blockCache.GetBlockData(p.curLeader)
	}
	if nil == blockData || blockData.state != blockStateReady {
		return common.Hash{}, HaveNoGenBlockError
	}

	insertHeader := blockData.block.Header
	if isSelf == false {
		if header.HashNoSignsAndNonce() != insertHeader.HashNoSignsAndNonce() {
			return common.Hash{}, HashNoSignNotMatchError
		}
		insertHeader = header
	}

	txs := blockData.block.Txs
	receipts := blockData.block.Receipts
	state := blockData.block.State
	block := types.NewBlockWithTxs(insertHeader, txs)

	stat, err := p.blockChain().WriteBlockWithState(block, receipts, state)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "Failed writing block to chain", err)
		return common.Hash{}, err
	}

	// Broadcast the block and announce chain insertion event
	hash := block.Hash()
	p.eventMux().Post(core.NewMinedBlockEvent{Block: block})
	var (
		events []interface{}
		logs   = state.Logs()
	)
	events = append(events, core.ChainEvent{Block: block, Hash: hash, Logs: logs})
	if stat == core.CanonStatTy {
		events = append(events, core.ChainHeadEvent{Block: block})
	}
	p.blockChain().PostChainEvents(events, logs)
	mc.PublishEvent(mc.BlockGenor_HeaderGenerateReq, p.number+1)
	return hash, nil
}
