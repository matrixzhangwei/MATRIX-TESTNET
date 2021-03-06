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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main matrix network.
var MainnetBootnodes = []string{
	"enode://b624a3fb585a48b4c96e4e6327752b1ba82a90a948f258be380ba17ead7c01f6d4ad43d665bb11c50475c058d3aad1ba9a35c0e0c4aa118503bf3ce79609bef6@10.42.100.185:30303",
	/*"enode://dbf8dcc4c82eb2ea2e1350b0ea94c7e29f5be609736b91f0faf334851d18f8de1a518def870c774649db443fbce5f72246e1c6bc4a901ef33429fdc3244a93b3@10.42.100.8:30303",
	"enode://a9f94b62067e993f3f02ada1a448c70ae90bdbe4c6b281f8ff16c6f4832e0e9aba1827531c260b380c776938b9975ac7170a7e822f567660333622ea3e529313@10.42.100.198:30303",
	"enode://80606b6c1eecb8ce91ca8a49a5a183aa42f335eb0d8628824e715571c1f9d1d757911b80ebc3afab06647da228f36ecf1c39cb561ef7684467c882212ce55cdb@10.42.100.164:30303",
	"enode://43b553fae2184b25e76b69a2386bfc9a014486db7da3df75bba9fa2e3eed8aaf063a5f1aab68488a8645fd6a230a27bfe4e8d3393232fe107ba0f68a9bf541ad@10.42.100.176:30303",
	"enode://8ce7defe2dde8297f7b55dd9ba8c5e13e0274371b716250ea0dd725974fa076ca379fc7226789a91678f4e38f8f60f8e6405ec9539cab77d4822614e80f743cf@10.42.100.70:30303",
	"enode://9f237f9842f70b0417d2c25ce987248c991310b2bd4034e300a6eec46b517bd8c4f7f31f157128d0732786181a481bcf725c41a655bdcce282a4bc95638d9aae@10.42.100.155:30303",
	"enode://68315573b123b44367f9fefcce38c4d5c4d5d2caf04158a9068de2060380b81f26b220543de7402745160141f932012a792722fd4dd2a7a2751771097eeef5f2@10.42.100.51:30303",
	"enode://bc5e761c9d0ba42f22433be14973b399662456763f033a4cdbb8ec37b80266526e6c56f92d0591825c7d644e487fcee828d537c58ce583a72578309ec6ebbd39@10.42.100.53:30303",
	"enode://25ea3bca7679192612aed14d5e83a4f2a30824ff2af705d2d7c6795470f9cbbc258d9b102a726c3982cda6c4732ba3715551b6fbf9c0ae4ddca4a6c80bc4bbe9@10.42.100.85:30303",
	"enode://14f62dfd8826734fe75120849e11614b0763bc584fba4135c2f32b19501525d55d217742893801ecc871023fc42ed7e80196357fb5b1f762d181e827e626637d@10.42.100.122:30303",
	"enode://df57387d6505d0f71d7000da9642cf16d44feb7fcaa5f3a8a7d9fa58b6cbb6d33d145746d4fb544c049d3ff9b534bf9245a5b8052231c51695fd298032bd4a79@10.42.100.9:30303",
	"enode://4b2f638f46c7ae5b1564ca7015d716621848a0d9be66f1d1e91d566d2a70eedc2f11e92b743acb8d97dec3fb412c1b2f66afd7fbb9399d4fb2423619eaa514c7@10.42.100.236:30303",*/
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	//	"enode://30b7ab30a01c124a6cceca36863ece12c4f5fa68e3ba9b0b51407ccc002eeed3b3102d20a88f1c1d3c3154e2449317b8ef95090e77b312d5cc39354f86d5d606@52.176.7.10:30303",    // US-Azure gman
	//	"enode://865a63255b3bb68023b6bffd5095118fcc13e79dcf014fe4e47e065c350c7cc72af2e53eff895f11ba1bbb6a2b33271c1116ee870f266618eadfc2e78aa7349c@52.176.100.77:30303",  // US-Azure parity
	//	"enode://6332792c4a00e3e4ee0926ed89e0d27ef985424d97b6a45bf0f23e51f0dcb5e66b875777506458aea7af6f9e4ffb69f43f3778ee73c81ed9d34c51c4b16b0b0f@52.232.243.152:30303", // Parity
	//	"enode://94c15d1b9e2fe7ce56e458b9a3b672ef11894ddedd0c6f247e0f1d3487f52b66208fb4aeb8179fce6e3a749ea93ed147c37976d67af557508d199d9594c35f09@192.81.208.223:30303", // @gpip
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	//	"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303", // IE
	//	"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303",  // INFURA
	//	"enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:30303", // AKASHA
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	//	"enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:30303",
	//	"enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30304",
	//	"enode://1c7a64d76c0334b0418c004af2f67c50e36a3be60b5e4790bdac0439d21603469a85fad36f2473c9a80eb043ae60936df905fa28f1ff614c3e5dc34f15dcd2dc@40.118.3.223:30306",
	//	"enode://85c85d7143ae8bb96924f2b54f1b3e70d8c4d367af305325d30a61385a432f247d2c75c45c6b4a60335060d072d7f5b35dd1d4c45f76941f62a4f83b6e75daaf@40.118.3.223:30307",
}

var BakMinernodes = []string{
	//	"enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:30303",
}
var Broadcastnodes = []string{
	//	"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
}
