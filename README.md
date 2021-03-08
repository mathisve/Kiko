[![build](https://github.com/Mathisco-01/Kiko/actions/workflows/build.yaml/badge.svg)](https://github.com/Mathisco-01/Kiko/actions)
[![Go Report Card](https://goreportcard.com/badge/Mathisco-01/Kiko)](https://goreportcard.com/report/Mathisco-01/Kiko)

[comment]: <[![GoDoc](https://godoc.org/github.com/Mathisco-01/Kiko?status.svg)](https://godoc.org/github.com/Mathisco-01/Kiko)>


# Kiko
Build lots of serverless functions at once, then archives them so terraform can easily upload them.

## Features:
- Parallelized
- Cached
- Fast

# Usage
Configure Kiko in a file called `functions.yaml`.
## Simple
```yaml
functions:
- path: example/functions/example1
  name: example1
- path: example/functions/example2
  name: example2
  ```
## S3 Backend
```yaml
backend:
  config:
    bucket: kikobackend
    key: /
    region: eu-central-1

functions:
  - path: example/functions/example1
    name: example1
  - path: example/functions/example2
    name: example2
```