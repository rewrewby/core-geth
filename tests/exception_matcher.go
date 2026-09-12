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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// A refusal fixture states which rule rejected a transaction. Comparing that
// statement against the refusal actually produced is the only discriminator
// available: a refusal leaves the pre-state untouched, so the post-state root is
// identical whichever rule did the refusing.
//
// The contract implemented here has four steps:
//
//  1. Split the stated field on '|', trim, and keep the names verbatim and
//     unmapped. Fixtures state several names because no single name is
//     understood by every consumer.
//  2. Intersect that set with the refusals this build can actually produce.
//  3. A non-empty intersection matching the actual refusal is a pass.
//  4. An empty intersection is a divergence naming the unmatched rule -- never a
//     skip and never a pass.
//
// Step 4 is the load-bearing one. An unrecognised name has to fail loudly, or
// the gap this contract closes simply moves: a fixture would go on asserting
// nothing, and the suite would go on reporting green.

// refusalMatcher reports whether err is the refusal a given fixture name denotes.
type refusalMatcher func(error) bool

// isErr matches any error in the chain of one of the given sentinels.
func isErr(targets ...error) refusalMatcher {
	return func(err error) bool {
		for _, t := range targets {
			if errors.Is(err, t) {
				return true
			}
		}
		return false
	}
}

// hasText matches on message content. Reserved for refusals this client raises
// as a formatted error rather than a sentinel, so errors.Is cannot reach them.
func hasText(subs ...string) refusalMatcher {
	return func(err error) bool {
		if err == nil {
			return false
		}
		msg := strings.ToLower(err.Error())
		for _, s := range subs {
			if strings.Contains(msg, strings.ToLower(s)) {
				return true
			}
		}
		return false
	}
}

// anyOf matches when any constituent matcher matches. Used for fixture names
// that deliberately name a disjunction, such as TR_NoFundsOrGas.
func anyOf(ms ...refusalMatcher) refusalMatcher {
	return func(err error) bool {
		for _, m := range ms {
			if m(err) {
				return true
			}
		}
		return false
	}
}

