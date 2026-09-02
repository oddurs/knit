---
title: Keep the agent running
order: 7
---
# Keep the agent running

```sh
knit up -d      # start in the background
knit down       # stop it
```

## Foreground or background

`knit up` alone runs the agent in your terminal and logs to it; Ctrl-C stops
it. `knit up -d` detaches: the agent keeps running after the terminal closes,
logs to `~/.knit/agent.log`, and records its process id in `~/.knit/agent.pid`
so `knit down` can find it.

Running `knit up -d` when the agent is already up does nothing except tell you
its pid. Running `knit down` when it is not up says so and exits 0. Both are
safe to put in scripts.

## What the agent does

Advertises the machine, answers capacity probes, and runs commands sent by
holders of the key, as the user who started it, in that user's home directory
unless the command brought its own directory. It has no configuration.

## The first time on macOS

macOS may ask whether to allow `knit` to accept incoming connections. Allow it;
that is the agent listening.

## After a reboot

The agent does not restart itself yet. Run `knit up -d` again, or add it to
whatever starts things on login. A built-in `knit up --forever` that installs a
launchd or systemd unit is planned.

## Upgrading

Stop the agent, install the new version, start it again:

```sh
knit down && brew upgrade knit && knit up -d
```

Agents and clients from the same minor release work together. An agent that is
too old for a client refuses it with a clear message rather than misbehaving.
