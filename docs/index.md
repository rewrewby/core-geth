---
hide:
  - navigation # Hide navigation
title: Core-Geth
---

# Core-Geth: the Ethereum Classic execution client

> A [go-ethereum](https://github.com/ethereum/go-ethereum) downstream that keeps chain
> configuration data-driven rather than hard-coded, so one binary serves Ethereum
> Classic, Mordor and private chains.

**Upstream go-ethereum has removed support for Ethereum Classic.** ETC consensus rules
are maintained here rather than inherited, which is what this client is for.

<div class="grid cards" markdown>

- **Running a node**

    [Install](getting-started/installation.md) · [Command line](getting-started/run-cli.md)

- **Upgrading from v1.12.x**

    [Migration guide](tutorials/v1.13.0-migration.md) — **read this before upgrading**

- **Building it yourself**

    [Build from source](developers/build-from-source.md)

- **What was audited**

    [Security and release audits](audits/2026-03-security-audit.md)

</div>

## Supported networks

| Network | Chain ID | Consensus | Flag |
| --- | --- | --- | --- |
| Ethereum Classic | 61 | Proof of Work (Etchash) | `--classic` |
| Mordor testnet | 63 | Proof of Work (Etchash) | `--mordor` |
| MintMe.com Coin | 24734 | Proof of Work | `--mintme` |
| Private chains | configurable | PoW / PoA | genesis configuration |

Ethereum Classic runs every hard fork from Frontier through Spiral. The
[README's consensus table](https://github.com/ethereumclassic/core-geth#etc-consensus-history)
lists each upgrade with its activation block and the EIPs it included; `params/` in the
source is the authority, and a fork schedule recalled from memory is a guess.

### Networks this client registers but does not maintain

**Ethereum mainnet, Sepolia and Holesky are inherited from upstream and are not
maintained here** — and `--mainnet` is still what a bare invocation selects. This client
implements Ethereum through Cancun and no further, so a node pointed at one of them
follows the real chain until the next fork it does not know about, then continues on its
own rules **without reporting anything**. They are scheduled for removal.

Pass `--classic` or `--mordor` explicitly.

**MintMe is carried deliberately.** `--mintme` works, and this release includes the
MintMe hardfork enabling PUSH0 and MCOPY. It is scheduled for deprecation in a later
release; it is here so that community has a modernized client to build from rather than
a fork of an abandoned one.

## How long this client is for

**The v1.13 series is the last for Core-Geth.** It exists to close the security gap and
carry Ethereum Classic on a supported Go toolchain while the client is retired — not to
begin a new line of development. Plan on that horizon.

On the wire this client speaks `eth/68`, and that is the version it will serve until it
is retired. `eth/69` and later reach Ethereum Classic through [Fukuii](https://fukuii.org)
and through the Ethereum Classic extensions maintained against mainstream Ethereum
clients.

## Where releases come from

Releases are published from
[`ethereumclassic/core-geth`](https://github.com/ethereumclassic/core-geth/releases).
Archives published under the previous `etclabscore` namespace are not built from this
source and do not carry the fixes released here — see the
[release artifacts audit](audits/2026-09-release-pipeline.md) for what measurably
differs between them.
