# kind

This guide illustrates the steps necessary to leverage [kind](https://kind.sigs.k8s.io) as a Kubernetes based environment to utilize the Quay Credential Provider.

## Prerequisites

The following must be satistifed before continuing with this guide:

1. `kind` installed and configured on the local machine
2. OIDC issuer configured (See the [Create an OIDC Issuer](#configure-an-oidc-issuer) section for how this can be achieved using the Red Hat build of Keycloak)
3. `envsubst` utility installed.

## Configure Quay Credential Provider binary

The Quay Credential Provider binary must be available on the local machine so that it can be injected within a mount to the _kind_ instance.

Create a new directory called `bin` at the root of the project repository

```shell
mkdir bin
```

Download the _linux_ version of the local architecture from the Quay Credential Provider [Releases page](https://github.com/sabre1041/quay-credential-provider/releases) or by building the provider locally as described in the [Development documentation](../docs/development.md).

Ensure the file in the `bin` directory has the name `quay-credential-provider`.

## Manifest Generation

A set of manifests will be created to support the configuration of the Credential Provider within the _kind_ instance along with the _kind_ instance itself.

Perform the following steps from the root of the project repository

Create a folder called `demo` within the `examples/kind` directory:

```shell
mkdir examples/kind/demo
```

Set a series of environment variables which will be injected into template files for the configuration of _kind_, the underlying Kubernetes platform, and the target workload.

Set `QUAY_CREDENTIAL_PROVIDER_PROJECT_PATH` as the location of the Quay Credential Provider path:

```shell
export QUAY_CREDENTIAL_PROVIDER_PROJECT_PATH=$(pwd)
```

Set `OIDC_ISSUER_URL` to the endpoint of the OIDC Issuer URL:

```shell
export OIDC_ISSUER_URL=<OIDC_ISSUER_URL>
```

Set the _Audience_ that should be used for the Service Account token and matched by the demonstration workload to the `SA_TOKEN_AUDIENCE` environment variable. A suggested value is `quay`.

```shell
export SA_TOKEN_AUDIENCE=<SA_TOKEN_AUDIENCE>
```

Set `QUAY_ROBOT_ACCOUNT` as the name of the Quay Robot Account associated with rights to the protected repository:

```shell
export QUAY_ROBOT_ACCOUNT=<QUAY_ROBOT_ACCOUNT>
```

Set `IMAGE` as  the location of the of the image published to the Quay private repository.

```shell
export IMAGE=<IMAGE_LOCATION>
```

Set `MATCH_IMAGES` as the location where images should be matched where the Quay Credential Provider will be invoked. An ideal value references the organization or user account containing the private repository (such as `quay.io/myorganization`)

```shell
export MATCH_IMAGES=<MATCH_IMAGES>
```

Populate the `demo` directory by copying and rendering the template resources:

```shell
cp cp examples/app/namespace.yaml examples/kind/demo
envsubst < examples/app/deployment.yaml.template > examples/kind/demo/deployment.yaml
envsubst < examples/app/serviceaccount.yaml.template > examples/kind/demo/serviceaccount.yaml
envsubst < examples/kind/kind-cluster.yaml.template > examples/kind/demo/kind-cluster.yaml
```

### Start the kind cluster

Start the _kind_ cluster using the rendered configuration file

```shell
kind create cluster --name=quay-credential-provider --config=examples/kind/demo/kind-cluster.yaml
```

### Update OIDC JWKS

Since Service Account Tokens issued by the kind Kubernetes are signed with their own private key, the associated public key needs to be added to the JWKS exposed by the OIDC provider. This step is dependent on the OIDC provider. Click [here](#oidc-issuer-configuration-using-red-hat-build-of-keycloak) to learn how this can be achieved using RHBK.

### Quay Robot Account Federation Configuration

To enable the kind cluster to obtain credentials to access resources in Quay, a Robot Account with access to the protected repository must be federated with a Kubernetes Service Account. Utilize the following steps to configure the federation:

1. [Ensure the Quay V2 UI is enabled](https://docs.redhat.com/en/documentation/red_hat_quay/3/html/manage_red_hat_quay/using-v2-ui)
2. Navigate to the target _Organization_ or User Account where the image was pushed and the Robot Account was created. The Robot account should be the same as was configured in the `QUAY_ROBOT_ACCOUNT` environment variable.
3. Click the **Robot Accounts** tab
4. Next to the target Robot Account, select the kabob and click **Set robot federation**
5. Click the **+** button
6. Enter the OIDC issuer (as configured in the `OIDC_ISSUER_URL` environment variable configured previously) along with the Subject. The Subject is the Kubernetes Service Account that will be associated with the target workload (next section) and it takes the form `system:serviceaccount:<namespace>:<service_account>`. For this guide, enter the value `system:serviceaccount:quay-credential-provider-demo:quay-credential-provider-demo`
7. Click **Save** and then **Close**

### Deploy the Sample Workload

With all of the components now in place, create the sample workload. Execute the following commands to create the resources in the kind cluster (either `kubectl` or `oc` can be used)

```shell
oc apply -f examples/kind/demo/namespace.yaml
oc apply -f examples/kind/demo/serviceaccount.yaml
oc apply -f examples/kind/demo/deployment.yaml
```

Confirm the image was able to be pulled successfully and the Pod is running by viewing pods in the `quay-credential-provider-demo` namespace.

```shell
oc get pods -n quay-credential-provider-demo
```

Thanks to the Quay Credential Provider, the Service Account token was exchanged for credentials to enable the retrieval of the image.

## Appendix

### OIDC Issuer Configuration using Red Hat Build of Keycloak

#### Create an OIDC Issuer

An accessible OIDC issuer must available so that Quay can verify the JWT that is submitted by the Quay Credential Provider. Since _kind_ runs locally and in most cases is not accessible by the Quay instance, an alternate OIDC issuer must be available. While a variety of options are available, this guide will demonstrate how to leverage the Red Hat build of Keycloak (RHBK) as an OIDC server for Service Accounts of the kind cluster.

1. Create a new _Realm_ within called `kind` by navigating to RHBK and clicking the **Create Realm** button, entering the name of the new realm and clicking **Create**.
2. Locate the OIDC Issuer URL by selecting the **Realm Settings** and then click the **OpenID Endpoint Configuration** within the _Endpoints_ section. The `issuer` field in the associated OIDC Discovery document contains the desired value and takes the form `https://$RHBK_HOST/realms/kind`. This value will be used when specifying the `service-account-issuer` Kubernetes API argument along with the Quay Robot Account Federation.

#### Update JWKS for kind

To update the Red Hat Build of Keycloak JWKS, you need to add the Private Key that is used to sign Kubernetes Services Accounts. Realm keys are managed in RHBK within user interface by selecting **Realm Settings** within the `kind` realm and then selecting **Keys**.

Click the **Add providers** tab, select the **Add provider** button and then select **rsa**.

To obtain the private key that is used to sign Service Account tokens within _kind_, execute the following:

```shell
podman exec -it quay-credential-provider-control-plane /bin/bash -c "cat /etc/kubernetes/pki/sa.key"
```

Paste the value of the key into the _Private RSA Key_ textbox. 

Deselect the **Active** option which will enable the key to be present in the JWKS, but ensure that it is not actively being used as the signer for JWT's issued by RHBK.

Feel free to optionally enter a more descriptive name for the key if you would like, such as "kind".

Hit **Save** to apply the changes.

The key should now be exposed within the realm JWKS located at `https://$RHBK_HOST/realms/kind/protocol/openid-connect/certs`.
