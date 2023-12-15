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
* **Updater**: Updates the Merkle root onchain. Only whitelisted addresses are allowed to update said root.
* **Verifier**: Calculates all balances and generates proofs. To be used by anyone.

## Running Smooth Oracle

```
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
