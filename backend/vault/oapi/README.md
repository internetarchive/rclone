# Generated API client code and notes

Using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen), via go generate:

```
$ oapi-codegen -generate types,client,spec -package oapi -o vault.gen.go schema.json
```

The `schema.json` is taken as-is from the vault-site repo.

As of 11/2024, vault emits openapi schema version 3.1.0, but openapi-codegen
only works with 3.0.X. A first attempt to fix the emitted schema failed.

## TODO

* [x] auth, via custom RequestEditorFn, as we need a CSRF token for each
  request; cf. https://github.com/deepmap/oapi-codegen/#using-securityproviders
* [.] transitional "oapi" wrapper, that look just like the current api, but
      uses the generated code under the hood
* [ ] improve test coverage
* [ ] delete manual api client

