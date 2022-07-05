# Apache Nifi Operator Design
Apache Nifi Operator design document

## Custom Resource Definitions
The following sections describe the CRD details supported by this operator

### Nifi CRD properties
| Resource Name              | Description                                                                                                |
|----------------------------|------------------------------------------------------------------------------------------------------------|
| apiVersion                 | API Version for the operator (bigdata.quay.io/v1alpha1)                                                    |
| Kind                       | Nifi                                                                                                       |
| spec.size                  | Determines the number of Nifi instances to be deployed                                                     |
| spec.useDefaultCredentials | Enables or not the builtin default credentials<br />*Username: administrator Password: administrator*      |
| spec.console.expose        | Enables Openshift Route management to expose the Nifi's User Interface                                     |
| spec.console.protocol      | Configures Nifi instance to start in HTTPS or HTTP mode. Both options are compatible with Openshift Routes |
| spec.console.routeHostname | Sets the Openshift Route's hostname (Optional)                                                             |

### Nifi CRD status:
| Resource Name  | Description                                                                            |
|----------------|----------------------------------------------------------------------------------------|
| status.nodes   | List with every currently pod name running for the Nifi instance                       |
| status.uiRoute | Openshift Route's hostname used to expose the Nifi's User Interface (if its setted up) |
