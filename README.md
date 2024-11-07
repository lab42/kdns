![kdns](assets/banner.svg)

<p align="center">
  <img src="https://img.shields.io/github/v/tag/lab42/kdns?label=latest%20tag&style=flat-square" alt="Latest Tag" height="30" />
  <img src="https://img.shields.io/github/actions/workflow/status/lab42/kdns/tag.yaml?style=flat-square" alt="Build Status" height="30" />
  <img src="https://img.shields.io/github/go-mod/go-version/lab42/kdns?style=flat-square" alt="Go Version" height="30" />
  <img src="https://img.shields.io/github/license/lab42/kdns?style=flat-square" alt="License" height="30" />
  <a href="https://goreportcard.com/report/github.com/lab42/kdns">
    <img src="https://goreportcard.com/badge/github.com/lab42/kdns?style=flat-square" alt="Go Report Card" height="30" />
  </a>
  <a href="https://sonarcloud.io/summary/overall?id=lab42_kdns">
  <img src="https://img.shields.io/sonar/quality_gate/lab42_kdns/main?server=https%3A%2F%2Fsonarcloud.io&style=flat-square" alt="SonarCloud Quality Gate" height="30">
  </a>
</p>

- [Overview](#overview)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Install using Helm](#install-using-helm)
- [Configuration](#configuration)
- [Usage](#usage)
  - [Example](#example)
- [Helm Chart Customization](#helm-chart-customization)
- [Contributors](#contributors)
- [Note on Windows Support](#note-on-windows-support)
- [Contributing](#contributing)
- [License](#license)

<br/>

<h2 align="center">Overview</h2>

kdns is a lightweight mDNS (Multicast DNS) server for Kubernetes that exposes services on the local network using DNS names with `.local` extensions. This service helps manage local DNS resolution in Kubernetes environments, facilitating service discovery for containers without the need for a full DNS solution.

<h2 align="center">Features</h2>

- **mDNS Server**: Exposes services over mDNS using `.local` DNS names.
- **Kubernetes Integration**: Works natively within Kubernetes environments.
- **Service Discovery**: Automatically discovers and advertises services to the local network.
- **Security**: Runs with minimal privileges for enhanced security.
- **Helm Chart**: Deployable via Helm for easy integration into your Kubernetes clusters.

<h3 align="center">Prerequisites</h3>

- Kubernetes Cluster
- Helm (v3.8.0 or higher recommended)
- A running Kubernetes environment (local only)

<h3 align="center">Install using Helm</h3>

To install `kdns` in your Kubernetes cluster, follow these steps:

1. Add the Helm repository:

```bash
helm repo add lab42 https://ghcr.io/lab42/charts
helm repo update
```

2. Install the `kdns` chart:

```bash
helm install kdns lab42/kdns
```

This will install the `kdns` service with default settings in your Kubernetes cluster.

<h3 align="center">Configuration</h3>

You can customize the installation by overriding values in your `values.yaml`. For example:

```yaml
serviceAccount:
  create: true
  name: ingress-kdns

hostNetwork: true  # Exposes the pod on the host network
```

To configure the `kdns` chart with your custom values, create a `values.yaml` file and pass it to Helm:

```bash
helm install kdns lab42/kdns -f values.yaml
```

<h2 align="center">Usage</h2>

Once `kdns` is installed, it will expose services on the local network using mDNS. For example, if you have a service named `my-service`, it will be available under `my-service.local`.

<h3 align="center">Example</h3>

- Deploy a service in your Kubernetes cluster (e.g., an HTTP server running in a pod).
- The `kdns` server will automatically expose this service on the local network.
- Access it using the `.local` domain name, like `http://my-service.local`.

<h2 align="center">Helm Chart Customization</h2>

For a full list of configurable parameters, check the [Helm Chart values](https://github.com/lab42/charts/blob/main/charts/kdns/values.yaml).

<h2 align="center">Contributors</h2>

<a href="https://github.com/lab42/kdns/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=lab42/kdns" />
</a>

<h2 align="center">Note on Windows Support</h2>

Please be aware that I do not use Windows as part of my workflow. As a result, I cannot provide support for Windows-related issues or configurations. However, I do generate Windows executables as a courtesy for those who need them.

Thank you for your understanding!

<h2 align="center">Contributing</h2>

I welcome contributions to this project! If you have ideas for new features or improvements, please submit a feature request or contribute directly to the project.

<h2 align="center">License</h2>

This project is licensed under the [MIT License](LICENSE).
