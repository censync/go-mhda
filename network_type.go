package go_mhda

import "errors"

type NetworkType string

const (
	Bitcoin     = NetworkType(`btc`)
	EthereumVM  = NetworkType(`evm`)
	AvalancheVM = NetworkType(`avm`)
	TronVM      = NetworkType(`tvm`)
	Cosmos      = NetworkType(`cosmos`)
	Solana      = NetworkType(`sol`)
)

var ntIndex = map[string]NetworkType{
	`btc`:    Bitcoin,
	`evm`:    EthereumVM,
	`avm`:    AvalancheVM,
	`tvm`:    TronVM,
	`cosmos`: Cosmos,
	`sol`:    Solana,
}

func NetworkTypeFromString(src string) (NetworkType, error) {
	result, ok := ntIndex[src]
	if ok {
		return result, nil
	}
	return result, errors.New("undefined network type")
}

func (nt NetworkType) IsValid() bool {
	_, ok := ntIndex[string(nt)]
	return ok
}

func (nt NetworkType) String() string {
	return string(nt)
}
