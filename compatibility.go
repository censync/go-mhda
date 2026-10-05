package go_mhda

import (
	"errors"
	"fmt"
)

// ErrIncompatible is returned by Validate when the combination of network
// type, algorithm and format is not supported by this library. Wrap-only
// sentinel: callers can errors.Is(err, ErrIncompatible).
var ErrIncompatible = errors.New("mhda: incompatible network/algorithm/format")

// networkCompat describes the algorithms, address formats and derivation
// schemes that are considered valid for a given network type, plus the
// defaults to apply when the URN omits the optional aa/af components.
//
// "format zero-value" (empty Format) is allowed for networks that have many
// reasonable choices (e.g. Bitcoin: legacy, segwit, taproot) - in that case
// the URN must spell the format out explicitly to use Strict mode.
//
// ROOT is implicitly accepted for every registered network; it represents
// the non-HD form (no derivation path) and is always structurally valid.
type networkCompat struct {
	algorithms       map[Algorithm]struct{}
	formats          map[Format]struct{}
	derivations      map[DerivationType]struct{}
	defaultAlgorithm Algorithm
	defaultFormat    Format
}

var networkCompatibility = map[NetworkType]networkCompat{
	Bitcoin: {
		algorithms:       set(Secp256k1),
		formats:          set(P2PKH, P2SH, P2WPKH, P2WSH, P2TR, Bech32, Bech32m),
		derivations:      set(BIP32, BIP44, BIP49, BIP84, BIP86),
		defaultAlgorithm: Secp256k1,
		// no default format: BTC has multiple legitimate scripts
	},
	EthereumVM: {
		algorithms:       set(Secp256k1),
		formats:          set(HEX),
		derivations:      set(BIP32, BIP44),
		defaultAlgorithm: Secp256k1,
		defaultFormat:    HEX,
	},
	AvalancheVM: {
		algorithms:       set(Secp256k1),
		formats:          set(HEX, Bech32),
		derivations:      set(BIP44),
		defaultAlgorithm: Secp256k1,
		// no default format: C-Chain uses hex, X/P-Chain use bech32
	},
	TronVM: {
		algorithms:       set(Secp256k1),
		formats:          set(Base58),
		derivations:      set(BIP44),
		defaultAlgorithm: Secp256k1,
		defaultFormat:    Base58,
	},
	Cosmos: {
		algorithms:       set(Secp256k1, Ed25519),
		formats:          set(Bech32),
		derivations:      set(BIP44, CIP11),
		defaultAlgorithm: Secp256k1,
		defaultFormat:    Bech32,
	},
	Solana: {
		algorithms:       set(Ed25519),
		formats:          set(Base58),
		derivations:      set(SLIP10),
		defaultAlgorithm: Ed25519,
		defaultFormat:    Base58,
	},
	// XRP Ledger: secp256k1 was the original signing curve (and is still the
	// historical default for BIP-44 derivations); ed25519 was added later.
	// Addresses are base58 over XRPL's custom alphabet (start with 'r').
	XRPLedger: {
		algorithms:       set(Secp256k1, Ed25519),
		formats:          set(Base58),
		derivations:      set(BIP44),
		defaultAlgorithm: Secp256k1,
		defaultFormat:    Base58,
	},
	// Stellar (SEP-0005, SEP-0023). Single-curve protocol (ed25519). StrKey
	// is base32 of version_byte || payload || CRC16-XMODEM. Canonical HD
	// path is m/44'/148'/account' (3 levels, all hardened) under SLIP-10.
	Stellar: {
		algorithms:       set(Ed25519),
		formats:          set(StrKey),
		derivations:      set(SLIP10),
		defaultAlgorithm: Ed25519,
		defaultFormat:    StrKey,
	},
	// NEAR Protocol. Two implicit-account flavours: ed25519-implicit (raw
	// 64-char hex of pubkey) and ETH-implicit (0x + 40 hex from secp256k1 +
	// keccak256). Named accounts (alice.near) are not HD-derived. The de
	// facto HD convention is SLIP-10 ed25519 at m/44'/397'/0'.
	NEARProtocol: {
		algorithms:       set(Ed25519, Secp256k1),
		formats:          set(HEX),
		derivations:      set(SLIP10, BIP44),
		defaultAlgorithm: Ed25519,
		defaultFormat:    HEX,
	},
	// Aptos. The Aptos TS SDK enforces two distinct path shapes:
	//   ed25519:   m/44'/637'/account'/change'/index'  (all 5 hardened)
	//   secp256k1: m/44'/637'/account'/change/index    (last two soft, BIP-44)
	// Address = sha3-256(pubkey || scheme_id) truncated to 32 bytes, rendered
	// as 0x-prefixed hex (AIP-40 zero-padded canonical form is 64 hex chars).
	Aptos: {
		algorithms:       set(Ed25519, Secp256k1),
		formats:          set(HEX),
		derivations:      set(SLIP10, BIP44),
		defaultAlgorithm: Ed25519,
		defaultFormat:    HEX,
	},
	// Sui. Three signature schemes, distinguished by the purpose field of
	// the derivation path (sui-keys/src/key_derive.rs):
	//   ed25519:    m/44'/784'/account'/change'/index'  (all 5 hardened, SLIP-10)
	//   secp256k1:  m/54'/784'/account'/change/index    (BIP-32, purpose=54')
	//   secp256r1:  m/74'/784'/account'/change/index    (BIP-32, purpose=74')
	// Address = Blake2b-256(flag || pubkey), 0x + 64 hex chars. Flag encodes
	// the scheme: 0x00 ed25519, 0x01 secp256k1, 0x02 secp256r1.
	Sui: {
		algorithms:       set(Ed25519, Secp256k1, Secp256r1),
		formats:          set(HEX),
		derivations:      set(SLIP10, BIP54, BIP74),
		defaultAlgorithm: Ed25519,
		defaultFormat:    HEX,
	},
	// Cardano (ADA). Single-curve protocol using BIP32-Ed25519 (extended
	// keys, soft derivation supported - distinct from SLIP-10 ed25519).
	// Canonical HD path is CIP-1852: m/1852'/1815'/account'/role/index.
	// Shelley addresses are bech32 (no BIP-173 length cap, per CIP-19) with
	// HRPs from CIP-5: addr/addr_test for payment, stake/stake_test for
	// staking, drep/cc_cold/cc_hot for governance keys. Byron-era addresses
	// use base58 and are still accepted on-chain.
	Cardano: {
		algorithms:       set(Ed25519),
		formats:          set(Bech32, Base58),
		derivations:      set(CIP1852),
		defaultAlgorithm: Ed25519,
		defaultFormat:    Bech32,
	},
	// Algorand (ALGO). Native scheme is non-HD: a 25-word BIP-39-style
	// mnemonic directly encodes the 32-byte ed25519 seed (no derivation
	// path). Canonical URNs therefore use dt:root. Some third-party wallets
	// layer SLIP-10 ed25519 at m/44'/283'/account'/0'/0' on top - that form
	// is also accepted but is not Algorand-canonical. Address = base32(no
	// padding) of pubkey(32) || sha512_256(pubkey)[28:32].
	Algorand: {
		algorithms:       set(Ed25519),
		formats:          set(Base32),
		derivations:      set(SLIP10),
		defaultAlgorithm: Ed25519,
		defaultFormat:    Base32,
	},
	// TON (Toncoin). Native scheme is non-HD: a 24-word TON-specific
	// mnemonic (different word list/rules from BIP-39) feeds PBKDF2-HMAC-
	// SHA512 to derive a single ed25519 keypair. The Ledger app uses SLIP-10
	// at m/44'/607'/account' but that is wallet-specific.
	// Address has two interchangeable forms (TL-B MsgAddressInt):
	//   raw:           workchain ":" 64 hex chars (e.g. "0:abcd...")
	//   user-friendly: base64url of tag(1)||workchain(1)||hash(32)||CRC16(2)
	// The friendly form is the default user-facing rendering; raw is used
	// inside protocol messages and for canonical equality comparisons.
	Toncoin: {
		algorithms:       set(Ed25519),
		formats:          set(Base64URL, HEX),
		derivations:      set(SLIP10),
		defaultAlgorithm: Ed25519,
		defaultFormat:    Base64URL,
	},
}

