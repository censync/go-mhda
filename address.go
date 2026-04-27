package go_mhda

import (
	"errors"
	"strings"
)

// normalize trims surrounding whitespace and lowercases the input. Used
// throughout the package to canonicalise component values (network type,
// algorithm, format, derivation type) before lookup.
func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

const (

	// Derivation algorithms

	Secp256k1  = Algorithm(`secp256k1`)
	Ed25519    = Algorithm(`ed25519`)
	Sr25519    = Algorithm(`sr25519`)   // Polkadot https://wiki.polkadot.network/docs/learn-account-advanced
	Secp256r1  = Algorithm(`secp256r1`) // SUI https://docs.sui.io/learn/cryptography/sui-wallet-specs
	Secp384r1  = Algorithm(`secp384r1`)
	Secp521r1  = Algorithm(`secp521r1`)
	Prime256v1 = Algorithm(`prime256v1`) // OpenSSL

	// Address formats

	HEX       = Format(`hex`)
	P2PKH     = Format(`p2pkh`)     // Pay to Public Key Hash (legacy BTC)
	P2SH      = Format(`p2sh`)      // Pay to Script Hash (nested SegWit BTC)
	P2WPKH    = Format(`p2wpkh`)    // Pay to Witness Public Key Hash
	P2WSH     = Format(`p2wsh`)     // Pay to Witness Script Hash
	P2TR      = Format(`p2tr`)      // Pay to Taproot (BIP-341)
	Bech32    = Format(`bech32`)    // BIP-173 (SegWit v0)
	Bech32m   = Format(`bech32m`)   // BIP-350 (SegWit v1+, used by Taproot)
	Base58    = Format(`base58`)    // Bitcoin alphabet base58check (BTC, XRP, TRX...)
	Base32    = Format(`base32`)    // RFC 4648 base32 + 4-byte SHA512/256 checksum (Algorand)
	StrKey    = Format(`strkey`)    // Stellar SEP-23: base32 of version || payload || CRC16
	Base64URL = Format(`base64url`) // RFC 4648 base64url, used by TON user-friendly addresses
	SS58      = Format(`ss58`)      // Substrate / Polkadot SS58
)

type Algorithm string

type Format string

var (
	indexAlgorithms = map[Algorithm]bool{
		Secp256k1:  true,
		Ed25519:    true,
		Sr25519:    true,
		Secp256r1:  true,
		Secp384r1:  true,
		Secp521r1:  true,
		Prime256v1: true,
	}

	indexFormats = map[Format]bool{
		HEX:       true,
		P2PKH:     true,
		P2SH:      true,
		P2WPKH:    true,
		P2WSH:     true,
		P2TR:      true,
		Bech32:    true,
		Bech32m:   true,
		Base58:    true,
		Base32:    true,
		StrKey:    true,
		Base64URL: true,
		SS58:      true,
	}
)

// IsValid reports whether the algorithm is one of the recognised constants.
func (a Algorithm) IsValid() bool { return indexAlgorithms[a] }

// String returns the algorithm as a plain string.
func (a Algorithm) String() string { return string(a) }

// AlgorithmFromString parses a string into an Algorithm. The lookup is
// case-insensitive; surrounding whitespace is stripped.
func AlgorithmFromString(src string) (Algorithm, error) {
	a := Algorithm(normalize(src))
	if !a.IsValid() {
		return "", errors.New("undefined algorithm")
	}
	return a, nil
}

// IsValid reports whether the format is one of the recognised constants.
func (f Format) IsValid() bool { return indexFormats[f] }

// String returns the format as a plain string.
func (f Format) String() string { return string(f) }

// FormatFromString parses a string into a Format. The lookup is
// case-insensitive; surrounding whitespace is stripped.
func FormatFromString(src string) (Format, error) {
	f := Format(normalize(src))
	if !f.IsValid() {
		return "", errors.New("undefined format")
	}
	return f, nil
}
