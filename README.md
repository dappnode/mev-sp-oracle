# mev-sp-oracle

The dappnode **mev smoothing pool** is made of three repositories:
* [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
* [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
* [mev-sp-trees](https://github.com/dappnode/mev-sp-trees): contains all rewards calculations for all subscribed validators organised per checkpoint, with all the merkle proofs and each checkpoint's merkle root.

## Build from source

```
go build
./mev-sp-oracle --help
```

## Docker images

Latest master is available and identified by its first 7 commit digits. Example:
```
dappnode/mev-sp-oracle:583e6e1
```

## Roles

The oracle can run in two modes:
* Updater: Recreates the state of all validator balances and every `checkpoint-size` updates the onchain contract with the new merkle root. Note that this mode requires to be configured with a valid keystore (containing the encrypted key) and this address must be allowed to update the contract onchain. Said account requires gas to pay for gas fees.
* Verifier: Recreates the state calculating all validator balances, and can be run by anyone, but does not update the onchain root.

## Goerli Example

Running in `updater` mode:

```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0x8eba4A4A8d4DFa78BCB734efD1eA9f33b61e3243 \
--updater-keystore-path=xxx \
--updater-keystore-pass=yyy
```

Running in verifier mode:

```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0xF21fbbA423f3a893A2402d68240B219308AbCA46 \
--dry-run
```

## How to deploy

### Goerli Testnet

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
export POOL_ADDRESS=0x8eba4A4A8d4DFa78BCB734efD1eA9f33b61e3243
export UPDATER_KEYSTORE_PASS=xxx
```

```
docker-compose up -d
```

Use to check that all env variables were correctly replaced
```console
docker compose convert
```
