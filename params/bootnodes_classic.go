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

// ClassicBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Ethereum Classic network.
var ClassicBootnodes = []string{
	"enode://5c1f4f1c8036879c3186f0632ce0fdf3514e67679f0b5e1814335f5093a21c6c2fda20266ca2b54aac3305085fa46d9711493da1b564489ecec7b37d1aebb5f6@178.104.220.119:30303", // FSN
	"enode://82100619a51fed73bce973a5f7ead3a9468535bdd4c87488037f99bf53b741e4701c5909b7c92897c863c23e494e27c83470a2fcbcc51f8b49655ed377dbd962@135.148.46.64:30303",   // IAD
	"enode://d77bc6696fc2a1419871c8819baef44089cd1847b1b895266382892fd04571cc01a97c45d4df6cbe4c37004195a3c578a76e706d12548916ca5c4341768b1972@62.72.47.101:30303",    // SIN

}

// dnsPrefixETC signs the discovery trees this project publishes and can update.
var dnsPrefixETC = "enrtree://APDLRZ2T7ERXPWXX4D5USB32NIFYHXMFVZQ3DZALK6JJJ5L4VSYIQ@"

// Three domains rather than one. All three zones are served by a single DNS
// account, so separate names are the only redundancy available if one zone
// breaks; they are not independent providers.
//
// A URL added here is inert until it is also named in the EthDiscoveryURLs
// slice in cmd/utils/flags.go, which enumerates these vars individually.
var ClassicDNSNetwork1 = dnsPrefixETC + "all.classic.ethereumclassic.net"
var ClassicDNSNetwork2 = dnsPrefixETC + "all.classic.ethclassic.net"
var ClassicDNSNetwork3 = dnsPrefixETC + "all.classic.ethereumclassic.network"
