package coregeth

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"
)

// The deactivation block is written under ecbp1100DeactivateFBlockFBlock. A chain
// configuration that used ecbp1100DeactivateFBlock, the key ecbp1100FBlock suggests, had its
// deactivation block ignored without an error.
func TestCoreGethChainConfig_ECBP1100DeactivateKeys(t *testing.T) {
	for _, c := range []struct {
		json    string
		want    *big.Int
		refused bool
	}{
		{json: `{"ecbp1100DeactivateFBlock": 20000000}`, want: big.NewInt(20_000_000)},
		{json: `{"ecbp1100DeactivateFBlockFBlock": 20000000}`, want: big.NewInt(20_000_000)},
		{json: `{"ecbp1100DeactivateFBlock": 20000000, "ecbp1100DeactivateFBlockFBlock": 20000000}`, want: big.NewInt(20_000_000)},
		{json: `{"ecbp1100DeactivateFBlock": 20000000, "ecbp1100DeactivateFBlockFBlock": 19250000}`, refused: true},
		{json: `{"ecbp1100FBlock": 11380000}`},
	} {
		var conf CoreGethChainConfig
		err := json.Unmarshal([]byte(c.json), &conf)
		got := conf.ECBP1100DeactivateFBlock
		switch {
		case c.refused:
			if err == nil {
				t.Errorf("%s: decoded, want refused", c.json)
			}
		case err != nil:
			t.Errorf("%s: %v", c.json, err)
		case (got == nil) != (c.want == nil) || (got != nil && got.Cmp(c.want) != 0):
			t.Errorf("%s: deactivation block is %v, want %v", c.json, got, c.want)
		}
	}

	// The rest of the configuration decodes as it did, and writing it back keeps the key
	// every earlier version reads.
	var conf CoreGethChainConfig
	in := `{"networkId": 61, "ecbp1100FBlock": 11380000, "ecbp1100DeactivateFBlock": 20000000}`
	if err := json.Unmarshal([]byte(in), &conf); err != nil {
		t.Fatal(err)
	}
	if conf.NetworkID != 61 || conf.ECBP1100FBlock == nil || conf.ECBP1100FBlock.Uint64() != 11_380_000 {
		t.Errorf("%s: networkId decodes as %d, ecbp1100FBlock as %v", in, conf.NetworkID, conf.ECBP1100FBlock)
	}
	enc, err := json.Marshal(&conf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(enc, []byte(`"ecbp1100DeactivateFBlockFBlock":20000000`)) || bytes.Contains(enc, []byte(`"ecbp1100DeactivateFBlock":`)) {
		t.Errorf("%s: encodes back as %s", in, enc)
	}
}
