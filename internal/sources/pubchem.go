package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PubChemConnector searches PubChem for chemical compounds.
//
// PubChem is NCBI's open chemical database (100M+ compounds). The connector
// searches by compound name and returns compound records with IUPAC names,
// molecular formulas, molecular weights, InChIKeys, and SMILES. All PubChem
// data is open (CC0). Compound IDs (CIDs) are stored as CrossrefID.
//
// Two-step query: (1) name→CID list, (2) CID list→properties batch fetch.
type PubChemConnector struct {
	http HTTPClient
}

// NewPubChemConnector creates a PubChem source connector.
func NewPubChemConnector(httpClient HTTPClient) PubChemConnector {
	return PubChemConnector{http: httpClient}
}

// Name returns the connector source name.
func (PubChemConnector) Name() string { return "pubchem" }

// Search queries PubChem by compound name and returns compound SourceRecords.
func (c PubChemConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	encodedQuery := url.PathEscape(query.Terms)
	cidsPath := "/rest/pug/compound/name/" + encodedQuery + "/cids/JSON"
	rawRef := "pubchem:" + cidsPath

	cidsBody, err := c.http.Get(ctx, cidsPath, nil)
	if err != nil {
		return SourceResponse{}, err
	}
	var cidsResp pubchemCIDsResponse
	if err := json.Unmarshal(cidsBody, &cidsResp); err != nil {
		return SourceResponse{}, err
	}
	cids := cidsResp.IdentifierList.CID
	if len(cids) == 0 {
		return SourceResponse{Records: []SourceRecord{}, RawRef: rawRef}, nil
	}
	if len(cids) > limit {
		cids = cids[:limit]
	}

	cidStrs := make([]string, len(cids))
	for i, cid := range cids {
		cidStrs[i] = strconv.Itoa(cid)
	}
	propsPath := "/rest/pug/compound/cid/" + strings.Join(cidStrs, ",") + "/property/IUPACName,MolecularFormula,MolecularWeight,InChIKey,CanonicalSMILES/JSON"
	propsBody, err := c.http.Get(ctx, propsPath, nil)
	if err != nil {
		// Best-effort: return minimal records using CIDs only.
		records := make([]SourceRecord, len(cids))
		for i, cid := range cids {
			cidStr := strconv.Itoa(cid)
			records[i] = SourceRecord{
				Source:      "pubchem",
				SourceID:    cidStr,
				Title:       "CID " + cidStr,
				Identifiers: Identifiers{CrossrefID: "pubchem:" + cidStr},
				OpenAccess:  true,
				URLs:        []string{pubchemURL(cidStr)},
			}
		}
		return SourceResponse{Records: records, RawRef: rawRef}, nil
	}
	var propsResp pubchemPropertiesResponse
	if err := json.Unmarshal(propsBody, &propsResp); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(propsResp.PropertyTable.Properties))
	for _, prop := range propsResp.PropertyTable.Properties {
		cidStr := strconv.Itoa(prop.CID)
		title := strings.TrimSpace(prop.IUPACName)
		if title == "" {
			title = "CID " + cidStr
		}
		abstract := prop.MolecularFormula
		if mw := strings.TrimSpace(prop.MolecularWeight); mw != "" {
			abstract += fmt.Sprintf("; MW: %s g/mol", mw)
		}
		records = append(records, SourceRecord{
			Source:      "pubchem",
			SourceID:    cidStr,
			Title:       title,
			Identifiers: Identifiers{CrossrefID: "pubchem:" + cidStr},
			Abstract:    abstract,
			OpenAccess:  true,
			URLs:        []string{pubchemURL(cidStr)},
			Metadata: map[string]string{
				"molecular_formula": prop.MolecularFormula,
				"molecular_weight":  prop.MolecularWeight,
				"inchikey":          prop.InChIKey,
				"canonical_smiles":  prop.CanonicalSMILES,
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

func pubchemURL(cid string) string {
	return "https://pubchem.ncbi.nlm.nih.gov/compound/" + cid
}

type pubchemCIDsResponse struct {
	IdentifierList struct {
		CID []int `json:"CID"`
	} `json:"IdentifierList"`
}

type pubchemPropertiesResponse struct {
	PropertyTable struct {
		Properties []pubchemProperty `json:"Properties"`
	} `json:"PropertyTable"`
}

type pubchemProperty struct {
	CID              int    `json:"CID"`
	IUPACName        string `json:"IUPACName"`
	MolecularFormula string `json:"MolecularFormula"`
	MolecularWeight  string `json:"MolecularWeight"`
	InChIKey         string `json:"InChIKey"`
	CanonicalSMILES  string `json:"CanonicalSMILES"`
}
