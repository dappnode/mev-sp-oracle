# Forced payments at any transaction position

## Incident and behavior

Mainnet slot `15194839`, execution block `25956850`, contains a payment of
`13131532323905851` wei at transaction index **227**. Thirteen transactions follow
it. The last transaction, index 240, has zero top-level value and emits an
`EtherReceived` donation of `103797680070839` wei. Version 1.2.12 skipped its
forced-payment check because it examined only the last transaction's value.

The new detector reads the pool balance before and after each eligible block,
subtracts event-explained inflows and outflows, and returns immediately if the
remaining inflow is zero. Otherwise it scans the signed transactions already
present in the beacon payload for a unique exact-value candidate at any index.
It validates that transaction's successful receipt against the beacon payload's
execution block hash, block number, transaction hash and index. It records the
correct transaction provenance and adds one synthetic asset event. Real
donations are preserved, including donations equal to the forced payment.

The new algorithm activates on mainnet at slot `15194839`. Earlier slots retain
the v1.2.12 detector and existing exceptions. This is an activation boundary,
not an exception for a particular payment amount, builder or transaction.

## RPC cost

Counts exclude existing event, header, receipt, validator and reconciliation
requests. Retries can increase the counts below.

| Case | v1.2.12 | New detector |
| --- | --- | --- |
| Missed slot | 0 | 0 |
| Pool is the execution fee recipient (existing vanilla path) | 0 | 0 |
| Other produced block, positive last-tx value with recipient outside pool | 2 balance reads | 2 balance reads |
| Other produced block, last-tx value zero or recipient is pool | 0 | 2 balance reads |
| Detected payment, receipts already fetched | 0 additional receipt reads | 0 additional receipt reads |
| Detected payment, receipts not fetched | header + every transaction receipt | 1 payment receipt |

There are **no per-candidate RPCs**, new consensus requests, tracing requests or
new RPC method requirements. Balance reads are made once per block, not once
per transaction. With zero unexplained inflow the new detector does not scan the
transactions. No cross-block balance cache is used.

The 39-block corpus takes 70 balance reads under the old eligibility rule and
78 under the new algorithm: 8 extra calls across the sample. The worst increase
is two balance calls per produced block (about 0.167 calls/second at 12-second
slots). These calls are sequential: for the newly checked blocks, additional
waiting time is roughly twice the balance RPC latency. Historical replay needs
an endpoint that serves the requested historical balances, as it already did
for the old detector and checkpoint reconciliation.

On the development AMD Ryzen 7 7800X3D, scanning the incident's 241 signed
transactions measured approximately **0.27 ms**, 359 KB allocated, and 5,035
allocations. This is a local CPU microbenchmark, not a production RPC or
end-to-end synchronization benchmark. The corpus is a regression sample, not a
representative estimate of long-term block traffic.

## Verification and limits

`mainnet.json.gz` contains 39 real execution payloads (10,441 original signed
transactions): 32 consecutive slots around the incident, the later successful
payment at slot `15195027`, and six earlier forced payments. Expected outcomes
are eight forced payments and 31 unrelated blocks. Each fixture includes source
URLs, real pool balances and event logs. Successful selfdestruct transfers are
independently identified from explorer traces.

Receipt JSON is normalized from Blockscout transaction metadata, not captured
verbatim from an execution RPC. The stub supplies it through the actual
`ethclient` JSON-RPC interface. The incident's signed transactions were also
cross-checked against `https://lodestar-mainnet.chainsafe.io/eth/v2/beacon/blocks/15194839`.

Tests cover:

- Correct amount, transaction hash and index for all eight real payments.
- Unchanged reward classification and donations for the 31 unrelated blocks.
- The incident through state advancement, keeping the proposer active,
  consolidating its pending rewards, and passing both reconciliations. Starting
  validator accounts are synthetic; this is not a replay of production state.
- Exact RPC counts, receipt reuse, first/middle/last positions, ordinary
  donations, subscriptions, claims, equal-sized donations and missed slots.
- Activation boundaries, receipt/RPC failures, ambiguous candidates, unmatched
  inflows, unexplained outflows and consensus withdrawals to the pool.

As with the earlier implementation, balance evidence is at **block**
granularity; it is not a transaction execution trace. A unique exact-value
transaction is a candidate, not cryptographic proof of the internal transfer's
origin. Multiple matching transactions, split payments without a single exact
match, pre-funded zero-value forwarding, and mixed direct/forced proposer
payments fail explicitly before state advancement. They require further
attribution work; the detector does not guess or assign arbitrary surplus to a
proposer. Pool-fee-recipient blocks keep the existing vanilla path. This change
does not implement accounting for arbitrary forced donations or consensus
withdrawals to the pool.

Run from the repository root:

```sh
go test ./...
go test -race ./oracle -run 'Test_(AnyPositionForcedPayment|ForcedMevPayment|UnexplainedPoolInflow|MevRewardException)' -count=1
go test ./oracle -run '^$' -bench Benchmark_AnyPositionForcedPayment -benchmem
```

To refresh the fixed historical corpus, run `python3
oracle/testdata/forced-payments/fetch.py`. This performs public read-only HTTP
requests and caches responses in the system temporary directory. CI uses only
the checked-in gzip file and requires no external service or credentials.

## Recovery

Deploy the same updated algorithm across operators and replay from a saved
state **before slot 15194839**, initially with `--dry-run`. Confirm the validator
was not banned, the historical roots still agree, the new operator roots agree,
and onchain reconciliation passes before resuming submissions. Patching only
the current total or manually unbanning the validator does not undo its earlier
pending-reward redistribution. Keep the reconciliation checks enabled.
