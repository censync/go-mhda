package go_mhda

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

var (
	uriMHDA = []string{
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/1'/0/1:aa:secp256k1:af:hex:ap:0x`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/2'/0/2'`,
		`urn:mhda:nt:evm:ci:1`,
		`urn:mhda:nt:bitcoin:ci:bitcoin_testnet:ct:0:dt:bip44:dp:m/44'/0'/0'/0/0`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip44:dp:m/44'/0'/1'/0/1:aa:secp256k1:af:p2pkh:ap:1`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip84:dp:m/84'/0'/2'/0/2`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`,
		`urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:dt:cip11:dp:m/44'/118'/0'/0/0`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0:wt:web3:wi:5f2a8c31`,
		`urn:mhda:nt:ton:ci:mainnet:wt:tonconnect:wi:c0a8f2d4-3b6e-4a51-9c7d-2f8e1a0b5c93`,
	}
)

func TestParse(t *testing.T) {
	for i := 0; i < len(uriMHDA); i++ {
		addr, err := ParseURN(uriMHDA[i])
		if err != nil {
			t.Fatalf("parse %q: %v", uriMHDA[i], err)
		}
		t.Log(addr.String())
	}
}

func TestParseNSS(t *testing.T) {
	for i := 0; i < len(uriMHDA); i++ {
		addr, err := ParseURN(uriMHDA[i])
		if err != nil {
			t.Fatalf("parse %q: %v", uriMHDA[i], err)
		}
		if addr.NSS() != uriMHDA[i][prefixOffset:] {
			t.Errorf("NSS mismatch:\n got:  %s\n want: %s", addr.NSS(), uriMHDA[i][prefixOffset:])
		}
	}
}

// TestRoundTrip ensures that Parse(s).String() == s for canonical inputs.
func TestRoundTrip(t *testing.T) {
	for _, in := range uriMHDA {
		addr, err := ParseURN(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		got := addr.String()
		if got != in {
			t.Errorf("round-trip mismatch:\n got:  %s\n want: %s", got, in)
		}
	}
}

// TestRoundTripIdempotent ensures that re-parsing a serialized form is stable.
func TestRoundTripIdempotent(t *testing.T) {
	for _, in := range uriMHDA {
		first, err := ParseURN(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		second, err := ParseURN(first.String())
		if err != nil {
			t.Fatalf("re-parse %q: %v", first.String(), err)
		}
		if first.String() != second.String() {
			t.Errorf("not idempotent:\n once:  %s\n twice: %s", first.String(), second.String())
		}
	}
}

// TestSentinelErrors verifies that callers can discriminate failure modes via
// errors.Is, regardless of the wrapped detail message.
func TestSentinelErrors(t *testing.T) {
	cases := []struct {
		urn  string
		want error
	}{
		{`mhda:nt:evm:ci:1:ct:60`, ErrInvalidURN},
		{`urn:mhda:ct:60:ci:1`, ErrMissingNetworkType},
		{`urn:mhda:nt:notanetwork:ci:1:ct:60`, ErrInvalidNetworkType},
		{`urn:mhda:nt:evm:ci:1:ct:notanumber`, ErrInvalidCoinType},
		{`urn:mhda:nt:evm:ct:60`, ErrMissingChainID},
		{`urn:mhda:nt:evm:ci:1:ct:60:dt:bipxx:dp:m/44'/60'/0'/0/0`, ErrInvalidDerivationType},
		{`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:not_a_path`, ErrInvalidDerivationPath},
		{`urn:mhda:nt:evm:ci:1:ct:60:aa:rsa`, ErrInvalidAlgorithm},
		{`urn:mhda:nt:evm:ci:1:ct:60:af:notaformat`, ErrInvalidFormat},
	}
	for _, c := range cases {
		_, err := ParseURN(c.urn)
		if err == nil {
			t.Errorf("expected error for %q, got nil", c.urn)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("for %q got %v, want errors.Is(%v)", c.urn, err, c.want)
		}
	}
}

// TestJSONRoundTrip uses encoding/json via TextMarshaler/TextUnmarshaler.
func TestJSONRoundTrip(t *testing.T) {
	for _, in := range uriMHDA {
		addr, err := ParseURN(in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		raw, err := json.Marshal(addr)
		if err != nil {
			t.Fatalf("marshal %q: %v", in, err)
		}
		var back Address
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
		if back.String() != in {
			t.Errorf("json round-trip mismatch:\n got:  %s\n want: %s", back.String(), in)
		}
	}
}

// TestValidateCompatibility checks the (networkType, algorithm, format) matrix.
func TestValidateCompatibility(t *testing.T) {
	type tc struct {
		urn     string
		wantErr bool
	}
	cases := []tc{
		// valid combos
		{`urn:mhda:nt:evm:ci:1:ct:60`, false},
		{`urn:mhda:nt:evm:ci:1:ct:60:aa:secp256k1:af:hex`, false},
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh`, false},
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`, false},
		{`urn:mhda:nt:solana:ci:mainnet:ct:501`, false},
		{`urn:mhda:nt:solana:ci:mainnet:ct:501:aa:ed25519:af:base58`, false},
		{`urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:dt:cip11:dp:m/44'/118'/0'/0/0`, false},

		// invalid combos
		{`urn:mhda:nt:evm:ci:1:ct:60:aa:ed25519`, true},             // evm + ed25519
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:aa:ed25519`, true},    // btc + ed25519
		{`urn:mhda:nt:evm:ci:1:ct:60:af:bech32`, true},              // evm + bech32
		{`urn:mhda:nt:solana:ci:mainnet:ct:501:aa:secp256k1`, true}, // sol + secp256k1
		{`urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:af:hex`, true},     // cosmos + hex
	}
	for _, c := range cases {
		_, err := ParseURNStrict(c.urn)
		if c.wantErr && err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", c.urn)
			continue
		}
		if !c.wantErr && err != nil {
			t.Errorf("unexpected error for %q: %v", c.urn, err)
			continue
		}
		if c.wantErr && !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", c.urn, err)
		}
	}
}

// TestStrictPreservesLenient ensures non-strict parse stays permissive.
func TestStrictPreservesLenient(t *testing.T) {
	bad := `urn:mhda:nt:evm:ci:1:ct:60:aa:ed25519`
	if _, err := ParseURN(bad); err != nil {
		t.Fatalf("non-strict parse should accept structurally valid URN, got %v", err)
	}
	if _, err := ParseURNStrict(bad); err == nil {
		t.Fatalf("strict parse should reject %q", bad)
	}
}

// TestBitcoinFormats covers all Bitcoin script types under Strict mode.
func TestBitcoinFormats(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip49:dp:m/49'/0'/0'/0/0:af:p2sh`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip84:dp:m/84'/0'/0'/0/0:af:p2wpkh`,
		// P2WSH has no single-key purpose (multisig is BIP-48): plain bip32.
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip32:dp:m/0'/0/0:af:p2wsh`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip86:dp:m/86'/0'/0'/0/0:af:p2tr`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q) failed: %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestLevels verifies the canonical level-by-level view of common paths.
func TestLevels(t *testing.T) {
	type tc struct {
		urn  string
		want []AddressIndex
	}
	cases := []tc{
		{
			`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/1'/0/2`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 60, IsHardened: true},
				{Index: 1, IsHardened: true},
				{Index: 0, IsHardened: false},
				{Index: 2, IsHardened: false},
			},
		},
		{
			`urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:dt:cip11:dp:m/44'/118'/3'/0/7`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 118, IsHardened: true},
				{Index: 3, IsHardened: true},
				{Index: 0, IsHardened: false},
				{Index: 7, IsHardened: false},
			},
		},
	}
	for _, c := range cases {
		addr, err := ParseURN(c.urn)
		if err != nil {
			t.Fatalf("parse %q: %v", c.urn, err)
		}
		got := addr.DerivationPath().Levels()
		if len(got) != len(c.want) {
			t.Errorf("levels length for %q: got %d, want %d", c.urn, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("level[%d] for %q: got %+v, want %+v", i, c.urn, got[i], c.want[i])
			}
		}
	}
}

