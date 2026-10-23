package research

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

type RankOptions struct {
	Topic                   string
	Sources                 []string
	Limit                   int
	GraphPath, Out, Project string
}
type Ranking struct {
	Rank  int     `json:"rank"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
	// RawScore preserves the original precision for text rendering.
	RawScore   float64 `json:"-"`
	DOI        string  `json:"doi,omitempty"`
	ArXivID    string  `json:"arxivId,omitempty"`
	Year       int     `json:"year"`
	Venue      string  `json:"venue,omitempty"`
	OpenAccess bool    `json:"openAccess"`
	Source     string  `json:"source"`
}

type RankResult struct {
	Topic    string    `json:"topic"`
	Total    int       `json:"total"`
	Rankings []Ranking `json:"rankings"`
	Out      string    `json:"-"`
}

// Rank searches configured sources, scores papers and commits rankings with a receipt.
func (s Service) Rank(ctx context.Context, opts RankOptions) (RankResult, error) {
	topic, limit, outFile, graphPath := opts.Topic, opts.Limit, opts.Out, opts.GraphPath
	if strings.TrimSpace(topic) == "" {
		return RankResult{}, invalid("usage", "ranking topic is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if outFile == "" {
		outFile = filepath.Join(opts.Project, "rankings", "rankings.json")
	}
	// Load citation graph if provided (exported {nodes:[{id}], edges:[{source,target}]} JSON)
	var prestige map[string]float64
	if graphPath != "" {
		data, err := os.ReadFile(graphPath)
		if err != nil {
			return RankResult{}, failure("rank_graph_read_failed", err)
		}
		var graph struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
			} `json:"edges"`
		}
		if err := json.Unmarshal(data, &graph); err != nil {
			return RankResult{}, failure("rank_graph_parse_failed", err)
		}
		// Prestige = log2(2 + indegree) as a smooth boost for heavily-cited nodes
		indegree := map[string]int{}
		for _, edge := range graph.Edges {
			indegree[edge.Target]++
		}
		prestige = make(map[string]float64, len(indegree))
		for id, count := range indegree {
			prestige[id] = math.Log2(float64(count) + 2)
		}
	}

	// Parse source list
	sourceNames := opts.Sources
	if len(sourceNames) == 0 {
		sourceNames = []string{"openalex", "crossref"}
	}

	// Collect papers from all sources
	type scoredPaper struct {
		record sources.SourceRecord
		score  float64
	}
	seen := map[string]bool{}
	var papers []scoredPaper

	for _, name := range sourceNames {
		name = strings.TrimSpace(name)
		if name != "openalex" && name != "crossref" && name != "semantic-scholar" {
			continue
		}
		if s.Connector == nil {
			continue
		}
		conn, ok := s.Connector(name)
		if !ok || conn == nil {
			continue
		}
		resp, err := conn.Search(ctx, sources.SourceQuery{Terms: topic, Limit: limit})
		if err != nil {
			continue
		}
		for _, rec := range resp.Records {
			if seen[rec.SourceID] {
				continue
			}
			seen[rec.SourceID] = true
			papers = append(papers, scoredPaper{record: rec})
		}
	}

	if len(papers) == 0 {
		return RankResult{}, failure("rank_no_results", fmt.Errorf("no papers found for topic"))
	}

	// Score each paper: citation count + recency + OA bonus + graph prestige
	now := s.now().Year()
	for i, p := range papers {
		citations := 0.0
		if v, err := strconv.ParseFloat(p.record.Metadata["cited_by_count"], 64); err == nil {
			citations = math.Log1p(v) // log scale dampens huge counts
		}
		if v, err := strconv.ParseFloat(p.record.Metadata["citation_count"], 64); err == nil {
			citations = math.Log1p(v)
		}
		recency := 0.0
		if p.record.Year > 0 {
			age := now - p.record.Year
			if age < 0 {
				age = 0
			}
			recency = 1.0 / (1.0 + float64(age))
		}
		oaBonus := 0.0
		if p.record.OpenAccess {
			oaBonus = 0.3
		}
		prestigeBoost := 0.0
		if prestige != nil {
			// Match paper to graph node by DOI, arXiv ID, or OpenAlex ID
			for _, id := range []string{p.record.Identifiers.DOI, p.record.Identifiers.ArXivID, p.record.Identifiers.OpenAlexID, p.record.SourceID} {
				if score, ok := prestige[id]; ok {
					prestigeBoost = score
					break
				}
			}
		}
		papers[i].score = citations*1.5 + recency*2.0 + oaBonus + prestigeBoost
	}

	// Sort by score descending
	sort.Slice(papers, func(i, j int) bool {
		return papers[i].score > papers[j].score
	})

	// Truncate to limit
	if len(papers) > limit {
		papers = papers[:limit]
	}

	// Persist the same complete ranked result regardless of display format.
	items := make([]Ranking, len(papers))
	for i, p := range papers {
		items[i] = Ranking{
			Rank:       i + 1,
			Title:      p.record.Title,
			Score:      math.Round(p.score*100) / 100,
			RawScore:   p.score,
			DOI:        p.record.Identifiers.DOI,
			ArXivID:    p.record.Identifiers.ArXivID,
			Year:       p.record.Year,
			Venue:      p.record.Venue,
			OpenAccess: p.record.OpenAccess,
			Source:     p.record.Source,
		}
	}
	data := RankResult{Topic: topic, Total: len(items), Rankings: items, Out: outFile}
	if outFile != "" {
		if filepath.Base(outFile) == "provenance.json" {
			return RankResult{}, failure("rank_write_failed", fmt.Errorf("rank output must not replace provenance.json; choose a rankings filename"))
		}
		if err := writeJSONArtifact(outFile, data, func() error { return s.writeRankProvenance(outFile, topic, sourceNames, len(papers)) }); err != nil {
			return RankResult{}, failure("rank_write_failed", err)
		}
	}
	return data, nil
}
func (s Service) writeRankProvenance(outFile, topic string, sourceNames []string, count int) error {
	if outFile == "" {
		return nil
	}
	metadata := map[string]any{
		"question":       "rank: " + topic,
		"depth":          "standard",
		"timestamp":      s.now().UTC().Format(time.RFC3339),
		"papers_ranked":  count,
		"sources":        sourceNames,
		"outputs":        []string{outFile},
		"errors":         []string{},
		"rforge_version": s.buildInfo(),
	}
	_, err := s.recordRun(filepath.Dir(outFile), "rank", metadata, []string{outFile}, outFile+".provenance.json")
	return err
}
