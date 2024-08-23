/*
Copyright 2018, 2019 the Velero contributors.

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

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	logger          logrus.FieldLogger
	configMapClient corev1.ConfigMapInterface
	restClient      *rest.RESTClient
}

// NewRestorePlugin instantiates a RestorePlugin.
func NewRestorePlugin(logger logrus.FieldLogger) *RestorePlugin {
	// Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("Failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatalf("Failed to create clientset: %v", err)
	}

	configMapClient := clientset.CoreV1().ConfigMaps("velero")

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		logger.Fatalf("Failed to create REST client: %v", err)
	}

	return &RestorePlugin{
		logger:          logger,
		configMapClient: configMapClient,
		restClient:      restClient,
	}
}

// AppliesTo returns a ResourceSelector that matches all resources
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.logger.Info("Executing CustomRestorePlugin")
	defer p.logger.Info("Done executing CustomRestorePlugin")

	patterns, err := p.getConfigMapDataByLabel("agoracalyce.io/replace-pattern=RestoreItemAction")
	if err != nil {
		p.logger.Warnf("No ConfigMap found or error fetching ConfigMap: %v", err)
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	output, err := replacePatternAction(p, input, patterns)
	if err != nil {
		return nil, err
	}

	originalPodName, _, err := unstructured.NestedString(input.Item.UnstructuredContent(), "metadata", "name")
	if err != nil {
		return nil, fmt.Errorf("failed to get original pod name: %v", err)
	}

	newPodName, _, err := unstructured.NestedString(output.UpdatedItem.UnstructuredContent(), "metadata", "name")
	if err != nil {
		return nil, fmt.Errorf("failed to get new pod name: %v", err)
	}

	if originalPodName != newPodName {
		if err := p.updatePodVolumeRestoreResources("velero", originalPodName, newPodName); err != nil {
			p.logger.Errorf("Failed to update PodVolumeRestore resources: %v", err)
			return nil, err
		}
	}

	return output, nil
}

func (p *RestorePlugin) getConfigMapDataByLabel(labelSelector string) (map[string]string, error) {
	configMaps, err := p.configMapClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %v", err)
	}

	if len(configMaps.Items) == 0 {
		return nil, fmt.Errorf("no configmap found with label selector: %s", labelSelector)
	}

	// So we can use this plugin simultaneously
	aggregatedPatterns := make(map[string]string)
	for _, configMap := range configMaps.Items {
		for key, value := range configMap.Data {
			aggregatedPatterns[key] = value
		}
	}

	return aggregatedPatterns, nil
}

func (p *RestorePlugin) updatePodVolumeRestoreResources(namespace, originalPodName, newPodName string) error {
	pvrList := &unstructured.UnstructuredList{}
	pvrList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Version: "v1",
		Kind:    "PodVolumeRestoreList",
	})

	err := p.restClient.Get().
		Namespace(namespace).
		Resource("podvolumerestores").
		Do(context.TODO()).
		Into(pvrList)
	if err != nil {
		return fmt.Errorf("failed to list PodVolumeRestores: %v", err)
	}

	for _, item := range pvrList.Items {
		podName, found, err := unstructured.NestedString(item.Object, "spec", "podName")
		if err != nil || !found {
			p.logger.Warnf("Failed to get podName from PodVolumeRestore: %v", err)
			continue
		}

		if podName == originalPodName {
			p.logger.Infof("Updating PodVolumeRestore %s to reference new pod name: %s", item.GetName(), newPodName)
			err := unstructured.SetNestedField(item.Object, newPodName, "spec", "pod", "name")
			if err != nil {
				return fmt.Errorf("failed to update PodVolumeRestore resource: %v", err)
			}

			// Update the resource with the new pod name
			_, err = p.restClient.Put().
				Namespace(namespace).
				Resource("podvolumerestores").
				Name(item.GetName()).
				Body(&item).
				Do(context.TODO()).
				Get()
			if err != nil {
				return fmt.Errorf("failed to update PodVolumeRestore resource: %v", err)
			}
		}
	}

	return nil
}

func replacePatternAction(p *RestorePlugin, input *velero.RestoreItemActionExecuteInput, patterns map[string]string) (*velero.RestoreItemActionExecuteOutput, error) {
	p.logger.Infof("Executing ReplacePatternAction on %v", input.Item.GetObjectKind().GroupVersionKind().Kind)

	jsonData, err := json.Marshal(input.Item)
	if err != nil {
		return nil, err
	}

	modifiedString := string(jsonData)
	for pattern, replacement := range patterns {
		modifiedString = strings.ReplaceAll(modifiedString, pattern, replacement)
	}

	// Create a new item from the modified JSON data
	var modifiedObj unstructured.Unstructured
	if err := json.Unmarshal([]byte(modifiedString), &modifiedObj); err != nil {
		return nil, err
	}
	return velero.NewRestoreItemActionExecuteOutput(&modifiedObj), nil
}
