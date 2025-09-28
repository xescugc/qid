# QID (Queue CI/CD)

It's a prototype for a CI/CD based on Queues

## How to use it

Right now the current implementation supports:
- `nats`
- `mem`

By default it'll use `mem` so you can run it with any dependencies. If you want to use NATS
then you need to pass the `PUBSUB_SUSTEM=nats` (on the `docker/develop.yml`)

First start the dependencies `make dev-env-up` and then `make serve`. By default the worker will
run on the same instance. If not then change the `RUN_WORKER=false` and the you would need to run
`make work`. Though with separated workers you need to use NATS so you would need to run before
`make nats-up`.

Now that the setup is up you can start creating Pipelines and Trigger Jobs.

To create a Pipeline you have to do it through API but you have the `go run . client pipelines create -pn name -c ./qid/testdata/pipeline.json -u localhost:4000`

To trigger a Job you need to `go run . client jobs trigger -pn name -jn echo -u localhost:4000`.

And with this you'll see what it runs on the server logs, which right now it's just a `echo potato 1` and then a `echo potato 2`

## Pipeline configuration

Currently is WIP but the current one is `qid/testadata/pipeline.json`
