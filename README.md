# rido

Keep sensitive files out of coding sandboxes without changing your workflow.

`rido` moves files outside your repository and leaves symlinks behind. Your local environment keeps working normally, but isolated agents that cannot access the private store simply see missing files.

The protection comes from filesystem capabilities, not instructions: an agent cannot read what it cannot reach.

## Why

Telling an agent "don't read secrets" is not a security boundary.

Configuration files, prompt rules, and allow-lists are only suggestions. A compromised dependency, a prompt injection, or a curious agent can still access anything available in its filesystem.

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
├── .env -> ~/.rido/store/.env
├── config/
│   └── credentials.json -> ~/.rido/store/config/credentials.json
└── src/

~/.rido/store/
├── .env
└── credentials.json
```

This is what your agent see inside an isolated sandbox:

```
.env -> ENOENT
```

This is what you see on your local machine:

```
.env -> real file
````

## Usage

### Add existing files:

```bash
rido add .env config/credentials.json
````

### Create new managed files:

```bash
rido create .env
```

### Restore files back into the repository:

Sometimes your agent will overwrite some symlinks because it needs valid files to execute your code. Simply restore them afterward.

```bash
rido restore .env
```

### List managed files:

```bash
rido list
```

## Limitations

`rido` protects against processes running in restricted environments that cannot access the private store.

It does **not** protect against:

* a process that already has access to your home directory
* malware running as your user
* someone with filesystem access to the private store
