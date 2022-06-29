package nifi

import (
	"context"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// newService returns a generic ClusterIP service for Nifi CRD
func newService(nifi *bigdatav1alpha1.Nifi) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
			Labels:    nifiutils.LabelsForNifi(nifi.Name),
		},
	}

}

// reconcileNifiUIService reconciles the ClusterIP Service for Nifi User Interface
func (r *Reconciler) reconcileNifiUIService(ctx context.Context, svc *corev1.Service, nifi *bigdatav1alpha1.Nifi) error {
	svc.Spec = corev1.ServiceSpec{
		Selector: nifiutils.LabelsForNifi(nifi.Name),
		Ports: []corev1.ServicePort{
			{
				Name:     nifiConsolePortName,
				Port:     nifiConsolePort,
				Protocol: "TCP",
			},
		},
	}

	// Checking if service already exists
	existingSVC := &corev1.Service{}
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, svc.Name, existingSVC) {
		// if it exists, do nothing
		return nil
	}

	// Set Nifi instance as the owner and controller
	if err := ctrl.SetControllerReference(nifi, svc, r.Scheme); err != nil {
		return err
	}

	return r.Client.Create(ctx, svc)
}

// reconcileServices reconciles every service resource for Nifi CRD
func (r *Reconciler) reconcileServices(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	svc := newService(nifi)

	// reconcile UI Service
	if err := r.reconcileNifiUIService(ctx, svc, nifi); err != nil {
		return err
	}

	return nil
}