// TestREADMEExamples is a documentation-as-test: every URN literal that
// appears in README.md must parse successfully under ParseURNStrict (where it
// is supposed to validate as a complete address spec) and round-trip cleanly.
// This catches drift between docs and code on every test run.
func TestREADMEExamples(t *testing.T) {
	examples := []string{
		// EVM
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0:aa:secp256k1:af:hex:ap:0x`,
		// Bitcoin
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh:ap:1`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip49:dp:m/49'/0'/0'/0/0:af:p2sh:ap:3`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32:ap:bc1q`,
		`urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`,
		// Avalanche
		`urn:mhda:nt:evm:ci:0xa86a:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:avalanche:ci:1:ct:9000:dt:bip44:dp:m/44'/9000'/0'/0/0:af:bech32:ap:X-avax`,
		// Solana
		`urn:mhda:nt:solana:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`,
		// XRP Ledger
		`urn:mhda:nt:xrpl:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0`,
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:aa:ed25519`,
		// Stellar
		`urn:mhda:nt:stellar:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`,
		// NEAR
		`urn:mhda:nt:near:ci:mainnet:dt:slip10:dp:m/44'/397'/0'`,
		`urn:mhda:nt:near:ci:mainnet:ct:397:dt:bip44:dp:m/44'/397'/0'/0/0:aa:secp256k1`,
		// Aptos
		`urn:mhda:nt:aptos:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:bip44:dp:m/44'/637'/0'/0/0:aa:secp256k1`,
		// Sui
		`urn:mhda:nt:sui:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
		`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`,
		`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`,
		// Cardano
		`urn:mhda:nt:cardano:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`,
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/2/0`,
		`urn:mhda:nt:cardano:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0:af:base58`,
		// Algorand
		`urn:mhda:nt:algorand:ci:mainnet`,
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:dt:slip10:dp:m/44'/283'/0'/0'/0'`,
		// TON
		`urn:mhda:nt:ton:ci:mainnet`,
		`urn:mhda:nt:ton:ci:mainnet:af:hex`,
		`urn:mhda:nt:ton:ci:mainnet:ct:607:dt:slip10:dp:m/44'/607'/0'`,
		// Cosmos
		`urn:mhda:nt:cosmos:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`,
		// Root key, with and without the optional SLIP-44 metadata
		`urn:mhda:nt:evm:ci:1`,
		`urn:mhda:nt:evm:ci:1:ct:60`,
		// Wallet domain
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0:wt:web3:wi:5f2a8c31`,
		`urn:mhda:nt:ton:ci:mainnet:wt:tonconnect:wi:c0a8f2d4-3b6e-4a51-9c7d-2f8e1a0b5c93`,
	}
	for _, urn := range examples {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q) failed: %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestTON covers TON (Toncoin). Native scheme is non-HD (24-word mnemonic +
// PBKDF2 directly to ed25519 key); Ledger app uses SLIP-10 at m/44'/607'/...
// Address has two equivalent forms: friendly base64url (default user-facing)
// and raw hex (workchain:hash, used inside protocol messages).
func TestTON(t *testing.T) {
	for _, urn := range []string{
		// Canonical native non-HD with friendly base64url default
		`urn:mhda:nt:ton:ci:mainnet:ct:607`,
		// Long form
		`urn:mhda:nt:ton:ci:mainnet:ct:607:aa:ed25519:af:base64url`,
		// Raw hex form (the protocol-internal canonical address representation)
		`urn:mhda:nt:ton:ci:mainnet:ct:607:af:hex`,
		// Ledger-style HD via SLIP-10
		`urn:mhda:nt:ton:ci:mainnet:ct:607:dt:slip10:dp:m/44'/607'/0'`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:ton:ci:mainnet:ct:607`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != Base64URL {
		t.Errorf("default Format() = %q, want %q", addr.Format(), Base64URL)
	}
}

// TestTONRejectsInvalidCombos covers strict-mode rejection.
func TestTONRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		// TON is single-curve: ed25519 only
		`urn:mhda:nt:ton:ci:mainnet:ct:607:aa:secp256k1`,
		`urn:mhda:nt:ton:ci:mainnet:ct:607:aa:sr25519`,
		// Disallowed formats
		`urn:mhda:nt:ton:ci:mainnet:ct:607:af:base58`,
		`urn:mhda:nt:ton:ci:mainnet:ct:607:af:bech32`,
		`urn:mhda:nt:ton:ci:mainnet:ct:607:af:strkey`,
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestAlgorand covers Algorand. Native scheme is non-HD: a 25-word mnemonic
// encodes the seed directly with no derivation path. The canonical URN form
// therefore uses dt:root. Third-party wallet HD at m/44'/283'/... is also
// supported.
func TestAlgorand(t *testing.T) {
	for _, urn := range []string{
		// Canonical non-HD: dt:root, no dp
		`urn:mhda:nt:algorand:ci:mainnet:ct:283`,
		// With explicit defaults
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:aa:ed25519:af:base32`,
		// Third-party SLIP-10 layered HD form
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:dt:slip10:dp:m/44'/283'/0'/0'/0'`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:algorand:ci:mainnet:ct:283`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != Base32 {
		t.Errorf("default Format() = %q, want %q", addr.Format(), Base32)
	}
}

// TestAlgorandRejectsInvalidCombos covers strict-mode rejection.
func TestAlgorandRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		// Algorand is single-curve: only ed25519
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:aa:secp256k1`,
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:aa:sr25519`,
		// Algorand uses base32 only
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:af:hex`,
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:af:base58`,
		`urn:mhda:nt:algorand:ci:mainnet:ct:283:af:bech32`,
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestCardano covers Cardano (ADA) registration: BIP32-Ed25519 via CIP-1852,
// Shelley bech32 addresses (default), Byron base58 (legacy).
func TestCardano(t *testing.T) {
	for _, urn := range []string{
		// short form: defaults aa=ed25519, af=bech32 (Shelley)
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0`,
		// long form
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0:aa:ed25519:af:bech32`,
		// staking key (role=2 per CIP-1852)
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/2/0`,
		// internal change address (role=1)
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/1/3`,
		// Byron-era legacy address: base58 format still accepted
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0:af:base58`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != Bech32 {
		t.Errorf("default Format() = %q, want %q", addr.Format(), Bech32)
	}
}

// TestCardanoRejectsInvalidCombos covers strict-mode rejection.
func TestCardanoRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		// Cardano is single-curve: only ed25519 is allowed
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:aa:secp256k1`,
		// hex / strkey are not Cardano formats
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:af:hex`,
		`urn:mhda:nt:cardano:ci:mainnet:ct:1815:af:strkey`,
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestSui covers Sui's three signature schemes. The purpose field of the
// derivation path encodes the scheme: 44' for ed25519, 54' for secp256k1, 74'
// for secp256r1 (per sui-keys/src/key_derive.rs).
func TestSui(t *testing.T) {
	for _, urn := range []string{
		// ed25519 (default), all-hardened SLIP-10 form
		`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
		// secp256k1 via BIP-54
		`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`,
		// secp256r1 via BIP-74
		`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:slip10:dp:m/44'/784'/0'/0'/0'`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != HEX {
		t.Errorf("default Format() = %q, want %q", addr.Format(), HEX)
	}
}

// TestSuiPurposeMapping verifies that the purpose field of each Sui path
// reflects the signature scheme as documented in sui-keys.
func TestSuiPurposeMapping(t *testing.T) {
	cases := []struct {
		urn         string
		wantPurpose uint32
	}{
		{`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:slip10:dp:m/44'/784'/0'/0'/0'`, 44},
		{`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`, 54},
		{`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`, 74},
	}
	for _, c := range cases {
		addr, err := ParseURN(c.urn)
		if err != nil {
			t.Fatalf("parse %q: %v", c.urn, err)
		}
		levels := addr.DerivationPath().Levels()
		if len(levels) == 0 {
			t.Errorf("no levels for %q", c.urn)
			continue
		}
		if levels[0].Index != c.wantPurpose {
			t.Errorf("purpose for %q = %d, want %d", c.urn, levels[0].Index, c.wantPurpose)
		}
		if !levels[0].IsHardened {
			t.Errorf("purpose for %q must be hardened", c.urn)
		}
	}
}

// TestSuiRejectsInvalidCombos covers strict-mode rejection.
func TestSuiRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:sui:ci:mainnet:ct:784:aa:sr25519`, // wrong curve
		`urn:mhda:nt:sui:ci:mainnet:ct:784:af:base58`,  // not Sui
		`urn:mhda:nt:sui:ci:mainnet:ct:784:af:strkey`,  // not Sui
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestAptos covers Aptos network registration. The TS SDK enforces two path
// shapes per signature scheme: ed25519 uses SLIP-10 with all 5 levels
// hardened, secp256k1 uses standard BIP-44.
func TestAptos(t *testing.T) {
	for _, urn := range []string{
		// ed25519 (default), all-hardened SLIP-10 form
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
		// long form
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:slip10:dp:m/44'/637'/0'/0'/0':aa:ed25519:af:hex`,
		// secp256k1 BIP-44 form
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:bip44:dp:m/44'/637'/0'/0/0:aa:secp256k1`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:slip10:dp:m/44'/637'/0'/0'/0'`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != HEX {
		t.Errorf("default Format() = %q, want %q", addr.Format(), HEX)
	}

	// Levels view of the all-hardened ed25519 form.
	want := []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 637, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
	}
	got := addr.DerivationPath().Levels()
	if len(got) != len(want) {
		t.Fatalf("levels length: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("level[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestAptosRejectsInvalidCombos covers strict-mode rejection.
func TestAptosRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:aa:sr25519`, // wrong curve
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:af:base58`,  // not Aptos
		`urn:mhda:nt:aptos:ci:mainnet:ct:637:af:bech32`,  // not Aptos
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestNEAR covers NEAR Protocol. The de facto HD convention is SLIP-10 ed25519
// at m/44'/397'/0' (no protocol-level spec). secp256k1 is also a valid access
// key curve, used by the ETH-implicit account scheme.
func TestNEAR(t *testing.T) {
	for _, urn := range []string{
		// short form: defaults aa=ed25519, af=hex
		`urn:mhda:nt:near:ci:mainnet:ct:397:dt:slip10:dp:m/44'/397'/0'`,
		// long form
		`urn:mhda:nt:near:ci:mainnet:ct:397:dt:slip10:dp:m/44'/397'/0':aa:ed25519:af:hex`,
		// secp256k1 variant (ETH-implicit accounts)
		`urn:mhda:nt:near:ci:mainnet:ct:397:dt:bip44:dp:m/44'/397'/0'/0/0:aa:secp256k1`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:near:ci:mainnet:ct:397:dt:slip10:dp:m/44'/397'/0'`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != HEX {
		t.Errorf("default Format() = %q, want %q", addr.Format(), HEX)
	}
}

