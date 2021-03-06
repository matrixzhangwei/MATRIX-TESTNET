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

package reelection

import (
	"github.com/matrix/go-matrix/accounts"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/election"
	"github.com/matrix/go-matrix/mandb"
	"github.com/matrix/go-matrix/event"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/mc"
	"github.com/matrix/go-matrix/params/man"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"time"
)

var (
	/*
		MinerTopologyAlreadyGenerate     = errors.New("Miner Topology Already Generate")
		ValidatorTopologyAlreadyGenerate = errors.New("Validator Topology Already Generate")
		MinerNotRecviveTopology          = errors.New("Miner Not Recvive Topology")
		ValidatorNotReceiveTopology      = errors.New("Validator Not Receive Topology")
		TopNotBeLocal                    = errors.New("Top Not Be Local")
	*/

	BroadCastInterval        = common.GetBroadcastInterval()
	MinerTopGenTiming        = common.GetReElectionInterval() - man.MinerTopologyGenerateUpTime
	MinerNetchangeTiming     = common.GetReElectionInterval() - man.MinerNetChangeUpTime
	ValidatorTopGenTiming    = common.GetReElectionInterval() - man.VerifyTopologyGenerateUpTime
	ValidatorNetChangeTiming = common.GetReElectionInterval() - man.VerifyNetChangeUpTime
	Time_Out_Limit           = 2 * time.Second
	ChanSize                 = 10
)

const (
	Module = "换届服务"
)

// Backend wraps all methods required for mining.
type Backend interface {
	AccountManager() *accounts.Manager
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	ChainDb() mandb.Database
}
type AllNative struct {
	MasterMiner        []mc.TopologyNodeInfo //矿工主节点
	BackUpMiner        []mc.TopologyNodeInfo //矿工备份
	MasterValidator    []mc.TopologyNodeInfo //验证者主节点
	BackUpValidator    []mc.TopologyNodeInfo //验证者备份
	CandidateValidator []mc.TopologyNodeInfo //验证者候选

}

type ElectMiner struct {
	MasterMiner []mc.TopologyNodeInfo
	BackUpMiner []mc.TopologyNodeInfo
}

type ElectValidator struct {
	MasterValidator    []mc.TopologyNodeInfo
	BackUpValidator    []mc.TopologyNodeInfo
	CandidateValidator []mc.TopologyNodeInfo
}

type ElectReturnInfo struct {
	MasterMiner     []mc.TopologyNodeInfo
	BackUpMiner     []mc.TopologyNodeInfo
	MasterValidator []mc.TopologyNodeInfo
	BackUpValidator []mc.TopologyNodeInfo
}
type ReElection struct {
	bc  *core.BlockChain //man实例：生成种子时获取一周期区块的最小hash
	ldb *leveldb.DB      //本都db数据库

	roleUpdateCh    chan *mc.RoleUpdatedMsg //身份变更信息通道
	roleUpdateSub   event.Subscription
	minerGenCh      chan *mc.MasterMinerReElectionRsp //矿工主节点生成消息通道
	minerGenSub     event.Subscription
	validatorGenCh  chan *mc.MasterValidatorReElectionRsq //验证者主节点生成消息通道
	validatorGenSub event.Subscription
	electionSeedCh  chan *mc.ElectionEvent //选举种子请求消息通道
	electionSeedSub event.Subscription

	//allNative AllNative

	currentID common.RoleType //当前身份

	elect *election.Elector
	lock  sync.Mutex
}

func New(bc *core.BlockChain, dbDir string) (*ReElection, error) {
	reelection := &ReElection{
		bc:             bc,
		roleUpdateCh:   make(chan *mc.RoleUpdatedMsg, ChanSize),
		minerGenCh:     make(chan *mc.MasterMinerReElectionRsp, ChanSize),
		validatorGenCh: make(chan *mc.MasterValidatorReElectionRsq, ChanSize),
		electionSeedCh: make(chan *mc.ElectionEvent, ChanSize),

		currentID: common.RoleDefault,
	}
	reelection.elect = election.NewEle()
	var err error
	dbDir = dbDir + "_reElection"
	reelection.ldb, err = leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}
	err = reelection.initSubscribeEvent()
	if err != nil {
		return nil, err
	}
	go reelection.update()
	return reelection, nil
}

