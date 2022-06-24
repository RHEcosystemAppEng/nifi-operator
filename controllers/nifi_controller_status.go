package controllers

import (
	corev1 "k8s.io/api/core/v1"
)

// labelsForNifi returns the labels for selecting the resources
// belonging to the given nifi CR name.
func labelsForNifi(name string) map[string]string {
	return map[string]string{"app": "nifi", "nifi_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
