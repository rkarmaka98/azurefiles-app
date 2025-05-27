package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var anomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Show current anomaly alerts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current Anomaly Alerts:")
		alerts.Range(func(key, value interface{}) bool {
			fmt.Printf(" - %s: %s\n", key.(string), value.(string))
			return true
		})
	},
}
