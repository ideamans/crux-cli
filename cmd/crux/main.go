package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ideamans/crux-cli/internal/bq"
	"github.com/ideamans/crux-cli/internal/cache"
	"github.com/ideamans/crux-cli/internal/config"
	"github.com/ideamans/crux-cli/internal/crux"
	"github.com/ideamans/crux-cli/internal/cruxapi"
	"github.com/ideamans/crux-cli/internal/formatter"
	"github.com/ideamans/crux-cli/internal/runner"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crux",
	Short: "Query Chrome UX Report (CrUX) data via BigQuery or the CrUX API",
}

// ── crux device ───────────────────────────────────────────────────────────────

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Query chrome-ux-report.materialized.device_summary",
	RunE:  runDevice,
}

var (
	flagOrigins     []string
	flagDevice      string
	flagMonths      int
	flagFrom        string
	flagTo          string
	flagFormat      string
	flagMetrics     string
	flagFullMetrics bool
	flagProject     string
	flagNoCache     bool
)

func init() {
	rootCmd.AddCommand(deviceCmd)
	f := deviceCmd.Flags()
	f.StringArrayVarP(&flagOrigins, "origin", "o", nil, "Origin(s) to query (repeat or comma-separate, max 50)")
	f.StringVarP(&flagDevice, "device", "d", "all", "Device filter: phone / desktop / tablet / all")
	f.IntVarP(&flagMonths, "months", "m", 0, "Number of months to look back (default from config or 12)")
	f.StringVar(&flagFrom, "from", "", "Start month YYYYMM (overrides --months)")
	f.StringVar(&flagTo, "to", "", "End month YYYYMM (overrides latest available from CrUX)")
	f.StringVarP(&flagFormat, "format", "f", "", "Output format: table / json / csv")
	f.StringVar(&flagMetrics, "metrics", "", "Metrics to display, comma-separated (default: lcp,cls,inp)\nAvailable: lcp,cls,inp,fcp,ttfb,ol,rtt,fid")
	f.BoolVar(&flagFullMetrics, "full-metrics", false, "Show all metrics except fid (lcp,cls,inp,fcp,ttfb,ol,rtt)")
	f.StringVar(&flagProject, "project", "", "BigQuery project ID")
	f.BoolVar(&flagNoCache, "no-cache", false, "Skip cache; always query BigQuery")
	_ = deviceCmd.MarkFlagRequired("origin")
}

func runDevice(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// resolve and validate origins
	origins := formatter.SplitOrigins(flagOrigins)
	if len(origins) == 0 {
		return fmt.Errorf("at least one --origin is required")
	}
	if len(origins) > 50 {
		return fmt.Errorf("too many origins: %d (max 50)", len(origins))
	}
	origins = dedup(origins)

	// validate device flag
	device := strings.ToLower(flagDevice)
	switch device {
	case "phone", "desktop", "tablet", "all":
	default:
		return fmt.Errorf("invalid --device %q: must be phone, desktop, tablet, or all", flagDevice)
	}

	// resolve format
	format := flagFormat
	if format == "" {
		format = cfg.DefaultFormat
	}
	if format == "" {
		format = "table"
	}

	// resolve project
	projectID := cfg.ProjectIDResolved(flagProject)
	if projectID == "" {
		return fmt.Errorf("BigQuery project ID is required: use --project, CRUX_PROJECT env, or `crux auth set-project <id>`")
	}

	// resolve months
	months := flagMonths
	if months == 0 {
		months = cfg.DefaultMonths
	}
	if months == 0 {
		months = 12
	}

	var metrics []string
	switch {
	case flagMetrics != "":
		for _, m := range strings.Split(flagMetrics, ",") {
			if t := strings.TrimSpace(m); t != "" {
				metrics = append(metrics, strings.ToLower(t))
			}
		}
	case flagFullMetrics:
		metrics = []string{"lcp", "cls", "inp", "fcp", "ttfb", "ol", "rtt"}
	}

	opts := runner.DeviceOptions{
		Origins: origins,
		Device:  device,
		Format:  format,
		Metrics: metrics,
		NoCache: flagNoCache,
		Months:  months,
	}

	if flagFrom != "" {
		if err := crux.ValidateYYYYMM(flagFrom); err != nil {
			return fmt.Errorf("--from: %w", err)
		}
		opts.MonthFrom = flagFrom
	}
	if flagTo != "" {
		if err := crux.ValidateYYYYMM(flagTo); err != nil {
			return fmt.Errorf("--to: %w", err)
		}
		opts.MonthTo = flagTo
	}

	// If both --from and --to are given, months is ignored.
	// If only --to is given (or resolved later), monthFrom is computed in runner.

	bqFactory := func(ctx context.Context) (runner.BQClient, error) {
		return bq.New(ctx, projectID)
	}
	ca := cache.New(cfg.CacheDirResolved())
	opts.Progress = os.Stdout // status messages to stdout
	return runner.RunDevice(context.Background(), bqFactory, ca, opts, os.Stdout)
}

