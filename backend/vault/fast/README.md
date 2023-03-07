# Faster Vault Deposits

There are two ways to faster deposits:

* using existing client, starting uploads immediately, by doing an `ls` first
  to gather metadata needed for the deposit, then fetch data as needed
* using v2 deposit client, which does not require us to register all the file
  up front

## TODO

* [ ] rework the client API to start uploads immediately, using the original
