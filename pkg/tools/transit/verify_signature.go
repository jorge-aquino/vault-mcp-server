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

// VerifySignature creates a tool for verifying a cryptographic signature against data.
func VerifySignature(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("verify_signature",
			mcp.WithDescription("Verify a cryptographic signature against data using an asymmetric Transit key."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(true),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithString("mount",
				mcp.Description("Transit mount path. Defaults to 'transit'."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the Transit key used to create the signature."),
			),
			mcp.WithString("input",
				mcp.Required(),
				mcp.Description("Original data that was signed. Raw string by default; set input_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("input_is_base64",
				mcp.Description("If true, treat input as already base64-encoded. Default false (auto-encodes)."),
			),
			mcp.WithString("signature",
				mcp.Required(),
				mcp.Description("The signature to verify (e.g. vault:v1:...)."),
			),
			mcp.WithString("hash_algorithm",
				mcp.DefaultString("sha2-256"),
				mcp.Description("Hash algorithm used during signing (e.g. sha2-256, sha2-512). Defaults to sha2-256."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return verifySignatureHandler(ctx, req, logger)
		},
	}
}

func verifySignatureHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling verify_signature request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	name, err := extractString(args, "name", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := validateKeyName(name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	input, err := extractString(args, "input", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	signature, err := extractString(args, "signature", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	inputIsBase64 := extractBool(args, "input_is_base64", false)
	if inputIsBase64 {
		if err := validateBase64(input); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	hashAlgorithm, err := extractString(args, "hash_algorithm", false)
	if err != nil || hashAlgorithm == "" {
		hashAlgorithm = "sha2-256"
	}

	mount := resolveMount(args)
	path := fmt.Sprintf("%s/%s", transitPath(mount, "verify", name), hashAlgorithm)

	b64input := input
	if !inputIsBase64 {
		b64input = base64.StdEncoding.EncodeToString([]byte(input))
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(path, map[string]interface{}{
		"input":     b64input,
		"signature": signature,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to verify signature: %v", err)), nil
	}

	if secret == nil || secret.Data == nil {
		return mcp.NewToolResultError("Empty response from Vault"), nil
	}

	valid, _ := secret.Data["valid"].(bool)
	logger.WithField("key", name).Debug("Successfully verified signature")
	return mcp.NewToolResultText(fmt.Sprintf("Signature is valid: %v", valid)), nil
}
