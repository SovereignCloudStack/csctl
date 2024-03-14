# CSCTL - Design document

# Introduction

The Cluster Stack Operator facilitates the usage of cluster stacks by automating all steps that can be automated. It takes cluster stacks release assets that consist mainly of two Helm charts, one to deploy in the management cluster, the other one to deploy in the workload clusters, as well as provider-specific node image (build) information.

Users can take existing releases of cluster stacks and the operator and will be able to create clusters easily.

However, there is no clear and nice way to work on cluster stacks, test, and release them.

This proposal will discuss a tool to improve the experience of developers implementing cluster stacks.

# Motivation

The current process of building cluster stacks is rather cumbersome and error-prone. There are multiple issues with the current approach:

1. The release assets have to follow a very specific (naming) pattern, to be usable with the operator. Currently, they have to be created manually. There are no docs for this manual process.
2. The cluster stacks can be versioned following the pattern v1, v2, … This is perfect from the user perspective, but not good for people implementing cluster stacks, as they can only do local tests by artificially creating a v2 and not releasing it. 
3. The versioning of the cluster stacks is not easy, as there are multiple versions involved. Cluster addons have their own version, for example. Currently, the versions have to be manually hard-coded in multiple places. This can be validated to some degree but is not developer-friendly and can still lead to mistakes. 

# Proposal

We propose a CLI tool called “csctl”, which stands for cluster-stack-manager-ctl. This CLI tool should take over all manual work from a developer implementing cluster stacks that can be taken over. The developer should concentrate only on implementing the cluster stacks themselves.

There will be still a certain way of dealing with “cluster stack-specific” jobs, e.g. following a certain templating pattern. This is necessary, as the configuration and Helm Charts that developers implement are very generic.

The tool should generate release assets, e.g. by using `helm package` for the helm charts. It should be able to create these release assets for different use cases, e.g. for creating a stable release, for testing a certain commit, and for creating a beta release.    

# User stories

### User story 1: Developer releasing cluster stacks

A developer who wants to release a cluster stack that was implemented can use the CLI tool to generate all release assets that are required. This should save much time compared to following a manual process.

### User story 2: Developer versioning cluster stacks

A developer who has to think about how to version a cluster stack that was implemented can use the tool to do the job. This saves a lot of time, as the developer would have to manually check whether anything was updated for cluster addons or node images to find the appropriate version (”Did anything change in the cluster addons so that they need a new version or not?”).

### User story 3: Multiple developers work in parallel on one cluster stack

If multiple developers work on one cluster stack, they might interfere with each other’s work. Assuming that node images have to be built, then one developer would upload the node images in version “v2”, as the previous version was “v1”. The second developer has the same thought and would either overwrite the already uploaded node images of the colleague or not be able to upload the images since they exist already. 

The csctl allows both developers to have independent versioning based on a git commit hash.

### User story 4: Developer updating cluster stack that is used in production

If a developer updates a cluster stack that is used in production, great care is needed. The csctl allows the developer to safely test cluster stacks, e.g. with a beta channel, without touching cluster stacks that are used in production. 
If everything works well, a production release can be generated with csctl.

### User story 5: Automated testing of cluster stacks

Cluster stacks cannot be tested in the CI and with a normal Git PR flow. The csctl allows this testing of individual PRs and therefore enables automated testing via CI.

# Risks & Mitigations

### Two forms of templating

Helm charts use Go templating with the notation `{{ .values.myvalue }}`. As a cluster stack consists usually of two Helm charts, this notation will be very common.

However, the csctl requires a different form of templating, additionally to the one of Helm. This comes from the versioning of the cluster stacks themselves. The Cluster addon version, for example, has to be the version of the respective Helm chart. The same goes for the `ClusterClass` object name.

 Users have to use the additional templating notation `<< .ClusterAddonVersion >>` while implementing cluster stacks.

The alternative to using a different notation for cluster stack templating would be to use the same one as Helm. However, this will be confusing for users, as they cannot differentiate it. Therefore, we cannot suggest to follow that path.

