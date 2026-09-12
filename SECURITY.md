# Security Policy

## Reporting a vulnerability

**Do not open a public issue for a security vulnerability.**

Report it privately through GitHub's private vulnerability reporting on this
repository:

<https://github.com/ethereumclassic/core-geth/security/advisories/new>

That channel is enabled and reaches this project's maintainers. It needs a GitHub
account and nothing else — no key exchange and no mailing list.

Include what you can: the affected version or commit, the network, what an
attacker gains, and a reproduction if you have one. A report without a
reproduction is still worth sending.

### What to expect

Reports are acknowledged and triaged privately. Where a fix is warranted it is
prepared under a GitHub security advisory and disclosed once a release carrying
it is available. Reporters are credited unless they ask not to be.

## Automated and AI-assisted review

**Findings from automated or agent-assisted review go through the private channel
above, the same as any other.** Not a public issue, and not a pull request whose
diff describes the defect before a fix exists.

The reason is unchanged by how a finding was produced. What matters is whether it
is exploitable against nodes that are running now, and an automated review can
surface something exploitable as readily as a manual one. Volume does not change
the obligation: a report with fifty findings and one that is real is still a
report that should arrive privately.

Such reviews are welcome. This client carries a large inherited surface, and
systematic review of it is worth more to the project than the noise it costs to
triage.

## Publication and credit

**Once a finding is resolved and a release carrying the fix is available, the
review may be published in [`docs/audits/`](docs/audits/)** — as the March 2026
audit and its August follow-up already are.

That serves two purposes. It gives operators the per-release detail they need to
decide what they are exposed to, and it credits the work by name. Security review
of an under-resourced client is largely unpaid, and the record of who did it
should be public and durable rather than a line in a changelog.

**Our thanks to the white hat community.** Several of the defects closed in this
release series were found and reported by people with no obligation to look. Where
a reporter wants credit they are named; where they prefer not to be, they are not.

## Scope

This repository is Core-Geth, the Ethereum Classic execution client. Consensus
rules, chain configuration, peer-to-peer networking, the JSON-RPC surface, key
handling and the node binaries are all in scope.

Vulnerabilities in other Ethereum Classic software — other clients, explorers,
wallets, bridges or infrastructure — are outside this repository. Report those to
the project that maintains them.

## Core-Geth is a derivative of go-ethereum

A defect here may be inherited from upstream rather than specific to this client.
Where it affects code shared with
[go-ethereum](https://github.com/ethereum/go-ethereum), it is worth reporting
there as well: <https://github.com/ethereum/go-ethereum/security/policy>.

**Neither project forwards reports to the other.** Reporting to both is welcome
and is the right move for a shared defect.

## `geth version-check` checks go-ethereum, not this client

The `version-check` command queries go-ethereum's vulnerability feed at
`geth.ethereum.org` and matches advisories against the running version string.

**That feed does not track Core-Geth**, and Core-Geth's version numbering has
diverged from go-ethereum's, so its output is not meaningful here in either
direction: a `No vulnerabilities found` result says nothing about this client,
and a reported match may describe a defect this client never carried.

Do not treat that command as a clean bill of health for Core-Geth.
