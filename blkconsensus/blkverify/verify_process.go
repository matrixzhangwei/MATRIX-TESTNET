//1542342714.7772532
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
package blkverify

import (
	"sync"
	"time"

	"github.com/matrix/go-matrix/accounts/signhelper"
	"github.com/matrix/go-matrix/blkconsensus/votepool"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/event"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/matrixwork"
	"github.com/matrix/go-matrix/mc"
	"github.com/matrix/go-matrix/reelection"
	"github.com/matrix/go-matrix/topnode"
)

type State uint16

const (
	StateIdle State = iota
	StateStart
	StateReqVerify
	StateTxsVerify
	StateDPOSVerify
	StateEnd
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "未运行状态"
	case StateStart:
		return "开始状态"
	case StateReqVerify:
		return "请求验证阶段"
	case StateTxsVerify:
		return "交易验证阶段"
	case StateDPOSVerify:
		return "DPOS共识阶段"
	case StateEnd:
		return "完成状态"
	default:
		return "未知状态"
	}
}

const (
	localVerifyResultProcessing uint8 = iota
	localVerifyResultSuccess
	localVerifyResultFailedButCanRecover
	localVerifyResultStateFailed
)

type Process struct {
	mu               sync.Mutex
	leaderCache      mc.LeaderChangeNotify
	number           uint64
	role             common.RoleType
	state            State
	curProcessReq    *reqData
	reqCache         *reqCache
	pm               *ProcessManage
	txsAcquireSeq    int
	voteMsgSender    *common.ResendMsgCtrl
	mineReqMsgSender *common.ResendMsgCtrl
	posedReqSender   *common.ResendMsgCtrl
}

func newProcess(number uint64, pm *ProcessManage) *Process {
	p := &Process{
		leaderCache: mc.LeaderChangeNotify{
			ConsensusState: false,
			Leader:         common.Address{},
			NextLeader:     common.Address{},
			Number:         number,
			ConsensusTurn:  0,
			ReelectTurn:    0,
			TurnBeginTime:  0,
			TurnEndTime:    0,
		},
		number:           number,
		role:             common.RoleNil,
		state:            StateIdle,
		curProcessReq:    nil,
		reqCache:         newReqCache(),
		pm:               pm,
		txsAcquireSeq:    0,
		voteMsgSender:    nil,
		mineReqMsgSender: nil,
		posedReqSender:   nil,
	}

	return p
}

func (p *Process) StartRunning(role common.RoleType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.role = role
	p.changeState(StateStart)

	if p.role == common.RoleBroadcast {
		p.startReqVerifyBC()
	} else if p.role == common.RoleValidator {
		p.startReqVerifyCommon()
	}
}

func (p *Process) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = StateIdle
	p.curProcessReq = nil
	p.stopSender()
}

func (p *Process) SetLeaderInfo(info *mc.LeaderChangeNotify) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.leaderCache.ConsensusState = info.ConsensusState
	if p.leaderCache.ConsensusState == false {
		p.stopProcess()
		return
	}

	if p.leaderCache.Leader == info.Leader && p.leaderCache.ConsensusTurn == info.ConsensusTurn {
		//已处理过的leader消息，不处理
		return
	}

	//leader或轮次变化了，更新缓存
	p.leaderCache.Leader.Set(info.Leader)
	p.leaderCache.NextLeader.Set(info.NextLeader)
	p.leaderCache.ConsensusTurn = info.ConsensusTurn
	p.leaderCache.ReelectTurn = info.ReelectTurn
	p.leaderCache.TurnBeginTime = info.TurnBeginTime
	p.leaderCache.TurnEndTime = info.TurnEndTime
	p.curProcessReq = nil

	//维护req缓存
	p.reqCache.SetCurTurn(p.leaderCache.ConsensusTurn)

	//重启process
	p.stopSender()
	if p.state > StateIdle {
		p.state = StateStart
		if p.role == common.RoleValidator {
			p.startReqVerifyCommon()
		} else if p.role == common.RoleBroadcast {
			log.WARN(p.logExtraInfo(), "广播身份下收到leader变更消息", "不处理")
		}
	}
}

