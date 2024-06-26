# CSCTL

## Table of Contents

- [CSCTL](#csctl)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
  - [Introduction](#introduction)
  - [Features of csctl](#features-of-csctl)
  - [Docs](#docs)

## Installation
To download `csctl` there are two ways. 
Go to https://github.com/SovereignCloudStack/csctl/releases/latest and then click on the binary. The name of the binary looks similar to this `
csctl_0.0.2_linux_amd64` for Linux amd64 architecture.

This will download the binary in your `~/Downloads` directory. Use the following commands to move it to your PATH. 
```bash
chmod u+x ~/Downloads/csctl_0.0.2_linux_amd64
sudo mv ~/Downloads/csctl_0.0.2_linux_amd64 /usr/local/bin/csctl
```

Alternative way of installing the binary is to use `[gh](https://github.com/cli/cli)` command line tool.
Use the following command to download the latest binary from GitHub. 
```bash
gh release download -p 'csctl_*_linux_amd64' -R SovereignCloudStack/csctl
chmod u+x csctl_0.0.2_linux_amd64
sudo mv ./csctl_0.0.2_linux_amd64 /usr/local/bin/csctl
```
For darwin based systems, the steps are similar, you'll have to choose darwin based binaries instead of linux one mentioned above. You'll also need to update your destination directory.

## Introduction

The [Cluster Stack Operator](https://github.com/SovereignCloudStack/cluster-stack-operator) facilitates the usage of [Cluster Stacks](https://github.com/SovereignCloudStack/cluster-stacks) by automating all steps that can be automated. It takes Cluster Stacks release assets that consist mainly of two Helm charts, one to deploy in the management cluster, the other one to deploy in the workload clusters, as well as provider-specific node image (build) information.

Users can take existing releases of Cluster Stacks and the operator and will be able to create clusters easily.

This project facilitates building node image artifacts and release assets that can be used with the Cluster Stack Operator.


## Features of csctl
1. Testing and quick iterations
csctl is created with a single focus of building Cluster Stacks and testing them with Cluster Stack Operator quickly. This tool helps in doing quick iterations and facilitates testing Cluster Stacks. 

2. Versioning
When configuring Cluster Stacks, it is necessary to put versions in the configuration, e.g. to version a Helm chart or node images. This process is facilitated by the csctl through its own templating and mechanism to generate the right version, based on the content hash (for testing) or on a previous version (stable or beta channel). Users only have to use the right templating and the csctl will do all the versioning automatically.

3. Plugin mechanism for providers
The plugin mechanism of csctl allows providers to implement all provider-specific steps that are needed for this provider. This can contain a fully automated building and uploading process for node images, which can be referenced in the Cluster Stack (using the templating logic for versioning). 

4. Automated testing of Cluster Stacks
The csctl enables automated testing of Cluster Stacks if integrated in a CI process that first builds all necessary files as well as node images (if needed) and then uses them to create a workload cluster based on the Cluster Stack.

## Docs

[Docs](./docs/README.md)
