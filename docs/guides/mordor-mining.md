---
title: Mine on Mordor
---

Mordor is Ethereum Classic's proof-of-work test network. Mining there tests a mining or pool setup without
real funds, and it is a way to get Mordor coins when the faucets are dry. Mordor uses Etchash from block
2,520,000.

## Before you start

- A synced Mordor node: [Run a Mordor node](../getting-started/run-mordor-node.md).
- An address for the rewards. The node needs no key for it.
- For CPU mining, disk and memory for the Etchash DAG, which is several gigabytes. The node generates it
  before it starts mining.

## Mine on the CPU

```sh
geth --mordor --mine --miner.threads 1 --miner.etherbase 0xYOUR_ADDRESS
```

The node writes the DAG to `--ethash.dagdir`, `~/.ethash` by default, then starts mining. A CPU finds
Mordor blocks slowly. For faster results, point GPU mining software at the node as
[Mining](mining.md#mine-with-external-mining-software) describes, with `--mordor` in place of `--classic`.

## Check the rewards

```sh
geth --mordor attach --exec 'eth.getBalance("0xYOUR_ADDRESS")' <datadir>/geth.ipc
```

The balance is in wei, and it grows as the node imports blocks mined to the address.

## Stop mining

Stop the node, and start it again without `--mine`.
[Stop it safely](../getting-started/run-classic-node.md#stop-it-safely) applies to Mordor too.
