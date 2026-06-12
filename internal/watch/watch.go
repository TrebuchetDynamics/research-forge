package watch

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

type Input struct {
	Name     string
	Source   string
	Query    string
	Interval string
}
type WatchedSearch struct {
	Name     string
	Source   string
	Query    string
	Interval string
}

func NewWatchedSearch(input Input) (WatchedSearch, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Query) == "" {
		return WatchedSearch{}, fmt.Errorf("watched search name and query are required")
	}
	return WatchedSearch{Name: input.Name, Source: input.Source, Query: input.Query, Interval: input.Interval}, nil
}

type Paper struct {
	ID    string
	Title string
}
type InboxItem struct {
	Paper
	PDFApproved bool
}
type Inbox struct{ items []InboxItem }

func NewInbox() *Inbox             { return &Inbox{} }
func (i *Inbox) Add(p Paper)       { i.items = append(i.items, InboxItem{Paper: p}) }
func (i *Inbox) List() []InboxItem { return append([]InboxItem{}, i.items...) }
func (i *Inbox) ApprovePDF(id string) error {
	for idx := range i.items {
		if i.items[idx].ID == id {
			i.items[idx].PDFApproved = true
			return nil
		}
	}
	return fmt.Errorf("paper not found")
}

type RefreshRun struct {
	WatchedName string
	NewCount    int
}

func Refresh(search WatchedSearch, papers []Paper, inbox *Inbox) RefreshRun {
	for _, p := range papers {
		inbox.Add(p)
	}
	return RefreshRun{WatchedName: search.Name, NewCount: len(papers)}
}
func (r RefreshRun) ProvenanceEvent() provenance.Event {
	return provenance.Event{SchemaVersion: "1", Action: "watch.refresh", Inputs: map[string]any{"watch": r.WatchedName}, Outputs: map[string]any{"newCount": r.NewCount}}
}
