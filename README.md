# quay-credential-provider

Kubernetes [Credential Provider](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider) for [Quay](https://quay.io).

## Overview

Provides capabilities to simplify the interaction between Kubernetes and Quay for supplying credentials for retrieving images stored within instances of Quay by leveraging the [Keyless Authentication for Robot Accounts](https://docs.redhat.com/en/documentation/red_hat_quay/3/html/manage_red_hat_quay/keyless-authentication-robot-accounts) of Quay.

## Deployment

Utilize the following to deploy the Quay Credential Provider to a Kubernetes Environment

### Prerequisites

As of Kubernetes 1.33, the following must be satisfied:

1. Kubernetes cluster version 1.33+
2. `ServiceAccountTokenForKubeletCredentialProviders` Feature Gate must be enabled

### Credential Provider Configuration

In order to integrate the Credential Provider with the Kubernetes cluster, the following must be configured.

1. The `quay-credential-provider` must be at an accessible location within each Kubernetes Node as specified by the `image-credential-provider-bin-dir` Kubelet argument. The binary can be obtained though the official [releases](https://github.com/sabre1041/quay-credential-provider/releases) or by building from source.
2. Credential Provider Configuration file at a location specified on each Kubernetes Node as specified by the `image-credential-provider-config` Kubelet argument. See [this section](#credential-provider-configuration-file) to create the configuration file.

### Credential Provider Configuration File

In order to begin leveraging the functionality of the Credential Provider, a configuration file must be created. An example can be found below:

```yaml
apiVersion: kubelet.config.k8s.io/v1
kind: CredentialProviderConfig
providers:
  - name: quay-credential-provider
    matchImages:
      - "quay.io"
    defaultCacheDuration: "12h"
    apiVersion: credentialprovider.kubelet.k8s.io/v1
    tokenAttributes:
      serviceAccountTokenAudience: quay
      requireServiceAccount: true
      optionalServiceAccountAnnotationKeys:
        - "quay.io/host"
      requiredServiceAccountAnnotationKeys:
        - "quay.io/federated-robot-account"
```

The following parameters can be configured:

1. The list of images in the `matchImages` property which specifies the images for which the Quay Credential Provider will be invoked
2. The _audience_ in the `serviceAccountTokenAudience` property. This value must match the _audience_ field of the projected Service Account Token volume within the workload.

Details related to each component of the configuration file can be found [here](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/#configure-a-kubelet-credential-provider).

## Service Account Token Verification

Since the Credential Provider federates the identity of a Kubernetes Service Account with Quay, the Kubernetes cluster must be configured with an OIDC Identity Provider. The location of the OIDC Issuer is configured within Kubernetes by setting the `service-account-issuer` API argument.

In addition, the public key that is used to sign Service Account tokens must be exposed by the JWKS (JSON Web Key Sets) OIDC endpoint so that the JWT that is submitted to Quay can be verified.

### Quay Configuration

Complete the following steps to configure Quay to serve an image and be accessible via a Federated Robot Account.

1. Publish an image to a Quay instance within a Private repository
2. Create or reuse an existing Robot Account and grant the Robot Account access, such as _Read_, to the image
3. Federate the Robot Account
    1. [Enable the Quay V2 UI](https://docs.redhat.com/en/documentation/red_hat_quay/3/html/manage_red_hat_quay/using-v2-ui)
    2. Navigate to the target _Organization_ or User Account where the image was pushed and the Robot Account was created
    3. Click the **Robot Accounts** tab
    4. Next to the target Robot Account, select the kabob and click **Set robot federation**
    5. Click the **+** button
    6. Enter the OIDC issuer (same as configured within the Kubernetes API) along with the Subject. The Subject is the Kubernetes Service Account associated with the target workload and it takes the form `system:serviceaccount:<namespace>:<service_account>`.
    7. Click **Save** and then **Close**

### Deploying a Workload

To make use of the functionality of the Quay Credential Provider, the following must be satisfied:

1. Protected Images must be stored and available within a Quay instance accessible by the Kubernetes cluster
2. Quay Robot account created and granted access to the desired image. See [this](#quay-configuration) section for additional details.
3. Kubernetes Services Account for the workload must be annotated with the `quay.io/federated-robot-account` annotation as the name of the Quay Robot Account to impersonate
4. Workload must contain the following:
    1. Associated Service Account must be configured with the requirements from the previous bullet.
    2. Must contain a Projected Service Account Token Volume with an audience matching the value specified in the `serviceAccountTokenAudience` property of the Credential Provider Configuration file

Once the workload has been added to the cluster, confirm that that the pod has successfully pulled the image. Otherwise, failures within the Credential Provider can be seen by inspecting the Kubelet logs on the Node the pod was scheduled on.

## Example Implementations

The following examples can be used to demonstrate this feature within self managed Kubernetes clusters:

* [kind](docs/kind.md)

## Development

See [this](docs/development.md) following document which details development activities, such as building the Credential Provider.

## License

* [Apache 2.0](LICENSE)