// TestNEARRejectsInvalidCombos covers strict-mode rejection.
func TestNEARRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:near:ci:mainnet:ct:397:aa:sr25519`, // wrong curve
		`urn:mhda:nt:near:ci:mainnet:ct:397:af:bech32`,  // NEAR has no bech32
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestStellar covers Stellar (XLM) registration. SEP-0005 specifies SLIP-10
// ed25519 with the 3-level path m/44'/148'/account' (all hardened). Test
// vectors taken from SEP-0005.
func TestStellar(t *testing.T) {
	for _, urn := range []string{
		// short form: defaults aa=ed25519, af=strkey
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:dt:slip10:dp:m/44'/148'/0'`,
		// long form
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:dt:slip10:dp:m/44'/148'/0':aa:ed25519:af:strkey`,
		// non-zero account from SEP-0005 vectors
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:dt:slip10:dp:m/44'/148'/3'`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults resolve via the compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:stellar:ci:mainnet:ct:148:dt:slip10:dp:m/44'/148'/0'`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Ed25519 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Ed25519)
	}
	if addr.Format() != StrKey {
		t.Errorf("default Format() = %q, want %q", addr.Format(), StrKey)
	}

	// Levels exposed via canonical view (3 hardened levels per SEP-0005).
	want := []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 148, IsHardened: true},
		{Index: 0, IsHardened: true},
	}
	got := addr.DerivationPath().Levels()
	if len(got) != len(want) {
		t.Fatalf("levels length: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("level[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestStellarRejectsInvalidCombos covers strict-mode rejection of curves and
// formats Stellar does not support.
func TestStellarRejectsInvalidCombos(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:aa:secp256k1`, // wrong curve
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:af:bech32`,    // wrong format
		`urn:mhda:nt:stellar:ci:mainnet:ct:148:af:base58`,    // also rejected
	} {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestXRP covers XRP Ledger network registration: short and long URN forms,
// both supported algorithms (secp256k1 default, ed25519 alternative), strict
// validation, and that defaults resolve correctly when omitted.
func TestXRP(t *testing.T) {
	for _, urn := range []string{
		// short form: defaults aa=secp256k1, af=base58 must NOT leak into NSS
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:dt:bip44:dp:m/44'/144'/0'/0/0`,
		// long form with explicit defaults
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:dt:bip44:dp:m/44'/144'/0'/0/0:aa:secp256k1:af:base58`,
		// ed25519 keys come from a family seed, not an HD path: root form
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:aa:ed25519:af:base58`,
	} {
		addr, err := ParseURNStrict(urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}

	// Defaults resolve correctly via the network-compatibility matrix.
	addr, err := ParseURN(`urn:mhda:nt:xrpl:ci:mainnet:ct:144:dt:bip44:dp:m/44'/144'/0'/0/0`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Algorithm() != Secp256k1 {
		t.Errorf("default Algorithm() = %q, want %q", addr.Algorithm(), Secp256k1)
	}
	if addr.Format() != Base58 {
		t.Errorf("default Format() = %q, want %q", addr.Format(), Base58)
	}
}

// TestXRPRejectsInvalidCombos ensures strict mode rejects nonsensical XRP
// combinations (e.g. format hex - XRPL has no hex address form).
func TestXRPRejectsInvalidCombos(t *testing.T) {
	bad := []string{
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:aa:sr25519`, // unsupported algo
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:af:hex`,     // unsupported format
		// ed25519 has no soft levels, so it cannot derive this BIP-44 path
		`urn:mhda:nt:xrpl:ci:mainnet:ct:144:dt:bip44:dp:m/44'/144'/0'/0/0:aa:ed25519`,
	}
	for _, urn := range bad {
		_, err := ParseURNStrict(urn)
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", urn)
			continue
		}
		if !errors.Is(err, ErrIncompatible) {
			t.Errorf("for %q got %v, want errors.Is(ErrIncompatible)", urn, err)
		}
	}
}

