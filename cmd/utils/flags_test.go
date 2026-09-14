// Copyright 2019 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"flag"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/vars"
	"github.com/urfave/cli/v2"
)

// networkContext returns a command-line context holding the network flags,
// parsed from args.
func networkContext(t *testing.T, args ...string) *cli.Context {
	t.Helper()
	app := cli.NewApp()
	app.Flags = slices.Concat(NetworkFlags, []cli.Flag{DeveloperFlag, DeveloperPoWFlag, NetworkIdFlag})
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range app.Flags {
		if err := f.Apply(set); err != nil {
			t.Fatal(err)
		}
	}
	if err := set.Parse(args); err != nil {
		t.Fatal(err)
	}
	return cli.NewContext(app, set, nil)
}

// classicArgs are the command lines that must run Ethereum Classic mainnet.
var classicArgs = [][]string{{}, {"--mainnet"}, {"--classic"}, {"--networkid", "1"}}

// privateArgs are command lines for a private network: no network flag, and a --networkid other
// than Ethereum Classic's. They keep the defaults a node without a network flag had before
// Ethereum Classic became the default, rather than taking Ethereum Classic's. Mordor's network ID
// is among them because only --mordor selects Mordor.
var privateArgs = [][]string{{"--networkid", "42"}, {"--networkid", "7"}}

func TestNetworkDataDir(t *testing.T) {
	base := vars.DefaultDataDir()
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--mordor"}, filepath.Join(base, "mordor")},
		{[]string{"--mintme"}, filepath.Join(base, "mintme")},
		{[]string{"--dev"}, ""},
	}
	for _, args := range classicArgs {
		tests = append(tests, struct {
			args []string
			want string
		}{args, filepath.Join(base, "classic")})
	}
	for _, args := range privateArgs {
		tests = append(tests, struct {
			args []string
			want string
		}{args, base})
	}
	for _, test := range tests {
		cfg := node.Config{DataDir: base}
		SetDataDir(networkContext(t, test.args...), &cfg)
		if cfg.DataDir != test.want {
			t.Errorf("%v: datadir %s, want %s", test.args, cfg.DataDir, test.want)
		}
	}
}

func TestNetworkGenesis(t *testing.T) {
	for _, args := range [][]string{{"--mainnet"}, {"--classic"}} {
		genesis := genesisForCtxChainConfig(networkContext(t, args...))
		if genesis == nil || !reflect.DeepEqual(genesis.Config, params.ClassicChainConfig) {
			t.Errorf("%v: want the Ethereum Classic genesis", args)
		}
	}
	// With no network flag the genesis is settled against the database, which
	// holds a private network's genesis when it was initialized with one.
	if genesis := genesisForCtxChainConfig(networkContext(t)); genesis != nil {
		t.Errorf("no flag: chain ID %v, want nil", genesis.Config.GetChainID())
	}
	if genesis := genesisForCtxChainConfig(networkContext(t, "--mordor")); genesis == nil || !reflect.DeepEqual(genesis.Config, params.MordorChainConfig) {
		t.Error("--mordor: want the Mordor genesis")
	}
}

func TestNetworkBootnodes(t *testing.T) {
	urls := func(nodes []*enode.Node) (out []string) {
		for _, n := range nodes {
			out = append(out, n.URLv4())
		}
		return out
	}
	classic := urls(mustParseBootnodes(params.ClassicBootnodes))
	for _, args := range classicArgs {
		var cfg p2p.Config
		ctx := networkContext(t, args...)
		setBootstrapNodes(ctx, &cfg)
		if have := urls(cfg.BootstrapNodes); !slices.Equal(have, classic) {
			t.Errorf("%v: discv4 bootnodes %v, want Ethereum Classic's", args, have)
		}
		setBootstrapNodesV5(ctx, &cfg)
		if have := urls(cfg.BootstrapNodesV5); !slices.Equal(have, classic) {
			t.Errorf("%v: discv5 bootnodes %v, want Ethereum Classic's", args, have)
		}
	}
	for _, args := range privateArgs {
		var cfg p2p.Config
		ctx := networkContext(t, args...)
		setBootstrapNodes(ctx, &cfg)
		setBootstrapNodesV5(ctx, &cfg)
		if len(cfg.BootstrapNodes) != 0 || len(cfg.BootstrapNodesV5) != 0 {
			t.Errorf("%v: bootnodes %v and %v, want none", args, urls(cfg.BootstrapNodes), urls(cfg.BootstrapNodesV5))
		}
	}
}

func TestUnmaintainedNetworkRefused(t *testing.T) {
	for _, name := range []string{"ethereum", "sepolia", "holesky"} {
		err := checkUnmaintainedNetwork(networkContext(t, "--"+name))
		if err == nil || !strings.Contains(err.Error(), "--"+name+" is deprecated") {
			t.Errorf("--%s: error %v, want the deprecation refusal", name, err)
		}
	}
	for _, args := range slices.Concat(classicArgs, privateArgs, [][]string{{"--mordor"}, {"--mintme"}, {"--dev"}}) {
		if err := checkUnmaintainedNetwork(networkContext(t, args...)); err != nil {
			t.Errorf("%v: refused with %v", args, err)
		}
	}
}

func TestNetworkMinerDefaults(t *testing.T) {
	for _, args := range slices.Concat(classicArgs, [][]string{{"--mordor"}}) {
		ctx := networkContext(t, args...)
		var mcfg miner.Config
		setMiner(ctx, &mcfg)
		if mcfg.GasCeil != 8_000_000 {
			t.Errorf("%v: gas ceiling %d, want 8000000", args, mcfg.GasCeil)
		}
		ecfg := ethconfig.Defaults
		setEthashCacheDir(ctx, &ecfg)
		if ecfg.Ethash.CacheDir != "etchash" {
			t.Errorf("%v: ethash cache dir %q, want etchash", args, ecfg.Ethash.CacheDir)
		}
	}
	for _, args := range privateArgs {
		ctx := networkContext(t, args...)
		var mcfg miner.Config
		setMiner(ctx, &mcfg)
		if mcfg.GasCeil != 0 {
			t.Errorf("%v: gas ceiling %d, want the generic default left in place", args, mcfg.GasCeil)
		}
		ecfg := ethconfig.Defaults
		setEthashCacheDir(ctx, &ecfg)
		if ecfg.Ethash.CacheDir != ethconfig.Defaults.Ethash.CacheDir {
			t.Errorf("%v: ethash cache dir %q, want %q", args, ecfg.Ethash.CacheDir, ethconfig.Defaults.Ethash.CacheDir)
		}
	}
}

func Test_SplitTagsFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args string
		want map[string]string
	}{
		{
			"2 tags case",
			"host=localhost,bzzkey=123",
			map[string]string{
				"host":   "localhost",
				"bzzkey": "123",
			},
		},
		{
			"1 tag case",
			"host=localhost123",
			map[string]string{
				"host": "localhost123",
			},
		},
		{
			"empty case",
			"",
			map[string]string{},
		},
		{
			"garbage",
			"smth=smthelse=123",
			map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SplitTagsFlag(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTagsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}
