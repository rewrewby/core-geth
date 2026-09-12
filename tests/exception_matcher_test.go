// Copyright 2026 The core-geth Authors
// This file is part of the core-geth library.
//
// The core-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The core-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the core-geth library. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestSplitExceptionNames covers step 1 of the contract.
func TestSplitExceptionNames(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"TR_IntrinsicGas", []string{"TR_IntrinsicGas"}},
		{"A|B", []string{"A", "B"}},
		{" A | B ", []string{"A", "B"}},
		{"A||B", []string{"A", "B"}},
		{"|", nil},
		{"", nil},
	} {
		got := splitExceptionNames(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitExceptionNames(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitExceptionNames(%q) = %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}

// refusalAcceptCases covers step 3: a stated name that matches the refusal
// actually raised is a pass.
//
// It is a package-level value because TestEveryVocabularyEntryIsExercised reads
// it back to prove the table covers every vocabulary entry. Adding a vocabulary
// entry without adding an arm here therefore fails rather than shipping untested.
//
// Where an entry matches on message content, the arm carries the literal string
// the client actually raises, cited to where it is raised, so that rewording
// that error is caught here rather than in a corpus run.
var refusalAcceptCases = []struct {
	name  string
	field string
	err   error
}{
	{"intrinsic gas, TR_ spelling", "TR_IntrinsicGas", core.ErrIntrinsicGas},
	{"intrinsic gas, unprefixed", "IntrinsicGas", core.ErrIntrinsicGas},
	{"type not supported", "TR_TypeNotSupported", types.ErrTxTypeNotSupported},
	{"type not supported, blob spelling", "TR_TypeNotSupportedBlob", types.ErrTxTypeNotSupported},
	{"no funds", "TR_NoFunds", core.ErrInsufficientFunds},
	{"no funds for transfer", "TR_NoFunds", core.ErrInsufficientFundsForTransfer},
	{"no funds, X spelling", "TR_NoFundsX", core.ErrInsufficientFunds},
	{"disjunction, funds arm", "TR_NoFundsOrGas", core.ErrInsufficientFunds},
	{"disjunction, gas arm", "TR_NoFundsOrGas", core.ErrIntrinsicGas},
	{"sender not eoa", "SenderNotEOA", core.ErrSenderNoEOA},
	{"nonce max", "TR_NonceHasMaxValue", core.ErrNonceMax},
	{"nonce too low", "TR_NonceTooLow", core.ErrNonceTooLow},
	{"nonce too high", "TR_NonceTooHigh", core.ErrNonceTooHigh},
	{"fee cap below base fee", "TR_FeeCapLessThanBlocks", core.ErrFeeCapTooLow},
	{"tip above fee cap", "TR_TipGtFeeCap", core.ErrTipAboveFeeCap},
	{"fee cap or funds, fee arm", "TR_FeeCapLessThanBlocksORNoFunds", core.ErrFeeCapTooLow},
	{"fee cap or funds, funds arm", "TR_FeeCapLessThanBlocksORNoFunds", core.ErrInsufficientFunds},
	{"fee cap or gas limit, fee arm", "TR_FeeCapLessThanBlocksORGasLimitReached", core.ErrFeeCapTooLow},
	{"fee cap or gas limit, limit arm", "TR_FeeCapLessThanBlocksORGasLimitReached", core.ErrGasLimitReached},
	{"gas limit reached", "TR_GasLimitReached", core.ErrGasLimitReached},
	{"initcode limit", "TR_InitCodeLimitExceeded", core.ErrMaxInitCodeSizeExceeded},
	{"wrapped error is still reached", "TR_IntrinsicGas",
		fmt.Errorf("applying transaction: %w", core.ErrIntrinsicGas)},
	{"bar-separated, second name matches", "TR_NoFunds|TR_IntrinsicGas", core.ErrIntrinsicGas},
	// An over-256-bit value is reported without the word "RLP"; the name
	// still denotes this refusal. Measured from stTransactionTest/ValueOverflow.
	{"wrong value, over-256-bit spelling", "TR_RLP_WRONGVALUE",
		errors.New(`invalid tx value "0x:bigint 0x10000000000000000000000000000000000000000000000000000000000000001"`)},
	// The bare "rlp" spelling is deliberately NOT accepted -- it would make this
	// name non-disjoint from TR_BLOBCREATE, whose refusal is also an RLP decode
	// failure. TestMatchExpectExceptionRejects holds it there. The two arms below
	// cover the sentinel branch, which nothing exercised before.
	{"wrong value, v/r/s spelling", "TR_RLP_WRONGVALUE",
		errors.New("invalid transaction v, r, s values")},
	{"wrong value, invalid signature sentinel", "TR_RLP_WRONGVALUE", types.ErrInvalidSig},
	{"wrong value, invalid tx type sentinel", "TR_RLP_WRONGVALUE", types.ErrInvalidTxType},
	{"bar-separated, unknown name alongside a match", "NotAName|TR_IntrinsicGas", core.ErrIntrinsicGas},

	// EIP-4844 blob refusals. The two content-matched arms carry the phrases
	// raised at core/state_transition.go's preCheck and in RunNoVerify below.
	{"empty blob list", "TR_EMPTYBLOB", core.ErrMissingBlobHashes},
	{"blob transaction of type create", "TR_BLOBCREATE", core.ErrBlobTxCreate},
	{"invalid blob hash version", "TR_BLOBVERSION_INVALID",
		fmt.Errorf("blob %d has invalid hash version", 0)},
	{"blob gas over the block maximum", "TR_BLOBLIST_OVERSIZE",
		errors.New("blob gas exceeds maximum")},

	// execution-spec-tests vocabulary. These names are the ones TestExecutionSpecState
	// exercises against the downloaded corpus; the arms here assert the mapping
	// itself, which that suite cannot do for a name no fixture happens to state.
	{"eest insufficient funds", "TransactionException.INSUFFICIENT_ACCOUNT_FUNDS", core.ErrInsufficientFunds},
	{"eest insufficient funds for transfer", "TransactionException.INSUFFICIENT_ACCOUNT_FUNDS",
		core.ErrInsufficientFundsForTransfer},
	{"eest intrinsic gas", "TransactionException.INTRINSIC_GAS_TOO_LOW", core.ErrIntrinsicGas},
	{"eest initcode size", "TransactionException.INITCODE_SIZE_EXCEEDED", core.ErrMaxInitCodeSizeExceeded},
	{"eest fee cap below base fee", "TransactionException.INSUFFICIENT_MAX_FEE_PER_GAS", core.ErrFeeCapTooLow},
	{"eest type 3 pre fork", "TransactionException.TYPE_3_TX_PRE_FORK", types.ErrTxTypeNotSupported},
	{"eest type 3 zero blobs", "TransactionException.TYPE_3_TX_ZERO_BLOBS", core.ErrMissingBlobHashes},
	{"eest blob fee cap", "TransactionException.INSUFFICIENT_MAX_FEE_PER_BLOB_GAS", core.ErrBlobFeeCapTooLow},
	{"eest invalid versioned hash", "TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH",
		fmt.Errorf("blob %d has invalid hash version", 0)},
	{"eest blob count exceeded", "TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED",
		errors.New("blob gas exceeds maximum")},
	// The one bar-separated value the corpus actually states, reproduced verbatim.
	{"eest bar-separated, as stated by the corpus",
		"TransactionException.TYPE_3_TX_PRE_FORK|TransactionException.TYPE_3_TX_ZERO_BLOBS",
		types.ErrTxTypeNotSupported},
}

func TestMatchExpectExceptionAccepts(t *testing.T) {
	for _, tc := range refusalAcceptCases {
		if err := matchExpectException(tc.field, tc.err); err != nil {
			t.Errorf("%s: matchExpectException(%q, %v) = %v, want nil",
				tc.name, tc.field, tc.err, err)
		}
	}
}

// TestMatchExpectExceptionRejects is the calibration. A check that cannot report
// a negative proves nothing, so every arm here must fail -- and the failure must
// name what it saw.
//
// The content-matched entries are the fragile ones, because a substring test
// accepts anything containing it. Each has an arm below stating a refusal that
// shares words with the phrase but is a different rule, so a matcher loosened to
// a shorter fragment fails here. Nothing else in the suite can catch that: a
// corpus run only ever exercises the accepting direction.
func TestMatchExpectExceptionRejects(t *testing.T) {
	for _, tc := range []struct {
		name      string
		field     string
		err       error
		wantMatch string // substring the failure must carry
	}{
		{
			name: "wrong rule refused: the whole point of the contract",
			// A fixture asserting the transaction type was rejected must not be
			// satisfied by an intrinsic-gas failure.
			field: "TR_TypeNotSupported", err: core.ErrIntrinsicGas,
			wantMatch: "expected refusal",
		},
		{
			name:  "unmapped name must fail loudly, not be waved through",
			field: "TR_SomeNameNobodyMapped", err: core.ErrIntrinsicGas,
			wantMatch: "unmapped expectException",
		},
		{
			name:  "all names unmapped",
			field: "Nope|AlsoNope", err: core.ErrIntrinsicGas,
			wantMatch: "unmapped expectException",
		},
		{
			// Mixed known and unknown, none matching. This is INDETERMINATE, not a
			// wrong refusal: one of the unknown alternatives may be the rule that
			// actually fired, so the remedy is to extend the vocabulary rather than
			// to go looking at the client. Matching the mixed branch's own wording
			// is deliberate -- "unmapped expectException" alone is satisfied by the
			// all-unknown arm above, which would leave this branch exercised by
			// nothing.
			name:  "a known name beside an unknown one is indeterminate, not a wrong refusal",
			field: "TR_IntrinsicGas|NotAName", err: core.ErrNonceTooLow,
			wantMatch: "not every stated name is",
		},
		{
			name:  "malformed field states no name",
			field: "|", err: core.ErrIntrinsicGas,
			wantMatch: "malformed expectException",
		},
		{
			name:  "nonce rules are distinguished from each other",
			field: "TR_NonceTooLow", err: core.ErrNonceTooHigh,
			wantMatch: "expected refusal",
		},
		{
			name:  "fee-cap rule is not satisfied by a balance failure",
			field: "TR_FeeCapLessThanBlocks", err: core.ErrInsufficientFunds,
			wantMatch: "expected refusal",
		},
		{
			name:  "an unrelated bug that happens to error is not a pass",
			field: "TR_IntrinsicGas", err: errors.New("some unrelated internal failure"),
			wantMatch: "expected refusal",
		},

		// --- content-matched entries: the loosening these guard against --------
		{
			// "exceeds maximum" alone would accept this. The entry matches the
			// full "blob gas exceeds maximum" for exactly this reason.
			name:      "blob list oversize is not satisfied by an unrelated size failure",
			field:     "TR_BLOBLIST_OVERSIZE",
			err:       errors.New("payload of 5 bytes exceeds maximum of 4"),
			wantMatch: "expected refusal",
		},
		{
			name:  "blob list oversize is not satisfied by a different blob rule",
			field: "TR_BLOBLIST_OVERSIZE", err: core.ErrMissingBlobHashes,
			wantMatch: "expected refusal",
		},
		{
			// A fragment of the phrase must not be enough.
			name:  "blob hash version needs the whole phrase, not a prefix of it",
			field: "TR_BLOBVERSION_INVALID", err: errors.New("blob 0 has an invalid hash"),
			wantMatch: "expected refusal",
		},
		{
			name:      "blob hash version is not satisfied by another invalid-something",
			field:     "TR_BLOBVERSION_INVALID",
			err:       errors.New("invalid transaction v, r, s values"),
			wantMatch: "expected refusal",
		},
		{
			// The comment on the entry says a bare "rlp" substring is deliberately
			// not matched; this is the arm that holds it to that.
			name:      "wrong value is not satisfied by an unrelated rlp decode failure",
			field:     "TR_RLP_WRONGVALUE",
			err:       errors.New("rlp: input string too short for common.Address, decoding into (types.BlobTx).To"),
			wantMatch: "expected refusal",
		},
		{
			name:  "wrong value is not satisfied by a nonce rule",
			field: "TR_RLP_WRONGVALUE", err: core.ErrNonceTooLow,
			wantMatch: "expected refusal",
		},
		{
			name:      "eest blob count is not satisfied by an unrelated size failure",
			field:     "TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED",
			err:       errors.New("payload of 5 bytes exceeds maximum of 4"),
			wantMatch: "expected refusal",
		},
		{
			name:      "eest versioned hash is not satisfied by a fragment",
			field:     "TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH",
			err:       errors.New("blob 0 has an invalid hash"),
			wantMatch: "expected refusal",
		},
		{
			// A nil refusal must never satisfy a content matcher. checkError does
			// not call this with nil, but hasText guards it and the guard is only
			// load-bearing if something proves it.
			name:  "a nil error satisfies no content matcher",
			field: "TR_BLOBLIST_OVERSIZE", err: nil,
			wantMatch: "expected refusal",
		},
	} {
		err := matchExpectException(tc.field, tc.err)
		if err == nil {
			t.Errorf("%s: matchExpectException(%q, %v) = nil, want a failure",
				tc.name, tc.field, tc.err)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantMatch) {
			t.Errorf("%s: failure %q does not carry %q", tc.name, err, tc.wantMatch)
		}
		// The failure must name the refusal actually raised, or it cannot be
		// acted on.
		if tc.err != nil && !strings.Contains(err.Error(), tc.err.Error()) {
			t.Errorf("%s: failure %q does not name the actual error %q",
				tc.name, err, tc.err)
		}
	}
}

// TestEveryVocabularyEntryIsExercised requires refusalAcceptCases to cover every
// vocabulary entry.
//
// A vocabulary entry with no arm is untested in both directions at once: a
// corpus run exercises only the names its fixtures happen to state, and a name
// no fixture states is exercised by nothing at all. This makes adding an entry
// without a test a failure rather than a silence.
func TestEveryVocabularyEntryIsExercised(t *testing.T) {
	// Credit a name only when its own matcher accepts the arm's error. Crediting
	// every name in a bar-separated field would over-report: in
	// "TR_NoFunds|TR_IntrinsicGas" against ErrIntrinsicGas only the second name
	// is actually exercised.
	exercised := make(map[string]bool)
	for _, tc := range refusalAcceptCases {
		for _, n := range splitExceptionNames(tc.field) {
			if m, ok := refusalVocabulary[n]; ok && m(tc.err) {
				exercised[n] = true
			}
		}
	}
	var missing []string
	for _, n := range knownRefusalNames() {
		if !exercised[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d vocabulary entries have no accepting test case: %s\n"+
			"(add an arm to refusalAcceptCases in tests/exception_matcher_test.go)",
			len(missing), strings.Join(missing, ", "))
	}
	// Calibration: the credit rule must be capable of NOT crediting. A name whose
	// matcher rejects the arm must not count as exercised.
	if m, ok := refusalVocabulary["TR_IntrinsicGas"]; !ok || m(core.ErrNonceTooLow) {
		t.Error("calibration failed: TR_IntrinsicGas must reject ErrNonceTooLow, " +
			"or the crediting rule above cannot discriminate")
	}
}

// anticipatoryRefusalNames are vocabulary entries no fixture in any corpus a
// state-test entry point walks currently states.
//
// They are not speculative. Every one of them is stated by fixtures in the
// BlockchainTests corpora, which no state-test entry point walks and which has
// the same unmapped-name gap this contract closes for state tests. Read the
// current counts rather than trusting a number here, which would rot:
//
//	grep -rc '"expectException"[[:space:]]*:[[:space:]]*"<name>"' \
//	    tests/testdata/BlockchainTests tests/testdata-etc/BlockchainTests
//
// Declaring them rather than deleting them keeps two things loud at once: a NEW
// entry added without corpus usage is not silently absorbed into this set, and
// an entry here that a corpus revision starts stating is reported so the
// declaration can be retired.
var anticipatoryRefusalNames = []string{
	"TR_FeeCapLessThanBlocksORGasLimitReached",
	"TR_FeeCapLessThanBlocksORNoFunds",
	"TR_NonceTooHigh",
	"TR_NonceTooLow",
}

// refusalCensusFile is the shape shared by every state-test fixture these suites
// walk: a map of test name to an object whose "post" field holds, per fork, the
// post-state entries a subtest runs against. Only the refusal statement is
// modelled; everything else is discarded.
type refusalCensusFile map[string]struct {
	Post map[string][]struct {
		ExpectException string `json:"expectException"`
	} `json:"post"`
}

// censusRefusalNames reads every fixture under dir and returns the refusal names
// stated there with their occurrence counts, plus the number of post entries
// carrying a statement.
func censusRefusalNames(t *testing.T, dir string) (map[string]int, int) {
	t.Helper()
	names := make(map[string]int)
	entries := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Prefilter. The corpora are hundreds of megabytes across thousands of
		// files and only a few dozen state a refusal, so decoding every file
		// would make this test too slow to keep. A JSON-escaped spelling of the
		// key would slip past this; step 4 of the contract is what catches that
		// at run time, naming the unmapped name in a hard failure.
		if !bytes.Contains(data, []byte(`"expectException"`)) {
			return nil
		}
		var file refusalCensusFile
		if err := json.Unmarshal(data, &file); err != nil {
			// A fixture that states a refusal and cannot be read is a blind spot,
			// not something to pass over.
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, test := range file {
			for _, posts := range test.Post {
				for _, p := range posts {
					if p.ExpectException == "" {
						continue
					}
					entries++
					for _, n := range splitExceptionNames(p.ExpectException) {
						names[n]++
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("censusing %s: %v", dir, err)
	}
	return names, entries
}

// TestRefusalVocabularyCoversExecutedCorpora asserts that every refusal name
// stated anywhere in the corpora the state-test entry points walk has a
// vocabulary entry, so that step 4 of the contract does not fire in bulk at run
// time on names that were simply never mapped.
//
// The name set is derived from the corpora rather than transcribed. A
// transcribed list is what this test carried before, and it passed while ~700
// execution-spec-tests subtests were about to fail as unmapped: the list was
// measured over the directories TestState walks, TestExecutionSpecState walks a
// third corpus, and nothing connected the two. The directory list now comes from
// refusalCorpusDirs, which is built from the entry points' own lists.
//
// A corpus that is absent cannot be censused, so the result is a SKIP naming
// what was missed rather than a pass over a partial reading. Coverage is still
// asserted over whatever IS present, and a miss there is a failure: t.Errorf
// followed by t.Skipf reports FAIL, so a real gap is never masked by a skip.
func TestRefusalVocabularyCoversExecutedCorpora(t *testing.T) {
	measured := make(map[string]int)
	total := 0
	var present, absent []string
	for _, dir := range refusalCorpusDirs() {
		if !common.FileExist(dir) {
			absent = append(absent, dir)
			continue
		}
		present = append(present, dir)
		names, entries := censusRefusalNames(t, dir)
		total += entries
		for n, c := range names {
			measured[n] += c
		}
	}

	// Calibration, and the one that matters most: a census that found nothing
	// would satisfy every assertion below while measuring nothing at all.
	if total == 0 {
		t.Fatalf("censused %d corpora and found no expectException entries at all; "+
			"the census is not measuring anything", len(present))
	}
	t.Logf("censused %d corpora: %d statements, %d distinct names", len(present), total, len(measured))

	stated := make([]string, 0, len(measured))
	for n := range measured {
		stated = append(stated, n)
	}
	sort.Strings(stated)
	for _, n := range stated {
		if _, ok := refusalVocabulary[n]; !ok {
			t.Errorf("refusal name %q is stated %d times in the executed corpora but has no "+
				"vocabulary entry; every subtest stating it fails as unmapped "+
				"(add it to refusalVocabulary in tests/exception_matcher.go)", n, measured[n])
		}
	}

	if len(absent) > 0 {
		t.Skipf("censused %d of %d corpora; NOT censused: %s. Coverage was asserted over the "+
			"corpora present, which is not a full reading -- initialise the test submodules and "+
			"fetch tests/spec-tests before treating this as a coverage result.",
			len(present), len(present)+len(absent), strings.Join(absent, ", "))
	}

	// Every corpus was read, so the vocabulary entries nothing stated are a real
	// reading rather than an artifact of a missing corpus. Report the difference
	// against what is declared, in both directions, as a list.
	declared := make(map[string]bool, len(anticipatoryRefusalNames))
	for _, n := range anticipatoryRefusalNames {
		declared[n] = true
	}
	var undeclaredUnused, declaredButUsed []string
	for _, n := range knownRefusalNames() {
		switch {
		case measured[n] == 0 && !declared[n]:
			undeclaredUnused = append(undeclaredUnused, n)
		case measured[n] > 0 && declared[n]:
			declaredButUsed = append(declaredButUsed, n)
		}
	}
	if len(undeclaredUnused) > 0 {
		t.Errorf("%d vocabulary entries are stated by no fixture in any executed corpus: %s\n"+
			"(if the rule is real but unexercised, declare it in anticipatoryRefusalNames; "+
			"if the name is wrong or obsolete, remove the entry)",
			len(undeclaredUnused), strings.Join(undeclaredUnused, ", "))
	}
	if len(declaredButUsed) > 0 {
		t.Errorf("%d entries declared anticipatory are now stated by a fixture: %s\n"+
			"(remove them from anticipatoryRefusalNames; the corpora have moved)",
			len(declaredButUsed), strings.Join(declaredButUsed, ", "))
	}
}
