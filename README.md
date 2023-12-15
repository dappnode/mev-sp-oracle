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
* **Verifier**: Calculates all balances and generates proofs. To be used by anyone. Optional.

## Running Your Smooth Oracle

You need a consensus + execution client + point the oracle to Smooth mainnet contract`0xAdFb8D27671F14f297eE94135e266aAFf8752e35`. Note that the consensus client shall be running in archival mode, `slots-per-restore-point=512` is enough. It will take some time to sync.
```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:3500" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0xAdFb8D27671F14f297eE94135e266aAFf8752e35 \
--relayers-endpoints=https://NOTNEEDED \
--log-level=debug \
--dry-run
```

The oracle exposes a REST API documented [here](https://github.com/dappnode/mev-sp-oracle/tree/main/api) that you can use to monitor its health, check your rewards, and calculate your Merkle proofs to claim your rewards. Note that this is optional and [smooth.dappnode.io](smooth.dappnode.io) is provided for convenience, but not needed, since a local hosted oracle can provide the same data.

You can check the sync status. The `oracle_sync_distance_slots` indicates how far the oracle is behind the latest finalized slot, where 0 means totally in sync.

```
curl localhost:7300/status
```

You can also get the Merkle proofs of your withdrawal address as follows. They can be submitted directly into the contract using the [claimRewards](https://etherscan.io/address/0xAdFb8D27671F14f297eE94135e266aAFf8752e35#writeProxyContract) call.
```
curl localhost:7300/onchain/proof/0X_YOUR_WITHDRAWAL_ADDRESS
```

If someone you trust runs an oracle you can use the `--checkpoint-sync-url=http://ip_address:7300/state` flag. This will get the state from that oracle, and continue syncing from there. Useful to avoid having to sync everything, but requires trust in the endpoint provider.

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
