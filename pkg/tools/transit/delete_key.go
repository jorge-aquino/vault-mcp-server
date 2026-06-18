// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/hashicorp/vault-mcp-server/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// DeleteTransitKey returns a server.ServerTool for deleting a Transit key.
func DeleteTransitKey(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("delete_transit_key",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(true),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithDescription("Delete a named Transit key. The key must not be in use for any active encryption operations. This action is irreversible."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the key to delete."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return deleteTransitKeyHandler(ctx, req, logger)
		},
	}
}

func deleteTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	mount := resolveMount(args)

	name, err := extractString(args, "name", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := validateKeyName(name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	// Vault requires deletion_allowed=true before a key can be deleted.
	_, err = vault.Logical().Write(transitPath(mount, "keys", name)+"/config", map[string]interface{}{
		"deletion_allowed": true,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to enable deletion for key %q: %v", name, err)), nil
	}

	_, err = vault.Logical().Delete(transitPath(mount, "keys", name))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete transit key %q: %v", name, err)), nil
	}

	logger.WithField("key", name).Info("Deleted transit key")
	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted transit key %q from mount %q.", name, mount)), nil
}
