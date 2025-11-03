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
	@go run . server -p 4000

.PHONY: worker
worker: ## Starts a worker
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml run --rm worker go run . worker

.PHONY: db-cli
db-cli: ## Locally connects to the DB
	@docker-compose -f docker/docker-compose.yml -f docker/develop.yml exec mariadb mariadb -uroot -proot123

.PHONY: gen
gen:
	@go generate ./...

.PHONY: test
test:
	@go test ./...

PLATFORMS := linux/amd64 windows/amd64 darwin/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

.PHONY: release $(PLATFORMS)
release: $(PLATFORMS) ## Creates the bin on the ./builds/

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o ./builds/'$(os)-$(arch)' .