// refusalVocabulary maps each refusal name appearing in the pinned corpora onto
// the refusals this build can produce.
//
// THREE vocabularies are in circulation and none is complete. The legacy `TR_`
// names from the retesteth era and the unprefixed names beside them both appear
// in the corpora TestState walks. The modern `TransactionException.*` enum is
// used by the execution-spec-tests fixtures, which TestExecutionSpecState runs
// from a directory CI downloads and a local checkout usually lacks. All three
// are held here verbatim; no name is rewritten into a canonical spelling,
// because a mapping step is where a wrong assumption would hide.
//
// **Covering only two of the three is how this table first shipped**, and the
// local suites could not catch it: `tests/spec-tests/` is gitignored, so
// TestExecutionSpecState skips locally and runs in CI. Adding a corpus means
// re-censusing it, not assuming the names resemble one already here.
//
// A name absent from this table is not passed over. It fails under step 4 and
// names itself in the failure, which is what makes an unmapped name visible
// rather than silently green.
var refusalVocabulary = map[string]refusalMatcher{
	// Intrinsic gas.
	"TR_IntrinsicGas": isErr(core.ErrIntrinsicGas),
	"IntrinsicGas":    isErr(core.ErrIntrinsicGas),

	// Transaction type not supported by the active fork. core.ErrTxTypeNotSupported
	// is a straight alias of the types sentinel, so naming both would pass the
	// same value twice.
	"TR_TypeNotSupported": isErr(types.ErrTxTypeNotSupported),
	// Blob-suffixed spelling. It is EIP-4844 material from the Ethereum corpus
	// and denotes the same client refusal; it is NOT the canonical spelling and
	// must not be adopted when authoring a fixture for a chain without blob
	// transactions.
	"TR_TypeNotSupportedBlob": isErr(types.ErrTxTypeNotSupported),

	// Balance.
	"TR_NoFunds":  isErr(core.ErrInsufficientFunds, core.ErrInsufficientFundsForTransfer),
	"TR_NoFundsX": isErr(core.ErrInsufficientFunds, core.ErrInsufficientFundsForTransfer),
	"TR_NoFundsOrGas": anyOf(
		isErr(core.ErrInsufficientFunds, core.ErrInsufficientFundsForTransfer),
		isErr(core.ErrIntrinsicGas),
	),

	// Sender must be an externally owned account.
	"SenderNotEOA": isErr(core.ErrSenderNoEOA),

	// Nonce.
	"TR_NonceHasMaxValue": isErr(core.ErrNonceMax),
	"TR_NonceTooLow":      isErr(core.ErrNonceTooLow),
	"TR_NonceTooHigh":     isErr(core.ErrNonceTooHigh),

	// Fee market. types.ErrGasFeeCapTooLow reads like a second spelling of
	// core.ErrFeeCapTooLow and is deliberately NOT matched here: it is raised
	// only by types.Transaction.EffectiveGasTip and by the miner's ordering, and
	// neither is on the path from RunNoVerify to checkError. Naming it would
	// widen the matcher by a value no state test can produce, which is a claim
	// about this build's refusals that is not true.
	"TR_FeeCapLessThanBlocks": isErr(core.ErrFeeCapTooLow),
	"TR_TipGtFeeCap":          isErr(core.ErrTipAboveFeeCap),
	"TR_FeeCapLessThanBlocksORNoFunds": anyOf(
		isErr(core.ErrFeeCapTooLow),
		isErr(core.ErrInsufficientFunds, core.ErrInsufficientFundsForTransfer),
	),
	"TR_FeeCapLessThanBlocksORGasLimitReached": anyOf(
		isErr(core.ErrFeeCapTooLow),
		isErr(core.ErrGasLimitReached),
	),

	// Block gas.
	"TR_GasLimitReached": isErr(core.ErrGasLimitReached),

	// EIP-3860 initcode metering.
	"TR_InitCodeLimitExceeded": isErr(core.ErrMaxInitCodeSizeExceeded),

	// Malformed transaction encoding. The name denotes "a value carried in the
	// transaction's RLP is wrong or unrepresentable". This client raises that as
	// a formatted error rather than a sentinel, and not every spelling mentions
	// RLP -- a value exceeding 256 bits is reported as "invalid tx value" -- so
	// the observed spellings are matched explicitly.
	//
	// **A bare "rlp" substring is deliberately NOT matched here.** It would accept
	// every RLP decode failure regardless of which field failed, including the one
	// a sibling fixture labels TR_BLOBCREATE ("rlp: input string too short for
	// common.Address, decoding into (types.BlobTx).To") -- making two names that
	// denote different rules non-disjoint, which is the original false-pass
	// narrowed to the encoding class rather than removed from it. The blockchain
	// corpus distinguishes RLP failures at field level, so a broad bucket here
	// could not express what that runner will need either.
	"TR_RLP_WRONGVALUE": anyOf(
		isErr(types.ErrInvalidSig, types.ErrInvalidTxType),
		hasText("invalid tx value", "invalid transaction v, r, s values"),
	),

	// EIP-4844 blob transactions. Present in the Ethereum corpus only; a chain
	// without blob transactions cannot produce these.
	"TR_EMPTYBLOB":           isErr(core.ErrMissingBlobHashes),
	"TR_BLOBCREATE":          isErr(core.ErrBlobTxCreate),
	"TR_BLOBVERSION_INVALID": hasText("invalid hash version"),
	// Matched on the harness's own full phrase, raised in RunNoVerify. The
	// shorter "exceeds maximum" would match any future error using those words,
	// and the "too many blobs" / "blob list" spellings this once carried are
	// raised nowhere reachable from checkError -- verified with a calibrated grep.
	"TR_BLOBLIST_OVERSIZE": hasText("blob gas exceeds maximum"),

	// --- execution-spec-tests (EEST) vocabulary -------------------------------
	//
	// Run by TestExecutionSpecState from tests/spec-tests/, which CI downloads
	// (build/ci.go's downloadSpecTestFixtures) and which is gitignored, so this
	// suite SKIPS in a local checkout unless the fixtures are fetched. Verify a
	// change to these names against that suite specifically; TestState passing
	// says nothing about them.
	//
	// Censused at the pinned version in build/checksums.txt: 175 post-entries
	// across 8 of the 72 state_tests fixture files, stating ten distinct field
	// values that reduce to the nine names below. One of the ten is
	// bar-separated, so the '|' handling above is exercised here even though no
	// TR_ fixture uses it. The census is no longer transcribed into a test:
	// TestRefusalVocabularyCoversExecutedCorpora rebuilds it from the corpora at
	// run time.
	"TransactionException.INSUFFICIENT_ACCOUNT_FUNDS": isErr(
		core.ErrInsufficientFunds, core.ErrInsufficientFundsForTransfer),
	"TransactionException.INTRINSIC_GAS_TOO_LOW":             isErr(core.ErrIntrinsicGas),
	"TransactionException.INITCODE_SIZE_EXCEEDED":            isErr(core.ErrMaxInitCodeSizeExceeded),
	"TransactionException.INSUFFICIENT_MAX_FEE_PER_GAS":      isErr(core.ErrFeeCapTooLow),
	"TransactionException.TYPE_3_TX_PRE_FORK":                isErr(types.ErrTxTypeNotSupported),
	"TransactionException.TYPE_3_TX_ZERO_BLOBS":              isErr(core.ErrMissingBlobHashes),
	"TransactionException.INSUFFICIENT_MAX_FEE_PER_BLOB_GAS": isErr(core.ErrBlobFeeCapTooLow),
	// Same client refusal as TR_BLOBVERSION_INVALID: a formatted error from
	// core/state_transition.go, not a sentinel.
	"TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH": hasText("invalid hash version"),
	// Raised by the harness itself in RunNoVerify, not by core.
	"TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED": hasText("blob gas exceeds maximum"),
}

