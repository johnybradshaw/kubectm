# kubectm

[![CodeQL](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql)

`kubectm` is a CLI tool designed to simplify the management of Kubernetes configurations (`kubeconfig`) across multiple cloud providers, including Linode, AWS, Azure, and GCP. It automatically retrieves credentials, downloads available kubeconfig files, and merges them into your main `~/.kube/config` file. Additionally, it supports renaming clusters and contexts to make them more meaningful and easier to manage.

## Features

- **Automatic Credential Discovery**: Automatically discovers and retrieves credentials for Linode, *(to be implemented for AWS, Azure, and GCP)*.
- **Kubeconfig Management**: Downloads and merges kubeconfig files from multiple cloud providers into a single `~/.kube/config`.
- **Context Renaming**: Automatically renames clusters and contexts to include cloud provider information, like the cluster name rather than the default randomly generated name.
- **User-Friendly Output**: Provides clear, colorised output to track the progress of operations.
- **Error Handling**: Handles edge cases, such as invalid or expired credentials, and provides meaningful error messages.
- **Customizable Extensions**: Adds custom extensions to the kubeconfig, including Linode's branding in the context's extension field to enable [Aptakube](https://aptakube.com/?ref=johnybradshaw) integration *affiliate link*.

### Supported Providers

- [Linode Kubernetes Engine (LKE)](https://www.linode.com/products/kubernetes/?utm_medium=website&utm_source=github-johnybradshaw)

## Usage

To get started, run the following command:

```bash
./kubectm
```

### Linode / Akamai Connected Cloud

The `kubectm` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN` or in your `linode-cli` config file.

It will merge the kubeconfig files of all Linode Kubernetes Engine (LKE) clusters into the main `~/.kube/config` file. The output will be a single file, and the output will look similar to:

```bash
❯ ./kubectm --reset-creds
[INFO] 2024-08-26T20:37:38+01:00 Starting kubectm...
[INFO] 2024-08-26T20:37:38+01:00 Looking for Linode config in directory: /Users/_user_/.config/linode-cli
[INFO] 2024-08-26T20:37:38+01:00 Default profile found:<<user>>
[INFO] 2024-08-26T20:37:38+01:00 Exiting section:<<user>>
[INFO] 2024-08-26T20:37:38+01:00 Parsing non-sensitive line: [DEFAULT]
[INFO] 2024-08-26T20:37:38+01:00 Parsing non-sensitive line: default-user =<<user>>
[INFO] 2024-08-26T20:37:38+01:00 Parsing non-sensitive line:
[INFO] 2024-08-26T20:37:38+01:00 Entering section:<<user>>
[INFO] 2024-08-26T20:37:38+01:00 Access token found: 3065********************************************************dbd7
[INFO] 2024-08-26T20:37:38+01:00 Linode credentials found: map[AccessToken:3065********************************************************dbd7]
[INFO] Only one set of credentials found, using it by default.
[INFO] 2024-08-26T20:37:38+01:00 Downloading kubeconfig from Linode
[ACTION] 2024-08-26T20:37:40+01:00 Downloading kubeconfig for cluster: o1g2-it-mil-lke
[INFO] 2024-08-26T20:37:41+01:00 Kubeconfig saved to /Users/_user_/.kube/o1g2-it-mil-lke-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:41+01:00 Downloading kubeconfig for cluster: komodor
[INFO] 2024-08-26T20:37:42+01:00 Kubeconfig saved to /Users/_user_/.kube/komodor-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:42+01:00 Downloading kubeconfig for cluster: zeet-acc
[INFO] 2024-08-26T20:37:44+01:00 Kubeconfig saved to /Users/_user_/.kube/zeet-acc-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:44+01:00 Merging kubeconfig from /Users/_user_/.kube/komodor-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:44+01:00 Context komodor already exists, skipping...
[ACTION] 2024-08-26T20:37:44+01:00 Merging kubeconfig from /Users/_user_/.kube/o1g2-it-mil-lke-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:44+01:00 Context o1g2-it-mil-lke already exists, skipping...
[ACTION] 2024-08-26T20:37:44+01:00 Merging kubeconfig from /Users/_user_/.kube/zeet-acc-kubeconfig.yaml
[ACTION] 2024-08-26T20:37:44+01:00 Context zeet-acc already exists, skipping...
[INFO] 2024-08-26T20:37:44+01:00 Successfully merged kubeconfigs into /Users/_user_/.kube/config
[INFO] 2024-08-26T20:37:44+01:00 Deleted file /Users/_user_/.kube/komodor-kubeconfig.yaml
[INFO] 2024-08-26T20:37:44+01:00 Deleted file /Users/_user_/.kube/o1g2-it-mil-lke-kubeconfig.yaml
[INFO] 2024-08-26T20:37:44+01:00 Deleted file /Users/_user_/.kube/zeet-acc-kubeconfig.yaml
[INFO] 2024-08-26T20:37:44+01:00 kubectm finished successfully.
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