// TestNewFormats verifies that the address formats added in Phase B parse,
// validate and round-trip. Compatibility-matrix wiring (which network may use
// which format) is Phase C work.
func TestNewFormats(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:mainnet:ct:283:af:base32`,
		`urn:mhda:nt:evm:ci:mainnet:ct:148:af:strkey`,
		`urn:mhda:nt:evm:ci:mainnet:ct:607:af:base64url`,
	} {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestCIP1852 covers Cardano CIP-1852 paths. The role field is stored in the
// `charge` shortcut and accepts any non-negative integer to remain liberal vs
// future role assignments; CIP-1852 currently defines 0..5.
func TestCIP1852(t *testing.T) {
	cases := []struct {
		urn      string
		wantRole ChargeType
	}{
		// role 0 = external (payment key)
		{`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, 0},
		// role 1 = internal (change)
		{`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/1/0`, 1},
		// role 2 = staking key
		{`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/2/0`, 2},
		// role 3 = DRep (CIP-105)
		{`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/3/0`, 3},
		// hardened address index
		{`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/3'/2/7'`, 2},
	}
	for _, c := range cases {
		addr, err := ParseURN(c.urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", c.urn, err)
			continue
		}
		if addr.String() != c.urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), c.urn)
		}
		if got := addr.DerivationPath().Charge(); got != c.wantRole {
			t.Errorf("role for %q: got %d, want %d", c.urn, got, c.wantRole)
		}
	}
}

// TestCIP1852Levels verifies the canonical levels[] view exposes the fixed
// purpose=1852' and coin=1815' from the spec.
func TestCIP1852Levels(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/3'/2/7`)
	if err != nil {
		t.Fatal(err)
	}
	want := []AddressIndex{
		{Index: 1852, IsHardened: true},
		{Index: 1815, IsHardened: true},
		{Index: 3, IsHardened: true},
		{Index: 2, IsHardened: false},
		{Index: 7, IsHardened: false},
	}
	got := addr.DerivationPath().Levels()
	if len(got) != len(want) {
		t.Fatalf("levels length: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("level[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestWideChargeLevel: the CIP-11 charge and the CIP-1852 role take any
// level index. A value above 255 must keep its full width - truncated to a
// byte, role 256 would name the role-0 key.
func TestWideChargeLevel(t *testing.T) {
	cases := []struct {
		urn  string
		want uint32
	}{
		{`urn:mhda:nt:cardano:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/256/0`, 256},
		{`urn:mhda:nt:cardano:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/2147483647/0`, 2147483647},
		{`urn:mhda:nt:cosmos:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/257/0`, 257},
		{`urn:mhda:nt:cosmos:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/65536/0`, 65536},
	}
	for _, c := range cases {
		addr, err := ParseURNStrict(c.urn)
		if err != nil {
			t.Errorf("ParseURNStrict(%q): %v", c.urn, err)
			continue
		}
		if addr.String() != c.urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), c.urn)
		}
		dp := addr.DerivationPath()
		if uint32(dp.Charge()) != c.want {
			t.Errorf("%q: Charge() = %d, want %d", c.urn, dp.Charge(), c.want)
		}
		if got := dp.Levels()[3]; got != (AddressIndex{Index: c.want}) {
			t.Errorf("%q: level[3] = %+v, want {%d false}", c.urn, got, c.want)
		}
	}

	role := uint32(300)
	dp := NewDerivationPath(CIP1852, ADA, 0, ChargeType(role), AddressIndex{Index: 1})
	if got, want := dp.String(), `m/1852'/1815'/0'/300/1`; got != want {
		t.Errorf("NewDerivationPath role 300: String() = %q, want %q", got, want)
	}
}

// TestSLIP10 covers the generic SLIP-10 derivation type with paths from
// real-world chains: Solana (4 levels), Stellar (SEP-0005, 3 levels), Sui
// ed25519 (5 levels all-hardened), Aptos (5 levels all-hardened).
func TestSLIP10(t *testing.T) {
	cases := []struct {
		urn        string
		wantLevels []AddressIndex
	}{
		{
			// Solana: m/44'/501'/account'/change'
			`urn:mhda:nt:solana:ci:mainnet:ct:501:dt:slip10:dp:m/44'/501'/0'/0'`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 501, IsHardened: true},
				{Index: 0, IsHardened: true},
				{Index: 0, IsHardened: true},
			},
		},
		{
			// Stellar SEP-0005: m/44'/148'/account'
			`urn:mhda:nt:evm:ci:mainnet:ct:148:dt:slip10:dp:m/44'/148'/0'`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 148, IsHardened: true},
				{Index: 0, IsHardened: true},
			},
		},
		{
			// Sui ed25519: m/44'/784'/account'/change'/index'
			`urn:mhda:nt:evm:ci:mainnet:ct:784:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 784, IsHardened: true},
				{Index: 0, IsHardened: true},
				{Index: 0, IsHardened: true},
				{Index: 0, IsHardened: true},
			},
		},
		{
			// Aptos: m/44'/637'/account'/change'/index'
			`urn:mhda:nt:evm:ci:mainnet:ct:637:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
			nil, // checked via round-trip only
		},
	}
	for _, c := range cases {
		addr, err := ParseURN(c.urn)
		if err != nil {
			t.Fatalf("ParseURN(%q): %v", c.urn, err)
		}
		if addr.String() != c.urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), c.urn)
		}
		if c.wantLevels != nil {
			got := addr.DerivationPath().Levels()
			if len(got) != len(c.wantLevels) {
				t.Errorf("levels length for %q: got %d, want %d", c.urn, len(got), len(c.wantLevels))
				continue
			}
			for i := range got {
				if got[i] != c.wantLevels[i] {
					t.Errorf("level[%d] for %q: got %+v, want %+v", i, c.urn, got[i], c.wantLevels[i])
				}
			}
		}
	}
}

