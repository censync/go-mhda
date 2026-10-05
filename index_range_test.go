package go_mhda

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"testing"
)

// testMaxLevelIndex is the largest index of a derivation-path level: a BIP-32
// child number keeps the hardened flag in its top bit, so the index has 31
// bits.
const testMaxLevelIndex = 1<<31 - 1

// TestLevelIndexAliasRefused: an index of 2^31 or more collides with the
// hardened bit of a BIP-32 child number. m/44'/60'/2147483648'/0/0 would
// derive the key of m/44'/60'/0'/0/0, and an unhardened 2147483648 would ask
// for a hardened child through the public-key formula.
func TestLevelIndexAliasRefused(t *testing.T) {
	for _, urn := range []string{
		`urn:mhda:nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/2147483648'/0/0`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/2147483648'/0/0`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/2147483648`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/4294967295`,
		`urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/2147483708'/0'/0/0`,
		`urn:mhda:nt:solana:ci:mainnet:dt:slip10:dp:m/44'/501'/2147483648'/0'`,
	} {
		if _, err := ParseURN(urn); !errors.Is(err, ErrInvalidDerivationPath) {
			t.Errorf("ParseURN(%q): got %v, want ErrInvalidDerivationPath", urn, err)
		}
		if _, err := ParseURNStrict(urn); !errors.Is(err, ErrInvalidDerivationPath) {
			t.Errorf("ParseURNStrict(%q): got %v, want ErrInvalidDerivationPath", urn, err)
		}
	}

	const largest = `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/2147483647'/0/2147483647'`
	addr, err := ParseURNStrict(largest)
	if err != nil {
		t.Fatalf("ParseURNStrict(%q): %v", largest, err)
	}
	if addr.String() != largest {
		t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), largest)
	}
	if got := addr.DerivationPath().Account(); got != testMaxLevelIndex {
		t.Errorf("Account() = %d, want %d", got, testMaxLevelIndex)
	}
}

// TestCoinTypeIsNotALevel: the ct component is SLIP-44 metadata, not a path
// level, and keeps the full 32-bit range.
func TestCoinTypeIsNotALevel(t *testing.T) {
	for urn, want := range map[string]CoinType{
		`urn:mhda:nt:evm:ci:1:ct:2147483648:dt:bip44:dp:m/44'/60'/0'/0/0`: 2147483648,
		`urn:mhda:nt:evm:ci:1:ct:4294967295:dt:bip44:dp:m/44'/60'/0'/0/0`: 4294967295,
	} {
		addr, err := ParseURN(urn)
		if err != nil {
			t.Errorf("ParseURN(%q): %v", urn, err)
			continue
		}
		if got := addr.Chain().CoinType(); got != want {
			t.Errorf("ParseURN(%q): ct = %d, want %d", urn, got, want)
		}
		if addr.String() != urn {
			t.Errorf("round-trip:\n got:  %s\n want: %s", addr.String(), urn)
		}
	}
}

// TestDerivationPathConformance runs the derivation-path table shared with
// the C++ port (testdata/dp_conformance.txt, a verbatim copy of the port's
// tests/data/dp_conformance.txt). Every row is parsed standalone and inside a
// URN; both must agree with the row.
func TestDerivationPathConformance(t *testing.T) {
	f, err := os.Open("testdata/dp_conformance.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rows := 0
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		accept := fields[0] == "accept"
		if !(accept && len(fields) == 4) && !(fields[0] == "refuse" && len(fields) == 3) {
			t.Fatalf("dp_conformance.txt:%d: malformed row %q", line, text)
		}
		rows++
		dt, path := DerivationType(fields[1]), fields[2]

		dp, err := ParseDerivationPath(dt, path)
		if accept {
			if err != nil {
				t.Errorf("line %d: ParseDerivationPath(%s, %q): %v", line, dt, path, err)
			} else {
				if got := dp.String(); got != fields[3] {
					t.Errorf("line %d: ParseDerivationPath(%s, %q).String() = %q, want %q", line, dt, path, got, fields[3])
				}
				for i, lvl := range dp.Levels() {
					if lvl.Index > testMaxLevelIndex {
						t.Errorf("line %d: level[%d] = %d is above 2^31-1", line, i, lvl.Index)
					}
				}
			}
		} else if !errors.Is(err, ErrInvalidDerivationPath) {
			t.Errorf("line %d: ParseDerivationPath(%s, %q): got %v, want ErrInvalidDerivationPath", line, dt, path, err)
		}

		urn := `urn:mhda:nt:evm:ci:1:dt:` + string(dt) + `:dp:` + path
		addr, err := ParseURN(urn)
		if accept {
			want := `urn:mhda:nt:evm:ci:1:dt:` + string(dt) + `:dp:` + fields[3]
			if err != nil {
				t.Errorf("line %d: ParseURN(%q): %v", line, urn, err)
			} else if addr.String() != want {
				t.Errorf("line %d: ParseURN(%q).String() = %q, want %q", line, urn, addr.String(), want)
			}
		} else if !errors.Is(err, ErrInvalidDerivationPath) {
			t.Errorf("line %d: ParseURN(%q): got %v, want ErrInvalidDerivationPath", line, urn, err)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if rows < 300 {
		t.Fatalf("dp_conformance.txt: %d rows, the table is truncated", rows)
	}
}
