# kubectm (aka acc-kubeconfig-cli)

[![CodeQL](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql)

## Overview

This CLI app will merge your `kubeconfig`s from all [Linode Kubernetes Engine (LKE)](https://www.linode.com/products/kubernetes/?utm_medium=website&utm_source=github-johnybradshaw) clusters into your `~/.kube/config`. It will either use your credentials stored in the `linode-cli` config file or the `LINODE_API_TOKEN` environment variable to authenticate with the Linode API.

## Usage

The `acc-kubeconfig-cli` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN` or your `linode-cli` config file.

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
  LINODE_API_TOKEN   Linode API token for authentication (optional)

For more information and source code, visit:
https://github.com/johnybradshaw/acc-kubeconfig-cli
```

## Build Instructions

To build the binary, run the following command:

```bash
go build ./cmd/lke-kubeconfigconfig
```
