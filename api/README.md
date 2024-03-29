# Oracle api docs

This document contains some non-exhaustive documentation for the oracle api, with some examples on how it can be used to retrieve different information from the validators such as their accumulated balance, claimable balances, proofs to claim their balance, wrong/valid/missed proposed blocks into the pool, etc.

## General endpoints

Fetches the status of the oracle, indicating if the underlying consensus and execution clients are in in sync, and if the oracle is in sync and how far behind head is.

```
curl url:7300/status
```

Returns the configuration parameters of the smoothing pool.
```
curl url:7300/config
```

Returns if a given validator public key is registered in a set of relays. Use to know if a validator
```
curl url:7300/registeredrelays/0xb1ce83f50ba296bdfedba0e4a42a65f8cee1bdeb2ba78aaa61b452141684930406412bbef6c0f65b4121f8fc82dbb6ba
```

Returns the whole oracle state.
```
curl url:7300/state
```


## Memory endpoints

Memory endpoints return the latest known state by the oracle.


Returns all the validators that are tracked by the smoothing pool in any state.
```
curl url:7300/memory/validators
```

Returns the configuration of the pool

```
curl url:7300/config
```

Return information of a given validator index registered in the pool.

```
curl url:7300/memory/validator/408120
```

Return information of all subscribed validators from a withdrawal address, including validators not tracked by the pool

```
curl url:7300/memory/validators/0xa111B576408B1CcDacA3eF26f22f082C49bcaa55
```

Returns information on the fees that the pool takes, such as percent, address and fees so far.

```
curl url:7300/memory/feesinfo
```

Return all proposed blocks to the pool

```
curl url:7300/memory/proposedblocks
```

Return all the missed blocks in the smoothing pool.
```
curl url:7300/memory/missedblocks
```

Return all blocks that were wrong (wrong fee recipient of subscribed validators)
```
curl url:7300/memory/wrongfeeblocks
```

Returns all blocks that the pool has knowledge of (`proposedblocks`, `missedblocks`, `wrongfeeblocks`)

```
curl url:7300/memory/allblocks
```

Return a list of all donations to the pool
```
curl url:7300/memory/donations
```

General statistics of the pool such as rewards received, amount of block, average reward, etc.
```
curl url:7300/memory/statistics
```

## Onchain endpoints

Onchain endpoints return information from the point of view of the latest stored state (as a merkle root) in the blockchain.

Returns the merkle proofs of the given withdrawal address, that can be used on chain to claim the rewards. Note that this endpoint can be used by the account that gets the fees of the pool.

```
curl url:7300/onchain/proof/0xa111b576408b1ccdaca3ef26f22f082c49bcaa55
```
