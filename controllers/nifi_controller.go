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
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
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

var (
	serviceName = "cluster"
	namespace   = "nifi-operator"
)

const (
	nifiPrefix = "nifi-"
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
	// nifiHTTPConsolePort specify the port for Nifi console
	nifiHTTPConsolePort = 8080
	// nifiHTTPSConsolePort specify the port for Nifi console
	nifiHTTPSConsolePort = 8443

	// StartupProbes
	startupProbeInitialDelay     = 1
	startupProbeTimeout          = 2
	startupProbePeriod           = 3
	startupProbeSuccessThreshold = 1
	startupProbeFailureThreshold = 15

	// LivenessProbes
	livenessProbeInitialDelay     = 20
	livenessProbeTimeout          = 5
	livenessProbePeriod           = 12
	livenessProbeSuccessThreshold = 1
	livenessProbeFailureThreshold = 3

	// ReadinessProbes
	readinessProbeInitialDelay     = 20
	readinessProbeTimeout          = 5
	readinessProbePeriod           = 12
	readinessProbeSuccessThreshold = 1
	readinessProbeFailureThreshold = 3

	probeCommand string = "/opt/nifi/nifi-current/run/nifi.pid"
)

var log = logf.Log.WithName("Nifi-Controller")

// SetupWithManager sets up the controller with the Manager.
func (r *NifiReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reqLogger := log.WithValues()
	reqLogger.Info("Watching Nifi")

	if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&bigdatav1alpha1.Nifi{}).
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
var _ reconcile.Reconciler = &NifiReconciler{}

