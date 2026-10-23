package research_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

type rankInputSource func(context.Context, sources.SourceQuery) (sources.SourceResponse, error)

func (f rankInputSource) Search(ctx context.Context, query sources.SourceQuery) (sources.SourceResponse, error) {
	return f(ctx, query)
}

// Catches blank topics reaching graph reads, connector selection or artifact writes.
func TestRankRejectsBlankTopicBeforeSideEffects(t *testing.T) {
	for _, topic := range []string{"", "   ", "\t", "\n", "\r\n\t ", "\u0085\u00a0\u1680\u2003\u2028\u2029\u202f\u205f\u3000"} {
		for _, withGraph := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/graph=%t", topic, withGraph), func(t *testing.T) {
				dir := t.TempDir()
				selected, searched := 0, 0
				s := research.Service{Connector: func(string) (research.SearchConnector, bool) {
					selected++
					return rankInputSource(func(ctx context.Context, query sources.SourceQuery) (sources.SourceResponse, error) {
						searched++
						return (rankedSource{}).Search(ctx, query)
					}), true
				}}
				opts := research.RankOptions{Topic: topic, Sources: []string{"openalex"}, Project: dir, Out: filepath.Join(dir, "results", "rankings.json")}
				if withGraph {
					opts.GraphPath = filepath.Join(dir, "missing-graph.json")
				}
				_, err := s.Rank(context.Background(), opts)
				var operation *research.Error
				if !errors.As(err, &operation) || operation.Code != "usage" || operation.ExitCode != 2 {
					t.Errorf("expected typed usage error/exit 2, got %v", err)
				}
				if selected != 0 || searched != 0 {
					t.Errorf("blank topic selected %d connectors and searched %d times", selected, searched)
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Errorf("blank topic left graph/result/provenance side effects: %v, %v", entries, err)
				}
			})
		}
	}

	t.Run("preserve nonblank query bytes", func(t *testing.T) {
		const topic = " \t scaling \u2003\n"
		var terms []string
		s := research.Service{Connector: func(string) (research.SearchConnector, bool) {
			return rankInputSource(func(ctx context.Context, query sources.SourceQuery) (sources.SourceResponse, error) {
				terms = append(terms, query.Terms)
				return (rankedSource{}).Search(ctx, query)
			}), true
		}}
		got, err := s.Rank(context.Background(), research.RankOptions{
			Topic: topic, Sources: []string{"openalex"}, Out: filepath.Join(t.TempDir(), "rankings.json"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(terms) != 1 || terms[0] != topic || got.Topic != topic {
			t.Fatalf("query bytes changed: terms=%q, result topic=%q", terms, got.Topic)
		}
	})
}