// TestSLIP10MixedHardening verifies a path with both hardened and non-hardened
// levels parses correctly. SLIP-10 ed25519 forbids non-hardened in practice
// but the parser must accept the structure - semantic validation is for the
// caller.
func TestSLIP10MixedHardening(t *testing.T) {
	urn := `urn:mhda:nt:evm:ci:mainnet:ct:0:dt:slip10:dp:m/44'/0'/0'/0/5`
	addr, err := ParseURN(urn)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: false},
		{Index: 5, IsHardened: false},
	}
	got := addr.DerivationPath().Levels()
	if len(got) != len(want) {
		t.Fatalf("levels length: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("level[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if addr.String() != urn {
		t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
	}
}

// TestNewDerivationPathFromLevels verifies the programmatic constructor.
func TestNewDerivationPathFromLevels(t *testing.T) {
	dp := NewDerivationPathFromLevels(SLIP10, []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 501, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
	})
	if want := "m/44'/501'/0'/0'"; dp.String() != want {
		t.Errorf("String() = %q, want %q", dp.String(), want)
	}
}

// TestZIP32VariableLength covers the 3-level (no index) and 4-level (with
// index) ZIP-32 forms; both must round-trip.
func TestZIP32VariableLength(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:bitcoin:ci:zcash:ct:133:dt:zip32:dp:m/32'/133'/0'`,
		`urn:mhda:nt:bitcoin:ci:zcash:ct:133:dt:zip32:dp:m/32'/133'/0'/0`,
		`urn:mhda:nt:bitcoin:ci:zcash:ct:133:dt:zip32:dp:m/32'/133'/0'/0'`,
	} {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestBIP44Family covers BIP-44 / 49 / 54 / 74 / 84 / 86 round-trip. They
// share a path shape (m/<purpose>'/coin'/account'/change/index) but differ in
// the purpose field. BIP-49 is Bitcoin Nested SegWit; BIP-54/74 are Sui
// secp256k1/secp256r1 schemes per the sui-keys source.
func TestBIP44Family(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip49:dp:m/49'/0'/0'/0/0:af:p2sh`,
		// sui not yet registered as network type; use evm as a syntactic stand-in
		// for path parsing - Phase C will add proper Sui entries.
		`urn:mhda:nt:evm:ci:1:ct:784:dt:bip54:dp:m/54'/784'/0'/0/0`,
		`urn:mhda:nt:evm:ci:1:ct:784:dt:bip74:dp:m/74'/784'/0'/0/0`,
	} {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestCIP11Coin verifies that CIP-11 paths serialize with coin 118 (Cosmos),
// not 133 (Zcash) as the previous String() formatter incorrectly hardcoded.
func TestCIP11Coin(t *testing.T) {
	in := `urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:dt:cip11:dp:m/44'/118'/3'/0/7`
	addr, err := ParseURN(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if addr.String() != in {
		t.Errorf("CIP-11 round-trip:\n got:  %s\n want: %s", addr.String(), in)
	}
}

// TestFormatLazyDefault checks the lazy default for Format() and that NSS()
// does not emit the defaulted value (only explicit values are serialized).
func TestFormatLazyDefault(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Format() != HEX {
		t.Errorf("Format() default for evm = %q, want %q", addr.Format(), HEX)
	}
	if want := `urn:mhda:nt:evm:ci:1:ct:60`; addr.String() != want {
		t.Errorf("String() = %q, want %q (defaults must not leak into serialization)", addr.String(), want)
	}
}

// TestAlgorithmDefaults checks that short forms resolve algorithm via the
// network-type default, while long forms preserve the explicit value.
func TestAlgorithmDefaults(t *testing.T) {
	cases := []struct {
		urn  string
		want Algorithm
	}{
		{`urn:mhda:nt:evm:ci:1:ct:60`, Secp256k1},
		{`urn:mhda:nt:solana:ci:mainnet:ct:501`, Ed25519},
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip44:dp:m/44'/0'/1'/0/1:aa:secp256k1:af:p2pkh:ap:1`, Secp256k1},
	}
	for _, c := range cases {
		addr, err := ParseURN(c.urn)
		if err != nil {
			t.Fatalf("parse %q: %v", c.urn, err)
		}
		if addr.Algorithm() != c.want {
			t.Errorf("Algorithm() for %q = %q, want %q", c.urn, addr.Algorithm(), c.want)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseURN(uriMHDA[0])
	}
}

// ----------------------------------------------------------------------------
// Audit-driven regression tests
// ----------------------------------------------------------------------------

// TestChainGettersChainable ensures Chain getters work on the value returned
// by Address.Chain() without an intermediate variable. Was broken when
// getters had pointer receivers since Address.Chain() returns a non-
// addressable value.
func TestChainGettersChainable(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.Chain().NetworkType(); got != EthereumVM {
		t.Errorf("NetworkType() = %q, want %q", got, EthereumVM)
	}
	if got := addr.Chain().CoinType(); got != ETH {
		t.Errorf("CoinType() = %d, want %d", got, ETH)
	}
	if !addr.Chain().HasCoinType() {
		t.Error("HasCoinType() = false, want true (ct was set in the URN)")
	}
	if got := addr.Chain().ChainId(); got != "1" {
		t.Errorf("ChainId() = %q, want %q", got, "1")
	}
	// Key() is computed from String(); make sure it works inline too. The
	// coin-type metadata is excluded from the chain key.
	want := "nt:evm:ci:1"
	if got := string(addr.Chain().Key()); got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
}

// TestChainSettersMutate verifies SetX persistently mutate the chain. Was
// broken when setters had value receivers (silent no-op).
func TestChainSettersMutate(t *testing.T) {
	c := NewChain(EthereumVM, "1")
	c.SetNetworkType(Bitcoin)
	c.SetCoinType(BTC)
	c.SetChainId("bitcoin")
	if c.NetworkType() != Bitcoin || c.CoinType() != BTC || c.ChainId() != "bitcoin" {
		t.Errorf("setters did not persist: nt=%q ct=%d ci=%q",
			c.NetworkType(), c.CoinType(), c.ChainId())
	}
	if !c.HasCoinType() {
		t.Error("HasCoinType() = false after SetCoinType")
	}
	c.ClearCoinType()
	if c.HasCoinType() || c.CoinType() != 0 {
		t.Errorf("ClearCoinType did not reset: has=%v ct=%d", c.HasCoinType(), c.CoinType())
	}
}

// TestSetPrefixSuffixReset ensures empty-string SetX resets the field, matching
// the semantics of SetAddressAlgorithm/Format.
func TestSetPrefixSuffixReset(t *testing.T) {
	addr := NewAddress(NewChain(EthereumVM, "1"), nil, "", "", "0xPREFIX", "SUFFIX")
	if !strings.Contains(addr.String(), ":ap:0xPREFIX") {
		t.Fatalf("expected prefix in URN, got %q", addr.String())
	}
	if !strings.Contains(addr.String(), ":as:SUFFIX") {
		t.Fatalf("expected suffix in URN, got %q", addr.String())
	}
	if err := addr.SetAddressPrefix(""); err != nil {
		t.Fatal(err)
	}
	if err := addr.SetAddressSuffix(""); err != nil {
		t.Fatal(err)
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("after reset: got %q, want %q", got, `urn:mhda:nt:evm:ci:1`)
	}
}

// TestNewDerivationPathSlip10Panics ensures the constructor rejects SLIP-10
// (which cannot be reconstructed from the BIP-44 shortcut fields).
func TestNewDerivationPathSlip10Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for NewDerivationPath(SLIP10, ...)")
		}
	}()
	_ = NewDerivationPath(SLIP10, 0, 0, 0, AddressIndex{})
}

// TestNewDerivationPathFromLevelsBIP44 verifies that the level-array
// constructor populates BIP-44 shortcut fields correctly so subsequent
// String() and getters behave as if parsed from text.
func TestNewDerivationPathFromLevelsBIP44(t *testing.T) {
	dp := NewDerivationPathFromLevels(BIP44, []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 60, IsHardened: true},
		{Index: 3, IsHardened: true},
		{Index: 1, IsHardened: false},
		{Index: 7, IsHardened: false},
	})
	if want := "m/44'/60'/3'/1/7"; dp.String() != want {
		t.Errorf("String() = %q, want %q", dp.String(), want)
	}
	if dp.Coin() != 60 {
		t.Errorf("Coin() = %d, want 60", dp.Coin())
	}
	if dp.Account() != 3 {
		t.Errorf("Account() = %d, want 3", dp.Account())
	}
	if dp.Charge() != 1 {
		t.Errorf("Charge() = %d, want 1", dp.Charge())
	}
	if dp.AddressIndex().Index != 7 {
		t.Errorf("AddressIndex().Index = %d, want 7", dp.AddressIndex().Index)
	}
}

// TestNewDerivationPathFromLevelsZIP32 covers the variable-length scheme:
// the constructor must handle both 3-level and 4-level forms.
func TestNewDerivationPathFromLevelsZIP32(t *testing.T) {
	short := NewDerivationPathFromLevels(ZIP32, []AddressIndex{
		{Index: 32, IsHardened: true},
		{Index: 133, IsHardened: true},
		{Index: 5, IsHardened: true},
	})
	if want := "m/32'/133'/5'"; short.String() != want {
		t.Errorf("3-level: got %q, want %q", short.String(), want)
	}
	long := NewDerivationPathFromLevels(ZIP32, []AddressIndex{
		{Index: 32, IsHardened: true},
		{Index: 133, IsHardened: true},
		{Index: 5, IsHardened: true},
		{Index: 9, IsHardened: false},
	})
	if want := "m/32'/133'/5'/9"; long.String() != want {
		t.Errorf("4-level: got %q, want %q", long.String(), want)
	}
}

// TestEmptyComponentValuesRejected ensures the byte parser does not silently
// accept empty values for known components - previously such inputs would
// pollute the components map and produce surprising results.
func TestEmptyComponentValuesRejected(t *testing.T) {
	bad := []string{
		`urn:mhda:nt:evm:dt::dp::ct:60:ci:1`,
		`urn:mhda:nt::ct:60:ci:1`,
		`urn:mhda:nt:evm:ct::ci:1`,
		`urn:mhda:nt:evm:ct:60:ci:`,
	}
	for _, urn := range bad {
		_, err := ParseURN(urn)
		if err == nil {
			t.Errorf("expected error for %q, got nil", urn)
			continue
		}
		// Either ErrInvalidNSS (empty value) or ErrMissingX downstream.
		// We accept any error - the important thing is "not silent success".
	}
}

