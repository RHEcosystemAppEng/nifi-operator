package nifi

import (
	"context"
	"reflect"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func (r *Reconciler) reconcileStatus(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	if err := r.reconcileNifiStatus(ctx, req, nifi); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reconcileNifiStatus(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	// Update the Nifi status with the pod names
	// List the pods for this nifi's StatefulSet
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(nifi.Namespace),
		client.MatchingLabels(labelsForNifi(nifi.Name)),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Nifi.Namespace", nifi.Namespace, "Nifi.Name", nifi.Name)
		return err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, nifi.Status.Nodes) {
		nifi.Status.Nodes = podNames
		err := r.Status().Update(ctx, nifi)
		if err != nil {
			log.Error(err, "Failed to update Nifi status")
			return err
		}
	}

	return nil
}