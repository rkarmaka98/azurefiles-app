package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rkarmaka98/azfiles-demo/cli/monitor"
	"github.com/spf13/cobra"
)

var (
	// thread-safe map of alerts
	alerts sync.Map
	// subscription ID and share list can be configured via flags or env
	subscriptionID string
	shareList      []string
)

var rootCmd = &cobra.Command{
	Use:   "azfilesctl",
	Short: "Azure Files demo CLI",
}

func init() {
	// global flags
	rootCmd.PersistentFlags().StringVarP(&subscriptionID, "subscription", "s", os.Getenv("AZURE_SUBSCRIPTION_ID"), "Azure Subscription ID (or set AZURE_SUBSCRIPTION_ID)")
	rootCmd.PersistentFlags().StringSliceVarP(&shareList, "shares", "l", nil, "Comma-separated list of share resource IDs in the form name:resourceID")
}

func main() {
	// register subcommands, including anomaliesCmd
	rootCmd.AddCommand(provisionCmd, listCmd, deleteCmd, anomaliesCmd)

	// validate subscription ID
	if subscriptionID == "" {
		log.Fatal("subscription ID must be provided via --subscription or AZURE_SUBSCRIPTION_ID env")
	}

	// kick off monitor loop if shares provided
	if len(shareList) > 0 {
		go runMonitorLoop()
	}

	// now run the CLI
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runMonitorLoop() {
	ctx := context.Background()
	monClient, err := monitor.NewMetricsClient(subscriptionID)
	if err != nil {
		log.Fatalf("monitor init failed: %v", err)
	}
	detector := monitor.NewZDetector(20)

	// parse shareList entries: "name:resourceID"
	type shareInfo struct{ Name, ResourceID string }
	shares := make([]shareInfo, 0, len(shareList))
	for _, entry := range shareList {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			log.Printf("invalid share entry '%s', expected name:resourceID, skipping", entry)
			continue
		}
		shares = append(shares, shareInfo{parts[0], parts[1]})
	}

	if len(shares) == 0 {
		log.Println("no shares configured for monitoring, skipping monitor loop")
		return
	}

	for {
		for _, share := range shares {
			sample, err := monClient.GetIOPS(ctx, share.ResourceID)
			if err != nil {
				fmt.Printf("❌ error fetching IOPS for %s: %v\n", share.Name, err)
				continue
			}
			if detector.Add(sample) {
				alert := fmt.Sprintf("IOPS spike: %.2f", sample)
				alerts.Store(share.Name, alert)
				fmt.Printf("🚨 %s: %s\n", share.Name, alert)
			}
		}
		// use context-aware sleep

		time.Sleep(1 * time.Minute)
	}
}
