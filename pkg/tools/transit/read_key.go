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

// ReadTransitKey returns a server.ServerTool for reading Transit key metadata.
func ReadTransitKey(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("read_transit_key",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(true),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithDescription("Read configuration and metadata for a Transit key."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the key to read."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return readTransitKeyHandler(ctx, req, logger)
		},
	}
}

func readTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	secret, err := vault.Logical().Read(transitPath(mount, "keys", name))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read transit key: %v", err)), nil
	}

	if secret == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Transit key %q not found in mount %q.", name, mount)), nil
	}

	d := secret.Data
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Transit key: %s\n", name))
	sb.WriteString(fmt.Sprintf("Mount:       %s\n", mount))

	if v, ok := d["type"]; ok {
		sb.WriteString(fmt.Sprintf("type:                   %v\n", v))
	}
	if v, ok := d["latest_version"]; ok {
		sb.WriteString(fmt.Sprintf("latest_version:         %v\n", v))
	}
	if v, ok := d["min_decryption_version"]; ok {
		sb.WriteString(fmt.Sprintf("min_decryption_version: %v\n", v))
	}
	if v, ok := d["exportable"]; ok {
		sb.WriteString(fmt.Sprintf("exportable:             %v\n", v))
	}
	if v, ok := d["supports_encryption"]; ok {
		sb.WriteString(fmt.Sprintf("supports_encryption:    %v\n", v))
	}
	if v, ok := d["supports_decryption"]; ok {
		sb.WriteString(fmt.Sprintf("supports_decryption:    %v\n", v))
	}
	if v, ok := d["supports_derivation"]; ok {
		sb.WriteString(fmt.Sprintf("supports_derivation:    %v\n", v))
	}
	if v, ok := d["supports_signing"]; ok {
		sb.WriteString(fmt.Sprintf("supports_signing:       %v\n", v))
	}

	if keys, ok := d["keys"].(map[string]interface{}); ok && len(keys) > 0 {
		sb.WriteString("key versions:\n")
		for ver, info := range keys {
			sb.WriteString(fmt.Sprintf("  version %s: %v\n", ver, info))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