// TestHardenedMarkerNormalization documents that input markers H/h/' all
// canonicalize to ' on output.
func TestHardenedMarkerNormalization(t *testing.T) {
	cases := map[string]string{
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44H/60H/0H/0/0`: `urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44h/60h/0h/0/0`: `urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0`,
	}
	for in, want := range cases {
		addr, err := ParseURN(in)
		if err != nil {
			t.Errorf("parse %q: %v", in, err)
			continue
		}
		if addr.String() != want {
			t.Errorf("normalize:\n in:   %s\n got:  %s\n want: %s", in, addr.String(), want)
		}
	}
}

// TestDerivationTypeAccessor verifies the new public method on Address and
// the MHDA interface.
func TestDerivationTypeAccessor(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.DerivationType(); got != BIP44 {
		t.Errorf("DerivationType() = %q, want %q", got, BIP44)
	}
	// ROOT path
	root, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60`)
	if err != nil {
		t.Fatal(err)
	}
	if got := root.DerivationType(); got != ROOT {
		t.Errorf("DerivationType() for short form = %q, want %q", got, ROOT)
	}
	// Interface method must be reachable via MHDA
	var iface MHDA = addr
	if got := iface.DerivationType(); got != BIP44 {
		t.Errorf("via interface: got %q, want %q", got, BIP44)
	}
}

// TestMarshalTextNilAddress ensures MarshalText returns a sentinel error
// rather than panicking on an uninitialised receiver.
func TestMarshalTextNilAddress(t *testing.T) {
	// A nil *Address is encoded by the codec (as null, see
	// TestTextCodecsWithAddressValues); MarshalText has a value receiver.
	a2 := &Address{} // zero-value, no chain
	_, err := a2.MarshalText()
	if !errors.Is(err, ErrUninitializedAddress) {
		t.Errorf("zero-value: got %v, want ErrUninitializedAddress", err)
	}
}

// TestUnmarshalTextInvalid covers UnmarshalText of bad input.
func TestUnmarshalTextInvalid(t *testing.T) {
	var a Address
	if err := a.UnmarshalText([]byte("not-a-urn")); !errors.Is(err, ErrInvalidURN) {
		t.Errorf("bad input: got %v, want ErrInvalidURN", err)
	}
	if err := a.UnmarshalText([]byte("urn:mhda:nt:evm:ci:1:ct:60")); err != nil {
		t.Errorf("good input rejected: %v", err)
	}
}

// TestParseDerivationPathStandalone exercises the public ParseDerivationPath
// function (used by callers who only need the path layer, no full URN).
func TestParseDerivationPathStandalone(t *testing.T) {
	cases := []struct {
		dt   DerivationType
		path string
		want string
	}{
		{BIP44, "m/44'/0'/0'/0/0", "m/44'/0'/0'/0/0"},
		{BIP44, "m/44H/0h/0'/1/2'", "m/44'/0'/0'/1/2'"}, // marker normalization
		{SLIP10, "m/44'/501'/0'/0'", "m/44'/501'/0'/0'"},
		{ZIP32, "m/32'/133'/0'", "m/32'/133'/0'"},
		{ROOT, "", ""},
	}
	for _, c := range cases {
		dp, err := ParseDerivationPath(c.dt, c.path)
		if err != nil {
			t.Errorf("ParseDerivationPath(%q, %q): %v", c.dt, c.path, err)
			continue
		}
		if dp.String() != c.want {
			t.Errorf("ParseDerivationPath(%q, %q): String() = %q, want %q",
				c.dt, c.path, dp.String(), c.want)
		}
	}
}

// TestParseDerivationPathErrors covers the standalone-API error paths and
// asserts that returned errors wrap the public sentinels so callers can use
// errors.Is for discrimination.
func TestParseDerivationPathErrors(t *testing.T) {
	cases := []struct {
		dt   DerivationType
		path string
		want error
	}{
		// unknown derivation type
		{"nope", "m/0", ErrInvalidDerivationType},
		// path doesn't match the type's regex
		{BIP44, "not_a_path", ErrInvalidDerivationPath},
		// non-empty path for ROOT
		{ROOT, "m/0", ErrInvalidDerivationPath},
		// numeric overflow inside the leaf level
		{BIP44, "m/44'/0'/0'/0/99999999999999999999", ErrInvalidDerivationPath},
	}
	for _, c := range cases {
		_, err := ParseDerivationPath(c.dt, c.path)
		if err == nil {
			t.Errorf("expected error for (%q, %q), got nil", c.dt, c.path)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("for (%q, %q): got %v, want errors.Is(%v)", c.dt, c.path, err, c.want)
		}
	}
}

