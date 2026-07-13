# Hexagonal Architecture — MUSGoS

## What is Hexagonal Architecture?

The core idea is simple: **the domain (your system's logic) does not depend on anything external**. It doesn't know whether the connection comes over TCP, HTTP, or a unit test. It doesn't know whether the cryptography is Blowfish, AES, or a mock. It only knows **contracts** (interfaces).

This is achieved by separating the code into three layers:

```
                    ┌─────────────────────────────┐
  outside world     │       ADAPTERS (inbound)     │  ← receives data from the outside world
  (TCP clients) ──► │   tcp_server, smus_handler   │
                    └──────────────┬───────────────┘
                                   │ implements / uses
                    ┌──────────────▼───────────────┐
                    │           DOMAIN              │  ← the core of the system
                    │   types (lingo, smus)         │
                    │   ports (interfaces)          │
                    └──────────────┬───────────────┘
                                   │ defines contracts
                    ┌──────────────▼───────────────┐
  outside world     │      ADAPTERS (outbound)      │  ← provides capabilities to the domain
  (crypto, logs) ◄──│     blowfish, file_logger     │
                    └─────────────────────────────┘
```

---

## Folder structure

```
internal/
├── config/                           ← loads environment variables
│   └── config.go                     ← ServerConfig struct + LoadServerConfig()
│
├── factory/                          ← resolves which concrete implementation to use
│   ├── cipher.go                     ← NewCipher() — picks the cipher by type
│   ├── database.go                   ← NewDatabase() — creates DB + migration runner
│   ├── handler.go                    ← NewHandler() — picks the handler by protocol
│   ├── logger.go                     ← NewLogger() — picks the logger by type
│   ├── queue.go                      ← NewMessageQueue() — memory, Redis or RabbitMQ
│   ├── script_engine.go              ← NewScriptEngine() — creates the Lua engine
│   └── session_store.go              ← NewSessionStore() — memory or Redis
│
├── domain/                           ← the core, depends on nothing external
│   ├── types/
│   │   ├── lingo/                    ← Lingo types (LValue, LString, LInteger, etc.)
│   │   │   ├── codec.go             ← JSON marshal/unmarshal of LValues
│   │   │   └── lua_convert.go       ← bidirectional conversion LValue ↔ Lua
│   │   └── smus/                     ← SMUS protocol types (MUSMessage, headers)
│   │       └── mus_error_code.go     ← ~54 MUS protocol error constants
│   ├── ports/                        ← interfaces (contracts)
│   │   ├── cipher.go                 ← Cipher interface
│   │   ├── connection_writer.go      ← ConnectionWriter interface (write + remap + disconnect)
│   │   ├── database.go               ← DBAdapter interface (users, bans, attributes)
│   │   ├── query_builder.go          ← QueryBuilder + Query interfaces (fluent table ops)
│   │   ├── handler.go                ← MessageHandler interface
│   │   ├── logger.go                 ← Logger interface + LogLevel
│   │   ├── message_sender.go         ← MessageSender interface (message sending)
│   │   ├── migration.go              ← Migration + MigrationTracker interfaces
│   │   ├── schema.go                 ← DSL for table/index definitions
│   │   ├── queue.go                  ← QueuePublisher, QueueConsumer, MessageQueue interfaces
│   │   ├── script_engine.go          ← ScriptEngine interface
│   │   └── session_store.go          ← SessionStore interface
│   └── services/
│       └── migration_runner.go       ← runs pending migrations in order
│
└── adapters/                         ← concrete implementations
    ├── inbound/                      ← INBOUND adapters
    │   ├── mus/                      ← MUS-protocol-specific logic
    │   │   ├── system_service.go     ← SystemService (handler map, logon, permission checks, DB command helper)
    │   │   ├── system_service_server.go  ← handlers: getVersion, getTime, getUserCount, getMovieCount, getMovies
    │   │   ├── system_service_movie.go   ← handlers: movie.getUserCount, movie.getGroups, movie.getGroupCount
    │   │   ├── system_service_group.go   ← handlers: group.join/leave/getUsers/getUserCount/set/get/deleteAttribute
    │   │   ├── system_service_user.go    ← handlers: user.getAddress, user.getGroups, user.delete (with cleanup)
    │   │   ├── system_service_db_player.go      ← handlers: DBPlayer.get/set/delete/getAttributeNames
    │   │   ├── system_service_db_application.go ← handlers: DBApplication.get/set/delete/getAttributeNames
    │   │   ├── system_service_db_admin.go       ← handlers: DBAdmin.create/deleteUser, create/deleteApp, ban/revokeBan
    │   │   ├── dispatcher.go         ← central routing by first recipient
    │   │   ├── sender.go             ← direct send (user-to-user) and broadcast (group)
    │   │   ├── movie.go              ← MovieManager — manages movies and groups
    │   │   ├── group.go              ← Group — membership and broadcast within movies
    │   │   └── response.go           ← helpers for building SMUS responses
    │   ├── tcp_server.go             ← TCP server, delegates connections to ConnPool
    │   ├── conn_pool.go              ← connection pool with per-conn write mutex
    │   ├── smus_handler.go           ← parses SMUS messages, delegates routing to Dispatcher
    │   └── console.go                ← interactive CLI (create user, etc.)
    └── outbound/                     ← OUTBOUND adapters
        ├── blowfish.go               ← Blowfish cryptography implementation
        ├── file_logger.go            ← file logger implementation
        ├── lua_script_engine.go      ← Lua script execution (gopher-lua)
        ├── lua_db_module.go          ← mus.db module for Lua (query builder + DB ops + bcrypt)
        ├── lua_server_module.go      ← mus.server module for Lua
        ├── sql_db.go                 ← storage core: users, bans, attributes, schema DSL — written once, dialect-agnostic
        ├── sql_dialect.go            ← dialect seam: placeholders, column types, now-expressions, DDL quirks
        ├── sql_dialect_sqlite.go     ← SQLite dialect
        ├── sql_dialect_postgres.go   ← Postgres dialect
        ├── sql_query_builder.go      ← fluent query builder + transactions (dialect-parameterized)
        ├── sqlite_db.go              ← SQLite constructor (file path handling; pragmas via the dialect)
        ├── postgres_db.go            ← Postgres constructor (DSN validation, fail-fast ping)
        ├── memory_queue.go           ← in-memory message queue (dev)
        ├── redis_queue.go            ← message queue via Redis pub/sub
        ├── rabbitmq_queue.go         ← message queue via RabbitMQ (AMQP)
        ├── queue_errors.go           ← shared errors for the queue adapters
        ├── memory_session_store.go   ← in-memory session store (dev)
        └── redis_session_store.go    ← session store via Redis (production)

external/
├── migrations/                       ← versioned SQL migrations
│   └── 00000000000000_initial_schema.go
├── queues/                           ← registry of queue consumers
│   └── registry.go                   ← topic→handler list for bootstrap
└── scripts/                          ← server-side Lua scripts
    └── echo.lua                      ← example script
```

---

## What each thing is

### Config

The `config/` package centralizes reading environment variables into a `ServerConfig` struct. It uses `os.LookupEnv` with default fallbacks for each variable. `main.go` calls `config.LoadServerConfig()` once and passes the values to the factories. The defaults are based on the original `Multiuser.cfg` from Shockwave Multiuser Server 3.0 (e.g. `DEFAULT_USER_LEVEL=20`, `MAX_MESSAGE_SIZE=2097151`, `TCP_NO_DELAY=1`).

### Factory

The `factory/` package contains constructor functions (`NewCipher`, `NewHandler`, `NewLogger`, `NewDatabase`, `NewSessionStore`, `NewScriptEngine`, `NewMessageQueue`) that receive a type (a string coming from the config) and return the corresponding domain interface. This isolates `main.go` from knowing the concrete implementations directly — it only needs to know the desired type.

### Domain

This is the **heart** of the system. Here live the rules and structures that define what MUSGoS **is**. In our case:

- **`types/lingo/`** — the Lingo language data types (strings, integers, lists, prop-lists, etc.). These types exist independently of how the data arrived or where it's going.

- **`types/smus/`** — the structure of a MUS message (`MUSMessage`). It knows how to parse the raw bytes into fields (subject, sender, recipients, content). When it needs to decrypt, it **doesn't know what Blowfish is** — it just asks for a `ports.Cipher` and calls `.Decrypt()`.

- **`ports/`** — the **interfaces** that the domain exposes. These are the "contracts" that say: *"I need someone who does X, I don't care how"*.

### Ports

Port is just a fancy name for **interface**. They are the connection points between the domain and the outside world.

We have eleven:

#### `SessionStore` (outbound port)
```go
type SessionStore interface {
    RegisterConnection(clientID, ip string) error
    UnregisterConnection(clientID string) error
    GetConnection(clientID string) (*ConnectionInfo, error)
    GetAllConnections() ([]ConnectionInfo, error)
    IsConnected(clientID string) (bool, error)
    SetUserAttribute(clientID, attrName string, value lingo.LValue) error
    GetUserAttribute(clientID, attrName string) (lingo.LValue, error)
    GetUserAttributeNames(clientID string) ([]string, error)
    DeleteUserAttribute(clientID, attrName string) error
    JoinRoom(roomName, clientID string) error
    LeaveRoom(roomName, clientID string) error
    GetRoomMembers(roomName string) ([]string, error)
    GetClientRooms(clientID string) ([]string, error)
    LeaveAllRooms(clientID string) error
    Close() error
}
```
Centralizes management of active connections, ephemeral session attributes (per clientID), and room/group membership. Two implementations: `MemorySessionStore` (in-memory, the default for development) and `RedisSessionStore` (for production, allowing multiple instances to share state). When a client disconnects (`UnregisterConnection`), all associated data (connection, attributes, rooms) is cleaned up automatically.

#### `ConnectionWriter` (outbound port)
```go
type ConnectionWriter interface {
    WriteToClient(clientID string, data []byte) error
    RemapClientID(oldID, newID string)
    DisconnectClient(clientID string) error
}
```
Abstraction for writing to network connections. `WriteToClient` sends bytes to a client by its ID. `RemapClientID` allows swapping a connection's ID (used during Logon, when the IP is replaced by the userID). `DisconnectClient` closes the TCP connection (used by `system.user.delete`). Implemented by `ConnPool`.

#### `MessageSender` (outbound port)
```go
type MessageSender interface {
    SendMessage(senderID, recipientID, subject string, content lingo.LValue) error
}
```
Contract for sending MUS messages. It routes automatically: if the recipientID begins with `@`, it broadcasts to the group; otherwise, it sends directly to the user. Implemented by `Sender`. Used by `LuaScriptEngine` for `mus.sendMessage()`.

#### `Cipher` (outbound port)
```go
type Cipher interface {
    Encrypt(data []byte) []byte
    Decrypt(data []byte) []byte
}
```
The domain (`mus_message.go`) needs to decrypt content. Instead of importing Blowfish directly, it receives anything that implements `Cipher`. Today it's Blowfish, tomorrow it could be AES, or a mock in a test.

#### `MessageHandler` (inbound port)
```go
type MessageHandler interface {
    HandleRawMessage(clientID string, data []byte) ([]byte, error)
}
```
The TCP server needs someone to process the bytes arriving over the network. Instead of knowing `SMUSHandler` directly, it receives anything that implements `MessageHandler`. Today it's SMUS, but it could be another protocol.

#### `Logger` (outbound port)
```go
type Logger interface {
    Debug(msg string, fields ...map[string]interface{})
    Info(msg string, fields ...map[string]interface{})
    Warn(msg string, fields ...map[string]interface{})
    Error(msg string, fields ...map[string]interface{})
    Fatal(msg string, fields ...map[string]interface{})
}
```
Any component that needs to log receives a `ports.Logger`. Today it's `FileLogger` (writes to a file), but it could be a logger to stdout, to an external service, or a mock in tests.

#### `DBAdapter` (outbound port)
```go
type DBAdapter interface {
    CreateUser(username, passwordHash string, userLevel int) error
    GetUserByUsername(username string) (*User, error)
    AuthenticateUser(username, password string) (*User, error)
    CreateBan(userID, ip, reason string, expiresAt *time.Time) error
    GetActiveBanByUserID(userID string) (*Ban, error)
    // ... app attributes, player attributes, schema operations
    Close() error
}
```
Complete persistence interface. It manages users (creation, authentication with bcrypt), bans (by user/IP, temporary or permanent), application and player attributes (stored as LValue via JSON), and schema operations for migrations. Implemented via SQLite (`sqlite_db.go`).

#### `QueryBuilder` + `Query` (outbound port)
```go
type QueryBuilder interface {
    Table(name string) Query
}
type Query interface {
    Where(column string, value interface{}) Query
    Insert(data map[string]interface{}) error
    Update(data map[string]interface{}) (int64, error)
    Delete() (int64, error)
    First() (QueryResult, error)
    Get() ([]QueryResult, error)
    Count() (int64, error)
}
```
Fluent interface for generic operations on tables. Exposed to Lua scripts via `mus.db.table("name")`, allowing arbitrary queries on custom tables without needing specific methods on `DBAdapter`. Implemented by `SQLiteQueryBuilder` with identifier validation via a whitelist regex.

#### `Migration` + `MigrationTracker` (outbound ports)
```go
type Migration interface {
    ID() string
    Up(adapter DBAdapter) error
}

type MigrationTracker interface {
    HasRun(migrationID string) (bool, error)
    MarkAsRun(migrationID string) error
}
```
Contracts for the migration system. Each migration has an orderable ID (timestamp) and an `Up` method. The `MigrationTracker` (implemented by `DBAdapter` itself) records which ones have already been executed. The `MigrationRunner` (service) orchestrates execution in order.

#### `MessageQueue` (outbound port)
```go
type QueuePublisher interface {
    Publish(topic string, payload []byte) error
    Close() error
}

type QueueConsumer interface {
    Subscribe(topic string, handler QueueSubscriber) error
    Unsubscribe(topic string) error
    Close() error
}

type MessageQueue interface {
    QueuePublisher
    QueueConsumer
}
```
Generic pub/sub messaging system with three implementations: `MemoryQueue` (in-memory, for dev/tests), `RedisQueue` (via Redis pub/sub), and `RabbitMQQueue` (via AMQP). The `ScriptEngine` receives only a `QueuePublisher` to publish messages via `mus.publish()` in Lua scripts. Consumers are registered at bootstrap via `external/queues/registry.go`.

#### `ScriptEngine` (outbound port)
```go
type ScriptEngine interface {
    HasScript(subject string) bool
    Execute(msg *ScriptMessage) (*ScriptResult, error)
}
```
Server-side script execution. When a message arrives, the handler checks whether a script exists for that subject and executes it. The interface is protocol-agnostic — it receives a generic `ScriptMessage` (Subject, SenderID, Content). Implemented via gopher-lua with a sandboxed VM (no access to `os`, `io`, `debug`).

### Adapters

Adapters are the **concrete implementations** that connect the domain to the real world. There are two kinds:

#### Inbound — "data coming into the system"

These are the adapters that **receive** data from the outside world and deliver it to the domain.

- **`tcp_server.go`** — opens a TCP port, accepts connections, reads bytes from the network. It doesn't manage connections directly — it delegates to `ConnPool`. When it receives data, it passes it to the `MessageHandler` (which it knows only through the interface). After `HandleRawMessage`, it re-fetches the connection's current ID from the pool (it may have been remapped during Logon). Configurable via `TCPServerConfig` (bind address, buffer size, TCP_NODELAY). Supports graceful shutdown. Receives the handler in the constructor (no `SetHandler`).

- **`conn_pool.go`** — TCP connection pool with bidirectional clientID↔conn mapping and a per-conn write mutex for thread safety. Operations: `Register`, `Unregister`, `CurrentID`, `WriteToClient`, `RemapClientID`, `CloseAll`. Implements `ports.ConnectionWriter`.

- **`smus_handler.go`** — receives the raw bytes from the TCP server and uses the domain (`smus.ParseMUSMessageWithDecryption`) to interpret the message. It delegates all routing logic to the `Dispatcher`. It's inbound because it's on the "receive and process" side of the request.

- **`mus/`** — sub-package with MUS-protocol-specific logic:
  - **`system_service.go`** — `SystemService` with a handler map (`map[string]handlerFunc`) for routing commands by subject. It includes logon with 3 modes (`none`/`open`/`strict`), a deny-by-default permission system (`checkCommandLevel` via `commandLevels` map), a generic helper `handleDBCommand` for DB commands (parse proplist + extract fields + execute + error mapping), and a `#movieID` cache in the session for O(1) lookup. `dbErrorCode` maps domain errors (`ErrUserNotFound`, `ErrBanNotFound`) to MUS protocol codes using `errors.Is`.
  - **`system_service_*.go`** — handlers organized by domain: `_server` (version, time, counts), `_movie` (movie users/groups), `_group` (join/leave/attributes), `_user` (address, groups, delete with session cleanup), `_db_player`/`_db_application`/`_db_admin` (DB operations via `handleDBCommand`).
  - **`dispatcher.go`** — central routing by first recipient: `System` → SystemService, `system.script` → ScriptEngine, `@Group` → Sender broadcast, `userName` → Sender direct.
  - **`sender.go`** — message sending. `SendMessage()` routes: groups (`@`) via `deliverToGroup()` (serializes once, delivers to all members), user-to-user via `ConnectionWriter.WriteToClient()`. Implements `ports.MessageSender`.
  - **`response.go`** — helpers for building SMUS responses (`NewResponse`), used by the handler and services.

- **`console.go`** — interactive CLI for server administration. Supports commands like `create user <username> <password>`. Uses bcrypt for password hashing. Accesses `DBAdapter` directly.

#### Outbound — "the system accessing external resources"

These are the adapters that the domain **uses** to do things it doesn't know (or doesn't want to know) how to do.

