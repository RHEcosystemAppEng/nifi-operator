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
	"reflect"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	readinessProbeDelay         = 60
	readinessProbePeriod        = 20
	livenessProbeDelay          = 30
	probeCommand         string = "/opt/nifi/nifi-current/run/nifi.pid"
)

// reconcileStatefulSet reconciles the StatefulSet to deploy Nifi instances.
func (r *Reconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	var readinessProbeHandler corev1.ProbeHandler

	ls := nifiutils.LabelsForNifi(nifi.Name)
	ssUser := nifiUser
	npam := nifiPropertiesAccessMode
	envFromSources := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getNifiPropertiesConfigMapName(nifi),
				},
			},
		},
	}

	readinessProbeHandler = corev1.ProbeHandler{
		Exec: &corev1.ExecAction{
			Command: []string{"[", "-f", "/opt/nifi/nifi-current/run/nifi.pid", "]"},
		},
	}

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &nifi.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "nifi",
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &ssUser,
					},
					Containers: []corev1.Container{{
						Image:   nifiImageRepo + nifiVersion,
						Name:    "nifi",
						EnvFrom: envFromSources,
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							`
							env
							bash -x ../scripts/start.sh ;
							`,
						},
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
							Name:          nifiConsolePortName,
							ContainerPort: int32(nifiConsolePort),
						}},
						LivenessProbe: &corev1.Probe{
							InitialDelaySeconds: int32(livenessProbeDelay),
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"test", probeCommand},
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: int32(readinessProbeDelay),
							PeriodSeconds:       int32(readinessProbePeriod),
							ProbeHandler:        readinessProbeHandler,
						},
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
										Name: nifi.Name + nifiPropertiesConfigMapNameSuffix,
									},
									DefaultMode: &npam,
								},
							},
						},
					},
				},
			},
		},
	}

	// Check if the StatefulSet exists
	existingSS := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
		},
	}
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, ss.Name, existingSS) {
		changed := false

		// Check Nifi.Spec.Size
		if !reflect.DeepEqual(ss.Spec.Replicas, existingSS.Spec.Replicas) {
			existingSS.Spec.Replicas = ss.Spec.Replicas
			changed = true
		}

		// Check Nifi.Spec.Size
		if !reflect.DeepEqual(ss.Spec.Template.Spec.Containers, existingSS.Spec.Template.Spec.Containers) {
			existingSS.Spec.Template.Spec.Containers = ss.Spec.Template.Spec.Containers
			changed = true
		}

		// Update Nifi's StatefulSet if it was modified
		if changed {
			log.Info("Updating Nifi StatefulSet")

			return r.Client.Update(ctx, existingSS)
		}

		return nil
	}

	// Set Nifi instance as the owner and controller
	if err := ctrl.SetControllerReference(nifi, ss, r.Scheme); err != nil {
		return err
	}

	return r.Client.Create(ctx, ss)
}
