package go_mhda

import (
	"encoding/json"
	"encoding/xml"
	"testing"
)

// TestTextCodecsWithAddressValues: an Address held by value (a struct field,
// a map value) encodes as its URN, not as an empty object, and decodes back.
func TestTextCodecsWithAddressValues(t *testing.T) {
	const urn = `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`
	parsed, err := ParseURN(urn)
	if err != nil {
		t.Fatal(err)
	}
	addr := *parsed.(*Address)

	type record struct {
		A Address
	}
	raw, err := json.Marshal(record{A: addr})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"A":"` + urn + `"}`; string(raw) != want {
		t.Errorf("json value field: %s, want %s", raw, want)
	}
	var back record
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.A.String() != urn {
		t.Errorf("json round-trip: %q", back.A.String())
	}

	raw, err = json.Marshal(map[string]Address{"k": addr})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"k":"` + urn + `"}`; string(raw) != want {
		t.Errorf("json map value: %s, want %s", raw, want)
	}

	raw, err = xml.Marshal(record{A: addr})
	if err != nil {
		t.Fatal(err)
	}
	var backXML record
	if err := xml.Unmarshal(raw, &backXML); err != nil {
		t.Fatalf("xml %s: %v", raw, err)
	}
	if backXML.A.String() != urn {
		t.Errorf("xml value field: %s decodes to %q", raw, backXML.A.String())
	}

	// A nil pointer is encoded by the codec itself, as null.
	raw, err = json.Marshal(struct{ P *Address }{})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"P":null}` {
		t.Errorf("json nil pointer: %s", raw)
	}
}
