package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all file shares",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ListShares()
	},
}

func ListShares() error {
	accountName := "<STORAGE_ACCOUNT>"
	accountKey := "<STORAGE_KEY>"

	cred, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return err
	}
	svcURL := fmt.Sprintf("https://%s.file.core.windows.net/", accountName)
	svcClient, err := azfile.NewServiceClientWithSharedKey(svcURL, cred, nil)
	if err != nil {
		return err
	}

	pager := svcClient.NewListSharesPager(nil)
	fmt.Println("Available Shares:")
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			return err
		}
		for _, share := range resp.Segment.ShareItems {
			fmt.Printf(" - %s (quota: %d GiB)\n", *share.Name, *share.Properties.Quota)
		}
	}
	return nil
}
