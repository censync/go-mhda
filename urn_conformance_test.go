package go_mhda

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"testing"
)

// conformanceErrors maps the error names of the shared URN table (the C++
// error_code names) to the sentinels.
var conformanceErrors = map[string]error{
	"invalid_urn":             ErrInvalidURN,
	"invalid_nss":             ErrInvalidNSS,
	"missing_network_type":    ErrMissingNetworkType,
	"invalid_network_type":    ErrInvalidNetworkType,
	"invalid_coin_type":       ErrInvalidCoinType,
	"missing_chain_id":        ErrMissingChainID,
	"invalid_derivation_type": ErrInvalidDerivationType,
	"invalid_derivation_path": ErrInvalidDerivationPath,
	"invalid_algorithm":       ErrInvalidAlgorithm,
	"invalid_format":          ErrInvalidFormat,
	"invalid_value":           ErrInvalidValue,
	"incompatible":            ErrIncompatible,
}

// TestURNConformance runs the URN table shared with the C++ port
// (testdata/urn_conformance.txt, a verbatim copy of the port's
// tests/data/urn_conformance.txt).
func TestURNConformance(t *testing.T) {
	f, err := os.Open("testdata/urn_conformance.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rows := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 4 || (fields[1] != "urn" && fields[1] != "nss" && fields[1] != "strict") {
			t.Fatalf("urn_conformance.txt:%d: malformed row %q", line, text)
		}
		rows++
		accept := fields[0] == "accept"
		input, want := fields[2], fields[3]
		var wantErr error
		switch {
		case accept:
		case fields[0] == "refuse":
			input, want = fields[3], ""
			if wantErr = conformanceErrors[fields[2]]; wantErr == nil {
				t.Fatalf("urn_conformance.txt:%d: unknown error %q", line, fields[2])
			}
		default:
			t.Fatalf("urn_conformance.txt:%d: malformed row %q", line, text)
		}

		parse := ParseURN
		switch fields[1] {
		case "nss":
			parse = ParseNSS
		case "strict":
			parse = ParseURNStrict
		}
		addr, err := parse(input)
		if accept {
			if err != nil {
				t.Errorf("line %d: parse %s %q: %v", line, fields[1], input, err)
			} else if got := addr.String(); got != want {
				t.Errorf("line %d: parse %s %q: String() = %q, want %q", line, fields[1], input, got, want)
			}
		} else if !errors.Is(err, wantErr) {
			t.Errorf("line %d: parse %s %q: got %v, want %v", line, fields[1], input, err, wantErr)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if rows == 0 {
		t.Fatal("urn_conformance.txt: no rows")
	}
}

// TestSetDerivationPathUnderRoot: the setter refuses a path while the type
// is root, and accepts an empty one.
func TestSetDerivationPathUnderRoot(t *testing.T) {
	parsed, err := ParseURN(`urn:mhda:nt:evm:ci:1`)
	if err != nil {
		t.Fatal(err)
	}
	addr := parsed.(*Address)
	if err := addr.SetDerivationPath("m/44'/60'/0'/0/0"); !errors.Is(err, ErrInvalidDerivationPath) {
		t.Errorf("SetDerivationPath under root: got %v, want ErrInvalidDerivationPath", err)
	}
	for _, empty := range []string{"", "  "} {
		if err := addr.SetDerivationPath(empty); err != nil {
			t.Errorf("SetDerivationPath(%q) under root: %v", empty, err)
		}
	}
	if got := addr.String(); got != `urn:mhda:nt:evm:ci:1` {
		t.Errorf("String() = %q", got)
	}
}
