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

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
)

var log = ctrllog.Log.WithName("Nifi-Controller")

// blank assignment to verify that Reconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles a Nifi object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// nifiImageRepo sets the repo URL for the Nifi image
	nifiImageRepo = "docker.io/apache/nifi"
	// nifiVersion sets the version for the Nifi Image
	nifiVersion = ":1.16.3"
	// nifiUser internal user ID for nifi
	nifiUser = int64(1000)
	// nifiPropertiesAccessMode establish the internal unix permissions for 'nifi.properties' file
	nifiPropertiesAccessMode = int32(420)
	// nifiDefaultUser sets Single User Access Username
	nifiDefaultUser = "administrator"
	// nifiDefaultUser sets Single User Access Password
	nifiDefaultPassword = "administrator"

	// nifiConsolePortName names the port for Nifi console
	nifiConsolePortName = "nifi-console"
	// nifiConsolePort specify the port for Nifi console
	nifiConsolePort = 8443
)

// Reconcile loop function
//+kubebuilder:rbac:groups=bigdata.quay.io,resources=nifi,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bigdata.quay.io,resources=nifi/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bigdata.quay.io,resources=nifi/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := ctrllog.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	rlog.Info("Reconciling Nifi instance")

	nifi := &bigdatav1alpha1.Nifi{}
	err := r.Get(ctx, req.NamespacedName, nifi)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Nifi resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Nifi instance")
		return ctrl.Result{}, err
	}

	if err = r.reconcileResources(ctx, req, nifi); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bigdatav1alpha1.Nifi{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&v1.Service{}).
		Owns(&v1.ConfigMap{}).
		Complete(r)
}