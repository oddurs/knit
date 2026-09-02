---
title: Install
order: 1
---
# Install

knit is one static binary for macOS and Linux, Apple silicon and x86. Install
it on every machine you want in the fabric.

## macOS

```sh
brew install oddurs/tap/knit
```

## Linux

Download the archive for your architecture from the
[releases page](https://github.com/oddurs/knit/releases) and put the binary on
your `PATH`:

```sh
curl -L https://github.com/oddurs/knit/releases/latest/download/knit_0.3.0_linux_arm64.tar.gz | tar xz knit
sudo install knit /usr/local/bin/knit
```

Replace `arm64` with `amd64` on x86 machines, and the version with the latest
one listed. The archive contains the binary, the license, and a README.

## From source

With Go 1.26 or newer:

```sh
go install github.com/oddurs/knit/cmd/knit@latest
```

A binary built this way reports `knit dev` for its version; that is expected.

## Check

```sh
knit --version
```

Every machine in a fabric should run the same version. Mixed versions work
across a minor release, and an agent that is too old to understand a newer
client says so plainly instead of misbehaving.

Next: [two machines in two minutes](quickstart.md).
