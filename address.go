package go_mhda

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

	HEX    = Format(`hex`)
	P2PKH  = Format(`p2pkh`)
	P2S4   = Format(`p2s4`)
	P2WPKH = Format(`p2wpkh`)
	Bech32 = Format(`bech32`)
	Base58 = Format(`base58`)

	SS58 = Format(`ss58`)
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
		HEX:    true,
		P2PKH:  true,
		P2S4:   true,
		P2WPKH: true,
		Bech32: true,
		Base58: true,
		SS58:   true,
	}
)
