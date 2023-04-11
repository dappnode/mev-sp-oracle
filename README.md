# mev-sp-oracle

The dappnode **mev smoothing pool** is made of three repositories:
* [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
* [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
* [mev-sp-trees](https://github.com/dappnode/mev-sp-trees): contains all rewards calculations for all subscribed validators organised per checkpoint, with all the merkle proofs and each checkpoint's merkle root.

```
Usage of ./mev-sp-oracle:
  -checkpoint-size uint
    	Size in slots for each checkpoint, used to generate dumps and update merkle roots
  -consensus-endpoint string
    	Ethereum consensus endpoint
  -debug-subscriptions-file string
    	Path to file containing a list of hardcoded validator indexes, one per line
  -deployed-slot uint
    	Deployed slot of the smart contract: slot, not block
  -deployer-private-key string
    	Private key of the deployer account
  -execution-endpoint string
    	Ethereum execution endpoint
  -network string
    	Network to run in: mainnet|goerli (default "mainnet")
  -pool-address string
    	Address of the smoothing pool contract
  -postgres-endpoint string
    	Postgres endpoint
  -version
    	Prints the release version and exits
```

## Goerli Example

Use `--dry-run` to avoid updating the contract, useful when you want to recreate the state to verify the merkle roots.
```console
$ go build
$ ./mev-sp-oracle \
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--postgres-endpoint="postgres://xxx:yyy@localhost:5432" \
--deployed-slot=5195000 \
--pool-address="0x455e5aa18469bc6ccef49594645666c587a3a71b" \
--checkpoint-size=100 \
--deployer-private-key="xxx" \ TODO: use file
--pool-fees-percent=10 \
--pool-fees-address=0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5 \
--network=goerli
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

Store in `.env` so that it's picked up by `docker-compose`

```
export NETWORK=goerli
export DEPLOYED_SLOT=4500000
export CHECKPOINT_SIZE=10000
export POOL_ADDRESS=0x455e5aa18469bc6ccef49594645666c587a3a71b
export POOL_FEES_ADDRESS=0xfeeB31E7E49d689F4368744B6F1efa40341cD33e
export POOL_FEES_PERCENT=10
export POSTGRES_USER=xxx
export POSTGRES_PASSWORD=yyy
export BLOCK_DEPOSIT_CONTRACT=4367322
export DEPLOYER_PRIVATE_KEY=xxx
```

```
docker-compose up -d
```

Use to check that all env variables were correctly replaced
```console
docker compose convert
```

### Mainnet

Store in `.env` so that it's picked up by `docker-compose`

```
TODO
```

```
docker-compose up -d
```
