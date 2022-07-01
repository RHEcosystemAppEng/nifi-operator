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
	"strconv"

	bigdatav1alpha1 "github.com/RHEcosystemAppEng/nifi-operator/api/v1alpha1"
	nifiutils "github.com/RHEcosystemAppEng/nifi-operator/controllers/nifiutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// nifiPropertiesConfigMapName
	nifiPropertiesConfigMapNameSuffix = "-nifi-properties"
)

// newConfigMap returns a brand new corev1.ConfigMap
func newConfigMap(nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nifi.Name,
			Namespace: nifi.Namespace,
			Labels:    nifiutils.LabelsForNifi(nifi.Name),
		},
	}
}

// newConfigMapWithName returns a corev1.ConfigMap object with a specific name
func newConfigMapWithName(name string, nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	cm := newConfigMap(nifi)
	cm.ObjectMeta.Name = name
	return cm
}

/*
*
* Nifi Console Config
*
 */

// getHTTPModeConfig returns the proper envVars to configure a serving HTTP instance
func getHTTPModeConfig() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_HTTPS_PORT",
		Value: "",
	})
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

// getHTTPSModeConfig returns the proper envVars to configure a serving HTTPS instance
func getHTTPSModeConfig() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_HTTPS_PORT",
		Value: "8443",
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_WEB_HTTP_PORT",
		Value: "",
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "NIFI_REMOTE_INPUT_SECURE",
		Value: "true",
	})

	return envVars
}

// getRouteHostnameConfig returns the proper envVars to add allowed HTTP hosts
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

// getNifiPropertiesConfigMapName
func getNifiPropertiesConfigMapName(nifi *bigdatav1alpha1.Nifi) string {
	return nifi.Name + nifiPropertiesConfigMapNameSuffix
}

// isEmbeddedZookeeperEnabled checks if the embedded Zookeeper should be created or not
func isEmbeddedZookeeperEnabled(nifi *bigdatav1alpha1.Nifi) bool {
	if nifi.Spec.Size > 1 {
		return true
	}
	return false
}

// getEmbeddedZookeeperConfig returns properly envVars to configure Nifi's embedded Zookeeper instance
func getEmbeddedZookeeperConfig(nifi *bigdatav1alpha1.Nifi) map[string]string {
	nifiConf := make(map[string]string)

	// Disable embedded Zookeeper if Nifi has only one instance
	nifiConf["NIFI_STATE_MANAGEMENT_EMBEDDED_ZOOKEEPER_START"] = strconv.FormatBool(isEmbeddedZookeeperEnabled(nifi))
	// Embedded Zookeeper properties
	nifiConf["NIFI_STATE_MANAGEMENT_EMBEDDED_ZOOKEEPER_PROPERTIES"] = ""

	return nifiConf
}

// getNifiPropertiesEnvVars generate the Environment Vars to apply to Nifi instance
func getNifiPropertiesEnvVars(nifi *bigdatav1alpha1.Nifi) *[]corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	if newVars := getConsoleSpec(nifi); newVars != nil {
		envVars = append(envVars, newVars...)
	}
	if newVars := getCredentials(nifi); newVars != nil {
		envVars = append(envVars, newVars...)
	}
	return &envVars
}

// newConfigMapNifiProperties returns a ConfigMap with the nifi.properties
// values already updated and ready to be applied
func newConfigMapNifiProperties(nifi *bigdatav1alpha1.Nifi) *corev1.ConfigMap {
	cm := newConfigMapWithName(getNifiPropertiesConfigMapName(nifi), nifi)
	nifiConf := getNifiPropertiesEnvVars(nifi)
	data := make(map[string]string)

	// Converting EnvVars to map of strings to be mounted as a Environment Var
	// Config Map
	for _, envVar := range *nifiConf {
		data[envVar.Name] = envVar.Value
	}

	// Assign Config Map Data field to store the nifi.properties file
	cm.Data = data
	return cm
}

/*
*
* Reconcile functions
*
 */
// reconcileNifiProperties create/update the nifi.properties config file.
func (r *Reconciler) reconcileNifiProperties(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	cm := newConfigMapNifiProperties(nifi)

	// Check if the nifi.properties config file already exists to update or create it.
	existingCM := newConfigMapWithName(nifi.Name+nifiPropertiesConfigMapNameSuffix, nifi)
	if nifiutils.IsObjectFound(r.Client, nifi.Namespace, cm.Name, existingCM) {
		changed := false

		if !reflect.DeepEqual(cm.Data, existingCM.Data) {
			existingCM.Data = cm.Data
			changed = true
		}

		if changed {
			return r.Client.Update(ctx, existingCM)
		}

		return nil
	}

	// Set Nifi instance as the owner and controller
	if err := ctrl.SetControllerReference(nifi, cm, r.Scheme); err != nil {
		return err
	}

	return r.Client.Create(ctx, cm)
}

// reconcileConfigMaps wrap every reconcile configmap function. This function should be called
// from the main reconcile function
func (r *Reconciler) reconcileConfigMaps(ctx context.Context, req ctrl.Request, nifi *bigdatav1alpha1.Nifi) error {
	if err := r.reconcileNifiProperties(ctx, req, nifi); err != nil {
		return err
	}
	return nil
}
