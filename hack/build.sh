#!/bin/sh
set -e
[ -z "$DIST" ] && DIST=.bin

[ -z "$VERSION" ] && VERSION=$(cat VERSION)
[ -z "$BUILD_TIME" ] && BUILD_TIME=$(TZ=GMT date "+%Y-%m-%d_%H:%M_GMT")
[ -z "$VCS_COMMIT_ID" ] && VCS_COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null)

pkg="github.com/codefresh-io/hermes/pkg"

echo "VERSION: $VERSION"
echo "BUILD_TIME: $BUILD_TIME"
echo "VCS_COMMIT_ID: $VCS_COMMIT_ID"

go_build() {
  [ -d "${DIST}" ] && rm -rf "${DIST:?}/*"
  [ -d "${DIST}" ] || mkdir -p "${DIST}"
  CGO_ENABLED=0 go build \
    -ldflags "-s -w -X $pkg/version.SemVer=${VERSION} -X $pkg/version.GitCommit=${VCS_COMMIT_ID} -X $pkg/version.BuildTime=${BUILD_TIME}" \
    -o "${DIST}/hermes" ./cmd
}

go_build