// splitExceptionNames implements step 1: split on '|', trim, drop empties, and
// keep what remains verbatim.
func splitExceptionNames(field string) []string {
	parts := strings.Split(field, "|")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		if n := strings.TrimSpace(p); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// matchExpectException implements steps 2 through 4. It is called only when a
// refusal was expected and a refusal occurred; the caller has already handled
// the two cases where one is present without the other.
//
// It returns nil when the stated names include one this build produces and that
// name matches the refusal actually raised. Otherwise it returns an error naming
// what was stated, what was raised, and which failure mode applies.
//
// The error compared here is whatever RunNoVerify returned, which is not always a
// client refusal: UnsupportedForkError arrives by the same path, for a fork this
// build has no configuration for. A fixture stating any refusal for such a fork
// therefore fails as unmapped -- a deliberate change, since before this
// comparison existed any error satisfied any stated name and such a fixture
// passed silently. The failure names the fork rather than a refusal rule, so read
// it as "this build cannot run that fork" rather than as a wrong refusal.
func matchExpectException(field string, err error) error {
	names := splitExceptionNames(field)
	if len(names) == 0 {
		// A field that is non-empty but contains no name after trimming is
		// malformed rather than absent, and saying so is more useful than
		// treating it as an unstated expectation.
		return fmt.Errorf("malformed expectException %q: states no name, got error: %v", field, err)
	}

	var known, unknown []string
	for _, n := range names {
		if m, ok := refusalVocabulary[n]; ok {
			if m(err) {
				return nil // step 3: a stated name matches the refusal raised.
			}
			known = append(known, n)
		} else {
			unknown = append(unknown, n)
		}
	}

	// Step 4. Every branch below is a failure. None is a skip.
	//
	// There are THREE outcomes here, not two, and the order matters: any unknown
	// name at all makes the result INDETERMINATE, because one of the stated
	// alternatives is a rule this build cannot recognise and might well be the
	// one that fired. Reporting that as "the client refused for the wrong reason"
	// points at the wrong remedy -- fix the client, when the fix is to extend the
	// vocabulary.
	switch {
	case len(unknown) > 0:
		verb := "no stated name is"
		if len(known) > 0 {
			verb = "not every stated name is"
		}
		return fmt.Errorf("unmapped expectException %q: %s known to this build, "+
			"so the refusal cannot be verified; got error: %v (add the name to refusalVocabulary "+
			"in tests/exception_matcher.go, or correct the fixture)",
			strings.Join(unknown, "|"), verb, err)
	default:
		// Every stated name is known and none matched, so this is a real
		// disagreement about which rule refused.
		return fmt.Errorf("expected refusal %q, got: %v", strings.Join(known, "|"), err)
	}
}

// knownRefusalNames returns the vocabulary's names, sorted. Used by tests to
// assert the table's contents rather than restating them.
func knownRefusalNames() []string {
	names := make([]string, 0, len(refusalVocabulary))
	for n := range refusalVocabulary {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
