package params_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/confp"
	"github.com/ethereum/go-ethereum/params/mutations"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
	"github.com/ethereum/go-ethereum/params/types/genesisT"
	"github.com/ethereum/go-ethereum/params/vars"
)

// The ETC consensus vectors are published for other Ethereum Classic clients to validate
// against, which makes a wrong value in them worse than an absent one. This test is their
// consumer in this client: every value is asserted against what the client declares or
// computes, never against a value restated in this file. The client is the reference, so a
// failure says the two disagree, not which of them is wrong.
//
// It sits beside the chain configuration it checks rather than beside the vectors, whose
// directory holds tests that need a live node and carry the `live` build tag. It is an
// external test package because the reward, epoch, difficulty and genesis functions it
// asserts against live in packages that import params.
const etcConsensusVectorsPath = "../tests/live_etc/testdata/etc_consensus_vectors.json"

type etcConsensusVectors struct {
	Comment string `json:"_comment"`
	Version string `json:"_version"`
	Date    string `json:"_date"`

	ECIP1017Rewards struct {
		Comment          string  `json:"_comment"`
		MainnetEraLength *uint64 `json:"mainnet_era_length"`
		MordorEraLength  *uint64 `json:"mordor_era_length"`
		Vectors          []struct {
			Era               *uint64 `json:"era"`
			BlockRewardWei    string  `json:"block_reward_wei"`
			UncleInclusionWei string  `json:"uncle_inclusion_wei"`
		} `json:"vectors"`
	} `json:"ecip1017_rewards"`

	ECIP1099Epochs struct {
		Comment string `json:"_comment"`
		Vectors []struct {
			Network       string  `json:"network"`
			Block         *uint64 `json:"block"`
			EpochLength   *uint64 `json:"epoch_length"`
			ExpectedEpoch *uint64 `json:"expected_epoch"`
		} `json:"vectors"`
	} `json:"ecip1099_epochs"`

	DifficultyConstraints struct {
		MinimumDifficulty        *uint64 `json:"minimum_difficulty"`
		BombPauseBlockClassic    *uint64 `json:"bomb_pause_block_classic"`
		BombPauseLengthClassic   *uint64 `json:"bomb_pause_length_classic"`
		BombDisposalBlockClassic *uint64 `json:"bomb_disposal_block_classic"`
	} `json:"difficulty_constraints"`

	ECBP1100Windows struct {
		Comment string          `json:"_comment"`
		Classic *ecbp1100Window `json:"classic"`
		Mordor  *ecbp1100Window `json:"mordor"`
	} `json:"ecbp1100_windows"`

	ChainIdentifiers map[string]struct {
		ChainID   *uint64 `json:"chain_id"`
		NetworkID *uint64 `json:"network_id"`
	} `json:"chain_identifiers"`

	GenesisHashes      map[string]string             `json:"genesis_hashes"`
	ForkBlocks         map[string]map[string]*uint64 `json:"fork_blocks"`
	PrecompilesPerFork map[string][]uint64           `json:"precompiles_per_fork"`
}

type ecbp1100Window struct {
	Activation   *uint64       `json:"activation"`
	Deactivation optionalBlock `json:"deactivation"`
}

// optionalBlock records whether its key was present, so a deleted value cannot read as the
// null that means "never".
type optionalBlock struct {
	present bool
	block   *uint64
}

func (b *optionalBlock) UnmarshalJSON(data []byte) error {
	b.present = true
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	return json.Unmarshal(data, &b.block)
}

type etcNetwork struct {
	config  ctypes.ChainConfigurator
	genesis *genesisT.Genesis
	// declaredGenesis is the hash the client identifies the network by, where it declares
	// one. It declares none for Classic, so Classic's genesis is asserted by computation alone.
	declaredGenesis *common.Hash
}

func etcNetworks() map[string]etcNetwork {
	return map[string]etcNetwork{
		"classic": {params.ClassicChainConfig, params.DefaultClassicGenesisBlock(), nil},
		"mordor":  {params.MordorChainConfig, params.DefaultMordorGenesisBlock(), &params.MordorGenesisHash},
	}
}

