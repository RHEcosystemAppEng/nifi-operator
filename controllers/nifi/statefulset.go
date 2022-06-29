package nifi

import (
	"context"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// getEnvVars generate the Environment Vars to apply to Nifi instance
func (r *Reconciler) getEnvVars(nifi *bigdatav1alpha1.Nifi) *[]corev1.EnvVar {
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
	}
	return &envVars
}

// reconcileStatefulSet reconciles the StatefulSet to deploy Nifi instances
func (r *Reconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	ls := nifiutils.LabelsForNifi(nifi.Name)
	envVars := r.getEnvVars(nifi)
	ssUser := nifiUser
	npam := nifiPropertiesAccessMode

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
						Image: nifiImageRepo + nifiVersion,
						Name:  "nifi",
						Env:   *envVars,
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
										Name: "nifi-properties",
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
	existingSS := &appsv1.StatefulSet{}
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, ss.Name, existingSS) {
		changed := false

		// Check Nifi.Spec.Size
		if &nifi.Spec.Size != existingSS.Spec.Replicas {
			existingSS.Spec.Replicas = &nifi.Spec.Size
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
