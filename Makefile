WORKDIR      ?= /ws

# Docker commands
DOCKER ?= docker
DOCKER_BUILD = $(DOCKER) build
DOCKER_RUN = $(DOCKER) run
DOCKER_RUN_MODE = $(shell if [ -t 0 ]; then echo -it; else echo -t; fi)
DOCKER_RM = $(DOCKER) rmi

DOCKER_RUN_ARGS = $(DOCKER_RUN_MODE) --rm -v $(PWD):$(WORKDIR) -w $(WORKDIR)
DOCKER_FILE = .docker/Dev.dockerfile
DOCKER_IMAGE = local/golang-mcp-servers-builder:dev
 
clear:
	@($(DOCKER_RM) $(DOCKER_IMAGE) &>/dev/null || true); echo "Done."

dev-image:
	@$(DOCKER_BUILD) -f $(DOCKER_FILE) -t $(DOCKER_IMAGE) .

unit-tests:
	$(DOCKER_RUN) $(DOCKER_RUN_ARGS) $(DOCKER_IMAGE) go-tests -v -no-cache -p=1 ./...

build-test:
	@for dir in $(shell ls -d cmd/*); do \
		printf "%-50s" "$${dir}:"; \
		($(DOCKER_RUN) $(DOCKER_RUN_ARGS) $(DOCKER_IMAGE) go build -o /dev/null -buildvcs=false ./$${dir}/... && echo "OK") || echo "FAIL"; \
	done
