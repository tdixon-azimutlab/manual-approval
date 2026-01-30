package main

import (
	"testing"
)

func TestParseLabels(t *testing.T) {
	testCases := []struct {
		name           string
		labelsRaw      string
		expectedLabels []string
	}{
		{
			name:           "empty_labels",
			labelsRaw:      "",
			expectedLabels: []string{},
		},
		{
			name:           "single_label",
			labelsRaw:      "bug",
			expectedLabels: []string{"bug"},
		},
		{
			name:           "multiple_labels",
			labelsRaw:      "bug,enhancement,help wanted",
			expectedLabels: []string{"bug", "enhancement", "help wanted"},
		},
		{
			name:           "labels_with_extra_spaces",
			labelsRaw:      "  bug  ,  enhancement  ,  help wanted  ",
			expectedLabels: []string{"bug", "enhancement", "help wanted"},
		},
		{
			name:           "labels_with_empty_entries",
			labelsRaw:      "bug,,enhancement",
			expectedLabels: []string{"bug", "enhancement"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := parseLabels(testCase.labelsRaw)

			if len(actual) != len(testCase.expectedLabels) {
				t.Fatalf("expected %d labels but got %d", len(testCase.expectedLabels), len(actual))
			}

			for i, label := range actual {
				if label != testCase.expectedLabels[i] {
					t.Fatalf("expected label %q but got %q at index %d", testCase.expectedLabels[i], label, i)
				}
			}
		})
	}
}
