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
	"reflect"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// getHTTPModeConfig returns the proper env variables to configure a serving HTTP instance
func getHTTPModeConfig() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_HTTP_PORT",
		Value: "8080",
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_REMOTE_INPUT_SECURE",
		Value: "false",
	})
	return envVars
}

func getHTTPSModeConfig() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_HTTPS_PORT",
		Value: "8443",
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_REMOTE_INPUT_SECURE",
		Value: "true",
	})
	return envVars
}
func getRouteHostnameConfig(nifi *bigdatav1alpha1.Nifi) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	var proxyHost string

	if len(nifi.Spec.Console.RouteHostname) > 0 {
		proxyHost = nifi.Spec.Console.RouteHostname
	} else {
		proxyHost = nifi.Status.UIRoute
	}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_PROXY_HOST",
		Value: proxyHost,
	})

	return envVars
}

func getConsoleSpec(nifi *bigdatav1alpha1.Nifi) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	if nifi.Spec.Console.Expose {
		if nifiutils.IsConsoleProtocolHTTP(nifi) {
			envVars = append(envVars, getHTTPModeConfig()...)
		} else if nifiutils.IsConsoleProtocolHTTPS(nifi) {
			envVars = append(envVars, getHTTPSModeConfig()...)
		}
		envVars = append(envVars, getRouteHostnameConfig(nifi)...)
		return envVars
	}
	return nil
}

func getCredentials(nifi *bigdatav1alpha1.Nifi) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	if nifi.Spec.UseDefaultCredentials {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SINGLE_USER_CREDENTIALS_USERNAME",
			Value: nifiDefaultUser,
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SINGLE_USER_CREDENTIALS_PASSWORD",
			Value: nifiDefaultPassword,
		})
		return envVars
	}
	return nil
}

// getEnvVars generate the Environment Vars to apply to Nifi instance
func (r *Reconciler) getEnvVars(nifi *bigdatav1alpha1.Nifi) *[]corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	if newVars := getConsoleSpec(nifi); newVars != nil {
		envVars = append(envVars, newVars...)
	}
	if newVars := getCredentials(nifi); newVars != nil {
		envVars = append(envVars, newVars...)
	}
	return &envVars
}

// reconcileStatefulSet reconciles the StatefulSet to deploy Nifi instances
func (r *Reconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	ls := nifiutils.LabelsForNifi(nifi.Name)
	envVars := r.getEnvVars(nifi)
	ssUser := nifiUser
	npam := nifiPropertiesAccessMode
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
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &ssUser,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:   nifiImageRepo + nifiVersion,
						Name:    "nifi",
						Env:     *envVars,
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							`cp ./conf/cm/nifi.properties ./conf/nifi.properties.new ;
							../scripts/start.sh ;
							echo "broke" ;
							sleep 3600 ;
							echo 'finish'; 
							`},
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
							ContainerPort: nifiConsolePort,
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
