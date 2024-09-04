# kubectm

[![CodeQL](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/github-code-scanning/codeql) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/6b136cf3913f4f08a62ad67ff78de949)](https://app.codacy.com/gh/johnybradshaw/kubectm/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade) [![Scan & Build](https://github.com/johnybradshaw/kubectm/actions/workflows/build.yml/badge.svg)](https://github.com/johnybradshaw/kubectm/actions/workflows/build.yml)

`kubectm` is a CLI tool designed to simplify the management of Kubernetes configurations (`kubeconfig`) across multiple cloud providers, including Linode, AWS, Azure, and GCP. It automatically retrieves credentials, downloads available kubeconfig files, and merges them into your main `~/.kube/config` file. Additionally, it supports renaming clusters and contexts to make them more meaningful and easier to manage.

It's inspired by [`kubectx`](https://github.com/ahmetb/kubectx), and works with it.

## Features

- **Automatic Credential Discovery**: Automatically discovers and retrieves credentials for Linode, *(to be implemented for AWS, Azure, and GCP)*.
- **Kubeconfig Management**: Downloads and merges kubeconfig files from multiple cloud providers into a single `~/.kube/config`.
- **Context Renaming**: Automatically renames clusters and contexts to include cloud provider information, like the cluster name rather than the default randomly generated name.
- **User-Friendly Output**: Provides clear, colorised output to track the progress of operations.
- **Error Handling**: Handles edge cases, such as invalid or expired credentials, and provides meaningful error messages.
- **Customizable Extensions**: Adds custom extensions to the kubeconfig, including Linode's branding in the context's extension field to enable [Aptakube](https://aptakube.com/?ref=johnybradshaw) integration (*affiliate link*).

## Supported Providers

- [Linode Kubernetes Engine (LKE)](https://www.linode.com/products/kubernetes/?utm_medium=website&utm_source=github-johnybradshaw)

### Linode / Akamai Connected Cloud

The `kubectm` requires you to have already set your Linode API token in the environment variable `LINODE_API_TOKEN` or in your `linode-cli` config file.

## Installation

To install `kubectm` download the appropriate binary for your platform and architecture, [here](https://github.com/johnybradshaw/kubectm/releases/latest), and add it to your `$PATH`.

### Checking

To check the authenticity of your downloaded binary, run the following command:

#### Import Public Keys

```zsh
❯ gpg --batch --import ./kubectm.-.Official.Signing.Key.F5494851.Public.asc kubectm.-.Release.Signing.Key.51B2B027.Public.asc
gpg: key EA219278F5494851: "kubectm - Official Signing Key <gpg@kubectm.app>" not changed
gpg: key 949E400051B2B027: "kubectm - Release Signing Key <release@kubectm.app>" not changed
gpg: Total number processed: 2
gpg:              unchanged: 2
```

### Verifying Signature

To verify the signature of the downloaded binary, in this example the macOS binary, run the following command:

```zsh
❯ gpg --verify ./kubectm-darwin-arm64.sig kubectm-darwin-arm64
gpg: Signature made Wed 28 Aug 22:00:43 2024 BST
gpg:                using RSA key E59838CF87C859691B0D87C6949E400051B2B027
gpg: Good signature from "kubectm - Release Signing Key <release@kubectm.app>" [ultimate]
```

### Verifying the Provenance of the Binary

To verify the provenance of the downloaded binary, in this example the macOS binary, run the following command:

```zsh
❯ gh attestation verify kubectm-darwin-arm64 --owner johnybradshaw
Loaded digest sha256:9dd6116c180977a9b2b0ca37d560f047e64f183eacbcab5980abd40d572425a7 for file://kubectm-darwin-arm64
Loaded 1 attestation from GitHub API
✓ Verification succeeded!

sha256:9dd6116c180977a9b2b0ca37d560f047e64f183eacbcab5980abd40d572425a7 was attested by:
REPO                   PREDICATE_TYPE                  WORKFLOW
johnybradshaw/kubectm  https://slsa.dev/provenance/v1  .github/workflows/release.yml@refs/heads/main
```

## Usage

To get started, run the following command:

### First Run on macOS

- On your first-run you will need to hold `Control` on the keyboard and right click on the `kubectm` binary and select `Open`.
- Select `Open` from the dialogue box that appears.
- *You may wish to rename from `kubectm-os-arch` to `kubectm` to make it easier to run.*

### Normal Runs

Run the following command to update the `kubectm` binary:

```zsh
❯ ./kubectm
```

### --reset-creds

To reset the stored credentials and prompt for new ones, run the following command:

```zsh
❯ ./kubectm --reset-creds
```

### --help

```zsh
❯ ./kubectm --help
kubectm - A tool to download and integrate Kubernetes configurations across multiple cloud providers.

Usage: kubectm [options]

Options:
  -h, --help        Show this help message and exit.
  -v, --version     Show the version of kubectm.
  --reset-creds     Reset the stored credentials and prompt for new ones.

For more information and source code, visit:
https://github.com/johnybradshaw/kubectm
```

## Build Instructions

To build the binary download the repo, and run the following command:

```zsh
❯ go build ./cmd
```
