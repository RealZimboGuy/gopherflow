package web

import (
	"sort"
	"strconv"
	"strings"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// ExecutedPath captures which edges and nodes a specific workflow instance
// actually traversed, derived from its WorkflowAction history.
type ExecutedPath struct {
	Edges   [][2]string     // deduped executed (from,to) pairs, in first-seen order
	Visited map[string]bool // every node touched, including the current state
	Current string          // the workflow's current state ("you are here")
}

// extractExecutedPath walks the action history and pulls out the transitions
// that were actually executed. Transitions are recorded by the executor as
// Type:"TRANSITION", Name:<fromState>, Text:"From <from> to <to>".
// Records whose Text does not match that exact shape are skipped defensively.
func extractExecutedPath(actions []domain.WorkflowAction, currentState string) ExecutedPath {
	p := ExecutedPath{Visited: make(map[string]bool)}
	seen := make(map[string]bool)
	for _, a := range actions {
		if a.Type != "TRANSITION" {
			continue
		}
		from := a.Name
		prefix := "From " + from + " to "
		if from == "" || !strings.HasPrefix(a.Text, prefix) {
			continue
		}
		to := strings.TrimPrefix(a.Text, prefix)
		if to == "" {
			continue
		}
		key := from + "\x00" + to
		if !seen[key] {
			seen[key] = true
			p.Edges = append(p.Edges, [2]string{from, to})
		}
		p.Visited[from] = true
		p.Visited[to] = true
	}
	if currentState != "" {
		p.Visited[currentState] = true
	}
	p.Current = currentState
	return p
}

// annotateFlowChart appends Mermaid override directives to a copy of the stored
// definition flowchart so the executed path stands out for one instance:
//   - executed transitions  -> bold blue linkStyle
//   - visited normal nodes  -> blue fill tint (semantic start/end/error/manual
//     nodes keep their meaningful colour)
//   - current state         -> bold blue ring, appended LAST so it always wins
//
// linkStyle indices are resolved by parsing the actual stored text: Mermaid
// numbers links by declaration order, and the only link statements emitted by
// buildFlowChart are bare "<from> --> <to>" lines (no edge labels).
func annotateFlowChart(flowChart string, path ExecutedPath, nodeTypes map[string]models.StateType) string {
	if strings.TrimSpace(flowChart) == "" {
		return flowChart
	}

	edgeIndex := make(map[string]int)
	linkIdx := 0
	for _, ln := range strings.Split(flowChart, "\n") {
		t := strings.TrimSpace(ln)
		if !strings.Contains(t, "-->") {
			continue
		}
		parts := strings.SplitN(t, "-->", 2)
		if len(parts) == 2 {
			from := strings.TrimSpace(parts[0])
			to := strings.TrimSpace(parts[1])
			edgeIndex[from+"\x00"+to] = linkIdx
		}
		linkIdx++
	}

	var b strings.Builder
	b.WriteString(strings.TrimRight(flowChart, "\n"))
	b.WriteString("\n")

	var idxs []string
	for _, e := range path.Edges {
		if i, ok := edgeIndex[e[0]+"\x00"+e[1]]; ok {
			idxs = append(idxs, strconv.Itoa(i))
		}
	}
	if len(idxs) > 0 {
		b.WriteString("    linkStyle " + strings.Join(idxs, ",") + " stroke:#2563eb,stroke-width:3px;\n")
	}

	// Deterministic order for stable output/tests.
	visited := make([]string, 0, len(path.Visited))
	for n := range path.Visited {
		visited = append(visited, n)
	}
	sort.Strings(visited)
	for _, name := range visited {
		if name == path.Current {
			continue
		}
		if nodeTypes[name] == models.StateNormal {
			b.WriteString("    style " + name + " fill:#dbeafe,stroke:#2563eb;\n")
		}
	}

	if path.Current != "" {
		b.WriteString("    style " + path.Current + " stroke:#1d4ed8,stroke-width:4px;\n")
	}

	return b.String()
}
