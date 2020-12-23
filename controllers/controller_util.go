package controllers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
)

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func buildOperationURL(operationID, resourceID, resourceURL string) string {
	return fmt.Sprintf("%s/%s%s/%s", resourceURL, resourceID, web.ResourceOperationsURL, operationID)
}

func normalizeCredentials(credentialsJSON json.RawMessage) (map[string][]byte, error) {
	var credentialsMap map[string]interface{}
	err := json.Unmarshal(credentialsJSON, &credentialsMap)
	if err != nil {
		return nil, err
	}

	normalized := make(map[string][]byte)
	for propertyName, value := range credentialsMap {
		keyString := strings.Replace(propertyName, " ", "_", -1)
		// need to re-marshal as json might have complex types, which need to be flattened in strings
		jString, err := json.Marshal(value)
		if err != nil {
			return normalized, err
		}
		// need to remove quotes from flattened objects
		strVal := strings.TrimPrefix(string(jString), "\"")
		strVal = strings.TrimSuffix(strVal, "\"")
		normalized[keyString] = []byte(strVal)
	}
	return normalized, nil
}
