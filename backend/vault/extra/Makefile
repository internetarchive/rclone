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

# SHA1 of the current "mc" tool, this may change; please adjust that as needed;
# so we a least recognize when a binary changed
MC_SHA1 = f287887779ebb2b069bdd76cab11b56f5c605f6b

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Makefile for vault images, run 'make image' to build vault image"

.PHONY: cache-mc
cache-mc:
	@if [ ! -f ./cache/mc ]; then \
		echo "$(green)[mc] Downloading mc to cache...$(reset)"; \
		mkdir -p ./cache; \
		curl -vL --fail https://dl.min.io/client/mc/release/linux-amd64/mc -o ./cache/mc; \
		echo "$(MC_SHA1)  ./cache/mc" | sha1sum --check; \
		chmod +x ./cache/mc; \
		echo "$(green)[mc] Downloaded and cached$(reset)"; \
	else \
		echo "$(MC_SHA1)  ./cache/mc" | sha1sum --check; \
		echo "$(green)[mc] Using cached mc$(reset)"; \
	fi

.PHONY: image
image: cache-mc
	# [image] copying mc from cache to build context
	cp ./cache/mc $(VAULT)/mc
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

.PHONY: clean-cache
clean-cache:
	rm -rf ./cache

.PHONY: clean-all
clean-all: clean clean-cache

.PHONY: up
up:
	docker compose up

.PHONY: down
down:
	docker compose down

