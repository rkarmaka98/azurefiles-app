package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "create [name] [quotaGB]",
	Short: "Create a file share",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		quotaInt, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid quota: %w", err)
		}
		quota := int32(quotaInt)
		return CreateShare(cmd.Context(), name, quota)
	},
}

func CreateShare(ctx context.Context, name string, quota int32) error {
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

	// CreateShare on the service client:
	_, err = svcClient.CreateShare(ctx, name, &service.CreateShareOptions{
		Quota: to.Ptr(quota),
	})
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}
	fmt.Printf("✅ Share %q created (quota: %d GiB)\n", name, quota)
	return nil
}
