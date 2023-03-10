# Faster Vault Deposits

There are two ways to faster deposits:

* (1) using existing client, starting uploads immediately, by doing an `ls` first
  to gather metadata needed for the deposit, then fetch data as needed
* (2) using v2 deposit client, which does not require us to register all the file
  up front

Note: Option (1) is brittle to implement, as `Put` only knows about a single
item and not the complete set of files to upload.

Therefore, we start with option (2), in a separate package "v2" and a flag "v2"
that allows to test the new functionality.

## TODO

* [ ] rework the client API to start uploads immediately, using the original

## Notes

It's possible to access global configuration options, but then not a single
list of files. One would have to reimplement listing and filtering and this may
be brittle. Filtering can become complex: https://rclone.org/filtering/

```
	fs.Debugf(f, "exp")
	ci := fs.GetConfig(ctx)
	fc := filter.GetConfig(ctx)
	fs.Debugf(f, "config-info: %#v", ci)
	fs.Debugf(f, "filter: %#v", fc)
	fs.Debugf(f, fc.DumpFilters())
	fs.Debugf(f, "exp-end")
```

## Chat (2023-03-08)

> rclone is an open source file synchronisation tool, its code is available on
> github - clients implement various interfaces to supply a storage backend
> implementation. In the `Put` method of a filesystem, is it possible to access
> the command line arguments that rclone was invoked with? For example to
> understand what the complete set of files is, that is to be transferred?

Response:

> In general, it's not possible for a function or method to access the command
> line arguments that were used to invoke the program that called it. However,
> in the case of rclone, it provides a number of environment variables that can
> be used to obtain information about the current synchronization operation,
> including the list of files that are being transferred.

The Put method of a filesystem implementation in rclone receives a context
object and a file handle, and is responsible for writing the contents of the
file to the storage backend. To obtain information about the current
synchronization operation, you can use the Context object passed to the method.

The context object contains a number of fields, including Args, which is an
array of strings representing the command line arguments used to invoke rclone.
You can use this array to obtain information about the current synchronization
operation, such as the names of the files that are being transferred.

Here's an example of how you might use the Context object in the Put method of
a filesystem implementation:

```
func (f *MyFs) Put(ctx context.Context, src io.Reader, remote string) error {
    // Access the command line arguments used to invoke rclone
    args := ctx.Value(rclone.ContextKey("args")).([]string)

    // Find the index of the source directory or file in the arguments
    srcIndex := -1
    for i, arg := range args {
        if arg == "my-source-directory" || arg == "my-source-file" {
            srcIndex = i
            break
        }
    }

    // Extract the list of files being transferred from the arguments
    var files []string
    if srcIndex != -1 && srcIndex < len(args)-1 {
        files = args[srcIndex+1:]
    }

    // Write the contents of the file to the storage backend
    // ...
}
```

> In this example, we're using the ContextKey function to obtain a context key
> that can be used to retrieve the Args field from the context. We're then
> searching the args array for the index of the source directory or file, and
> using that index to extract the list of files being transferred. Note that
> this example assumes that the source directory or file is specified directly
> after the command line argument used to select the storage backend
> implementation (my-remote: in the case of rclone). If your command line
> arguments have a different format, you'll need to adjust the code
> accordingly.
