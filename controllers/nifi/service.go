/*
Copyright 2022.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nifi

import (
	"context"
	"errors"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// default suffix for Nifi's UI Service
	nifiUIServiceSuffix = "-console"
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

// newUIService returns a default Service pointing to Nifi's UI
func newUIService(nifi *bigdatav1alpha1.Nifi) *corev1.Service {
	svc := newService(nifi)
	svc.Name += nifiUIServiceSuffix
	return svc
}

// reconcileNifiUIService reconciles the ClusterIP Service for Nifi User Interface
func (r *Reconciler) reconcileNifiUIService(ctx context.Context, nifi *bigdatav1alpha1.Nifi) error {
	svc := newUIService(nifi)
	var nifiConsolePort int32

	if nifiutils.IsConsoleProtocolHTTP(nifi) {
		nifiConsolePort = nifiHTTPConsolePort
	} else if nifiutils.IsConsoleProtocolHTTPS(nifi) {
		nifiConsolePort = nifiHTTPSConsolePort
	} else {
		err := errors.New("Console Protocol Invalid")
		log.Error(err, "")
		return err
	}

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
	// reconcile UI Service
	if err := r.reconcileNifiUIService(ctx, nifi); err != nil {
		return err
	}

	return nil
}