func (p *Process) stopProcess() {
	p.closeMineReqMsgSender()
	p.leaderCache.Leader.Set(common.Address{})
	p.leaderCache.NextLeader.Set(common.Address{})
	p.leaderCache.ConsensusTurn = 0
	p.leaderCache.ReelectTurn = 0
	p.leaderCache.TurnBeginTime = 0
	p.leaderCache.TurnEndTime = 0
	p.curProcessReq = nil

	if p.state > StateIdle {
		p.state = StateStart
	}
}

func (p *Process) AddReq(reqMsg *mc.HD_BlkConsensusReqMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.reqCache.AddReq(reqMsg)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "请求添加缓存失败", err, "from", reqMsg.From, "高度", p.number)
		return
	}
	log.INFO(p.logExtraInfo(), "请求添加缓存成功", err, "from", reqMsg.From, "高度", p.number)

	if p.role == common.RoleBroadcast {
		p.startReqVerifyBC()
	} else if p.role == common.RoleValidator {
		p.startReqVerifyCommon()
	}
}

func (p *Process) AddLocalReq(localReq *mc.LocalBlockVerifyConsensusReq) {
	p.mu.Lock()
	defer p.mu.Unlock()

	leader := localReq.BlkVerifyConsensusReq.Header.Leader
	err := p.reqCache.AddLocalReq(localReq)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "本地请求添加缓存失败", err, "高度", p.number, "leader", leader.Hex())
		return
	}
	log.INFO(p.logExtraInfo(), "本地请求添加成功, 高度", p.number, "leader", leader.Hex())

	if p.role == common.RoleBroadcast {
		p.startReqVerifyBC()
	} else if p.role == common.RoleValidator {
		p.startReqVerifyCommon()
	}
}

func (p *Process) ProcessDPOSOnce() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processDPOSOnce()
}

func (p *Process) ProcessRecoveryMsg(msg *mc.RecoveryStateMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	msgHeaderHash := msg.Header.HashNoSignsAndNonce()
	reqData, err := p.reqCache.GetLeaderReqByHash(msgHeaderHash)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "处理状态恢复消息", "本地请求获取失败", "err", err)
		return
	}
	if reqData.hash != msgHeaderHash {
		log.ERROR(p.logExtraInfo(), "处理状态恢复消息", "本地请求hash不匹配，忽略消息",
			"本地hash", reqData.hash.TerminalString(), "消息hash", msgHeaderHash.TerminalString())
		return
	}

	log.INFO(p.logExtraInfo(), "处理状态恢复消息", "开始重置POS投票")
	p.votePool().DelVotes(reqData.hash)
	//添加投票
	for _, sign := range msg.Header.Signatures {
		p.votePool().AddVote(reqData.hash, sign, common.Address{}, p.number, false)
	}
	p.processDPOSOnce()
	log.INFO(p.logExtraInfo(), "处理状态恢复消息", "完成")
}

func (p *Process) startReqVerifyCommon() {
	if p.checkState(StateStart) == false {
		log.WARN(p.logExtraInfo(), "准备开始请求验证阶段，状态错误", p.state.String(), "高度", p.number)
		return
	}

	if p.leaderCache.ConsensusState == false {
		log.WARN(p.logExtraInfo(), "请求验证阶段", "当前leader未共识完成，等待leader消息", "高度", p.number)
		return
	}

	req, err := p.reqCache.GetLeaderReq(p.leaderCache.Leader, p.leaderCache.ConsensusTurn)
	if err != nil {
		log.WARN(p.logExtraInfo(), "请求验证阶段,寻找leader的请求错误,继续等待请求", err,
			"Leader", p.leaderCache.Leader.Hex(), "轮次", p.leaderCache.ConsensusTurn, "高度", p.number)
		return
	}

	p.curProcessReq = req
	log.INFO(p.logExtraInfo(), "请求验证阶段", "开始", "高度", p.number, "HeaderHash", p.curProcessReq.hash.TerminalString(), "parent hash", p.curProcessReq.req.Header.ParentHash.TerminalString(), "之前状态", p.state.String())
	p.state = StateReqVerify
	p.processReqOnce()
}

