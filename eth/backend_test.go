package eth

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

func TestMakeExtraDataDefault(t *testing.T) {
	if !bytes.Contains(makeExtraData(nil), []byte("CoreGeth")) {
		t.Error("missing extra data default client identifier")
	}
}

// TestNewResolvesDefaultGenesis checks that a backend given no genesis, on an
// empty database, settles on Ethereum Classic's before it builds the consensus
// engine. The engine takes its ECIP-1099 epoch schedule from that genesis; left
// nil, it would verify every block after the transition with the wrong epoch.
func TestNewResolvesDefaultGenesis(t *testing.T) {
	stack, err := node.New(&node.Config{P2P: p2p.Config{ListenAddr: "127.0.0.1:0", NoDiscovery: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer stack.Close()

	config := ethconfig.Defaults
	config.Ethash.PowMode = ethash.ModeFake
	backend, err := New(stack, &config)
	if err != nil {
		t.Fatal(err)
	}
	if config.Genesis == nil {
		t.Fatal("genesis left nil, want Ethereum Classic's")
	}
	want := params.ClassicChainConfig.GetEthashECIP1099Transition()
	if have := config.Genesis.GetEthashECIP1099Transition(); have == nil || *have != *want {
		t.Errorf("ECIP-1099 transition %v, want %d", have, *want)
	}
	// Ethereum Classic's network ID is 1, not its chain ID. A node that took
	// the chain ID would fail the status handshake with every peer.
	if want := *params.ClassicChainConfig.GetNetworkID(); backend.networkID != want {
		t.Errorf("network ID %d, want %d", backend.networkID, want)
	}
}
