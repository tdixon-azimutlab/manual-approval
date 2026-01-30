package main

import (
	"strings"
)

func parseLabels(labelsRaw string) []string {
	if strings.TrimSpace(labelsRaw) == "" {
		return []string{}
	}

	rawLabels := strings.Split(labelsRaw, ",")
	labels := []string{}

	for _, label := range rawLabels {
		trimmed := strings.TrimSpace(label)
		if trimmed != "" {
			labels = append(labels, trimmed)
		}
	}

	return labels
}
