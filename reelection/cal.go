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
	"errors"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/mc"
	"github.com/matrix/go-matrix/log"
)

//todo
func (self *ReElection) GetNetTopologyAll(height uint64) (*ElectReturnInfo, error) {
	if common.IsReElectionNumber(height + 1) {
		heightMiner := height + 1
		if err:=self.checkTopGenStatus(heightMiner);err!=nil{
			log.ERROR(Module,"error checking top generation err",err)
		}
		ans, _, err := self.readElectData(common.RoleMiner, heightMiner)
		if err != nil {
			return nil, err
		}

		heightValidator := height + 1
		if err:=self.checkTopGenStatus(heightValidator);err!=nil{
			log.ERROR(Module,"error checking top generation",err)
		}
		_, ans1, err := self.readElectData(common.RoleValidator, heightValidator)
		if err != nil {
			return nil, err
		}
		result := &ElectReturnInfo{
			MasterMiner:     ans.MasterMiner,
			BackUpMiner:     ans.BackUpMiner,
			MasterValidator: ans1.MasterValidator,
			BackUpValidator: ans1.BackUpValidator,
		}
		return result, nil

	}

	result := &ElectReturnInfo{
		MasterMiner:     make([]mc.TopologyNodeInfo, 0),
		BackUpMiner:     make([]mc.TopologyNodeInfo, 0),
		MasterValidator: make([]mc.TopologyNodeInfo, 0),
		BackUpValidator: make([]mc.TopologyNodeInfo, 0),
	}
	return result, nil
}

func (self *ReElection) ParseTopNodeOffline(topologyChg common.NetTopology, prevTopology *mc.TopologyGraph) []common.Address {
	if topologyChg.Type != common.NetTopoTypeChange {
		return nil
	}

	offline := make([]common.Address, 0)

	for _, v := range topologyChg.NetTopologyData {

		if v.Position == common.PosOffline || v.Position == common.PosOnline {
			continue
		}

		account := checkInGraph(prevTopology, v.Position)
		if checkInDiff(topologyChg, account) == false {
			offline = append(offline, account)
		}

	}
	return offline
}

func (self *ReElection) ParsePrimaryTopNodeState(topologyChg common.NetTopology) ([]common.Address, []common.Address) {
	if topologyChg.Type != common.NetTopoTypeChange {
		return nil, nil
	}

	online := make([]common.Address, 0)
	offline := make([]common.Address, 0)
	for _, v := range topologyChg.NetTopologyData {

		if v.Position == common.PosOffline {
			offline = append(offline, v.Account)
			continue
		}
		if v.Position == common.PosOnline {
			online = append(online, v.Account)
			continue
		}
	}

	return online, offline
}

func (self *ReElection) TransferToElectionStu(info *ElectReturnInfo) []common.Elect {
	result := make([]common.Elect, 0)

	srcMap := make(map[common.ElectRoleType][]mc.TopologyNodeInfo)
	srcMap[common.ElectRoleMiner] = info.MasterMiner
	srcMap[common.ElectRoleMinerBackUp] = info.BackUpMiner
	srcMap[common.ElectRoleValidator] = info.MasterValidator
	srcMap[common.ElectRoleValidatorBackUp] = info.BackUpValidator
	orderIndex := []common.ElectRoleType{common.ElectRoleValidator, common.ElectRoleValidatorBackUp, common.ElectRoleMiner, common.ElectRoleMinerBackUp}

	for _, role := range orderIndex {
		src := srcMap[role]
		for _, node := range src {
			e := common.Elect{
				Account: node.Account,
				Stock:   node.Stock,
				Type:    role,
			}

			result = append(result, e)
		}
	}

	return result
}

func (self *ReElection) TransferToNetTopologyAllStu(info *ElectReturnInfo) *common.NetTopology {
	result := &common.NetTopology{
		Type:            common.NetTopoTypeAll,
		NetTopologyData: make([]common.NetTopologyData, 0),
	}

	srcMap := make(map[common.ElectRoleType][]mc.TopologyNodeInfo)
	srcMap[common.ElectRoleMiner] = info.MasterMiner
	srcMap[common.ElectRoleMinerBackUp] = info.BackUpMiner
	srcMap[common.ElectRoleValidator] = info.MasterValidator
	srcMap[common.ElectRoleValidatorBackUp] = info.BackUpValidator
	orderIndex := []common.ElectRoleType{common.ElectRoleValidator, common.ElectRoleValidatorBackUp, common.ElectRoleMiner, common.ElectRoleMinerBackUp}

	for _, role := range orderIndex {
		src := srcMap[role]
		for i, node := range src {
			data := common.NetTopologyData{
				Account:  node.Account,
				Position: common.GeneratePosition(uint16(i), role),
			}
			result.NetTopologyData = append(result.NetTopologyData, data)
		}
	}

	return result
}

func (self *ReElection) TransferToNetTopologyChgStu(alterInfo []mc.Alternative,
	onlinePrimaryNods []common.Address,
	offlinePrimaryNodes []common.Address) *common.NetTopology {
	result := &common.NetTopology{
		Type:            common.NetTopoTypeChange,
		NetTopologyData: make([]common.NetTopologyData, 0),
	}

	for _, alter := range alterInfo {
		data := common.NetTopologyData{
			Account:  alter.A,
			Position: alter.Position,
		}
		result.NetTopologyData = append(result.NetTopologyData, data)
	}

	for _, onlineNode := range onlinePrimaryNods {
		data := common.NetTopologyData{
			Account:  onlineNode,
			Position: common.PosOnline,
		}
		result.NetTopologyData = append(result.NetTopologyData, data)
	}

	for _, offlineNode := range offlinePrimaryNodes {
		data := common.NetTopologyData{
			Account:  offlineNode,
			Position: common.PosOffline,
		}
		result.NetTopologyData = append(result.NetTopologyData, data)
	}
	return result
}

func (self *ReElection) paraseNetTopology(topo *common.NetTopology) ([]mc.TopologyNodeInfo, []mc.TopologyNodeInfo, []mc.TopologyNodeInfo, []mc.TopologyNodeInfo, error) {
	if topo.Type != common.NetTopoTypeAll {
		return nil, nil, nil, nil, errors.New("Net Topology is not all data")
	}

	MasterMiner := make([]mc.TopologyNodeInfo, 0)
	BackUpMiner := make([]mc.TopologyNodeInfo, 0)
	MasterValidator := make([]mc.TopologyNodeInfo, 0)
	BackUpValidator := make([]mc.TopologyNodeInfo, 0)

	for _, data := range topo.NetTopologyData {
		node := mc.TopologyNodeInfo{
			Account:  data.Account,
			Position: data.Position,
			Type:     common.GetRoleTypeFromPosition(data.Position),
			Stock:    0,
		}

		switch node.Type {
		case common.RoleMiner:
			MasterMiner = append(MasterMiner, node)
		case common.RoleBackupMiner:
			BackUpMiner = append(BackUpMiner, node)
		case common.RoleValidator:
			MasterValidator = append(MasterValidator, node)
		case common.RoleBackupValidator:
			BackUpValidator = append(BackUpValidator, node)
		}
	}
	return MasterMiner, BackUpMiner, MasterValidator, BackUpValidator, nil
}
