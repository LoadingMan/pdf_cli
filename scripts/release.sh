#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

VERSION=$(node -p "require('./package.json').version")
NAME="pdf-cli"
MODULE="pdf-cli"
DATE=$(date +%Y-%m-%d)
LDFLAGS="-s -w -X ${MODULE}/internal/build.Version=v${VERSION} -X ${MODULE}/internal/build.Date=${DATE}"

DIST="${REPO_ROOT}/dist"
rm -rf "${DIST}"
mkdir -p "${DIST}"

# platform/arch matrix must match scripts/install.js
TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for target in "${TARGETS[@]}"; do
  read -r GOOS GOARCH <<< "${target}"
  ext=""
  if [ "${GOOS}" = "windows" ]; then
    ext=".exe"
  fi

  build_dir=$(mktemp -d)
  binary="${build_dir}/${NAME}${ext}"

  echo ">> building ${GOOS}/${GOARCH}"
  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build -trimpath -ldflags "${LDFLAGS}" -o "${binary}" .

  archive_base="${NAME}-${VERSION}-${GOOS}-${GOARCH}"
  if [ "${GOOS}" = "windows" ]; then
    archive="${DIST}/${archive_base}.zip"
    ( cd "${build_dir}" && zip -q "${archive}" "${NAME}${ext}" )
  else
    archive="${DIST}/${archive_base}.tar.gz"
    tar -czf "${archive}" -C "${build_dir}" "${NAME}${ext}"
  fi

  rm -rf "${build_dir}"
  echo "   -> ${archive}"
done

echo
echo "Built artifacts for v${VERSION}:"
ls -lh "${DIST}"
