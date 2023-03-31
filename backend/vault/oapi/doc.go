// Package oapi wraps the vault API. The openapi schema.json file should match
// that from vault-site repo.
//
//go:generate oapi-codegen -generate types,client,spec -package oapi -o vault.gen.go schema.json
package oapi
