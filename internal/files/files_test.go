package files

import (
	"testing"
)

func Test_FindSemverActions(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected []SemverAction
	}{
		{
			name:     "no semver actions",
			input:    "foo bar",
			expected: nil,
		},
		{
			name:     "one semver action",
			input:    "uses: foo/bar@v1.2.3",
			expected: []SemverAction{{Action: "foo/bar", Version: "v1.2.3", Full: "uses: foo/bar@v1.2.3"}},
		},
		{
			name: "one semver action with content around",
			input: `before content
					uses: foo/bar@v1.2.3
					after content`,
			expected: []SemverAction{{Action: "foo/bar", Version: "v1.2.3", Full: "uses: foo/bar@v1.2.3"}},
		},
		{
			name: "several semver actions",
			input: `uses: foo/bar@v1.2.3
 				    uses: baz/qux@v4.5.6`,
			expected: []SemverAction{
				{Action: "foo/bar", Version: "v1.2.3", Full: "uses: foo/bar@v1.2.3"},
				{Action: "baz/qux", Version: "v4.5.6", Full: "uses: baz/qux@v4.5.6"},
			},
		},
		{
			name:  "semver sub-action",
			input: "uses: foo/bar/sub@v1.2.3",
			expected: []SemverAction{
				{Action: "foo/bar", Version: "v1.2.3", Full: "uses: foo/bar/sub@v1.2.3"},
			},
		},
		{
			name: "local action",
			input: `uses: ./foo/bar
					uses: baz/qux@v4.5.6`,
			expected: []SemverAction{
				{Action: "baz/qux", Version: "v4.5.6", Full: "uses: baz/qux@v4.5.6"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSemverActions(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d semver actions, got %d", len(tt.expected), len(got))
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Fatalf("expected semver action %q, got %q", tt.expected[i], v)
				}
			}
		})
	}
}
