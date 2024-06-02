# acc-kubeconfig-cli

## Overview

This CLI is used to manage access to Kubernetes clusters using [kubectl](https://kubernetes.io/docs/reference/kubectl/) by exporting and integrating `kubeconfig` files for your [Akamai Connected Cloud (Linode)](https://www.akamai.com/cloud/) Kubernetes clusters into your `~/.kube/config`.

## Usage

The `acc-kubeconfig-cli` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN`.

To get started, run the following command:

```bash
./acc-kubeconfig-cli
```

It will merge the kubeconfig files of all Linode Kubernetes Engine (LKE) clusters into a single file, and the outut will look similar to:

```bash
Added cluster eu-west (Region: eu-west) to the KUBECONFIG
Added cluster dev (Region: fr-par) to the KUBECONFIG
Success: Kubeconfig updated successfully
```

### --help

```bash
❯ ./acc-kubeconfig-cli --help
Usage: acc-kubeconfig-cli [--debug] [--help]
Merges the kubeconfig files of all Linode Kubernetes Engine (LKE) clusters into a single file.

Options:
  --debug   Enable debug mode to print additional information during script execution
  --help    Display this help information

Environment Variables:
  LINODE_API_TOKEN   Linode API token for authentication

For more information and source code, visit:
https://github.com/johnybradshaw/acc-kubeconfig-cli
```

## Build Instructions

To build the binary, run the following command:

```bash
go build -o acc-kubeconfig-cli
```
