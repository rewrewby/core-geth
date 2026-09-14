package ethash

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/types/coregeth"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/holiman/uint256"
)

// A difficulty bomb delay scheduled at a block no chain reaches must change nothing. Its key
// is a uint64 compared against a *big.Int, and converted through int64 a key above
// math.MaxInt64 wrapped negative and applied the delay from genesis.
func TestDifficultyBombDelayScheduleAboveMaxInt64(t *testing.T) {
	parent := &types.Header{
		Number:     big.NewInt(9_999_999),
		Difficulty: big.NewInt(1_000_000_000_000),
		Time:       1_000_000,
		UncleHash:  types.EmptyUncleHash,
	}
	difficulty := func(schedule ctypes.Uint64Uint256MapEncodesHex) *big.Int {
		c := &coregeth.CoreGethChainConfig{
			Ethash:                      new(ctypes.EthashConfig),
			EIP2FBlock:                  big.NewInt(0),
			EIP7FBlock:                  big.NewInt(0),
			DifficultyBombDelaySchedule: schedule,
		}
		return CalcDifficulty(c, parent.Time+13, parent)
	}
	undelayed := difficulty(nil)

	unreachable := uint64(math.MaxUint64 - 1)
	if got := difficulty(ctypes.Uint64Uint256MapEncodesHex{unreachable: uint256.NewInt(3_000_000)}); got.Cmp(undelayed) != 0 {
		t.Errorf("delay scheduled at block %d changed difficulty to %v, want %v", unreachable, got, undelayed)
	}
	// Control: the same delay scheduled at a block already reached does change it.
	if got := difficulty(ctypes.Uint64Uint256MapEncodesHex{1: uint256.NewInt(3_000_000)}); got.Cmp(undelayed) == 0 {
		t.Error("control: a delay scheduled at block 1 left difficulty unchanged")
	}
}
