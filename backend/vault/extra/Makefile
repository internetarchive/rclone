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
	# [make] temporarily copying bootstrap.sh into build context
	cp ./bootstrap.sh $(VAULT)/dev
	# copy patch file to disable @ensure_fresh_db_connection
	cp ./0001-disable-ensure-fresh-db-connection-decorator.patch $(VAULT)
	# [make] to debug, you may want to add `--no-cache`
	DOCKER_BUILDKIT=1 docker build -t vault:latest -f Dockerfile $(VAULT)
	# [make] cleanup bootstrap
	rm -f $(VAULT)/dev/bootstrap.sh
	# [make] image built, now run: docker-compose up then log in under http://0.0.0.0:8000/ with admin:admin

.PHONY: clean
clean:
	docker rmi -f vault:latest

.PHONY: up
up:
	docker-compose up

.PHONY: down
down:
	docker-compose down

