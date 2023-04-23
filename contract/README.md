# contract

The content of this folder is automatically generated from the smoothing pool [smart contract](https://github.com/dappnode/mev-sp-contracts). The are two components that can be generated as follows. Note that if the smart contract changes, this would need to be updated:
* `abi.abi`
* `contract.go`


## Generating `abi.go`

This contains the smart contract abi. If the contract is verified in etherscan, it can be copied from there. [Example](https://goerli.etherscan.io/address/0x2df7254059f6a3ddcd1af0cb17797b26b6825e89#code) see "Contract ABI" section. Make sure you get the abi of the contract itself and not the proxy.


## Generating `contract.go`

With the abi, now we can generate the `contract.go` file using [abigen](https://geth.ethereum.org/docs/tools/abigen). This allows to easily interact with the contract using golang. First you need to install abigen.

```console
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum
go build ./cmd/abigen
```

And now from the root of this repo, generate the contract.

```console
./abigen --abi=../mev-sp-oracle/contract/abi.abi --pkg=contract --out=../mev-sp-oracle/contract/contract.go
```

