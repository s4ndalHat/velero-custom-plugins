package plugin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var log = logrus.New()

func TestRestorePlugin_Execute(t *testing.T) {
	plugin := NewRestorePlugin(log)

	// Mock ingress to be restored
	item := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]interface{}{
				"name": "abav-production-app",
				"labels": map[string]interface{}{
					"env": "production",
				},
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"host": "production.example.com",
						"http": map[string]interface{}{
							"paths": []interface{}{
								map[string]interface{}{
									"path": "/",
									"backend": map[string]interface{}{
										"service": map[string]interface{}{
											"name": "production-service",
											"port": map[string]interface{}{
												"number": 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

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
}
