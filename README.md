# mev-sp-oracle

The dappnode **mev smoothing pool** is made of three repositories:
* [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
* [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
* [mev-sp-trees](https://github.com/dappnode/mev-sp-trees): contains all rewards calculations for all subscribed validators organised per checkpoint, with all the merkle proofs and each checkpoint's merkle root.
* [specification](https://github.com/dappnode/mev-sp-oracle/blob/main/spec/README.md): contains a detailed version of how the oracle works and calculates the fees.

## Build from source

```
go build
./mev-sp-oracle --help
```

## Docker images

Latest master is available and identified by its first 7 commit digits. Example:
```
dappnode/mev-sp-oracle:583e6e1
```

## Tests

Note that some files used for testing are bigger than what Github allows, so you may have to fetch it with `git lfs`.
```
git lfs checkout
go test ./... -v
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
--pool-address=0xF21fbbA423f3a893A2402d68240B219308AbCA46 \
--updater-keystore-path=keystore-path \
--updater-keystore-pass=password-of-keystore
```

Running in `verifier` mode:

```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0xF21fbbA423f3a893A2402d68240B219308AbCA46 \
--dry-run
```

Note that syncing might take some time, but if you trust another oracle you can use it as a checkpoint sync with `--checkpoint-sync-url=http://ip_address:7300/state`. This will get the state from that oracle, and continue syncing from there. Not recommended to be used in `updater` mode.

## Deployments

This repo contains not only the oracle software but also a docker-compose with all required components such as consensus, execution clients and prometheus/grafana to monitor the setup. Everything is provisioned, so you just need to do the following.

### Goerli Testnet

```console
$ sudo openssl rand -hex 32 > jwtsecret
$ chmod 777 teku
```

Store in `.env` so that it's picked up by `docker-compose`. Notice that REGISTERED_RELAYS should be a comma separated list of relays that 
operate in the same network that the oracle is running on. The following is an example for goerli:

```
export NETWORK=goerli
export POOL_ADDRESS=0xF21fbbA423f3a893A2402d68240B219308AbCA46
export UPDATER_KEYSTORE_PASS=xxx
export REGISTERED_RELAYS=https://builder-relay-goerli.flashbots.net,https://bloxroute.max-profit.builder.goerli.blxrbdn.com,https://relay-goerli.edennetwork.io,https://goerli-relay.securerpc.com
```
And manually modify the following placeholders in `deployments/monitoring/prometheus-config.yml`:
* Change `OPERATOR_PLACEHOLDER` for your identifier eg `monitor: "oracle_someentity"`.
* Replace the `URL_PLACEHOLDER` (or whole url) to the one you want to push metrics to.
* Replace `USERNAME_PLACEHOLDER`.
* And `PASSWORD_PLACEHOLDER`.


```
docker-compose up -d
```

Use to check that all env variables were correctly replaced
```console
docker compose convert
```
