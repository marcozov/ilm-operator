# ILM Operator
This is a simple controller that allows to manage Elasticsearch Index Lifecycle Management Policies.

The operator has been developed for Elasticsearch clusters running with the ECK operator on the same Kubernetes cluster.

It is based on the opereator sdk and the following resources were very useful for this proof of concept operator:
 - https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/
 - https://book.kubebuilder.io/reference/using-finalizers.html

# Installation
The main components of this operator are
 - the controller
 - the crd
 - the kubernetes rbac policies

 # Usage
 The operator needs to run in the same kubernetes cluster where the elasticsearch cluster runs.

 Configuration is deployed via the ILMPolicy custom resource, which takes the policy body and the elasticseearch cluster reference
 as fields.

 Examples can be found in the `config/samples` folder.
