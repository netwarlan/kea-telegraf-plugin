package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"kea-telegraf-plugin/internal/kea"
	"kea-telegraf-plugin/internal/lineprotocol"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type config struct {
	URL     string
	Server  string
	Timeout time.Duration
	JSON    bool
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "keastats",
	Short: "Query Kea DHCP4 stats and output InfluxDB line protocol",
	RunE:  runQuery,
}

func init() {
	hostname, _ := os.Hostname()

	rootCmd.Flags().StringVarP(&cfg.URL, "url", "u", "http://localhost:8000/", "Kea Control Agent URL")
	rootCmd.Flags().StringVarP(&cfg.Server, "server", "s", hostname, "Server tag for line protocol output")
	rootCmd.Flags().DurationVarP(&cfg.Timeout, "timeout", "t", 5*time.Second, "HTTP request timeout")
	rootCmd.Flags().BoolVarP(&cfg.JSON, "json", "j", false, "Output raw Kea API JSON (debug mode)")

	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runQuery(cmd *cobra.Command, args []string) error {
	if err := validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(10)
	}

	client := kea.NewClient(cfg.URL, cfg.Timeout)

	// JSON debug mode — print raw response and exit
	if cfg.JSON {
		raw, err := client.GetRawJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		var pretty json.RawMessage
		if err := json.Unmarshal(raw, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Println(string(formatted))
		} else {
			fmt.Println(string(raw))
		}
		return nil
	}

	stats, err := client.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output := lineprotocol.Format(stats, cfg.Server)
	if output != "" {
		fmt.Println(output)
	}

	return nil
}

func validate() error {
	if cfg.URL == "" {
		return fmt.Errorf("--url is required")
	}
	if cfg.Server == "" {
		return fmt.Errorf("--server is required")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	return nil
}
