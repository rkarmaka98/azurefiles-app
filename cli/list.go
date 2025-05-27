package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/spf13/cobra"
)

// listCmd remains the same, but now prints from returned data
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all file shares",
	RunE: func(cmd *cobra.Command, args []string) error {
		shares, err := ListShares(cmd.Context())
		if err != nil {
			return err
		}
		for _, s := range shares {
			fmt.Printf(" - %s (%d GiB)\n", s.Name, s.QuotaGB)
		}
		return nil
	},
}

// ListShares returns a slice of ShareInfo for both CLI and JSON API
func ListShares(ctx context.Context) ([]ShareInfo, error) {
	acct := os.Getenv("AZURE_STORAGE_ACCOUNT")
	key := os.Getenv("AZURE_STORAGE_KEY")
	if acct == "" || key == "" {
		return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT & AZURE_STORAGE_KEY must be set")
	}

	svcURL := fmt.Sprintf("https://%s.file.core.windows.net/", acct)
	cred, err := service.NewSharedKeyCredential(acct, key)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}
	svcClient, err := service.NewClientWithSharedKeyCredential(svcURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}

	pager := svcClient.NewListSharesPager(nil)
	var out []ShareInfo
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range page.Shares {
			out = append(out, ShareInfo{
				Name:    *s.Name,
				QuotaGB: *s.Properties.Quota,
			})
		}
	}
	return out, nil
}
