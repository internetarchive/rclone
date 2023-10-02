// Package vault adds support for the Vault Digital Preservation System
// developed and hosted at the Internet Archive. Find out more at:
// https://vault-webservices.zendesk.com/hc/en-us
//
//go:generate oapi-codegen -generate types,client,spec -package vault -o v2.gen.go deposit-v2-openapi.json
package vault
