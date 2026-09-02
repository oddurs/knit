---
title: Send files with the command
order: 2
---
# Send files with the command

```sh
knit run --dir -- make
knit run --sync -- make
```

`--dir` sends the current working directory to the target machine and runs the
command inside a copy of it. `--sync` does the same and, when the command
finishes, copies back every file it created or changed. `--sync` implies
`--dir`.

## What travels

The whole directory tree under your current directory: regular files and
folders, with their permissions. Symbolic links, sockets, and device files are
skipped. There is no ignore file; if the directory holds a large `node_modules`
or `.git`, that travels too, so run from the directory that holds only what the
command needs.

The tree is streamed straight into a temporary directory on the target, not
buffered, so it is bounded by the link speed rather than memory. The temporary
directory is deleted when the command ends.

## What comes back with --sync

Files whose content differs from what was sent, and files that did not exist
before. knit compares content hashes taken before and after the run, so a
command that rewrites a file with identical bytes sends nothing back. Deleted
files are not deleted here.

Changed files are written into your current directory, overwriting the local
copy. Run `--sync` from a directory where that is what you want.

## When to use which

| You want | Use |
| -------- | --- |
| a command that only needs stdin | plain `knit run` |
| a command that reads files but produces output on stdout | `--dir`, redirect stdout |
| a build, conversion, or script that writes files | `--sync` |

```sh
# Compile on the fast machine, get the binary back
knit run --on studio --sync -- go build -o app .

# Transcode with inputs from here, output to stdout
knit run --dir -- ffmpeg -i in.mov -f matroska - > out.mkv
```
