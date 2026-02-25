#!/usr/bin/env bash
set -euo pipefail

TAG="${1:-v1.0.1}"
IMAGE_NAME="${IMAGE_REGISTRY:-docker.io}/ericcaverly/spb"

docker build -t "${IMAGE_NAME}:${TAG}" .
docker push "${IMAGE_NAME}:${TAG}"
docker tag "${IMAGE_NAME}:${TAG}" "${IMAGE_NAME}:latest"
docker push "${IMAGE_NAME}:latest"
