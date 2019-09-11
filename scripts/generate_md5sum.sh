#!/bin/bash

for archive in artifacts/*.zip; do
  md5sum $archive >> artifacts/md5sums.txt
done
