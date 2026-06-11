# kploy CLI

Command-line client for [Kploy](https://github.com/bitcomplete/kploy).

## Install

```sh
brew install bitcomplete/tap/kploy
```

Or download a binary from the [releases page](https://github.com/bitcomplete/kploy-cli/releases).

## Authenticate

```sh
kploy auth login
```

Runs GitHub Device Flow: prints a code, opens the verification URL, polls until you approve the kploy GitHub App in your browser. Tokens are persisted at `~/.config/kploy/config.yaml` (mode `0600`).

```sh
kploy auth whoami    # show the orgs your token can see
kploy auth logout    # forget the saved token
```

## Validating `kploy.yaml`

If your project has a `kploy.yaml` at its repo root, you can sanity-check it locally before pushing:

```sh
kploy validate-config              # defaults to ./kploy.yaml
kploy validate-config -f my.yaml   # different path
```

Prints rendered hostnames for production and development on success — plus an example preview env if preview environments are enabled. Validation errors include the field path. Does not require authentication.

## Common workflows

```sh
# Pick a default org so you can drop --org from later commands.
export KPLOY_ORG=bitcomplete

kploy org list
kploy repo list
kploy env list --repo my-service
kploy env get  --repo my-service production
kploy env create \
    --repo my-service --name staging \
    --cluster <cluster-id> --branch staging --namespace my-service-staging \
    --tracked-image my-registry/my-service

kploy image list --repo my-service --env staging
kploy image add  my-registry/sidecar --repo my-service --env staging
kploy image remove my-registry/sidecar --repo my-service --env staging

kploy cluster list
kploy cluster create   # prints a one-shot bearer token — save it!

kploy deploy list --repo my-service --env staging
kploy deploy logs 1234567 --repo my-service
```

## Configuration

`~/.config/kploy/config.yaml`:

```yaml
server: https://kploy.app
org: bitcomplete
access_token: ghu_...
refresh_token: ghr_...
expiry: 2026-08-13T12:34:56Z
```

Environment variables override the file:

| Var            | Effect                                                       |
| -------------- | ------------------------------------------------------------ |
| `KPLOY_SERVER` | Kploy server URL (defaults to production)                    |
| `KPLOY_ORG`    | Default org for commands that take `--org`                   |
| `KPLOY_TOKEN`  | Reserved; not currently consumed (login writes the file)     |

## Output formats

`--output json` on any `list` / `get` / `create` command produces a JSON value suitable for piping to `jq`.

## Building from source

```sh
go build -o kploy .
```

CI builds via [GoReleaser](https://goreleaser.com/) (`.goreleaser.yaml`). Tagging `vX.Y.Z` on `main` cuts a release.

## Updating the API spec

`api.yaml` is a copy of the canonical spec maintained in [`bitcomplete/kploy`](https://github.com/bitcomplete/kploy/blob/main/api.yaml). When the server's API changes, run:

```sh
./scripts/sync-api.sh
```

This pulls the latest `api.yaml` from kploy, regenerates `client/client.gen.go`, and reports whether anything changed. Commit the diff with a `sync(api): …` message.
