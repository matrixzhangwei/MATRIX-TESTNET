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
package core

import (
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/mandb"
	"io/ioutil"
	"math/big"
	"testing"
)

type testChainReader struct {
	testHeader map[uint64]*types.Header
	curNumber  uint64
}

func newTestChainReader() *testChainReader {
	tcr := &testChainReader{
		testHeader: make(map[uint64]*types.Header),
		curNumber:  0,
	}

	header0 := &types.Header{
		ParentHash: common.Hash{},
		Number:     big.NewInt(0),
	}
	header0.Elect = make([]common.Elect, 0)
	header0.Elect = append(header0.Elect, common.Elect{
		Account: common.HexToAddress("100A"),
		Stock:   2,
		Type:    common.ElectRoleValidator,
	})
	header0.Elect = append(header0.Elect, common.Elect{
		Account: common.HexToAddress("100B"),
		Stock:   3,
		Type:    common.ElectRoleValidator,
	})
	header0.Elect = append(header0.Elect, common.Elect{
		Account: common.HexToAddress("100C"),
		Stock:   4,
		Type:    common.ElectRoleValidator,
	})
	header0.Elect = append(header0.Elect, common.Elect{
		Account: common.HexToAddress("200A"),
		Stock:   0,
		Type:    common.ElectRoleMiner,
	})

	header0.NetTopology = common.NetTopology{
		Type:            common.NetTopoTypeAll,
		NetTopologyData: make([]common.NetTopologyData, 0),
	}
	header0.NetTopology.NetTopologyData = append(header0.NetTopology.NetTopologyData, common.NetTopologyData{
		Account:  common.HexToAddress("100A"),
		Position: common.GeneratePosition(uint16(0), common.ElectRoleValidator),
	})
	header0.NetTopology.NetTopologyData = append(header0.NetTopology.NetTopologyData, common.NetTopologyData{
		Account:  common.HexToAddress("100B"),
		Position: common.GeneratePosition(uint16(1), common.ElectRoleValidator),
	})
	header0.NetTopology.NetTopologyData = append(header0.NetTopology.NetTopologyData, common.NetTopologyData{
		Account:  common.HexToAddress("100C"),
		Position: common.GeneratePosition(uint16(2), common.ElectRoleValidator),
	})
	header0.NetTopology.NetTopologyData = append(header0.NetTopology.NetTopologyData, common.NetTopologyData{
		Account:  common.HexToAddress("200A"),
		Position: common.GeneratePosition(uint16(0), common.ElectRoleMiner),
	})

	tcr.testHeader[0] = header0

	return tcr
}

func (tc *testChainReader) GetHeaderByNumber(number uint64) *types.Header {
	header, ok := tc.testHeader[number]
	if ok {
		return header
	} else {
		return nil
	}
}

func (tc *testChainReader) CurrentHeader() *types.Header {
	header, _ := tc.testHeader[tc.curNumber]
	return header
}

func TestTopologyStore_GetTopologyGraphByNumber(t *testing.T) {
	workspace, err := ioutil.TempDir("", "topology_store_test-")
	if err != nil {
		t.Fatalf("创建workspace失败, %v", err)
	}

	chainReader := newTestChainReader()
	db, err := mandb.NewLDBDatabase(workspace, 0, 0)
	if err != nil {
		t.Fatalf("创建db错误, %v", err)
	}

	store := NewTopologyStore(chainReader, db)
	store.WriteTopologyGraph(chainReader.GetHeaderByNumber(0))
}