func (p *Process) processReqOnce() {
	if p.checkState(StateReqVerify) == false {
		return
	}

	// if is local req, skip local verify step
	if p.curProcessReq.localReq {
		log.INFO(p.logExtraInfo(), "请求为本地请求", "跳过验证阶段", "高度", p.number)
		p.startDPOSVerify(localVerifyResultSuccess)
		return
	}

	// verify timestamp
	headerTime := p.curProcessReq.req.Header.Time.Int64()
	if headerTime < p.leaderCache.TurnBeginTime || headerTime > p.leaderCache.TurnEndTime {
		log.ERROR(p.logExtraInfo(), "验证请求头时间戳", "时间戳不合法", "头时间", headerTime,
			"轮次开始时间", p.leaderCache.TurnBeginTime, "轮次结束时间", p.leaderCache.TurnEndTime,
			"轮次", p.leaderCache.ConsensusTurn, "高度", p.number)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}

	// verify header
	if err := p.blockChain().VerifyHeader(p.curProcessReq.req.Header); err != nil {
		log.ERROR(p.logExtraInfo(), "预验证头信息失败", err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}

	// verify election info
	if err := p.verifyElection(p.curProcessReq.req.Header); err != nil {
		log.ERROR(p.logExtraInfo(), "验证选举信息失败", err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}

	// verify net topology info
	if err := p.verifyNetTopology(p.curProcessReq.req.Header); err != nil {
		log.ERROR(p.logExtraInfo(), "验证拓扑信息失败", err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultFailedButCanRecover)
		return
	}

	//todo Version

	p.startTxsVerify()
}

func (p *Process) startTxsVerify() {
	if p.checkState(StateReqVerify) == false {
		return
	}
	log.INFO(p.logExtraInfo(), "交易获取", "开始", "当前身份", p.role.String(), "高度", p.number)

	p.changeState(StateTxsVerify)

	p.txsAcquireSeq++
	leader := p.curProcessReq.req.Header.Leader
	//todo 交易数量为空时，跳过交易验证阶段
	log.INFO(p.logExtraInfo(), "开始交易获取,seq", p.txsAcquireSeq, "数量", len(p.curProcessReq.req.TxsCode), "leader", leader.Hex(), "高度", p.number)
	txAcquireCh := make(chan *core.RetChan, 1)
	go p.txPool().ReturnAllTxsByN(p.curProcessReq.req.TxsCode, p.txsAcquireSeq, leader, txAcquireCh)
	go p.processTxsAcquire(txAcquireCh, p.txsAcquireSeq)
}

func (p *Process) processTxsAcquire(txsAcquireCh <-chan *core.RetChan, seq int) {
	log.INFO(p.logExtraInfo(), "交易获取协程", "启动", "当前身份", p.role.String(), "高度", p.number)
	defer log.INFO(p.logExtraInfo(), "交易获取协程", "退出", "当前身份", p.role.String(), "高度", p.number)

	outTime := time.NewTimer(time.Second * 5)
	select {
	case txsResult := <-txsAcquireCh:

		go p.VerifyTxs(txsResult)
	case <-outTime.C:
		log.INFO(p.logExtraInfo(), "交易获取协程", "获取交易超时", "高度", p.number, "seq", seq)
		go p.ProcessTxsAcquireTimeOut(seq)
		return
	}
}

func (p *Process) ProcessTxsAcquireTimeOut(seq int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.INFO(p.logExtraInfo(), "交易获取超时处理", "开始", "高度", p.number, "seq", seq, "cur seq", p.txsAcquireSeq)
	defer log.INFO(p.logExtraInfo(), "交易获取超时处理", "结束", "高度", p.number, "seq", seq)

	if seq != p.txsAcquireSeq {
		log.WARN(p.logExtraInfo(), "交易获取超时处理", "Seq不匹配，忽略", "高度", p.number, "seq", seq, "cur seq", p.txsAcquireSeq)
		return
	}

	if p.checkState(StateTxsVerify) == false {
		log.INFO(p.logExtraInfo(), "交易获取超时处理", "状态不正确，不处理", "高度", p.number, "seq", seq)
		return
	}

	p.startDPOSVerify(localVerifyResultFailedButCanRecover)
}

func (p *Process) VerifyTxs(result *core.RetChan) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.checkState(StateTxsVerify) == false {
		return
	}

	log.INFO(p.logExtraInfo(), "交易验证，交易数据 result.seq", result.Resqe, "当前 reqSeq", p.txsAcquireSeq, "高度", p.number)
	if result.Resqe != p.txsAcquireSeq {
		log.WARN(p.logExtraInfo(), "交易验证", "seq不匹配，跳过", "高度", p.number)
		return
	}

	if result.Err != nil {
		log.ERROR(p.logExtraInfo(), "交易验证，交易数据错误", result.Err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultFailedButCanRecover)
		return
	}

	log.INFO(p.logExtraInfo(), "开始交易验证, 数量", len(result.Rxs), "高度", p.number)
	p.curProcessReq.txs = result.Rxs

	//跑交易交易验证， Root TxHash ReceiptHash Bloom GasLimit GasUsed
	remoteHeader := p.curProcessReq.req.Header
	localHeader := types.CopyHeader(remoteHeader)
	localHeader.GasUsed = 0

	work, err := matrixwork.NewWork(p.blockChain().Config(), p.blockChain(), nil, localHeader)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "交易验证，创建work失败!", err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultFailedButCanRecover)
		return
	}
	//todo add handleuptime
	/*	if common.IsBroadcastNumber(p.number-1) && p.number > common.GetBroadcastInterval() {
		upTimeAccounts, err := work.GetUpTimeAccounts(p.number)
		if err != nil {
			log.ERROR(p.logExtraInfo(), "获取所有抵押账户错误!", err, "高度", p.number)
			return
		}
		calltherollMap, heatBeatUnmarshallMMap, err := work.GetUpTimeData(p.number)
		if err != nil {
			log.WARN(p.logExtraInfo(), "获取心跳交易错误!", err, "高度", p.number)
		}
		err = work.HandleUpTime(work.State, upTimeAccounts, calltherollMap, heatBeatUnmarshallMMap, p.number, p.pm.bc)
		if nil != err {
			log.ERROR(p.logExtraInfo(), "处理uptime错误", err)
			return
		}
	}*/
	err = work.ConsensusTransactions(p.pm.event, p.curProcessReq.txs, p.pm.bc)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "交易验证，共识执行交易出错!", err, "高度", p.number)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}
	_, err = p.blockChain().Engine().Finalize(p.blockChain(), localHeader, work.State,
		p.curProcessReq.txs, nil, work.Receipts)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "交易验证,错误", "Failed to finalize block for sealing", "err", err)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}
	//localBlock check
	localHash := localHeader.HashNoSignsAndNonce()

	if localHash != p.curProcessReq.hash {
		log.ERROR(p.logExtraInfo(), "交易验证，错误", "block hash不匹配",
			"local hash", localHash.TerminalString(), "remote hash", p.curProcessReq.hash.TerminalString(),
			"local root", localHeader.Root.TerminalString(), "remote root", remoteHeader.Root.TerminalString(),
			"local txHash", localHeader.TxHash.TerminalString(), "remote txHash", remoteHeader.TxHash.TerminalString(),
			"local ReceiptHash", localHeader.ReceiptHash.TerminalString(), "remote ReceiptHash", remoteHeader.ReceiptHash.TerminalString(),
			"local Bloom", localHeader.Bloom.Big(), "remote Bloom", remoteHeader.Bloom.Big(),
			"local GasLimit", localHeader.GasLimit, "remote GasLimit", remoteHeader.GasLimit,
			"local GasUsed", localHeader.GasUsed, "remote GasUsed", remoteHeader.GasUsed)
		p.startDPOSVerify(localVerifyResultStateFailed)
		return
	}

	p.curProcessReq.receipts = work.Receipts
	p.curProcessReq.stateDB = work.State

	// 开始DPOS共识验证
	p.startDPOSVerify(localVerifyResultSuccess)
}

