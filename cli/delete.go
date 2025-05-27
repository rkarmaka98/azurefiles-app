package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a file share",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return DeleteShare(cmd.Context(), args[0])
	},
}

func DeleteShare(ctx context.Context, name string) error {
	acct := os.Getenv("AZURE_STORAGE_ACCOUNT")
	key := os.Getenv("AZURE_STORAGE_KEY")
	if acct == "" || key == "" {
		return fmt.Errorf("AZURE_STORAGE_ACCOUNT & AZURE_STORAGE_KEY must be set")
	}

	svcURL := fmt.Sprintf("https://%s.file.core.windows.net/", acct)
	cred, err := service.NewSharedKeyCredential(acct, key)
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	svcClient, err := service.NewClientWithSharedKeyCredential(svcURL, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create service client: %w", err)
	}

	_, err = svcClient.DeleteShare(ctx, name, nil)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	fmt.Printf("🗑️ Share %q deleted\n", name)
	return nil
}
