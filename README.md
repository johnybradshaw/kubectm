# kubectm

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/f51f920bdb1142b28feae02aace1cc43)](https://app.codacy.com/gh/johnybradshaw/kubectm?utm_source=github.com&utm_medium=referral&utm_content=johnybradshaw/kubectm&utm_campaign=Badge_Grade)
[![CodeQL](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql)

`kubectm` is a CLI tool designed to simplify the management of Kubernetes configurations (`kubeconfig`) across multiple cloud providers, including Linode, AWS, Azure, and GCP. It automatically retrieves credentials, downloads available kubeconfig files, and merges them into your main `~/.kube/config` file. Additionally, it supports renaming clusters and contexts to make them more meaningful and easier to manage.

## Features

- **Automatic Credential Discovery**: Automatically discovers and retrieves credentials for Linode, *(to be implemented for AWS, Azure, and GCP)*.
- **Kubeconfig Management**: Downloads and merges kubeconfig files from multiple cloud providers into a single `~/.kube/config`.
- **Context Renaming**: Automatically renames clusters and contexts to include cloud provider information, like the cluster name rather than the default randomly generated name.
- **User-Friendly Output**: Provides clear, colorised output to track the progress of operations.
- **Error Handling**: Handles edge cases, such as invalid or expired credentials, and provides meaningful error messages.
- **Customizable Extensions**: Adds custom extensions to the kubeconfig, including Linode's branding in the context's extension field to enable [Aptakube](https://aptakube.com/?ref=johnybradshaw) integration *affiliate link*.

## Supported Providers

- [Linode Kubernetes Engine (LKE)](https://www.linode.com/products/kubernetes/?utm_medium=website&utm_source=github-johnybradshaw)

### Linode / Akamai Connected Cloud

The `kubectm` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN` or in your `linode-cli` config file.

## Installation

To install `kubectm` download the appropriate binary for your platform and architecture, and add it to your `$PATH`.

## Usage

To get started, run the following command:

```bash
./kubectm
```

### --reset-creds

To reset the stored credentials and prompt for new ones, run the following command:

```bash
❯ ./kubectm --reset-creds
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

To build the binary download the repo, and run the following command:

```bash
go build ./cmd
```
