#!/usr/bin/env bash

set -e


FILES_NEED_FORMATTING=$(find . -name "*.go" | xargs golines -l)
if [[ -z $FILES_NEED_FORMATTING ]]
then
    exit 0
else
    echo "Files are unformatted, need fix"
    echo
    echo $FILES_NEED_FORMATTING
    echo
    echo "run command to fix: find . -name '*.go' -exec golines -w {} \\;"
    exit 1
fi