func (self *ReElection) initSubscribeEvent() error {
	var err error

	self.roleUpdateSub, err = mc.SubscribeEvent(mc.CA_RoleUpdated, self.roleUpdateCh)

	if err != nil {
		return err
	}
	log.INFO(Module, "CA_RoleUpdated", "订阅成功")
	return nil
}
func (self *ReElection) update() {
	defer func() {
		if self.roleUpdateSub != nil {
			self.roleUpdateSub.Unsubscribe()
		}

	}()
	for {
		select {
		case roleData := <-self.roleUpdateCh:
			log.INFO(Module, "roleData", roleData)
			go self.roleUpdateProcess(roleData)
		}
	}
}

func (self *ReElection) GetTopoChange(height uint64, offline []common.Address) ([]mc.Alternative, error) {

	log.INFO(Module, "获取拓扑改变 start height", height, "offline", offline)
	//if height <= common.GetReElectionInterval() {
		//log.Error(Module, "小于第一个选举周期返回空的拓扑差值 height", height)
		return []mc.Alternative{}, nil

	//}
	antive, err := self.readNativeData(height - 1)
	if err != nil {
		log.Error(Module, "获取上一个高度的初选列表失败 height-1", height-1)
		return []mc.Alternative{}, err
	}

	//aim := 0x04 + 0x08
	TopoGrap, err := GetCurrentTopology(height-1, common.RoleMiner|common.RoleValidator)
	if err != nil {
		log.Error(Module, "获取CA当前拓扑图失败 err", err)
		return []mc.Alternative{}, err
	}

	Diff := self.TopoUpdate(antive.MasterMiner, antive.BackUpMiner, []mc.TopologyNodeInfo{}, *TopoGrap, offline)

	DiffValidatot := self.TopoUpdate(antive.MasterValidator, antive.BackUpValidator, antive.CandidateValidator, *TopoGrap, offline)
	log.INFO(Module, "获取拓扑改变 end ", append(Diff, DiffValidatot...))
	return append(Diff, DiffValidatot...), nil

}

func (self *ReElection) GetElection(height uint64) (*ElectReturnInfo, error) {

	log.INFO(Module, "GetElection start height", height)
	if common.IsReElectionNumber(height + man.MinerNetChangeUpTime) {
		log.Error(Module, "是矿工网络生成切换时间点 height", height)
		if err:=self.checkTopGenStatus(height+man.MinerNetChangeUpTime);err!=nil{
			log.ERROR(Module,"检查top生成出错 err",err)
		}
		ans, _, err := self.readElectData(common.RoleMiner, height+ man.MinerNetChangeUpTime)
		if err != nil {
			log.ERROR(Module, "获取本地矿工选举信息失败", "miner", "heightminer", height+ man.MinerNetChangeUpTime)
			return nil, err
		}
		resultM := &ElectReturnInfo{
			MasterMiner: ans.MasterMiner,
			BackUpMiner: ans.BackUpMiner,
		}
		return resultM, nil
	} else if common.IsReElectionNumber(height + man.VerifyNetChangeUpTime) {
		log.Error(Module, "是验证者网络切换时间点 height", height)
		if err:=self.checkTopGenStatus(height+man.VerifyNetChangeUpTime);err!=nil{
			log.ERROR(Module,"检查top生成出错 err",err)
		}
		_, ans, err := self.readElectData(common.RoleValidator, height+man.VerifyNetChangeUpTime)
		if err != nil {
			log.ERROR(Module, "获取本地验证者选举信息失败", "miner", "heightValidator",height+man.VerifyNetChangeUpTime)
			return nil, err
		}
		resultV := &ElectReturnInfo{
			MasterValidator: ans.MasterValidator,
			BackUpValidator: ans.BackUpValidator,
		}
		return resultV, nil
	}
	log.INFO(Module, "GetElection end height", height)
	log.INFO(Module, "不是任何网络切换时间点 height", height)
	temp := &ElectReturnInfo{}
	return temp, nil

}
