# RedHat Ecosystem Application Engineering Apache Nifi Operator
**Project Status:** Under development!

The API, spec, status and other aspects of the operator could change in future
versions.


## Overview
This operator pretends to provide a Kubernetes/Openshift deployment of [Apache
Nifi](https://nifi.apache.org/) instances and cluster deployments.

## Operator Deployment
To run the operator locally you will need to have `kubectl` or `oc` CLIs already
configured and with a login session. To start, run the following commands:
```sh
make install run
```


Run the following to deploy the operator. This will also install the RBAC manifests
from `config/rbac`.

```sh
make deploy
```

## Nifi Deployment
If you are using Openshift, please go to [Openshift Nifi
Prepare](#openshift-nifi-prepare) before to start, some pre-steps should be
performed. This includes to create a ServiceAccount which binds with the correct
permissions and SCC to be able to create the Pods. If you are on Kubernetes, go
directly to [Nifi example deployment](#nifi-example-deployment).

### Openshift Nifi prepare
Please run the following steps before to start:
1. Define your Namespace where you're going to deploy Nifi.
```sh
export NIFI_NAMESPACE=<your-namespace>
kubectl create ns $NIFI_NAMESPACE
```

2. Create the `nifi` ServiceAccount.
```sh
kubectl apply -f - << _EOF_
kind: ServiceAccount
apiVersion: v1
metadata:
  name: nifi
  namespace: $NIFI_NAMESPACE

_EOF_
```

3. Assign the correct role
```sh
kubectl apply -f - << _EOF_
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nifi
  namespace: $NIFI_NAMESPACE
subjects:
  - kind: ServiceAccount
    name: nifi
    namespace: $NIFI_NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin

_EOF_
```

### Nifi example deployment
To deploy a Nifi instance start from the following example. This provides a
simple Nifi instance with HTTPS protocol enabled and the default admin user
credentials.
```yaml
kubectl apply -f - << _EOF_
apiVersion: bigdata.quay.io/v1alpha1
kind: Nifi
metadata:
  name: nifi-example
  namespace: $NIFI_NAMESPACE
spec:
  size: 1
  useDefaultCredentials: true
  console:
    expose: true
    protocol: "https"

_EOF_
```

The default credentials are (*WARNING: This mode should be only used for
development purposes.*):
* **Admin user:** `administrator`
* **Admin password:** `administrator`

## Nifi Spec
This section explains every field in the Nifi's spec:

| Parameter | Description | Values |
|-----------|-------------|--------|
| spec.size | Number of Nifi instances to be deployed. Cluster features under development. Still using 1. | Integer >= 0 |
| spec.useDefaultCredentials | Configure Nifi with the default admin user credentials. If not, check the Nifi's logs to figure out the default credentials provisioned by Nifi | Boolean (true, false) |
| spec.console | Nifi Console Spec | *struct* |
| spec.console.expose | Creates a Openshift route if it sets to 'true' | Boolean (true, false) |
| spec.console.protocol | Enables the HTTPS protocol at Nifi's instance with a self-signed certificate, or deploy it using HTTP | "https", "http" |
