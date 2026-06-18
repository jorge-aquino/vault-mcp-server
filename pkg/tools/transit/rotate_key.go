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

// RotateTransitKey returns a server.ServerTool for rotating a Transit key.
func RotateTransitKey(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("rotate_transit_key",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithDescription("Rotate a Transit key to a new version. Old ciphertext remains decryptable with the previous version."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the key to rotate."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return rotateTransitKeyHandler(ctx, req, logger)
		},
	}
}

func rotateTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	rotatePath := fmt.Sprintf("%s/rotate", transitPath(mount, "keys", name))
	_, err = vault.Logical().Write(rotatePath, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to rotate transit key: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Successfully rotated transit key %q in mount %q. A new key version is now active. "+
			"Use 'rewrap_data' to upgrade old ciphertext to the latest key version.",
		name, mount,
	)), nil
}
