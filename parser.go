package go_mhda

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	prefixMHDA          = `urn:mhda:`
	prefixOffset        = len(prefixMHDA)
	indexComponentIndex = 1
	indexComponentValue = 2

	// NSS components

	// Chain domain
	// compNetworkType is Network Type description, e.g. "evm", "tvm", "avm", "btc", "cosmos"
	compNetworkType = `nt`
	// compCoinType is Coin Type description, according SLIP-44 list
	// (https://github.com/satoshilabs/slips/blob/master/slip-0044.md), e.g. "0", "60", "195", "118".
	compCoinType = `ct`
	// compChainId is Network Id (Chain Id) description, e.g. for evm hex: "0x1", "0x10",
	// for Cosmos - string "axelar", etc.
	compChainId = `ci`

	// Derivation path domain
	compDerivationType = `dt`
	compDerivationPath = `dp`

	// Address format domain
	compAddressAlgorithm = `aa`
	compAddressFormat    = `af`
	compAddressPrefix    = `ap`
	compAddressSuffix    = `as`
)

var (
	// componentsNames is the canonical emission order for NSS:
	//   nt:ct:ci:dt:dp:aa:af:ap:as
	// chain-domain (nt/ct/ci) first, then derivation, then address-format
	// metadata.
	componentsNames = []string{
		compNetworkType,
		compCoinType,
		compChainId,
		compDerivationType,
		compDerivationPath,
		compAddressAlgorithm,
		compAddressFormat,
		compAddressPrefix,
		compAddressSuffix,
	}

	rxComponent = regexp.MustCompile(`:(nt|ct|ci|dt|dp|aa|af|ap|as):([0-9a-z-._~*+=%$&@?'()!,;/#]+)`)
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
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
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

// ParseURNRx parses a URN MHDA via a single regex pass. Behaviour matches
// ParseURN; the regex variant exists primarily for benchmarking.
func ParseURNRx(src string) (MHDA, error) {
	src = strings.TrimSpace(src)
	if !hasPrefixFold(src, prefixMHDA) {
		return nil, ErrInvalidURN
	}

	nss := stripRQF(src[prefixOffset:])
	// Re-prepend a leading colon so the regex's `:(component):` pattern
	// can match the very first component.
	submatches := rxComponent.FindAllStringSubmatch(":"+nss, len(componentsNames))

	if len(submatches) == 0 {
		return nil, fmt.Errorf("%w: no components", ErrInvalidNSS)
	}

	components := map[string]string{}
	for i := range submatches {
		if len(submatches[i]) != 3 {
			continue
		}
		components[submatches[i][indexComponentIndex]] = submatches[i][indexComponentValue]
	}

	return parseAddress(components)
}

// ParseURN is the lenient parsing entry point. RFC 8141 §5.1 case-insensitive
// prefix and §2 rq/f-components are accepted; surrounding whitespace is
// trimmed.
func ParseURN(src string) (MHDA, error) {
	src = strings.TrimSpace(src)
	if !hasPrefixFold(src, prefixMHDA) {
		return nil, ErrInvalidURN
	}
	return ParseNSS(stripRQF(src[prefixOffset:]))
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
// Requires the network type ("nt") component to be present.
func ParseNSS(nss string) (MHDA, error) {
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
// Form: a sequence of `key:value` pairs joined by `:` separators. Unknown
// keys are silently skipped (forward-compat with future URN extensions);
// duplicate keys and empty values are rejected.
//
// Values may not contain ':'; this holds for every component currently
// defined in MHDA. Adding a value type that needs ':' would require
// percent-encoding support.
func parseNSS(nss string) (map[string]string, error) {
	parts := strings.Split(nss, ":")
	components := make(map[string]string, len(componentsNames))

	for i := 0; i < len(parts); {
		key := parts[i]
		if _, ok := knownComponents[key]; !ok {
			// Unknown token (could be an unrelated word, a future component
			// name, or part of a value we mis-identified). Skip and move on.
			i++
			continue
		}
		if i+1 >= len(parts) {
			return nil, fmt.Errorf("%w: missing value for %q", ErrInvalidNSS, key)
		}
		// RFC 8141 NSS does not permit unescaped whitespace; trim it so any
		// trailing space (e.g. from "ci:0 #frag" where stripRQF leaves the
		// space) does not leak into the canonical form and break round-trip.
		value := strings.TrimSpace(parts[i+1])
		if value == "" {
			return nil, fmt.Errorf("%w: empty value for %q", ErrInvalidNSS, key)
		}
		if _, dup := components[key]; dup {
			return nil, fmt.Errorf("%w: duplicate component %q", ErrInvalidNSS, key)
		}
		components[key] = value
		i += 2
	}
	return components, nil
}
