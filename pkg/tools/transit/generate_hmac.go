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

// GenerateHMAC creates a tool for generating an HMAC for data using a Transit key.
func GenerateHMAC(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("generate_hmac",
			mcp.WithDescription("Generate an HMAC for data using a Transit key. Use for data-integrity verification."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithString("mount",
				mcp.Description("Transit mount path. Defaults to 'transit'."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the Transit key to use for HMAC generation."),
			),
			mcp.WithString("input",
				mcp.Required(),
				mcp.Description("Data to HMAC. Raw string by default; set input_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("input_is_base64",
				mcp.Description("If true, treat input as already base64-encoded. Default false (auto-encodes)."),
			),
			mcp.WithString("algorithm",
				mcp.DefaultString("sha2-256"),
				mcp.Description("HMAC algorithm (e.g. sha2-256, sha2-512). Defaults to sha2-256."),
			),
			mcp.WithNumber("key_version",
				mcp.Description("Key version to use for HMAC generation. Optional."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return generateHMACHandler(ctx, req, logger)
		},
	}
}

func generateHMACHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling generate_hmac request")

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
	path := fmt.Sprintf("%s/%s", transitPath(mount, "hmac", name), algorithm)

	b64input := input
	if !inputIsBase64 {
		b64input = base64.StdEncoding.EncodeToString([]byte(input))
	}

	data := map[string]interface{}{"input": b64input}
	keyVersion := extractInt(args, "key_version", 0)
	if keyVersion > 0 {
		data["key_version"] = keyVersion
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(path, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate HMAC: %v", err)), nil
	}

	hmacVal, err := dataString(secret, "hmac")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Unexpected Vault response: %v", err)), nil
	}

	logger.WithField("key", name).Debug("Successfully generated HMAC")
	return mcp.NewToolResultText(fmt.Sprintf("HMAC: %s", hmacVal)), nil
}
