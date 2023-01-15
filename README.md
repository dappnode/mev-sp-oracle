# mev-sp-oracle

The dappnode **mev smoothing pool** is made of three repositories:
* [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
* [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
* [mev-sp-trees](https://github.com/dappnode/mev-sp-trees): contains all rewards calculations for all subscribed validators organised per checkpoint, with all the merkle proofs and each checkpoint's merkle root.

## Goerli Example

```console
$ go build
$ ./mev-sp-oracle --consensus-endpoint="http://127.0.0.1:5051" --execution-endpoint="http://127.0.0.1:8545" --deployed-slot=4500000 --pool-address="0x455e5aa18469bc6ccef49594645666c587a3a71b" --checkpoint-size=10
```

## How to deploy

### Goerli Testnet

Debug tools:
* https://prater.beaconcha.in
* https://goerli.etherscan.io

```console
$ sudo openssl rand -hex 32 > jwtsecret
$ chmod 777 teku
```

Goerli deposit contract is `0xff50ed3d0ec03aC01D4C79aAd74928BFF48a7b2b` but some validators were predeposited, so not all of the will be indexes and most likely doesn't have a deposit address observable onchain.

```
export NETWORK=goerli
export DEPLOYED_SLOT=4500000
export CHECKPOINT_SIZE=10
export POOL_ADDRESS=0x455e5aa18469bc6ccef49594645666c587a3a71b
export POSTGRES_USER=xxx
export POSTGRES_PASSWORD=yyy
export BLOCK_DEPOSIT_CONTRACT=6711090
docker-compose up -d
```

Use to check that all env variables were correctly replaced
```console
docker compose convert
```

### Mainnet


```
export NETWORK=mainnet
export DEPLOYED_SLOT=xxx
export CHECKPOINT_SIZE=xxx
export POOL_ADDRESS=0xxx
export POSTGRES_USER=xxx
export POSTGRES_PASSWORD=yyy
docker-compose up -d
```