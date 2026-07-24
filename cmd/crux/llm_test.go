package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ideamans/go-llm-cli-kit/llmcmd"

	"github.com/ideamans/crux-cli/internal/llmdocs"
)

func TestEmbeddedReference(t *testing.T) {
	g, err := llmdocs.Docs().Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	for _, want := range []string{
		"reference for AI agents",
		"chrome-ux-report.materialized.device_summary", // data source
		"CRUX_API_KEY",                                 // auth
		"Core Web Vitals threshold",                    // metrics chapter
		"JSON output schemas",                          // schema chapter
		"Gotchas and limits",                           // gotchas chapter
		"# Command catalog",                            // generated
		"`crux device`",                                // a known command
	} {
		if !strings.Contains(g, want) {
			t.Errorf("embedded reference missing %q", want)
		}
	}
}

func TestChapterOrder(t *testing.T) {
	sections, err := llmdocs.Docs().Sections()
	if err != nil {
		t.Fatalf("Sections: %v", err)
	}
	var files []string
	for _, s := range sections {
		files = append(files, s.File)
	}
	want := "00-guide.md,10-metrics.md,20-schemas.md,30-gotchas.md,90-commands.md"
	if got := strings.Join(files, ","); got != want {
		t.Errorf("chapters = %s, want %s", got, want)
	}
}

// TestLegacyLLMFlag guards the compatibility promise: --llm used to work at any
// position on the command line, and callers still rely on it.
func TestLegacyLLMFlag(t *testing.T) {
	for _, args := range [][]string{{"--llm"}, {"device", "-o", "https://example.com", "--llm"}} {
		var out bytes.Buffer
		handled, err := llmcmd.HandleLegacy(args, llmConfig(), &out)
		if err != nil {
			t.Fatalf("HandleLegacy(%v): %v", args, err)
		}
		if !handled {
			t.Errorf("HandleLegacy(%v) did not handle --llm", args)
		}
		if !strings.Contains(out.String(), "CRUX_API_KEY") {
			t.Errorf("HandleLegacy(%v) printed the wrong thing", args)
		}
	}
}
