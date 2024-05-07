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

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// nifiUIdefaultRouteHostname default route hostname
	nifiUIdefaultRouteHostname = "nifi-console"
	// nifiUIdefaultRouteHostnameWeight default backend weight
	nifiUIdefaultRouteHostnameWeight = int32(100)
	// nifiUIRouteSuffix default Route's name suffix
	nifiUIRouteSuffix = "-console"
)

// newRoute returns Route objects for Nifi CRD
func newRoute(nifi *bigdatav1alpha1.Nifi) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
			Labels:    nifiutils.LabelsForNifi(nifi.Name),
		},
	}
}

// newUIRoute returns a default Route pointing to Nifi's UI
func newUIRoute(nifi *bigdatav1alpha1.Nifi) *routev1.Route {
	rt := newRoute(nifi)
	rt.Name += nifiUIRouteSuffix
	return rt
}

// reconcileNifiUIRoute reconciles the Route for Nifi User Interface
func (r *Reconciler) reconcileNifiUIRoute(ctx context.Context, nifi *bigdatav1alpha1.Nifi) error {
	rt := newUIRoute(nifi)
	weight := nifiUIdefaultRouteHostnameWeight
	routeHostname := ""

	// Check if a specific route hostname was defined in Nifi console instance config
	if len(nifi.Spec.Console.RouteHostname) > 0 {
		routeHostname = nifi.Spec.Console.RouteHostname
	}

	// Configuring new Route Spec
	rt.Spec = routev1.RouteSpec{
		Host: routeHostname,
		To: routev1.RouteTargetReference{
			Kind:   "Service",
			Name:   newUIService(nifi).Name,
			Weight: &weight,
		},
		TLS: &routev1.TLSConfig{
			Termination:                   "edge",
			InsecureEdgeTerminationPolicy: "Redirect",
		},
	}

	// Checking if service already exists
	existingRT := newUIRoute(nifi)
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, rt.Name, existingRT) {
		if !nifi.Spec.Console.Expose {
			// Route should be deleted and recreated instead of updated because there
			// are some fields whose doesn't support update actions
			return r.Client.Delete(ctx, rt)
		}
		return nil
	}

	// skip if UI doesn't want to be exposed
	if !nifi.Spec.Console.Expose {
		return nil
	}

	// Set Nifi instance as the owner and controller
	if err := ctrl.SetControllerReference(nifi, rt, r.Scheme); err != nil {
		return err
	}

	return r.Client.Create(ctx, rt)
}

// reconcileRoutes reconciles every Route resource for Nifi CRD
func (r *Reconciler) reconcileRoutes(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	// reconcile UI Service
	if err := r.reconcileNifiUIRoute(ctx, nifi); err != nil {
		return err
	}

	return nil
}
