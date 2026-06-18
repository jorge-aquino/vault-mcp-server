// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/hashicorp/vault-mcp-server/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// EncryptData returns a ServerTool that encrypts data with a named Transit key.
func EncryptData(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("encrypt_data",
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithDescription("Encrypt data using a named Transit key. Accepts raw text (auto-base64-encoded) or pre-encoded base64 input."),
			mcp.WithString("mount",
				mcp.Description("Transit mount path (default: transit)."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the Transit encryption key."),
			),
			mcp.WithString("plaintext",
				mcp.Required(),
				mcp.Description("The data to encrypt. Raw text by default; set plaintext_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("plaintext_is_base64",
				mcp.Description("Set to true if plaintext is already base64-encoded. Defaults to false."),
			),
			mcp.WithString("context",
				mcp.Description("Base64-encoded context for derived keys."),
			),
			mcp.WithNumber("key_version",
				mcp.Description("Specific key version to use for encryption."),
			),
			mcp.WithString("nonce",
				mcp.Description("Base64-encoded nonce for convergent encryption."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return encryptDataHandler(ctx, req, logger)
		},
	}
}

func encryptDataHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
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

	plaintext, err := extractString(args, "plaintext", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	plaintextIsBase64 := extractBool(args, "plaintext_is_base64", false)
	if plaintextIsBase64 {
		if err := validateBase64(plaintext); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	} else {
		plaintext = base64.StdEncoding.EncodeToString([]byte(plaintext))
	}

	payload := map[string]interface{}{
		"plaintext": plaintext,
	}

	if ctxVal, _ := extractString(args, "context", false); ctxVal != "" {
		payload["context"] = ctxVal
	}
	if keyVersion := extractInt(args, "key_version", 0); keyVersion > 0 {
		payload["key_version"] = keyVersion
	}
	if nonce, _ := extractString(args, "nonce", false); nonce != "" {
		payload["nonce"] = nonce
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(transitPath(mount, "encrypt", name), payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encrypt data: %v", err)), nil
	}

	ciphertext, err := dataString(secret, "ciphertext")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ciphertext: %s", ciphertext)), nil
}
