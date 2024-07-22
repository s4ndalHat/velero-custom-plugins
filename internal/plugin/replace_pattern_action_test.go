package plugin

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/wrkt/velero-custom-plugins/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const (
	labelSelector = "agoracalyce.io/replace-pattern=RestoreItemAction"
	pattern1      = "example.com"
	replacement1  = "replaced.com"
	pattern2      = "foo"
	replacement2  = "bar"
	pattern3      = "production"
	replacement3  = "review-3"
)

func TestRestorePlugin_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfigMapClient := mocks.NewMockConfigMapInterface(ctrl)
	plugin := &RestorePlugin{
		logger:          logrus.New(),
		configMapClient: mockConfigMapClient,
	}

	// Setup expected behavior for the mock
	mockConfigMapClient.EXPECT().
		List(gomock.Any(), metav1.ListOptions{
			LabelSelector: labelSelector,
		}).
		Return(&corev1.ConfigMapList{
			Items: []corev1.ConfigMap{
				{
					Data: map[string]string{
						pattern1: replacement1,
					},
				},
				{
					Data: map[string]string{
						pattern2: replacement2,
					},
				},
				{
					Data: map[string]string{
						pattern3: replacement3,
					},
				},
			},
		}, nil)

	yamlFile, err := os.ReadFile("./mock-data/sample-ingress.yaml")
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

	if !strings.Contains(string(jsonData), replacement1) || !strings.Contains(string(jsonData), replacement2) || !strings.Contains(string(jsonData), replacement3) {
		t.Errorf("pattern replacement not found, replacements: %q, %q, %q", replacement1, replacement2, replacement3)
	}

	yamlData, err := yaml.Marshal(output.UpdatedItem)
	assert.NoError(t, err)

	// Print the output YAML
	t.Log(string(yamlFile))
	t.Log(string(yamlData))
}
