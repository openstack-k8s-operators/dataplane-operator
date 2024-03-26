#!/usr/bin/env bash
set -ex

# Update docs from kustomize examples
docs/kustomize_to_docs.sh

# Sample to kuttl tests
COUNT=0
for sample in */assemblies/samples/* ; do
    FILENAME=$(basename $sample)
    mkdir -p tests/kuttl/tests/docs-test-0${COUNT}
    sed "0,/----/d" $sample | sed -e '/----/,$d' > tests/kuttl/tests/docs-test-0${COUNT}/01-${FILENAME/adoc/yaml}
    COUNT=$((COUNT + 1))
done
