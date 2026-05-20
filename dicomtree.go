package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

type treeNode struct {
	label       string
	children    []string
	sortKey     float64
	isPrivate   bool
	isMalformed bool
	elmTag      tag.Tag
	hasTag      bool
}

type dicomTreeModel struct {
	mu         sync.RWMutex
	nodes      map[string]*treeNode
	counter    int
	filterText string
}

func newDicomTreeModel() *dicomTreeModel {
	return &dicomTreeModel{
		nodes: map[string]*treeNode{
			"": {label: "", children: nil},
		},
	}
}

func (m *dicomTreeModel) clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = map[string]*treeNode{"": {label: "", children: nil}}
	m.counter = 0
}

func (m *dicomTreeModel) nextID() string {
	m.counter++
	return fmt.Sprintf("%d", m.counter)
}

// setFilter updates the text used to filter the tree. Case-insensitive.
func (m *dicomTreeModel) setFilter(text string) {
	m.mu.Lock()
	m.filterText = strings.ToLower(text)
	m.mu.Unlock()
}

// subtreeMatchesFilter reports whether node id or any descendant has a label
// containing m.filterText. Must be called with m.mu read lock held.
func (m *dicomTreeModel) subtreeMatchesFilter(id string) bool {
	n, ok := m.nodes[id]
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(n.label), m.filterText) {
		return true
	}
	for _, childID := range n.children {
		if m.subtreeMatchesFilter(childID) {
			return true
		}
	}
	return false
}

func (m *dicomTreeModel) childUIDs(id string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[id]
	if !ok {
		return nil
	}
	if m.filterText == "" {
		return n.children
	}
	var visible []string
	for _, childID := range n.children {
		if m.subtreeMatchesFilter(childID) {
			visible = append(visible, childID)
		}
	}
	return visible
}