- **`blowfish.go`** — the concrete Blowfish cryptography implementation. Implements the `ports.Cipher` interface. It's outbound because it's a **resource** the domain consumes.

- **`file_logger.go`** — the concrete file logger implementation. Implements the `ports.Logger` interface. Writes formatted logs to a file and to stdout. It's outbound because logging is an **infrastructure resource** the system consumes.

- **`sql_db.go` + `sql_dialect*.go`** — one storage core implements `ports.DBAdapter` and `ports.MigrationTracker` for every SQL backend (RFC-007). All persistence logic — users (with bcrypt), bans (by user/IP, with expiration), application and player attributes (LValues serialized as JSON), and schema operations for the migration system — is written once with `?`-placeholders; an internal `dialect` seam supplies the per-backend differences (placeholder rebinding, column types, now-expressions, DDL quirks). `sqlite_db.go` and `postgres_db.go` are thin constructors that pair a connection pool with their dialect. Adding a backend means writing a new dialect, not a new adapter.

- **`memory_session_store.go`** — in-memory session store with `sync.RWMutex`. Implements `ports.SessionStore`. Ideal for local development — no external dependencies, but no persistence across restarts.

- **`redis_session_store.go`** — session store via Redis. Implements `ports.SessionStore`. Manages connections, ephemeral attributes, and rooms using Redis structures (HASH, SET) with key prefixing and TTL. For production and multi-instance scenarios.

