# go-mhda

MultiChain Hierarchical Deterministic Address format (MHDA) with Uniform Resource Name (URN)
format [RFC 8141](https://datatracker.ietf.org/doc/rfc8141/) compatibility.

This is address notation expands BIP32/44/84, and providing additional possibilities for configuring 
any blockchain address types.

URN:

```
urn:mhda:nt:{network_type}:ct:{coin_type}:ci:{chain_id}:dt:{derivation_type}:dp:{derivation_path}:aa:{algorithm}:af:{address_format}:ap:{address_prefix}:as:{address_suffix}
```

| **Parameter** |       **Name**       |          |    **Type**    | **Description**                                                                                                      |
|:-------------:|:--------------------:|:--------:|:--------------:|----------------------------------------------------------------------------------------------------------------------|
|      urn      |    URN Namespace     | constant |     string     | "mhda"                                                                                                               |
|      nt       |     Network Type     | required |     string     | Network type, grouped by name: "evm", "tvm", "avm", "btc", "cosmos"                                                  |
|      ct       |      Coin Type       | required |    numeric     | Coin type, according slip44: Bitcoin/BTC=0, Litecoin/LTC=2, Ethereum/ETH=60, Tron/TRX=195, Polygon/MATIC=966         |
|      ci       |       Chain Id       | required |     string     | Chain id: for numeric - "0x1", "0x10", for another - string "axelar"                                                 ||
|      dt       | Derivation Path Type | optional |     string     | Derivation path type by name: "root", "bip32", "bip44", "bip49", "bip84", "cip11"                                    |
|      dp       |   Derivation Path    | optional | string \| null | Derivation path, according *dt* parameter: null, "m/0'/0/0", "m/44'/0'/0'/0/0", "m/49'/0'/0'/0/0", "m/84h/0h/0h/0/0" |
|      aa       |  Address Algorithm   | optional | string \| null | Address hierarchical algorithm by name: "ed25519", "secp256k1"                                                       |
|      af       |    Address Format    | optional | string \| null | Address format by name: "hex", "p2pkh", "p2s4", "bech32"                                                             |
|      ap       |    Address Prefix    | optional | string \| null | Address prefix: "0x", "1\|3\|bc1"                                                                                    |
|      as       |    Address Suffix    | optional | string \| null | Address suffix                                                                                                       |

## Examples Ethereum

### BIP-44

```
# Short format
# default filled: aa=secp256k1, af=hex, ap=0x
urn:mhda:nt:evm:ct:60:ci:0x1:dt:bip44:dp:m/44h/60h/0h/0/0

# Long format
urn:mhda:nt:evm:ct:60:ci:0x1:dt:bip44:dp:m/44h/60h/0h/0/0:aa:secp256k1:af:hex:ap:0x
```

## Examples Bitcoin

### BIP-44
```
# Legacy (P2PKH) // ap=1
urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/44h/0h/0h/0/0:aa:secp256k1:af:p2pkh:ap:1

# Nested SegWit (P2SH) // ap=3
urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/84h/0h/0h/0/0:aa:secp256k1:af:p2sh:ap:3

# Native SegWit (Bech32) // ap=bc1q
urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip44:dp:m/84h/0h/0h/0/0:aa:secp256k1:af:p2pkh:ap:bc1q

# Taproot (P2TR)  // ap=bc1p
urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip84:dp:m/86h/0h/0h/0/0:aa:secp256k1:af:p2pkh:ap:bc1p

```

## Examples Avalanche

### BIP-44

```
# C-Chain
urn:mhda:nt:evm:ct:60:ci:0xa86a:dt:bip44:dp:m/44h/60h/0h/0/0

# X-Chain   https://subnets.avax.network/x-chain
# default filled: aa=secp256k1, af=hex, ap=X-avax
urn:mhda:nt:avm:ct:9000:ci:0x1:dt:bip44:dp:m/44h/9000h/0h/0/0
# Long
urn:mhda:nt:avm:ct:9000:ci:0x1:dt:bip44:dp:m/44h/9000h/0h/0/0:aa:secp256k1:af:hex:ap:X-avax

```

### Root key (no derivation path)

```
# Short format
# default filled: dp=null, aa=secp256k1, af=hex, ap=0x
urn:mhda:nt:evm:ct:60:ci:0x1

# Long format
urn:mhda:nt:evm:ct:60:ci:0x1:aa:secp256k1:af:hex:ap:0x
```
