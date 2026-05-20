package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// exportNode is the recursive structure written for JSON export.
type exportNode struct {
	Label    string        `json:"label"`
	Children []*exportNode `json:"children,omitempty"`
}

// walkAll calls fn for every node in display order, with its indentation depth.
// It always walks the full dataset, ignoring any active search filter.
func (m *dicomTreeModel) walkAll(fn func(label string, depth int)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.walkLocked("", 0, fn)
}

func (m *dicomTreeModel) walkLocked(parentID string, depth int, fn func(label string, depth int)) {
	n, ok := m.nodes[parentID]
	if !ok {
		return
	}
	for _, childID := range n.children {
		child, ok := m.nodes[childID]
		if !ok {
			continue
		}
		fn(child.label, depth)
		m.walkLocked(childID, depth+1, fn)
	}
}

// exportTree returns the full dataset as a slice of root-level exportNodes.
func (m *dicomTreeModel) exportTree() []*exportNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buildExportNodes("")
}

func (m *dicomTreeModel) buildExportNodes(parentID string) []*exportNode {
	n, ok := m.nodes[parentID]
	if !ok {
		return nil
	}
	result := make([]*exportNode, 0, len(n.children))
	for _, childID := range n.children {
		child, ok := m.nodes[childID]
		if !ok {
			continue
		}
		result = append(result, &exportNode{
			Label:    child.label,
			Children: m.buildExportNodes(childID),
		})
	}
	return result
}

// parseTagLabel splits a formatted tag label into its components.
// Labels follow the pattern "(GGGG,EEEE) [VR] Name: Value" for tag nodes and
// are plain strings for structural nodes (patient, study, series, item).
func parseTagLabel(label string) (tagCode, vr, name, value string) {
	if !strings.HasPrefix(label, "(") {
		return "", "", label, ""
	}
	closeParen := strings.Index(label, ")")
	if closeParen < 0 {
		return "", "", label, ""
	}
	tagCode = label[1:closeParen]
	rest := strings.TrimSpace(label[closeParen+1:])

	if strings.HasPrefix(rest, "[") {
		closeBracket := strings.Index(rest, "]")
		if closeBracket >= 0 {
			vr = rest[1:closeBracket]
			rest = strings.TrimSpace(rest[closeBracket+1:])
		}
	}

	if i := strings.Index(rest, ": "); i >= 0 {
		name = rest[:i]
		value = rest[i+2:]
	} else {
		name = rest
	}
	return
}

// ── Format writers ────────────────────────────────────────────────────────────

func exportText(w io.Writer, model *dicomTreeModel) error {
	model.walkAll(func(label string, depth int) {
		fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", depth), label)
	})
	return nil
}

func exportCSV(w io.Writer, model *dicomTreeModel) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"Depth", "Tag", "VR", "Name", "Value"}); err != nil {
		return err
	}
	model.walkAll(func(label string, depth int) {
		tagCode, vr, name, value := parseTagLabel(label)
		cw.Write([]string{strconv.Itoa(depth), tagCode, vr, name, value}) //nolint:errcheck
	})
	cw.Flush()
	return cw.Error()
}

func exportJSON(w io.Writer, model *dicomTreeModel) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(model.exportTree())
}

// ── File write wrappers ───────────────────────────────────────────────────────

func writeText(path string, model *dicomTreeModel) error {
	return writeExport(path, model, exportText)
}

func writeCSV(path string, model *dicomTreeModel) error {
	return writeExport(path, model, exportCSV)
}

func writeJSON(path string, model *dicomTreeModel) error {
	return writeExport(path, model, exportJSON)
}

func writeExport(path string, model *dicomTreeModel, fn func(io.Writer, *dicomTreeModel) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	werr := fn(f, model)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}
