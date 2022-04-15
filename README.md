# DOMAIN CHECKER

## Description

Domain checker - checks ssl for domains


## External dependencies
- postgres (mainDB)

## Configuration

Service is configured with environment variables. Run `--help` to see
full list of configuration variables and defaults.

## Build & run process

### Local

1. build an executable file for your platform:
```bash
$ make build
```

2. run the service, e.g. the following command runs the service from the repo root:
```bash
$ ./bin/go.domain-checker
```

### Docker-based build and run

1. build docker image to run
```bash
make docker-build-local
```

2. run docker image:
```bash
docker run --rm \
  registry.lucky-team.pro/luckyads/go.domain-checker:local
```

## Testing

### Automated

Simply run `make fulltest` or `make docker-fulltest`.

_Note_: you should have all proto files we use in your `/usr/local/include` directory.
[Here](https://gitlab.lucky-team.pro/luckyads/go.docker-images#how-to-set-up-local-environment) is
an instruction on how to setup your local development tools.
