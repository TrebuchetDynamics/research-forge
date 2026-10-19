package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executeRank(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge rank <topic> [--sources openalex,crossref,semantic-scholar] [--limit N] [--out <file>] [--graph <graph.json>]")
	}

	topic := args[0]
	limit := 20
	outFile := ""
	sourceList := "openalex,crossref"
	graphPath := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				v, err := strconv.Atoi(args[i+1])
				if err == nil && v > 0 {
					limit = v
				}
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		case "--sources":
			if i+1 < len(args) {
				sourceList = args[i+1]
				i++
			}
		case "--graph":
			if i+1 < len(args) {
				graphPath = args[i+1]
				i++
			}
		}
	}

	// Even stdout-oriented runs retain checksummed output and a receipt.
	// Keep default artifacts in a dedicated directory, not the topic index.
	if outFile == "" {
		outFile = filepath.Join(opts.Project, "rankings", "rankings.json")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Load citation graph if provided (exported {nodes:[{id}], edges:[{source,target}]} JSON)
	var prestige map[string]float64
	if graphPath != "" {
		data, err := os.ReadFile(graphPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "rank_graph_read_failed", err.Error())
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
			return writeError(stdout, stderr, opts, 1, "rank_graph_parse_failed", err.Error())
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
	sourceNames := strings.Split(sourceList, ",")

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
		conn, _ := searchConnector(name)
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
		return writeError(stdout, stderr, opts, 1, "rank_no_results", "no papers found for topic")
	}

	// Score each paper: citation count + recency + OA bonus + graph prestige
	now := time.Now().Year()
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
	type rankItem struct {
		Rank       int     `json:"rank"`
		Title      string  `json:"title"`
		Score      float64 `json:"score"`
		DOI        string  `json:"doi,omitempty"`
		ArXivID    string  `json:"arxivId,omitempty"`
		Year       int     `json:"year"`
		Venue      string  `json:"venue,omitempty"`
		OpenAccess bool    `json:"openAccess"`
		Source     string  `json:"source"`
	}
	items := make([]rankItem, len(papers))
	for i, p := range papers {
		items[i] = rankItem{
			Rank:       i + 1,
			Title:      p.record.Title,
			Score:      math.Round(p.score*100) / 100,
			DOI:        p.record.Identifiers.DOI,
			ArXivID:    p.record.Identifiers.ArXivID,
			Year:       p.record.Year,
			Venue:      p.record.Venue,
			OpenAccess: p.record.OpenAccess,
			Source:     p.record.Source,
		}
	}
	data := map[string]any{
		"topic":    topic,
		"total":    len(items),
		"rankings": items,
	}
	if outFile != "" {
		if filepath.Base(outFile) == "provenance.json" {
			return writeError(stdout, stderr, opts, 1, "rank_write_failed", "rank output must not replace provenance.json; choose a rankings filename")
		}
		if err := writeJSONFileThen(outFile, data, func() error { return writeRankProvenance(outFile, topic, sourceNames, len(papers)) }); err != nil {
			return writeError(stdout, stderr, opts, 1, "rank_write_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, data)
	}

	// Text output
	title := fmt.Sprintf("PaperRank: %s", topic)
	fmt.Fprintf(stdout, "%s\n%s\n\n", title, strings.Repeat("=", len(title)))
	for i, p := range papers {
		symbol := ""
		if p.record.OpenAccess {
			symbol = " 🔓"
		}
		fmt.Fprintf(stdout, "%3d. [%.2f] %s%s\n", i+1, p.score, p.record.Title, symbol)
		if p.record.Identifiers.DOI != "" {
			fmt.Fprintf(stdout, "       DOI: %s\n", p.record.Identifiers.DOI)
		}
		if p.record.Identifiers.ArXivID != "" {
			fmt.Fprintf(stdout, "       arXiv: %s\n", p.record.Identifiers.ArXivID)
		}
		if p.record.Year > 0 {
			fmt.Fprintf(stdout, "       Year: %d", p.record.Year)
			if p.record.Venue != "" {
				fmt.Fprintf(stdout, " — %s", p.record.Venue)
			}
			fmt.Fprintln(stdout)
		}
	}

	if outFile != "" {
		fmt.Fprintf(stdout, "\nrankings saved to %s\n", outFile)
	}

	return 0
}

func writeRankProvenance(outFile, topic string, sourceNames []string, count int) error {
	if outFile == "" {
		return nil
	}
	metadata := map[string]any{
		"question":       "rank: " + topic,
		"depth":          "standard",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"papers_ranked":  count,
		"sources":        sourceNames,
		"outputs":        []string{outFile},
		"errors":         []string{},
		"rforge_version": map[string]string{"version": Version, "commit": Commit, "date": Date},
	}
	_, err := recordEngineRun(filepath.Dir(outFile), "rank", metadata, []string{outFile}, outFile+".provenance.json")
	return err
}
