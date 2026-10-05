package go_mhda

import (
	"errors"
	"testing"
)

// TestReparseLeavesNoStaleState: parsing replaces the whole path, and a
// failed parse leaves it as it was.
func TestReparseLeavesNoStaleState(t *testing.T) {
	// ZIP-32: a 3-level path after a 4-level one has no index left over.
	dp, err := ParseDerivationPath(ZIP32, "m/32'/133'/0'/5")
	if err != nil {
		t.Fatal(err)
	}
	if err := dp.ParsePath("m/32'/133'/1'"); err != nil {
		t.Fatal(err)
	}
	if got := dp.String(); got != "m/32'/133'/1'" {
		t.Errorf("ZIP-32 re-parse: String() = %q, want m/32'/133'/1'", got)
	}
	if n := len(dp.Levels()); n != 3 {
		t.Errorf("ZIP-32 re-parse: %d levels, want 3", n)
	}
	if dp.AddressIndex() != (AddressIndex{}) {
		t.Errorf("ZIP-32 re-parse kept index %+v", dp.AddressIndex())
	}

	// SLIP-10 after BIP-44: no BIP-44 shortcut survives.
	sl, err := ParseDerivationPath(BIP44, "m/44'/60'/5'/1/9")
	if err != nil {
		t.Fatal(err)
	}
	sl.derivationType = SLIP10
	if err := sl.ParsePath("m/0'"); err != nil {
		t.Fatal(err)
	}
	if sl.Coin() != 0 || sl.Account() != 0 || sl.Charge() != 0 || sl.AddressIndex() != (AddressIndex{}) {
		t.Errorf("SLIP-10 re-parse kept shortcuts: coin=%d account=%d charge=%d index=%+v",
			sl.Coin(), sl.Account(), sl.Charge(), sl.AddressIndex())
	}

	// A failed parse changes nothing.
	keep, err := ParseDerivationPath(BIP44, "m/44'/60'/3'/1/9")
	if err != nil {
		t.Fatal(err)
	}
	if err := keep.ParsePath("m/44'/60'/4'/0/2147483648"); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Fatalf("ParsePath: got %v, want ErrInvalidDerivationPath", err)
	}
	if got := keep.String(); got != "m/44'/60'/3'/1/9" || keep.Account() != 3 {
		t.Errorf("failed ParsePath changed the path: %q account=%d", got, keep.Account())
	}
}

func mustAddress(t *testing.T, urn string) *Address {
	t.Helper()
	a, err := ParseURN(urn)
	if err != nil {
		t.Fatalf("ParseURN(%q): %v", urn, err)
	}
	return a.(*Address)
}

// TestDerivationTypeChangeDropsPath: a new derivation type invalidates the
// old path. Until a path is set the address names no path at all: its URN
// carries dt without dp (and so does not parse), and Validate / MarshalText
// refuse it, instead of serialising some other path under the new type.
func TestDerivationTypeChangeDropsPath(t *testing.T) {
	const bip44 = `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/3'/1/9`

	a := mustAddress(t, bip44)
	if err := a.SetDerivationType("zip32"); err != nil {
		t.Fatal(err)
	}
	if got := a.String(); got != `urn:mhda:nt:evm:ci:1:dt:zip32` {
		t.Errorf("after a type change: String() = %q, want urn:mhda:nt:evm:ci:1:dt:zip32", got)
	}
	if len(a.DerivationPath().Levels()) != 0 {
		t.Errorf("after a type change: levels %v survive", a.DerivationPath().Levels())
	}
	if err := a.Validate(); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("Validate without a path: got %v, want ErrInvalidDerivationPath", err)
	}
	if _, err := a.MarshalText(); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("MarshalText without a path: got %v, want ErrInvalidDerivationPath", err)
	}
	if _, err := ParseURN(a.String()); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("re-parse of %q: got %v, want ErrInvalidDerivationPath", a.String(), err)
	}
	if err := a.SetDerivationPath("m/32'/133'/7'"); err != nil {
		t.Fatal(err)
	}
	if got := a.String(); got != `urn:mhda:nt:evm:ci:1:dt:zip32:dp:m/32'/133'/7'` {
		t.Errorf("String() = %q", got)
	}

	// The same type, in any spelling, keeps the path.
	b := mustAddress(t, bip44)
	if err := b.SetDerivationType(" BIP44 "); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != bip44 {
		t.Errorf("same type: String() = %q, want %q", got, bip44)
	}
}

// TestSetDerivationIsAtomic: SetDerivation validates the type and the path
// before changing anything.
func TestSetDerivationIsAtomic(t *testing.T) {
	const bip44 = `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/3'/1/9`
	a := mustAddress(t, bip44)
	if err := a.SetDerivation("slip10", "garbage"); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("SetDerivation(slip10, garbage): got %v, want ErrInvalidDerivationPath", err)
	}
	if err := a.SetDerivation("bogus", "m/0'"); !errors.Is(err, ErrInvalidDerivationType) {
		t.Errorf("SetDerivation(bogus): got %v, want ErrInvalidDerivationType", err)
	}
	if got := a.String(); got != bip44 {
		t.Errorf("failed SetDerivation changed the address: %q", got)
	}
	if err := a.SetDerivation("slip10", "m/0'/1'"); err != nil {
		t.Fatal(err)
	}
	if got := a.String(); got != `urn:mhda:nt:evm:ci:1:dt:slip10:dp:m/0'/1'` {
		t.Errorf("String() = %q", got)
	}
	if a.DerivationPath().Account() != 0 {
		t.Errorf("SLIP-10 path kept the BIP-44 account %d", a.DerivationPath().Account())
	}
	if err := a.SetDerivation("", ""); err != nil {
		t.Fatal(err)
	}
	if got := a.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("root: String() = %q", got)
	}
}

