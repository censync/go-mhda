package go_mhda

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

var (
	uriMHDA = []string{
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/1'/0/1:aa:secp256k1:af:hex:ap:0x`,
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/2'/0/2'`,
		`urn:mhda:nt:evm:ct:60:ci:1`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin_testnet:dt:bip44:dp:m/44'/0'/0'/0/0`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/1'/0/1:aa:secp256k1:af:p2pkh:ap:1`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/2'/0/2`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`,
		`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`,
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
		{`mhda:nt:evm:ct:60:ci:1`, ErrInvalidURN},
		{`urn:mhda:ct:60:ci:1`, ErrMissingNetworkType},
		{`urn:mhda:nt:notanetwork:ct:60:ci:1`, ErrInvalidNetworkType},
		{`urn:mhda:nt:evm:ci:1`, ErrMissingCoinType},
		{`urn:mhda:nt:evm:ct:notanumber:ci:1`, ErrInvalidCoinType},
		{`urn:mhda:nt:evm:ct:60`, ErrMissingChainID},
		{`urn:mhda:nt:evm:ct:60:ci:1:dt:bipxx:dp:m/44'/60'/0'/0/0`, ErrInvalidDerivationType},
		{`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:not_a_path`, ErrInvalidDerivationPath},
		{`urn:mhda:nt:evm:ct:60:ci:1:aa:rsa`, ErrInvalidAlgorithm},
		{`urn:mhda:nt:evm:ct:60:ci:1:af:notaformat`, ErrInvalidFormat},
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
		{`urn:mhda:nt:evm:ct:60:ci:1`, false},
		{`urn:mhda:nt:evm:ct:60:ci:1:aa:secp256k1:af:hex`, false},
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh`, false},
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`, false},
		{`urn:mhda:nt:sol:ct:501:ci:mainnet`, false},
		{`urn:mhda:nt:sol:ct:501:ci:mainnet:aa:ed25519:af:base58`, false},
		{`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`, false},

		// invalid combos
		{`urn:mhda:nt:evm:ct:60:ci:1:aa:ed25519`, true},          // evm + ed25519
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:aa:ed25519`, true},     // btc + ed25519
		{`urn:mhda:nt:evm:ct:60:ci:1:af:bech32`, true},           // evm + bech32
		{`urn:mhda:nt:sol:ct:501:ci:mainnet:aa:secp256k1`, true}, // sol + secp256k1
		{`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:af:hex`, true},  // cosmos + hex
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
	bad := `urn:mhda:nt:evm:ct:60:ci:1:aa:ed25519`
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
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2sh`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:p2wpkh`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:p2wsh`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:p2tr`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m`,
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
			`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/1'/0/2`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 60, IsHardened: true},
				{Index: 1, IsHardened: true},
				{Index: 0, IsHardened: false},
				{Index: 2, IsHardened: false},
			},
		},
		{
			`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/3'/0/7`,
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
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0:aa:secp256k1:af:hex:ap:0x`,
		// Bitcoin
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/0'/0/0:af:p2pkh:ap:1`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip49:dp:m/49'/0'/0'/0/0:af:p2sh:ap:3`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/84'/0'/0'/0/0:af:bech32:ap:bc1q`,
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`,
		// Avalanche
		`urn:mhda:nt:evm:ct:60:ci:0xa86a:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:avm:ct:9000:ci:1:dt:bip44:dp:m/44'/9000'/0'/0/0:af:bech32:ap:X-avax`,
		// Solana
		`urn:mhda:nt:sol:ct:501:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`,
		// XRP
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0`,
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0:aa:ed25519`,
		// Stellar
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`,
		// NEAR
		`urn:mhda:nt:near:ct:397:ci:mainnet:dt:slip10:dp:m/44'/397'/0'`,
		`urn:mhda:nt:near:ct:397:ci:mainnet:dt:bip44:dp:m/44'/397'/0'/0/0:aa:secp256k1`,
		// Aptos
		`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
		`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:bip44:dp:m/44'/637'/0'/0/0:aa:secp256k1`,
		// Sui
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`,
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`,
		// Cardano
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`,
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/2/0`,
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0:af:base58`,
		// Algorand
		`urn:mhda:nt:algo:ct:283:ci:mainnet`,
		`urn:mhda:nt:algo:ct:283:ci:mainnet:dt:slip10:dp:m/44'/283'/0'/0'/0'`,
		// TON
		`urn:mhda:nt:ton:ct:607:ci:mainnet`,
		`urn:mhda:nt:ton:ct:607:ci:mainnet:af:hex`,
		`urn:mhda:nt:ton:ct:607:ci:mainnet:dt:slip10:dp:m/44'/607'/0'`,
		// Cosmos
		`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`,
		// Root key
		`urn:mhda:nt:evm:ct:60:ci:1`,
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
		`urn:mhda:nt:ton:ct:607:ci:mainnet`,
		// Long form
		`urn:mhda:nt:ton:ct:607:ci:mainnet:aa:ed25519:af:base64url`,
		// Raw hex form (the protocol-internal canonical address representation)
		`urn:mhda:nt:ton:ct:607:ci:mainnet:af:hex`,
		// Ledger-style HD via SLIP-10
		`urn:mhda:nt:ton:ct:607:ci:mainnet:dt:slip10:dp:m/44'/607'/0'`,
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
	addr, err := ParseURN(`urn:mhda:nt:ton:ct:607:ci:mainnet`)
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
		`urn:mhda:nt:ton:ct:607:ci:mainnet:aa:secp256k1`,
		`urn:mhda:nt:ton:ct:607:ci:mainnet:aa:sr25519`,
		// Disallowed formats
		`urn:mhda:nt:ton:ct:607:ci:mainnet:af:base58`,
		`urn:mhda:nt:ton:ct:607:ci:mainnet:af:bech32`,
		`urn:mhda:nt:ton:ct:607:ci:mainnet:af:strkey`,
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
		`urn:mhda:nt:algo:ct:283:ci:mainnet`,
		// With explicit defaults
		`urn:mhda:nt:algo:ct:283:ci:mainnet:aa:ed25519:af:base32`,
		// Third-party SLIP-10 layered HD form
		`urn:mhda:nt:algo:ct:283:ci:mainnet:dt:slip10:dp:m/44'/283'/0'/0'/0'`,
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
	addr, err := ParseURN(`urn:mhda:nt:algo:ct:283:ci:mainnet`)
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
		`urn:mhda:nt:algo:ct:283:ci:mainnet:aa:secp256k1`,
		`urn:mhda:nt:algo:ct:283:ci:mainnet:aa:sr25519`,
		// Algorand uses base32 only
		`urn:mhda:nt:algo:ct:283:ci:mainnet:af:hex`,
		`urn:mhda:nt:algo:ct:283:ci:mainnet:af:base58`,
		`urn:mhda:nt:algo:ct:283:ci:mainnet:af:bech32`,
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
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`,
		// long form
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0:aa:ed25519:af:bech32`,
		// staking key (role=2 per CIP-1852)
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/2/0`,
		// internal change address (role=1)
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/1/3`,
		// Byron-era legacy address: base58 format still accepted
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0:af:base58`,
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
	addr, err := ParseURN(`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`)
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
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:aa:secp256k1`,
		// hex / strkey are not Cardano formats
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:af:hex`,
		`urn:mhda:nt:ada:ct:1815:ci:mainnet:af:strkey`,
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
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
		// secp256k1 via BIP-54
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`,
		// secp256r1 via BIP-74
		`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`,
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
	addr, err := ParseURN(`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`)
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
		{`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`, 44},
		{`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`, 54},
		{`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`, 74},
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
		`urn:mhda:nt:sui:ct:784:ci:mainnet:aa:sr25519`, // wrong curve
		`urn:mhda:nt:sui:ct:784:ci:mainnet:af:base58`,  // not Sui
		`urn:mhda:nt:sui:ct:784:ci:mainnet:af:strkey`,  // not Sui
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
		`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
		// long form
		`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0':aa:ed25519:af:hex`,
		// secp256k1 BIP-44 form
		`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:bip44:dp:m/44'/637'/0'/0/0:aa:secp256k1`,
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
	addr, err := ParseURN(`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0'`)
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
		`urn:mhda:nt:apt:ct:637:ci:mainnet:aa:sr25519`, // wrong curve
		`urn:mhda:nt:apt:ct:637:ci:mainnet:af:base58`,  // not Aptos
		`urn:mhda:nt:apt:ct:637:ci:mainnet:af:bech32`,  // not Aptos
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
		`urn:mhda:nt:near:ct:397:ci:mainnet:dt:slip10:dp:m/44'/397'/0'`,
		// long form
		`urn:mhda:nt:near:ct:397:ci:mainnet:dt:slip10:dp:m/44'/397'/0':aa:ed25519:af:hex`,
		// secp256k1 variant (ETH-implicit accounts)
		`urn:mhda:nt:near:ct:397:ci:mainnet:dt:bip44:dp:m/44'/397'/0'/0/0:aa:secp256k1`,
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
	addr, err := ParseURN(`urn:mhda:nt:near:ct:397:ci:mainnet:dt:slip10:dp:m/44'/397'/0'`)
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
		`urn:mhda:nt:near:ct:397:ci:mainnet:aa:sr25519`, // wrong curve
		`urn:mhda:nt:near:ct:397:ci:mainnet:af:bech32`,  // NEAR has no bech32
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
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`,
		// long form
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0':aa:ed25519:af:strkey`,
		// non-zero account from SEP-0005 vectors
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/3'`,
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
	addr, err := ParseURN(`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`)
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
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:aa:secp256k1`, // wrong curve
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:af:bech32`,    // wrong format
		`urn:mhda:nt:xlm:ct:148:ci:mainnet:af:base58`,    // also rejected
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
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0`,
		// long form with explicit defaults
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0:aa:secp256k1:af:base58`,
		// ed25519 variant (XLS-10 / XUMM-style)
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0:aa:ed25519:af:base58`,
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
	addr, err := ParseURN(`urn:mhda:nt:xrp:ct:144:ci:mainnet:dt:bip44:dp:m/44'/144'/0'/0/0`)
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
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:aa:sr25519`, // unsupported algo
		`urn:mhda:nt:xrp:ct:144:ci:mainnet:af:hex`,     // unsupported format
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
		`urn:mhda:nt:evm:ct:283:ci:mainnet:af:base32`,
		`urn:mhda:nt:evm:ct:148:ci:mainnet:af:strkey`,
		`urn:mhda:nt:evm:ct:607:ci:mainnet:af:base64url`,
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
		{`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, 0},
		// role 1 = internal (change)
		{`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/1/0`, 1},
		// role 2 = staking key
		{`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/2/0`, 2},
		// role 3 = DRep (CIP-105)
		{`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/3/0`, 3},
		// hardened address index
		{`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/3'/2/7'`, 2},
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
	addr, err := ParseURN(`urn:mhda:nt:evm:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/3'/2/7`)
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
			`urn:mhda:nt:sol:ct:501:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 501, IsHardened: true},
				{Index: 0, IsHardened: true},
				{Index: 0, IsHardened: true},
			},
		},
		{
			// Stellar SEP-0005: m/44'/148'/account'
			`urn:mhda:nt:evm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`,
			[]AddressIndex{
				{Index: 44, IsHardened: true},
				{Index: 148, IsHardened: true},
				{Index: 0, IsHardened: true},
			},
		},
		{
			// Sui ed25519: m/44'/784'/account'/change'/index'
			`urn:mhda:nt:evm:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`,
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
			`urn:mhda:nt:evm:ct:637:ci:mainnet:dt:slip10:dp:m/44'/637'/0'/0'/0'`,
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
	urn := `urn:mhda:nt:evm:ct:0:ci:mainnet:dt:slip10:dp:m/44'/0'/0'/0/5`
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
		`urn:mhda:nt:btc:ct:133:ci:zcash:dt:zip32:dp:m/32'/133'/0'`,
		`urn:mhda:nt:btc:ct:133:ci:zcash:dt:zip32:dp:m/32'/133'/0'/0`,
		`urn:mhda:nt:btc:ct:133:ci:zcash:dt:zip32:dp:m/32'/133'/0'/0'`,
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
		`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip49:dp:m/49'/0'/0'/0/0:af:p2sh`,
		// sui not yet registered as network type; use evm as a syntactic stand-in
		// for path parsing - Phase C will add proper Sui entries.
		`urn:mhda:nt:evm:ct:784:ci:1:dt:bip54:dp:m/54'/784'/0'/0/0`,
		`urn:mhda:nt:evm:ct:784:ci:1:dt:bip74:dp:m/74'/784'/0'/0/0`,
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
	in := `urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/3'/0/7`
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
	addr, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Format() != HEX {
		t.Errorf("Format() default for evm = %q, want %q", addr.Format(), HEX)
	}
	if want := `urn:mhda:nt:evm:ct:60:ci:1`; addr.String() != want {
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
		{`urn:mhda:nt:evm:ct:60:ci:1`, Secp256k1},
		{`urn:mhda:nt:sol:ct:501:ci:mainnet`, Ed25519},
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44'/0'/1'/0/1:aa:secp256k1:af:p2pkh:ap:1`, Secp256k1},
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

func BenchmarkParseRx(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseURNRx(uriMHDA[0])
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
	addr, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.Chain().NetworkType(); got != EthereumVM {
		t.Errorf("NetworkType() = %q, want %q", got, EthereumVM)
	}
	if got := addr.Chain().CoinType(); got != ETH {
		t.Errorf("CoinType() = %d, want %d", got, ETH)
	}
	if got := addr.Chain().ChainId(); got != "1" {
		t.Errorf("ChainId() = %q, want %q", got, "1")
	}
	// Key() is computed from String(); make sure it works inline too.
	want := "nt:evm:ct:60:ci:1"
	if got := string(addr.Chain().Key()); got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
}

// TestChainSettersMutate verifies SetX persistently mutate the chain. Was
// broken when setters had value receivers (silent no-op).
func TestChainSettersMutate(t *testing.T) {
	c := NewChain(EthereumVM, ETH, "1")
	c.SetNetworkType(Bitcoin)
	c.SetCoinType(BTC)
	c.SetChainId("bitcoin")
	if c.NetworkType() != Bitcoin || c.CoinType() != BTC || c.ChainId() != "bitcoin" {
		t.Errorf("setters did not persist: nt=%q ct=%d ci=%q",
			c.NetworkType(), c.CoinType(), c.ChainId())
	}
}

// TestSetPrefixSuffixReset ensures empty-string SetX resets the field, matching
// the semantics of SetAddressAlgorithm/Format.
func TestSetPrefixSuffixReset(t *testing.T) {
	addr := NewAddress(NewChain(EthereumVM, ETH, "1"), nil, "", "", "0xPREFIX", "SUFFIX")
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
	if got := addr.String(); got != `urn:mhda:nt:evm:ct:60:ci:1` {
		t.Errorf("after reset: got %q, want %q", got, `urn:mhda:nt:evm:ct:60:ci:1`)
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
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44H/60H/0H/0/0`: `urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44h/60h/0h/0/0`: `urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`,
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
	addr, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.DerivationType(); got != BIP44 {
		t.Errorf("DerivationType() = %q, want %q", got, BIP44)
	}
	// ROOT path
	root, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1`)
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
	var a *Address // nil
	_, err := a.MarshalText()
	if !errors.Is(err, ErrUninitializedAddress) {
		t.Errorf("nil receiver: got %v, want ErrUninitializedAddress", err)
	}
	a2 := &Address{} // zero-value, no chain
	_, err = a2.MarshalText()
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
	if err := a.UnmarshalText([]byte("urn:mhda:nt:evm:ct:60:ci:1")); err != nil {
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
	_, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/0'/0'/0/99999999999999999999`)
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
		{"xrp", XRPLedger, false},
		{"  XRP  ", XRPLedger, false}, // case + whitespace tolerance
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

// TestChainStringFormat closes the loop on a previously broken Chain.String()
// that emitted networkType twice instead of (networkType, coinType). The
// canonical form uses ':' separators throughout so it is itself a valid NSS
// key consumable by ChainFromKey / ChainFromNSS.
func TestChainStringFormat(t *testing.T) {
	c := NewChain(EthereumVM, ETH, "0x1")
	want := "nt:evm:ct:60:ci:0x1"
	if got := c.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := string(c.Key()); got != want {
		t.Errorf("Key() = %q, want %q", got, want)
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
		{`urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`, nil},
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m`, nil},
		{`urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`, nil},
		{`urn:mhda:nt:sol:ct:501:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`, nil},
		{`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, nil},
		// ROOT is always permitted (non-HD form)
		{`urn:mhda:nt:algo:ct:283:ci:mainnet`, nil},
		{`urn:mhda:nt:ton:ct:607:ci:mainnet`, nil},
		// Sui's purpose-variant schemes
		{`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip54:dp:m/54'/784'/0'/0/0:aa:secp256k1`, nil},
		{`urn:mhda:nt:sui:ct:784:ci:mainnet:dt:bip74:dp:m/74'/784'/0'/0/0:aa:secp256r1`, nil},

		// disallowed combos (network has no business with this derivation)
		// EVM does not use cip1852
		{`urn:mhda:nt:evm:ct:1815:ci:1:dt:cip1852:dp:m/1852'/1815'/0'/0/0`, ErrIncompatible},
		// Solana uses slip10 only, not bip44
		{`urn:mhda:nt:sol:ct:501:ci:mainnet:dt:bip44:dp:m/44'/501'/0'/0/0`, ErrIncompatible},
		// Cardano uses cip1852 only, not bip44
		{`urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:bip44:dp:m/44'/1815'/0'/0/0`, ErrIncompatible},
		// Bitcoin does not use cip11 (Cosmos-specific)
		{`urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:cip11:dp:m/44'/118'/0'/0/0`, ErrIncompatible},
		// Sui's purpose-54 path makes no sense on Aptos
		{`urn:mhda:nt:apt:ct:637:ci:mainnet:dt:bip54:dp:m/54'/637'/0'/0/0:aa:secp256k1`, ErrIncompatible},
		// Stellar is slip10-only
		{`urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:bip44:dp:m/44'/148'/0'/0/0`, ErrIncompatible},
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
	canonical := `urn:mhda:nt:evm:ct:60:ci:1`
	for _, in := range []string{
		`urn:mhda:nt:evm:ct:60:ci:1`, // baseline
		`URN:MHDA:nt:evm:ct:60:ci:1`,
		`Urn:Mhda:nt:evm:ct:60:ci:1`,
		`URN:mhda:nt:evm:ct:60:ci:1`,
		`urn:MHDA:nt:evm:ct:60:ci:1`,
		`  urn:mhda:nt:evm:ct:60:ci:1  `, // surrounding whitespace
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
	canonical := `urn:mhda:nt:evm:ct:60:ci:1`
	for _, in := range []string{
		`urn:mhda:nt:evm:ct:60:ci:1?+resolver=example.com`,
		`urn:mhda:nt:evm:ct:60:ci:1?=v=1`,
		`urn:mhda:nt:evm:ct:60:ci:1#fragment`,
		`urn:mhda:nt:evm:ct:60:ci:1?+a=b#frag`,
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
	in := "urn:mhdA:nt:BtC:ct:0:ci:0 #"
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

// TestRFC8141ParseURNRx asserts the regex parser also honours RFC 8141
// (was previously case-sensitive and lacked rq/f-component handling).
func TestRFC8141ParseURNRx(t *testing.T) {
	addr, err := ParseURNRx(`URN:MHDA:nt:evm:ct:60:ci:1?+x=y`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ct:60:ci:1` {
		t.Errorf("got %q", got)
	}
}

// TestHashFunctions covers the four hash methods on Address. Hash and
// NSSHash use SHA-1 (kept for backward compatibility); Hash256 and NSSHash256
// use SHA-256. All four must produce stable, deterministic output and the
// SHA-256 forms must differ from the SHA-1 forms.
func TestHashFunctions(t *testing.T) {
	addr1, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	addr2, err := ParseURN(`urn:mhda:nt:evm:ct:60:ci:1`)
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
	addrB, _ := ParseURN(`urn:mhda:nt:evm:ct:60:ci:2`)
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

// TestSLIP10ProgrammaticConstruction ensures NewDerivationPathFromLevels for
// SLIP-10 produces a path that round-trips through a full URN.
func TestSLIP10ProgrammaticConstruction(t *testing.T) {
	dp := NewDerivationPathFromLevels(SLIP10, []AddressIndex{
		{Index: 44, IsHardened: true},
		{Index: 501, IsHardened: true},
		{Index: 0, IsHardened: true},
		{Index: 0, IsHardened: true},
	})
	chain := NewChain(Solana, SOL, "mainnet")
	addr := NewAddress(chain, dp)
	want := `urn:mhda:nt:sol:ct:501:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`
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
