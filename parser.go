package go_mhda

import (
	"fmt"
	"strings"
)

const (
	prefixMHDA   = `urn:mhda:`
	prefixOffset = len(prefixMHDA)

	// NSS components

	// Chain domain
	// compNetworkType is Network Type description, e.g. "evm", "tron",
	// "avalanche", "bitcoin", "cosmos"
	compNetworkType = `nt`
	// compChainId is Network Id (Chain Id) description, e.g. for evm hex: "0x1", "0x10",
	// for Cosmos - string "axelar", etc.
	compChainId = `ci`
	// compCoinType is the OPTIONAL Coin Type metadata, according SLIP-44 list
	// (https://github.com/satoshilabs/slips/blob/master/slip-0044.md), e.g.
	// "0", "60", "195", "118". Not part of the chain identity.
	compCoinType = `ct`

	// Derivation path domain
	compDerivationType = `dt`
	compDerivationPath = `dp`

	// Address format domain
	compAddressAlgorithm = `aa`
	compAddressFormat    = `af`
	compAddressPrefix    = `ap`
	compAddressSuffix    = `as`

	// Wallet domain (optional)
	// compWalletType is a free-form wallet/client type, e.g. "web3",
	// "metamask", "tonconnect".
	compWalletType = `wt`
	// compWalletId is a free-form wallet instance identifier, e.g. a UUID or
	// an HD root key fingerprint.
	compWalletId = `wi`
)

var (
	// componentsNames is the canonical emission order for NSS:
	//   nt:ci:ct:dt:dp:aa:af:ap:as:wt:wi
	// chain identity (nt/ci) first — so a chain key is always a strict prefix
	// of the NSS — then the optional coin-type metadata, then derivation,
	// address-format metadata, and the wallet domain last.
	componentsNames = []string{
		compNetworkType,
		compChainId,
		compCoinType,
		compDerivationType,
		compDerivationPath,
		compAddressAlgorithm,
		compAddressFormat,
		compAddressPrefix,
		compAddressSuffix,
		compWalletType,
		compWalletId,
	}
)

// knownComponents is the lookup set for the split-based parser. Built from
// componentsNames at init time.
var knownComponents = func() map[string]struct{} {
	m := make(map[string]struct{}, len(componentsNames))
	for _, n := range componentsNames {
		m[n] = struct{}{}
	}
	return m
}()

// hasPrefixFold reports whether s begins with prefix, ignoring ASCII case.
// RFC 8141 §5.1: the leading "urn:" sequence and the NID are case-insensitive
// (e.g. "URN:MHDA:..." must parse identically to "urn:mhda:...").
func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && asciiLower(s[:len(prefix)]) == asciiLower(prefix)
}

// asciiTrim trims ASCII whitespace only. Unicode spaces (NBSP, ideographic
// space, …) are NOT trimmed: an NSS is ASCII by definition (RFC 8141), so a
// Unicode space is not decoration to strip — it stays in place and the value
// check rejects it loudly. Trimming it instead (as a Unicode-aware trim
// would) silently accepts a malformed URN and diverges from the C++ port.
func asciiTrim(s string) string {
	return strings.Trim(s, asciiSpace)
}

// asciiSpace is the ASCII whitespace asciiTrim removes.
const asciiSpace = " \t\n\v\f\r"

// nssByte reports whether c may appear in an NSS key or value (SPEC §1.5):
// an RFC 3986 pchar or "/" - letters, digits and -._~!$&'()*+,;=@/ - except
// ':' (the component separator) and '%' (percent-encoding is not supported,
// and a raw "%41" would be a second spelling of "A"). Whitespace, control
// bytes, non-ASCII bytes and the printable ASCII outside pchar ('"', '<',
// '>', '\', '^', '`', '{', '|', '}', '[', ']', '?', '#') cannot appear in a
// conforming URN.
func nssByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("-._~!$&'()*+,;=@/", c) >= 0
}

// validateValueCharset enforces SPEC §1.5 on an NSS value (see nssByte).
func validateValueCharset(key, value string) error {
	for i := 0; i < len(value); i++ {
		if !nssByte(value[i]) {
			return fmt.Errorf("%w: byte %q not allowed in value for %q",
				ErrInvalidNSS, value[i], key)
		}
	}
	return nil
}

// validateKeyCharset requires a component key to be non-empty and made of
// the same bytes as values (see nssByte).
func validateKeyCharset(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty component key", ErrInvalidNSS)
	}
	for i := 0; i < len(key); i++ {
		if !nssByte(key[i]) {
			return fmt.Errorf("%w: byte %q not allowed in component key %q", ErrInvalidNSS, key[i], key)
		}
	}
	return nil
}

