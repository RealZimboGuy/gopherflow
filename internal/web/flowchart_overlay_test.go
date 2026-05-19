package web

import (
	"strings"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func edgeSet(edges [][2]string) map[string]bool {
	m := make(map[string]bool)
	for _, e := range edges {
		m[e[0]+">"+e[1]] = true
	}
	return m
}

func TestExtractExecutedPath_WellFormed(t *testing.T) {
	actions := []domain.WorkflowAction{
		// FindAllByWorkflowID returns newest-first; order must not matter.
		{Type: "TRANSITION", Name: "Process", Text: "From Process to Wait"},
		{Type: "LOG", Name: "Process", Text: "did work"},
		{Type: "TRANSITION", Name: "Start", Text: "From Start to Process"},
		{Type: "TRANSITION", Name: "Process", Text: "From Process to Wait"}, // dup
	}
	p := extractExecutedPath(actions, "Wait")

	es := edgeSet(p.Edges)
	if !es["Start>Process"] || !es["Process>Wait"] {
		t.Fatalf("missing expected edges, got %v", p.Edges)
	}
	if len(p.Edges) != 2 {
		t.Fatalf("expected 2 deduped edges, got %d (%v)", len(p.Edges), p.Edges)
	}
	for _, n := range []string{"Start", "Process", "Wait"} {
		if !p.Visited[n] {
			t.Fatalf("expected %s visited, got %v", n, p.Visited)
		}
	}
	if p.Current != "Wait" {
		t.Fatalf("expected current Wait, got %q", p.Current)
	}
}

func TestExtractExecutedPath_MalformedSkipped(t *testing.T) {
	actions := []domain.WorkflowAction{
		{Type: "TRANSITION", Name: "Start", Text: "garbage no prefix"},
		{Type: "TRANSITION", Name: "Start", Text: "From Start to Process"},
	}
	p := extractExecutedPath(actions, "Process")
	if len(p.Edges) != 1 || p.Edges[0] != [2]string{"Start", "Process"} {
		t.Fatalf("expected only the well-formed edge, got %v", p.Edges)
	}
}

func TestExtractExecutedPath_NoTransitions(t *testing.T) {
	p := extractExecutedPath(nil, "Start")
	if len(p.Edges) != 0 {
		t.Fatalf("expected no edges, got %v", p.Edges)
	}
	if !p.Visited["Start"] || p.Current != "Start" {
		t.Fatalf("expected current state visited, got %v / %q", p.Visited, p.Current)
	}
}

const sampleFlow = `flowchart TD
    Start --> Process
    Process --> Wait
    Wait --> Wait
    Wait --> Done
    classDef normalClass fill:#F0F4F8;
    class Start startClass;
    class Process normalClass;
    class Wait normalClass;
    class Done doneClass;`

func TestAnnotateFlowChart_EdgesNodesCurrent(t *testing.T) {
	path := ExecutedPath{
		Edges:   [][2]string{{"Start", "Process"}, {"Process", "Wait"}},
		Visited: map[string]bool{"Start": true, "Process": true, "Wait": true},
		Current: "Wait",
	}
	nodeTypes := map[string]models.StateType{
		"Start":   models.StateStart,
		"Process": models.StateNormal,
		"Wait":    models.StateNormal,
		"Done":    models.StateEnd,
	}
	out := annotateFlowChart(sampleFlow, path, nodeTypes)

	// Executed edges are link indices 0 (Start-->Process) and 1 (Process-->Wait).
	if !strings.Contains(out, "linkStyle 0,1 stroke:#2563eb,stroke-width:3px;") {
		t.Fatalf("missing/incorrect linkStyle line:\n%s", out)
	}
	// Process is a visited NORMAL non-current node -> gets fill tint.
	if !strings.Contains(out, "style Process fill:#dbeafe,stroke:#2563eb;") {
		t.Fatalf("missing visited-normal style for Process:\n%s", out)
	}
	// Start is visited but a start node -> no fill override.
	if strings.Contains(out, "style Start fill:") {
		t.Fatalf("start node must keep its semantic fill:\n%s", out)
	}
	// Current (Wait) gets the ring and is excluded from the generic-visited pass,
	// and its style line is the LAST appended line.
	if strings.Contains(out, "style Wait fill:#dbeafe") {
		t.Fatalf("current node must not get the generic visited fill:\n%s", out)
	}
	ring := "style Wait stroke:#1d4ed8,stroke-width:4px;"
	if !strings.Contains(out, ring) {
		t.Fatalf("missing current-state ring:\n%s", out)
	}
	if strings.TrimSpace(out[strings.LastIndex(out, "style "):]) != ring {
		t.Fatalf("current-state ring must be the last appended line:\n%s", out)
	}
}

func TestAnnotateFlowChart_NoHistoryOnlyRing(t *testing.T) {
	path := ExecutedPath{
		Visited: map[string]bool{"Start": true},
		Current: "Start",
	}
	out := annotateFlowChart(sampleFlow, path, map[string]models.StateType{"Start": models.StateStart})
	if strings.Contains(out, "linkStyle") {
		t.Fatalf("expected no linkStyle with empty history:\n%s", out)
	}
	if !strings.Contains(out, "style Start stroke:#1d4ed8,stroke-width:4px;") {
		t.Fatalf("expected current-state ring even with no transitions:\n%s", out)
	}
}

func TestAnnotateFlowChart_EmptyFlowChartUnchanged(t *testing.T) {
	if got := annotateFlowChart("", ExecutedPath{Current: "X"}, nil); got != "" {
		t.Fatalf("empty flowchart must be returned unchanged, got %q", got)
	}
}
