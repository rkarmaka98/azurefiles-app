package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rkarmaka98/azurefiles-app/cli/monitor"
	"github.com/spf13/cobra"
)

// ShareInfo is the JSON shape for a file share, now with four metrics.
type ShareInfo struct {
	Name         string  `json:"name"`
	QuotaGB      int32   `json:"quotaGB"`
	IOPS         float64 `json:"iops"`
	BandwidthMiB float64 `json:"bandwidthMiB"`
	LatencyMs    float64 `json:"latencyMs"`
	Transactions float64 `json:"transactions"`
}

// shareEntry holds the parsed --shares entries.
type shareEntry struct {
	Name          string
	FileServiceID string
}

var (
	alerts         sync.Map
	subscriptionID string
	shareList      []string
)

var rootCmd = &cobra.Command{
	Use:   "azfilesctl",
	Short: "Azure Files demo CLI",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP JSON API and anomaly monitor",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		go runMonitorLoop(ctx)
		serveAPI(ctx)
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&subscriptionID, "subscription", "s", os.Getenv("AZURE_SUBSCRIPTION_ID"),
			"Azure Subscription ID (or set AZURE_SUBSCRIPTION_ID)")
	rootCmd.PersistentFlags().
		StringSliceVarP(&shareList, "shares", "l", nil,
			"Comma-separated list of name:fileServiceResourceID")

	rootCmd.AddCommand(provisionCmd, listCmd, deleteCmd, anomaliesCmd, serveCmd)
}

func main() {
	if subscriptionID == "" {
		log.Fatal("subscription ID required via --subscription or AZURE_SUBSCRIPTION_ID")
	}
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// runMonitorLoop fetches four metrics per share, detects anomalies, and populates alerts.
func runMonitorLoop(ctx context.Context) {
	// Parse the shareList into shareEntry structs once.
	entries := make([]shareEntry, 0, len(shareList))
	for _, e := range shareList {
		parts := strings.SplitN(e, ":", 2)
		if len(parts) == 2 {
			entries = append(entries, shareEntry{parts[0], parts[1]})
		} else {
			log.Printf("skipping invalid share entry: %q", e)
		}
	}
	if len(entries) == 0 {
		log.Println("no valid shares to monitor; exiting monitor loop")
		return
	}

	// Initialize the Monitor client and detector.
	monClient, err := monitor.NewMetricsClient(subscriptionID)
	if err != nil {
		log.Fatalf("failed to create metrics client: %v", err)
	}
	detector := monitor.NewZDetector(20)

	// Metric definitions: metricName → label for alerts.
	metrics := []struct{ Key, Label string }{
		{"FileShareMaxUsedIOPS", "IOPS"},
		{"FileShareMaxUsedBandwidthMiBps", "Bandwidth MiB/s"},
		{"SuccessE2ELatency", "Latency ms"},
		{"Transactions", "Transactions/s"},
	}

	// Poll loop
	for {
		for _, se := range entries {
			for _, m := range metrics {
				value, err := monClient.GetMetric(ctx, se.FileServiceID, m.Key, se.Name)
				if err != nil {
					log.Printf("❌ %s fetch failed for %s: %v", m.Key, se.Name, err)
					continue
				}
				if detector.Add(value) {
					alert := fmt.Sprintf("%s spike (%s): %.2f", se.Name, m.Label, value)
					alerts.Store(fmt.Sprintf("%s-%s", se.Name, m.Key), alert)
					log.Println("🚨", alert)
				}
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

func serveAPI(ctx context.Context) {
	// CORS wrapper
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h(w, r)
		}
	}

	http.HandleFunc("/api/shares", wrap(func(w http.ResponseWriter, r *http.Request) {
		shares, err := fetchShares(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(shares)
	}))
	http.HandleFunc("/api/anomalies", wrap(func(w http.ResponseWriter, r *http.Request) {
		m := make(map[string]string)
		alerts.Range(func(k, v interface{}) bool {
			m[k.(string)] = v.(string)
			return true
		})
		json.NewEncoder(w).Encode(m)
	}))

	log.Println("📡 JSON API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// fetchShares returns each share’s quota plus the latest four metrics.
func fetchShares(ctx context.Context) ([]ShareInfo, error) {
	// 1) Retrieve base quotas via ListShares (from list.go)
	base, err := ListShares(ctx)
	if err != nil {
		return nil, err
	}
	quotaMap := make(map[string]int32, len(base))
	for _, s := range base {
		quotaMap[s.Name] = s.QuotaGB
	}

	// 2) Parse the same shareList entries
	entries := make([]shareEntry, 0, len(shareList))
	for _, e := range shareList {
		parts := strings.SplitN(e, ":", 2)
		if len(parts) == 2 {
			entries = append(entries, shareEntry{parts[0], parts[1]})
		}
	}

	// 3) Metrics client
	monClient, err := monitor.NewMetricsClient(subscriptionID)
	if err != nil {
		return nil, err
	}

	// 4) Build enriched ShareInfo slice
	metrics := []struct{ Key, Label string }{
		{"FileShareMaxUsedIOPS", "IOPS"},
		{"FileShareMaxUsedBandwidthMiBps", "BandwidthMiB"},
		{"SuccessE2ELatency", "LatencyMs"},
		{"Transactions", "Transactions"},
	}
	out := make([]ShareInfo, 0, len(entries))
	for _, se := range entries {
		si := ShareInfo{
			Name:    se.Name,
			QuotaGB: quotaMap[se.Name],
		}
		// fetch each metric
		si.IOPS, _ = monClient.GetMetric(ctx, se.FileServiceID, metrics[0].Key, se.Name)
		si.BandwidthMiB, _ = monClient.GetMetric(ctx, se.FileServiceID, metrics[1].Key, se.Name)
		si.LatencyMs, _ = monClient.GetMetric(ctx, se.FileServiceID, metrics[2].Key, se.Name)
		si.Transactions, _ = monClient.GetMetric(ctx, se.FileServiceID, metrics[3].Key, se.Name)
		out = append(out, si)
	}
	return out, nil
}
