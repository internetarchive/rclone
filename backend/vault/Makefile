# Makefile for documenting various steps.
#
# go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

deposit-v2-openapi.json:
	# e.g. start a local instance and access openapi.json
	curl -s http://localhost:8000/api/deposits/v2/openapi.json > deposit-v2-openapi.json

v2.gen.go: deposit-v2-openapi.json
	oapi-codegen -generate types,client,spec -package vault -o v2.gen.go deposit-v2-openapi.json

