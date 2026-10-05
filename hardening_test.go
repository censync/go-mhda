package go_mhda

import (
	"errors"
	"strings"
	"testing"
)

// TestDuplicateComponentsRejected pins the duplicate-key rule for old and new
// components alike.
func TestDuplicateComponentsRejected(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:1:ci:2`,
		`urn:mhda:nt:evm:ci:1:ct:60:ct:61`,
		`urn:mhda:nt:evm:ci:1:wt:web3:wt:metamask`,
		`urn:mhda:nt:evm:ci:1:wi:a:wi:b`,
	} {
		if _, err := ParseURN(urn); !errors.Is(err, ErrInvalidNSS) {
			t.Errorf("ParseURN(%q): got %v, want errors.Is(ErrInvalidNSS)", urn, err)
		}
	}
}

// TestChainFromKeyCanonicalOnly: a chain key must BE the canonical identity
// string — unknown tokens, reordering and surrounding junk are all rejected,
// not silently normalised.
func TestChainFromKeyCanonicalOnly(t *testing.T) {
	for _, key := range []string{
		`nt:evm:ci:1:zz:junk`, // unknown trailing token
		`nt:evm:ci:1:foo`,     // dangling token
		`ci:1:nt:evm`,         // reordered
		`nt:EVM:ci:1`,         // non-canonical case in the enum value
	} {
		_, err := ChainFromKey(ChainKey(key))
		if !errors.Is(err, ErrInvalidChainKey) {
			t.Errorf("ChainFromKey(%q): got %v, want ErrInvalidChainKey", key, err)
		}
	}
	// Surrounding whitespace is tolerated (trimmed before the canonical
	// comparison) — a key embedded in config files commonly carries it.
	if _, err := ChainFromKey(ChainKey("  nt:evm:ci:1  ")); err != nil {
		t.Errorf("ChainFromKey with surrounding whitespace: %v", err)
	}
}

// TestFreeFormSettersRejectNSSCorruptingValues: the free-form components
// carry client-supplied strings; a value able to smuggle ':' (component
// injection), '?' / '#' (RFC 8141 truncation) or whitespace would break the
// round-trip guarantee — the setters must reject them loudly.
func TestFreeFormSettersRejectNSSCorruptingValues(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	a := addr.(*Address)
	cases := []struct {
		name string
		set  func(string) error
	}{
		{"SetWalletType", a.SetWalletType},
		{"SetWalletId", a.SetWalletId},
		{"SetAddressPrefix", a.SetAddressPrefix},
		{"SetAddressSuffix", a.SetAddressSuffix},
	}
	for _, c := range cases {
		for _, v := range []string{
			"x:dt:bip44", "ci:2", "abc#def", "x?y", "a b",
		} {
			if err := c.set(v); !errors.Is(err, ErrInvalidValue) {
				t.Errorf("%s(%q): got %v, want ErrInvalidValue", c.name, v, err)
			}
		}
	}
	// The rejected values must not have partially mutated the address.
	if got := a.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("address mutated by rejected values: %q", got)
	}
}

// TestNewAddressPanicsOnInvalidParams: constructor params ride the same
// validation as parsed input; an invalid value is a programmer error and
// panics (mirroring the NewDerivationPath precedent).
func TestNewAddressPanicsOnInvalidParams(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for an NSS-corrupting prefix param")
		}
	}()
	_ = NewAddress(NewChain(EthereumVM, "1"), nil, "", "", "x:evil", "")
}

// TestCoinTypeSpellings pins the documented ct grammar: plain decimal and
// 0x-prefixed hex only. Go integer-literal extras must be rejected.
func TestCoinTypeSpellings(t *testing.T) {
	valid := map[string]CoinType{
		`urn:mhda:nt:evm:ci:1:ct:60`:   60,
		`urn:mhda:nt:evm:ci:1:ct:0x3c`: 60,
		`urn:mhda:nt:evm:ci:1:ct:0X3C`: 60,
	}
	for urn, want := range valid {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if got := addr.Chain().CoinType(); got != want {
			t.Errorf("ParseURN(%q): ct = %d, want %d", urn, got, want)
		}
	}
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:1:ct:6_0`,
		`urn:mhda:nt:evm:ci:1:ct:0o74`,
		`urn:mhda:nt:evm:ci:1:ct:0b111100`,
		`urn:mhda:nt:evm:ci:1:ct:-1`,
		`urn:mhda:nt:evm:ci:1:ct:0x`,
	} {
		if _, err := ParseURN(urn); !errors.Is(err, ErrInvalidCoinType) {
			t.Errorf("ParseURN(%q): got %v, want ErrInvalidCoinType", urn, err)
		}
	}
}

