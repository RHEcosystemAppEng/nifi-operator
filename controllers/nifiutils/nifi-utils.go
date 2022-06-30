package nifiutils

import (
	"context"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsConsoleProtocolHTTP returns true if console is on HTTP mode
func IsConsoleProtocolHTTP(nifi *bigdatav1alpha1.Nifi) bool {
	if nifi.Spec.Console.Protocol == "http" || nifi.Spec.Console.Protocol == "HTTP" {
		return true
	}
	return false
}

// IsConsoleProtocolHTTPS returns true if console is on HTTPS mode
func IsConsoleProtocolHTTPS(nifi *bigdatav1alpha1.Nifi) bool {
	if nifi.Spec.Console.Protocol == "https" || nifi.Spec.Console.Protocol == "HTTPS" {
		return true
	}
	return false
}

// LabelsForNifi returns the default set of labels for selecting the resources belonging to the given Nifi CR name.
func LabelsForNifi(name string) map[string]string {
	return map[string]string{"app": "nifi", "nifi_cr": name}
}

// FetchObject performs a Get operation in the K8s API to lookup the object passed by arguments
func FetchObject(client client.Client, namespace string, name string, obj client.Object) error {
	return client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, obj)
}

//IsObjectFound Returns a boolean value to denote if the objects already exists or not
func IsObjectFound(client client.Client, namespace string, name string, obj client.Object) bool {
	return !apierrors.IsNotFound(FetchObject(client, namespace, name, obj))
}
