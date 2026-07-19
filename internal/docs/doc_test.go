// Package docs holds drift tests that keep docs/backend.md, docs/frontend.md,
// and docs/service.md honest. Each doc carries CONTRACT blocks; these tests
// assert those blocks equal the live source of truth (registered routes,
// service names, CLI subcommands). A drifting change fails here — and the
// pre-commit hook (scripts/git-hooks/pre-commit) blocks the commit.
//
// The frontend routes contract is checked separately in the Vue suite
// (dashboard/src/docs/frontend.doc.test.js) where the router is importable.
//
// Paths are relative to this package dir (internal/docs); repo root is ../..
package docs

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	agentsvc "arcvault/agent/service"
	coordsvc "arcvault/coordinator/service"
)

const (
	backendDoc      = "../../docs/backend.md"
	serviceDoc      = "../../docs/service.md"
	serverGo        = "../../coordinator/server/server.go"
	coordinatorMain = "../../coordinator/main.go"
	agentMain       = "../../agent/main.go"
)

// parseContract returns the items inside <!-- CONTRACT:name --> ... <!-- /CONTRACT:name -->.
// Each item is a markdown bullet ("- ..."); surrounding backticks and spaces are stripped.
func parseContract(t *testing.T, mdPath, name string) []string {
	t.Helper()
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read %s: %v", mdPath, err)
	}
	begin := "<!-- CONTRACT:" + name
	end := "<!-- /CONTRACT:" + name + " -->"
	var items []string
	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, begin) {
				inBlock = true
			}
			continue
		}
		if trimmed == end {
			return items
		}
		if strings.HasPrefix(trimmed, "-") {
			item := strings.Trim(strings.TrimPrefix(trimmed, "-"), " `")
			if item != "" {
				items = append(items, item)
			}
		}
	}
	if !inBlock {
		t.Fatalf("%s: CONTRACT block %q not found", mdPath, name)
	}
	t.Fatalf("%s: CONTRACT block %q not closed with %q", mdPath, name, end)
	return nil
}

var routeRe = regexp.MustCompile(`HandleFunc\("([A-Z]+ [^"]+)"`)

// extractRoutes returns every "METHOD /path" registered via router.HandleFunc in goSrc.
func extractRoutes(t *testing.T, goSrc string) []string {
	t.Helper()
	data, err := os.ReadFile(goSrc)
	if err != nil {
		t.Fatalf("read %s: %v", goSrc, err)
	}
	var out []string
	for _, m := range routeRe.FindAllStringSubmatch(string(data), -1) {
		out = append(out, m[1])
	}
	return out
}

var caseStrRe = regexp.MustCompile(`"([^"]+)"`)

// extractSubcommands returns every string literal on a `case "...":` line — i.e. the
// CLI subcommands dispatched from main()'s switch over os.Args[1].
func extractSubcommands(t *testing.T, goSrc string) []string {
	t.Helper()
	data, err := os.ReadFile(goSrc)
	if err != nil {
		t.Fatalf("read %s: %v", goSrc, err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "case \"") {
			continue
		}
		for _, m := range caseStrRe.FindAllStringSubmatch(trimmed, -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// assertSetEqual fails if the documented set differs from the source-of-truth set,
// printing the diff and the exact corrected block to paste back into the doc.
func assertSetEqual(t *testing.T, contract string, documented, actual []string) {
	t.Helper()
	docSet := toSet(documented)
	actSet := toSet(actual)

	var missing, extra []string // missing: in code, absent from doc. extra: in doc, absent from code.
	for k := range actSet {
		if !docSet[k] {
			missing = append(missing, k)
		}
	}
	for k := range docSet {
		if !actSet[k] {
			extra = append(extra, k)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return
	}
	sort.Strings(missing)
	sort.Strings(extra)

	corrected := make([]string, len(actual))
	copy(corrected, actual)
	sort.Strings(corrected)
	var block strings.Builder
	for _, item := range corrected {
		fmt.Fprintf(&block, "- `%s`\n", item)
	}

	t.Errorf(`CONTRACT:%s in the doc is out of sync with the code.
  in code but NOT documented (add these): %v
  documented but NOT in code (remove these): %v

Corrected block — replace the CONTRACT:%s bullets with:
%s`, contract, missing, extra, contract, block.String())
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// --- Backend ---------------------------------------------------------------

func TestBackendDoc_routesMatchRegistered(t *testing.T) {
	documented := parseContract(t, backendDoc, "routes")
	actual := extractRoutes(t, serverGo)
	if len(actual) == 0 {
		t.Fatalf("no routes extracted from %s — regex out of date?", serverGo)
	}
	assertSetEqual(t, "routes", documented, actual)
}

// --- Service ---------------------------------------------------------------

func TestServiceDoc_serviceNames(t *testing.T) {
	documented := parseContract(t, serviceDoc, "service-names")
	actual := []string{coordsvc.CoordinatorServiceName, agentsvc.AgentServiceName}
	assertSetEqual(t, "service-names", documented, actual)
}

func TestServiceDoc_coordinatorCommands(t *testing.T) {
	documented := parseContract(t, serviceDoc, "coordinator-commands")
	actual := extractSubcommands(t, coordinatorMain)
	if len(actual) == 0 {
		t.Fatalf("no subcommands extracted from %s", coordinatorMain)
	}
	assertSetEqual(t, "coordinator-commands", documented, actual)
}

func TestServiceDoc_agentCommands(t *testing.T) {
	documented := parseContract(t, serviceDoc, "agent-commands")
	actual := extractSubcommands(t, agentMain)
	if len(actual) == 0 {
		t.Fatalf("no subcommands extracted from %s", agentMain)
	}
	assertSetEqual(t, "agent-commands", documented, actual)
}