// TestInteriorWhitespaceRejected: values with embedded whitespace cannot
// appear in a conforming NSS and would serialise into a non-parseable form.
func TestInteriorWhitespaceRejected(t *testing.T) {
	for _, urn := range []string{
		"urn:mhda:nt:evm:ci:a b",
		"urn:mhda:nt:evm:ci:1:wt:a b",
		"urn:mhda:nt:evm:ci:1:ap:0x 1",
	} {
		if _, err := ParseURN(urn); !errors.Is(err, ErrInvalidNSS) {
			t.Errorf("ParseURN(%q): got %v, want errors.Is(ErrInvalidNSS)", urn, err)
		}
	}
}

// TestNonASCIIValuesRejected: an NSS is printable ASCII by definition
// (RFC 8141). Unicode spaces must not be silently trimmed into a different
// chain identity, and non-ASCII value bytes must fail loudly.
func TestNonASCIIValuesRejected(t *testing.T) {
	for _, urn := range []string{
		"urn:mhda:nt:evm:ci:1\u3000",                  // ideographic space in ci
		"urn:mhda:nt:evm:ci:1\u00a0",                  // NBSP in ci
		"urn:mhda:nt:evm:ci:\u0442\u0435\u0441\u0442", // Cyrillic value
		"urn:mhda:nt:evm:ci:1:wt:web\u00a03",          // NBSP inside wt
	} {
		if _, err := ParseURN(urn); err == nil {
			t.Errorf("ParseURN(%q): expected rejection of non-ASCII value", urn)
		}
	}
	// Setters enforce the same rule for programmatic input.
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	a := addr.(*Address)
	if err := a.SetWalletId("a\u00a0b"); !errors.Is(err, ErrInvalidValue) {
		t.Errorf("SetWalletId(NBSP): got %v, want ErrInvalidValue", err)
	}
	// Keys with a trailing Unicode space are not the canonical string.
	if _, err := ChainFromKey(ChainKey("nt:evm:ci:1\u3000")); err == nil {
		t.Error("ChainFromKey with trailing ideographic space must fail")
	}
}

// TestFreeFormValuesPreserveCase: enum-valued components normalise to
// lowercase; free-form values (ci/ap/as/wt/wi) are case-preserving and
// round-trip verbatim.
func TestFreeFormValuesPreserveCase(t *testing.T) {
	in := `urn:mhda:nt:evm:ci:0xAbC:ap:0xPREFIX:wt:Web3:wi:ABCDEF`
	addr, err := ParseURN(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.String(); got != in {
		t.Errorf("round-trip:\n got:  %s\n want: %s", got, in)
	}
	if !strings.Contains(addr.String(), ":wt:Web3") {
		t.Error("wallet type lost its case")
	}
}

// panicErr runs f and returns the error it panicked with, or nil when it did
// not panic.
func panicErr(t *testing.T, f func()) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				t.Fatalf("panicked with a non-error value %v", r)
			}
			err = e
		}
	}()
	f()
	return nil
}

