package go_mhda

import "errors"

// Sentinel errors. Callers may use errors.Is to discriminate failure modes.
var (
	ErrInvalidURN            = errors.New("mhda: not a valid mhda urn")
	ErrInvalidNSS            = errors.New("mhda: cannot parse nss")
	ErrMissingNetworkType    = errors.New(`mhda: "nt" is required`)
	ErrInvalidNetworkType    = errors.New(`mhda: invalid "nt"`)
	ErrInvalidCoinType       = errors.New(`mhda: invalid "ct"`)
	ErrMissingChainID        = errors.New(`mhda: "ci" is required`)
	ErrInvalidDerivationType = errors.New(`mhda: invalid "dt"`)
	ErrInvalidDerivationPath = errors.New(`mhda: invalid "dp"`)
	ErrInvalidAlgorithm      = errors.New(`mhda: invalid "aa"`)
	ErrInvalidFormat         = errors.New(`mhda: invalid "af"`)
	ErrUninitializedAddress  = errors.New("mhda: address is not initialized")

	// ErrInvalidValue is returned by the free-form component setters
	// (ap/as/wt/wi) when a value contains characters that would corrupt the
	// serialised NSS: the ':' separator, the RFC 8141 r/q/f delimiters
	// ('?', '#') or whitespace.
	ErrInvalidValue = errors.New("mhda: invalid component value")

	// ErrCoinTypeInChainKey is returned by ChainFromKey when the input carries
	// a "ct" component. Chain keys are the bare identity "nt:<network>:ci:<id>";
	// a key with "ct" is the pre-1.1 format and must be regenerated.
	ErrCoinTypeInChainKey = errors.New(`mhda: "ct" is not allowed in a chain key`)
	// ErrInvalidChainKey is returned by ChainFromKey when the input carries any
	// known component beyond the chain identity ("nt", "ci").
	ErrInvalidChainKey = errors.New("mhda: not a valid chain key")
)
