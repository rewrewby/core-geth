// Copyright 2019 The multi-geth Authors
// This file is part of the multi-geth library.
//
// The multi-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The multi-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the multi-geth library. If not, see <http://www.gnu.org/licenses/>.

package params

// MordorBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Ethereum Classic Mordor network.
// https://github.com/etclabscore/mordor/blob/master/static-nodes.json
var MordorBootnodes = []string{
	"enode://d6f2e535a2d183f930bd780913d1d4169d3479f22c2941117d10184d65adbb614866db83049438c17657d1b24a53f3560280f563305426e93868a9a98a02c5fc@2.29.31.123:30303",    // HEL
	"enode://b0b8da5708a22a0bc92e993c6fb93801d1bc5956efaf3ef7cafcd3af2167f634a2147abb6913a719d61ed03e180aa2045ffea7bd2e367db589165e83058df487@40.160.140.180:30303", // PDX

	// Not operated by this project, so this entry cannot be reissued here. Retained
	// until the entries above are proven, then removed with the Old* DNS entries in
	// bootnodes_classic.go.
	"enode://4539a067ae1f6a7ffac509603ba37baf772fc832880ddc67c53f292b6199fb048267f0311c820bc90bfd39ec663bc6b5256bdf787ec38425c82bde6bc2bcfe3c@24.199.107.164:30303", // @etccoop-sfo
}

// See the note on ClassicDNSNetwork1 in bootnodes_classic.go, where dnsPrefixETC
// is declared: three names across one DNS account, and a URL added here is inert
// until cmd/utils/flags.go names it too.
//
// Mordor is the weaker of the two networks. Its published tree carries nodes
// inherited from the deprecated trees below, so when those retire the crawl that
// builds these falls under its own floor and stops publishing rather than
// shrinking. A bootnode this project operates is what keeps that from happening.
var MordorDNSNetwork1 = dnsPrefixETC + "all.mordor.ethereumclassic.net"
var MordorDNSNetwork2 = dnsPrefixETC + "all.mordor.ethclassic.net"
var MordorDNSNetwork3 = dnsPrefixETC + "all.mordor.ethereumclassic.network"
