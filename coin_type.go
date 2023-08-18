package go_mhda

const (
	// btc
	BTC  = CoinType(0)
	LTC  = CoinType(2)
	DOGE = CoinType(3)

	// evm
	ETH   = CoinType(60)
	BNB   = CoinType(714)
	MATIC = CoinType(966)

	DASH = CoinType(5)

	XMR  = CoinType(128)
	ZEC  = CoinType(133)
	ATOM = CoinType(168)
	TRX  = CoinType(195)
	SOL  = CoinType(501)

	//https://support.avax.network/en/articles/7004986-what-derivation-paths-does-avalanche-use
	AVAX = CoinType(9000)
)

type CoinType uint
