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

// RevokePkiCertificate creates a tool for revoking a PKI certificate by serial number.
func RevokePkiCertificate(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("revoke_pki_certificate",
			mcp.WithDescription("Revoke a PKI certificate by its serial number. The certificate will be added to the CRL and can no longer be considered valid. The serial number is returned when a certificate is issued via issue_pki_certificate."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint:    utils.ToBoolPtr(false),
				DestructiveHint: utils.ToBoolPtr(true),
				IdempotentHint:  utils.ToBoolPtr(true),
			}),
			mcp.WithString("mount",
				mcp.DefaultString("pki"),
				mcp.Description("The PKI mount where the certificate was issued. Defaults to 'pki'."),
			),
			mcp.WithString("serial_number",
				mcp.Required(),
				mcp.Description("The serial number of the certificate to revoke, as returned by issue_pki_certificate (e.g. '40:1d:4a:bb:18:03:64:a9:00:c3:64:43:12:1f:9a:41:62:6d:9e:05')."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return revokePkiCertificateHandler(ctx, req, logger)
		},
	}
}

func revokePkiCertificateHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	logger.Debug("Handling revoke_pki_certificate request")

	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Missing or invalid arguments format"), nil
	}

	mount, err := utils.ExtractMountPath(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	serialNumber, ok := args["serial_number"].(string)
	if !ok || serialNumber == "" {
		return mcp.NewToolResultError("Missing or invalid 'serial_number' parameter"), nil
	}

	vault, err := client.GetVaultClientFromContext(ctx, logger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
	}

	fullPath := fmt.Sprintf("%s/revoke", mount)
	secret, err := vault.Logical().Write(fullPath, map[string]interface{}{
		"serial_number": serialNumber,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to revoke certificate '%s': %v", serialNumber, err)), nil
	}

	var revokedAt interface{}
	if secret != nil && secret.Data != nil {
		revokedAt = secret.Data["revocation_time_rfc3339"]
	}

	logger.WithFields(log.Fields{
		"mount":         mount,
		"serial_number": serialNumber,
	}).Info("Revoked PKI certificate")

	msg := fmt.Sprintf("Successfully revoked certificate with serial number '%s' on mount '%s'.", serialNumber, mount)
	if revokedAt != nil {
		msg += fmt.Sprintf(" Revoked at: %v.", revokedAt)
	}
	return mcp.NewToolResultText(msg), nil
}