// TestProgrammaticValuesCannotInjectComponents: the network type, the chain
// id and the derivation type are written verbatim into every URN. A value
// carrying ':' would inject components on re-parse (a root address whose URN
// reads back as a bip44 path), '?' / '#' would truncate it. Constructors
// panic on such a value, setters return the error and keep the old value.
func TestProgrammaticValuesCannotInjectComponents(t *testing.T) {
	for _, c := range []struct {
		nt   NetworkType
		ci   ChainId
		want error
	}{
		{EthereumVM, "1:dt:bip44:dp:m/44'/60'/0'/0/666", ErrInvalidValue},
		{EthereumVM, "1:wi:attacker", ErrInvalidValue},
		{EthereumVM, "1?=q", ErrInvalidValue},
		{EthereumVM, "1#f", ErrInvalidValue},
		{EthereumVM, "a b", ErrInvalidValue},
		{EthereumVM, " 1", ErrInvalidValue},
		{EthereumVM, "", ErrMissingChainID},
		{EthereumVM, " \t", ErrMissingChainID},
		{NetworkType("evm:ci:1:dt:bip44"), "1", ErrInvalidNetworkType},
		{NetworkType("polkadot"), "1", ErrInvalidNetworkType},
		{NetworkType("EVM"), "1", ErrInvalidNetworkType},
		{NetworkType(""), "1", ErrInvalidNetworkType},
	} {
		err := panicErr(t, func() { NewChain(c.nt, c.ci) })
		if !errors.Is(err, c.want) {
			t.Errorf("NewChain(%q, %q): panic %v, want %v", c.nt, c.ci, err, c.want)
		}
	}

	if got := NewChain(EthereumVM, " 0x1 ").ChainId(); got != "0x1" {
		t.Errorf("NewChain trims ASCII whitespace: ChainId() = %q, want %q", got, "0x1")
	}

	ch := NewChain(EthereumVM, "1")
	if err := ch.SetChainId("1:dt:bip44"); !errors.Is(err, ErrInvalidValue) {
		t.Errorf("SetChainId: got %v, want ErrInvalidValue", err)
	}
	if err := ch.SetChainId(""); !errors.Is(err, ErrMissingChainID) {
		t.Errorf("SetChainId(\"\"): got %v, want ErrMissingChainID", err)
	}
	if err := ch.SetNetworkType(NetworkType("evm:x")); !errors.Is(err, ErrInvalidNetworkType) {
		t.Errorf("SetNetworkType: got %v, want ErrInvalidNetworkType", err)
	}
	if got := ch.String(); got != "nt:evm:ci:1" {
		t.Errorf("failed setters changed the chain: %q", got)
	}
	if err := ch.SetChainId("56"); err != nil {
		t.Errorf("SetChainId(56): %v", err)
	}
	if err := ch.SetNetworkType(Bitcoin); err != nil {
		t.Errorf("SetNetworkType(Bitcoin): %v", err)
	}
	if got := ch.String(); got != "nt:bitcoin:ci:56" {
		t.Errorf("setters: String() = %q", got)
	}

	for _, dt := range []DerivationType{"bip44:wi:x", "bogus", "BIP44", ""} {
		err := panicErr(t, func() { NewDerivationPath(dt, 60, 0, 0, AddressIndex{}) })
		if !errors.Is(err, ErrInvalidDerivationType) {
			t.Errorf("NewDerivationPath(%q): panic %v, want ErrInvalidDerivationType", dt, err)
		}
		err = panicErr(t, func() { NewDerivationPathFromLevels(dt, nil) })
		if !errors.Is(err, ErrInvalidDerivationType) {
			t.Errorf("NewDerivationPathFromLevels(%q): panic %v, want ErrInvalidDerivationType", dt, err)
		}
	}

	// A zero DerivationPath has no type; it serialises like a root address
	// instead of emitting empty dt/dp components.
	addr := NewAddress(NewChain(EthereumVM, "1"), &DerivationPath{})
	if got := addr.String(); got != "urn:mhda:nt:evm:ci:1" {
		t.Errorf("zero DerivationPath: String() = %q, want urn:mhda:nt:evm:ci:1", got)
	}
}