# Design details

## Generic vs provider-specific work

Just like the Cluster Stack Operator, the csctl also has a generic and a provider-specific part. The provider-specific part is optional.

The generic work is done with a CLI tool that exists in the repository csctl in SCS. The tool can be initialized with provider-specific binaries, similar to the way [packer](https://github.com/hashicorp/packer) does it.

## Generic work

The generic part of the csctl is 

1. Calculate the right versions based on git commit hash or previous releases
2. Template everything with csctl templating (NOT Helm templating!!)
3. Package the ClusterClass Helm Chart
4. Package the ClusterAddon Helm Chart
5. Generate metadata.yaml

## Provider-specific work

The provider-specific part of csctl would do anything necessary to provide node images to users. One common task could be to use packer to build images and to upload them somewhere they can be accessed by users.

Of course, one task would also be to find the right version for the node images (e.g. v2 if something changed since v1, or simply the git commit hash)

## Configuration

There are multiple ways of configuring the csctl. They all have specific use cases and will be explained in the following

### Configuration file

There is a configuration file called `csctl.yaml` which contains all values that will never have to be changed for a specific cluster stack. It follows this pattern:

```yaml
apiVersion: csctl.clusterstack.x-k8s.io/v1alpha1
config:
  kubernetesVersion: v1.27.7
  clusterStackName: ferrol
  provider:
    type: myprovider
    apiVersion: myprovider.csctl.clusterstack.x-k8s.io/v1alpha1
		config: xyz
```

There is mainly the Kubernetes version, the name of the cluster stack, as well as the provider. Additionally, there is a provider-specific configuration. Both the generic and the provider-specific configuration is versioned.

### Flags

Via flags the user can specifiy everything that is important but which might change, e.g. the mode “stable” or “hash”, giving you release assets for a stable release or creating release assets based on the latest git commit hash.

### Environment variables

Environment variables can be used, for example, to specify tokens and passwords. csctl has to validate that all required environment variables have been specified.

## Commands of CLI tool

Multiple commands can make sense for developers. The most important one is the `create` command that creates release assets, as well as `provider install` to install the binary of a provider that carries out all provider-specific work.

This is a full list:

```yaml
subcommands:
provider			Is used for the provider lifecycle			
create 				creates release assets of a cluster stack.
generate			Generates a specific resource from a cluster stack
list 				  show all cluster-stacks from a repo
version				shows the version of this cli tool
help				  print a overview of available flags etc.

subcommands for provider:
install				Installs a cluster-stack-release-provider at a version
installed			Lists installed csctl release providers
remove				removes a csctl release provider at a version `csctl provider remove docker <version>`
```

## Modes

There are multiple modes to create release assets following different versioning patterns.

### Stable

The stable mode requires the developer to specify an existing GitHub repository (in the future other ways of storing release assets are possible) via environment variables. The csctl will search for the latest release fitting to the configuration of provider, cluster stack name, and Kubernetes version (e.g. docker-ferrol-1-27-vXXX). Then it will download the required release assets and check whether anything has changed in the cluster addon and node image section. Depending on that information it will calculate the next version, e.g. v2 after v1, or will leave the version the same if nothing changed.

### Hash

The hash mode is useful for developing cluster stacks. It will use the hash of the last git commit and generate a version of the form `v0-hash.<hash>`. following semver.

This version will be used for cluster class, cluster add-ons, and node images. Unlike the stable version, the versions in hash mode always update to the latest commit and do not depend on any previous release.

### Beta

The beta mode is similar to the stable mode, except that it generates releases following the version pattern `v0-beta.0`, `v0-beta.1`, etc.

### Custom (e.g. for PRs)

The custom mode is designed for PR purposes and supports automated testing. It accommodates versions formatted as v0.custom-pr123. Crucially, these versions must adhere to semantic versioning standards (semver) and are specifically intended as inputs for the csctl tool.
