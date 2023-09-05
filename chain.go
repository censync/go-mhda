package go_mhda

import (
	"fmt"
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

func (c Chain) SetNetworkType(networkType NetworkType) {
	c.networkType = networkType
}

func (c Chain) SetCoinType(coinType CoinType) {
	c.coinType = coinType
}

func (c Chain) SetChainId(chainId ChainId) {
	c.chainId = chainId
}

func (c Chain) NetworkType() NetworkType {
	return c.networkType
}

func (c Chain) CoinType() CoinType {
	return c.coinType
}

func (c Chain) ChainId() ChainId {
	return c.chainId
}
func (c Chain) Key() ChainKey {
	return ChainKey(c.String())
}

func (c Chain) String() string {
	return fmt.Sprintf("nt=%s:ct=%s:ci=%s", c.networkType, c.networkType, c.chainId)
}
