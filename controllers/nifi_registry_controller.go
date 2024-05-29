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

package controllers

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/go-logr/logr"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
)

const (
	nifiRegistryPort      = 18080
	nifiRegistryImageRepo = "docker.io/apache/nifi-registry"
	nifiRegistryVersion   = ":1.16.3"
)

var logRegistry = logf.Log.WithName("NifiRegistry-Controller")

// SetupWithManager sets up the controller with the Manager.
func (r *NifiRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reqLogger := logRegistry.WithValues()
	reqLogger.Info("Watching Nifi Registries")

	if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
		logRegistry.Error(err, "")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&bigdatav1alpha1.NifiRegistry{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&routev1.Route{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}

// blank assignment to verify that Reconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &NifiRegistryReconciler{}

// Reconciler reconciles a Nifi object
type NifiRegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile loop function
// +kubebuilder:rbac:groups=bigdata.quay.io,resources=nifis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bigdata.quay.io,resources=nifis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bigdata.quay.io,resources=nifis/finalizers,verbs=update
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=role,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=rolebinding,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=clusterrolebinding,verbs=get;list;watch;create;update;patch;delete
func (r *NifiRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := logf.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	rlog.Info("Reconciling Nifi Registry instance: ")

	// Fetch the NifiRegistry instance
	nifi := &bigdatav1alpha1.NifiRegistry{}
	err := r.Get(ctx, req.NamespacedName, nifi)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logRegistry.Info("NifiRegistry resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logRegistry.Error(err, "Failed to get Nifi instance")
		return ctrl.Result{}, err
	}

	nifiNamespacedName := types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}

	if result, err := r.reconcileNifiRegistry(nifiNamespacedName, nifi, logRegistry); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *NifiRegistryReconciler) reconcileNifiRegistry(nifiNamespacedName types.NamespacedName, instance *bigdatav1alpha1.NifiRegistry, rlog logr.Logger) (reconcile.Result, error) {

	// Define ServiceAccount for Nifi
	{
		serviceAccount := newServiceAccount(nifiNamespacedName)

		// Set Nifi instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, serviceAccount, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		existingServiceAccount := &corev1.ServiceAccount{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: serviceAccount.Namespace, Name: serviceAccount.Name}, existingServiceAccount)
		if err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new ServiceAccount", "Namespace", serviceAccount.Namespace, "Name", serviceAccount.Name)
				err = r.Client.Create(context.TODO(), serviceAccount)
				if err != nil {
					return reconcile.Result{}, err

				}
			} else {
				return reconcile.Result{}, err
			}
		}
	}

	// Define a new cluster role for Nifi
	{
		role := newRole(nifiNamespacedName)

		// Set GitopsService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, role, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		existingRole := &rbacv1.Role{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, existingRole)
		if err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new Role", "Namespace", role.Namespace, "Name", role.Name)
				err = r.Client.Create(context.TODO(), role)
				if err != nil {
					return reconcile.Result{}, err
				}
			} else {
				return reconcile.Result{}, err
			}
		} else if !reflect.DeepEqual(existingRole.Rules, role.Rules) {
			rlog.Info("Reconciling existing Role", "Name", role.Name)
			existingRole.Rules = role.Rules
			err = r.Client.Update(context.TODO(), existingRole)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Define Cluster Role Binding for Nifi
	{
		roleBinding := newRoleBinding(nifiNamespacedName)

		// Set GitopsService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, roleBinding, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		existingClusterRoleBinding := &rbacv1.RoleBinding{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, existingClusterRoleBinding)
		if err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new Cluster Role Binding", "Namespace", roleBinding.Namespace, "Name", roleBinding.Name)
				err = r.Client.Create(context.TODO(), roleBinding)
				if err != nil {
					return reconcile.Result{}, err
				}
			} else {
				return reconcile.Result{}, err
			}
		}
	}

	// Define a new Nifi StatefulSet
	{
		statefulSet := newNifiRegistryStatefulSet(nifiNamespacedName, instance)

		// Set Nifi instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, statefulSet, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		existingStatefulSet := &appsv1.StatefulSet{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, existingStatefulSet); err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new Deployment", "Namespace", statefulSet.Namespace, "Name", statefulSet.Name)
				err = r.Client.Create(context.TODO(), statefulSet)
				if err != nil {
					return reconcile.Result{}, err
				}
			} else {
				return reconcile.Result{}, err
			}

		} else {
			// Reconcile current instance
			changed := false

			// Reconcile ss spec
			if !reflect.DeepEqual(existingStatefulSet.Spec.Template, statefulSet.Spec.Template) {
				existingStatefulSet.Spec.Template = statefulSet.Spec.Template
				changed = true
			}

			if changed {
				rlog.Info("Reconciling existing NifiRegistry StatefulSet", "Namespace", statefulSet.Namespace, "Name", statefulSet.Name)
				err = r.Client.Update(context.TODO(), existingStatefulSet)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	// Create SCC binding for allowing Nifi run as UserID 1000
	{
		crb := newCRBForSCC(nifiNamespacedName)
		existingCRB := &rbacv1.ClusterRoleBinding{}

		// Get CRB
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: crb.Name}, existingCRB); err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new CRB for the SCC", "Name", crb.Name)
				err = r.Client.Create(context.TODO(), crb)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		} else {
			changed := false

			if !reflect.DeepEqual(existingCRB.RoleRef, crb.RoleRef) {
				existingCRB.RoleRef = crb.RoleRef
				changed = true
			}

			if !reflect.DeepEqual(existingCRB.Subjects, crb.Subjects) {
				existingCRB.Subjects = crb.Subjects
				changed = true
			}

			if changed {
				rlog.Info("Reconciling existing CRB", "Name", crb.Name)
				err = r.Client.Update(context.TODO(), existingCRB)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}

	}

	// Create Service
	{
		service := newNifiRegistryService(nifiNamespacedName)
		// Set Nifi instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this Service already exists
		existingService := &corev1.Service{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, existingService); err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new Service", "Namespace", service.Namespace, "Name", service.Name)
				err = r.Client.Create(context.TODO(), service)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		} else {
			changed := false

			if !reflect.DeepEqual(existingService.Spec, service.Spec) {
				existingService.Spec = service.Spec
				changed = true
			}

			if changed {
				rlog.Info("Reconciling existing Service", "Name", service.Name)
				err = r.Client.Update(context.TODO(), existingService)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// newNifiRegistryStatefulSet returns an updated instance of the StatefulSet for deploying Nifi
func newNifiRegistryStatefulSet(ns types.NamespacedName, instance *bigdatav1alpha1.NifiRegistry) *appsv1.StatefulSet {
	// Image
	var image string
	if instance.Spec.Image != "" {
		image = instance.Spec.Image
	} else {
		image = nifiImageRepo + nifiVersion
	}

	// Replicas
	var replicas int32
	replicas = 1

	envFromSources := []corev1.EnvFromSource{}

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resourcev1.MustParse("768Mi"),
			corev1.ResourceCPU:    resourcev1.MustParse("250m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resourcev1.MustParse("1Gi"),
			corev1.ResourceCPU:    resourcev1.MustParse("1"),
		},
	}
	userID := nifiUser
	fsGroupChangePolicy := corev1.FSGroupChangeOnRootMismatch

	podSpec := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsUser:           &userID,
			RunAsGroup:          &userID,
			FSGroup:             &userID,
			FSGroupChangePolicy: &fsGroupChangePolicy,
		},
		Containers: []corev1.Container{
			{
				Name:    ns.Name,
				Image:   image,
				Command: []string{"/bin/sh", "-c"},
				Args: []string{
					`
env
bash -x ../scripts/start.sh
					`,
				},
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot:             &[]bool{true}[0],
					RunAsUser:                &userID,
					AllowPrivilegeEscalation: &[]bool{false}[0],
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{
							"ALL",
						},
					},
				},

				Ports: []corev1.ContainerPort{
					{
						Name:          "http",
						Protocol:      corev1.ProtocolTCP,
						ContainerPort: nifiRegistryPort,
					},
				},
				Env:          []corev1.EnvVar{},
				EnvFrom:      envFromSources,
				VolumeMounts: []corev1.VolumeMount{},
				Resources:    resources,
				LivenessProbe: &corev1.Probe{
					InitialDelaySeconds: int32(livenessProbeInitialDelay),
					TimeoutSeconds:      int32(livenessProbeTimeout),
					PeriodSeconds:       int32(livenessProbePeriod),
					SuccessThreshold:    int32(livenessProbeSuccessThreshold),
					FailureThreshold:    int32(livenessProbeFailureThreshold),
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"test", probeCommand},
						},
					},
				},
				StartupProbe: &corev1.Probe{
					InitialDelaySeconds: int32(startupProbeInitialDelay),
					TimeoutSeconds:      int32(startupProbeTimeout),
					PeriodSeconds:       int32(startupProbePeriod),
					SuccessThreshold:    int32(startupProbeSuccessThreshold),
					FailureThreshold:    int32(startupProbeFailureThreshold),
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"test", probeCommand},
						},
					},
				},
				ReadinessProbe: &corev1.Probe{
					InitialDelaySeconds: int32(readinessProbeInitialDelay),
					TimeoutSeconds:      int32(readinessProbeTimeout),
					PeriodSeconds:       int32(readinessProbePeriod),
					SuccessThreshold:    int32(readinessProbeSuccessThreshold),
					FailureThreshold:    int32(readinessProbeFailureThreshold),
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"test", probeCommand},
						},
					},
				},
			},
		},
		Volumes:            []corev1.Volume{},
		ServiceAccountName: nifiPrefix + ns.Name,
	}

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/name": ns.Name,
			},
		},
		Spec: podSpec,
	}

	vcTemplates := []corev1.PersistentVolumeClaim{}
	statefulSetSpec := appsv1.StatefulSetSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": ns.Name,
			},
		},
		Template:             template,
		VolumeClaimTemplates: vcTemplates,
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: objectMeta(ns.Name, ns.Namespace),
		Spec:       statefulSetSpec,
	}

	return statefulSet
}

func newNifiRegistryService(ns types.NamespacedName) *corev1.Service {
	spec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Port:       nifiRegistryPort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(nifiRegistryPort)),
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/name": ns.Name,
		},
	}

	svc := &corev1.Service{
		ObjectMeta: objectMeta(ns.Name, ns.Namespace, func(o *metav1.ObjectMeta) {
		}),
		Spec: spec,
	}
	return svc
}
