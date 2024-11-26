# Makefile for vault test image
#
# First, clone the vault repository to /some/path, then:
#
#   $ VAULT=/path/to/vault make image
#
# $ docker images -f 'reference=vault*'
# REPOSITORY   TAG       IMAGE ID       CREATED          SIZE
# vault        latest    850998bd7ea0   22 minutes ago   1.66GB
#
# To remove the image:
#
#   $ make clean

# Location of vault checkout, will be used as build context.
VAULT ?= $(HOME)/code/git.archive.org/dps/vault-site

higreen := $(shell tput setaf 10)
green := $(shell tput setaf 2)
reset := $(shell tput sgr0)

.PHONY: help
help:
	@echo "Makefile for vault images, run `make image` to build vault image"

.PHONY: image
image:
	# temporary copy "mc" binary, to avoid download from
	# https://dl.min.io/client/mc/release/linux-amd64/mc (cf. bootstrap.sh)
	# cp ./mc $(VAULT)
	#
	# [image] temporarily copying bootstrap.sh into build context
	cp ./bootstrap.sh $(VAULT)/dev
	# [image] copy patch file to run minimal workers only
	cp ./0001-minimal-workers.patch $(VAULT)
	# [image] to debug, you may want to add `--no-cache`
	DOCKER_BUILDKIT=1 docker build -t vault:latest -f Dockerfile $(VAULT)
	# [image] cleanup bootstrap and patch file
	rm -f $(VAULT)/dev/bootstrap.sh
	rm -f $(VAULT)/0001-minimal-workers.patch
	rm -f $(VAULT)/mc
	# [image] image built, now run: docker-compose up then log in under http://0.0.0.0:8000/ with admin:admin
	@printf 'done: $(higreen)vault test image successfully created$(reset)\n'
	@printf 'next: $(green)docker-compose up$(reset)\n'

.PHONY: clean
clean:
	docker rmi -f vault:latest

.PHONY: up
up:
	docker-compose up

.PHONY: down
down:
	docker-compose down

