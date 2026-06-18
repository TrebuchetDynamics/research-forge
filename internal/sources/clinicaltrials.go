package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ClinicalTrialsConnector searches the ClinicalTrials.gov registry.
type ClinicalTrialsConnector struct {
	http HTTPClient
}

// NewClinicalTrialsConnector creates a ClinicalTrials.gov source connector.
func NewClinicalTrialsConnector(httpClient HTTPClient) ClinicalTrialsConnector {
	return ClinicalTrialsConnector{http: httpClient}
}

// Name returns the connector source name.
func (ClinicalTrialsConnector) Name() string { return "clinicaltrials" }

// Search queries the ClinicalTrials.gov v2 API and normalizes results into SourceRecords.
func (c ClinicalTrialsConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query.term": query.Terms,
		"pageSize":   strconv.Itoa(limit),
		"format":     "json",
	}
	body, err := c.http.Get(ctx, "/api/v2/studies", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload clinicalTrialsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Studies))
	for _, study := range payload.Studies {
		id := study.Protocol
		nctID := strings.TrimSpace(id.Identification.NctID)
		title := strings.TrimSpace(id.Identification.BriefTitle)
		if title == "" {
			title = strings.TrimSpace(id.Identification.OfficialTitle)
		}
		abstract := strings.TrimSpace(id.Description.BriefSummary)
		sponsor := strings.TrimSpace(id.Sponsors.LeadSponsor.Name)
		dateStr := strings.TrimSpace(id.Status.StartDate.Date)
		year := 0
		if len(dateStr) >= 4 {
			year, _ = strconv.Atoi(dateStr[:4])
		}
		htmlURL := fmt.Sprintf("https://clinicaltrials.gov/study/%s", nctID)
		records = append(records, SourceRecord{
			Source:   "clinicaltrials",
			SourceID: nctID,
			Title:    title,
			// NCT IDs are not standard scholarly identifiers; CrossrefID carries
			// the NCT ID so library.PaperRecords passes identifier validation.
			Identifiers: Identifiers{CrossrefID: nctID},
			Year:        year,
			Abstract:    abstract,
			Venue:       "ClinicalTrials.gov",
			URLs:        nonEmptyStrings(htmlURL),
			Metadata: map[string]string{
				"nct_id":         nctID,
				"sponsor":        sponsor,
				"overall_status": strings.TrimSpace(id.Status.OverallStatus),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawClinicalTrialsRef(params)}, nil
}

type clinicalTrialsResponse struct {
	Studies       []clinicalTrialsStudy `json:"studies"`
	NextPageToken string                `json:"nextPageToken"`
}

type clinicalTrialsStudy struct {
	Protocol clinicalTrialsProtocol `json:"protocolSection"`
}

type clinicalTrialsProtocol struct {
	Identification struct {
		NctID         string `json:"nctId"`
		BriefTitle    string `json:"briefTitle"`
		OfficialTitle string `json:"officialTitle"`
	} `json:"identificationModule"`
	Description struct {
		BriefSummary string `json:"briefSummary"`
	} `json:"descriptionModule"`
	Status struct {
		OverallStatus string `json:"overallStatus"`
		StartDate     struct {
			Date string `json:"date"`
		} `json:"startDateStruct"`
	} `json:"statusModule"`
	Sponsors struct {
		LeadSponsor struct {
			Name string `json:"name"`
		} `json:"leadSponsor"`
	} `json:"sponsorCollaboratorsModule"`
}

func rawClinicalTrialsRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"format", "pageSize", "query.term"} {
		if v := strings.TrimSpace(params[key]); v != "" {
			values.Set(key, v)
		}
	}
	return fmt.Sprintf("clinicaltrials:/api/v2/studies?%s", values.Encode())
}
