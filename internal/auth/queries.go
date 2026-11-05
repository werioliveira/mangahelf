package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgconn"
)

var (
)

type Queries struct {
	db *pgxpool.Pool
}

func NewQueries(db *pgxpool.Pool) *Queries {
	return &Queries{db: db}
}

func (q *Queries) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := q.db.Exec(ctx, query, args...)
	return err
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var emailVerifiedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`SELECT id, email, email_verified_at, username, password_hash, status, created_at, updated_at
		 FROM users WHERE email = $1`, email).Scan(
		&u.ID, &u.Email, &emailVerifiedAt, &u.Username, &passwordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = &passwordHash.String
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var emailVerifiedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`SELECT id, email, email_verified_at, username, password_hash, status, created_at, updated_at
		 FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Email, &emailVerifiedAt, &u.Username, &passwordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = &passwordHash.String
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var emailVerifiedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`SELECT id, email, email_verified_at, username, password_hash, status, created_at, updated_at
		 FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Email, &emailVerifiedAt, &u.Username, &passwordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = &passwordHash.String
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (q *Queries) CreateUser(ctx context.Context, email, username, passwordHash string) (*User, error) {
	var u User
	var emailVerifiedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`INSERT INTO users (email, username, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, email_verified_at, username, password_hash, status, created_at, updated_at`,
		email, username, passwordHash).Scan(
		&u.ID, &u.Email, &emailVerifiedAt, &u.Username, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			if containsEmail(err) {
				return nil, ErrEmailExists
			}
			return nil, ErrUsernameExists
		}
		return nil, err
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

func (q *Queries) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`,
		passwordHash, id)
	return err
}

func (q *Queries) UpdateUserEmail(ctx context.Context, id uuid.UUID, email string, verified bool) error {
	var emailVerifiedAt sql.NullTime
	if verified {
		emailVerifiedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	_, err := q.db.Exec(ctx,
		`UPDATE users SET email = $1, email_verified_at = $2, updated_at = now() WHERE id = $3`,
		email, emailVerifiedAt, id)
	return err
}

func (q *Queries) UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE users SET status = $1, updated_at = now() WHERE id = $2`,
		status, id)
	return err
}

func (q *Queries) InsertUserSession(ctx context.Context, sid, userID uuid.UUID, authTime time.Time, mfaAt *time.Time, ip netip.Addr, ua string) error {
	var mfaAtVal sql.NullTime
	if mfaAt != nil {
		mfaAtVal = sql.NullTime{Time: *mfaAt, Valid: true}
	}
	_, err := q.db.Exec(ctx,
		`INSERT INTO user_sessions (id, user_id, auth_time, mfa_at, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		sid, userID, authTime, mfaAtVal, ip, ua)
	return err
}

func (q *Queries) SessionPrincipal(ctx context.Context, sid uuid.UUID) (*SessionPrincipalRow, error) {
	var r SessionPrincipalRow
	var mfaAt sql.NullTime
	var revokedAt sql.NullTime
	var lastSeenAt sql.NullTime
	var ip netip.Addr
	err := q.db.QueryRow(ctx,
		`SELECT us.id, us.user_id, u.status, us.auth_time, us.mfa_at, us.revoked_at, us.revoked_reason, us.ip, us.user_agent, us.last_seen_at
		 FROM user_sessions us
		 JOIN users u ON u.id = us.user_id
		 WHERE us.id = $1`, sid).Scan(
		&r.SID, &r.UserID, &r.Status, &r.AuthTime, &mfaAt, &revokedAt, &r.RevokedReason, &ip, &r.UserAgent, &lastSeenAt)
	if err == pgx.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if mfaAt.Valid {
		r.MFAAt = &mfaAt.Time
	}
	if revokedAt.Valid {
		r.RevokedAt = &revokedAt.Time
	}
	if lastSeenAt.Valid {
		r.LastSeenAt = &lastSeenAt.Time
	}
	return &r, nil
}

func (q *Queries) RevokeSession(ctx context.Context, sid uuid.UUID, reason string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = now(), revoked_reason = $1 WHERE id = $2`,
		reason, sid)
	return err
}

