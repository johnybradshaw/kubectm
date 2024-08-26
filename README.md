# kubectm

[![CodeQL](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql)

## Overview

This CLI app will download and merge your `kubeconfig`s from all Cloud providers into your current `kubeconfig` file *(typically in `~/.kube/config`)*.

### Supported Providers

- [Linode Kubernetes Engine (LKE)](https://www.linode.com/products/kubernetes/?utm_medium=website&utm_source=github-johnybradshaw)

## Usage

To get started, run the following command:

```bash
./kubectm
```

### Linode / Akamai Connected Cloud

The `kubectm` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN` or in your `linode-cli` config file.

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
go build ./cmd
```
