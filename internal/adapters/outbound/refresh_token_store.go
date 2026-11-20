package outbound

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	"fsos-server/internal/domain/ports"
)

var _ ports.RefreshTokenStore = (*RefreshTokenStore)(nil)

const RefreshTokenTTL = 30 * 24 * time.Hour

// RefreshTokenConfig maps identity policy and persisted family fields to the application's schema.
type RefreshTokenConfig struct {
	IdentityTable          string
	IDColumn               string
	CredentialColumn       string
	BanColumn              string
	FamilyIdentityColumn   string
	FamilyCredentialColumn string
	FamilyTable            string
	FamilyIDColumn         string
	DeviceLabelColumn      string
	CreatedAtColumn        string
	LastUsedAtColumn       string
	ExpiresAtColumn        string
	RevokedColumn          string
	RevisionColumn         string
	TokenTable             string
	TokenHashColumn        string
	TokenFamilyColumn      string
	TokenUsedColumn        string
}

// RefreshTokenStore retains only SHA-256 digests, including spent tokens for replay detection.
// Every device family is serialized by a database row write, across server processes.
type RefreshTokenStore struct {
	config RefreshTokenConfig
	qb     ports.QueryBuilder
	now    func() time.Time
}

func NewRefreshTokenStore(qb ports.QueryBuilder, now func() time.Time, config RefreshTokenConfig) *RefreshTokenStore {
	if now == nil {
		now = time.Now
	}
	return &RefreshTokenStore{qb: qb, now: now, config: config}
}
func refreshSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func refreshDigest(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func refreshInt(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}
func (s *RefreshTokenStore) begin() (ports.Tx, error) {
	qb, ok := s.qb.(ports.TransactionalQueryBuilder)
	if !ok {
		return nil, fmt.Errorf("refresh tokens require transactions")
	}
	return qb.Begin()
}

func (s *RefreshTokenStore) Issue(userID int64, label string) (string, error) {
	var token string
	err := s.retry(func() error {
		var err error
		token, err = s.issueOnce(userID, label)
		return err
	})
	return token, err
}

func (s *RefreshTokenStore) retry(operation func() error) error {
	if qb, ok := s.qb.(ports.QueryRetryer); ok {
		return qb.Retry(operation)
	}
	return operation()
}

func (s *RefreshTokenStore) issueOnce(userID int64, label string) (string, error) {
	if len(label) > 128 {
		label = label[:128]
		for !utf8.ValidString(label) {
			label = label[:len(label)-1]
		}
	}
	token, err := refreshSecret()
	if err != nil {
		return "", err
	}
	family, err := refreshSecret()
	if err != nil {
		return "", err
	}
	tx, err := s.begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	identity, err := tx.Table(s.config.IdentityTable).Where(s.config.IDColumn, userID).First()
	if err != nil {
		return "", err
	}
	if identity == nil {
		return "", fmt.Errorf("invalid identity")
	}
	now := s.now().Unix()
	err = tx.Table(s.config.FamilyTable).Insert(map[string]interface{}{
		s.config.FamilyIDColumn:         family,
		s.config.FamilyIdentityColumn:   userID,
		s.config.DeviceLabelColumn:      label,
		s.config.FamilyCredentialColumn: identity[s.config.CredentialColumn],
		s.config.CreatedAtColumn:        now,
		s.config.LastUsedAtColumn:       now,
		s.config.ExpiresAtColumn:        now + int64(RefreshTokenTTL/time.Second),
	})
	if err != nil {
		return "", err
	}
	if err = tx.Table(s.config.TokenTable).Insert(map[string]interface{}{s.config.TokenHashColumn: refreshDigest(token), s.config.TokenFamilyColumn: family}); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

// Rotate returns zero identity for every rejected credential. A spent token revokes its
// entire family in a committed transaction, even though authentication is refused.
func (s *RefreshTokenStore) Rotate(token string) (int64, string, error) {
	var identity int64
	var rotated string
	err := s.retry(func() error {
		var err error
		identity, rotated, err = s.rotateOnce(token)
		return err
	})
	return identity, rotated, err
}

func (s *RefreshTokenStore) rotateOnce(token string) (int64, string, error) {
	// Resolve only the immutable family id before opening the transaction. The
	// first transactional statement then obtains the write lock without upgrading
	// a read lock (which can deadlock two SQLite refreshes).
	row, err := s.qb.Table(s.config.TokenTable).Where(s.config.TokenHashColumn, refreshDigest(token)).First()
	if err != nil || row == nil {
		return 0, "", err
	}
	familyID := row[s.config.TokenFamilyColumn]

	tx, err := s.begin()
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()
	if _, err = tx.Table(s.config.FamilyTable).Where(s.config.FamilyIDColumn, familyID).Increment(s.config.RevisionColumn, 1); err != nil {
		return 0, "", err
	}
	family, err := tx.Table(s.config.FamilyTable).Where(s.config.FamilyIDColumn, familyID).First()
	if err != nil || family == nil {
		return 0, "", err
	}
	// Read again after acquiring the family lock: a concurrent rotation may have spent it.
	row, err = tx.Table(s.config.TokenTable).Where(s.config.TokenHashColumn, refreshDigest(token)).First()
	if err != nil || row == nil {
		return 0, "", err
	}
	userID := refreshInt(family[s.config.FamilyIdentityColumn])
	identity, err := tx.Table(s.config.IdentityTable).Where(s.config.IDColumn, userID).First()
	if err != nil {
		return 0, "", err
	}
	now := s.now().Unix()
	if refreshInt(row[s.config.TokenUsedColumn]) != 0 || refreshInt(family[s.config.RevokedColumn]) != 0 || refreshInt(family[s.config.ExpiresAtColumn]) <= now || identity == nil || identity[s.config.CredentialColumn] != family[s.config.FamilyCredentialColumn] || refreshInt(identity[s.config.BanColumn]) != 0 {
		if _, err = tx.Table(s.config.FamilyTable).Where(s.config.FamilyIDColumn, familyID).Update(map[string]interface{}{s.config.RevokedColumn: 1}); err != nil {
			return 0, "", err
		}
		return 0, "", tx.Commit()
	}
	rotated, err := refreshSecret()
	if err != nil {
		return 0, "", err
	}
	if _, err = tx.Table(s.config.TokenTable).Where(s.config.TokenHashColumn, refreshDigest(token)).Update(map[string]interface{}{s.config.TokenUsedColumn: 1}); err != nil {
		return 0, "", err
	}
	if err = tx.Table(s.config.TokenTable).Insert(map[string]interface{}{s.config.TokenHashColumn: refreshDigest(rotated), s.config.TokenFamilyColumn: familyID}); err != nil {
		return 0, "", err
	}
	if _, err = tx.Table(s.config.FamilyTable).Where(s.config.FamilyIDColumn, familyID).Update(map[string]interface{}{s.config.LastUsedAtColumn: now, s.config.ExpiresAtColumn: now + int64(RefreshTokenTTL/time.Second)}); err != nil {
		return 0, "", err
	}
	if err = tx.Commit(); err != nil {
		return 0, "", err
	}
	return userID, rotated, nil
}

func (s *RefreshTokenStore) Revoke(token string) error {
	return s.retry(func() error { return s.revokeOnce(token) })
}

func (s *RefreshTokenStore) revokeOnce(token string) error {
	row, err := s.qb.Table(s.config.TokenTable).Where(s.config.TokenHashColumn, refreshDigest(token)).First()
	if err != nil || row == nil {
		return err
	}
	// As in rotation, resolve the immutable family id before opening the transaction.
	// Its first statement writes the family row, avoiding a SQLite read-lock upgrade.
	tx, err := s.begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Table(s.config.FamilyTable).Where(s.config.FamilyIDColumn, row[s.config.TokenFamilyColumn]).Update(map[string]interface{}{s.config.RevokedColumn: 1}); err != nil {
		return err
	}
	return tx.Commit()
}