func (p *Process) sendVote(validate bool) {
	signHash := p.curProcessReq.hash
	sign, err := p.signHelper().SignHashWithValidate(signHash.Bytes(), validate)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "投票签名失败", err, "高度", p.number)
		return
	}

	p.startVoteMsgSender(&mc.HD_ConsensusVote{SignHash: signHash, Sign: sign, Round: p.number})

	//将自己的投票加入票池
	if err := p.votePool().AddVote(signHash, sign, common.Address{}, p.number, false); err != nil {
		log.ERROR(p.logExtraInfo(), "自己的投票加入票池失败", err, "高度", p.number)
	}

	// notify block genor server the result
	result := mc.BlockLocalVerifyOK{
		Header:    p.curProcessReq.req.Header,
		BlockHash: p.curProcessReq.hash,
		Txs:       p.curProcessReq.txs,
		Receipts:  p.curProcessReq.receipts,
		State:     p.curProcessReq.stateDB,
	}
	//log.INFO(p.logExtraInfo(), "发出区块共识结果消息", result, "高度", p.number)
	mc.PublishEvent(mc.BlkVerify_VerifyConsensusOK, &result)
}

func (p *Process) startDPOSVerify(lvResult uint8) {
	if p.state >= StateDPOSVerify {
		return
	}

	if p.role == common.RoleBroadcast {
		//广播节点，跳过DPOS投票验证阶段
		p.bcFinishedProcess(lvResult)
		return
	}

	log.INFO(p.logExtraInfo(), "开始DPOS阶段,验证结果", lvResult, "高度", p.number)

	if lvResult == localVerifyResultSuccess {
		p.sendVote(true)
	}
	p.curProcessReq.localVerifyResult = lvResult

	p.state = StateDPOSVerify
	p.processDPOSOnce()
}

