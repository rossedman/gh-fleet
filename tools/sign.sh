#!/usr/bin/env sh
# Sign a release artifact with keyless cosign: sign.sh <bundle> <artifact>.
# Sigstore's transparency log sometimes answers slowly; cosign then retries
# its own upload and gets 409 "an equivalent entry already exists" for the
# entry it just made. A fresh cosign run signs with a new ephemeral key, so
# it can't collide: retry the whole command a few times before failing.
set -u
bundle=$1
artifact=$2
for attempt in 1 2 3; do
  if cosign sign-blob --bundle="$bundle" "$artifact" --yes; then
    exit 0
  fi
  if [ "$attempt" -lt 3 ]; then
    echo "cosign sign-blob failed (attempt $attempt of 3); retrying in $((attempt * 15))s" >&2
    sleep $((attempt * 15))
  fi
done
echo "cosign sign-blob failed 3 times" >&2
exit 1
