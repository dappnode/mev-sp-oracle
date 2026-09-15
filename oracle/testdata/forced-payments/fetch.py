#!/usr/bin/env python3
"""Refresh the offline mainnet corpus; public reads only, no RPC credentials.

Run from this directory. Original signed transactions come from the beacon API.
Balances, event logs, and receipt/trace facts come from Blockscout. Receipt JSON
is normalized from explorer fields; it is not an unmodified JSON-RPC response.
"""
import concurrent.futures
import gzip
import json
import hashlib
from pathlib import Path
import tempfile
import time
import urllib.error
import urllib.request

POOL = "0xAdFb8D27671F14f297eE94135e266aAFf8752e35"
BEACON = "https://ethereum-beacon-api.publicnode.com"
EXPLORER = "https://eth.blockscout.com"
GENESIS = 1606824023
OUT = Path(__file__).resolve().parent


def get(url):
    cache = Path(tempfile.gettempdir()) / "oracle-forced-payment-fixtures"
    cache.mkdir(exist_ok=True)
    cached = cache / (hashlib.sha256(url.encode()).hexdigest() + ".json")
    if cached.exists():
        return json.loads(cached.read_text())
    request = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    for attempt in range(5):
        try:
            with urllib.request.urlopen(request, timeout=45) as response:
                result = json.load(response)
            cached.write_text(json.dumps(result))
            return result
        except urllib.error.HTTPError as error:
            if error.code not in (429, 500, 502, 503, 504) or attempt == 4:
                print("Failed URL:", url)
                raise
            time.sleep(2 ** (attempt + 1))


history_url = f"{EXPLORER}/api/v2/addresses/{POOL}/coin-balance-history?block_number=25957038"
history = get(history_url)["items"]
logs_url = f"{EXPLORER}/api/v2/addresses/{POOL}/logs?block_number=25957038&index=0&items_count=50"
log_page = get(logs_url)
assert log_page["next_page_params"] is None or log_page["next_page_params"]["block_number"] < history[7]["block_number"]
all_logs = log_page["items"]
trace_url = f"{EXPLORER}/api/v2/addresses/{POOL}/internal-transactions?block_number=25957038&transaction_index=0&index=0&items_count=50"
trace_page = get(trace_url)
assert trace_page["next_page_params"] is None or trace_page["next_page_params"]["block_number"] < history[7]["block_number"]
all_traces = trace_page["items"]


def balance_at(number):
    return next(item["value"] for item in history if item["block_number"] <= number)


def slot_for(item):
    from datetime import datetime
    block = get(f'{EXPLORER}/api/v2/blocks/{item["block_number"]}')
    timestamp = int(datetime.fromisoformat(block["timestamp"].replace("Z", "+00:00")).timestamp())
    return (timestamp - GENESIS) // 12


with concurrent.futures.ThreadPoolExecutor(max_workers=4) as workers:
    previous_slots = list(workers.map(slot_for, history[2:8]))
slots = sorted(set(range(15194824, 15194856)) | {15195027} | set(previous_slots))


def fetch(slot):
    beacon_url = f"{BEACON}/eth/v2/beacon/blocks/{slot}"
    block = get(beacon_url)
    assert block["finalized"] and not block["execution_optimistic"]
    message = block["data"]["message"]
    payload = message["body"]["execution_payload"]
    number = int(payload["block_number"])
    logs = []
    for log in all_logs:
        if log["block_number"] != number:
            continue
        tx = get(f'{EXPLORER}/api/v2/transactions/{log["transaction_hash"]}')
        logs.append({
            "address": log["address"]["hash"], "data": log["data"],
            "topics": [topic for topic in log["topics"] if topic],
            "blockHash": log["block_hash"], "blockNumber": hex(number),
            "transactionHash": log["transaction_hash"], "transactionIndex": hex(tx["position"]),
            "logIndex": hex(log["index"]), "removed": False,
        })
    fixture = {
        "slot": slot, "proposer_index": message["proposer_index"],
        "payload": payload, "balance_before": balance_at(number - 1),
        "balance_after": balance_at(number), "logs": logs,
        "expected_reward": "0", "receipt": None,
        "sources": [beacon_url, history_url, logs_url],
    }
    if fixture["balance_before"] != fixture["balance_after"]:
        change = next(item for item in history if item["block_number"] == number)
        tx_hash = change["transaction_hash"]
        tx_url = f"{EXPLORER}/api/v2/transactions/{tx_hash}"
        tx = get(tx_url)
        transfers = [trace for trace in all_traces if trace["transaction_hash"] == tx_hash and trace["type"] == "selfdestruct"
                     and trace["to"]["hash"].lower() == POOL.lower() and trace["success"]]
        assert len(transfers) == 1, (number, transfers)
        fixture["expected_reward"] = transfers[0]["value"]
        fixture["receipt"] = {
            "transactionHash": tx_hash, "transactionIndex": hex(tx["position"]),
            "blockHash": payload["block_hash"], "blockNumber": hex(number),
            "status": "0x1" if tx["status"] == "ok" else "0x0",
            "type": hex(tx["type"]), "gasUsed": hex(int(tx["gas_used"])),
            "cumulativeGasUsed": "0x0", "logsBloom": "0x" + "00" * 256,
            "logs": [], "contractAddress": None,
        }
        fixture["sources"] += [tx_url, trace_url]
    return fixture


with concurrent.futures.ThreadPoolExecutor(max_workers=4) as workers:
    fixtures = list(workers.map(fetch, slots))
data = json.dumps(fixtures, separators=(",", ":")).encode()
(OUT / "mainnet.json.gz").write_bytes(gzip.compress(data, mtime=0))
print(f"Saved {len(fixtures)} blocks, {sum(len(f['payload']['transactions']) for f in fixtures)} signed transactions")
