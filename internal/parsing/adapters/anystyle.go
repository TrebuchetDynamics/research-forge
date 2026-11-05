package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/text"
	"os/exec"
	"strings"
)

// CommandRunner runs an optional external parser command.
type CommandRunner interface {
	Run(ctx context.Context, input []byte) ([]byte, error)
}

// ExecCommandRunner invokes an external command and passes the bibliography on stdin.
type ExecCommandRunner struct{ Command []string }

func (r ExecCommandRunner) Run(ctx context.Context, input []byte) ([]byte, error) {
	if len(r.Command) == 0 || strings.TrimSpace(r.Command[0]) == "" {
		return nil, fmt.Errorf("anystyle command is required")
	}
	cmd := exec.CommandContext(ctx, r.Command[0], r.Command[1:]...)
	cmd.Stdin = strings.NewReader(string(input))
	return cmd.Output()
}

// AnyStyleReferenceParser adapts Anystyle-like JSON reference output into model.ParsedDocument references.
type AnyStyleReferenceParser struct {
	Runner  CommandRunner
	Version string
}

func (p AnyStyleReferenceParser) ParseReferences(ctx context.Context, paperID string, bibliography []byte) (model.ParsedDocument, error) {
	if p.Runner == nil {
		return model.ParsedDocument{}, fmt.Errorf("anystyle runner is required")
	}
	out, err := p.Runner.Run(ctx, bibliography)
	if err != nil {
		return model.ParsedDocument{}, err
	}
	var refs []anyStyleReference
	if err := json.Unmarshal(out, &refs); err != nil {
		return model.ParsedDocument{}, err
	}
	version := p.Version
	if version == "" {
		version = "external"
	}
	doc := model.ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(paperID), ParserName: "anystyle", ParserVersion: version}
	for _, ref := range refs {
		title := text.CompactSpace(firstNonEmptyAnyStyle(ref.Title, ref.ContainerTitle, ref.Raw))
		doi := text.NormalizeReferenceDOI(ref.DOI)
		raw := text.CompactSpace(ref.Raw)
		if title == "" && doi == "" && raw == "" {
			continue
		}
		doc.References = append(doc.References, model.Reference{Title: title, DOI: doi, Raw: raw, Confidence: ref.Confidence})
	}
	if len(doc.References) == 0 {
		doc.Warnings = append(doc.Warnings, "no references parsed")
	}
	return model.EnrichParsedDocumentModel(doc), nil
}

type anyStyleReference struct {
	Title          string  `json:"title"`
	ContainerTitle string  `json:"container-title"`
	DOI            string  `json:"doi"`
	Raw            string  `json:"raw"`
	Confidence     float64 `json:"confidence"`
}

func firstNonEmptyAnyStyle(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
