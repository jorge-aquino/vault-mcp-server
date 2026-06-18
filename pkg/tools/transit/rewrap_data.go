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

// RewrapData returns a ServerTool that re-encrypts ciphertext with the latest Transit key version.
func RewrapData(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("rewrap_data",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithDescription("Re-encrypt ciphertext with the Transit key's latest version without exposing plaintext. Use this after rotating a key to upgrade old ciphertext."),
			mcp.WithString("mount",
				mcp.Description("Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the Transit encryption key."),
			),
			mcp.WithString("ciphertext",
				mcp.Required(),
				mcp.Description("The ciphertext to rewrap. Must start with 'vault:v'."),
			),
			mcp.WithString("context",
				mcp.Description("Base64-encoded context for derived keys."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return rewrapDataHandler(ctx, req, logger)
		},
	}
}

func rewrapDataHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	ciphertext, err := extractString(args, "ciphertext", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := validateCiphertext(ciphertext); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	payload := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	if ctxVal, _ := extractString(args, "context", false); ctxVal != "" {
		payload["context"] = ctxVal
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(transitPath(mount, "rewrap", name), payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to rewrap data: %v", err)), nil
	}

	newCiphertext, err := dataString(secret, "ciphertext")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Rewrapped ciphertext: %s", newCiphertext)), nil
}
