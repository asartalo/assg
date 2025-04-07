#!/bin/sh

git ls-files -cdmo --exclude-standard | entr -dc go test ./...

