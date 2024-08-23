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
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	logger          logrus.FieldLogger
	configMapClient corev1.ConfigMapInterface
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

	return &RestorePlugin{
		logger:          logger,
		configMapClient: configMapClient,
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

	return replacePatternAction(p, input, patterns)
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

func replacePatternAction(p *RestorePlugin, input *velero.RestoreItemActionExecuteInput, patterns map[string]string) (*velero.RestoreItemActionExecuteOutput, error) {
	p.logger.Infof("Executing ReplacePatternAction on %v", input.Item.GetObjectKind().GroupVersionKind().Kind)

	originalPodName, found, err := unstructured.NestedString(input.Item.UnstructuredContent(), "metadata", "name")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get original pod name: %v", err)
	}

	jsonData, err := json.Marshal(input.Item)
	if err != nil {
		return nil, err
	}

	modifiedString := string(jsonData)
	for pattern, replacement := range patterns {
		modifiedString = strings.ReplaceAll(modifiedString, pattern, replacement)
	}

	var modifiedObj unstructured.Unstructured
	if err := json.Unmarshal([]byte(modifiedString), &modifiedObj); err != nil {
		return nil, err
	}

	err = unstructured.SetNestedField(modifiedObj.UnstructuredContent(), originalPodName, "metadata", "name")
	if err != nil {
		return nil, fmt.Errorf("failed to set original pod name: %v", err)
	}

	p.logger.Infof("Pod name remains unchanged: %s", originalPodName)
	return velero.NewRestoreItemActionExecuteOutput(&modifiedObj), nil
}
