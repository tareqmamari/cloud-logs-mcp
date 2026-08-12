package server_test

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// goldenToolEntry mirrors one record in testdata/registered_tools.golden.json:
// the name and the JSON-serialized input schema of a registered tool.
type goldenToolEntry struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

// assertServedToolsMatchGolden is the byte-identity safety net for Task 5.
//
// It asserts that the exact set of tools the MCP server serves — every tool
// NAME and its input schema — is identical to the checked-in golden contract
// captured from the pre-refactor server. The whole point of collapsing the
// four drifting tool lists into a single descriptor registry is that this set
// must not change; this test fails loudly if it ever does.
func assertServedToolsMatchGolden(t *testing.T, served []*mcp.Tool) {
	t.Helper()

	raw, err := os.ReadFile("testdata/registered_tools.golden.json")
	if err != nil {
		t.Fatalf("reading golden contract: %v", err)
	}
	var golden []goldenToolEntry
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatalf("parsing golden contract: %v", err)
	}

	// Build the actual served set as sorted {name, schema-json} entries.
	actual := make([]goldenToolEntry, 0, len(served))
	for _, tool := range served {
		sb, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshaling schema for %q: %v", tool.Name, err)
		}
		actual = append(actual, goldenToolEntry{Name: tool.Name, Schema: sb})
	}
	sort.Slice(actual, func(i, j int) bool { return actual[i].Name < actual[j].Name })
	sort.Slice(golden, func(i, j int) bool { return golden[i].Name < golden[j].Name })

	// 1) Hard gate: the sorted NAME list must be identical.
	goldenNames := names(golden)
	actualNames := names(actual)
	if len(goldenNames) != len(actualNames) {
		t.Errorf("served tool COUNT changed: golden=%d actual=%d", len(goldenNames), len(actualNames))
	}
	reportSetDiff(t, goldenNames, actualNames)

	// 2) Extended gate: each tool's input schema must be byte-identical.
	goldenByName := make(map[string]json.RawMessage, len(golden))
	for _, g := range golden {
		goldenByName[g.Name] = g.Schema
	}
	for _, a := range actual {
		g, ok := goldenByName[a.Name]
		if !ok {
			continue // name diff already reported above
		}
		if !equalJSON(g, a.Schema) {
			t.Errorf("input schema for tool %q changed:\n golden: %s\n actual: %s", a.Name, g, a.Schema)
		}
	}
}

func names(entries []goldenToolEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

// reportSetDiff reports names present in only one of the two sorted lists.
func reportSetDiff(t *testing.T, golden, actual []string) {
	t.Helper()
	gset := make(map[string]bool, len(golden))
	for _, n := range golden {
		gset[n] = true
	}
	aset := make(map[string]bool, len(actual))
	for _, n := range actual {
		aset[n] = true
	}
	for _, n := range golden {
		if !aset[n] {
			t.Errorf("tool %q is in golden contract but NO LONGER served (regression)", n)
		}
	}
	for _, n := range actual {
		if !gset[n] {
			t.Errorf("tool %q is served but NOT in golden contract (new/unexpected tool)", n)
		}
	}
}

// equalJSON compares two JSON documents for semantic (order-independent) equality.
func equalJSON(a, b json.RawMessage) bool {
	var av, bv interface{}
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	an, _ := json.Marshal(av)
	bn, _ := json.Marshal(bv)
	return string(an) == string(bn)
}
