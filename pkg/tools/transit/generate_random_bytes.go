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

// GenerateRandomBytes creates a tool for generating cryptographically secure random bytes from the Vault Transit engine.
func GenerateRandomBytes(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("generate_random_bytes",
			mcp.WithDescription("Generate cryptographically secure random bytes from the Vault Transit engine."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(true),
				DestructiveHint: utils.ToBoolPtr(false),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithString("mount",
				mcp.Description("Transit mount path. Defaults to 'transit'."),
			),
			mcp.WithNumber("bytes",
				mcp.Description("Number of random bytes to generate. Defaults to 32."),
			),
			mcp.WithString("format",
				mcp.DefaultString("base64"),
				mcp.Description("Output format: 'base64' or 'hex'. Defaults to base64."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return generateRandomBytesHandler(ctx, req, logger)
		},
	}
}

func generateRandomBytesHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling generate_random_bytes request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		// No arguments provided — use defaults.
		args = map[string]interface{}{}
	}

	numBytes := extractInt(args, "bytes", 32)
	if numBytes <= 0 {
		numBytes = 32
	}

	format, err := extractString(args, "format", false)
	if err != nil || format == "" {
		format = "base64"
	}

	mount := resolveMount(args)
	path := fmt.Sprintf("%s/%d", transitPath(mount, "random", ""), numBytes)

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	secret, err := vault.Logical().Write(path, map[string]interface{}{
		"format": format,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate random bytes: %v", err)), nil
	}

	randomBytes, err := dataString(secret, "random_bytes")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Unexpected Vault response: %v", err)), nil
	}

	logger.WithField("bytes", numBytes).Debug("Successfully generated random bytes")
	return mcp.NewToolResultText(fmt.Sprintf("Random bytes (%s): %s", format, randomBytes)), nil
}
