package nifi

import (
	"context"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// labelsForNifi returns the labels for selecting the resources belonging to the given nifi CR name.
func labelsForNifi(name string) map[string]string {
	return map[string]string{"app": "nifi", "nifi_cr": name}
}

// reconcileResources will reconcile every Nifi resource
func (r *Reconciler) reconcileResources(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	log.Info("Reconciling Status")
	if err := r.reconcileStatus(ctx, req, nifi); err != nil {
		return err
	}

	log.Info("Reconciling ConfigMaps")
	if err := r.reconcileConfigMaps(ctx, req, nifi); err != nil {
		return err
	}

	log.Info("Reconciling Services")
	if err := r.reconcileServices(ctx, req, nifi); err != nil {
		return err
	}

	log.Info("Reconciling StatefulSet")
	if err := r.reconcileStatefulSet(ctx, req, nifi); err != nil {
		return err
	}

	return nil
}
