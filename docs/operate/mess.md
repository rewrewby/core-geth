---
title: MESS
---

MESS, Modified Exponential Subjective Scoring (ECBP-1100), makes a node resist deep chain
reorganizations. When a competing chain would replace blocks the node has already accepted, MESS
asks that chain for more total difficulty the older the point where the two chains split. A short
reorganization passes as it always did. A deep one, the kind a majority-hashrate attack produces,
needs far more work than the chain it replaces. The [MESS confirmation calculator](../guides/mess-calculator.md)
shows how much.

MESS is not a consensus rule. It never changes whether a block is valid, only which of two valid
chains the node prefers. Nodes that must agree on the chain, such as an exchange's deposit and
withdrawal nodes, should all run with the same MESS setting: during a deep reorganization, a node
with MESS on and a node with MESS off can prefer different chains.

## Where it applies

| Network | MESS applies from block | Deactivation block |
|---|---|---|
| Ethereum Classic | 11,380,000 | none |
| Mordor | 2,380,000 | none |

It is on by default on both networks. [MESS on this node](../getting-started/run-classic-node.md#mess-on-this-node)
shows how `admin.ecbp1100Status()` reports whether it is in force.

## When the node switches it off by itself

MESS is not unconditional. The node switches it off when either of these holds, and logs
`Disabled artificial finality features` with the reason:

- **`low peers`:** the node has fewer than five peers.
- **`stale safety interval`:** the node's newest block is more than 390 seconds old. The node checks
  on the same interval, so this happens roughly 6.5 to 13 minutes after the head stops advancing.

It switches MESS back on once the node is in sync with enough peers and no peer advertises a chain
with more total difficulty.

That is the limit of the defense. An eclipse attack, a network partition or a network-wide stall
can switch MESS off, and a heavier chain advertised by a peer keeps it off through the
reorganization. A node in that state follows the chain with the most total difficulty, as it would
without MESS. [Troubleshooting](troubleshooting.md#what-do-disabled-artificial-finality-features-and-reorg-disallowed-mean)
explains the log lines.

`--mess.nodisable` keeps MESS on through both conditions, so it also applies while the node is out
of sync or short of peers. Where the node would have switched it off, the log shows
`Preventing disable artificial finality`.

## Turn MESS off

```sh
geth --classic --mess=false
```

`--mess=false` moves the activation block out of reach. `geth dumpconfig` writes it into a config
file as:

```toml
[Eth]
OverrideECBP1100 = 18446744073709551614
```

## Turn MESS back on

Remove `--mess=false`, or pass `--mess`. `--mess` also overrides that line in a config file, and
`--mess.nodisable=false` overrides `ECBP1100NoDisable = true`.

## Flags and config file keys

| Flag | Older spelling, still accepted | Config file key, under `[Eth]` | Effect |
|---|---|---|---|
| `--mess=false` | none | `OverrideECBP1100 = 18446744073709551614` | Turns MESS off |
| `--mess.activate=<block>` | `--ecbp1100` | `OverrideECBP1100` | Sets the activation block, and wins over `--mess=false` |
| `--mess.deactivate=<block>` | `--override.ecbp1100.deactivate` | `OverrideECBP1100Deactivate` | Sets a deactivation block |
| `--mess.nodisable` | `--ecbp1100.nodisable` | `ECBP1100NoDisable = true` | Keeps MESS on, bypassing both automatic switch-offs |

A negative block number, or one of 2^64 or more, is refused when the flags are parsed.

## Change it on a running node

`admin_ecbp1100` sets the activation block in the running node. It is in the `admin` namespace,
which the node serves over IPC. At the next start, the flags and the config file decide again. Its
one parameter is a block number:

- a hex quantity such as `0x1`, up to `0x7fffffffffffffff`. A decimal number is refused, and so is
  the `--mess=false` value;
- `earliest` for block 0, or `latest` and `pending` for the current head. `finalized` and `safe` are
  refused.

Check the result with `admin.ecbp1100Status()`. The [admin module](../JSON-RPC-API/modules/admin.md)
lists both methods.
