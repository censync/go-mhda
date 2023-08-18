package go_mhda

type NetworkType string

const (
	Bitcoin     = NetworkType(`btc`)
	EthereumVM  = NetworkType(`evm`)
	AvalancheVM = NetworkType(`avm`)
	TronVM      = NetworkType(`tvm`)
	Cosmos      = NetworkType(`cosmos`)
	Solana      = NetworkType(`sol`)
)