- **`lua_script_engine.go`** — implementation of `ports.ScriptEngine` via gopher-lua. Creates a fresh Lua VM per execution, with unsafe libs removed (`os`, `io`, `debug`). Registers the `mus` module with `getSender()`, `getContent()`, `response()`, `publish()`, and `sendMessage()`. Scripts live in `external/scripts/` with a 1:1 mapping by subject. Execution timeout configurable via `SCRIPT_TIMEOUT`.

- **`lua_db_module.go`** — `mus.db` module for Lua scripts. Exposes DBPlayer, DBApplication, and DBAdmin operations (with bcrypt in `createUser`), plus the fluent query builder (`mus.db.table("name"):where(...):get()`).

- **`lua_server_module.go`** — `mus.server` module for Lua scripts with server information.

- **`sql_query_builder.go`** — one implementation of `ports.QueryBuilder`, `ports.Query`, and `ports.Tx` for every backend: statements are built with `?`-placeholders (parameterized, identifiers validated via a whitelist regex) and rebound through the dialect at execution time.

- **`memory_queue.go`** — in-memory message queue with `sync.RWMutex`. Implements `ports.MessageQueue`. Ideal for development and tests — no external dependencies.

- **`redis_queue.go`** — message queue via Redis pub/sub. Implements `ports.MessageQueue`. Uses Redis `PUBLISH`/`SUBSCRIBE` to distribute messages across instances.

