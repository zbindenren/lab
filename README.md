# Lab

[![CI](https://github.com/Ackerr/lab/workflows/CI/badge.svg)](https://github.com/Ackerr/lab)
[![Go Report Card](https://goreportcard.com/badge/github.com/ackerr/lab)](https://goreportcard.com/report/github.com/ackerr/lab)
[![release](https://img.shields.io/github/v/release/ackerr/lab.svg)](https://github.com/ackerr/lab/releases)

[![ackerr/lab](https://res.cloudinary.com/marcomontalbano/image/upload/v1606925692/video_to_markdown/images/youtube--qqKW9SQqjF0-c05b58ac6eb4c4700831b2b3070cd403.jpg)](https://www.youtube.com/watch?v=qqKW9SQqjF0 "ackerr/lab")

A fuzzy finder command line tool for gitlab. [中文文档](./README-CN.md)

## Feature

```
lab sync        Sync gitlab projects
lab browser     Fuzzy find gitlab repo and open it in $BROWSER
lab clone       Fuzzy find gitlab repo and clone it
lab cs          Fuzzy find repo in your codespace
lab lint        Check .gitlab-ci.yml syntax
lab open        Open the current repo remote in $BROWSER
lab config      Use $EDITOR open config file, support custom config path, use --config filepath
```

For more information, please use `lab help`.

## Install

### homebrew

```bash
$ brew install ackerr/tap/lab
```

### scoop

```bash
$ scoop bucket add ackerr https://github.com/Ackerr/scoop-bucket
$ scoop install ackerr/lab
```

### go get

```bash
$ go get -u "github.com/ackerr/lab"
```

## Configure

Recommend use `lab config` to edit config file (`~/.config/lab/config.toml`), this command will open the config file use `$EDITOR`. If the file don't exist, it will auto generate by [config template](https://github.com/Ackerr/lab/blob/master/config.toml).

> `base_url` is required. The token comes from the OS keychain, from `GITLAB_TOKEN`, or from `token` in the config file, see [Token storage](#token-storage). The way to get gitlab token, see [this](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html#creating-a-personal-access-token)

`lab sync` collects the projects of the groups listed in `gitlab.groups` plus all their subgroups, and writes them to the projects file. The list defaults to `["linux", "kubernetes"]` when it is missing or empty, so a repo under any other top-level group is not synced until that group is added:

```toml
[gitlab]
groups = ["linux", "kubernetes", "infrastructure"]
```

Only the groups given are queried: a project that is visible to your token through a group you are not a member of, or through a group left out of the list, does not appear.

## Token storage

`lab auth login` stores a personal access token in the OS keychain, under `lab:<host>` for the host in `base_url`. It reads the token from a hidden prompt, checks it against the GitLab API and stores it only once the API accepts it. Pipe the token on stdin to provision a machine without a prompt.

```bash
$ lab auth login
Token:
authenticated as jane

$ printenv GITLAB_TOKEN | lab auth login
authenticated as jane
GITLAB_TOKEN is set and takes precedence over the token just stored
```

The second form has the variable set by construction, so `lab` warns that the entry it just stored is shadowed. Unset `GITLAB_TOKEN` for the stored token to take effect.

`lab auth status` shows the host, the source the active token came from and whether the API accepts it. Neither `status` nor any other command prints the token.

```bash
$ lab auth status
host: gitlab.example.com
source: keychain
authenticated as jane
```

`lab auth logout` deletes the entry and says whether one existed. A missing entry is not an error.

The token is taken from the first source that has one:

| # | Source | Notes |
|---|--------|-------|
| 1 | `GITLAB_TOKEN` | wins over everything, so CI jobs and one-off overrides (`GITLAB_TOKEN=... lab sync`) keep working |
| 2 | OS keychain | written by `lab auth login`, entry `lab:<host>` |
| 3 | `gitlab.token` | the config file, expanded from the environment before it is parsed |
| 4 | nothing | `lab` fails with a message naming the three ways to provide a token |

A missing keychain entry falls through to the next source. A keychain that cannot be read, for example a locked macOS keychain or a headless Linux box without a Secret Service, is an error naming the service instead of a silent fallback to the config file.

The keychain is the one the platform provides: Keychain on macOS, Secret Service on Linux, Credential Manager on Windows. There is no option to write the token back into the config file.
