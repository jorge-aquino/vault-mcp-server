// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"github.com/hashicorp/vault-mcp-server/pkg/tools/kv"
	"github.com/hashicorp/vault-mcp-server/pkg/tools/pki"
	"github.com/hashicorp/vault-mcp-server/pkg/tools/sys"
	"github.com/hashicorp/vault-mcp-server/pkg/tools/transit"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

func InitTools(hcServer *server.MCPServer, logger *log.Logger) {

	// Tools for Vault mount management
	listMountsTool := sys.ListMounts(logger)
	hcServer.AddTool(listMountsTool.Tool, listMountsTool.Handler)

	createMountTool := sys.CreateMount(logger)
	hcServer.AddTool(createMountTool.Tool, createMountTool.Handler)

	deleteMountTool := sys.DeleteMount(logger)
	hcServer.AddTool(deleteMountTool.Tool, deleteMountTool.Handler)

	// Tools for KV secrets management
	listSecretsTool := kv.ListSecrets(logger)
	hcServer.AddTool(listSecretsTool.Tool, listSecretsTool.Handler)

	readSecretTool := kv.ReadSecret(logger)
	hcServer.AddTool(readSecretTool.Tool, readSecretTool.Handler)

	writeSecretTool := kv.WriteSecret(logger)
	hcServer.AddTool(writeSecretTool.Tool, writeSecretTool.Handler)

	deleteSecretTool := kv.DeleteSecret(logger)
	hcServer.AddTool(deleteSecretTool.Tool, deleteSecretTool.Handler)

	// Tools for PKI management
	enablePkiTool := pki.EnablePki(logger)
	hcServer.AddTool(enablePkiTool.Tool, enablePkiTool.Handler)

	createPkiIssuer := pki.CreatePkiIssuer(logger)
	hcServer.AddTool(createPkiIssuer.Tool, createPkiIssuer.Handler)

	listPkiIssuers := pki.ListPkiIssuers(logger)
	hcServer.AddTool(listPkiIssuers.Tool, listPkiIssuers.Handler)

	readPkiIssuer := pki.ReadPkiIssuer(logger)
	hcServer.AddTool(readPkiIssuer.Tool, readPkiIssuer.Handler)

	listPkiRoles := pki.ListPkiRoles(logger)
	hcServer.AddTool(listPkiRoles.Tool, listPkiRoles.Handler)

	readPkiRole := pki.ReadPkiRole(logger)
	hcServer.AddTool(readPkiRole.Tool, readPkiRole.Handler)

	createPkiRole := pki.CreatePkiRole(logger)
	hcServer.AddTool(createPkiRole.Tool, createPkiRole.Handler)

	deletePkiRole := pki.DeletePkiRole(logger)
	hcServer.AddTool(deletePkiRole.Tool, deletePkiRole.Handler)

	issuePkiCertificate := pki.IssuePkiCertificate(logger)
	hcServer.AddTool(issuePkiCertificate.Tool, issuePkiCertificate.Handler)

	// Tools for Transit encryption-as-a-service

	// Key management
	createKey := transit.CreateTransitKey(logger)
	hcServer.AddTool(createKey.Tool, createKey.Handler)

	readKey := transit.ReadTransitKey(logger)
	hcServer.AddTool(readKey.Tool, readKey.Handler)

	rotateKey := transit.RotateTransitKey(logger)
	hcServer.AddTool(rotateKey.Tool, rotateKey.Handler)

	listKeys := transit.ListTransitKeys(logger)
	hcServer.AddTool(listKeys.Tool, listKeys.Handler)

	// Encryption operations
	encryptData := transit.EncryptData(logger)
	hcServer.AddTool(encryptData.Tool, encryptData.Handler)

	decryptData := transit.DecryptData(logger)
	hcServer.AddTool(decryptData.Tool, decryptData.Handler)

	rewrapData := transit.RewrapData(logger)
	hcServer.AddTool(rewrapData.Tool, rewrapData.Handler)

	// Integrity and crypto utilities
	generateHMAC := transit.GenerateHMAC(logger)
	hcServer.AddTool(generateHMAC.Tool, generateHMAC.Handler)

	verifyHMAC := transit.VerifyHMAC(logger)
	hcServer.AddTool(verifyHMAC.Tool, verifyHMAC.Handler)

	signData := transit.SignData(logger)
	hcServer.AddTool(signData.Tool, signData.Handler)

	verifySignature := transit.VerifySignature(logger)
	hcServer.AddTool(verifySignature.Tool, verifySignature.Handler)

	hashData := transit.HashData(logger)
	hcServer.AddTool(hashData.Tool, hashData.Handler)

	randomBytes := transit.GenerateRandomBytes(logger)
	hcServer.AddTool(randomBytes.Tool, randomBytes.Handler)
}
