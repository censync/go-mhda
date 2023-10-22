package go_mhda

import (
	"errors"
	"fmt"
	"regexp"
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

var (
	chainComponents = []string{
		compNetworkType,
		compCoinType,
		compChainId,
	}

	rxChainComponents = regexp.MustCompile(`:(nt|ct|ci):([0-9a-z-._~*+=%$&@?'()!,;/#]+)`)
)

func NewChain(networkType NetworkType, coinType CoinType, chainId ChainId) *Chain {
	return &Chain{networkType: networkType, coinType: coinType, chainId: chainId}
}

func ChainFromKey(chainKey ChainKey) (*Chain, error) {
	return ChainFromNSS(string(chainKey))
}

func ChainFromNSS(src string) (*Chain, error) {
	var componentsNamesTmp = make([]string, len(componentsNames))

	copy(componentsNamesTmp, componentsNames)

	components, err := parseNSS(src, componentsNamesTmp)

	if err != nil {
		return nil, err
	}

	if _, ok := components[compNetworkType]; !ok {
		return nil, errors.New(`"nt" not defined`)
	}

	return parseChain(components)
}

func parseChain(m map[string]string) (*Chain, error) {
	networkType := strings.TrimSpace(m[compNetworkType])

	// TODO: Check coin type extraction from derivation path??? subnets???
	if networkType == `` {
		return nil, errors.New(`"networkType" required`)
	}

	ct := strings.TrimSpace(m[compCoinType])
	// TODO: Check coin type extraction from derivation path??? subnets???
	if ct == `` {
		return nil, errors.New(`"ct" required`)
	}

	coinType, err := strconv.ParseUint(ct, 0, 32)
	if err != nil {
		return nil, errors.New(`cannot parse "ct"`)
	}

	// TODO: Add ci validation
	if _, ok := m[compChainId]; !ok {
		return nil, errors.New(`numeric "ci" required for "ct=evm"`)
	}
	return &Chain{
		networkType: NetworkType(networkType), // TODO: Add validation
		coinType:    CoinType(coinType),
		chainId:     ChainId(m[compChainId]),
	}, nil
}

func (c *Chain) SetNetworkType(networkType NetworkType) {
	c.networkType = networkType
}

func (c *Chain) SetCoinType(coinType CoinType) {
	c.coinType = coinType
}

func (c *Chain) SetChainId(chainId ChainId) {
	c.chainId = chainId
}

func (c *Chain) NetworkType() NetworkType {
	return c.networkType
}

func (c *Chain) CoinType() CoinType {
	return c.coinType
}

func (c *Chain) ChainId() ChainId {
	return c.chainId
}
func (c *Chain) Key() ChainKey {
	return ChainKey(c.String())
}

func (c *Chain) String() string {
	return fmt.Sprintf("nt:%s:ct:%d:ci:%s", c.networkType, c.CoinType(), c.chainId)
}