// stripRQF strips the optional rq-components ("?+" / "?=") and f-component
// ("#") trailing the assigned-name part, per RFC 8141 §2. The current parser
// does not interpret resource/query/fragment metadata; they are silently
// discarded.
func stripRQF(nss string) string {
	if i := strings.IndexAny(nss, "?#"); i >= 0 {
		return nss[:i]
	}
	return nss
}

// ParseURN is the lenient parsing entry point. RFC 8141 §5.1 case-insensitive
// prefix and §2 rq/f-components are accepted; surrounding whitespace is
// trimmed.
func ParseURN(src string) (MHDA, error) {
	src = asciiTrim(src)
	if !hasPrefixFold(src, prefixMHDA) {
		return nil, ErrInvalidURN
	}
	// Whitespace left before a stripped r/q/f component ("ci:0 #frag") ends
	// the URN like the whitespace trimmed above; any other whitespace in the
	// NSS is malformed and parseNSS refuses it.
	return parseAddressNSS(strings.TrimRight(stripRQF(src[prefixOffset:]), asciiSpace))
}

// ParseURNStrict is ParseURN + Validate(). It rejects URNs whose
// (networkType, algorithm, format, derivation) combination is not in the
// known-good compatibility matrix.
func ParseURNStrict(src string) (MHDA, error) {
	addr, err := ParseURN(src)
	if err != nil {
		return nil, err
	}
	if v, ok := addr.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}
	return addr, nil
}

// ParseNSS parses an MHDA namespace-specific string into an MHDA address.
// Requires the network type ("nt") component to be present. Surrounding ASCII
// whitespace is trimmed; whitespace inside the NSS is refused.
func ParseNSS(nss string) (MHDA, error) {
	return parseAddressNSS(asciiTrim(nss))
}

// parseAddressNSS parses an NSS without surrounding whitespace into an
// address.
func parseAddressNSS(nss string) (MHDA, error) {
	components, err := parseNSS(nss)
	if err != nil {
		return nil, err
	}
	if _, ok := components[compNetworkType]; !ok {
		return nil, ErrMissingNetworkType
	}
	return parseAddress(components)
}

// parseNSS is the shared low-level NSS parser. It returns the raw component
// map; callers (ParseNSS, ChainFromNSS) interpret the map per their domain.
//
// Form: a sequence of `key:value` pairs joined by `:` separators; an empty
// NSS has no components. A pair with an unknown key is skipped together with
// its value (forward-compat with future URN extensions), so a value is never
// read as a key and a following known key is never read as a value. A key
// that differs from a known key only by case is rejected, not skipped: it
// would drop the component silently. A key without a value, an empty key or
// value and a duplicate known key are rejected too. Nothing is trimmed: the
// caller removes whitespace around the whole NSS, and whitespace around a key
// or value is malformed input.
//
// '?' and '#' open the RFC 8141 r/q/f components. ParseURN strips them before
// the NSS reaches this parser; an NSS that still carries one (ParseNSS,
// ChainFromNSS, ChainFromKey) is rejected, since the URN emitted from it would
// be truncated at that byte on the next parse.
//
// Values may not contain ':'; this holds for every component currently
// defined in MHDA. Adding a value type that needs ':' would require
// percent-encoding support.
func parseNSS(nss string) (map[string]string, error) {
	components := make(map[string]string, len(componentsNames))
	if nss == "" {
		return components, nil
	}
	if i := strings.IndexAny(nss, "?#"); i >= 0 {
		return nil, fmt.Errorf("%w: %q inside the NSS", ErrInvalidNSS, nss[i])
	}
	parts := strings.Split(nss, ":")
	if len(parts)%2 != 0 {
		return nil, fmt.Errorf("%w: missing value for %q", ErrInvalidNSS, parts[len(parts)-1])
	}

	for i := 0; i < len(parts); i += 2 {
		key := parts[i]
		if err := validateKeyCharset(key); err != nil {
			return nil, err
		}
		value := parts[i+1]
		if value == "" {
			return nil, fmt.Errorf("%w: empty value for %q", ErrInvalidNSS, key)
		}
		// A value is printable ASCII: whitespace, control bytes and Unicode
		// spaces are malformed input, never silently normalised.
		if err := validateValueCharset(key, value); err != nil {
			return nil, err
		}
		if _, ok := knownComponents[key]; !ok {
			if _, ok := knownComponents[asciiLower(key)]; ok {
				return nil, fmt.Errorf("%w: component key %q must be lowercase", ErrInvalidNSS, key)
			}
			continue // unknown component, skipped with its value
		}
		if _, dup := components[key]; dup {
			return nil, fmt.Errorf("%w: duplicate component %q", ErrInvalidNSS, key)
		}
		components[key] = value
	}
	return components, nil
}
