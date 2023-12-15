# mev-sp-oracle


[**Smooth**](https://smooth.dappnode.io/) is made of:

- [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
- [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
- [mev-sp-fe](https://github.com/dappnode/mev-sp-fe): contains a frontend that allows to interact with the oracle in a user friendly way.

## Description

This software allows anyone to reconstruct the same state as the one available in [smooth.dappnode.io](https://smooth.dappnode.io/), which is just provided for convenience. Using a consensus + execution client and this oracle, you will be able to calculate the rewards of all subscribed validators, generate your own Merkle proofs, and use them to claim your rewards, without relaying in any 3rd party hosted service. Technical details of the implementation can be found in the [specification](https://github.com/dappnode/mev-sp-oracle/blob/main/spec/README.md).

## Build from source

```
go build
./mev-sp-oracle --help
```

## Docker images

```
docker pull dappnode/mev-sp-oracle:1.0.4
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

## Tests

Note that some files used for testing are bigger than what Github allows, so you may have to fetch it with `git lfs`.
```
git lfs checkout
go test ./... -v
```

## License

[GNU General Public License v3.0](https://github.com/dappnode/mev-sp-oracle/blob/main/LICENSE)

## Legal Disclaimer
[Terms Of Service](https://github.com/dappnode/mev-sp-oracle/blob/main/TERMS_OF_SERVICE.md)