func (p *Process) processDPOSOnce() {
	if p.checkState(StateDPOSVerify) == false {
		return
	}

	if p.curProcessReq.req == nil {
		return
	}

	signs := p.votePool().GetVotes(p.curProcessReq.hash)
	log.INFO(p.logExtraInfo(), "执行DPOS, 投票数量", len(signs), "hash", p.curProcessReq.hash.TerminalString(), "高度", p.number)
	rightSigns, err := p.blockChain().DPOSEngine().VerifyHashWithVerifiedSignsAndNumber(p.blockChain(), signs, p.number)
	if err != nil {
		log.ERROR(p.logExtraInfo(), "共识引擎验证失败", err, "高度", p.number)
		return
	}
	log.INFO(p.logExtraInfo(), "DPOS通过，正确签名数量", len(rightSigns), "高度", p.number)
	p.curProcessReq.req.Header.Signatures = rightSigns

	p.finishedProcess()
}

func (p *Process) finishedProcess() {
	result := p.curProcessReq.localVerifyResult
	if result == localVerifyResultProcessing {
		log.ERROR(p.logExtraInfo(), "req is processing now, can't finish!", "validator", "高度", p.number)
		return
	}
	if result == localVerifyResultStateFailed {
		log.Error(p.logExtraInfo(), "local verify header err, but dpos pass! please check your state!", "validator", "高度", p.number)
		//todo 硬分叉了，以后加需要处理
		return
	}

	if result == localVerifyResultSuccess {
		// notify leader server the verify state
		notify := mc.BlockPOSFinishedNotify{
			Number:        p.number,
			Header:        p.curProcessReq.req.Header,
			ConsensusTurn: p.curProcessReq.req.ConsensusTurn,
			TxsCode:       p.curProcessReq.req.TxsCode,
		}
		mc.PublishEvent(mc.BlkVerify_POSFinishedNotify, &notify)
	}

	//给矿工发送区块验证结果
	p.startSendMineReq(&mc.HD_MiningReqMsg{Header: p.curProcessReq.req.Header})
	//给广播节点发送区块验证请求(带签名列表)
	p.startPosedReqSender(p.curProcessReq.req)

	p.votePool().DelVotes(p.curProcessReq.hash)
	p.state = StateEnd
}

func (p *Process) checkState(state State) bool {
	return p.state == state
}

func (p *Process) changeState(targetState State) {
	if p.state == targetState-1 {
		log.WARN(p.logExtraInfo(), "切换状态成功, 原状态", p.state.String(), "新状态", targetState.String(), "高度", p.number)
		p.state = targetState
	} else {
		log.WARN(p.logExtraInfo(), "切换状态失败, 原状态", p.state.String(), "目标状态", targetState.String(), "高度", p.number)
	}
}

func (p *Process) votePool() *votepool.VotePool { return p.pm.votePool }

func (p *Process) signHelper() *signhelper.SignHelper { return p.pm.signHelper }

func (p *Process) blockChain() *core.BlockChain { return p.pm.bc }

func (p *Process) txPool() *core.TxPool { return p.pm.txPool }

func (p *Process) reElection() *reelection.ReElection { return p.pm.reElection }

func (p *Process) logExtraInfo() string { return p.pm.logExtraInfo() }

func (p *Process) eventMux() *event.TypeMux { return p.pm.event }

func (p *Process) topNode() *topnode.TopNodeService { return p.pm.topNode }