func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ── crux history / record (CrUX API) ───────────────────────────────────────────

var (
	flagAPIOrigins []string
	flagAPIUrls    []string
	flagAPIDevice  string
	flagAPIMetrics string
	flagAPIFormat  string
	flagAPIKey     string
	flagAPIPeriods int
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Query the CrUX History API (weekly time series) for an origin or url",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAPI(true)
	},
}

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Query the CrUX API for the latest record (28-day snapshot) of an origin or url",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAPI(false)
	},
}

func addAPIFlags(cmd *cobra.Command, history bool) {
	f := cmd.Flags()
	f.StringArrayVarP(&flagAPIOrigins, "origin", "o", nil, "Origin(s) to query (repeat or comma-separate)")
	f.StringArrayVarP(&flagAPIUrls, "url", "u", nil, "URL(s) to query a specific page (repeat or comma-separate)")
	f.StringVarP(&flagAPIDevice, "device", "d", "all", "Form factor: phone / desktop / tablet / all (all = aggregate)")
	f.StringVar(&flagAPIMetrics, "metrics", "", "Metrics to query, comma-separated (default: lcp,cls,inp)\nAvailable: lcp,cls,inp,fcp,ttfb,rtt")
	f.StringVarP(&flagAPIFormat, "format", "f", "", "Output format: table / json / csv")
	f.StringVar(&flagAPIKey, "api-key", "", "CrUX API key (overrides CRUX_API_KEY env and config)")
	if history {
		f.IntVar(&flagAPIPeriods, "periods", 0, "Number of weekly collection periods, 1-40 (default 25)")
	}
}

func init() {
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(recordCmd)
	addAPIFlags(historyCmd, true)
	addAPIFlags(recordCmd, false)
}

func runAPI(history bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	origins := dedup(formatter.SplitOrigins(flagAPIOrigins))
	urls := dedup(formatter.SplitOrigins(flagAPIUrls))
	if len(origins) == 0 && len(urls) == 0 {
		return fmt.Errorf("at least one --origin or --url is required")
	}

	device := strings.ToLower(flagAPIDevice)
	switch device {
	case "phone", "desktop", "tablet", "all":
	default:
		return fmt.Errorf("invalid --device %q: must be phone, desktop, tablet, or all", flagAPIDevice)
	}

	if history && flagAPIPeriods != 0 && (flagAPIPeriods < 1 || flagAPIPeriods > 40) {
		return fmt.Errorf("--periods must be between 1 and 40")
	}

	format := flagAPIFormat
	if format == "" {
		format = cfg.DefaultFormat
	}
	if format == "" {
		format = "table"
	}

	var metrics []string
	if flagAPIMetrics != "" {
		for _, m := range strings.Split(flagAPIMetrics, ",") {
			t := strings.ToLower(strings.TrimSpace(m))
			if t == "" {
				continue
			}
			if _, ok := cruxapi.MetricByKey(t); !ok {
				return fmt.Errorf("invalid metric %q: available are lcp,cls,inp,fcp,ttfb,rtt", t)
			}
			metrics = append(metrics, t)
		}
	}

	apiKey := cfg.APIKeyResolved(flagAPIKey)
	if apiKey == "" {
		return fmt.Errorf("CrUX API key is required: use --api-key, CRUX_API_KEY env, or `crux auth set-api-key <key>`")
	}

	client := cruxapi.New(apiKey)
	opts := runner.APIOptions{
		Origins:  origins,
		URLs:     urls,
		Device:   device,
		Metrics:  metrics,
		Format:   format,
		History:  history,
		Periods:  flagAPIPeriods,
		Progress: os.Stdout,
	}
	return runner.RunAPI(context.Background(), client, opts, os.Stdout)
}

