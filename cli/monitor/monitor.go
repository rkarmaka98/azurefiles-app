package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
)

// MetricsClient wraps the ARM Monitor client.
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

// GetMetric fetches the most recent 'Average' of metricName for a given fileServiceResourceID,
// filtered by the FileShare dimension (the shareName).
func (m *MetricsClient) GetMetric(ctx context.Context, fileServiceResourceID, metricName, shareName string) (float64, error) {
	// 5-minute window
	now := time.Now().UTC()
	timespan := fmt.Sprintf(
		"%s/%s",
		now.Add(-5*time.Minute).Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	// Use FileShare dimension
	filter := fmt.Sprintf("FileShare eq '%s'", shareName)

	resp, err := m.client.List(ctx, fileServiceResourceID, &armmonitor.MetricsClientListOptions{
		Timespan:    &timespan,
		Metricnames: to.Ptr(metricName),
		Aggregation: to.Ptr("Average"),
		Filter:      &filter,
	})
	if err != nil {
		return 0, fmt.Errorf("metrics query failed: %w", err)
	}
	// Find the first average data point
	for _, metric := range resp.Value {
		for _, series := range metric.Timeseries {
			for _, point := range series.Data {
				if point.Average != nil {
					return *point.Average, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("no data points for metric %s on share %s", metricName, shareName)
}
