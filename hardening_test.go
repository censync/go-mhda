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
