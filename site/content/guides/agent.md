---
title: Keep the agent running
order: 7
---
# Keep the agent running

```sh
knit up -d          # start in the background, until you stop it or log out
knit up --forever   # start at every login, restart if it stops
knit down           # stop it, either way
```

## Foreground, background, or forever

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

## Surviving reboots: --forever

```sh
knit up --forever
```

installs a launchd agent (macOS, `~/Library/LaunchAgents/io.knit.agent.plist`)
or a systemd user unit (Linux, `~/.config/systemd/user/knit.service`) that
starts the agent at login, restarts it if it ever stops, and logs to the same
`~/.knit/agent.log`. A detached agent that was already running is replaced,
so there is one agent, not two.

`knit down` stops it and removes the unit; `knit up -d` while the unit is
installed just tells you so. On Linux, knit also asks `loginctl` to keep your
user session alive across logouts so the agent runs at boot without a login;
that may require your password once, and is best effort.

## Upgrading

Stop the agent, install the new version, start it again:

```sh
knit down && brew upgrade knit && knit up -d
```

Agents and clients from the same minor release work together. An agent that is
too old for a client refuses it with a clear message rather than misbehaving.
