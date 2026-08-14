# rido

Moves sensitive files out of a repo into a private store and leaves a symlink behind. A sandbox that
can't see the store gets ENOENT; local runs follow the link and work as before.

Not a security control. Anything that can read the store can read the secret, including a local
agent that follows the symlink.

## Store

```
$STORE/
  01J8XQ4M7K/
    .env                            payload, original filename
    meta.json                       {"origin": "/home/you/code/myrepo/.env", "filename": ".env", "v": 1}
```

One ULID-named directory per entry. Store root comes from `~/.config/rido/config.json`
(`{"store_root": "..."}`), defaulting to `~/.rido/store` if the file or the key is absent. Store dir
is `0700`.

Symlinks are always absolute. There is no log file: a ULID embeds its creation time in milliseconds,
which is where `list`'s ADDED column comes from, and ULIDs sort chronologically.

## Commands

| Command                              | Effect                                           |
| ------------------------------------ | ------------------------------------------------ |
| `rido add <path>…`                   | move file or dir into the store, leave a symlink |
| `rido list`                          | every entry: ID, status, added, origin           |
| `rido restore <path>…\|<id>…\|--all` | recreate a symlink something removed             |
| `rido revert <path>…\|<id>…\|--all`  | payload back in the repo, entry dropped          |

`restore` and `revert` take an ID as well as a path, since a STALE entry has no live path to name.

`--all` covers entries whose origin is under the cwd. `--store-wide` covers the whole store. Nothing
under the cwd is an error:

```
$ cd ~/code/myrepo && rido restore --all
  relinked  .env
  relinked  config/creds.json
  2 restored (3 store entries outside /home/you/code/myrepo untouched)

$ cd /tmp && rido restore --all
  no entries under /tmp  (5 in the store — use --store-wide)
  exit 1
```

`list` always covers the whole store.

## add

Git check first:

| State                     | Behaviour                                                     |
| ------------------------- | ------------------------------------------------------------- |
| tracked by git            | refuse, print the `git rm --cached` line, never touch history |
| untracked, not gitignored | offer to append the path to `.gitignore`, then proceed        |
| not a git repo            | proceed                                                       |

Then, per path:

```
1  copy   /repo/.env → $STORE/<ULID>/.env      mode preserved
2  verify sha256 match, else rm -rf entry and fail
3  park   /repo/.env → /repo/.env.rido-tmp
4  link   /repo/.env → $STORE/<ULID>/.env      absolute
5  rm     /repo/.env.rido-tmp
```

Failure at 3 or 4 rolls back: park moves back, entry is removed. Copy preserves mode, so `meta.json`
carries no mode field.

`os.Rename` is deliberately not used. The store is expected to live on a separate volume (LUKS),
where rename fails with `EXDEV`, so the copy path has to exist and be tested anyway; a rename fast
path would only add a second sequence to maintain. Plaintext therefore exists in two places between
steps 1 and 5.

`add` never overwrites an existing entry.

### Directories

A directory becomes a single entry, and the directory itself is the symlink, so files created inside
it later are hidden too.

```
$ rido add secrets/
  copy    3 files, 1 dir  → $STORE/01J8XQA1/secrets/
  verify  sha256 × 3     ok
  park    secrets → secrets.rido-tmp
  link    secrets → $STORE/01J8XQA1/secrets
  rm      secrets.rido-tmp
  added   secrets/  (3 files, 2.1 KB)
```

| Found while walking  | Treatment                                            |
| -------------------- | ---------------------------------------------------- |
| regular file         | copied, mode preserved, sha256 verified              |
| directory            | recreated, mode preserved, empty dirs kept           |
| symlink              | copied as a symlink; target not followed, not copied |
| socket, fifo, device | abort the whole add before the park, nothing moved   |
| hardlink             | copied as a separate file, link count not preserved  |

The park is one rename of the top directory, so rollback is one move whatever the tree size.

## restore

| At the origin                   | Behaviour                          |
| ------------------------------- | ---------------------------------- |
| nothing, or a dangling symlink  | relink, no prompt                  |
| anything that isn't our symlink | confirm, then delete it and relink |

Confirmation defaults to No. Without a TTY there is no prompt: conflicting paths are skipped,
listed, and the command exits 1. `-f/--force` answers yes.