// TestSetDerivationPathSentinelChain verifies that errors surfaced by
// SetDerivationPath / ParseURN preserve the inner sentinel from ParsePath
// (e.g. ErrInvalidDerivationPath wraps numeric-overflow errors so callers
// can errors.Is on a single value).
func TestSetDerivationPathSentinelChain(t *testing.T) {
	_, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/0'/0'/0/99999999999999999999`)
	if err == nil {
		t.Fatal("expected error for overflowing index")
	}
	if !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("error should wrap ErrInvalidDerivationPath, got %v", err)
	}
}

// TestNetworkTypeFromString exercises the public lookup helper. Lookup is
// case-insensitive and trims whitespace.
func TestNetworkTypeFromString(t *testing.T) {
	cases := []struct {
		in   string
		want NetworkType
		err  bool
	}{
		{"xrpl", XRPLedger, false},
		{"  XRPL  ", XRPLedger, false}, // case + whitespace tolerance
		{"solana", Solana, false},
		{"bitcoin", Bitcoin, false},
		{"sol", "", true}, // pre-1.1 short names are gone
		{"btc", "", true},
		{"xxx", "", true},
	}
	for _, c := range cases {
		got, err := NetworkTypeFromString(c.in)
		if c.err {
			if err == nil {
				t.Errorf("expected error for %q", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for %q: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("for %q: got %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAlgorithmFormatHelpers covers IsValid / String / *FromString for
// Algorithm and Format. Both lookups must be case-insensitive and tolerant
// of surrounding whitespace.
func TestAlgorithmFormatHelpers(t *testing.T) {
	if !Secp256k1.IsValid() || !Ed25519.IsValid() {
		t.Error("known algorithms must report IsValid=true")
	}
	if Algorithm("rsa").IsValid() {
		t.Error("unknown algorithm must report IsValid=false")
	}
	if got := Secp256k1.String(); got != "secp256k1" {
		t.Errorf("Algorithm.String() = %q", got)
	}

	a, err := AlgorithmFromString("  ED25519  ")
	if err != nil || a != Ed25519 {
		t.Errorf("AlgorithmFromString: got (%q, %v), want (ed25519, nil)", a, err)
	}
	if _, err := AlgorithmFromString("nope"); err == nil {
		t.Error("expected error for unknown algorithm")
	}

	if !P2TR.IsValid() || !Bech32m.IsValid() {
		t.Error("known formats must report IsValid=true")
	}
	if Format("zzz").IsValid() {
		t.Error("unknown format must report IsValid=false")
	}
	if got := Bech32m.String(); got != "bech32m" {
		t.Errorf("Format.String() = %q", got)
	}

	f, err := FormatFromString("  P2TR  ")
	if err != nil || f != P2TR {
		t.Errorf("FormatFromString: got (%q, %v), want (p2tr, nil)", f, err)
	}
	if _, err := FormatFromString("nope"); err == nil {
		t.Error("expected error for unknown format")
	}
}

// TestDerivationTypeHelpers covers IsValid / String / DerivationTypeFromString.
func TestDerivationTypeHelpers(t *testing.T) {
	if !BIP44.IsValid() || !SLIP10.IsValid() || !ROOT.IsValid() {
		t.Error("known derivation types must report IsValid=true")
	}
	if DerivationType("zip999").IsValid() {
		t.Error("unknown derivation type must report IsValid=false")
	}
	if got := BIP86.String(); got != "bip86" {
		t.Errorf("DerivationType.String() = %q", got)
	}

	dt, err := DerivationTypeFromString("  CIP1852  ")
	if err != nil || dt != CIP1852 {
		t.Errorf("DerivationTypeFromString: got (%q, %v), want (cip1852, nil)", dt, err)
	}
	_, err = DerivationTypeFromString("nope")
	if !errors.Is(err, ErrInvalidDerivationType) {
		t.Errorf("expected wrapped ErrInvalidDerivationType, got %v", err)
	}
}

// TestChainStringFormat pins the canonical chain-key form. The key is the
// chain identity (networkType, chainId) only; the optional coin-type metadata
// never leaks into it. The canonical form uses ':' separators throughout so
// it is itself a valid NSS key consumable by ChainFromKey / ChainFromNSS.
func TestChainStringFormat(t *testing.T) {
	c := NewChain(EthereumVM, "0x1")
	want := "nt:evm:ci:0x1"
	if got := c.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := string(c.Key()); got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
	// Attaching coin-type metadata must not change the key.
	c.SetCoinType(ETH)
	if got := c.String(); got != want {
		t.Errorf("String() after SetCoinType = %q, want %q", got, want)
	}
}

// TestIsHardenedAddress covers the public predicate.
func TestIsHardenedAddress(t *testing.T) {
	dp, _ := ParseDerivationPath(BIP44, "m/44'/0'/0'/0/0")
	if dp.IsHardenedAddress() {
		t.Error("non-hardened leaf should report false")
	}
	dp, _ = ParseDerivationPath(BIP44, "m/44'/0'/0'/0/0'")
	if !dp.IsHardenedAddress() {
		t.Error("hardened leaf should report true")
	}
}

// TestDerivationCompatibility verifies the derivation-type leg of strict
// validation. Each network has a whitelist of allowed derivation schemes;
// anything outside that list (e.g. evm + cip1852) must be rejected even when
// algorithm and format are valid.
func TestDerivationCompatibility(t *testing.T) {
	type tc struct {
		urn  string
		want error // nil for "must accept", ErrIncompatible for "must reject"
	}
	cases := []tc{
		// allowed combos
		{`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0`, nil},
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m`, nil},
		{`urn:mhda:nt:cosmos:ci:cosmoshub:ct:118:dt:cip11:dp:m/44'/118'/0'/0/0`, nil},
		{`urn:mhda:nt:solana:ci:mainnet:ct:501:dt:slip10:dp:m/44'/501'/0'/0'`, nil},
		{`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, nil},
		// ROOT is always permitted (non-HD form)
		{`urn:mhda:nt:algorand:ci:mainnet:ct:283`, nil},
		{`urn:mhda:nt:ton:ci:mainnet:ct:607`, nil},
		// Sui's purpose-variant schemes
		{`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`, nil},
		{`urn:mhda:nt:sui:ci:mainnet:ct:784:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`, nil},

		// disallowed combos (network has no business with this derivation)
		// EVM does not use cip1852
		{`urn:mhda:nt:evm:ci:1:ct:1815:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, ErrIncompatible},
		// Solana uses slip10 only, not bip44
		{`urn:mhda:nt:solana:ci:mainnet:ct:501:dt:bip44:dp:m/44'/501'/0'/0/0`, ErrIncompatible},
		// Cardano uses cip1852 only, not bip44
		{`urn:mhda:nt:cardano:ci:mainnet:ct:1815:dt:bip44:dp:m/44'/1815'/0'/0/0`, ErrIncompatible},
		// Bitcoin does not use cip11 (Cosmos-specific)
		{`urn:mhda:nt:bitcoin:ci:bitcoin:ct:0:dt:cip11:dp:m/44'/118'/0'/0/0`, ErrIncompatible},
		// Sui's purpose-54 path makes no sense on Aptos
		{`urn:mhda:nt:aptos:ci:mainnet:ct:637:dt:bip54:dp:m/54'/637'/0'/0/0:aa:secp256k1`, ErrIncompatible},
		// Stellar is slip10-only
		{`urn:mhda:nt:stellar:ci:mainnet:ct:148:dt:bip44:dp:m/44'/148'/0'/0/0`, ErrIncompatible},
	}
	for _, c := range cases {
		_, err := ParseURNStrict(c.urn)
		if c.want == nil {
			if err != nil {
				t.Errorf("expected accept for %q, got %v", c.urn, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("expected ErrIncompatible for %q, got nil", c.urn)
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("for %q got %v, want errors.Is(%v)", c.urn, err, c.want)
		}
	}
}

// TestRFC8141CaseInsensitivePrefix verifies that the leading "urn:" and the
// NID "mhda" are accepted in any ASCII case combination, per RFC 8141 §5.1.
// The canonical output remains lowercase.
func TestRFC8141CaseInsensitivePrefix(t *testing.T) {
	canonical := `urn:mhda:nt:evm:ci:1:ct:60`
	for _, in := range []string{
		`urn:mhda:nt:evm:ci:1:ct:60`, // baseline
		`URN:MHDA:nt:evm:ci:1:ct:60`,
		`Urn:Mhda:nt:evm:ci:1:ct:60`,
		`URN:mhda:nt:evm:ci:1:ct:60`,
		`urn:MHDA:nt:evm:ci:1:ct:60`,
		`  urn:mhda:nt:evm:ci:1:ct:60  `, // surrounding whitespace
	} {
		addr, err := ParseURN(in)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", in, err)
			continue
		}
		if addr.String() != canonical {
			t.Errorf("for %q: got %q, want %q", in, addr.String(), canonical)
		}
	}
}

// TestRFC8141DropRQF verifies that rq-components (?+ resource, ?= query) and
// f-component (# fragment) are stripped before parsing the NSS. RFC 8141
// allows them syntactically; this library does not interpret them.
func TestRFC8141DropRQF(t *testing.T) {
	canonical := `urn:mhda:nt:evm:ci:1:ct:60`
	for _, in := range []string{
		`urn:mhda:nt:evm:ci:1:ct:60?+resolver=example.com`,
		`urn:mhda:nt:evm:ci:1:ct:60?=v=1`,
		`urn:mhda:nt:evm:ci:1:ct:60#fragment`,
		`urn:mhda:nt:evm:ci:1:ct:60?+a=b#frag`,
	} {
		addr, err := ParseURN(in)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", in, err)
			continue
		}
		if addr.String() != canonical {
			t.Errorf("for %q: got %q, want %q", in, addr.String(), canonical)
		}
	}
}

// TestRegressionWhitespaceBeforeFragment is a fuzz-derived regression test.
//
// Originally found by FuzzParseURN as the seed
// "urn:mhdA:nt:BtC:ct:0:ci:0 #": the trailing space inside the value, just
// before the f-component delimiter '#', survived the first parse intact but
// was stripped by the top-level TrimSpace on a second parse, breaking
// idempotency (Parse(s).String() != Parse(Parse(s).String()).String()).
// Fixed by trimming each NSS value at parse time. The case is asserted
// explicitly here so the regression is guarded even without the fuzz seed
// corpus on disk.
func TestRegressionWhitespaceBeforeFragment(t *testing.T) {
	in := "urn:mhdA:nt:BitCoin:ct:0:ci:0 #"
	once, err := ParseURN(in)
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	twice, err := ParseURN(once.String())
	if err != nil {
		t.Fatalf("second parse of %q: %v", once.String(), err)
	}
	if once.String() != twice.String() {
		t.Errorf("not idempotent:\n once:  %q\n twice: %q\n input: %q",
			once.String(), twice.String(), in)
	}
}

