// Package vault adds support for the Vault Digital Preservation System
// developed and hosted at the Internet Archive. Find out more at:
// https://support.archive-it.org/hc/en-us/sections/7581093252628-Vault.
//
//go:generate oapi-codegen -generate types,client,spec -package vault -o v2.gen.go deposit-v2-openapi.json
package vault
