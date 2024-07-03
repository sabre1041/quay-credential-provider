# quay-credential-provider

Kubernetes [Credential Provider](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider) for [Quay](https://quay.io).

# Overview

Provides capabilities to simplify supplying credentials for retrieving images stored within instances of Quay.

## Current Functionality

Prior to future state features being supported within Quay, this Proof of Concept (PoC) implementation loads credentials (in `containers-auth.json` format) stored on Node filesystem and supply's them to invoking Kubelet's to retrieve images from remote repositories. See the [Deployment](#Deployment) section for further details.

# Building

To build the provider for the current build Operating System and Architecture, execute the following:

```shell
make build
```

To build the provider for multiple Operating Systems and Architectures, execute the following:

```shell
make cross
```

The resulting binaries will be placed in the [bin](bin) directory

# Deployment

Utilize the follow steps to make use of the current capabilities included within the provider.

## Prerequisites

1. AWS OpenShift based Cluster 4.15+
2. Cluster Administrator rights
3. An image in a private/protected registry that is used as a test harness
4. Build tools
    1. [butane](https://docs.openshift.com/container-platform/4.15/installing/install_config/installing-customizing.html#installation-special-config-butane-about_installing-customizing)
    2. Podman/Docker

## Binary Build

Ensure a binary for the desired Operating System and Architecture has been built. Otherwise, run `make cross` which will populate binaries within the [bin](bin)
 directory.

## Layered Image Build

[OpenShift Image Layering](https://docs.openshift.com/container-platform/4.15/post_installation_configuration/coreos-layering.html) is used to extend the existing OpenShift node content and inject the provider binary.

First, obtain the release OpenShift RHCOS release image.

```shell
RHCOS_BASE_IMAGE=$(oc adm release info --image-for rhel-coreos)
```

Build an image specifying the destination location where the image will be published

```shell
podman build --build-arg RHCOS_BASE_IMAGE=$RHCOS_BASE_IMAGE -f Dockerfile-layered --ignorefile .dockerignore-layered -t <DESTINATION_IMAGE> .
```

Push the image to a remote registry that is accessible by the OpenShift cluster

```shell
podman push <DESTINATION_IMAGE>
```

## Populate Authfile

The current implementation leverages a Authfile that is stored on the OpenShift node to authenticate against a protected image to demonstrate the functionality of the provider. Execute the following command to create an _authfile_ within the [butane/files](butane/files/) directory.

```shell
podman login <REGISTRY> --authfile=butane/files/authfile.json
```

### Customize CredentialProviderConfig

Update the [CredentialProviderConfig.yaml](butane/files/CredentialProviderConfig.yaml) with the images that should be matched. Replace the `<REGISTRY> with the value

```yaml
apiVersion: kubelet.config.k8s.io/v1
kind: CredentialProviderConfig
providers:
  - name: quay-credential-provider
    matchImages:
      - "<REGISTRY>"
    defaultCacheDuration: "12h"
    env:
      - name: CREDENTIAL_FILE
        value: /etc/quay-credential-provider/authfile.json
    apiVersion: credentialprovider.kubelet.k8s.io/v1
```

## Create MachineConfig

A `MachineConfig` will be used to over lay both the custom layered image and the configuration files to the OpenShift nodes.

Generate the `MachineConfig` from  the [worker.bu](butane/worker.bu) Butane file

```shell
MACHINECONFIG=$(butane -d butane butane/worker.bu)
```

[yq](https://mikefarah.gitbook.io/yq) can be used to inject the Layed Image configuration into the MachineConfig produced by butane. Use the following command to create a file called `99-quay-credential-provider-worker.yaml` in the [machineconfigs](machineconfigs) directory.

```shell
echo $MACHINECONFIG | yq '.spec.osImageURL |= env(RHCOS_BASE_IMAGE)' > machineconfigs/99-quay-credential-provider-worker.yaml
```

## Apply the MachineConfig

Apply the `MachineConfig` to the cluster to update the `worker` MachineConfigPooll

```shell
oc apply -f machineconfigs/99-quay-credential-provider-worker.yaml
```

Wait for the _worker_ `MachineConfigPool` to rollout

## Deploy Workload

Create a workload that makes use of an image stored in a private registry that was configured within the provider in the preceeding steps

Confirm the image was pulled successfully from the private registry.
