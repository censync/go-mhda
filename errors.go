package go_mhda

import "errors"

// Sentinel errors. Callers may use errors.Is to discriminate failure modes.
var (
	ErrInvalidURN            = errors.New("mhda: not a valid mhda urn")
	ErrInvalidNSS            = errors.New("mhda: cannot parse nss")
	ErrMissingNetworkType    = errors.New(`mhda: "nt" is required`)
	ErrInvalidNetworkType    = errors.New(`mhda: invalid "nt"`)
	ErrMissingCoinType       = errors.New(`mhda: "ct" is required`)
	ErrInvalidCoinType       = errors.New(`mhda: invalid "ct"`)
	ErrMissingChainID        = errors.New(`mhda: "ci" is required`)
	ErrInvalidDerivationType = errors.New(`mhda: invalid "dt"`)
	ErrInvalidDerivationPath = errors.New(`mhda: invalid "dp"`)
	ErrInvalidAlgorithm      = errors.New(`mhda: invalid "aa"`)
	ErrInvalidFormat         = errors.New(`mhda: invalid "af"`)
	ErrUninitializedAddress  = errors.New("mhda: address is not initialized")
)
