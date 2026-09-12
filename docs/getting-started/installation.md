---
title: Installation
---

There are three ways to get `geth`: download a pre-built archive, build a
Docker image from this source, or build the binary from source. Building from
source is covered on its own page, [Build from Source](../developers/build-from-source.md).

!!! warning "Install from this repository only"
    Core-Geth moved to [`ethereumclassic/core-geth`](https://github.com/ethereumclassic/core-geth).
    Archives and images published under the previous `etclabscore` namespace are
    not built from this source and do not carry the fixes released here. Do not
    install from them.

## Pre-built executable

Binary archives are attached to each tagged release on this repository's
[releases page](https://github.com/ethereumclassic/core-geth/releases). Download the
archive for your platform, verify it against its checksum, unarchive it, and run
the binary.

Each release publishes two archives per platform:

| Archive | Contains |
| --- | --- |
| `core-geth-<platform>-<tag>.zip` | the `geth` node binary alone |
| `core-geth-alltools-<platform>-<tag>.zip` | `geth` plus the other tools built from this source |

`<platform>` is one of:

| Platform | Machine |
| --- | --- |
| `linux` | Linux, x86_64 |
| `osx` | macOS, Intel |
| `osx-arm64` | macOS, Apple Silicon |
| `win64` | Windows, x86_64 |
| `arm64` | Linux, 64-bit ARM |
| `arm5`, `arm6`, `arm7` | Linux, 32-bit ARM, by ARM architecture version |
| `arm` | deprecated; identical to `arm5`. Use `arm5` |

Every archive is published alongside a `.sha256` file. Verify before you run:

```shell
$ sha256sum -c core-geth-linux-v1.13.0.zip.sha256
$ unzip core-geth-linux-v1.13.0.zip
$ ./geth --help
```

On macOS use `shasum -a 256 -c` in place of `sha256sum -c`.

### Verify where the archive came from

**A checksum proves the download was not corrupted. It does not prove what built the
file**, because whoever publishes the archive publishes the checksum beside it. From
v1.13.0 each archive also carries a build attestation, which records the workflow,
commit and ref that produced it:

```shell
$ gh attestation verify core-geth-linux-v1.13.0.zip --repo ethereumclassic/core-geth
```

That needs the [GitHub CLI](https://cli.github.com/). It succeeds only for an artifact
built by this repository's release workflow — an archive from anywhere else fails, as
does one whose bytes have changed. No signing key is involved: the attestation is minted
against a short-lived certificate issued to the workflow run itself, so there is no key
to steal or rotate.

Worth doing once on any binary you are about to run on a node that holds value.

## With Docker

Images are published to the GitHub Container Registry for each tagged release,
built for `linux/amd64` and `linux/arm64`:

```shell
$ docker pull ghcr.io/ethereumclassic/core-geth:latest
```

Tags mirror the release tags. `ghcr.io/ethereumclassic/core-geth:v1.13.0` is a
specific release; `latest` follows the most recent non-prerelease. The image
built from `Dockerfile.alltools`, containing the full tool set rather than the
node alone, is published under the same name with an `alltools-` prefix, as
`ghcr.io/ethereumclassic/core-geth:alltools-latest`.

!!! warning "Images under the previous namespace are not these images"
    Images published as `etclabscore/core-geth` on Docker Hub are not built from
    this source and receive nothing released here.

You can also build an image yourself — the `Dockerfile` produces an image
containing `geth`, and `Dockerfile.alltools` one containing the full tool
set:

```shell
$ git clone https://github.com/ethereumclassic/core-geth.git
$ cd core-geth
$ docker build -t core-geth .
```

Run it either way. The image's entry point is `geth`, so flags are passed
directly — substitute the published image name for `geth` below to run the
published one:

```shell
$ docker run -d \
    --name core-geth \
    -v $LOCAL_DATADIR:/root \
    -p 30303:30303 -p 30303:30303/udp \
    -p 127.0.0.1:8545:8545 \
    core-geth \
    --classic \
    --http --http.addr 0.0.0.0 --http.port 8545
```

That maps the devp2p port over both TCP and UDP, keeps chain data in
`$LOCAL_DATADIR` on the host so it survives the container, and reaches the
JSON-RPC endpoint from the host and nowhere else.

!!! warning "Both halves of that RPC line are deliberate"
    `--http` alone binds the listener to `localhost` **inside the container**, while
    Docker forwards a published port to the container's *external* interface — so
    `-p 8545:8545` with the default address publishes a port that reaches nothing, and
    the RPC appears dead with no error explaining why.

    `--http.addr 0.0.0.0` binds every interface inside the container's own network
    namespace, which is isolated. `-p 127.0.0.1:8545:8545` is what keeps it off the
    host's public interfaces. **Publishing as `-p 8545:8545` instead exposes your RPC
    on every interface of the host, which on a VPS is the open internet.**

    If you do not need RPC from the host at all, drop both the `-p 127.0.0.1:8545:8545`
    and the `--http*` flags.

The image also exposes 8546 for the WebSocket endpoint, which needs
`--ws --ws.addr 0.0.0.0` to be served.
