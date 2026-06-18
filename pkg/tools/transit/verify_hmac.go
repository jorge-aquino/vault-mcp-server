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

// VerifyHMAC creates a tool for verifying an HMAC against the original input.
func VerifyHMAC(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("verify_hmac",
			mcp.WithDescription("Verify an HMAC against the original input. Returns whether the HMAC is valid."),
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
				mcp.Description("Name of the Transit key used to generate the HMAC."),
			),
			mcp.WithString("input",
				mcp.Required(),
				mcp.Description("Original data to verify against. Raw string by default; set input_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("input_is_base64",
				mcp.Description("If true, treat input as already base64-encoded. Default false (auto-encodes)."),
			),
			mcp.WithString("hmac",
				mcp.Required(),
				mcp.Description("The HMAC value to verify (e.g. vault:v1:...)."),
			),
			mcp.WithString("algorithm",
				mcp.DefaultString("sha2-256"),
				mcp.Description("HMAC algorithm (e.g. sha2-256, sha2-512). Defaults to sha2-256."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return verifyHMACHandler(ctx, req, logger)
		},
	}
}

func verifyHMACHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling verify_hmac request")

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

	hmacVal, err := extractString(args, "hmac", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	inputIsBase64 := extractBool(args, "input_is_base64", false)
	if inputIsBase64 {
		if err := validateBase64(input); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	algorithm, err := extractString(args, "algorithm", false)
	if err != nil || algorithm == "" {
		algorithm = "sha2-256"
	}

	mount := resolveMount(args)
	path := fmt.Sprintf("%s/%s", transitPath(mount, "verify", name), algorithm)

	b64input := input
	if !inputIsBase64 {
		b64input = base64.StdEncoding.EncodeToString([]byte(input))
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(path, map[string]interface{}{
		"input": b64input,
		"hmac":  hmacVal,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to verify HMAC: %v", err)), nil
	}

	if secret == nil || secret.Data == nil {
		return mcp.NewToolResultError("Empty response from Vault"), nil
	}

	valid, _ := secret.Data["valid"].(bool)
	logger.WithField("key", name).Debug("Successfully verified HMAC")
	return mcp.NewToolResultText(fmt.Sprintf("HMAC is valid: %v", valid)), nil
}
