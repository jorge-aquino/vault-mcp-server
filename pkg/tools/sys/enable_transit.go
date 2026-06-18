// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/hashicorp/vault-mcp-server/pkg/utils"
	"github.com/hashicorp/vault/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// EnableTransit creates a tool for enabling the Vault Transit secrets engine.
func EnableTransit(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("enable_transit",
			mcp.WithDescription("Enable the Vault Transit secrets engine at a given mount path. Transit provides encryption-as-a-service: encrypt, decrypt, sign, verify, HMAC, and random byte generation without exposing key material."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithString("path",
				mcp.DefaultString("transit"),
				mcp.Description("The mount path for the Transit engine. Defaults to 'transit'."),
			),
			mcp.WithString("description",
				mcp.DefaultString(""),
				mcp.Description("Optional description for the mount."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return enableTransitHandler(ctx, req, logger)
		},
	}
}

func enableTransitHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling enable_transit request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	path, _ := args["path"].(string)
	if path == "" {
		path = "transit"
	}

	description, _ := args["description"].(string)

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	mounts, err := vault.Sys().ListMounts()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list mounts: %v", err)), nil
	}

	if _, exists := mounts[path+"/"]; exists {
		return mcp.NewToolResultError(fmt.Sprintf("A mount already exists at path '%s'. Use 'delete_mount' first if you want to re-create it.", path)), nil
	}

	err = vault.Sys().Mount(path, &api.MountInput{
		Type:        "transit",
		Description: description,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to enable Transit engine at '%s': %v", path, err)), nil
	}

	logger.WithField("path", path).Info("Enabled Transit secrets engine")
	return mcp.NewToolResultText(fmt.Sprintf("Successfully enabled Transit secrets engine at path '%s'. You can now create keys with 'create_transit_key'.", path)), nil
}
