// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/hashicorp/vault-mcp-server/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// ListTransitKeys returns a server.ServerTool for listing Transit key names.
func ListTransitKeys(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("list_transit_keys",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(true),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithDescription("List all Transit key names in the mount."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return listTransitKeysHandler(ctx, req, logger)
		},
	}
}

func listTransitKeysHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	mount := resolveMount(args)

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().List(transitPath(mount, "keys", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list transit keys: %v", err)), nil
	}

	if secret == nil || secret.Data == nil {
		return mcp.NewToolResultText("No transit keys found."), nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok || len(keys) == 0 {
		return mcp.NewToolResultText("No transit keys found."), nil
	}

	var names []string
	for _, k := range keys {
		if s, ok := k.(string); ok {
			names = append(names, s)
		}
	}
	if len(names) == 0 {
		return mcp.NewToolResultText("No transit keys found."), nil
	}

	return mcp.NewToolResultText(strings.Join(names, "\n")), nil
}
