package go_mhda

const (
	// btc
	BTC  = CoinType(0)
	LTC  = CoinType(2)
	DOGE = CoinType(3)

	// evm
	ETH   = CoinType(60)
	BNB   = CoinType(714) // old stype
	BSC   = CoinType(9006)
	MATIC = CoinType(966)
	GLMR  = CoinType(1284)

	DASH = CoinType(5)

	XMR  = CoinType(128)
	ZEC  = CoinType(133)
	XRP  = CoinType(144)
	XLM  = CoinType(148)
	ATOM = CoinType(168)
	TRX  = CoinType(195)
	ALGO = CoinType(283)
	NEAR = CoinType(397)
	SOL  = CoinType(501)
	APT  = CoinType(637)
	TON  = CoinType(607)
	SUI  = CoinType(784)
	ADA  = CoinType(1815)

	//https://support.avax.network/en/articles/7004986-what-derivation-paths-does-avalanche-use
	AVAX = CoinType(9000)
)

// CoinType is a SLIP-44 coin type (32-bit unsigned integer).
type CoinType uint32