func set[T comparable](items ...T) map[T]struct{} {
	m := make(map[T]struct{}, len(items))
	for _, it := range items {
		m[it] = struct{}{}
	}
	return m
}

func defaultAlgorithm(nt NetworkType) Algorithm {
	if c, ok := networkCompatibility[nt]; ok {
		return c.defaultAlgorithm
	}
	return ""
}

func defaultFormat(nt NetworkType) Format {
	if c, ok := networkCompatibility[nt]; ok {
		return c.defaultFormat
	}
	return ""
}

// Validate checks that the address' network type, algorithm and format form a
// known-good combination. The check uses the resolved algorithm/format (i.e.
// network-type defaults are applied for unspecified values) so a short URN is
// validated as if it had been written in long form.
func (a *Address) Validate() error {
	if a == nil || a.chain == nil {
		return ErrUninitializedAddress
	}
	if err := a.checkPathSet(); err != nil {
		return err
	}
	compat, ok := networkCompatibility[a.chain.networkType]
	if !ok {
		return fmt.Errorf("%w: unknown network type %q", ErrIncompatible, a.chain.networkType)
	}

	algo := a.Algorithm()
	if algo == "" {
		return fmt.Errorf("%w: no algorithm resolved for network %q", ErrIncompatible, a.chain.networkType)
	}
	if _, ok := compat.algorithms[algo]; !ok {
		return fmt.Errorf("%w: algorithm %q not allowed for network %q",
			ErrIncompatible, algo, a.chain.networkType)
	}

	if format := a.Format(); format != "" {
		if _, ok := compat.formats[format]; !ok {
			return fmt.Errorf("%w: format %q not allowed for network %q",
				ErrIncompatible, format, a.chain.networkType)
		}
	}

	// ROOT (no derivation path) is always permitted; it represents the non-HD
	// form. Any other derivation type must be in the per-network whitelist.
	if a.path != nil && a.path.derivationType != ROOT {
		if _, ok := compat.derivations[a.path.derivationType]; !ok {
			return fmt.Errorf("%w: derivation %q not allowed for network %q",
				ErrIncompatible, a.path.derivationType, a.chain.networkType)
		}
	}

	return nil
}