// TestConstructorsMakeParseablePaths: a constructed path is exactly what
// parsing its String() gives back, so Levels() and the URN cannot disagree.
// Anything else panics with ErrInvalidDerivationPath.
func TestConstructorsMakeParseablePaths(t *testing.T) {
	for name, f := range map[string]func(){
		"bip44 purpose 49": func() {
			NewDerivationPathFromLevels(BIP44, []AddressIndex{{49, true}, {60, true}, {0, true}, {0, false}, {0, false}})
		},
		"bip44 soft account": func() {
			NewDerivationPathFromLevels(BIP44, []AddressIndex{{44, true}, {60, true}, {0, false}, {0, false}, {0, false}})
		},
		"bip44 two levels":       func() { NewDerivationPathFromLevels(BIP44, []AddressIndex{{44, true}, {60, true}}) },
		"slip10 no levels":       func() { NewDerivationPathFromLevels(SLIP10, nil) },
		"slip10 index 2^31":      func() { NewDerivationPathFromLevels(SLIP10, []AddressIndex{{1 << 31, false}}) },
		"root with a level":      func() { NewDerivationPathFromLevels(ROOT, []AddressIndex{{0, true}}) },
		"bip44 account 2^31":     func() { NewDerivationPath(BIP44, 60, 1<<31, 0, AddressIndex{}) },
		"bip44 charge 7":         func() { NewDerivationPath(BIP44, 60, 0, 7, AddressIndex{}) },
		"bip32 index 4294967295": func() { NewDerivationPath(BIP32, 0, 0, 0, AddressIndex{Index: 4294967295}) },
	} {
		if err := panicErr(t, f); !errors.Is(err, ErrInvalidDerivationPath) {
			t.Errorf("%s: panic %v, want ErrInvalidDerivationPath", name, err)
		}
	}

	dp := NewDerivationPathFromLevels(BIP44, []AddressIndex{{44, true}, {60, true}, {0, true}, {0, false}, {5, false}})
	if got := dp.String(); got != "m/44'/60'/0'/0/5" {
		t.Errorf("String() = %q", got)
	}
	parsed, err := ParseDerivationPath(BIP44, "m/44'/60'/0'/0/5")
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(dp, parsed) {
		t.Errorf("constructed path %+v differs from the parsed one %+v", *dp, *parsed)
	}
	// BIP-32 has no coin level: the constructor drops the coin like the parser.
	if c := NewDerivationPath(BIP32, 60, 0, 0, AddressIndex{}).Coin(); c != 0 {
		t.Errorf("BIP-32 path keeps coin %d", c)
	}
}

// TestLevelsReturnsACopy: the caller cannot change a path through Levels().
func TestLevelsReturnsACopy(t *testing.T) {
	for dt, p := range map[DerivationType]string{SLIP10: "m/44'/501'/0'/0'", BIP44: "m/44'/60'/0'/0/0"} {
		dp, err := ParseDerivationPath(dt, p)
		if err != nil {
			t.Fatal(err)
		}
		lv := dp.Levels()
		lv[2].Index = 9
		lv[2].IsHardened = false
		if got := dp.String(); got != p {
			t.Errorf("%s: String() = %q after changing a copy", dt, got)
		}
		if got := dp.Levels()[2]; got != (AddressIndex{Index: 0, IsHardened: true}) {
			t.Errorf("%s: Levels()[2] = %+v after changing a copy", dt, got)
		}
	}
}

// TestNewAddressCopiesChainAndPath: two addresses built from one chain and
// one path are independent of each other and of the originals.
func TestNewAddressCopiesChainAndPath(t *testing.T) {
	p, err := ParseDerivationPath(BIP44, "m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatal(err)
	}
	c := NewChain(EthereumVM, "1")
	x := NewAddress(c, p)
	y := NewAddress(c, p)
	if err := x.SetDerivationPath("m/44'/60'/0'/0/9"); err != nil {
		t.Fatal(err)
	}
	if err := x.SetCoinType("60"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetChainId("56"); err != nil {
		t.Fatal(err)
	}
	const want = `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`
	if got := y.String(); got != want {
		t.Errorf("y changed with x or c: %q, want %q", got, want)
	}
	if got := p.String(); got != "m/44'/60'/0'/0/0" {
		t.Errorf("the original path changed: %q", got)
	}
	if got := x.String(); got != `urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/9` {
		t.Errorf("x: %q", got)
	}
}

// samePath compares two paths through their public view.
func samePath(a, b *DerivationPath) bool {
	la, lb := a.Levels(), b.Levels()
	if len(la) != len(lb) {
		return false
	}
	for i := range la {
		if la[i] != lb[i] {
			return false
		}
	}
	return a.DerivationType() == b.DerivationType() && a.String() == b.String() &&
		a.Coin() == b.Coin() && a.Account() == b.Account() && a.Charge() == b.Charge() &&
		a.AddressIndex() == b.AddressIndex()
}
