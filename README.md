# rido

Keep sensitive files out of coding sandboxes without changing your workflow.

`rido` moves files outside your repository and leaves symlinks behind. Your local environment keeps
working normally, but isolated agents that cannot access the private store simply see missing files.

## Why

Telling an agent "don't read secrets" is not a security boundary.

Configuration files, prompt rules, and allow-lists are only suggestions. A compromised dependency, a
prompt injection, or a curious agent can still access anything available in its filesystem.

### `rido` takes a different approach

Before rido

```
repo/
├── .env
├── config/
│   └── credentials.json
└── src/
```

After rido

```
repo/
├── .env -> ~/.rido/store/01J8XQ4M7K/.env
├── config/
│   └── credentials.json -> ~/.rido/store/01J8XQ52PB/credentials.json
└── src/

~/.rido/store/
├── 01J8XQ4M7K/
│   ├── .env
│   └── meta.json
└── 01J8XQ52PB/
    ├── credentials.json
    └── meta.json
```

One directory per entry, named with a ULID. `meta.json` records where the file came from.

This is what your agent see inside an isolated sandbox:

```
.env -> ENOENT
```

This is what you see on your local machine:

```
.env -> real file
```

## Installation

```sh
go install github.com/marftn/rido@latest
```

## Usage

### Add existing files:

```bash
rido add .env config/credentials.json
```

### Add a whole directory:

```bash
rido add secrets/
```

The tree moves into one entry, unchanged, and the directory itself becomes the symlink:

```
repo/
└── secrets -> ~/.rido/store/01J8XQA1/secrets

~/.rido/store/01J8XQA1/
├── meta.json
└── secrets/
    ├── db.json
    └── prod/
        └── api.key
```

Subdirectories do not get entries of their own. Because the whole path is hidden, files created
inside `secrets/` later are covered too, with no re-add.

Add the files individually when you only want some of them hidden, or when a build script needs to
write into that directory:

```bash
rido add secrets/db.json secrets/prod/api.key
```

`rido` refuses to add a file that git already tracks, and offers to add it to `.gitignore` if it is
untracked.

### Restore files back into the repository:

Sometimes your agent will overwrite some symlinks because it needs valid files to execute your code.
Simply restore them afterward.

```bash
rido restore .env      # one file
rido restore --all     # everything under the current directory
```

If a real file sits where the symlink should be, `rido` asks before deleting it.

### List managed files:

```bash
rido list
```

```
ID          STATUS    ADDED       ORIGIN
01J8XQ4M7K  linked    2026-08-07  /home/you/code/myrepo/.env
01J8XQ6F3D  MISSING   2026-08-07  /home/you/code/myrepo/.env.staging
```

### Stop managing a file:

Puts the file back in the repository and drops the store entry.

```bash
rido revert .env
```

Note this is the opposite of `git restore` / `git revert`: here `restore` recreates the symlink,
`revert` hands the file back.

## Configuration

Optional. `~/.config/rido/config.json`:

```json
{ "storeRoot": "/media/luks/rido-store" }
```

Defaults to `~/.rido/store`.

## Limitations

`rido` protects against processes running in restricted environments that cannot access the private
store.

It does **not** protect against:

- a process that already has access to your home directory
- malware running as your user
- someone with filesystem access to the private store

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
