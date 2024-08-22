# Quickstart

## Installation
To download `csctl` there are two ways.
Go to https://github.com/SovereignCloudStack/csctl/releases/latest and then click on the binary. The name of the binary looks similar to this `csctl_0.0.3_linux_amd64.tar.gz` for Linux amd64 architecture.

This will download the binary in your `~/Downloads` directory. Use the following commands to move it to your PATH.
```bash
tar xvzf ~/Downloads/csctl_<version>_linux_amd64.tar.gz
chmod u+x ~/Downloads/csctl
sudo mv ~/Downloads/csctl /usr/local/bin/csctl
```

Alternative way of installing the binary is to use `[gh](https://github.com/cli/cli)` command line tool.
Use the following command to download the latest binary from GitHub.
```bash
gh release download -p 'csctl_<version>_linux_amd64.tar.gz' -R SovereignCloudStack/csctl
tar xvzf csctl_<version>_linux_amd64.tar.gz
chmod u+x csctl
sudo mv ./csctl /usr/local/bin/csctl
```
For darwin based systems, the steps are similar, you'll have to choose darwin based binaries instead of linux one mentioned above. You'll also need to update your destination directory.

## Creating Cluster Stacks

The most important subcommand is `create`. This command takes a path to the directory where you configured your Cluster Stack and generates the necessary files in the output directory via the `--output` flag:

```bash
$ csctl create <path-to-cluster-stack-configuration-directory> --output <path-to-output-directory>
```

You can specify your node image registry with the flag `--node-image-registry`. The plugin of your provider will update the node images in the respective container registry.

You can use the `--mode` flag to specify the mode you want to use.

For example:

```bash
$ csctl create <path-to-cluster-stack-directory> --output <path-to-output-directory>  --mode hash --node-image-registry <url-of-registry>
```

You have to be authenticated to your cloud provider and container registry to which you want to upload the node images.

## Different modes of csctl

The csctl has multiple modes that can be used for different use cases.

### Hash mode

This mode is the most used one, as it allows quick iterations and testing of a cluster stack. It takes the hash of the content of the cluster stack and generates a semver version on this. You can combine it with the `custom` channel of Cluster Stack Operator and test your Cluster Stacks easily!

### Stable mode

This mode checks for existing releases of cluster stacks and versions your cluster stack accordingly. If you have an existing release of "v1", then it would use "v2" for the new one. It also checks whether the node images and cluster addons have changed or not and will only update the versions if something actually changed.

### Beta mode

Similar to stable mode, but for a beta release channel. It versions according to "v0-beta.1", etc.

### Custom mode

The custom mode can be used to define your own version. You can input any semver version and your cluster stack will be versioned accordingly.
