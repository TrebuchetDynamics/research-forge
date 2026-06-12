package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// UnpaywallConnector looks up open-access metadata for a DOI.
type UnpaywallConnector struct {
	http  HTTPClient
	email string
}

// NewUnpaywallConnector creates an Unpaywall DOI lookup connector.
func NewUnpaywallConnector(httpClient HTTPClient, email string) UnpaywallConnector {
	return UnpaywallConnector{http: httpClient, email: strings.TrimSpace(email)}
}

// Name returns the connector source name.
func (UnpaywallConnector) Name() string { return "unpaywall" }

// OpenAccessRecord is normalized Unpaywall open-access metadata for one DOI.
type OpenAccessRecord struct {
	DOI        string
	OpenAccess bool
	OAStatus   string
	License    string
	BestURL    string
	PDFURL     string
	SourceRef  library.SourceRef
}

// LookupDOI fetches and normalizes Unpaywall metadata for a DOI.
func (c UnpaywallConnector) LookupDOI(ctx context.Context, doi string) (OpenAccessRecord, error) {
	doi = normalizeSourceDOI(doi)
	if doi == "" {
		return OpenAccessRecord{}, fmt.Errorf("doi is required")
	}
	if c.email == "" {
		return OpenAccessRecord{}, fmt.Errorf("unpaywall email is required")
	}
	path := "/v2/" + url.PathEscape(doi)
	body, err := c.http.Get(ctx, path, map[string]string{"email": c.email})
	if err != nil {
		return OpenAccessRecord{}, err
	}
	var payload unpaywallResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return OpenAccessRecord{}, err
	}
	if payload.DOI != "" {
		doi = normalizeSourceDOI(payload.DOI)
	}
	location := payload.BestOALocation
	return OpenAccessRecord{
		DOI:        doi,
		OpenAccess: payload.IsOA,
		OAStatus:   strings.TrimSpace(payload.OAStatus),
		License:    strings.TrimSpace(location.License),
		BestURL:    strings.TrimSpace(location.URL),
		PDFURL:     strings.TrimSpace(location.PDFURL),
		SourceRef: library.SourceRef{
			Source:        "unpaywall",
			RawPayloadRef: "unpaywall:/v2/" + url.PathEscape(doi),
			Metadata: map[string]string{
				"oa_status": strings.TrimSpace(payload.OAStatus),
				"host_type": strings.TrimSpace(location.HostType),
			},
		},
	}, nil
}

type unpaywallResponse struct {
	DOI            string              `json:"doi"`
	IsOA           bool                `json:"is_oa"`
	OAStatus       string              `json:"oa_status"`
	BestOALocation unpaywallOALocation `json:"best_oa_location"`
}

type unpaywallOALocation struct {
	URL      string `json:"url"`
	PDFURL   string `json:"url_for_pdf"`
	License  string `json:"license"`
	HostType string `json:"host_type"`
}
