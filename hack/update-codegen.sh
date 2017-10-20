#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ${GOPATH}/src/k8s.io/code-generator)}

vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/IanLewis/memcached-operator/pkg/client github.com/IanLewis/memcached-operator/pkg/apis \
  ianlewis.org:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/hack/copyright-header.go.txt

# Fix results of bug that creates case-insensitive imports 
find ./pkg/client -type f -print0 | xargs -0 sed -i '' 's/github\.com\/ianlewis/github\.com\/IanLewis/g'
