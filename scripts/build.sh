#!/usr/bin/env bash
set -euo pipefail

REGISTRY="ghcr.io/pennomi"
NAME="rutherford"
VERSION="0.1.0"

docker build --no-cache -t "$REGISTRY/$NAME:$VERSION" -t "$REGISTRY/$NAME:latest" .
docker push "$REGISTRY/$NAME:$VERSION"
docker push "$REGISTRY/$NAME:latest"

helm package ./chart
helm push "$NAME-$VERSION.tgz" "oci://$REGISTRY"
