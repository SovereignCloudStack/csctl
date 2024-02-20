# Developing and Testing csctl

## Clone the Repo

Go to [csctl at Github](https://github.com/SovereignCloudStack/csctl/) and clone the repository to your local device:

```shell
git clone git@github.com:SovereignCloudStack/csctl.git
```

```shell
cd csctl
```

## Makefile

We use a `Makefile` to building the binary.

You can see the available build targets with `make help`.

## make build

With `make build` you create the executable.

## csctl --help

With `./csctl --help` you can see the available sub-commands.

BTW: Be sure to use `./`, so that you don't accidentally use a different `csctl` from your `$PATH`.

Up to now only `create` is a feasible sub-command.

## go run main.go ...     

If you modify the source of `csctl`, you can skip the build step by using `go run`:

```shell
go run main.go create --help
```

## Create Docker Cluster Stack

In the `tests` directory is a cluster stack for the docker provider.

You can create the cluster stack like this:

```shell
❯ go run main.go create tests/cluster-stacks/docker/ferrol -m hash                                                                                                                  
Created releases/docker-ferrol-1-27-v0-sha-7ff9188
```