// Reconciler reconciles a Nifi object
type NifiReconciler struct {
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
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=role,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=rolebinding,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac,resources=clusterrolebinding,verbs=get;list;watch;create;update;patch;delete
func (r *NifiReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := logf.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	rlog.Info("Reconciling Nifi instance: ")

	// Fetch the Nifi instance
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

	nifiNamespacedName := types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}

	if result, err := r.reconcileNifi(nifiNamespacedName, nifi, log); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *NifiReconciler) reconcileNifi(nifiNamespacedName types.NamespacedName, instance *bigdatav1alpha1.Nifi, rlog logr.Logger) (reconcile.Result, error) {

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

	// Define a new cluster role for backend service
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

	// Define Cluster Role Binding for backend service
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
		statefulSet := newNifiStatefulSet(nifiNamespacedName, instance)

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

			// Reconcile size
			if existingStatefulSet.Spec.Replicas != &(instance.Spec.Size) {
				existingStatefulSet.Spec.Replicas = &(instance.Spec.Size)
				changed = true
			}

			// Reconcile ss spec
			if !reflect.DeepEqual(existingStatefulSet.Spec.Template, statefulSet.Spec.Template) {
				existingStatefulSet.Spec.Template = statefulSet.Spec.Template
				changed = true
			}

			if changed {
				rlog.Info("Reconciling existing Nifi StatefulSet", "Namespace", statefulSet.Namespace, "Name", statefulSet.Name)
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
		service := newNifiService(nifiNamespacedName)
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
			} else {
				return reconcile.Result{}, err
			}
		}
	}

	// Reconcile Route
	{
		route := newNifiRoute(nifiNamespacedName)
		// Set Nifi Route instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, route, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this Service already exists
		existingRoute := &routev1.Route{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, existingRoute); err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new Route", "Namespace", route.Namespace, "Name", route.Name)
				err = r.Client.Create(context.TODO(), route)
				if err != nil {
					return reconcile.Result{}, err
				}
			} else {
				return reconcile.Result{}, err
			}
		}
	}

	// Reconcile ConfigMaps and Secrets
	{
		configMap := newNifiConfigMap(nifiNamespacedName)
		if err := controllerutil.SetControllerReference(instance, configMap, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this ConfigMap already exists
		existingConfigMap := &corev1.ConfigMap{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existingConfigMap); err != nil {
			if errors.IsNotFound(err) {
				rlog.Info("Creating a new ConfigMap", "Namespace", configMap.Namespace, "Name", configMap.Name)
				err = r.Client.Create(context.TODO(), configMap)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		} else {
			changed := false
			if !reflect.DeepEqual(configMap.Data, existingConfigMap.Data) {
				existingConfigMap.Data = configMap.Data
				changed = true
			}

			if changed {
				rlog.Info("Reconciling existing ConfigMap", "Namespace", configMap.Namespace, "Name", configMap.Name)
				err = r.Client.Update(context.TODO(), existingConfigMap)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// newServiceAccount returns a new ServiceAccount for a Nifi instance
func newServiceAccount(meta types.NamespacedName) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiPrefix + meta.Name,
			Namespace: meta.Namespace,
		},
	}
}

// newNifiStatefulSet returns an updated instance of the StatefulSet for deploying Nifi
func newNifiStatefulSet(ns types.NamespacedName, instance *bigdatav1alpha1.Nifi) *appsv1.StatefulSet {
	image := nifiImageRepo + nifiVersion
	nifiPropertiesAccessMode := int32(420)
	var replicas int32 = 1

	envFromSources := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: nifiPrefix + ns.Name + "-properties",
				},
			},
		},
	}

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
	if instance.Spec.Resources != nil {
		resources = *instance.Spec.Resources
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
						ContainerPort: nifiHTTPConsolePort, // should come from flag
					},
				},
				Env: []corev1.EnvVar{
					{
						Name:  "NIFI_SENSITIVE_PROPS_KEY",
						Value: "Th1s1sAS3crEt",
					},
					{
						Name:  "NIFI_WEB_HTTP_PORT",
						Value: "8080",
					},
				},
				EnvFrom: envFromSources,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "flow",
						MountPath: "/opt/nifi/nifi-current/conf/flow",
						SubPath:   "flow",
					},
					{
						Name:      "nifi-properties",
						MountPath: "/opt/nifi/nifi-current/conf/nifi.properties",
						SubPath:   "nifi.properties",
					},
					{
						Name:      "content-repository",
						MountPath: "/opt/nifi/nifi-current/content_repository",
						SubPath:   "content-repository",
					},
					{
						Name:      "database-repository",
						MountPath: "/opt/nifi/nifi-current/database_repository",
						SubPath:   "database-repository",
					},
					{
						Name:      "flowfile-repository",
						MountPath: "/opt/nifi/nifi-current/flowfile_repository",
						SubPath:   "flowfile-repository",
					},
					{
						Name:      "provenance-repository",
						MountPath: "/opt/nifi/nifi-current/provenance_repository",
						SubPath:   "provenance-repository",
					},
				},
				Resources: resources,
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
						HTTPGet: &corev1.HTTPGetAction{
							Path: "nifi-api/system-diagnostics",
							// Use a numeric value because the request send pod IP.
							Port: intstr.FromInt(nifiHTTPConsolePort),
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
						HTTPGet: &corev1.HTTPGetAction{
							Path: "nifi-api/system-diagnostics",
							// Use a numeric value because the request send pod IP.
							Port: intstr.FromInt(nifiHTTPConsolePort),
						},
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "nifi-properties",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: nifiPrefix + ns.Name + "-properties",
						},
						DefaultMode: &nifiPropertiesAccessMode,
					},
				},
			},
			{
				Name: "flow",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ns.Name + "-flow-" + ns.Name + "-" + fmt.Sprint(replicas-1),
						ReadOnly:  false,
					},
				},
			},
			{
				Name: "content-repository",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ns.Name + "-content-repository-" + ns.Name + "-" + fmt.Sprint(replicas-1),
						ReadOnly:  false,
					},
				},
			},
			{
				Name: "database-repository",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ns.Name + "-database-repository-" + ns.Name + "-" + fmt.Sprint(replicas-1),
						ReadOnly:  false,
					},
				},
			},
			{
				Name: "flowfile-repository",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ns.Name + "-flowfile-repository-" + ns.Name + "-" + fmt.Sprint(replicas-1),
						ReadOnly:  false,
					},
				},
			},
			{
				Name: "provenance-repository",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ns.Name + "-provenance-repository-" + ns.Name + "-" + fmt.Sprint(replicas-1),
						ReadOnly:  false,
					},
				},
			},
		},

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

	var volumeMode corev1.PersistentVolumeMode
	volumeMode = "Filesystem"
	vcTemplates := []corev1.PersistentVolumeClaim{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(ns.Name+"-flow", ns.Namespace),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				// StorageClassName: Usign Default
				VolumeMode: &volumeMode,
			},
		},
		// content_repository
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(ns.Name+"-content-repository", ns.Namespace),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				// StorageClassName: Usign Default
				VolumeMode: &volumeMode,
			},
		},
		// database_repository
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(ns.Name+"-database-repository", ns.Namespace),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				// StorageClassName: Usign Default
				VolumeMode: &volumeMode,
			},
		},
		// flowfile_repository
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(ns.Name+"-flowfile-repository", ns.Namespace),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				// StorageClassName: Usign Default
				VolumeMode: &volumeMode,
			},
		},
		// provenance_repository
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(ns.Name+"-provenance-repository", ns.Namespace),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				// StorageClassName: Usign Default
				VolumeMode: &volumeMode,
			},
		},
	}

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

