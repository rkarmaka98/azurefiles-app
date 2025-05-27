package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "create [name] [quotaGB]",
	Short: "Create a file share",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		quota, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid quota: %w", err)
		}
		return CreateShare(name, quota)
	},
}

// CreateShare calls Azure Files to provision a share with the given quota.
func CreateShare(name string, quotaGB int64) error {
	// TODO: replace these with your values or pull from env
	accountName := "<STORAGE_ACCOUNT>"
	accountKey := "<STORAGE_KEY>"

	// build a ServiceURL
	cred, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return err
	}
	svcURL := fmt.Sprintf("https://%s.file.core.windows.net/", accountName)
	svcClient, err := azfile.NewServiceClientWithSharedKey(svcURL, cred, nil)
	if err != nil {
		return err
	}

	// get a reference to the share
	shareClient := svcClient.NewShareClient(name)
	_, err = shareClient.Create(context.Background(), &azfile.CreateShareOptions{
		ShareQuota: &quotaGB,
	})
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}
	fmt.Printf("✅ Share %q created (quota: %d GiB)\n", name, quotaGB)
	return nil
}
