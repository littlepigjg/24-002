// Package jsonutil provides JSON read/write/validation utilities.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ReadJSON reads and decodes a JSON request body into the target value.
func ReadJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	defer r.Body.Close()
	return DecodeJSON(r.Body, target)
}

// DecodeJSON decodes JSON from a reader into the target value.
func DecodeJSON(reader io.Reader, target interface{}) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read JSON: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("empty JSON body")
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// WriteJSON encodes a value as JSON and writes it to the HTTP response.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// EncodeJSON encodes a value as a JSON byte slice.
func EncodeJSON(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// DecodeJSONBytes decodes a JSON byte slice into the target value.
func DecodeJSONBytes(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

// PrettyFormat formats a JSON string with indentation.
func PrettyFormat(data interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Validate checks if a string is valid JSON.
func Validate(s string) bool {
	if s == "" {
		return false
	}
	var jsonData interface{}
	err := json.Unmarshal([]byte(s), &jsonData)
	return err == nil
}

// IsJSONObject checks if a byte slice is a JSON object (starts with '{').
func IsJSONObject(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// IsJSONArray checks if a byte slice is a JSON array (starts with '[').
func IsJSONArray(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '['
}

// MergeObjects merges multiple JSON objects into one.
// Later values override earlier ones for the same key.
func MergeObjects(objects ...string) (string, error) {
	merged := make(map[string]interface{})

	for _, obj := range objects {
		if obj == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(obj), &m); err != nil {
			return "", fmt.Errorf("failed to merge JSON object: %w", err)
		}
		for k, v := range m {
			merged[k] = v
		}
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged JSON: %w", err)
	}
	return string(result), nil
}

// GetString extracts a string value from a JSON string by key.
func GetString(jsonStr, key string) (string, bool) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", false
	}

	val, ok := obj[key]
	if !ok {
		return "", false
	}

	str, ok := val.(string)
	return str, ok
}

// GetInt extracts an integer value from a JSON string by key.
func GetInt(jsonStr, key string) (int, bool) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return 0, false
	}

	val, ok := obj[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

// GetNestedString extracts a nested string value using a dot-separated path.
func GetNestedString(jsonStr, path string) (string, bool) {
	parts := strings.Split(path, ".")
	var current interface{}

	if err := json.Unmarshal([]byte(jsonStr), &current); err != nil {
		return "", false
	}

	for _, part := range parts {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return "", false
		}
		current, ok = obj[part]
		if !ok {
			return "", false
		}
	}

	str, ok := current.(string)
	return str, ok
}

// MapFromJSON creates a map from a JSON string.
func MapFromJSON(jsonStr string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return nil, err
	}
	return m, nil
}
