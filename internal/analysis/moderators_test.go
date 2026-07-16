package analysis

import (
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestModeratorValuesFromEvidenceImproveSubgroupAndMetaRegressionUX(t *testing.T) {
	items := []evidence.EvidenceItem{
		{PaperID: "p1", Status: evidence.StatusAccepted, Values: map[string]string{"region": "EU", "dose": "1.5"}},
		{PaperID: "p2", Status: evidence.StatusAccepted, Values: map[string]string{"region": "US", "dose": "2.5"}},
		{PaperID: "p3", Status: evidence.StatusSuggested, Values: map[string]string{"region": "ignored", "dose": "9"}},
	}
	subgroups, err := SubgroupValuesFromEvidence(items, "region")
	if err != nil {
		t.Fatalf("SubgroupValuesFromEvidence: %v", err)
	}
	if subgroups["p1"] != "EU" || subgroups["p2"] != "US" || subgroups["p3"] != "" {
		t.Fatalf("subgroups = %#v", subgroups)
	}
	values, err := MetaRegressionValuesFromEvidence(items, "dose")
	if err != nil {
		t.Fatalf("MetaRegressionValuesFromEvidence: %v", err)
	}
	if values["p1"] != 1.5 || values["p2"] != 2.5 {
		t.Fatalf("values = %#v", values)
	}
	preview := ModeratorPreviewFromEvidence(items)
	if len(preview.Fields) != 2 || preview.Fields[0].Name != "dose" || preview.Fields[1].Name != "region" {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestMetaRegressionValuesFromEvidenceRejectsNonfiniteModerator(t *testing.T) {
	items := []evidence.EvidenceItem{{PaperID: "p1", Status: evidence.StatusAccepted, Values: map[string]string{"dose": "NaN"}}}
	if _, err := MetaRegressionValuesFromEvidence(items, "dose"); err == nil {
		t.Fatal("MetaRegressionValuesFromEvidence returned nil error for a non-finite moderator")
	}
}

func TestModeratorPreviewDoesNotClassifyNonfiniteValuesAsNumeric(t *testing.T) {
	items := []evidence.EvidenceItem{{PaperID: "p1", Status: evidence.StatusAccepted, Values: map[string]string{"dose": "NaN"}}}
	preview := ModeratorPreviewFromEvidence(items)
	if len(preview.Fields) != 1 {
		t.Fatalf("preview fields = %#v", preview.Fields)
	}
	if preview.Fields[0].Numeric {
		t.Fatalf("non-finite moderator classified as numeric: %#v", preview.Fields[0])
	}
}
