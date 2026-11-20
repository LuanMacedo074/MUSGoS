# Engine drift against FsosServer

Report from `scripts/engine-drift.sh ../FsosServer`, taken when FsosServer's
engine was ported back into MUSGoS (FSOSDocs #665). Re-run the script after
every sync and update this file when its output changes.

## Engine code

`internal/` and `cmd/gameserver/` are identical in both repos once the module
path is rewritten, except for one MUSGoS-first fix that FsosServer has not
taken yet:

| File | Difference |
|---|---|
| `internal/domain/services/logon_service.go` | Token-mode Logon rejects a token with an empty `user_id`, and logs a warning when it rejects a token for a missing `user_id` or an invalid `user_level`. Carry it to FsosServer with the README recipe, together with `TestLogonService_TokenMode_RejectsEmptyUserID` in `_tests/domain/services/logon_service_test.go`. |

FsosServer's HTTP scripts already issue tokens with an empty `user_id` for
accounts without a character. Those tokens are consumed by the HTTP
create-character flow, which is unaffected. A Logon with one of them used to
bind a session to an empty identity; once carried over, it is refused.

## Engine tests

All remaining drift is in `_tests/`. None of it changes engine behaviour.

### Only in MUSGoS

| File | Why |
|---|---|
| `_tests/adapters/outbound/cache_token_store_test.go` | Token store tests added when MUSGoS first ported the refresh store; FsosServer covers the same store in `generic_token_test.go` only. |
| `_tests/adapters/outbound/lua_auth_modules_test.go` | `mus.bcrypt` and `mus.token` script tests, MUSGoS only. |
| `_tests/adapters/outbound/refresh_token_store_test.go` | Refresh store tests over a fixture schema; FsosServer tests the store through its game migrations. |

### Only in FsosServer

| File | Why |
|---|---|
| `_tests/adapters/inbound/script_catalog_test.go` | Tests FSOS's NPC-script catalog through `external/hooks` and `external/migrations`. It is a game test that sits in the engine tree; it should move to FsosServer's `_tests/game`. |

### In both repos, with differences

| File | Difference |
|---|---|
| `_tests/adapters/inbound/mus/db_commands_test.go` | Comment wrapping and blank lines only. |
| `_tests/adapters/outbound/cache_lock_test.go` | MUSGoS uses neutral lock names (`operation`) and keeps `TestMemoryCache_OperationLockRequiresADeadline`; FsosServer uses `battle`. |
| `_tests/adapters/outbound/lua_cache_lock_test.go` | Lock names, as above. |
| `_tests/adapters/outbound/lua_prefix_binary_test.go` | FsosServer uses a neutral `example/chunk` subject; MUSGoS still uses `painting/saveLayer1`. |
| `_tests/adapters/outbound/lua_script_engine_test.go` | MUSGoS keeps `whereEqualFold` and adds `mus.random` and recipient-list tests; FsosServer moved its helpers to `testutil` and has its own audience test. |
| `_tests/adapters/outbound/memory_cache_test.go` | FsosServer uses neutral set keys; MUSGoS still uses `battle:` keys. |
| `_tests/adapters/outbound/redis_cache_integration_test.go` | The operation-lock Redis test lives in this file in FsosServer and in `redis_cache_lock_integration_test.go` in MUSGoS. |
| `_tests/adapters/outbound/redis_cache_lock_integration_test.go` | MUSGoS skips when Redis is unreachable and prefixes keys per run; FsosServer fails instead and uses `battle` names. |
| `_tests/testutil/mocks.go` | The `MockCache` doc comment only. |

Converging these means taking the better side of each file into the other repo
with the recipe in the README.
