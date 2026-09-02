---
title: Documentation
---
# knit documentation

knit turns the machines around you into one pool of compute. Start an agent on
each, pair them once, and `knit run -- <cmd>` executes wherever there is the
most headroom, with stdin, stdout, stderr, and the exit code behaving exactly as
if it ran here.

**New here?** [Install](getting-started/install.md), then do the
[two-machine quickstart](getting-started/quickstart.md). It takes about two
minutes, most of which is copying a key.

**Looking for a flag?** The [command reference](reference/commands.md) has all
eight commands. That is the whole surface; there is no configuration file.

**Running models?** [AI workloads](guides/ai-workloads.md) has copy-paste
launches for MLX distributed, torchrun, and llama.cpp RPC.

Before you hand the key to another machine, read
[Security](trust/security.md): every connection is encrypted and both ends
prove the key, and holding the key is a shell on every machine in the fabric.
