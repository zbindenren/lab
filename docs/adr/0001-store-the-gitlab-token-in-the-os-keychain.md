# 1. Read the GitLab token from the OS keychain

## Decision

`lab` resolves its GitLab token in a fixed order: `GITLAB_TOKEN`, then the OS keychain (service `lab:<host>`, empty account), then `gitlab.token` in the config file, and finally a hard error naming all three ways to provide one. The environment is read directly instead of through the config file, because `viper` is fed the file after `envsubst` expansion: `token = "$GITLAB_TOKEN"` and a hand-typed literal arrive as the same string, so a parsed config can no longer tell "came from the environment" from "typed into the file", and the environment has to be able to outrank a stored credential. The file token ranks last because the keychain exists to get long-lived tokens out of plaintext: `lab auth login` must take effect for a user whose file already carries a token, and after `lab auth logout` that file token becomes the fallback again. `glab`'s own keychain entries (`glab:<host>:token`) are not read: when logged in over OAuth, `glab` stores a short-lived access token that `lab` cannot refresh, and `glab auth logout` deletes the entry, so the reuse fails silently for the users who have one and does nothing for the users who do not.

## Considered Options

- Keep the config file as the only token source and document `GITLAB_TOKEN` more loudly. Rejected: it leaves a long-lived token in the shell environment or in a plaintext file, which is the problem this change addresses.
- Fall back to the keychain only when the config file has no token. Rejected: a literal file token would then silently shadow the credential `lab auth login` just stored, which is the surprise this ladder exists to avoid.
- Reuse `glab`'s keychain entries. Rejected for the staleness and deletion reasons above.

## Consequences

- A literal token in the config file is outranked by a keychain entry for the same host without any warning. `lab auth status` is how a user sees which source won.
- The keychain service keeps the port and drops the path, so two GitLab instances on one host under different subfolder paths share a single entry. This matches `glab`, which keeps the subfolder in a separate config key rather than in the entry name.
- A keychain that is locked, denied, or absent (headless Linux without a Secret Service) is a hard error naming the service, not a fallback to the file token, so a credential problem is not reported as a GitLab permission problem.
- The order is covered by table-driven tests in `internal/token_test.go`, so changing the precedence is a failing test rather than a silent behaviour change.
