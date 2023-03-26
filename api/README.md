# Oracle api docs

This document contains some non-exhaustive documentation for the oracle api, with some examples on how it can be used to retrieve different information from the validators such as their accumulated balance, claimable balances, proofs to claim their balance, wrong/valid/missed proposed blocks into the pool, etc.


// TODO: modify this to validatorindex=xxx

curl localhost:7300/status

curl localhost:7300/depositadddress/459438

curl localhost:7300/proof/0x005CD1608e40d1e775a97d12e4f594029567C071

curl localhost:7300/validatoronchainstate/179434

curl localhost:7300/donations