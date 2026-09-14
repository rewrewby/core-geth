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

## Start here

| If you are | Go to |
| --- | --- |
| **Upgrading from v1.12.x** | [Migration guide](tutorials/v1.13.0-migration.md) — read this before you upgrade |
| Running a node for the first time | [Installation](getting-started/installation.md), then [Command line](getting-started/run-cli.md) |
| Building it yourself | [Build from source](developers/build-from-source.md) |
| Reviewing what was audited | [The four audit reports](#what-was-audited) |

!!! danger "The v1.12.x line is insecure — upgrade to v1.13.0"
    Six CVEs and a GraphQL denial of service are documented against it, one of them
    exploited against Ethereum Classic bootnodes in March 2026.

    | Identifier | Severity | What it allows |
    | --- | --- | --- |
    | [CVE-2026-22862](audits/2026-03-security-audit.md#cve-2026-22862-ecies-elliptic-curve-integrated-encryption-scheme-decrypt-ciphertext-length-undercheck) | High | Any peer crashes the node during the RLPx handshake. **Exploited against Ethereum Classic bootnodes, 18 March 2026.** |
    | [CVE-2026-26315](audits/2026-03-security-audit.md#cve-2026-26315-ecies-generateshared-accepts-unvalidated-public-key) | High | Repeated handshakes leak bits of the node's own P2P key |
    | [CVE-2026-26314](audits/2026-03-security-audit.md#cve-2026-26314-secp256k1-isoncurve-field-boundary-bypass) | High | secp256k1 coordinate field boundary bypass |
    | [CVE-2025-24883](audits/2026-03-security-audit.md#cve-2025-24883-off-curve-public-key-in-unmarshalpubkey) | High | Off-curve public key accepted by `UnmarshalPubkey` |
    | [CVE-2026-26313](audits/2026-03-security-audit.md#cve-2026-26313-p2p-rlp-recursive-length-prefix-item-count-memory-exhaustion) | High | One crafted p2p message exhausts the node's memory |
    | [CVE-2026-22868](audits/2026-03-security-audit.md#cve-2026-22868-kzg-kate-zaverucha-goldberg-blob-proof-verification-dos) | Medium | KZG proof verification denial of service |
    | [GraphQL query depth](audits/2026-03-security-audit.md#graphql-query-depth-dos) | Medium | Unbounded GraphQL query nesting; no CVE identifier assigned |

    Every release in the line is also built on Go 1.21, end of life since August
    2024. The [migration guide](tutorials/v1.13.0-migration.md) covers the upgrade,
    the required node-key rotation, and how to roll back.

## Supported networks

| Network | Chain ID | Consensus | Flag |
| --- | --- | --- | --- |
| Ethereum Classic | 61 | Proof of Work (Etchash) | `--classic`, `--mainnet` or no flag |
| Mordor testnet | 63 | Proof of Work (Etchash) | `--mordor` |
| MintMe.com Coin | 24734 | Proof of Work | `--mintme` |
| Private chains | configurable | PoW / PoA | genesis configuration |

Ethereum Classic runs every hard fork from Frontier through Spiral. The
[README's consensus table](https://github.com/ethereumclassic/core-geth#etc-consensus-history)
lists each upgrade with its activation block and the EIPs it included; `params/` in the
source is the authority, and a fork schedule recalled from memory is a guess.

**With no network flag, `geth` runs Ethereum Classic mainnet**, and `--mainnet` is the
same as `--classic`. A data directory initialized with a private network's genesis keeps
running that network.

### Networks this client does not support

**Ethereum mainnet, Sepolia and Holesky are not supported.** This client implements
Ethereum upgrades only through Cancun, so it cannot follow any of them. `--ethereum`,
`--sepolia` and `--holesky` are deprecated and refuse to start, rather than start a node
that would fall off its chain.

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
source and do not carry the fixes released here.

## What was audited

Four reports, each measuring a different layer. Together they are the evidence for the
paragraph above — no single one of them answers "what differs".

| Report | What it measures |
| --- | --- |
| [March 2026 security audit](audits/2026-03-security-audit.md) | The six CVEs and the GraphQL denial of service, with the per-release breakdown |
| [August 2026 security follow-up](audits/2026-08-security-followup.md) | `v1.12.23` measured at the tag against the advisory records — what it fixed and what it left open |
| [Dependency and toolchain modernization](audits/2026-08-dependency-modernization.md) | What changed underneath the code between the December 2024 archive and this release: the Go toolchain, 83 modules, the linter |
| [Release artifacts](audits/2026-09-release-pipeline.md) | What the published archives actually contain — platform floors, architectures, provenance |

The first three describe the source. **The last one describes the files you download**,
and the two can disagree: a release is not what the build configuration says it builds.
