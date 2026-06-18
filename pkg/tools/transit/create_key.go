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

var validKeyTypes = map[string]struct{}{
	"aes256-gcm96":      {},
	"aes128-gcm96":      {},
	"chacha20-poly1305": {},
	"ed25519":           {},
	"ecdsa-p256":        {},
	"ecdsa-p384":        {},
	"ecdsa-p521":        {},
	"rsa-2048":          {},
	"rsa-3072":          {},
	"rsa-4096":          {},
	"hmac":              {},
}

// CreateTransitKey returns a server.ServerTool for creating Transit keys.
func CreateTransitKey(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("create_transit_key",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithDescription("Create a new named encryption key in the Vault Transit engine."),
			mcp.WithString("mount",
				mcp.Description("The Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the key to create."),
			),
			mcp.WithString("type",
				mcp.DefaultString("aes256-gcm96"),
				mcp.Description("Key type (e.g. aes256-gcm96, rsa-2048, ed25519). Default: aes256-gcm96."),
			),
			mcp.WithBoolean("exportable",
				mcp.DefaultBool(false),
				mcp.Description("Whether the key is exportable. Default: false."),
			),
			mcp.WithBoolean("allow_plaintext_backup",
				mcp.DefaultBool(false),
				mcp.Description("Whether plaintext backup is allowed. Default: false."),
			),
			mcp.WithBoolean("derived",
				mcp.DefaultBool(false),
				mcp.Description("Whether key derivation is enabled. Default: false."),
			),
			mcp.WithString("auto_rotate_period",
				mcp.Description("Duration string for automatic rotation (e.g. 720h). Leave empty to disable."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return createTransitKeyHandler(ctx, req, logger)
		},
	}
}

func createTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	keyType, err := extractString(args, "type", false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if keyType == "" {
		keyType = "aes256-gcm96"
	}
	if _, ok := validKeyTypes[keyType]; !ok {
		return mcp.NewToolResultError(fmt.Sprintf("unsupported key type %q; valid types: aes256-gcm96, aes128-gcm96, chacha20-poly1305, ed25519, ecdsa-p256, ecdsa-p384, ecdsa-p521, rsa-2048, rsa-3072, rsa-4096, hmac", keyType)), nil
	}

	exportable := extractBool(args, "exportable", false)
	allowPlaintextBackup := extractBool(args, "allow_plaintext_backup", false)
	derived := extractBool(args, "derived", false)

	autoRotatePeriod, err := extractString(args, "auto_rotate_period", false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	payload := map[string]interface{}{
		"type":                   keyType,
		"exportable":             exportable,
		"allow_plaintext_backup": allowPlaintextBackup,
		"derived":                derived,
	}
	if autoRotatePeriod != "" {
		payload["auto_rotate_period"] = autoRotatePeriod
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	_, err = vault.Logical().Write(transitPath(mount, "keys", name), payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create transit key: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created transit key %q of type %q in mount %q.", name, keyType, mount)), nil
}