func (q *Queries) RevokeUserSessions(ctx context.Context, userID uuid.UUID, exceptSID *uuid.UUID, reason string) error {
	if exceptSID != nil {
		_, err := q.db.Exec(ctx,
			`UPDATE user_sessions SET revoked_at = now(), revoked_reason = $1
			 WHERE user_id = $2 AND id != $3 AND revoked_at IS NULL`,
			reason, userID, *exceptSID)
		return err
	}
	_, err := q.db.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = now(), revoked_reason = $1
		 WHERE user_id = $2 AND revoked_at IS NULL`,
		reason, userID)
	return err
}

func (q *Queries) RevokeOldestBeyond(ctx context.Context, userID uuid.UUID, max int, reason string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_sessions SET revoked_at = now(), revoked_reason = $1
		 WHERE id IN (
			 SELECT id FROM user_sessions
			 WHERE user_id = $2 AND revoked_at IS NULL
			 ORDER BY created_at ASC
			 LIMIT GREATEST((SELECT count(*) FROM user_sessions WHERE user_id = $2 AND revoked_at IS NULL) - $3, 0)
		 )`, reason, userID, max)
	return err
}

func (q *Queries) TouchSession(ctx context.Context, sid uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_sessions SET last_seen_at = now() WHERE id = $1
		 AND last_seen_at < now() - interval '1 minute'`, sid)
	return err
}

func (q *Queries) UpdateSessionMFA(ctx context.Context, sid uuid.UUID, mfaAt time.Time) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_sessions SET mfa_at = $1 WHERE id = $2`, mfaAt, sid)
	return err
}

func (q *Queries) InsertAuthToken(ctx context.Context, userID uuid.UUID, purpose string, tokenHash []byte, meta json.RawMessage, expiresAt time.Time) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO auth_tokens (user_id, purpose, token_hash, meta, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, purpose, tokenHash, meta, expiresAt)
	return err
}

func (q *Queries) InvalidateAuthTokens(ctx context.Context, userID uuid.UUID, purpose string) error {
	_, err := q.db.Exec(ctx,
		`UPDATE auth_tokens SET used_at = now()
		 WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL`,
		userID, purpose)
	return err
}

func (q *Queries) ConsumeToken(ctx context.Context, tokenHash []byte, purpose string) (uuid.UUID, json.RawMessage, error) {
	var userID uuid.UUID
	var meta json.RawMessage
	err := q.db.QueryRow(ctx,
		`UPDATE auth_tokens
		 SET used_at = now()
		 WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		 RETURNING user_id, meta`,
		tokenHash, purpose).Scan(&userID, &meta)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil, ErrTokenInvalid
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	return userID, meta, nil
}

func (q *Queries) InsertTOTP(ctx context.Context, userID uuid.UUID, secretEnc []byte, keyID int16) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO user_totp (user_id, secret_enc, key_id, last_used_step)
		 VALUES ($1, $2, $3, 0)
		 ON CONFLICT (user_id) DO UPDATE SET secret_enc = $2, key_id = $3, confirmed_at = NULL, last_used_step = 0`,
		userID, secretEnc, keyID)
	return err
}

func (q *Queries) GetTOTP(ctx context.Context, userID uuid.UUID) (*TOTPRecord, error) {
	var r TOTPRecord
	var confirmedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`SELECT user_id, secret_enc, key_id, confirmed_at, last_used_step
		 FROM user_totp WHERE user_id = $1`, userID).Scan(
		&r.UserID, &r.SecretEnc, &r.KeyID, &confirmedAt, &r.LastUsedStep)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if confirmedAt.Valid {
		r.ConfirmedAt = &confirmedAt.Time
	}
	return &r, nil
}

func (q *Queries) ConfirmTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_totp SET confirmed_at = now() WHERE user_id = $1`, userID)
	return err
}

func (q *Queries) AdvanceTOTPStep(ctx context.Context, userID uuid.UUID, step int64) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx,
		`UPDATE user_totp SET last_used_step = $1
		 WHERE user_id = $2 AND last_used_step < $1
		 RETURNING 1`, step, userID).Scan(&n)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (q *Queries) DeleteTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := q.db.Exec(ctx, `DELETE FROM user_totp WHERE user_id = $1`, userID)
	return err
}

