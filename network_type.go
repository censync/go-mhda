package go_mhda

import "errors"

type NetworkType string

// Network types use the commonly accepted network names, lowercase. Family
// types that cover a single ecosystem carry that ecosystem's name (tron,
// avalanche); "evm" stays as-is because it covers many independent networks.
const (
	Bitcoin      = NetworkType(`bitcoin`)
	EthereumVM   = NetworkType(`evm`)
	AvalancheVM  = NetworkType(`avalanche`)
	TronVM       = NetworkType(`tron`)
	Cosmos       = NetworkType(`cosmos`)
	Solana       = NetworkType(`solana`)
	XRPLedger    = NetworkType(`xrpl`)
	Stellar      = NetworkType(`stellar`)
	NEARProtocol = NetworkType(`near`)
	Aptos        = NetworkType(`aptos`)
	Sui          = NetworkType(`sui`)
	Cardano      = NetworkType(`cardano`)
	Algorand     = NetworkType(`algorand`)
	Toncoin      = NetworkType(`ton`)
)

var ntIndex = map[string]NetworkType{
	`bitcoin`:   Bitcoin,
	`evm`:       EthereumVM,
	`avalanche`: AvalancheVM,
	`tron`:      TronVM,
	`cosmos`:    Cosmos,
	`solana`:    Solana,
	`xrpl`:      XRPLedger,
	`stellar`:   Stellar,
	`near`:      NEARProtocol,
	`aptos`:     Aptos,
	`sui`:       Sui,
	`cardano`:   Cardano,
	`algorand`:  Algorand,
	`ton`:       Toncoin,
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
