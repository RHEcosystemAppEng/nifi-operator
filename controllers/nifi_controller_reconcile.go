package controllers

import (
	// Go libs
	"context"
	"reflect"
	"time"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nifiImageRepo = "docker.io/apache/nifi"
)

//
//
//
//
// Nifi workload resources reconciling
//

// statefulSetForNifi returns a nifi Deployment object
func (r *NifiReconciler) getNifiStatefulSet(nifi *bigdatav1alpha1.Nifi) *appsv1.StatefulSet {
	ls := labelsForNifi(nifi.Name)
	replicas := nifi.Spec.Size
	var user int64
	user = 1000
	nifiPropertiesExecuteMode := int32(420)

	envVars := []corev1.EnvVar{}
	if nifi.Spec.UseDefaultCredentials {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SINGLE_USER_CREDENTIALS_USERNAME",
			Value: "administrator",
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SINGLE_USER_CREDENTIALS_PASSWORD",
			Value: "administrator",
		})
	}

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &user,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image: nifiImageRepo + ":1.16.3",
						Name:  "nifi",
						Env:   envVars,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8443,
							Name:          "nifi-console",
						}},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "nifi-properties",
								MountPath: "/opt/nifi/nifi-current/conf/cm/nifi.properties",
								SubPath:   "nifi.properties",
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "nifi-properties",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nifi-properties",
									},
									DefaultMode: &nifiPropertiesExecuteMode,
								},
							},
						},
					},
				},
			},
		},
	}
	// Set Nifi instance as the owner and controller
	ctrl.SetControllerReference(nifi, ss, r.Scheme)

	return ss
}

// lookupForNifiStatefulSet locates if there is an existing instance of the requested StatefulSet
func (r *NifiReconciler) lookupForNifiStatefulSet(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (*appsv1.StatefulSet, error) {
	ss := &appsv1.StatefulSet{}

	if err := r.Get(ctx, types.NamespacedName{Name: nifi.Name, Namespace: nifi.Namespace}, ss); err != nil && errors.IsNotFound(err) {
		log.Info("StatefulSet not found", "StatefulSet.Namespace", nifi.Namespace, "StatefulSet.Name", nifi.Name)
		return nil, err
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		return nil, err
	}

	return ss, nil
}

// updateNifiStatefulSetSize ensure the StatefulSet size is the same as the spec
func (r *NifiReconciler) updateNifiStatefulSetSize(ctx context.Context, ss *appsv1.StatefulSet, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	size := nifi.Spec.Size
	if *ss.Spec.Replicas != size {
		ss.Spec.Replicas = &size
		err := r.Update(ctx, ss)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *NifiReconciler) reconcileNifiStatefulSet(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	// Check if the StatefulSet already exists, if not create a new one
	ss, err := r.lookupForNifiStatefulSet(ctx, req, nifi, log)
	if err != nil {
		ss := r.getNifiStatefulSet(nifi)
		log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
		err = r.Create(ctx, ss)
		if err != nil {
			log.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
			return ctrl.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	if _, err := r.updateNifiStatefulSetSize(ctx, ss, nifi, log); err != nil {
		// StatefulSet size update error - return
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// reconcileWorkloads reconciles every workload associated to Nifi CRD
func (r *NifiReconciler) reconcileWorkloads(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	if _, err := r.reconcileNifiStatefulSet(ctx, req, nifi, log); err != nil {
		log.Error(err, "Recincile process for Nifi StatefulSet failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

//
//
//
//
// Nifi Networking resources reconciling
//

func (r *NifiReconciler) getNifiService(nifi *bigdatav1alpha1.Nifi) *corev1.Service {
	ls := labelsForNifi(nifi.Name)
	sName := nifi.Name

	srv := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sName,
			Namespace: nifi.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Name:     "nifi-console",
					Port:     8443,
					Protocol: "TCP",
				},
			},
		},
	}
	// Set Nifi instance as the owner and controller
	ctrl.SetControllerReference(nifi, srv, r.Scheme)

	return srv
}

// lookupForNifiService reconciles Nifi Services
func (r *NifiReconciler) lookupForNifiService(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (*corev1.Service, error) {
	svc := &corev1.Service{}

	if err := r.Get(ctx, types.NamespacedName{Name: nifi.Name, Namespace: nifi.Namespace}, svc); err != nil && errors.IsNotFound(err) {
		log.Info("Service not found", "StatefulSet.Namespace", nifi.Namespace, "StatefulSet.Name", nifi.Name)
		return nil, err
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return nil, err
	}

	return svc, nil
}
func (r *NifiReconciler) reconcileNifiService(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	_, err := r.lookupForNifiService(ctx, req, nifi, log)
	if err != nil && errors.IsNotFound(err) {
		srv := r.getNifiService(nifi)
		log.Info("Creating a new Service", "Service.Namespace", srv.Namespace, "Service.Name", srv.Name)
		err = r.Create(ctx, srv)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", srv.Namespace, "Service.Name", srv.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileNetwork reconciles every network resource associated to Nifi CRD
func (r *NifiReconciler) reconcileNetwork(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	// Check if the service already exists, if not create a new one
	if _, err := r.reconcileNifiService(ctx, req, nifi, log); err != nil {
		log.Error(err, "Recincile process for Nifi Service failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

//
//
//
//
// Nifi Status information sync
//

func (r *NifiReconciler) reconcileStatus(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi, log *logr.Logger) (ctrl.Result, error) {
	// Update the Nifi status with the pod names
	// List the pods for this nifi's StatefulSet
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(nifi.Namespace),
		client.MatchingLabels(labelsForNifi(nifi.Name)),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Nifi.Namespace", nifi.Namespace, "Nifi.Name", nifi.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, nifi.Status.Nodes) {
		nifi.Status.Nodes = podNames
		err := r.Status().Update(ctx, nifi)
		if err != nil {
			log.Error(err, "Failed to update Nifi status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
