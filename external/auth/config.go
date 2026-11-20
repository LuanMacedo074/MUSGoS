package auth

import (
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/factory"
)

// Config maps mus.refresh onto the application schema described in
// docs/refresh-tokens.md. Applications with another schema edit the names here;
// LegacyUserIDClaim stays empty because there are no pre-claims tokens to read.
func Config() factory.AuthConfig {
	return factory.AuthConfig{
		Refresh: outbound.RefreshTokenConfig{
			IdentityTable: "accounts", IDColumn: "id", CredentialColumn: "password_hash", BanColumn: "banned",
			FamilyIdentityColumn: "account_id", FamilyCredentialColumn: "password_hash",
			FamilyTable: "refresh_families", FamilyIDColumn: "id", DeviceLabelColumn: "device_label",
			CreatedAtColumn: "created_at", LastUsedAtColumn: "last_used_at", ExpiresAtColumn: "expires_at",
			RevokedColumn: "revoked", RevisionColumn: "revision",
			TokenTable: "refresh_tokens", TokenHashColumn: "token_hash", TokenFamilyColumn: "family_id", TokenUsedColumn: "used",
		},
	}
}
