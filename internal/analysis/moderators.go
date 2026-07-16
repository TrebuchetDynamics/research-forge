package analysis

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

type ModeratorPreview struct {
	Fields []ModeratorField `json:"fields"`
}

type ModeratorField struct {
	Name        string `json:"name"`
	Papers      int    `json:"papers"`
	Numeric     bool   `json:"numeric"`
	Categorical bool   `json:"categorical"`
}

func SubgroupValuesFromEvidence(items []evidence.EvidenceItem, field string) (map[string]string, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil, fmt.Errorf("moderator field is required")
	}
	values := map[string]string{}
	for _, item := range items {
		if item.Status != evidence.StatusAccepted {
			continue
		}
		value := strings.TrimSpace(item.Values[field])
		if value != "" {
			values[item.PaperID] = value
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no accepted evidence values for moderator field %s", field)
	}
	return values, nil
}

func MetaRegressionValuesFromEvidence(items []evidence.EvidenceItem, field string) (map[string]float64, error) {
	raw, err := SubgroupValuesFromEvidence(items, field)
	if err != nil {
		return nil, err
	}
	values := map[string]float64{}
	for paperID, value := range raw {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return nil, fmt.Errorf("moderator field %s for paper %s is not a finite number", field, paperID)
		}
		values[paperID] = parsed
	}
	return values, nil
}

func ModeratorPreviewFromEvidence(items []evidence.EvidenceItem) ModeratorPreview {
	byField := map[string]map[string]string{}
	for _, item := range items {
		if item.Status != evidence.StatusAccepted {
			continue
		}
		for field, value := range item.Values {
			field = strings.TrimSpace(field)
			value = strings.TrimSpace(value)
			if field == "" || value == "" {
				continue
			}
			if byField[field] == nil {
				byField[field] = map[string]string{}
			}
			byField[field][item.PaperID] = value
		}
	}
	names := make([]string, 0, len(byField))
	for field := range byField {
		names = append(names, field)
	}
	sort.Strings(names)
	preview := ModeratorPreview{}
	for _, field := range names {
		numeric := true
		for _, value := range byField[field] {
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
				numeric = false
				break
			}
		}
		preview.Fields = append(preview.Fields, ModeratorField{Name: field, Papers: len(byField[field]), Numeric: numeric, Categorical: true})
	}
	return preview
}
