package main

import (
	"testing"
)

func TestToSRTTime(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "00:00:00,000"},
		{"milliseconds", 123456, "00:00:00,123"},
		{"seconds", 5000000, "00:00:05,000"},
		{"minutes", 125000000, "00:02:05,000"},
		{"hours", 7200000000, "02:00:00,000"},
		{"complex", 3723000123, "01:02:03,000"},
		{"negative", -1000000, "00:00:00,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSRTTime(tt.input)
			if got != tt.expected {
				t.Errorf("toSRTTime(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"no tags", "Hello world", "Hello world"},
		{"html tags", "<div>Hello</div>", "Hello"},
		{"square brackets", "[Hello] world", "Hello world"}, // Updated expectation
		{"html entities", "&lt;Hello&gt; &amp; &quot;world&quot;", "<Hello> & \"world\""},
		{"mixed", "<div>[Hello] &lt;world&gt;</div>", "Hello <world>"}, // Updated expectation
		{"nbsp", "Hello&nbsp;world", "Hello world"},
		{"apos", "Don't", "Don't"},
		{"nested tags", "<b><i>Hello</i></b>", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractText(tt.input)
			if got != tt.expected {
				t.Errorf("extractText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildTextMaterialMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []TextMaterial
		expected map[string]TextMaterial
	}{
		{
			"empty",
			[]TextMaterial{},
			map[string]TextMaterial{},
		},
		{
			"single item",
			[]TextMaterial{
				{ID: "1", Content: "Hello"},
			},
			map[string]TextMaterial{
				"1": {ID: "1", Content: "Hello"},
			},
		},
		{
			"multiple items",
			[]TextMaterial{
				{ID: "1", Content: "Hello"},
				{ID: "2", Content: "World"},
			},
			map[string]TextMaterial{
				"1": {ID: "1", Content: "Hello"},
				"2": {ID: "2", Content: "World"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTextMaterialMap(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("buildTextMaterialMap() length = %d, want %d", len(got), len(tt.expected))
			}
			for k, v := range tt.expected {
				if got[k].ID != v.ID || got[k].Content != v.Content {
					t.Errorf("buildTextMaterialMap()[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
