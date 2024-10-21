# Getting started

A [Cluster Stack](https://github.com/SovereignCloudStack/cluster-stacks) is full template of a Kubernetes cluster. A Cluster Stack can be configured on every provider that supports Cluster API.

The Cluster Stack Operator facilitates using Cluster Stacks by automating all steps that users would have to do manually given they have a Cluster API management cluster.

The csctl helps to generate all files and build node images based on provided scripts in a format that the Cluster Stack Operator can use.

The csctl requires a certain directory structure and uses a special form of templating to insert the right versions in your configuration files (e.g. Helm charts).

## Overview
The directory structure is very important. If the directories are not configured properly, csctl will not be able to build the cluster-stack for you.

You should must have the following content inside your directory:
- csctl.yaml: the configuration of csctl
- cluster-addon directory: the directory containing the Helm chart for cluster addons (Chart.yaml, templates and Helm related files if required)
- cluster-class directory: the directory containing the Helm chart for Cluster API resources, e.g. ClusterClass (Chart.yaml, templates and Helm related files if required)
- node-image directory (optional): the directory containing config and associated scripts to build node images


## Configuring csctl
The configuration of csctl has to be specified in the `csctl.yaml`. It needs to follow this structure:

```yaml
apiVersion: csctl.clusterstack.x-k8s.io/v1alpha1
config:
  kubernetesVersion: v1.27.7
  clusterStackName: ferrol
  provider:
    type: <myprovider>
    apiVersion: <myprovider>.csctl.clusterstack.x-k8s.io/v1alpha1
    config:
```

The apiVersion specifies the version of this configuration. Currently, there is only the version `csctl.clusterstack.x-k8s.io/v1alpha1`.

Furthermore, the Kubernetes version in the format `v<major>.<minor>.<patch>` (e.g. 1.27.5) has to be specified as well as the name that should be given to the Cluster Stack.

Depending on your plugin, there might be a provider-specific configuration.
