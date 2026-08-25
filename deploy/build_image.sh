#!/usr/bin/env bash
# 本地构建镜像的快速脚本，避免在命令行反复输入构建参数。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_TAG="${1:-sub2api:latest}"

ensure_local_base_image() {
    local source_image="$1"
    local local_image="$2"

    if [ "${SUB2API_REFRESH_BASE_IMAGES:-0}" = "1" ]; then
        docker pull "${source_image}"
        docker tag "${source_image}" "${local_image}"
        return
    fi

    if docker image inspect "${local_image}" >/dev/null 2>&1; then
        return
    fi

    if ! docker image inspect "${source_image}" >/dev/null 2>&1; then
        docker pull "${source_image}"
    fi
    docker tag "${source_image}" "${local_image}"
}

NODE_IMAGE="${SUB2API_NODE_IMAGE:-node:24-alpine}"
GOLANG_IMAGE="${SUB2API_GOLANG_IMAGE:-golang:1.27.0-alpine}"
ALPINE_IMAGE="${SUB2API_ALPINE_IMAGE:-alpine:3.21}"
POSTGRES_IMAGE="${SUB2API_POSTGRES_IMAGE:-postgres:18-alpine}"

LOCAL_NODE_IMAGE="${SUB2API_LOCAL_NODE_IMAGE:-localhost/sub2api-base-node:24-alpine}"
LOCAL_GOLANG_IMAGE="${SUB2API_LOCAL_GOLANG_IMAGE:-localhost/sub2api-base-golang:1.27.0-alpine}"
LOCAL_ALPINE_IMAGE="${SUB2API_LOCAL_ALPINE_IMAGE:-localhost/sub2api-base-alpine:3.21}"
LOCAL_POSTGRES_IMAGE="${SUB2API_LOCAL_POSTGRES_IMAGE:-localhost/sub2api-base-postgres:18-alpine}"

ensure_local_base_image "${NODE_IMAGE}" "${LOCAL_NODE_IMAGE}"
ensure_local_base_image "${GOLANG_IMAGE}" "${LOCAL_GOLANG_IMAGE}"
ensure_local_base_image "${ALPINE_IMAGE}" "${LOCAL_ALPINE_IMAGE}"
ensure_local_base_image "${POSTGRES_IMAGE}" "${LOCAL_POSTGRES_IMAGE}"

proxy_args=()
if [ -n "${SUB2API_DOCKER_BUILD_PROXY:-}" ]; then
    proxy_args=(
        --build-arg "HTTP_PROXY=${SUB2API_DOCKER_BUILD_PROXY}"
        --build-arg "HTTPS_PROXY=${SUB2API_DOCKER_BUILD_PROXY}"
        --build-arg "http_proxy=${SUB2API_DOCKER_BUILD_PROXY}"
        --build-arg "https_proxy=${SUB2API_DOCKER_BUILD_PROXY}"
        --build-arg "NO_PROXY=${SUB2API_DOCKER_BUILD_NO_PROXY:-localhost,127.0.0.1,host.docker.internal,sub2api-postgres,sub2api-redis}"
    )
fi

docker build --pull=false -t "${IMAGE_TAG}" \
    --build-arg "NODE_IMAGE=${LOCAL_NODE_IMAGE}" \
    --build-arg "GOLANG_IMAGE=${LOCAL_GOLANG_IMAGE}" \
    --build-arg "ALPINE_IMAGE=${LOCAL_ALPINE_IMAGE}" \
    --build-arg "POSTGRES_IMAGE=${LOCAL_POSTGRES_IMAGE}" \
    --build-arg GOPROXY=https://goproxy.cn,direct \
    --build-arg GOSUMDB=sum.golang.google.cn \
    "${proxy_args[@]}" \
    -f "${REPO_ROOT}/Dockerfile" \
    "${REPO_ROOT}"
