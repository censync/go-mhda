package go_mhda

import (
	"fmt"
	"strconv"
	"strings"
)

type ChainId string

// ChainKey - string identifier for declaration any chain or subchain
type ChainKey string

// Chain describes a network: the network type plus the chain id. The pair
// (nt, ci) is the chain identity; Key() and String() serialize exactly that
// pair, and chain keys compare as plain strings.
//
// An optional SLIP-44 coin type may be attached as metadata (SetCoinType).
// It is carried by the full URN form (the "ct" component) but is NOT part of
// the chain identity: it never appears in Key()/String().
type Chain struct {
	networkType NetworkType
	chainId     ChainId
	coinType    CoinType
	hasCoinType bool
}

// NewChain builds a chain from a registered network type and a chain id.
// Both are written verbatim into every URN and chain key, so they are
// validated like parsed input: the network type must be one of the
// registered constants, and the chain id (ASCII-trimmed) must be non-empty
// printable ASCII without ':', '?' or '#' - a ':' would inject components on
// re-parse. An invalid value panics (programmer error at a construction
// site, as with NewAddress). For untrusted input use SetNetworkType /
// SetChainId, which return the error, or ChainFromKey.
func NewChain(networkType NetworkType, chainId ChainId) *Chain {
	c := &Chain{}
	if err := c.SetNetworkType(networkType); err != nil {
		panic(err)
	}
	if err := c.SetChainId(chainId); err != nil {
		panic(err)
	}
	return c
}

// ChainFromKey parses a Chain from a key produced by Chain.Key(). A chain key
// is the canonical identity form "nt:<network>:ci:<chain_id>" and nothing
// else. "ct" (the pre-1.1 key format) is rejected with the dedicated
// ErrCoinTypeInChainKey so that legacy keys fail loudly instead of being
// silently reinterpreted; any other known component and any residue (unknown
// tokens, reordering, non-canonical spelling) is rejected with
// ErrInvalidChainKey — keys compare as plain strings, so every accepted
// input must BE the canonical string.
func ChainFromKey(chainKey ChainKey) (*Chain, error) {
	trimmed := asciiTrim(string(chainKey))
	// A dangling token is residue like any other: the input is not the
	// canonical key, whatever else it carries.
	if trimmed != "" && strings.Count(trimmed, ":")%2 == 0 {
		return nil, fmt.Errorf("%w: not a sequence of key:value pairs: %q", ErrInvalidChainKey, trimmed)
	}
	components, err := parseNSS(trimmed)
	if err != nil {
		return nil, err
	}
	if _, ok := components[compCoinType]; ok {
		return nil, ErrCoinTypeInChainKey
	}
	for key := range components {
		if key != compNetworkType && key != compChainId {
			return nil, fmt.Errorf("%w: unexpected component %q", ErrInvalidChainKey, key)
		}
	}
	if _, ok := components[compNetworkType]; !ok {
		return nil, ErrMissingNetworkType
	}
	chain, err := parseChain(components)
	if err != nil {
		return nil, err
	}
	if chain.String() != trimmed {
		return nil, fmt.Errorf("%w: not in canonical form: %q", ErrInvalidChainKey, trimmed)
	}
	return chain, nil
}

// ChainFromNSS parses the chain-domain components ("nt", "ci" and the
// optional "ct" metadata) from the given NSS string. Other components are
// tolerated and ignored, so a full address NSS is valid input.
func ChainFromNSS(src string) (*Chain, error) {
	components, err := parseNSS(src)
	if err != nil {
		return nil, err
	}
	if _, ok := components[compNetworkType]; !ok {
		return nil, ErrMissingNetworkType
	}
	return parseChain(components)
}

// parseChain extracts and validates the chain-domain components from a
// component map produced by the NSS parser. Shared by parseAddress,
// ChainFromNSS and ChainFromKey.
func parseChain(m map[string]string) (*Chain, error) {
	networkType := normalize(m[compNetworkType])
	if networkType == `` {
		return nil, ErrMissingNetworkType
	}
	if !NetworkType(networkType).IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidNetworkType, networkType)
	}

	chainID, ok := m[compChainId]
	if !ok || strings.TrimSpace(chainID) == "" {
		return nil, ErrMissingChainID
	}

	chain := &Chain{
		networkType: NetworkType(networkType),
		chainId:     ChainId(chainID),
	}

	if ct, ok := m[compCoinType]; ok {
		coinType, err := parseCoinType(strings.TrimSpace(ct))
		if err != nil {
			return nil, err
		}
		chain.coinType = coinType
		chain.hasCoinType = true
	}

	return chain, nil
}

// parseCoinType parses a SLIP-44 value from its two documented spellings:
// plain decimal or 0x-prefixed hex. Go integer-literal extras (0o/0b
// prefixes, digit-group underscores) are deliberately rejected — allowing
// several spellings of one value would defeat duplicate detection and the
// canonical-form guarantees.
func parseCoinType(s string) (CoinType, error) {
	var v uint64
	var err error
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		v, err = strconv.ParseUint(s[2:], 16, 32)
	} else {
		v, err = strconv.ParseUint(s, 10, 32)
	}
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidCoinType, s)
	}
	return CoinType(v), nil
}

// SetNetworkType sets the network type. It must be one of the registered
// constants (ErrInvalidNetworkType); on error the chain is left unchanged.
func (c *Chain) SetNetworkType(networkType NetworkType) error {
	if !networkType.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidNetworkType, networkType)
	}
	c.networkType = networkType
	return nil
}

// SetChainId sets the chain id. The value is ASCII-trimmed and must be
// non-empty (ErrMissingChainID) printable ASCII without ':', '?' or '#'
// (ErrInvalidValue, see validateFreeFormValue); on error the chain is left
// unchanged.
func (c *Chain) SetChainId(chainId ChainId) error {
	id := asciiTrim(string(chainId))
	if id == "" {
		return ErrMissingChainID
	}
	if err := validateFreeFormValue(compChainId, id); err != nil {
		return err
	}
	c.chainId = ChainId(id)
	return nil
}

// SetCoinType attaches the optional SLIP-44 coin-type metadata.
func (c *Chain) SetCoinType(coinType CoinType) {
	c.coinType = coinType
	c.hasCoinType = true
}

// ClearCoinType removes the optional SLIP-44 coin-type metadata.
func (c *Chain) ClearCoinType() {
	c.coinType = 0
	c.hasCoinType = false
}

func (c *Chain) NetworkType() NetworkType { return c.networkType }
func (c *Chain) ChainId() ChainId         { return c.chainId }

// CoinType returns the optional SLIP-44 coin-type metadata, or 0 when unset.
// Use HasCoinType to distinguish an explicit 0 (Bitcoin) from "not set".
func (c *Chain) CoinType() CoinType { return c.coinType }

// HasCoinType reports whether the optional coin-type metadata is set.
func (c *Chain) HasCoinType() bool { return c.hasCoinType }

func (c *Chain) Key() ChainKey {
	return ChainKey(c.String())
}

// String returns the canonical NSS-style chain key:
//
//	nt:<network>:ci:<chainid>
//
// This format is round-trippable via ChainFromKey / ChainFromNSS. The
// optional coin-type metadata is deliberately excluded: the chain identity
// is the (network type, chain id) pair.
func (c *Chain) String() string {
	return fmt.Sprintf("nt:%s:ci:%s", c.networkType, c.chainId)
}
