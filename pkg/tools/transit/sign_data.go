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

// SignData creates a tool for generating a cryptographic signature for data using an asymmetric Transit key.
func SignData(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("sign_data",
			mcp.WithDescription("Generate a cryptographic signature for data using an asymmetric Transit key (ed25519, rsa-*, ecdsa-*)."),
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
				mcp.Description("Name of the asymmetric Transit key to sign with (e.g. ed25519, rsa-2048, ecdsa-p256)."),
			),
			mcp.WithString("input",
				mcp.Required(),
				mcp.Description("Data to sign. Raw string by default; set input_is_base64=true if already base64-encoded."),
			),
			mcp.WithBoolean("input_is_base64",
				mcp.Description("If true, treat input as already base64-encoded. Default false (auto-encodes)."),
			),
			mcp.WithString("hash_algorithm",
				mcp.DefaultString("sha2-256"),
				mcp.Description("Hash algorithm for signing (e.g. sha2-256, sha2-512). Defaults to sha2-256."),
			),
			mcp.WithString("signature_algorithm",
				mcp.DefaultString("pss"),
				mcp.Description("Signature algorithm for RSA keys (e.g. pss, pkcs1v15). Defaults to pss."),
			),
			mcp.WithNumber("key_version",
				mcp.Description("Key version to use for signing. Optional."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return signDataHandler(ctx, req, logger)
		},
	}
}

func signDataHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling sign_data request")

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

	hashAlgorithm, err := extractString(args, "hash_algorithm", false)
	if err != nil || hashAlgorithm == "" {
		hashAlgorithm = "sha2-256"
	}

	sigAlgorithm, err := extractString(args, "signature_algorithm", false)
	if err != nil || sigAlgorithm == "" {
		sigAlgorithm = "pss"
	}

	mount := resolveMount(args)
	path := fmt.Sprintf("%s/%s", transitPath(mount, "sign", name), hashAlgorithm)

	b64input := input
	if !inputIsBase64 {
		b64input = base64.StdEncoding.EncodeToString([]byte(input))
	}

	data := map[string]interface{}{
		"input":               b64input,
		"signature_algorithm": sigAlgorithm,
	}
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
		return mcp.NewToolResultError(fmt.Sprintf("Failed to sign data: %v", err)), nil
	}

	signature, err := dataString(secret, "signature")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Unexpected Vault response: %v", err)), nil
	}

	logger.WithField("key", name).Debug("Successfully signed data")
	return mcp.NewToolResultText(fmt.Sprintf("Signature: %s", signature)), nil
}
