package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

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
		go runMonitorLoop()
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

func runMonitorLoop() {
	// unchanged...
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

// fetchShares now combines ListShares + per-share metrics.
func fetchShares(ctx context.Context) ([]ShareInfo, error) {
	// 1) get base quotas
	base, err := ListShares(ctx)
	if err != nil {
		return nil, err
	}
	quotaMap := make(map[string]int32, len(base))
	for _, s := range base {
		quotaMap[s.Name] = s.QuotaGB
	}

	// 2) parse shareList entries (name:fileServiceID)
	entries := make([]shareEntry, 0, len(shareList))
	for _, e := range shareList {
		p := strings.SplitN(e, ":", 2)
		if len(p) == 2 {
			entries = append(entries, shareEntry{p[0], p[1]})
		}
	}

	// 3) metrics client
	monClient, err := monitor.NewMetricsClient(subscriptionID)
	if err != nil {
		return nil, err
	}

	// 4) assemble enriched ShareInfo
	var out []ShareInfo
	for _, se := range entries {
		si := ShareInfo{
			Name:    se.Name,
			QuotaGB: quotaMap[se.Name],
		}
		// fetch each metric
		si.IOPS, _ = monClient.GetMetric(ctx, se.FileServiceID, "FileShareMaxUsedIOPS", se.Name)
		si.BandwidthMiB, _ = monClient.GetMetric(ctx, se.FileServiceID, "FileShareMaxUsedBandwidthMiBps", se.Name)
		si.LatencyMs, _ = monClient.GetMetric(ctx, se.FileServiceID, "SuccessE2ELatency", se.Name)
		si.Transactions, _ = monClient.GetMetric(ctx, se.FileServiceID, "Transactions", se.Name)
		out = append(out, si)
	}
	return out, nil
}
