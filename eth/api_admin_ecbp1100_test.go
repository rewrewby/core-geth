package eth

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params/types/coregeth"
	"github.com/ethereum/go-ethereum/rpc"
)

// A block tag is a negative rpc.BlockNumber. Converted straight to a height, "latest"
// became math.MaxUint64-1: once heights stopped wrapping, admin.ecbp1100("latest") would
// have scheduled MESS for a block no chain reaches instead of the head it names.
func TestEcbp1100ActivationBlock(t *testing.T) {
	const head = 20_000_000
	for _, c := range []struct {
		in      rpc.BlockNumber
		want    uint64
		refused bool
	}{
		{in: rpc.EarliestBlockNumber, want: 0},
		{in: rpc.BlockNumber(11_380_000), want: 11_380_000},
		{in: rpc.LatestBlockNumber, want: head},
		{in: rpc.PendingBlockNumber, want: head},
		{in: rpc.FinalizedBlockNumber, refused: true},
		{in: rpc.SafeBlockNumber, refused: true},
	} {
		got, err := ecbp1100ActivationBlock(c.in, head)
		switch {
		case c.refused && err == nil:
			t.Errorf("%v: resolved to %d, want refused", c.in, got)
		case !c.refused && err != nil:
			t.Errorf("%v: refused: %v", c.in, err)
		case !c.refused && got != c.want:
			t.Errorf("%v: resolved to %d, want %d", c.in, got, c.want)
		}
	}
}

// Block numbers in admin_ecbp1100Status are hex quantities. As JSON numbers they lost
// precision in the console, which showed the off switch's block, 18446744073709551614, as
// 18446744073709552000.
func TestEcbp1100StatusEncoding(t *testing.T) {
	config := &coregeth.CoreGethChainConfig{}
	off := uint64(math.MaxUint64 - 1)
	if err := config.SetECBP1100Transition(&off); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(ecbp1100Status(config, big.NewInt(20_000_000), true))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"enabled":false,"nodeSwitch":true,"activatedAtBlock":"0xfffffffffffffffe","defaultDisabledAtBlock":null,"head":"0x1312d00"}`
	if string(got) != want {
		t.Errorf("status encodes as\n%s\nwant\n%s", got, want)
	}
}
