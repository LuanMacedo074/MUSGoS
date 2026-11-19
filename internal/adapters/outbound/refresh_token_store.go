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

// RefreshTokenStore retains only SHA-256 digests, including spent tokens for replay detection.
// Every device family is serialized by a database row write, across server processes.
type RefreshTokenStore struct {
	qb  ports.QueryBuilder
	now func() time.Time
}

func NewRefreshTokenStore(qb ports.QueryBuilder, now func() time.Time) *RefreshTokenStore {
	if now == nil {
		now = time.Now
	}
	return &RefreshTokenStore{qb: qb, now: now}
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

func (s *RefreshTokenStore) Issue(accountID int64, label string) (string, error) {
	var token string
	err := s.retry(func() error {
		var err error
		token, err = s.issueOnce(accountID, label)
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

func (s *RefreshTokenStore) issueOnce(accountID int64, label string) (string, error) {
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
	account, err := tx.Table("accounts").Where("id", accountID).First()
	if err != nil {
		return "", err
	}
	if account == nil {
		return "", fmt.Errorf("invalid account")
	}
	now := s.now().Unix()
	err = tx.Table("refresh_families").Insert(map[string]interface{}{"id": family, "account_id": accountID, "device_label": label, "password_hash": account["password_hash"], "created_at": now, "last_used_at": now, "expires_at": now + int64(RefreshTokenTTL/time.Second)})
	if err != nil {
		return "", err
	}
	if err = tx.Table("refresh_tokens").Insert(map[string]interface{}{"token_hash": refreshDigest(token), "family_id": family}); err != nil {
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
	var account int64
	var rotated string
	err := s.retry(func() error {
		var err error
		account, rotated, err = s.rotateOnce(token)
		return err
	})
	return account, rotated, err
}

func (s *RefreshTokenStore) rotateOnce(token string) (int64, string, error) {
	// Resolve only the immutable family id before opening the transaction. The
	// first transactional statement then obtains the write lock without upgrading
	// a read lock (which can deadlock two SQLite refreshes).
	row, err := s.qb.Table("refresh_tokens").Where("token_hash", refreshDigest(token)).First()
	if err != nil || row == nil {
		return 0, "", err
	}
	familyID := row["family_id"]

	tx, err := s.begin()
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()
	if _, err = tx.Table("refresh_families").Where("id", familyID).Increment("revision", 1); err != nil {
		return 0, "", err
	}
	family, err := tx.Table("refresh_families").Where("id", familyID).First()
	if err != nil || family == nil {
		return 0, "", err
	}
	// Read again after acquiring the family lock: a concurrent rotation may have spent it.
	row, err = tx.Table("refresh_tokens").Where("token_hash", refreshDigest(token)).First()
	if err != nil || row == nil {
		return 0, "", err
	}
	accountID := refreshInt(family["account_id"])
	account, err := tx.Table("accounts").Where("id", accountID).First()
	if err != nil {
		return 0, "", err
	}
	now := s.now().Unix()
	if refreshInt(row["used"]) != 0 || refreshInt(family["revoked"]) != 0 || refreshInt(family["expires_at"]) <= now || account == nil || account["password_hash"] != family["password_hash"] || refreshInt(account["banned"]) != 0 {
		if _, err = tx.Table("refresh_families").Where("id", familyID).Update(map[string]interface{}{"revoked": 1}); err != nil {
			return 0, "", err
		}
		return 0, "", tx.Commit()
	}
	rotated, err := refreshSecret()
	if err != nil {
		return 0, "", err
	}
	if _, err = tx.Table("refresh_tokens").Where("token_hash", refreshDigest(token)).Update(map[string]interface{}{"used": 1}); err != nil {
		return 0, "", err
	}
	if err = tx.Table("refresh_tokens").Insert(map[string]interface{}{"token_hash": refreshDigest(rotated), "family_id": familyID}); err != nil {
		return 0, "", err
	}
	if _, err = tx.Table("refresh_families").Where("id", familyID).Update(map[string]interface{}{"last_used_at": now, "expires_at": now + int64(RefreshTokenTTL/time.Second)}); err != nil {
		return 0, "", err
	}
	if err = tx.Commit(); err != nil {
		return 0, "", err
	}
	return accountID, rotated, nil
}

func (s *RefreshTokenStore) Revoke(token string) error {
	return s.retry(func() error { return s.revokeOnce(token) })
}

func (s *RefreshTokenStore) revokeOnce(token string) error {
	row, err := s.qb.Table("refresh_tokens").Where("token_hash", refreshDigest(token)).First()
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
	if _, err = tx.Table("refresh_families").Where("id", row["family_id"]).Update(map[string]interface{}{"revoked": 1}); err != nil {
		return err
	}
	return tx.Commit()
}
