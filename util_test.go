package svcutil

import (
	"reflect"
	"testing"
)

func TestGetJSONTags(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	type WithoutTags struct {
		Name string
		Age  int
	}

	type MixedTags struct {
		Name   string `json:"name"`
		Age    int
		Email  string `json:"email"`
		Active bool
	}

	type EmbeddedStruct struct {
		SimpleStruct
		Address string `json:"address"`
	}

	tests := []struct {
		name     string
		input    any
		expected map[string]string
	}{
		{
			name:  "struct with json tags",
			input: SimpleStruct{Name: "John", Age: 30},
			expected: map[string]string{
				"Name": "name",
				"Age":  "age",
			},
		},
		{
			name:  "pointer to struct with json tags",
			input: &SimpleStruct{Name: "John", Age: 30},
			expected: map[string]string{
				"Name": "name",
				"Age":  "age",
			},
		},
		{
			name:     "struct without json tags",
			input:    WithoutTags{Name: "John", Age: 30},
			expected: map[string]string{},
		},
		{
			name:     "non-struct value",
			input:    "not a struct",
			expected: nil,
		},
		{
			name:     "integer value",
			input:    42,
			expected: nil,
		},
		{
			name:  "struct with mixed tags",
			input: MixedTags{Name: "John", Age: 30, Email: "john@example.com", Active: true},
			expected: map[string]string{
				"Name":  "name",
				"Email": "email",
			},
		},
		{
			name:  "struct with embedded fields",
			input: EmbeddedStruct{SimpleStruct: SimpleStruct{Name: "John", Age: 30}, Address: "123 Main St"},
			expected: map[string]string{
				"Address": "address",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJSONTags(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getJSONTags() = %v, want %v", result, tt.expected)
			}
		})
	}
}
