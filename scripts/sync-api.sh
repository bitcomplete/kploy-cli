#!/usr/bin/env bash
# Pull the canonical api.yaml from bitcomplete/kploy and regenerate the
# Go client. Run this whenever the kploy server's API surface changes;
# commit the resulting diff (api.yaml + client/client.gen.go).
#
# Requires the GitHub CLI (`gh`) authenticated against an account with
# read access to bitcomplete/kploy.
set -euo pipefail
cd "$(dirname "$0")/.."

gh api -H 'Accept: application/vnd.github.v3.raw' \
    repos/bitcomplete/kploy/contents/api.yaml > api.yaml

go generate ./client/...

if git diff --quiet; then
    echo "api.yaml already up to date."
else
    echo "Updated api.yaml and regenerated client/. Review the diff and commit."
fi
