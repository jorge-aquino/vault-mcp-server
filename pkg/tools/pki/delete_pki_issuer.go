// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package pki

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/hashicorp/vault-mcp-server/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// DeletePkiIssuer creates a tool for deleting a PKI issuer.
func DeletePkiIssuer(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("delete_pki_issuer",
			mcp.WithDescription("Delete a PKI issuer (CA) from a mount. This does not revoke certificates already issued by this CA. The issuer's key material is retained in Vault unless the key itself is also deleted."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(true),
				IdempotentHint:  utils.ToBoolPtr(false),
			}),
			mcp.WithString("mount",
				mcp.DefaultString("pki"),
				mcp.Description("The PKI mount containing the issuer. Defaults to 'pki'."),
			),
			mcp.WithString("issuer_name",
				mcp.Required(),
				mcp.Description("The name of the issuer to delete, as returned by list_pki_issuers."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return deletePkiIssuerHandler(ctx, req, logger)
		},
	}
}

func deletePkiIssuerHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling delete_pki_issuer request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	mount, err := utils.ExtractMountPath(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	issuerName, ok := args["issuer_name"].(string)
	if !ok || issuerName == "" {
		return mcp.NewToolResultError("Missing or invalid 'issuer_name' parameter"), nil
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	// Resolve issuer name → issuer ID via the issuer/name endpoint.
	readPath := fmt.Sprintf("%s/issuer/%s", mount, issuerName)
	secret, err := vault.Logical().Read(readPath)
	if err != nil || secret == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Issuer '%s' not found on mount '%s'.", issuerName, mount)), nil
	}

	issuerID, ok := secret.Data["issuer_id"].(string)
	if !ok || issuerID == "" {
		return mcp.NewToolResultError(fmt.Sprintf("Could not resolve issuer ID for '%s'.", issuerName)), nil
	}

	deletePath := fmt.Sprintf("%s/issuer/%s", mount, issuerID)
	_, err = vault.Logical().Delete(deletePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete issuer '%s': %v", issuerName, err)), nil
	}

	logger.WithFields(log.Fields{
		"mount":       mount,
		"issuer_name": issuerName,
		"issuer_id":   issuerID,
	}).Info("Deleted PKI issuer")

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted PKI issuer '%s' (id: %s) from mount '%s'.", issuerName, issuerID, mount)), nil
}