// ── crux auth ─────────────────────────────────────────────────────────────────

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage BigQuery credentials and the CrUX API key",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication and project configuration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		projectID := cfg.ProjectIDResolved("")
		fmt.Printf("Config file : %s\n", config.Path())
		if projectID == "" {
			fmt.Println("Project ID  : (not set)")
		} else {
			fmt.Printf("Project ID  : %s\n", projectID)
		}
		fmt.Println("Credentials : Application Default Credentials")
		fmt.Println("             (run `gcloud auth application-default login` to configure)")
		apiKey := cfg.APIKeyResolved("")
		if apiKey == "" {
			fmt.Println("CrUX API key: (not set)")
		} else {
			fmt.Printf("CrUX API key: %s\n", maskKey(apiKey))
		}
		return nil
	},
}

// maskKey returns an API key with all but the last 4 characters hidden.
func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}

var authSetAPIKeyCmd = &cobra.Command{
	Use:   "set-api-key <api-key>",
	Short: "Set the CrUX API key in the config file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.APIKey = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("CrUX API key set to %s\n", maskKey(args[0]))
		fmt.Printf("Saved: %s\n", config.Path())
		return nil
	},
}

var authSetProjectCmd = &cobra.Command{
	Use:   "set-project <project-id>",
	Short: "Set the default BigQuery project ID in the config file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.ProjectID = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Project ID set to %q\n", args[0])
		fmt.Printf("Saved: %s\n", config.Path())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSetProjectCmd)
	authCmd.AddCommand(authSetAPIKeyCmd)
}

// ── crux cache ────────────────────────────────────────────────────────────────

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage local cache",
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print the cache directory path",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Println(cfg.CacheDirResolved())
		return nil
	},
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached entries",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ca := cache.New(cfg.CacheDirResolved())

		lm, _ := ca.GetLatestMonth()
		if lm != nil {
			fmt.Printf("Latest month : %s (checked %s)\n\n",
				crux.FormatMonth(lm.Yyyymm), lm.CheckedAt.Format("2006-01-02 15:04 UTC"))
		} else {
			fmt.Println("Latest month : (not cached)")
			fmt.Println()
		}

		files, err := ca.ListFiles()
		if err != nil {
			return err
		}
		if len(files) == 0 {
			fmt.Println("No device cache files.")
			return nil
		}
		fmt.Printf("Device cache (%d files):\n", len(files))
		for _, f := range files {
			fmt.Printf("  %s\n", f)
		}
		return nil
	},
}

var cacheClearOrigin string

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete cache files",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ca := cache.New(cfg.CacheDirResolved())

		if cacheClearOrigin != "" {
			if err := ca.ClearOrigin(cacheClearOrigin); err != nil {
				return err
			}
			fmt.Printf("Cleared cache for: %s\n", cacheClearOrigin)
		} else {
			if err := ca.ClearAll(); err != nil {
				return err
			}
			fmt.Println("All device cache cleared.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheDirCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheClearCmd.Flags().StringVarP(&cacheClearOrigin, "origin", "o", "", "Clear cache for a specific origin only")
}
