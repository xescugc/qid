<p align="center">
    <img src="pikoci/transport/http/assets/images/logo.svg" width="100">
</p>

![](docs/images/pikoci.gif)

# PikoCI

PikoCI is a small CI/CD system inspired by Concourse CI, built on top of a Queue/Pub-Sub system — the ones running the Jobs are basically queue workers.

:warning: This is still a PoC and under heavily development, may cause breaking changes on the API or configuration :warning:

## Why?

PikoCI is heavily inspired in [ConcourseCI](https://concourse-ci.org/) and basically trying to make a simple
configuration (HCL instead of YAML) and simple installation/usage by allowing to run everything even in-memory
(workers and DB) for small usecases.

## Documentation

The full documentation is available at [docs/README.md](./docs/README.md)
