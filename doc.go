// Package go_mhda implements the MultiChain Hierarchical Deterministic Address
// (MHDA) format - a URN-based descriptor for blockchain HD addresses across
// EVM, Bitcoin, Cosmos, Solana, XRP, Stellar, NEAR, Aptos, Sui, Cardano,
// Algorand, TON and others. The format is RFC 8141 compatible.
//
// # URN structure
//
//	urn:mhda:nt:<network>:ct:<slip44>:ci:<chain_id>:dt:<derivation>:dp:<path>:aa:<algorithm>:af:<format>:ap:<prefix>:as:<suffix>
//
// Only nt, ct and ci are required. Optional fields aa/af/ap/as are emitted on
// String() only when explicitly set, preserving short-form round-trip. The
// chain-domain prefix nt:X:ct:Y:ci:Z is itself a valid ChainKey returned by
// Chain.String() and consumed by ChainFromKey / ChainFromNSS.
//
// # Parsing
//
// ParseURN performs structural parsing only - any (network, algorithm, format)
// triple that passes the per-field validity check is accepted. ParseURNStrict
// additionally consults the network compatibility matrix in compatibility.go
// and rejects nonsensical combinations (for example an EVM URN claiming
// ed25519, or a Solana URN claiming hex format).
//
// # Errors
//
// All parser errors wrap the sentinel values exported in errors.go and
// compatibility.go (ErrInvalidURN, ErrInvalidNetworkType, ErrIncompatible,
// etc). Use errors.Is to discriminate failure modes; do not depend on the
// surrounding error message text.
//
// # Concurrency
//
// Address, Chain and DerivationPath are mutable value types: their Set*
// methods modify the receiver in place and are NOT safe for concurrent use.
// Concurrent calls to Set* on the same instance race; the caller must
// synchronise externally if mutation from multiple goroutines is required.
//
// Read-only operations (String, NSS, Hash, Hash256, MarshalText, Validate,
// the parser entry points ParseURN/ParseURNStrict/ParseNSS, and all getter
// methods) are safe to invoke concurrently provided the receiver is not being
// mutated at the same time.
//
// The package-level lookup tables (networkCompatibility, derivationIndex,
// ntIndex, indexAlgorithms, indexFormats, knownComponents) are populated at
// init time and thereafter read-only; concurrent reads are safe.
//
// # Hashing
//
// Hash and NSSHash use SHA-1 and are retained for backward compatibility with
// pre-existing identifiers. New code should use Hash256 / NSSHash256, which
// produce SHA-256 digests.
package go_mhda
