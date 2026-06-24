package agents

import (
	"reflect"
	"testing"
)

var realOpencodeAgentListOutput = `build (primary)
  [
  {
    "permission": "*",
    "action": "allow",
    "pattern": "*"
  }
  ]
compaction (primary)
  [
  {
    "permission": "read",
    "action": "allow",
    "pattern": "*"
  }
  ]
explore (subagent)
  [
  {
    "permission": "read",
    "action": "allow",
    "pattern": "*"
  }
  ]
chat (primary)
  [
  {
    "permission": "*",
    "action": "allow",
    "pattern": "*"
  }
  ]
code-reviewer (subagent)
  [
  {
    "permission": "read",
    "action": "allow",
    "pattern": "*"
  }
  ]
`

func TestParseModes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "RealOpencodeOutput",
			input:    realOpencodeAgentListOutput,
			expected: []string{"build", "compaction", "explore", "chat", "code-reviewer"},
		},
		{
			name:     "SingleMode",
			input:    "build (primary)\n  [\n  ]\n",
			expected: []string{"build"},
		},
		{
			name:     "EmptyInput",
			input:    "",
			expected: []string{},
		},
		{
			name:     "NoParens",
			input:    "build\nchat\n",
			expected: []string{"build", "chat"},
		},
		{
			name:     "Duplicates",
			input:    "build (primary)\nbuild (primary)\n",
			expected: []string{"build"},
		},
		{
			name:     "JSONBrackets",
			input:    "]\n]\n[\n{\n}\n",
			expected: []string{},
		},
		{
			name:     "MixedIndentation",
			input:    "  indented (sub)\nreal (primary)\n",
			expected: []string{"real"},
		},
		{
			name:     "CarriageReturns",
			input:    "build (primary)\r\nchat (primary)\r\n",
			expected: []string{"build", "chat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModes(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseModes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "SimpleList",
			input:    "model-a\nmodel-b\nmodel-c\n",
			expected: []string{"model-a", "model-b", "model-c"},
		},
		{
			name:     "EmptyInput",
			input:    "",
			expected: []string{},
		},
		{
			name:     "WhitespaceLines",
			input:    "  \nmodel-a\n  \nmodel-b\n",
			expected: []string{"model-a", "model-b"},
		},
		{
			name:     "CarriageReturns",
			input:    "model-a\r\nmodel-b\r\n",
			expected: []string{"model-a", "model-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseLines() = %v, want %v", got, tt.expected)
			}
		})
	}
}
