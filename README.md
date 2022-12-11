# mev-sp-oracle

The dappnode **mev smoothing pool** is made of three repositories:
* [mev-sp-contracts](https://github.com/dappnode/mev-sp-contracts): contains the smoothing pool contracts where validators must send their rewards to, used to subscribe/unsubscribe and claim their share.
* [mev-sp-oracle](https://github.com/dappnode/mev-sp-oracle): contains the rewards calculation algorithm and utilities to both update the merkle root of the tree and create proofs to be used in the smart contract for claiming rewards.
* [mev-sp-trees](https://github.com/dappnode/mev-sp-trees): contains all rewards calculations for all subscribed validators organised per checkpoint, with all the merkle proofs and each checkpoint's merkle root.

## Example
Note that this software is under development in a very early stage. Do not use in any production environment.

Add some hardcoded subscriptions:
```
$ cat file.txt
543
57654
36
334
```

Compile and run:
```
go build
./mev-sp-oracle --consensus-endpoint="http://127.0.0.1:5051" --execution-endpoint="http://127.0.0.1:8545" --deployed-slot=5324453 --pool-address="0x" --checkpoint-size=10 --debug-hardcoded-subscriptions-file=file.txt
```