// etcForkTransitions maps each fork the vectors name to the transitions that make it up in
// this client's configuration. Only the grouping is stated here; heights are read from the
// configuration. A fork resolves only when all of its transitions agree, so one EIP moved
// away from its fork fails rather than only the fork's headline EIP.
func etcForkTransitions(c ctypes.ChainConfigurator) map[string][]func() *uint64 {
	return map[string][]func() *uint64{
		"homestead":         {c.GetEIP2Transition, c.GetEIP7Transition},
		"tangerine_whistle": {c.GetEIP150Transition},
		"diehard":           {c.GetEIP155Transition, c.GetEIP160Transition, c.GetEthashECIP1010PauseTransition},
		"gotham":            {c.GetEthashECIP1017Transition},
		"defuse":            {c.GetEthashECIP1041Transition},
		"atlantis": {
			c.GetEIP161abcTransition, c.GetEIP161dTransition, c.GetEIP170Transition, c.GetEthashEIP100BTransition,
			c.GetEIP140Transition, c.GetEIP198Transition, c.GetEIP211Transition, c.GetEIP212Transition,
			c.GetEIP213Transition, c.GetEIP214Transition, c.GetEIP658Transition,
		},
		"agharta": {c.GetEIP145Transition, c.GetEIP1014Transition, c.GetEIP1052Transition},
		"phoenix": {
			c.GetEIP152Transition, c.GetEIP1108Transition, c.GetEIP1344Transition,
			c.GetEIP1884Transition, c.GetEIP2028Transition, c.GetEIP2200Transition,
		},
		"thanos":   {c.GetEthashECIP1099Transition},
		"magneto":  {c.GetEIP2565Transition, c.GetEIP2718Transition, c.GetEIP2929Transition, c.GetEIP2930Transition},
		"mystique": {c.GetEIP3529Transition, c.GetEIP3541Transition},
		"spiral":   {c.GetEIP3651Transition, c.GetEIP3855Transition, c.GetEIP3860Transition, c.GetEIP6049Transition},
	}
}