```
$ rido restore --all
  relinked  .env.staging        (was missing)
  relinked  config/creds.json   (was dangling)
  .env is a regular file (412 B, modified 12m ago)
  delete it and relink? [y/N] y
  relinked  .env

$ rido restore --all < /dev/null
  relinked  .env.staging
  SKIPPED   .env  regular file; re-run with a terminal or -f
  exit 1
```

## revert

Payload goes back to its origin and the entry is dropped. Our own symlink at the origin is replaced
without a prompt; anything else goes through the same confirmation as `restore` (same default, same
`-f`, same non-TTY skip).

```
$ rido revert .env
  .env is a regular file (412 B, modified 12m ago)
  delete it and put the payload back? [y/N] y
  reverted  .env   (entry 01J8XQ4M7K dropped)
```

If the origin's directory is gone, `MkdirAll` recreates it, so no entry can be stranded in the
store:

```
$ rido revert 01J8V3R7QC
  origin directory is gone — recreating /home/you/code/oldrepo/
  reverted  /home/you/code/oldrepo/.env   (entry dropped)
```

## list

`linked` is lowercase, anything needing a decision is uppercase.

```
$ rido list
ID          STATUS    ADDED       ORIGIN
01J8XQ4M7K  linked    2026-08-07  /home/you/code/myrepo/.env
01J8XQ52PB  linked    2026-08-07  /home/you/code/myrepo/config/creds.json
01J8XQ6F3D  MISSING   2026-08-07  /home/you/code/myrepo/.env.staging
01J8XQ7T9W  OCCUPIED  2026-08-07  /home/you/code/myrepo/.env.local  (regular file, 412 B)
01J8V3R7QC  STALE     2026-06-14  /home/you/code/oldrepo/.env       (dir gone)
01J8XQ9R4X  BROKEN    2026-08-07  /home/you/code/myrepo/.env.ci     (payload missing from store)

6 entries: 2 linked, 3 need `rido restore`, 1 BROKEN
```

| Status     | Meaning                                                        | Fix                             |
| ---------- | -------------------------------------------------------------- | ------------------------------- |
| `linked`   | symlink at origin points at our payload                        | —                               |
| `MISSING`  | nothing at origin, its dir exists                              | `rido restore`                  |
| `OCCUPIED` | a regular file, or a symlink pointing elsewhere, is in the way | `rido restore`, prompts         |
| `STALE`    | the origin's dir is gone                                       | `rido revert <id>`, then re-add |
| `BROKEN`   | entry exists, payload missing from the store                   | none available                  |

A stale origin is never healed automatically. Re-point it by hand:

```
cd <new location> && rido revert .env && rido add .env
```

## Resolving an argument to an entry

1. Argument parses as a ULID → use it.
2. Symlink exists at the path → `readlink`, ID is `basename(dirname(target))`. Works with a stale
   `origin`, which is what makes the re-point above work.
3. Otherwise scan `$STORE/*/meta.json` for `origin == abspath(arg)`.

## Exit codes and partial failure

Each path is handled independently and is individually atomic, so a run never leaves a half-applied
state. A failure doesn't stop the run; every path prints one line and the command ends with a count.

```
$ rido add .env creds.json secrets/ /etc/shadow
  added     .env         01J8XQA1..
  added     creds.json   01J8XQA2..
  added     secrets/     01J8XQA3..  (3 files)
  REFUSED   /etc/shadow  permission denied
  3 added, 1 failed
  exit 1
```

`0` everything succeeded, `1` at least one path failed or was skipped. No per-failure-kind codes.

## Tests

Table-driven, against a temp store and temp repo.

- add: same-filesystem and forced `EXDEV`; sha256 mismatch rolls back; injected failure at the park
  and at the symlink each restore the starting state; tree containing an inner symlink, an empty
  dir, a fifo; git-tracked refusal; gitignore prompt.
- restore: missing, dangling, occupied with yes / no / no TTY / `-f`.
- revert: our symlink present, occupied origin, STALE with the dir gone.
- list: one entry per status, `BROKEN` included.
- resolution: `readlink` with a stale origin, `meta.json` scan, bare ID.
- `--all`: cwd prefix match, `--store-wide`, no match exits 1.
