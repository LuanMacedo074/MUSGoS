# Refresh tokens (`mus.refresh`)

The engine provides a remembered-device credential store, `RefreshTokenStore`
(`internal/adapters/outbound/refresh_token_store.go`), exposed to scripts as
`mus.refresh` whenever the script engine has a query builder:

- `mus.refresh.issue(account_id, device_label)` returns a new opaque token:
  32 random bytes as 64 hex digits. Labels longer than 128 bytes are cut on a
  UTF-8 boundary. Each call starts a new device family.
- `mus.refresh.rotate(token)` returns `account_id, rotated_token`, or `nil`
  when the token is unknown, spent, revoked, expired, or the account is gone,
  banned, or its password hash has changed since issue. Replaying a spent
  token revokes its whole family, including newer tokens, and leaves other
  families alone. A successful rotation extends the family's expiry to 30 days
  after that use.
- `mus.refresh.revoke(token)` revokes the token's family. Unknown tokens are
  ignored.

Failures raise a generic Lua error.

## Schema the application supplies

MUSGoS ships no migration for this store. Its tables hang off an `accounts`
table, which is application schema: the engine's own identity table is `users`.
An application that uses `mus.refresh` must provide:

- `accounts`, with at least `id`, `password_hash` and `banned` (read by the store)
- `refresh_families` (`id`, `account_id` referencing `accounts(id)`,
  `device_label`, `password_hash`, `created_at`, `last_used_at`, `expires_at`
  in Unix seconds, `revoked`, `revision`)
- `refresh_tokens` (`token_hash`, `family_id` referencing
  `refresh_families(id)`, `used`)

FsosServer's `20261117000028_remembered_login` migration is the reference
schema, for both SQLite and Postgres.

`mus.refresh` is registered even when these tables are missing. On such a
database every call raises the same generic Lua error (`refresh issue failed`,
`refresh rotation failed` or `refresh revocation failed`). The underlying SQL
error is not passed to the script.

Only SHA-256 digests of tokens are stored, never the tokens themselves. To
revoke every device when a password changes, an application can add a trigger
on `accounts.password_hash` that sets `refresh_families.revoked`.

Spent hashes are what make replay detection work. A retention job must keep
them for any family that can still accept a newer token.

## Concurrency and retry

Each operation runs in its own transaction and serializes its family with a row
write (`revision`), which holds across server processes. The store uses the
query builder's optional `ports.QueryRetryer` capability. The SQL builder's
`Retry` re-runs the whole operation after rollback, up to 10 times with
exponential backoff, while the dialect's `RetryableConflict` classifies the
error. SQLite retries `SQLITE_BUSY` lock contention. Postgres retries nothing
and keeps its native row-lock waiting.
