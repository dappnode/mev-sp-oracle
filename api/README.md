# Oracle api docs

This document contains some non-exhaustive documentation for the oracle api, with some examples on how it can be used to retrieve different information from the validators such as their accumulated balance, claimable balances, proofs to claim their balance, wrong/valid/missed proposed blocks into the pool, etc.


// TODO: modify this to validatorindex=xxx

curl 157.90.93.245:7300/status
curl 157.90.93.245:7300/depositadddress/408070
curl 157.90.93.245:7300/proof/0xa111B576408B1CcDacA3eF26f22f082C49bcaa55
curl 157.90.93.245:7300/validatoronchainstate/408070
curl 157.90.93.245:7300/donations
curl 157.90.93.245:7300/poolstatistics
curl 157.90.93.245:7300/registeredrelays/0xb1ce83f50ba296bdfedba0e4a42a65f8cee1bdeb2ba78aaa61b452141684930406412bbef6c0f65b4121f8fc82dbb6ba