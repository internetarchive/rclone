# Generated API client code and notes

Using [oapi-codegen](https://github.com/deepmap/oapi-codegen), via go generate:

```
$ oapi-codegen -generate types,client,spec -package oapi -o vault.gen.go schema.json
```

## TODO

* [x] auth, via custom RequestEditorFn, as we need a CSRF token for each
  request; cf. https://github.com/deepmap/oapi-codegen/#using-securityproviders
* [.] transitional "oapi" wrapper, that look just like the current api, but
      uses the generated code under the hood
* [ ] improve test coverage
* [ ] delete manual api client

