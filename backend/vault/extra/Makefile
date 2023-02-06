# Makefile for vault test images
#
#   $ VAULT=/path/to/vault make images
#   $ docker images -f 'reference=vault*'
#   $ make clean-images
#
# $ docker images -f 'reference=vault*'
# REPOSITORY        TAG       IMAGE ID       CREATED          SIZE
# vault-bootstrap   latest    d719a2910092   17 seconds ago   1.76GB
# vault-base        latest    435fe0b728a3   36 seconds ago   1.76GB
#
# Location of vault checkout, will be used as build context.
VAULT ?= $(HOME)/code/git.archive.org/dps/vault-site/

.PHONY: help
help:
	@echo "Makefile for vault containers, run `make images` to build vault-base and vault-bootstrap images"

.PHONY: images
images: image-base image-bootstrap
	docker images -f 'reference=vault*'

.PHONY: image-base
image-base:
	DOCKER_BUILDKIT=1 docker build -t vault-base:latest -f Dockerfile.base $(VAULT)

.PHONY: image-bootstrap
image-bootstrap:
	DOCKER_BUILDKIT=1 docker build --no-cache -t vault-bootstrap:latest -f Dockerfile.bootstrap .

.PHONY: clean
clean:
	docker rmi -f vault-base:latest vault-bootstrap:latest

.PHONY: up
up:
	docker-compose up

.PHONY: down
down:
	docker-compose down
