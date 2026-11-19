package bq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/api/option"
)

// recordingProvider is a TracerProvider that remembers the names of the spans
// started through it. The BigQuery client creates its spans through the global
// OpenTelemetry provider, so this observes the otel path without an exporter.
type recordingProvider struct {
	noop.TracerProvider
	mu    sync.Mutex
	spans []string
}

func (p *recordingProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return &recordingTracer{Tracer: p.TracerProvider.Tracer(name, opts...), p: p}
}

type recordingTracer struct {
	trace.Tracer
	p *recordingProvider
}

func (t *recordingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	t.p.mu.Lock()
	t.p.spans = append(t.p.spans, name)
	t.p.mu.Unlock()
	return t.Tracer.Start(ctx, name, opts...)
}

// fakeBigQuery answers jobs.query (and getQueryResults) with one row.
func fakeBigQuery(t *testing.T) *httptest.Server {
	t.Helper()
	resp := map[string]any{
		"kind":         "bigquery#queryResponse",
		"jobComplete":  true,
		"jobReference": map[string]any{"projectId": "test-project", "jobId": "job1", "location": "US"},
		"schema": map[string]any{"fields": []map[string]any{
			{"name": "yyyymm", "type": "STRING"},
		}},
		"rows":      []map[string]any{{"f": []map[string]any{{"v": "202610"}}}},
		"totalRows": "1",
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/queries"),
			r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/queries/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

func TestQueryLatestMonth_EmitsTraceSpans(t *testing.T) {
	prev := otel.GetTracerProvider()
	rec := &recordingProvider{}
	otel.SetTracerProvider(rec)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	srv := fakeBigQuery(t)
	defer srv.Close()

	ctx := context.Background()
	c, err := newClient(ctx, "test-project",
		option.WithEndpoint(srv.URL),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	defer c.Close()

	got, err := c.QueryLatestMonth(ctx)
	if err != nil {
		t.Fatalf("QueryLatestMonth: %v", err)
	}
	if got != "202610" {
		t.Errorf("QueryLatestMonth = %q, want %q", got, "202610")
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.spans) == 0 {
		t.Fatal("no spans were started through the OpenTelemetry provider")
	}
	t.Logf("spans: %v", rec.spans)
}
