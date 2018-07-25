#!/bin/bash

if [ $# != 1 ]; then
  echo Specify the version to release.
  exit 1
fi

sed -i "" -e "/var version =/ s/\".*\"/\"$1\"/" main.go || exit 1
git add main.go
git commit -m "Release $1" || exit 1

git tag -a "$1" || exit 1
git push --tags