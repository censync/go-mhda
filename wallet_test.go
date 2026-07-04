package go_mhda

import (
	"errors"
	"testing"
)

// TestWalletDomainRoundTrip covers the optional wallet domain (wt/wi): both
// components together, each alone, and their canonical trailing position.
func TestWalletDomainRoundTrip(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:1:wt:web3:wi:5f2a8c31`,
		`urn:mhda:nt:evm:ci:1:wt:metamask`,
		`urn:mhda:nt:evm:ci:1:wi:c0a8f2d4-3b6e-4a51-9c7d-2f8e1a0b5c93`,
		`urn:mhda:nt:ton:ci:mainnet:wt:tonconnect`,
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0:aa:secp256k1:af:hex:ap:0x:wt:web3:wi:5f2a8c31`,
	} {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if got := addr.String(); got != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", got, urn)
		}
	}
}

// TestWalletDomainAccessors verifies the getters on Address and via the MHDA
// interface.
func TestWalletDomainAccessors(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1:wt:web3:wi:5f2a8c31`)
	if err != nil {
		t.Fatal(err)
	}
	if got := addr.WalletType(); got != "web3" {
		t.Errorf("WalletType() = %q, want %q", got, "web3")
	}
	if got := addr.WalletId(); got != "5f2a8c31" {
		t.Errorf("WalletId() = %q, want %q", got, "5f2a8c31")
	}
	var iface MHDA = addr
	if iface.WalletType() != "web3" || iface.WalletId() != "5f2a8c31" {
		t.Error("wallet-domain accessors not reachable via the MHDA interface")
	}

	// Unset wallet domain reads as empty.
	bare, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	if bare.WalletType() != "" || bare.WalletId() != "" {
		t.Error("unset wallet domain must read as empty strings")
	}
}

// TestWalletDomainSettersReset ensures empty-string SetX resets the field,
// matching the semantics of the other optional components.
func TestWalletDomainSettersReset(t *testing.T) {
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	concrete := addr.(*Address)
	if err := concrete.SetWalletType("web3"); err != nil {
		t.Fatal(err)
	}
	if err := concrete.SetWalletId("5f2a8c31"); err != nil {
		t.Fatal(err)
	}
	if want := `urn:mhda:nt:evm:ci:1:wt:web3:wi:5f2a8c31`; concrete.String() != want {
		t.Errorf("String() = %q, want %q", concrete.String(), want)
	}
	if err := concrete.SetWalletType(""); err != nil {
		t.Fatal(err)
	}
	if err := concrete.SetWalletId(""); err != nil {
		t.Fatal(err)
	}
	if want := `urn:mhda:nt:evm:ci:1`; concrete.String() != want {
		t.Errorf("after reset: String() = %q, want %q", concrete.String(), want)
	}
}

// TestWalletDomainOrderIndependence: parsers accept any component order on
// input; the wallet domain re-serializes in the canonical trailing position.
func TestWalletDomainOrderIndependence(t *testing.T) {
	canonical := `urn:mhda:nt:evm:ci:1:ct:60:wt:web3:wi:5f2a8c31`
	for _, in := range []string{
		`urn:mhda:wt:web3:wi:5f2a8c31:nt:evm:ci:1:ct:60`,
		`urn:mhda:nt:evm:wt:web3:ci:1:wi:5f2a8c31:ct:60`,
		`urn:mhda:nt:evm:ct:60:ci:1:wt:web3:wi:5f2a8c31`, // pre-1.1 ct position
	} {
		addr, err := ParseURN(in)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", in, err)
			continue
		}
		if got := addr.String(); got != canonical {
			t.Errorf("for %q:\n got:  %s\n want: %s", in, got, canonical)
		}
	}
}

// TestWalletDomainEmptyValuesRejected: empty values for wt/wi are malformed,
// consistent with every other component.
func TestWalletDomainEmptyValuesRejected(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:1:wt:`,
		`urn:mhda:nt:evm:ci:1:wt::wi:x`,
		`urn:mhda:nt:evm:ci:1:wi:`,
	} {
		if _, err := ParseURN(urn); err == nil {
			t.Errorf("expected error for %q, got nil", urn)
		}
	}
}

// TestWalletDomainStrictValidation: the wallet domain is orthogonal metadata;
// strict validation must pass with it present and keep rejecting incompatible
// triples regardless of it.
func TestWalletDomainStrictValidation(t *testing.T) {
	if _, err := ParseURNStrict(`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0:wt:web3:wi:5f2a8c31`); err != nil {
		t.Errorf("strict parse with wallet domain failed: %v", err)
	}
	_, err := ParseURNStrict(`urn:mhda:nt:evm:ci:1:aa:ed25519:wt:web3`)
	if !errors.Is(err, ErrIncompatible) {
		t.Errorf("wallet domain must not mask incompatibility: got %v", err)
	}
}

// TestCoinTypeOptional pins the 1.1 semantics of ct: absent stays absent,
// present round-trips in the canonical position after ci, and the value
// normalizes to decimal.
func TestCoinTypeOptional(t *testing.T) {
	// Absent: no ct in output, HasCoinType false.
	addr, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	if addr.Chain().HasCoinType() {
		t.Error("HasCoinType() = true for a URN without ct")
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("String() = %q, want %q", got, `urn:mhda:nt:evm:ci:1`)
	}

	// Present: captured, emitted after ci, decimal-normalized (0x3c -> 60).
	addr, err = ParseURN(`urn:mhda:nt:evm:ci:1:ct:0x3c`)
	if err != nil {
		t.Fatal(err)
	}
	if !addr.Chain().HasCoinType() || addr.Chain().CoinType() != 60 {
		t.Errorf("ct not captured: has=%v ct=%d", addr.Chain().HasCoinType(), addr.Chain().CoinType())
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ci:1:ct:60` {
		t.Errorf("String() = %q, want %q", got, `urn:mhda:nt:evm:ci:1:ct:60`)
	}

	// SetCoinType with an empty string clears the metadata.
	concrete := addr.(*Address)
	if err := concrete.SetCoinType(""); err != nil {
		t.Fatal(err)
	}
	if concrete.Chain().HasCoinType() {
		t.Error("SetCoinType(\"\") must clear the metadata")
	}
	if got := concrete.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("after clear: String() = %q, want %q", got, `urn:mhda:nt:evm:ci:1`)
	}
}
