package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// anomaliesCmd prints out any IOPS‐spike alerts detected by the monitor loop.
var anomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Show current IOPS anomaly alerts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current Alerts:")
		alerts.Range(func(key, value interface{}) bool {
			fmt.Printf(" - %s: %s\n", key.(string), value.(string))
			return true
		})
	},
}
