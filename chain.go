package go_mhda

import (
	"fmt"
	"strconv"
	"strings"
)

type ChainId string

// ChainKey - string identifier for declaration any chain or subchain
type ChainKey string

type Chain struct {
	networkType NetworkType
	coinType    CoinType
	chainId     ChainId
}

func NewChain(networkType NetworkType, coinType CoinType, chainId ChainId) *Chain {
	return &Chain{networkType: networkType, coinType: coinType, chainId: chainId}
}

// ChainFromKey parses a Chain from a key produced by Chain.Key().
func ChainFromKey(chainKey ChainKey) (*Chain, error) {
	return ChainFromNSS(string(chainKey))
}

// ChainFromNSS parses just the chain-domain components ("nt", "ct", "ci")
// from the given NSS string. Other components are tolerated and ignored.
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
// component map produced by the NSS parser. Shared by parseAddress and
// ChainFromNSS.
func parseChain(m map[string]string) (*Chain, error) {
	networkType := normalize(m[compNetworkType])
	if networkType == `` {
		return nil, ErrMissingNetworkType
	}
	if !NetworkType(networkType).IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidNetworkType, networkType)
	}

	ct := strings.TrimSpace(m[compCoinType])
	if ct == `` {
		return nil, ErrMissingCoinType
	}
	coinType, err := strconv.ParseUint(ct, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidCoinType, ct)
	}

	chainID, ok := m[compChainId]
	if !ok || strings.TrimSpace(chainID) == "" {
		return nil, ErrMissingChainID
	}

	return &Chain{
		networkType: NetworkType(networkType),
		coinType:    CoinType(coinType),
		chainId:     ChainId(chainID),
	}, nil
}

func (c *Chain) SetNetworkType(networkType NetworkType) { c.networkType = networkType }
func (c *Chain) SetCoinType(coinType CoinType)          { c.coinType = coinType }
func (c *Chain) SetChainId(chainId ChainId)             { c.chainId = chainId }

func (c *Chain) NetworkType() NetworkType { return c.networkType }
func (c *Chain) CoinType() CoinType       { return c.coinType }
func (c *Chain) ChainId() ChainId         { return c.chainId }

func (c *Chain) Key() ChainKey {
	return ChainKey(c.String())
}

// String returns the canonical NSS-style chain key:
//
//	nt:<network>:ct:<coin>:ci:<chainid>
//
// This format is round-trippable via ChainFromNSS / ChainFromKey.
func (c *Chain) String() string {
	return fmt.Sprintf("nt:%s:ct:%d:ci:%s", c.networkType, c.coinType, c.chainId)
}
