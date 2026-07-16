package screening

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type Stage string

const (
	StageTitleAbstract  Stage = "title_abstract"
	StageFullText       Stage = "full_text"
	StageFinalInclusion Stage = "final_inclusion"
)

type Decision string

const (
	DecisionInclude   Decision = "include"
	DecisionExclude   Decision = "exclude"
	DecisionUncertain Decision = "uncertain"
)

type Options struct{ ExclusionReasons []string }

type Workflow struct {
	Stages           []Stage
	Decisions        []Decision
	ExclusionReasons []string
}

func Configure(options Options) (Workflow, error) {
	reasons := []string{}
	seen := map[string]bool{}
	for _, reason := range options.ExclusionReasons {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		if seen[reason] {
			return Workflow{}, fmt.Errorf("duplicate exclusion reason")
		}
		seen[reason] = true
		reasons = append(reasons, reason)
	}
	return Workflow{Stages: []Stage{StageTitleAbstract, StageFullText, StageFinalInclusion}, Decisions: []Decision{DecisionInclude, DecisionExclude, DecisionUncertain}, ExclusionReasons: reasons}, nil
}

func (w Workflow) ValidateReason(reason string) error {
	reason = strings.TrimSpace(reason)
	for _, allowed := range w.ExclusionReasons {
		if reason == allowed {
			return nil
		}
	}
	return fmt.Errorf("unknown exclusion reason")
}

type DecisionInput struct {
	PaperID     string
	Stage       Stage
	Decision    Decision
	Reason      string
	Reviewer    string
	Adjudicated bool
}

func NormalizeDecisionInput(input DecisionInput) DecisionInput {
	input.PaperID = strings.TrimSpace(input.PaperID)
	input.Reviewer = strings.TrimSpace(input.Reviewer)
	input.Reason = strings.TrimSpace(input.Reason)
	return input
}

type DecisionEvent struct {
	PaperID     string
	Stage       Stage
	Decision    Decision
	Reason      string
	Reviewer    string
	Adjudicated bool
}

func NormalizeDecisionEvent(event DecisionEvent) DecisionEvent {
	input := NormalizeDecisionInput(DecisionInput{PaperID: event.PaperID, Reason: event.Reason, Reviewer: event.Reviewer})
	event.PaperID = input.PaperID
	event.Reason = input.Reason
	event.Reviewer = input.Reviewer
	return event
}

type QueueFilter struct {
	Stage    Stage
	Decision Decision
}
type PRISMACounts struct {
	Included  int
	Excluded  int
	Uncertain int
}

type MemoryStore struct {
	workflow Workflow
	events   []DecisionEvent
}

func NewMemoryStore(workflow Workflow) *MemoryStore { return &MemoryStore{workflow: workflow} }

func (s *MemoryStore) Decide(input DecisionInput) error {
	input = NormalizeDecisionInput(input)
	if input.PaperID == "" {
		return fmt.Errorf("paper id is required")
	}
	if input.Reviewer == "" {
		return fmt.Errorf("reviewer is required")
	}
	if !slices.Contains(s.workflow.Stages, input.Stage) {
		return fmt.Errorf("unknown screening stage %q", input.Stage)
	}
	if !slices.Contains(s.workflow.Decisions, input.Decision) {
		return fmt.Errorf("unknown screening decision %q", input.Decision)
	}
	if input.Decision == DecisionExclude {
		if err := s.workflow.ValidateReason(input.Reason); err != nil {
			return err
		}
	}
	s.events = append(s.events, DecisionEvent{PaperID: input.PaperID, Stage: input.Stage, Decision: input.Decision, Reason: input.Reason, Reviewer: input.Reviewer, Adjudicated: input.Adjudicated})
	return nil
}

func (s *MemoryStore) History(paperID string) []DecisionEvent {
	var out []DecisionEvent
	for _, event := range s.events {
		if event.PaperID == paperID {
			out = append(out, event)
		}
	}
	return out
}

func (s *MemoryStore) Conflicts(stage Stage) []string {
	byPaper := map[string]map[Decision]bool{}
	adjudicated := map[string]bool{}
	for _, event := range s.events {
		if event.Stage == stage {
			if event.Adjudicated {
				adjudicated[event.PaperID] = true
			}
			if byPaper[event.PaperID] == nil {
				byPaper[event.PaperID] = map[Decision]bool{}
			}
			byPaper[event.PaperID][event.Decision] = true
		}
	}
	for paper := range adjudicated {
		delete(byPaper, paper)
	}
	var out []string
	for paper, decisions := range byPaper {
		if decisions[DecisionInclude] && decisions[DecisionExclude] {
			out = append(out, paper)
		}
	}
	sort.Strings(out)
	return out
}

func (s *MemoryStore) Queue(filter QueueFilter) []string {
	seen := map[string]bool{}
	for _, event := range s.events {
		if event.Stage == filter.Stage && event.Decision == filter.Decision {
			seen[event.PaperID] = true
		}
	}
	out := make([]string, 0, len(seen))
	for paper := range seen {
		out = append(out, paper)
	}
	sort.Strings(out)
	return out
}

func (s *MemoryStore) PRISMACounts() PRISMACounts {
	counts := PRISMACounts{}
	for _, event := range s.events {
		switch event.Decision {
		case DecisionInclude:
			counts.Included++
		case DecisionExclude:
			counts.Excluded++
		case DecisionUncertain:
			counts.Uncertain++
		}
	}
	return counts
}
