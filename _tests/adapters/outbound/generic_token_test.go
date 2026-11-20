package outbound_test

import (
	"reflect"
	"testing"
	"time"

	"musgos/_tests/testutil"
	"musgos/internal/adapters/outbound"
	"musgos/internal/domain/ports"
	"musgos/internal/factory"
)

func TestTokenClaimsRoundTrip(t *testing.T) {
	store := outbound.NewCacheTokenStore(testutil.NewMockCache(), time.Minute)
	want := ports.TokenPayload{UserID: "reader", Claims: map[string]string{"user_level": "40", "tenant": "blue", "empty": "", "unicode": "olá"}}
	token, err := store.Issue(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Validate(token)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
	if _, err := store.Validate(token); err != ports.ErrTokenInvalid {
		t.Fatal("token reused")
	}
}

func TestTokenReadsLegacyFlatPayload(t *testing.T) {
	cache := testutil.NewMockCache()
	if err := cache.Set("authtoken:before-deploy", []byte(`{"character_name":"Hero","user_level":40,"account_id":7}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	store := outbound.NewCacheTokenStore(cache, time.Minute).WithLegacyUserIDClaim("character_name")
	got, err := store.Validate("before-deploy")
	want := ports.TokenPayload{UserID: "Hero", Claims: map[string]string{"character_name": "Hero", "user_level": "40", "account_id": "7"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
}

func TestRefreshStoreUsesConfiguredIdentityColumns(t *testing.T) {
	db := newTestDBWithMigrations(t, nil)
	mustNoErr(t, db.CreateTable(ports.Table{Name: "device_grants", Columns: []ports.Column{
		ports.Col("grant_key", ports.ColText).NotNull(),
		ports.Col("principal_id", ports.ColInteger).NotNull(),
		ports.Col("credential", ports.ColText).NotNull(),
		ports.Col("client_label", ports.ColText).NotNull(),
		ports.Col("issued_on", ports.ColInteger).NotNull(),
		ports.Col("seen_on", ports.ColInteger).NotNull(),
		ports.Col("valid_until", ports.ColInteger).NotNull(),
		ports.Col("version", ports.ColInteger).NotNull().Default(0),
		ports.Col("disabled_grant", ports.ColInteger).NotNull().Default(0),
	}, PrimaryKeys: []string{"grant_key"}}))
	mustNoErr(t, db.CreateTable(ports.Table{Name: "device_secrets", Columns: []ports.Column{
		ports.Col("digest", ports.ColText).NotNull(),
		ports.Col("grant_ref", ports.ColText).NotNull(),
		ports.Col("spent", ports.ColInteger).NotNull().Default(0),
	}, PrimaryKeys: []string{"digest"}}))
	err := db.CreateTable(ports.Table{Name: "members", Columns: []ports.Column{
		ports.Col("member_key", ports.ColInteger).NotNull(), ports.Col("secret", ports.ColText).NotNull(), ports.Col("disabled", ports.ColInteger).NotNull().Default(0),
	}, PrimaryKeys: []string{"member_key"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.QueryBuilder().Table("members").Insert(map[string]interface{}{"member_key": 91, "secret": "digest"}); err != nil {
		t.Fatal(err)
	}
	cfg := outbound.RefreshTokenConfig{IdentityTable: "members", IDColumn: "member_key", CredentialColumn: "secret", BanColumn: "disabled", FamilyIdentityColumn: "principal_id", FamilyCredentialColumn: "credential",
		FamilyTable: "device_grants", FamilyIDColumn: "grant_key", DeviceLabelColumn: "client_label",
		CreatedAtColumn: "issued_on", LastUsedAtColumn: "seen_on", ExpiresAtColumn: "valid_until",
		RevisionColumn: "version", RevokedColumn: "disabled_grant",
		TokenTable: "device_secrets", TokenHashColumn: "digest", TokenFamilyColumn: "grant_ref", TokenUsedColumn: "spent",
	}
	store := factory.NewAuthStores(nil, db.QueryBuilder(), factory.AuthConfig{Refresh: cfg}).Refresh
	token, err := store.Issue(91, "browser")
	if err != nil {
		t.Fatal(err)
	}
	id, rotated, err := store.Rotate(token)
	if err != nil || id != 91 || rotated == "" {
		t.Fatalf("rotation: %d, %q, %v", id, rotated, err)
	}
	if err := store.Revoke(rotated); err != nil {
		t.Fatal(err)
	}
	if id, _, err := store.Rotate(rotated); err != nil || id != 0 {
		t.Fatalf("revoked custom grant accepted: %d %v", id, err)
	}
	token, err = store.Issue(91, "other browser")
	if err != nil {
		t.Fatal(err)
	}
	_, rotated, err = store.Rotate(token)
	if err != nil {
		t.Fatal(err)
	}
	if id, _, err := store.Rotate(token); err != nil || id != 0 {
		t.Fatalf("spent custom token accepted: %d %v", id, err)
	}
	if id, _, err := store.Rotate(rotated); err != nil || id != 0 {
		t.Fatalf("custom family replay revocation lost: %d %v", id, err)
	}
	rotated, err = store.Issue(91, "third browser")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.QueryBuilder().Table("members").Where("member_key", 91).Update(map[string]interface{}{"disabled": 1}); err != nil {
		t.Fatal(err)
	}
	id, _, err = store.Rotate(rotated)
	if err != nil || id != 0 {
		t.Fatalf("disabled member accepted: %d, %v", id, err)
	}
}

func TestLegacyTokenRejectsNonScalarClaims(t *testing.T) {
	for _, value := range []string{`true`, `null`, `{"nested":"value"}`, `["item"]`} {
		t.Run(value, func(t *testing.T) {
			cache := testutil.NewMockCache()
			mustNoErr(t, cache.Set("authtoken:malformed", []byte(`{"login":"reader","extra":`+value+`}`), time.Minute))
			store := outbound.NewCacheTokenStore(cache, time.Minute).WithLegacyUserIDClaim("login")
			if _, err := store.Validate("malformed"); err != ports.ErrTokenInvalid {
				t.Fatalf("invalid legacy claim accepted: %v", err)
			}
			if _, err := store.Validate("malformed"); err != ports.ErrTokenInvalid {
				t.Fatal("malformed token reused")
			}
		})
	}
}
