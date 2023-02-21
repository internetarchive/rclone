# Makefile for vault test image
#
#   $ VAULT=/path/to/vault make image
#   $ docker images -f 'reference=vault*'
#   $ make clean
#
# $ docker images -f 'reference=vault*'
# REPOSITORY        TAG       IMAGE ID       CREATED          SIZE
# vault             latest    d719a2910092   17 seconds ago   1.76GB
#
# Location of vault checkout, will be used as build context.
VAULT ?= $(HOME)/code/git.archive.org/dps/vault-site/

.PHONY: help
help:
	@echo "Makefile for vault images, run `make image` to build vault image"

.PHONY: image
image:
	DOCKER_BUILDKIT=1 docker build --no-cache -t vault:latest -f Dockerfile $(VAULT)

.PHONY: clean
clean:
	docker rmi -f vault:latest

.PHONY: up
up:
	docker-compose up

.PHONY: down
down:
	docker-compose down