func TestETCConsensusVectors(t *testing.T) {
	v := loadETCConsensusVectors(t)
	nets := etcNetworks()
	names := slices.Sorted(maps.Keys(nets))

	t.Run("ecip1017_rewards", func(t *testing.T) {
		r := v.ECIP1017Rewards
		if len(r.Vectors) == 0 {
			t.Fatal("no vectors")
		}
		eraLengths := map[string]*uint64{"classic": r.MainnetEraLength, "mordor": r.MordorEraLength}
		var asserted int
		for _, name := range names {
			c := nets[name].config
			eraLength := c.GetEthashECIP1017EraRounds()
			if !checkU64(t, name+" era length", eraLengths[name], eraLength) {
				continue
			}
			for _, vec := range r.Vectors {
				if vec.Era == nil {
					t.Error("vector without an era")
					continue
				}
				era, length := *vec.Era, *eraLength
				wantReward, wantBonus := parseWei(t, vec.BlockRewardWei), parseWei(t, vec.UncleInclusionWei)
				// The first and the last block of the era: a reduction landing a block early or
				// late puts one of them in the wrong era.
				for _, n := range []uint64{era*length + 1, (era + 1) * length} {
					header := &types.Header{Number: new(big.Int).SetUint64(n)}
					uncle := &types.Header{Number: new(big.Int).SetUint64(n - 1)}
					alone, _ := mutations.GetRewards(c, header, nil)
					withUncle, _ := mutations.GetRewards(c, header, []*types.Header{uncle})
					bonus := new(big.Int).Sub(withUncle.ToBig(), alone.ToBig())
					if alone.ToBig().Cmp(wantReward) != 0 {
						t.Errorf("%s era %d block %d: vectors say block reward %v, client pays %v", name, era, n, wantReward, alone)
					}
					if bonus.Cmp(wantBonus) != 0 {
						t.Errorf("%s era %d block %d: vectors say uncle inclusion %v, client pays %v", name, era, n, wantBonus, bonus)
					}
					asserted++
				}
			}
		}
		t.Logf("%d era reward pairs asserted through mutations.GetRewards", asserted)
	})

	t.Run("ecip1099_epochs", func(t *testing.T) {
		e := v.ECIP1099Epochs
		if len(e.Vectors) == 0 {
			t.Fatal("no vectors")
		}
		var asserted int
		for i, vec := range e.Vectors {
			if vec.Block == nil || vec.EpochLength == nil || vec.ExpectedEpoch == nil {
				t.Errorf("vector %d: block, epoch_length and expected_epoch are all required", i)
				continue
			}
			// A vector naming no network must hold on every network. Near an activation the
			// two networks disagree, and there a vector has to say which one it describes.
			on := names
			if vec.Network != "" {
				if _, ok := nets[vec.Network]; !ok {
					t.Errorf("vector %d: unknown network %q", i, vec.Network)
					continue
				}
				on = []string{vec.Network}
			}
			for _, name := range on {
				length := ethash.CalcEpochLength(*vec.Block, nets[name].config.GetEthashECIP1099Transition())
				epoch := ethash.CalcEpoch(*vec.Block, length)
				if length != *vec.EpochLength || epoch != *vec.ExpectedEpoch {
					t.Errorf("%s block %d: vectors say epoch %d of length %d, client computes epoch %d of length %d",
						name, *vec.Block, *vec.ExpectedEpoch, *vec.EpochLength, epoch, length)
				}
				asserted++
			}
		}
		t.Logf("%d epoch vectors asserted through ethash.CalcEpochLength and ethash.CalcEpoch", asserted)
	})

	t.Run("difficulty_constraints", func(t *testing.T) {
		d := v.DifficultyConstraints
		classic := nets["classic"].config

		if d.MinimumDifficulty == nil {
			t.Error("minimum_difficulty: missing from the vectors")
		} else {
			want := new(big.Int).SetUint64(*d.MinimumDifficulty)
			if vars.MinimumDifficulty.Cmp(want) != 0 {
				t.Errorf("minimum_difficulty: vectors say %v, client declares %v", want, vars.MinimumDifficulty)
			}
			// And the floor the difficulty calculation applies: a parent far below it, at
			// the bomb's disposal so nothing is added on top, adjusts to exactly the floor.
			if disposal := classic.GetEthashECIP1041Transition(); disposal == nil {
				t.Error("minimum_difficulty: client has no bomb disposal block to measure the floor past")
			} else {
				parent := &types.Header{
					Number:     new(big.Int).SetUint64(*disposal),
					Difficulty: big.NewInt(1),
					UncleHash:  types.EmptyUncleHash,
				}
				if got := ethash.CalcDifficulty(classic, 1_000_000, parent); got.Cmp(want) != 0 {
					t.Errorf("minimum_difficulty: vectors say %v, ethash.CalcDifficulty floors at %v", want, got)
				}
			}
		}
		checkU64(t, "bomb_pause_block_classic", d.BombPauseBlockClassic, classic.GetEthashECIP1010PauseTransition())

		// The pause length as the difficulty calculation derives it: continue minus pause.
		var length *uint64
		if pause, cont := classic.GetEthashECIP1010PauseTransition(), classic.GetEthashECIP1010ContinueTransition(); pause != nil && cont != nil {
			l := *cont - *pause
			length = &l
		}
		checkU64(t, "bomb_pause_length_classic", d.BombPauseLengthClassic, length)
		checkU64(t, "bomb_disposal_block_classic", d.BombDisposalBlockClassic, classic.GetEthashECIP1041Transition())
	})

	t.Run("ecbp1100_windows", func(t *testing.T) {
		windows := map[string]*ecbp1100Window{"classic": v.ECBP1100Windows.Classic, "mordor": v.ECBP1100Windows.Mordor}
		for _, name := range names {
			w := windows[name]
			if w == nil {
				t.Errorf("%s: missing from the vectors", name)
				continue
			}
			c := nets[name].config
			checkU64(t, name+" activation", w.Activation, c.GetECBP1100Transition())
			// null says MESS never switches off. An absent key says nothing, so it fails.
			client := c.GetECBP1100DeactivateTransition()
			switch {
			case !w.Deactivation.present:
				t.Errorf("%s deactivation: missing from the vectors (client says %s)", name, formatU64(client))
			case !equalU64(w.Deactivation.block, client):
				t.Errorf("%s deactivation: vectors say %s, client says %s", name, formatU64(w.Deactivation.block), formatU64(client))
			}
		}
	})

	t.Run("chain_identifiers", func(t *testing.T) {
		requireNetworks(t, v.ChainIdentifiers, names)
		for _, name := range names {
			ids, ok := v.ChainIdentifiers[name]
			if !ok {
				continue
			}
			c := nets[name].config
			var chainID *uint64
			if id := c.GetChainID(); id != nil && id.IsUint64() {
				u := id.Uint64()
				chainID = &u
			}
			checkU64(t, name+" chain_id", ids.ChainID, chainID)
			checkU64(t, name+" network_id", ids.NetworkID, c.GetNetworkID())
		}
	})

	t.Run("genesis_hashes", func(t *testing.T) {
		requireNetworks(t, v.GenesisHashes, names)
		for _, name := range names {
			s, ok := v.GenesisHashes[name]
			if !ok {
				continue
			}
			// Decoded strictly: common.HexToHash pads or truncates malformed input silently.
			b, err := hexutil.Decode(s)
			if err != nil || len(b) != common.HashLength {
				t.Errorf("%s: malformed genesis hash %q in vectors", name, s)
				continue
			}
			want := common.BytesToHash(b)
			n := nets[name]
			if computed := core.GenesisToBlock(n.genesis, nil).Hash(); computed != want {
				t.Errorf("%s: vectors say %s, client computes %s from its genesis", name, want.Hex(), computed.Hex())
			}
			if n.declaredGenesis != nil && *n.declaredGenesis != want {
				t.Errorf("%s: vectors say %s, client declares %s", name, want.Hex(), n.declaredGenesis.Hex())
			}
		}
	})

	t.Run("fork_blocks", func(t *testing.T) {
		requireNetworks(t, v.ForkBlocks, names)
		var asserted int
		for _, name := range names {
			forks, ok := v.ForkBlocks[name]
			if !ok {
				continue
			}
			c := nets[name].config
			listed := make(map[uint64]bool)
			for _, fork := range slices.Sorted(maps.Keys(forks)) {
				// A null would decode into a plain integer as 0, which is a real fork height.
				if forks[fork] == nil {
					t.Errorf("%s %s: missing from the vectors", name, fork)
					continue
				}
				want := *forks[fork]
				listed[want] = true
				asserted++
				if fork == "dao_fork_rejected" {
					checkDAORejection(t, name, c, want)
					continue
				}
				got, err := forkBlock(c, fork)
				if err != nil {
					t.Errorf("%s %s: %v", name, fork, err)
					continue
				}
				if got != want {
					t.Errorf("%s %s: vectors say %d, client activates at %d", name, fork, want, got)
				}
			}
			// The other direction: every block the client derives fork identifiers from
			// must be named, so a fork scheduled later cannot be missing from the reference.
			for _, n := range confp.BlockForks(c) {
				if !listed[n] {
					t.Errorf("%s: client forks at block %d, which the vectors do not name", name, n)
				}
			}
		}
		t.Logf("%d fork blocks asserted, and both networks' fork lists checked for completeness", asserted)
	})

	t.Run("precompiles_per_fork", func(t *testing.T) {
		if len(v.PrecompilesPerFork) == 0 {
			t.Fatal("no vectors")
		}
		var asserted int
		for _, fork := range slices.Sorted(maps.Keys(v.PrecompilesPerFork)) {
			want := slices.Sorted(slices.Values(v.PrecompilesPerFork[fork]))
			before := asserted
			for _, name := range names {
				c := nets[name].config
				var heights []uint64
				if fork == "pre_atlantis" {
					atlantis, err := forkBlock(c, "atlantis")
					if err != nil {
						t.Errorf("%s %s: %v", name, fork, err)
						continue
					}
					if atlantis == 0 {
						continue // the network begins at Atlantis, so it has no pre-Atlantis block
					}
					heights = []uint64{0, atlantis - 1}
				} else {
					n, err := forkBlock(c, fork)
					if err != nil {
						t.Errorf("%s %s: %v", name, fork, err)
						continue
					}
					heights = []uint64{n}
				}
				for _, n := range heights {
					if got := activePrecompiles(c, n); !slices.Equal(got, want) {
						t.Errorf("%s %s (block %d): vectors say %v, client installs %v", name, fork, n, want, got)
					}
					asserted++
				}
			}
			if asserted == before {
				t.Errorf("%s: asserted on no network", fork)
			}
		}
		t.Logf("%d precompile sets asserted through vm.PrecompiledContractsForConfig", asserted)
	})
}