// TestZIP32HasNoNetwork pins SPEC §12: zip32 parses leniently, but no
// network registers it, so strict parsing refuses it on every network.
func TestZIP32HasNoNetwork(t *testing.T) {
	for nt := range ntIndex {
		urn := `urn:mhda:nt:` + nt + `:ci:x:dt:zip32:dp:m/32'/133'/0'`
		if _, err := ParseURN(urn); err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
		}
		if _, err := ParseURNStrict(urn); !errors.Is(err, ErrIncompatible) {
			t.Errorf("ParseURNStrict(%q): got %v, want ErrIncompatible", urn, err)
		}
	}
}

// TestFromStringHelpersWrapSentinels: every lookup helper reports an unknown
// value with the sentinel of its component, as doc.go promises.
func TestFromStringHelpersWrapSentinels(t *testing.T) {
	if _, err := AlgorithmFromString("rsa"); !errors.Is(err, ErrInvalidAlgorithm) {
		t.Errorf("AlgorithmFromString: got %v, want ErrInvalidAlgorithm", err)
	}
	if _, err := FormatFromString("zzz"); !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("FormatFromString: got %v, want ErrInvalidFormat", err)
	}
	if _, err := NetworkTypeFromString("doge"); !errors.Is(err, ErrInvalidNetworkType) {
		t.Errorf("NetworkTypeFromString: got %v, want ErrInvalidNetworkType", err)
	}
	if _, err := DerivationTypeFromString("bip99"); !errors.Is(err, ErrInvalidDerivationType) {
		t.Errorf("DerivationTypeFromString: got %v, want ErrInvalidDerivationType", err)
	}
}

// TestNormalisationIsASCIIOnly: setters and lookup helpers trim ASCII
// whitespace and lowercase ASCII letters only, like the URN parser and the
// C++ port. A Unicode space is not trimmed and a letter that lowercases to
// ASCII (the Kelvin sign) is not folded: both are malformed input.
func TestNormalisationIsASCIIOnly(t *testing.T) {
	const kelvin, nbsp, ideo = "K", " ", "　"
	addr := mustAddress(t, `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`)
	for name, err := range map[string]error{
		"AlgorithmFromString":   func() error { _, e := AlgorithmFromString("secp256" + kelvin + "1"); return e }(),
		"FormatFromString":      func() error { _, e := FormatFromString(nbsp + "hex"); return e }(),
		"NetworkTypeFromString": func() error { _, e := NetworkTypeFromString(nbsp + "evm"); return e }(),
		"DerivationTypeFromStr": func() error { _, e := DerivationTypeFromString(nbsp + "BIP44"); return e }(),
		"SetAddressAlgorithm":   addr.SetAddressAlgorithm("secp256" + kelvin + "1"),
		"SetAddressFormat":      addr.SetAddressFormat(nbsp + "hex" + nbsp),
		"SetDerivationType":     addr.SetDerivationType("bip44" + ideo),
		"SetDerivationPath":     addr.SetDerivationPath(nbsp + "m/44'/60'/0'/0/1"),
		"SetCoinType":           addr.SetCoinType(nbsp + "60"),
	} {
		if err == nil {
			t.Errorf("%s accepted a non-ASCII spelling", name)
		}
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0` {
		t.Errorf("refused values changed the address: %q", got)
	}

	// ASCII whitespace and case are still normalised.
	if a, err := AlgorithmFromString(" SECP256K1\t"); err != nil || a != Secp256k1 {
		t.Errorf("AlgorithmFromString: %q, %v", a, err)
	}
	if nt, err := NetworkTypeFromString("\tEVM "); err != nil || nt != EthereumVM {
		t.Errorf("NetworkTypeFromString: %q, %v", nt, err)
	}
	if err := addr.SetDerivation(" BIP44 ", " M/44H/60H/0H/0/1 "); err != nil {
		t.Errorf("SetDerivation with ASCII case and spaces: %v", err)
	}
}