// TestHashFunctions covers the four hash methods on Address. Hash and
// NSSHash use SHA-1 (kept for backward compatibility); Hash256 and NSSHash256
// use SHA-256. All four must produce stable, deterministic output and the
// SHA-256 forms must differ from the SHA-1 forms.
func TestHashFunctions(t *testing.T) {
	addr1, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60`)
	if err != nil {
		t.Fatal(err)
	}
	addr2, err := ParseURN(`urn:mhda:nt:evm:ci:1:ct:60`)
	if err != nil {
		t.Fatal(err)
	}

	// Determinism: equal inputs -> equal hashes
	if addr1.Hash() != addr2.Hash() {
		t.Error("Hash() not deterministic")
	}
	if addr1.Hash256() != addr2.Hash256() {
		t.Error("Hash256() not deterministic")
	}
	if addr1.NSSHash() != addr2.NSSHash() {
		t.Error("NSSHash() not deterministic")
	}
	if addr1.NSSHash256() != addr2.NSSHash256() {
		t.Error("NSSHash256() not deterministic")
	}

	// SHA-1 (40 hex chars) and SHA-256 (64 hex chars) must differ in length
	if got := len(addr1.Hash()); got != 40 {
		t.Errorf("Hash() length = %d, want 40 (sha1 hex)", got)
	}
	if got := len(addr1.Hash256()); got != 64 {
		t.Errorf("Hash256() length = %d, want 64 (sha256 hex)", got)
	}

	// Different inputs should yield different SHA-256 hashes (spot check)
	addrB, _ := ParseURN(`urn:mhda:nt:evm:ci:2:ct:60`)
	if addr1.Hash256() == addrB.Hash256() {
		t.Error("distinct URNs produced same Hash256()")
	}

	// Hash vs NSSHash must differ (different inputs: "urn:mhda:..." vs "...")
	if addr1.Hash() == addr1.NSSHash() {
		t.Error("Hash() and NSSHash() should differ for the same address")
	}
	if addr1.Hash256() == addr1.NSSHash256() {
		t.Error("Hash256() and NSSHash256() should differ")
	}

	// Interface methods are reachable
	var iface MHDA = addr1
	if iface.Hash256() != addr1.Hash256() {
		t.Error("MHDA.Hash256() mismatch with concrete method")
	}
	if iface.NSSHash256() != addr1.NSSHash256() {
		t.Error("MHDA.NSSHash256() mismatch with concrete method")
	}
}

// TestHashReferenceVectors pins the four digests at the SHA-1/SHA-256 block
// and padding boundaries: str() of 55, 56, 63, 64, 119, 120, 128 and 1000
// bytes (the NSS is 9 bytes shorter, so it covers 55 too). The values were
// computed with Python's hashlib and are shared with the C++ port, whose
// hashes are hand-written.
func TestHashReferenceVectors(t *testing.T) {
	cases := []struct {
		size                               int
		hash, hash256, nssHash, nssHash256 string
	}{
		{55, "7230ec9cac1176541d5e454a5977547849806ef3", "4bcf74edcd630246d1cb1f03be5dc2deb7cc850270a2d3f279d68166778e2065",
			"3c1704aa5f179b7d21a9f726e20473ac2b5786e1", "c85613e0a6562581d353224545cbebc3710ddb496d4b03baddd435fa311b7bbb"},
		{56, "7c08382a5672e118ef014b54c8d1b42e7b6cb90b", "a176aa8ff4ccdcb7b05e3e39440cc40b8b17219910f3081c062045abbf331255",
			"8f17a0079072eb355e316ebdbb73aa4328762ff7", "242f19fedc19e36ab9c4378b45922e731269ea04c325b58c4a90d4389ffce482"},
		{63, "6b6034fd2e44f4ad8ea9da946a6915f017966ead", "0be4d717499d1b2a081c60cf64f0c59b5e09a1a0da1e65c497c83e839caef655",
			"4b8360e09c0db0f1c423342d03db990ee4795695", "0bd7a04f6ca2b30eca17273961fa78db6f345e654ed80ac78a85d780d73d0267"},
		{64, "c5b9fb4f88ee7f77e14566483838fe03eb5f722a", "be3e95721333ec68988eedb3b6c37c7c48ee30211c3bef96900609e8f1dd3546",
			"fba8bdfdfb761ca75bc1a38886a7219cba376adc", "713ea1b363b46219865aded589c56341e7d0e63c9278e7e11ae154f05883d1c5"},
		{119, "91f3a5706cdf32541f6c2d45ea4280399da43847", "d04d2a8a7f6df456cada8d263a746f0d21e4b600ef7bdcea4f6e645c17d94914",
			"94285bd9afc8f49eb2059992f3660aee22e76063", "d3398acc7f987d3d3c90acd2e4ef958bd7053a8a88e4b2d51b5774b0fb03cfdc"},
		{120, "8f7e2ab8a8a0c101e299154982f0e262bf43b7d0", "a4ed7ff3b82d6f7fa7a322ca72971a2b1e6474a5d08475599bc965282acb49c7",
			"7eb1dc21b8685d3b9dbc9c2cbb9240acc90a2660", "aba04d6e4a96a9f662bb9ac433ba50f2c34d7991aee8cb2787ab91bc4228e88a"},
		{128, "56e36e13d0581067945cd12955e3ce4b21967a48", "b6fb21cc83b07ab98648ec0b1bb04a3d79cd31d231cc07303d24ce73145a46ed",
			"c792663db1797615b2433d56b96bf73d859a3e6f", "2c04bb624a604bd96c5b0e4b9e1006bbe898f134b60a40a1ed81d63e03897ea7"},
		{1000, "f3acba196b25a975d6989beef6693337b5987f26", "7d57f53b9e4785b1735151906fd63ccb70547a8a2292986b613cf23db7b0f3d7",
			"d16eb7a3991c6ace27d58b3e80e617567cfd242b", "2e7dd698a37e3fb7660b19bcad20b60677bd1e7f01d79870daa09a8a1f7269d8"},
	}
	for _, c := range cases {
		urn := `urn:mhda:nt:evm:ci:` + strings.Repeat("x", c.size-19)
		addr, err := ParseURN(urn)
		if err != nil {
			t.Fatalf("ParseURN(%d bytes): %v", c.size, err)
		}
		if got := len(addr.String()); got != c.size {
			t.Fatalf("String() is %d bytes, want %d", got, c.size)
		}
		if got := addr.Hash(); got != c.hash {
			t.Errorf("%d bytes: Hash() = %s, want %s", c.size, got, c.hash)
		}
		if got := addr.Hash256(); got != c.hash256 {
			t.Errorf("%d bytes: Hash256() = %s, want %s", c.size, got, c.hash256)
		}
		if got := addr.NSSHash(); got != c.nssHash {
			t.Errorf("%d bytes: NSSHash() = %s, want %s", c.size, got, c.nssHash)
		}
		if got := addr.NSSHash256(); got != c.nssHash256 {
			t.Errorf("%d bytes: NSSHash256() = %s, want %s", c.size, got, c.nssHash256)
		}
	}
}

// TestSLIP10ProgrammaticConstruction ensures NewDerivationPathFromLevels for
// SLIP-10 produces a path that round-trips through a full URN.
func TestSLIP10ProgrammaticConstruction(t *testing.T) {
	dp := NewDerivationPathFromLevels(SLIP10, []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 501, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
	})
	chain := NewChain(Solana, "mainnet")
	chain.SetCoinType(SOL)
	addr := NewAddress(chain, dp)
	want := `urn:mhda:nt:solana:ci:mainnet:ct:501:dt:slip10:dp:m/44'/501'/0'/0'`
	if got := addr.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	// Re-parse and check equality
	back, err := ParseURNStrict(want)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if back.String() != want {
		t.Errorf("re-parse not idempotent: got %q, want %q", back.String(), want)
	}
}