func loadETCConsensusVectors(t *testing.T) *etcConsensusVectors {
	t.Helper()
	blob, err := os.ReadFile(etcConsensusVectorsPath)
	if err != nil {
		t.Fatalf("reading vectors: %v", err)
	}
	if err := checkVectorSyntax(blob); err != nil {
		t.Fatalf("vectors: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(blob))
	// A key this test does not read is a vector nothing asserts. Refuse it rather than skip it.
	dec.DisallowUnknownFields()
	v := new(etcConsensusVectors)
	if err := dec.Decode(v); err != nil {
		t.Fatalf("parsing vectors: %v", err)
	}
	return v
}

// checkVectorSyntax refuses what the decoder accepts silently. A key repeated within one
// object keeps only its last value here, while another client's parser may keep the first.
// A key is matched to a struct field regardless of case, so it must be written in lower
// case as every key in the vectors is. And the decoder reads one document, so anything
// after it would be published and never read.
func checkVectorSyntax(blob []byte) error {
	type container struct {
		keys    map[string]bool // nil for an array
		wantKey bool
	}
	var (
		dec   = json.NewDecoder(bytes.NewReader(blob))
		stack []*container
		done  bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if done {
			return fmt.Errorf("content after the document at offset %d", dec.InputOffset())
		}
		if top := len(stack) - 1; top >= 0 && stack[top].keys != nil && stack[top].wantKey && tok != json.Delim('}') {
			key, ok := tok.(string)
			switch {
			case !ok:
				return fmt.Errorf("object key %v is not a string", tok)
			case stack[top].keys[key]:
				return fmt.Errorf("repeated key %q", key)
			case key != strings.ToLower(key):
				return fmt.Errorf("key %q is not lower case", key)
			}
			stack[top].keys[key] = true
			stack[top].wantKey = false
			continue
		}
		switch tok {
		case json.Delim('{'):
			stack = append(stack, &container{keys: make(map[string]bool), wantKey: true})
			continue
		case json.Delim('['):
			stack = append(stack, &container{})
			continue
		case json.Delim('}'), json.Delim(']'):
			stack = stack[:len(stack)-1]
		}
		// A value just ended: a scalar, or the container that closed.
		if top := len(stack) - 1; top < 0 {
			done = true
		} else if stack[top].keys != nil {
			stack[top].wantKey = true
		}
	}
}

// forkBlock returns the height at which the client activates every transition of the
// named fork, or an error if one is unset or they disagree.
func forkBlock(c ctypes.ChainConfigurator, fork string) (uint64, error) {
	fns, ok := etcForkTransitions(c)[fork]
	if !ok {
		return 0, fmt.Errorf("no transitions known for fork %q", fork)
	}
	var at *uint64
	for _, fn := range fns {
		n := fn()
		switch {
		case n == nil:
			return 0, fmt.Errorf("%s is not configured", transitionName(fn))
		case at == nil:
			at = n
		case *n != *at:
			return 0, fmt.Errorf("transitions disagree: %s is at %d, %s at %d", transitionName(fns[0]), *at, transitionName(fn), *n)
		}
	}
	return *at, nil
}

// checkDAORejection asserts a dao_fork_rejected vector from the network's own configuration.
// No transition names that height: the client never applies EIP-779, and its configuration
// declares its own chain's block hash there instead. It declares hashes at some of the
// network's own forks as well, so the vector must name the single declared height that is
// not a fork. A configuration declaring more than one such height cannot say which fork was
// rejected, and fails rather than accepting any of them.
func checkDAORejection(t *testing.T, network string, c ctypes.ChainConfigurator, block uint64) {
	t.Helper()
	if n := c.GetEthashEIP779Transition(); n != nil {
		t.Errorf("%s dao_fork_rejected: client applies the DAO fork at %d", network, *n)
	}
	forks := confp.BlockForks(c)
	var declined []uint64
	for n, hash := range c.GetForkCanonHashes() {
		if hash != (common.Hash{}) && !slices.Contains(forks, n) {
			declined = append(declined, n)
		}
	}
	slices.Sort(declined)
	if len(declined) != 1 || declined[0] != block {
		t.Errorf("%s dao_fork_rejected: vectors say %d, client declares canonical hashes outside its own forks at %v",
			network, block, declined)
	}
}

// activePrecompiles returns the addresses the EVM installs at block n, as the small
// integers the vectors use.
func activePrecompiles(c ctypes.ChainConfigurator, n uint64) []uint64 {
	var out []uint64
	for addr := range vm.PrecompiledContractsForConfig(c, new(big.Int).SetUint64(n), nil) {
		out = append(out, new(big.Int).SetBytes(addr.Bytes()).Uint64())
	}
	slices.Sort(out)
	return out
}

// requireNetworks fails unless a network-keyed section covers exactly the client's networks.
func requireNetworks[V any](t *testing.T, section map[string]V, names []string) {
	t.Helper()
	if got := slices.Sorted(maps.Keys(section)); !slices.Equal(got, names) {
		t.Errorf("networks: vectors cover %v, client has %v", got, names)
	}
}

// checkU64 reports whether a vector matches the client. A missing vector is a failure, not
// agreement.
func checkU64(t *testing.T, what string, vector, client *uint64) bool {
	t.Helper()
	switch {
	case vector == nil:
		t.Errorf("%s: missing from the vectors (client says %s)", what, formatU64(client))
	case !equalU64(vector, client):
		t.Errorf("%s: vectors say %d, client says %s", what, *vector, formatU64(client))
	default:
		return true
	}
	return false
}

func equalU64(a, b *uint64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func formatU64(p *uint64) string {
	if p == nil {
		return "none"
	}
	return strconv.FormatUint(*p, 10)
}

func parseWei(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad decimal %q in vectors", s)
	}
	return v
}

func transitionName(fn func() *uint64) string {
	name := strings.TrimSuffix(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(), "-fm")
	return name[strings.LastIndex(name, ".")+1:]
}
