# dutil
Docker command-line utility

## Install

```bash
go get -u -v github.com/djmarkoz/dutil
```

## Features

1. Batch re-tag one or more Docker images with a new repository and/or version
2. Pushes the re-tagged Docker images directly (in parallel) to the new repository

## Usage

```
% dutil retag -h
Change the repository of existing Docker images

Usage:
  dutil retag [flags]

Flags:
  -d, --dest-repository string     destination Docker repository
  -V, --dest-version string        destination Docker image version (default "latest")
  -e, --exclude-filter string      exclude filter Docker images by name; name must not contain the provided exclude filter
  -f, --filter string              filter Docker images by name; name must contain the provided filter
  -h, --help                       help for retag
  -s, --source-repository string   source Docker image repository
  -v, --source-version string      source Docker image version (default "latest")

Global Flags:
  -t, --threads int     Number of concurrent Docker pushes (default 8)
```
