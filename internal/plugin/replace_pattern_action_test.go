package plugin

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/applyconfigurations/core/v1"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

// Mock implementation of the ConfigMapInterface
type mockConfigMapClient struct {
	client *fake.Clientset
}

func (m *mockConfigMapClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement me")
}

func (m *mockConfigMapClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.ConfigMap, err error) {
	panic("implement me")
}

func (m *mockConfigMapClient) Apply(ctx context.Context, configMap *v1.ConfigMapApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.ConfigMap, err error) {
	panic("implement me")
}

func (m *mockConfigMapClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*corev1.ConfigMap, error) {
	data := map[string]string{
		"foo-production": "bar-staging",
	}
	return &corev1.ConfigMap{Data: data}, nil
}

func (m *mockConfigMapClient) Create(ctx context.Context, configMap *corev1.ConfigMap, options metav1.CreateOptions) (*corev1.ConfigMap, error) {
	return nil, nil
}

func (m *mockConfigMapClient) Update(ctx context.Context, configMap *corev1.ConfigMap, options metav1.UpdateOptions) (*corev1.ConfigMap, error) {
	return nil, nil
}

func (m *mockConfigMapClient) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return nil
}

func (m *mockConfigMapClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
	return nil, nil
}

func (m *mockConfigMapClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func TestRestorePlugin_Execute(t *testing.T) {
	configMapClient := &mockConfigMapClient{}
	plugin := &RestorePlugin{
		logger:          logrus.New(),
		configMapClient: configMapClient,
	}

	yamlFile, err := os.ReadFile("./mock/sample-ingress.yaml")
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	// Convert YAML to JSON
	var itemMap map[string]interface{}
	if err := yaml.Unmarshal(yamlFile, &itemMap); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	item := &unstructured.Unstructured{Object: itemMap}

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
	}

	output, err := plugin.Execute(input)
	assert.NoError(t, err)

	jsonData, err := json.Marshal(output.UpdatedItem)
	assert.NoError(t, err)

	// Test validation
	assert.False(t, strings.Contains(string(jsonData), "foo-production"), "The output JSON should not contain 'production'")
	assert.True(t, strings.Contains(string(jsonData), "bar-staging"), "The output JSON should contain 'staging'")

	yamlData, err := yaml.Marshal(output.UpdatedItem)
	assert.NoError(t, err)

	// Print the output YAML
	t.Log(string(yamlData))
}
