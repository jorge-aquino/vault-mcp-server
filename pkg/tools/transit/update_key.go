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

// UpdateTransitKey returns a server.ServerTool for updating Transit key configuration.
func UpdateTransitKey(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("update_transit_key",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithDescription("Update configuration for an existing Transit key. Use this to enforce key rotation policy by setting min_decryption_version, configure automatic rotation, or update exportability."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the key to update."),
			),
			mcp.WithString("min_decryption_version",
				mcp.Description("Minimum key version that can be used to decrypt. Set this after rotating to retire old versions. E.g. '2' means version 1 ciphertext can no longer be decrypted."),
			),
			mcp.WithString("min_encryption_version",
				mcp.Description("Minimum key version that can be used to encrypt. '0' means the latest version is always used."),
			),
			mcp.WithString("auto_rotate_period",
				mcp.Description("How often to automatically rotate the key (e.g. '720h' for 30 days). Set to '0' to disable automatic rotation."),
			),
			mcp.WithBoolean("deletion_allowed",
				mcp.Description("Whether the key is allowed to be deleted. Must be set to true before calling delete_transit_key."),
			),
			mcp.WithBoolean("exportable",
				mcp.Description("Whether the raw key material may be exported. Can only be changed from false to true, never back."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return updateTransitKeyHandler(ctx, req, logger)
		},
	}
}

func updateTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	// Build the config payload from whichever optional fields were provided.
	config := map[string]interface{}{}

	if v, err := extractString(args, "min_decryption_version", false); err == nil && v != "" {
		config["min_decryption_version"] = v
	}
	if v, err := extractString(args, "min_encryption_version", false); err == nil && v != "" {
		config["min_encryption_version"] = v
	}
	if v, err := extractString(args, "auto_rotate_period", false); err == nil && v != "" {
		config["auto_rotate_period"] = v
	}
	if v, ok := args["deletion_allowed"].(bool); ok {
		config["deletion_allowed"] = v
	}
	if v, ok := args["exportable"].(bool); ok {
		config["exportable"] = v
	}

	if len(config) == 0 {
		return mcp.NewToolResultError("No configuration fields provided. Specify at least one of: min_decryption_version, min_encryption_version, auto_rotate_period, deletion_allowed, exportable."), nil
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	_, err = vault.Logical().Write(transitPath(mount, "keys", name)+"/config", config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update transit key %q: %v", name, err)), nil
	}

	logger.WithFields(log.Fields{"key": name, "config": config}).Info("Updated transit key config")
	return mcp.NewToolResultText(fmt.Sprintf("Successfully updated configuration for transit key %q in mount %q.", name, mount)), nil
}
