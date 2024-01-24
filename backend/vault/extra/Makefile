# Makefile for vault test image
#
# First, clone the vault repository to /some/path, then:
#
#   $ VAULT=/path/to/vault make image
#   $ make clean
#
# $ docker images -f 'reference=vault*'
# REPOSITORY   TAG       IMAGE ID       CREATED          SIZE
# vault        latest    850998bd7ea0   22 minutes ago   1.66GB
#
# Location of vault checkout, will be used as build context.
VAULT ?= $(HOME)/code/git.archive.org/dps/vault-site

.PHONY: help
help:
	@echo "Makefile for vault images, run `make image` to build vault image"

.PHONY: image
image:
	# temporarily copying bootstrap.sh into build context
	cp ./bootstrap.sh $(VAULT)/dev
	#\033[33m to debug, you may want to add `--no-cache` \033[39m
	DOCKER_BUILDKIT=1 docker build -t vault:latest -f Dockerfile $(VAULT)
	# cleanup bootstrap
	rm -f $(VAULT)/dev/bootstrap.sh
	#\033[32m image built, now run: docker-compose up then log in under http://0.0.0.0:8000/ with admin:admin \033[39m

.PHONY: clean
clean:
	docker rmi -f vault:latest

.PHONY: up
up:
	docker-compose up

.PHONY: down
down:
	docker-compose down

