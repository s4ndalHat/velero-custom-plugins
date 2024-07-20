package plugin

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var log = logrus.New()

func TestRestorePlugin_Execute(t *testing.T) {
	plugin := NewRestorePlugin(log)

	// Read YAML file
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

	// Convert the output item to JSON
	jsonData, err := json.Marshal(output.UpdatedItem)
	assert.NoError(t, err)

	// Test validation
	assert.False(t, strings.Contains(string(jsonData), "production"), "The output JSON should not contain 'production'")
	assert.True(t, strings.Contains(string(jsonData), "staging"), "The output JSON should contain 'staging'")

	// Print the output JSON
	t.Log(string(jsonData))
}