- **`rabbitmq_queue.go`** — message queue via RabbitMQ (AMQP 0-9-1). Implements `ports.MessageQueue`. Uses topic exchanges for flexible message routing. Each subscription creates a dedicated channel with an anonymous, auto-delete queue.

---

## How it all connects

The "glue" happens in `main.go` with help from `config` and the `factories`. The config loads the environment variables, and the factories create the concrete instances:

```go
cfg := config.LoadServerConfig()

// factory creates the logger (outbound) based on the configured type
gameLogger, _ := factory.NewLogger(cfg.LoggerType, cfg.ApplicationName, ...)

// factory creates the database (outbound) + migration runner
dbResult, _ := factory.NewDatabase(cfg.DatabaseType, cfg.DatabasePath, migrations.All)
dbResult.MigrationRunner.RunPending()

// factory creates the cipher (outbound) based on the configured type
cipher, _ := factory.NewCipher(cfg.CipherType, cfg.EncryptionKey)

// factory creates the session store (outbound) based on the configured type
sessionStore, _ := factory.NewSessionStore(cfg.SessionStoreType, cfg.Redis)

// factory creates the message queue (outbound) based on the configured type
queue, _ := factory.NewMessageQueue(cfg.QueueType, cfg.QueueRedis, cfg.RabbitMQ)

// 1. ConnPool — standalone, no dependencies
pool := inbound.NewConnPool()

// 2. Sender — uses pool as ConnectionWriter
sender := mus.NewSender(pool, sessionStore, gameLogger)

// 3. ScriptEngine — can send messages via Sender
scriptEngine := factory.NewScriptEngine(cfg.ScriptsPath, gameLogger, cfg.ScriptTimeout, queue, sender)

// 4. Handler — Dispatcher receives ScriptEngine + Sender + pool
handler, _ := factory.NewHandler(cfg.Protocol, gameLogger, cipher, scriptEngine,
    dbResult.Adapter, sessionStore, queue, pool, sender, cfg.AuthMode, cfg.DefaultUserLevel)

// 5. TCPServer — fully constructed, no SetHandler
server := inbound.NewTCPServer(inbound.TCPServerConfig{
    Port: cfg.Port, ServerIP: cfg.ServerIP,
    MaxMessageSize: cfg.MaxMessageSize, TCPNoDelay: cfg.TCPNoDelay,
}, handler, pool, gameLogger, sessionStore)

// interactive console for administration (uses DefaultUserLevel when creating users)
console := inbound.NewConsole(dbResult.Adapter, gameLogger, os.Stdin, cfg.DefaultUserLevel)
```

