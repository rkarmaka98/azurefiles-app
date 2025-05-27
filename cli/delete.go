package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a file share",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return DeleteShare(args[0])
	},
}

func DeleteShare(name string) error {
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

	shareClient := svcClient.NewShareClient(name)
	_, err = shareClient.Delete(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	fmt.Printf("🗑️ Share %q deleted\n", name)
	return nil
}
