GOCACHE := $(shell go env GOCACHE)

.PHONY: help
help: Makefile ## This help dialog
	@IFS=$$'\n' ; \
	help_lines=(`grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v grep -F | sed -e 's/\\$$//'`); \
	for help_line in $${help_lines[@]}; do \
		IFS=$$'#' ; \
		help_split=($$help_line) ; \
		help_command=`echo $${help_split[0]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		printf "%-30s %s\n" $$help_command $$help_info ; \
	done

export GOCACHE

.PHONY: dev-env-up
dev-env-up: ## Starts the dependencies
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml up -d mariadb

.PHONY: nats-up
nats-up: ## Starts the dependencies
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml up -d nats

.PHONY: down
down: ## Stops the containers
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml down -v --remove-orphans

.PHONY: dserve
dserve: ## Serves the server
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml run --name qid --rm -p 4000:4000 qid go run . server

.PHONY: serve
serve: ## Serves the server
	@go run . server -p 4000 -log-level=debug -jwt-secret potato -users 'pepito:$$2a$$14$$rwQk8Qvc2rij7qhFO4P1W.OiSF6AkgVU1RCrLaY2wawJcpkPEKwbm,grillo:$$2a$$14$$SvWir17.jlXxiZfe0pJuDedznetc/HWKv43YPsQQNo6MJiuypS2q6' -pipeline-name=test -pipeline-config=./qid/testdata/cron.hcl

.PHONY: worker
worker: ## Starts a worker
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml run --rm worker go run . worker

.PHONY: db-cli
db-cli: ## Locally connects to the DB
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml exec mariadb mariadb -uroot -proot123

.PHONY: gen
gen: ## Runs go generate
	@go generate ./...

.PHONY: lint
lint: ## Runs staticcheck linter
	@go tool staticcheck ./...

ARGS ?= ./...

.PHONY: test
test: ## Runs all tests
	@go test $(ARGS)

.PHONY: test-backends
test-backends: ## Run backend integration tests (mem-only, no external deps)
	@go test ./integration/backends/ -v -timeout 60s

.PHONY: test-backends-all
test-backends-all: ## Run all backend integration tests (requires docker services)
	@QID_TEST_DB_SYSTEMS=mem,sqlite,mysql,postgresql,cockroachdb,tidb QID_TEST_PUBSUB_SYSTEMS=mem,nats,rabbit,kafka go test ./integration/backends/ -v -timeout 120s

PLATFORMS := linux/amd64 windows/amd64 darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

.PHONY: release $(PLATFORMS)
release: $(PLATFORMS) ## Creates the bin on the ./builds/

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o ./builds/'$(os)-$(arch)' .
