package go_mhda

import (
	"strings"
	"testing"
)

// FuzzParseURN exercises ParseURN against arbitrary inputs. The contract is:
//   - parsing must never panic (any input is allowed)
//   - if parsing succeeds, the result must serialize and be idempotent on
//     re-parse (Parse(s).String() round-trips through itself)
func FuzzParseURN(f *testing.F) {
	for _, s := range uriMHDA {
		f.Add(s)
	}
	// extra seeds: degenerate inputs that have historically caused trouble
	// in the byte-by-byte parser of ParseNSS.
	for _, s := range []string{
		"",
		"urn:mhda:",
		"urn:mhda:n",
		"urn:mhda:nt",
		"urn:mhda:nt:",
		"urn:mhda:nt:evm",
		"urn:mhda:nt:evm:",
		"urn:mhda:nt:evm:ct",
		"urn:mhda:nt:evm:ct:",
		"urn:mhda:nt:evm:ct:60:ci",
		"urn:mhda:nt:evm:ct:60:ci:",
		"urn:mhda:nt:evm:ci:1:ct:60:xx:y",
		"urn:mhda:::::::",
		"urn:mhda:nt:evm:ci:1:ct:60:aa:",
		"urn:mhda:nt:evm:ci:1:ct:60:af:",
		"urn:mhda:nt:evm:ci:1:wt:web3:wi:5f2a8c31",
		"urn:mhda:nt:evm:ci:1:wt:",
		"urn:mhda:nt:evm:ci:1:wi:",
		"urn:mhda:wt:web3:nt:evm:ci:1",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, src string) {
		addr, err := ParseURN(src)
		if err != nil {
			return
		}
		if addr == nil {
			t.Fatalf("ParseURN returned (nil, nil) for %q", src)
		}
		serialized := addr.String()
		if !strings.HasPrefix(serialized, prefixMHDA) {
			t.Fatalf("serialized form lacks urn:mhda: prefix: %q (input: %q)", serialized, src)
		}
		again, err := ParseURN(serialized)
		if err != nil {
			t.Fatalf("re-parse failed for %q (serialized from %q): %v", serialized, src, err)
		}
		if again.String() != serialized {
			t.Fatalf("not idempotent:\n once:  %s\n twice: %s\n input: %q", serialized, again.String(), src)
		}
	})
}

// FuzzDerivationPath exercises the public ParseDerivationPath API with pairs
// of (derivation-type, path). Contract:
//   - parsing must never panic
//   - on success, String() must be re-parseable to an equivalent path
//   - the result must be idempotent: Parse(s).String() == Parse(Parse(s).String()).String()
func FuzzDerivationPath(f *testing.F) {
	// Seeds: one valid path per known derivation type plus a few degenerate
	// inputs that have historically been weak spots (variable-length ZIP-32,
	// SLIP-10 with mixed hardening, hardened leaf, the largest index, etc).
	seeds := []struct{ dt, path string }{
		{"bip32", "m/0'/0/0"},
		{"bip32", "m/2147483647'/1/2147483647'"}, // largest index, 2^31-1
		{"bip44", "m/44'/60'/0'/0/0"},
		{"bip44", "m/44'/0'/0'/0/0'"}, // hardened leaf
		{"bip49", "m/49'/0'/0'/0/0"},
		{"bip54", "m/54'/784'/0'/0/0"},
		{"bip74", "m/74'/784'/0'/0/0"},
		{"bip84", "m/84'/0'/0'/0/0"},
		{"bip86", "m/86'/0'/0'/0/0"},
		{"slip10", "m/44'/501'/0'/0'"},
		{"slip10", "m/44'/148'/0'"},
		{"slip10", "m/0"}, // shortest valid form
		{"cip11", "m/44'/118'/0'/0/0"},
		{"cip1852", "m/1852'/1815'/0'/0/0"},
		{"cip1852", "m/1852'/1815'/0'/2/0"}, // staking key role
		{"zip32", "m/32'/133'/0'"},          // 3-level form
		{"zip32", "m/32'/133'/0'/0"},        // 4-level form
		{"zip32", "m/32'/133'/0'/0'"},       // 4-level form, hardened leaf
		{"root", ""},
		// degenerate / invalid inputs - parser must reject without panic
		{"bip44", ""},
		{"bip44", "m/"},
		{"bip44", "m/44'/0'/0'/0"}, // missing leaf
		{"slip10", "m"},
		{"slip10", "m/"},
		{"slip10", "m/99999999999999999999"},   // overflow
		{"bip32", "m/2147483648'/1/0"},         // index 2^31
		{"bip44", "m/44'/60'/0'/0/4294967295"}, // index 2^32-1
		{"unknown", "m/0/0/0"},
	}
	for _, s := range seeds {
		f.Add(s.dt, s.path)
	}

	f.Fuzz(func(t *testing.T, dt, path string) {
		dp, err := ParseDerivationPath(DerivationType(dt), path)
		if err != nil {
			return
		}
		if dp == nil {
			t.Fatalf("ParseDerivationPath returned (nil, nil) for (%q, %q)", dt, path)
		}

		// String() should at least produce something that can be re-parsed.
		serialized := dp.String()
		dp2, err := ParseDerivationPath(dp.derivationType, serialized)
		if err != nil {
			t.Fatalf("re-parse of %q (from input dt=%q path=%q) failed: %v",
				serialized, dt, path, err)
		}
		if dp2.String() != serialized {
			t.Fatalf("not idempotent:\n once:  %s\n twice: %s\n input: dt=%q path=%q",
				serialized, dp2.String(), dt, path)
		}

		// No accepted level may carry an index of 2^31 or more.
		for i, lvl := range dp.Levels() {
			if lvl.Index > 1<<31-1 {
				t.Fatalf("level[%d] = %d is above 2^31-1 (input: dt=%q path=%q)", i, lvl.Index, dt, path)
			}
		}

		// Levels() must always be safe to call (may be empty for ROOT).
		_ = dp.Levels()
	})
}

// FuzzParseNSS targets the lower-level byte parser. Same contract as
// FuzzParseURN: no panics, and on success the emitted URN re-parses to
// itself.
func FuzzParseNSS(f *testing.F) {
	for _, s := range uriMHDA {
		f.Add(s[prefixOffset:])
	}
	for _, s := range []string{
		"", "n", "nt", "nt:", "nt:evm", "nt:evm:ci:1:ct:60", "nt:evm:ci:1",
		"nt:evm:ci:1:wt:web3:wi:5f2a8c31",
		"nt:evm:ci:1?=q:dt:bip44:dp:m/44'/60'/0'/0/0",
		"nt:evm:ci:1:wi:a#b",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, src string) {
		addr, err := ParseNSS(src)
		if err != nil {
			return
		}
		if addr == nil {
			t.Fatalf("ParseNSS returned (nil, nil) for %q", src)
		}
		serialized := addr.String()
		if serialized != prefixMHDA+addr.NSS() {
			t.Fatalf("String() %q is not the prefix plus NSS() %q", serialized, addr.NSS())
		}
		again, err := ParseURN(serialized)
		if err != nil {
			t.Fatalf("re-parse of %q (from NSS %q) failed: %v", serialized, src, err)
		}
		if again.String() != serialized {
			t.Fatalf("not idempotent:\n once:  %s\n twice: %s\n input: %q", serialized, again.String(), src)
		}
	})
}
