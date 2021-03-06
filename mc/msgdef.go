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
package mc

import (
	"math/big"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core/state"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/core/vm"
	"github.com/matrix/go-matrix/p2p/discover"
)

//common

type HdRev struct {
	FromNodeId string
	Input      interface{}
}

type BlockData struct {
	Header *types.Header
	Txs    types.Transactions
}

//Miner Module
type HD_MiningReqMsg struct {
	From   common.Address
	Header *types.Header
}

type HD_MiningRspMsg struct {
	From       common.Address
	Number     uint64
	Blockhash  common.Hash
	Difficulty *big.Int
	Nonce      types.BlockNonce
	Coinbase   common.Address
	MixDigest  common.Hash
	Signatures []common.Signature
}

type BlockGenor_BroadcastMiningReqMsg struct {
	BlockMainData *BlockData
}

type HD_BroadcastMiningRspMsg struct {
	From          common.Address
	BlockMainData *BlockData
}

//拓扑生成模块
type DepositDetail struct {
	Address    common.Address
	NodeID     discover.NodeID
	Deposit    *big.Int
	WithdrawH  *big.Int
	OnlineTime *big.Int
}
type TopologyNodeInfo struct {
	Account    common.Address
	Position   uint16
	Type       common.RoleType
	Stock      uint16
	NodeNumber uint8 //0-99
	//	OnlineState bool
}
type Alternative struct {
	A        common.Address
	B        common.Address
	Position uint16
}

type TopologyGraph struct {
	Number        uint64
	NodeList      []TopologyNodeInfo
	CurNodeNumber uint8
}

//矿工主节点生成请求
type MasterMinerReElectionReqMsg struct {
	SeqNum    uint64
	RandSeed  *big.Int
	MinerList []vm.DepositDetail
}

//验证者主节点生成请求
type MasterValidatorReElectionReqMsg struct {
	SeqNum                  uint64
	RandSeed                *big.Int
	ValidatorList           []vm.DepositDetail
	FoundationValidatoeList []vm.DepositDetail
}

//矿工主节点生成响应
type MasterMinerReElectionRsp struct {
	SeqNum      uint64
	MasterMiner []TopologyNodeInfo
	BackUpMiner []TopologyNodeInfo
}

//验证者主节点生成响应
type MasterValidatorReElectionRsq struct {
	SeqNum             uint64
	MasterValidator    []TopologyNodeInfo
	BackUpValidator    []TopologyNodeInfo
	CandidateValidator []TopologyNodeInfo
}

type RoleUpdatedMsg struct {
	Role      common.RoleType
	BlockNum  uint64
	BlockHash common.Hash
	Leader    common.Address
}

type LeaderChangeNotify struct {
	ConsensusState bool //共识结果
	Leader         common.Address
	NextLeader     common.Address
	Number         uint64
	ConsensusTurn  uint32
	ReelectTurn    uint32
	TurnBeginTime  int64
	TurnEndTime    int64
}

//block verify server
type HD_BlkConsensusReqMsg struct {
	From          common.Address
	Header        *types.Header
	ConsensusTurn uint32
	TxsCode       []uint32
}

type LocalBlockVerifyConsensusReq struct {
	BlkVerifyConsensusReq *HD_BlkConsensusReqMsg
	Txs                   types.Transactions // 交易列表
	Receipts              []*types.Receipt   // 收据
	State                 *state.StateDB     // apply state changes here 状态数据库
}

type BlockPOSFinishedNotify struct {
	Number        uint64
	Header        *types.Header // 包含签名列表的header
	ConsensusTurn uint32
	TxsCode       []uint32
}

type BlockLocalVerifyOK struct {
	Header    *types.Header // 包含签名列表的header
	BlockHash common.Hash
	Txs       types.Transactions // 交易列表
	Receipts  []*types.Receipt   // 收据
	State     *state.StateDB     // apply state changes here 状态数据库
}

//BolckGenor
type HD_BlockInsertNotify struct {
	From   common.Address
	Header *types.Header
}

type NewBlockReadyMsg struct {
	Header *types.Header
}

//随机数生成请求
type RandomRequest struct {
	MinHash    common.Hash
	PrivateMap map[common.Address][]byte
	PublicMap  map[common.Address][]byte
}

//随机数生成响应
type ElectionEvent struct {
	Seed *big.Int
}

//在线状态共识请求
type OnlineConsensusReq struct {
	Leader      common.Address //leader地址
	Seq         uint64         //共识轮次
	Node        common.Address // node 地址
	OnlineState int            //在线状态
}

//在线状态共识请求消息
type HD_OnlineConsensusReqs struct {
	From    common.Address
	ReqList []*OnlineConsensusReq //请求结构
}

//共识投票消息
type HD_ConsensusVote struct {
	SignHash common.Hash
	Round    uint64
	Sign     common.Signature
	From     common.Address
}

type HD_OnlineConsensusVotes struct {
	Votes []HD_ConsensusVote
}

//共识结果
type HD_OnlineConsensusVoteResultMsg struct {
	Req      *OnlineConsensusReq //请求结构
	SignList []common.Signature  //签名列表
}

//特殊交易
type BroadCastEvent struct {
	Txtyps string
	Height *big.Int
	Data   []byte
}

//
type HD_ReelectInquiryReqMsg struct {
	Number        uint64
	ConsensusTurn uint32
	ReelectTurn   uint32
	TimeStamp     int64 // TODO  考虑作恶，提前时间
	Master        common.Address
	From          common.Address
}

type ReelectRSPType uint8

const (
	ReelectRSPTypeNone ReelectRSPType = iota
	ReelectRSPTypePOS
	ReelectRSPTypeAlreadyRL
	ReelectRSPTypeAgree
	ReelectRSPTypeNewBlockReady
)

type HD_ReelectInquiryRspMsg struct {
	Number    uint64
	ReqHash   common.Hash
	Type      ReelectRSPType
	AgreeSign common.Signature
	POSResult *HD_BlkConsensusReqMsg
	RLResult  *HD_ReelectLeaderConsensus
	NewBlock  *types.Header
	From      common.Address
}

type HD_ReelectLeaderReqMsg struct {
	InquiryReq *HD_ReelectInquiryReqMsg
	AgreeSigns []common.Signature
	TimeStamp  int64
}

type HD_ReelectLeaderVoteMsg struct {
	Vote   HD_ConsensusVote
	Number uint64
}

type HD_ReelectLeaderConsensus struct {
	Req       *HD_ReelectLeaderReqMsg
	Votes     []common.Signature
	TimeStamp int64
}

type HD_ReelectResultBroadcastMsg struct {
	Number    uint64
	Type      ReelectRSPType
	POSResult *HD_BlkConsensusReqMsg
	RLResult  *HD_ReelectLeaderConsensus
	TimeStamp int64
	From      common.Address
}

type HD_ReelectResultRspMsg struct {
	Number     uint64
	ResultHash common.Hash
	Sign       common.Signature
	From       common.Address
}

type RecoveryType uint8

const (
	RecoveryTypePOS RecoveryType = iota
	RecoveryTypeFullHeader
)

type RecoveryStateMsg struct {
	Type   RecoveryType
	Header *types.Header
	From   common.Address
}

type HD_FullBlockReqMsg struct {
	HeaderHash common.Hash
	Number     uint64
	From       common.Address
}

type HD_FullBlockRspMsg struct {
	Header *types.Header
	Txs    types.Transactions
	From   common.Address
}