func (q *Queries) InsertRecoveryCodes(ctx context.Context, userID uuid.UUID, codes [][]byte) error {
	tx, err := q.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM user_recovery_codes WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	for _, code := range codes {
		_, err = tx.Exec(ctx,
			`INSERT INTO user_recovery_codes (user_id, code_hmac) VALUES ($1, $2)`,
			userID, code)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (q *Queries) ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, codeHMAC []byte) (bool, error) {
	var n int64
	err := q.db.QueryRow(ctx,
		`UPDATE user_recovery_codes SET used_at = now()
		 WHERE user_id = $1 AND code_hmac = $2 AND used_at IS NULL
		 RETURNING 1`, userID, codeHMAC).Scan(&n)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (q *Queries) GetRecoveryCodeCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := q.db.QueryRow(ctx,
		`SELECT count(*) FROM user_recovery_codes WHERE user_id = $1 AND used_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (q *Queries) InsertExternalIdentity(ctx context.Context, provider, subject string, userID uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO external_identities (provider, subject, user_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider, subject) DO NOTHING`,
		provider, subject, userID)
	return err
}

func (q *Queries) GetUserByExternalIdentity(ctx context.Context, provider, subject string) (*User, error) {
	var u User
	var passwordHash sql.NullString
	var emailVerifiedAt sql.NullTime
	err := q.db.QueryRow(ctx,
		`SELECT u.id, u.email, u.email_verified_at, u.username, u.password_hash, u.status, u.created_at, u.updated_at
		 FROM users u
		 JOIN external_identities ei ON ei.user_id = u.id
		 WHERE ei.provider = $1 AND ei.subject = $2`,
		provider, subject).Scan(
		&u.ID, &u.Email, &emailVerifiedAt, &u.Username, &passwordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		u.PasswordHash = &passwordHash.String
	}
	if emailVerifiedAt.Valid {
		u.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return &u, nil
}

type SessionPrincipalRow struct {
	SID           uuid.UUID
	UserID        uuid.UUID
	Status        string
	AuthTime      time.Time
	MFAAt         *time.Time
	RevokedAt     *time.Time
	RevokedReason string
	IP            netip.Addr
	UserAgent     string
	LastSeenAt    *time.Time
}

type User struct {
	ID              uuid.UUID
	Email           string
	EmailVerifiedAt *time.Time
	Username        string
	PasswordHash    *string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TOTPRecord struct {
	UserID       uuid.UUID
	SecretEnc    []byte
	KeyID        int16
	ConfirmedAt  *time.Time
	LastUsedStep int64
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func containsEmail(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName == "users_email_key" || pgErr.Message == "users_email_key"
	}
	return false
}

func (q *Queries) CreateOIDCState(ctx context.Context, state string, expiresAt time.Time) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO oidc_states (state, expires_at) VALUES ($1, $2)`, state, expiresAt)
	return err
}

func (q *Queries) ConsumeOIDCState(ctx context.Context, state string) (bool, error) {
	var n int64
	err := q.db.QueryRow(ctx,
		`DELETE FROM oidc_states WHERE state = $1 AND expires_at > now() RETURNING 1`, state).Scan(&n)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return n == 1, err
}

func (q *Queries) GetUserByCredentialID(ctx context.Context, credentialID []byte) (uuid.UUID, error) {
	var userID uuid.UUID
	err := q.db.QueryRow(ctx,
		`SELECT user_id FROM user_webauthn_credentials WHERE id = $1`, credentialID).Scan(&userID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, ErrCredentialNotFound
	}
	return userID, err
}

func (q *Queries) GetCredentialSignCount(ctx context.Context, credentialID []byte) (uint32, error) {
	var count uint32
	err := q.db.QueryRow(ctx,
		`SELECT sign_count FROM user_webauthn_credentials WHERE id = $1`, credentialID).Scan(&count)
	return count, err
}

func (q *Queries) UpdateCredentialSignCount(ctx context.Context, credentialID []byte, count uint32) error {
	_, err := q.db.Exec(ctx,
		`UPDATE user_webauthn_credentials SET sign_count = $1, last_used_at = now() WHERE id = $2`,
		count, credentialID)
	return err
}

func (q *Queries) ListCredentials(ctx context.Context, userID uuid.UUID) ([]CredentialInfo, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, name, transports, backup_eligible, backup_state, last_used_at, created_at
		 FROM user_webauthn_credentials WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []CredentialInfo
	for rows.Next() {
		var c CredentialInfo
		var lastUsed sql.NullTime
		if err := rows.Scan(&c.ID, &c.Name, &c.Transports, &c.BackupEligible, &c.BackupState, &lastUsed, &c.CreatedAt); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			c.LastUsedAt = &lastUsed.Time
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (q *Queries) DeleteCredential(ctx context.Context, userID uuid.UUID, credentialID []byte) error {
	_, err := q.db.Exec(ctx,
		`DELETE FROM user_webauthn_credentials WHERE user_id = $1 AND id = $2`,
		userID, credentialID)
	return err
}

func (q *Queries) GetCredentialCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := q.db.QueryRow(ctx,
		`SELECT count(*) FROM user_webauthn_credentials WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}