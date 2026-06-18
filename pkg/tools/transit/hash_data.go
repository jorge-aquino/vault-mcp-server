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

// HashData creates a tool for hashing data using the Transit engine hash function.
func HashData(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("hash_data",
			mcp.WithDescription("Hash data using a Transit engine hash function (SHA-2/SHA-3 family)."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(true),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithString("mount",
				mcp.Description("Transit mount path. Defaults to 'transit'."),
			),
			mcp.WithString("input",
				mcp.Required(),
				mcp.Description("Data to hash. Raw string by default; set input_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("input_is_base64",
				mcp.Description("If true, treat input as already base64-encoded. Default false (auto-encodes)."),
			),
			mcp.WithString("algorithm",
				mcp.DefaultString("sha2-256"),
				mcp.Description("Hash algorithm (e.g. sha2-256, sha2-512, sha3-256). Defaults to sha2-256."),
			),
			mcp.WithString("format",
				mcp.DefaultString("hex"),
				mcp.Description("Output format: 'hex' or 'base64'. Defaults to hex."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return hashDataHandler(ctx, req, logger)
		},
	}
}

func hashDataHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling hash_data request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
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

	format, err := extractString(args, "format", false)
	if err != nil || format == "" {
		format = "hex"
	}

	mount := resolveMount(args)
	path := fmt.Sprintf("%s/%s", transitPath(mount, "hash", ""), algorithm)

	b64input := input
	if !inputIsBase64 {
		b64input = base64.StdEncoding.EncodeToString([]byte(input))
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(path, map[string]interface{}{
		"input":  b64input,
		"format": format,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to hash data: %v", err)), nil
	}

	sum, err := dataString(secret, "sum")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Unexpected Vault response: %v", err)), nil
	}

	logger.Debug("Successfully hashed data")
	return mcp.NewToolResultText(fmt.Sprintf("Hash (%s): %s", algorithm, sum)), nil
}
