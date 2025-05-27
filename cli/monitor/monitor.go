package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/armmonitor"
)

type MetricsClient struct {
	client *armmonitor.MetricsClient
}

// NewMetricsClient constructs a MetricsClient using DefaultAzureCredential.
func NewMetricsClient(subscriptionID string) (*MetricsClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	cli, err := armmonitor.NewMetricsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}
	return &MetricsClient{client: cli}, nil
}

// GetIOPS fetches the latest FileServerIOPS metric point for the given resourceID.
func (m *MetricsClient) GetIOPS(ctx context.Context, resourceID string) (float64, error) {
	// Use a very short timespan to get the most recent point
	now := time.Now().UTC()
	ts := fmt.Sprintf("%s/%s", now.Add(-5*time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
	resp, err := m.client.List(ctx, resourceID, &armmonitor.MetricsClientListOptions{
		Timespan:    &ts,
		Metricnames: to.Ptr("FileServerIOPS"),
		Interval:    nil, // default
		Aggregation: to.Ptr("Average"),
	})
	if err != nil {
		return 0, fmt.Errorf("metrics query failed: %w", err)
	}
	// Iterate to find the latest timestamped value
	for _, metric := range resp.Value {
		for _, ts := range *metric.Timeseries {
			for _, point := range *ts.Data {
				if point.Average != nil {
					return *point.Average, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("no data points for FileServerIOPS")
}