func newBackendService(ns types.NamespacedName) *corev1.Service {
	spec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Port:       nifiHTTPConsolePort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(nifiHTTPConsolePort)),
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/name": ns.Name,
		},
	}
	svc := &corev1.Service{
		Spec: spec,
	}
	return svc
}

func objectMeta(resourceName string, namespace string, opts ...func(*metav1.ObjectMeta)) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      resourceName,
		Namespace: namespace,
	}
	for _, o := range opts {
		o(&objectMeta)
	}
	return objectMeta
}

func newRole(meta types.NamespacedName) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiPrefix + meta.Name,
			Namespace: meta.Namespace,
		},
		Rules: policyRuleForNifiRole(),
	}
}

func policyRuleForNifiRole() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"bigdata.quay.io",
			},
			Resources: []string{
				"nifi",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
	}
}

func newRoleBinding(meta types.NamespacedName) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifiPrefix + meta.Name,
			Namespace: meta.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      nifiPrefix + meta.Name,
				Namespace: meta.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     nifiPrefix + meta.Name,
		},
	}
}

func newNifiService(ns types.NamespacedName) *corev1.Service {
	spec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Port:       nifiHTTPConsolePort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(nifiHTTPConsolePort)),
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

func newNifiConfigMap(ns types.NamespacedName) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: objectMeta(nifiPrefix+ns.Name+"-properties", ns.Namespace, func(o *metav1.ObjectMeta) {
		}),
		Data: map[string]string{"nifi.properties": defaultNifiProperties},
	}
	return cm
}

func newNifiRoute(ns types.NamespacedName) *routev1.Route {
	weight := int32(100)
	var termination routev1.TLSTerminationType
	var insecureEdgeTerminationPolicy routev1.InsecureEdgeTerminationPolicyType

	routeSpec := routev1.RouteSpec{
		Path: "", // No needed specific path required yet
		To: routev1.RouteTargetReference{
			Kind:   "Service",
			Name:   ns.Name,
			Weight: &weight,
		},
		TLS: &routev1.TLSConfig{
			Termination:                   termination,
			InsecureEdgeTerminationPolicy: insecureEdgeTerminationPolicy,
		},
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns.Name,
			Namespace: ns.Namespace,
		},
		Spec: routeSpec,
	}
}

func newCRBForSCC(ns types.NamespacedName) *rbacv1.ClusterRoleBinding {
	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "system:openshift:scc:anyuid",
	}
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      nifiPrefix + ns.Name,
			Namespace: ns.Namespace,
		},
	}

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: objectMeta(nifiPrefix+ns.Name+"-scc-binding", ns.Namespace, func(o *metav1.ObjectMeta) {
		}),
		Subjects: subjects,
		RoleRef:  roleRef,
	}
}
