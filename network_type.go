package go_mhda

import "errors"

type NetworkType string

const (
	Bitcoin      = NetworkType(`btc`)
	EthereumVM   = NetworkType(`evm`)
	AvalancheVM  = NetworkType(`avm`)
	TronVM       = NetworkType(`tvm`)
	Cosmos       = NetworkType(`cosmos`)
	Solana       = NetworkType(`sol`)
	XRPLedger    = NetworkType(`xrp`)
	Stellar      = NetworkType(`xlm`)
	NEARProtocol = NetworkType(`near`)
	Aptos        = NetworkType(`apt`)
	Sui          = NetworkType(`sui`)
	Cardano      = NetworkType(`ada`)
	Algorand     = NetworkType(`algo`)
	Toncoin      = NetworkType(`ton`)
)

var ntIndex = map[string]NetworkType{
	`btc`:    Bitcoin,
	`evm`:    EthereumVM,
	`avm`:    AvalancheVM,
	`tvm`:    TronVM,
	`cosmos`: Cosmos,
	`sol`:    Solana,
	`xrp`:    XRPLedger,
	`xlm`:    Stellar,
	`near`:   NEARProtocol,
	`apt`:    Aptos,
	`sui`:    Sui,
	`ada`:    Cardano,
	`algo`:   Algorand,
	`ton`:    Toncoin,
}

// NetworkTypeFromString parses a string into a NetworkType. The lookup is
// case-insensitive; surrounding whitespace is stripped.
func NetworkTypeFromString(src string) (NetworkType, error) {
	if result, ok := ntIndex[normalize(src)]; ok {
		return result, nil
	}
	return "", errors.New("undefined network type")
}

func (nt NetworkType) IsValid() bool {
	_, ok := ntIndex[string(nt)]
	return ok
}

func (nt NetworkType) String() string {
	return string(nt)
}