Notice that:
- `main.go` does not import `outbound` directly — the factories do that.
- Each factory returns the domain **interface** (`ports.Logger`, `ports.Cipher`, `ports.MessageHandler`, `ports.ScriptEngine`), never the concrete type.
- Components know each other only through the contracts, never through the concrete types.

This is **dependency inversion**: the adapters are the ones that depend (on the domain), never the other way around.

---

## The dependency rule

The golden rule is that `import` arrows **always point inward**:

```
adapters/inbound   ──imports──►  domain/ports
adapters/inbound   ──imports──►  domain/types
adapters/outbound  ──imports──►  domain/ports
domain/types/smus  ──imports──►  domain/ports
domain/types/smus  ──imports──►  domain/types/lingo
config/            ──imports──►  (nothing from domain, only stdlib)
factory/           ──imports──►  adapters + domain/ports (resolves implementations)
cmd/main.go        ──imports──►  config + factory + adapters/inbound (is the glue)
```

The domain **never** imports adapters, config, or factory. Adapters import the domain. The factories import adapters and ports to assemble the pieces. `main.go` imports config, factory, inbound adapters (TCP server, console), and the external migrations.

---

## Quick summary

| Concept | What it is | In MUSGoS |
|---|---|---|
| **Domain** | The system's core logic and types | `types/lingo/`, `types/smus/`, `services/` |
| **Port** | Interface that defines a contract | `Cipher`, `ConnectionWriter`, `MessageHandler`, `MessageSender`, `Logger`, `DBAdapter`, `QueryBuilder`, `SessionStore`, `ScriptEngine`, `MessageQueue`, `Migration` |
| **Inbound Adapter** | Receives data from the outside world | `TCPServer`, `ConnPool`, `SMUSHandler`, `Dispatcher`, `Sender`, `SystemService`, `Console`, `MovieManager`, `Group` |
| **Outbound Adapter** | Provides capabilities to the domain | `Blowfish`, `FileLogger`, `SQLiteDB`, `MemorySessionStore`, `RedisSessionStore`, `LuaScriptEngine`, `MemoryQueue`, `RedisQueue`, `RabbitMQQueue` |
| **Config** | Loads environment variables | `ServerConfig`, `LoadServerConfig()` |
| **Factory** | Creates concrete implementations by type | `NewCipher()`, `NewHandler()`, `NewLogger()`, `NewDatabase()`, `NewSessionStore()`, `NewScriptEngine()`, `NewMessageQueue()` |
| **main.go** | Glues everything together using config + factories | Manual dependency injection |

---

## Services Layer

In traditional hexagonal architectures there is an "Application Service" layer with use cases like `LoginUseCase`, `SendMessageUseCase`, etc.

MUS servers are **script-driven**: the Shockwave/Director client sends Lingo scripts, and the server reacts. However, there are standard MUS protocol messages (such as `Logon`, `Logoff`, `joinGroup`, `leaveGroup`) that involve real business logic — authentication, room management, state persistence.

Currently the `domain/services/` layer contains:

- **`MigrationRunner`** — orchestrates the execution of pending migrations in order.

MUS-protocol-specific logic (such as `LogonService` and response helpers) lives in `adapters/inbound/mus/`, since it depends directly on the SMUS types and is not pure domain logic.

MUS-protocol-specific logic (`Dispatcher`, `Sender`, `SystemService`, `MovieManager`, `GroupManager`) lives in `adapters/inbound/mus/`, since it depends directly on the SMUS types. Future protocol-agnostic domain services will be added under `domain/services/`.
