# Using csctl


## What does csctl do? 

As a user, you can create clusters based on Cluster Stacks with the help of the Cluster Stack Operator. The operator needs certain files, e.g. to apply the required Helm charts, and to get the necessary information about the versions in the cluster stack.

In order to not generate these files manually, this CLI tool takes a certain pre-defined directory structure, in which users can configure all necessary Helm charts and build scripts for node images, and generates the assets that the Cluster Stack Operator can process.

Therefore, this tool can be used to configure Cluster Stacks and to test them with the Cluster Stack Operator. It can also be used to release stable releases of Cluster Stacks that can be published for a broader community.

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


## Installing csctl
You can click on the respective release of csctl on GitHub and download the binary.

Assuming, you have downloaded the `csctl_0.0.2_linux_amd64` binary in your Downloads directory, you will need the following commands to rename the binary and to give it executable permissions.

```bash
$ sudo chmod u+x ~/Downloads/csctl_0.0.2_linux_amd64
$ sudo mv ~/Downloads/csctl_0.0.2_linux_amd64 /usr/local/bin/csctl # or use any bin directory from your PATH
```

Then you can check whether everything worked by printing the version of csctl.

```bash
$ csctl version
csctl version: 0.0.2
commit: f252304eb013014b35f8a91abf1f61aff2062601
```

If you don't see a version there, then something has gone wrong. Re-check above steps and open an issue if it still does not work!


If you're using `gh` CLI then you can also use the following to download it. 
```bash
$ gh release download -p csctl_0.0.2_linux_amd64 -R SovereignCloudStack/csctl
```

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