// tooltipFor returns a formatted DICOM standard description for the node's tag.
// Returns "" for structural nodes (patient/study/series/instance) and unknown tags.
func (m *dicomTreeModel) tooltipFor(id string) string {
	m.mu.RLock()
	n, ok := m.nodes[id]
	m.mu.RUnlock()
	if !ok || !n.hasTag {
		return ""
	}
	info, err := tag.Find(n.elmTag)
	if err != nil {
		return "" // private or non-standard tag — no dictionary entry
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tag:     (%04X,%04X)\n", n.elmTag.Group, n.elmTag.Element)
	if info.Name != "" {
		fmt.Fprintf(&sb, "Name:    %s\n", info.Name)
	}
	if info.Keyword != "" {
		fmt.Fprintf(&sb, "Keyword: %s\n", info.Keyword)
	}
	if len(info.VRs) > 0 {
		fmt.Fprintf(&sb, "VR:      %s\n", strings.Join(info.VRs, ", "))
	}
	if info.VM != "" {
		fmt.Fprintf(&sb, "VM:      %s", info.VM)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m *dicomTreeModel) isBranch(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[id]
	if !ok {
		return false
	}
	return id == "" || len(n.children) > 0
}

func (m *dicomTreeModel) labelFor(id string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n, ok := m.nodes[id]; ok {
		return n.label
	}
	return id
}

func (m *dicomTreeModel) isPrivateNode(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n, ok := m.nodes[id]; ok {
		return n.isPrivate
	}
	return false
}

func (m *dicomTreeModel) isMalformedNode(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n, ok := m.nodes[id]; ok {
		return n.isMalformed
	}
	return false
}

func (m *dicomTreeModel) nodeTag(id string) (tag.Tag, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n, ok := m.nodes[id]; ok {
		return n.elmTag, n.hasTag
	}
	return tag.Tag{}, false
}

func (m *dicomTreeModel) addDataset(ds dicom.Dataset) {
	patientName := stringVal(ds, tag.Tag{Group: 0x0010, Element: 0x0010})
	studyDesc := stringVal(ds, tag.Tag{Group: 0x0008, Element: 0x1030})
	seriesDesc := stringVal(ds, tag.Tag{Group: 0x0008, Element: 0x103E})
	instanceNum := stringVal(ds, tag.Tag{Group: 0x0020, Element: 0x0013})
	sliceLoc := stringVal(ds, tag.Tag{Group: 0x0020, Element: 0x1041})

	sortKey := math.MaxFloat64
	var instanceLabel string
	if sliceLoc != "Unknown" {
		instanceLabel = "Slice location: " + sliceLoc
		if f, err := strconv.ParseFloat(strings.TrimSpace(sliceLoc), 64); err == nil {
			sortKey = f
		}
	} else {
		instanceLabel = "Instance number: " + instanceNum
		if f, err := strconv.ParseFloat(strings.TrimSpace(instanceNum), 64); err == nil {
			sortKey = f
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	patientID := m.findOrCreate("", patientName)
	studyID := m.findOrCreate(patientID, studyDesc)
	seriesID := m.findOrCreate(studyID, seriesDesc)
	instanceID := m.findOrCreateSorted(seriesID, instanceLabel, sortKey)

	for _, el := range ds.Elements {
		m.addElement(instanceID, el)
	}
}

func (m *dicomTreeModel) findOrCreate(parentID, label string) string {
	parent := m.nodes[parentID]
	for _, childID := range parent.children {
		if m.nodes[childID].label == label {
			return childID
		}
	}
	id := m.nextID()
	m.nodes[id] = &treeNode{label: label}
	parent.children = append(parent.children, id)
	return id
}

func (m *dicomTreeModel) findOrCreateSorted(parentID, label string, sortKey float64) string {
	parent := m.nodes[parentID]
	for _, childID := range parent.children {
		if m.nodes[childID].label == label {
			return childID
		}
	}
	id := m.nextID()
	m.nodes[id] = &treeNode{label: label, sortKey: sortKey}
	parent.children = append(parent.children, id)
	sort.Slice(parent.children, func(i, j int) bool {
		return m.nodes[parent.children[i]].sortKey < m.nodes[parent.children[j]].sortKey
	})
	return id
}

// vrMatchesStandard returns false (malformed) when a known tag carries a VR
// that is not in the standard's acceptable list for that tag.
// Empty VR is exempt: it appears when the parser cannot determine the VR
// (e.g. unknown tag in implicit transfer syntax).
// "UN" is intentionally NOT exempt: in explicit VR transfer syntax a known
// public tag encoded as UN is a real VR mismatch and should be flagged.
func vrMatchesStandard(vr string, acceptable []string) bool {
	if vr == "" {
		return true
	}
	for _, v := range acceptable {
		if v == vr {
			return true
		}
	}
	return false
}

func (m *dicomTreeModel) addElement(parentID string, el *dicom.Element) {
	info, err := tag.Find(el.Tag)
	name := info.Name
	if name == "" {
		name = "Unknown"
	}
	prefix := fmt.Sprintf("(%04X,%04X) [%s] %s", el.Tag.Group, el.Tag.Element, el.RawValueRepresentation, name)

	private := el.Tag.Group%2 != 0
	// Malformed: public tag found in the DICOM dictionary but carrying a VR
	// that is not acceptable per the standard. Private tags and tags absent
	// from the dictionary are not flagged — they get other treatment (italic /
	// no highlight) or cannot be checked against a standard entry.
	malformed := !private && err == nil && !vrMatchesStandard(el.RawValueRepresentation, info.VRs)

	if el.RawValueRepresentation == "SQ" {
		seqID := m.nextID()
		m.nodes[seqID] = &treeNode{label: prefix, isPrivate: private, isMalformed: malformed, elmTag: el.Tag, hasTag: true}
		m.nodes[parentID].children = append(m.nodes[parentID].children, seqID)

		if items, ok := el.Value.GetValue().([]*dicom.SequenceItemValue); ok {
			for i, item := range items {
				itemID := m.nextID()
				m.nodes[itemID] = &treeNode{label: fmt.Sprintf("Item %d", i+1)}
				m.nodes[seqID].children = append(m.nodes[seqID].children, itemID)
				if subEls, ok := item.GetValue().([]*dicom.Element); ok {
					for _, subEl := range subEls {
						m.addElement(itemID, subEl)
					}
				}
			}
		}
	} else {
		leafID := m.nextID()
		m.nodes[leafID] = &treeNode{label: prefix + ": " + formatValue(el), isPrivate: private, isMalformed: malformed, elmTag: el.Tag, hasTag: true}
		m.nodes[parentID].children = append(m.nodes[parentID].children, leafID)
	}
}

func stringVal(ds dicom.Dataset, t tag.Tag) string {
	el, err := ds.FindElementByTag(t)
	if err != nil || el == nil {
		return "Unknown"
	}
	vals, ok := el.Value.GetValue().([]string)
	if !ok || len(vals) == 0 || vals[0] == "" {
		return "Unknown"
	}
	return vals[0]
}

func formatValue(el *dicom.Element) string {
	switch vals := el.Value.GetValue().(type) {
	case []string:
		cleaned := make([]string, len(vals))
		for i, s := range vals {
			cleaned[i] = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
		}
		return strings.Join(cleaned, `\`)
	case []int:
		parts := make([]string, len(vals))
		for i, n := range vals {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return strings.Join(parts, ", ")
	case []float64:
		parts := make([]string, len(vals))
		for i, f := range vals {
			parts[i] = fmt.Sprintf("%g", f)
		}
		return strings.Join(parts, ", ")
	case []byte:
		return fmt.Sprintf("[binary, %d bytes]", len(vals))
	case dicom.PixelDataInfo:
		return fmt.Sprintf("[%d frame(s)]", len(vals.Frames))
	default:
		return fmt.Sprintf("%v", el.Value.GetValue())
	}
}
