package genesisT

import (
	"encoding/json"
	"testing"
)

// A genesis file's chain configuration goes through the configuration's own decoder, so the
// deactivation block is read under either of its keys there too.
func TestGenesisECBP1100DeactivateKey(t *testing.T) {
	const genesis = `{
		"config": {"networkId": 7, "chainId": 63, "eip2FBlock": 0, "ecbp1100FBlock": 2380000, "ecbp1100DeactivateFBlock": 20000000},
		"difficulty": "0x20000",
		"gasLimit": "0x7a1200",
		"alloc": {}
	}`
	var g Genesis
	if err := json.Unmarshal([]byte(genesis), &g); err != nil {
		t.Fatal(err)
	}
	got := g.Config.GetECBP1100DeactivateTransition()
	if got == nil {
		t.Fatal("deactivation block not read")
	}
	if *got != 20_000_000 {
		t.Errorf("deactivation block reads back %d, want 20000000", *got)
	}
}
