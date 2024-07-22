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

const pattern = "example.com"
const replacement = "replaced.com"

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
		Get(gomock.Any(), "replace-pattern", metav1.GetOptions{}).
		Return(&corev1.ConfigMap{
			Data: map[string]string{
				pattern:      replacement,
				"foo":        "bar",
				"production": "review-3",
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

	if !strings.Contains(string(jsonData), replacement) && strings.Contains(string(jsonData), pattern) {
		t.Errorf("still got %q, want %q", pattern, replacement)
	}

	yamlData, err := yaml.Marshal(output.UpdatedItem)
	assert.NoError(t, err)

	// Print the output YAML
	t.Log(string(yamlFile))
	t.Log(string(yamlData))
}
