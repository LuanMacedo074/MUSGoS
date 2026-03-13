# Arquitetura Hexagonal — MUSGoS

## O que é Arquitetura Hexagonal?

A ideia central é simples: **o domínio (a lógica do seu sistema) não depende de nada externo**. Ele não sabe se a conexão vem por TCP, HTTP ou um teste unitário. Ele não sabe se a criptografia é Blowfish, AES ou um mock. Ele só conhece **contratos** (interfaces).

Isso é feito separando o código em três camadas:

```
                    ┌─────────────────────────────┐
  mundo externo     │       ADAPTERS (inbound)     │  ← recebe dados do mundo externo
  (clients TCP) ──► │   tcp_server, smus_handler   │
                    └──────────────┬───────────────┘
                                   │ implementa / usa
                    ┌──────────────▼───────────────┐
                    │           DOMAIN              │  ← núcleo do sistema
                    │   types (lingo, smus)         │
                    │   ports (interfaces)          │
                    └──────────────┬───────────────┘
                                   │ define contratos
                    ┌──────────────▼───────────────┐
  mundo externo     │      ADAPTERS (outbound)      │  ← fornece capacidades ao domínio
  (cripto, logs) ◄──│     blowfish, file_logger     │
                    └─────────────────────────────┘
```

---

## Estrutura de pastas

```
internal/
├── config/                           ← carrega variáveis de ambiente
│   └── config.go                     ← struct ServerConfig + LoadServerConfig()
│
├── factory/                          ← resolve qual implementação concreta usar
│   ├── cipher.go                     ← NewCipher() — escolhe o cipher pelo tipo
│   ├── database.go                   ← NewDatabase() — cria DB + migration runner
│   ├── handler.go                    ← NewHandler() — escolhe o handler pelo protocolo
│   ├── logger.go                     ← NewLogger() — escolhe o logger pelo tipo
│   ├── script_engine.go              ← NewScriptEngine() — cria o engine Lua
│   └── session_store.go              ← NewSessionStore() — memory ou Redis
│
├── domain/                           ← núcleo, não depende de nada externo
│   ├── types/
│   │   ├── lingo/                    ← tipos do Lingo (LValue, LString, LInteger, etc.)
│   │   │   ├── codec.go             ← JSON marshal/unmarshal de LValues
│   │   │   └── lua_convert.go       ← conversão bidirecional LValue ↔ Lua
│   │   └── smus/                     ← tipos do protocolo SMUS (MUSMessage, headers)
│   │       └── mus_error_code.go     ← ~54 constantes de erro do protocolo MUS
│   ├── ports/                        ← interfaces (contratos)
│   │   ├── cipher.go                 ← interface Cipher
│   │   ├── database.go               ← interface DBAdapter (users, bans, atributos)
│   │   ├── handler.go                ← interface MessageHandler
│   │   ├── logger.go                 ← interface Logger + LogLevel
│   │   ├── migration.go              ← interfaces Migration + MigrationTracker
│   │   ├── schema.go                 ← DSL para definição de tabelas/índices
│   │   ├── script_engine.go          ← interface ScriptEngine
│   │   └── session_store.go          ← interface SessionStore
│   └── services/
│       └── migration_runner.go       ← executa migrations pendentes em ordem
│
└── adapters/                         ← implementações concretas
    ├── inbound/                      ← adaptadores de ENTRADA
    │   ├── mus/                      ← lógica específica do protocolo MUS
    │   │   ├── logon.go              ← autenticação com 3 modos (none/open/strict)
    │   │   └── response.go           ← helpers para construção de respostas SMUS
    │   ├── tcp_server.go             ← servidor TCP que aceita conexões
    │   ├── smus_handler.go           ← processa mensagens SMUS, dispara scripts
    │   └── console.go                ← CLI interativo (create user, etc.)
    └── outbound/                     ← adaptadores de SAÍDA
        ├── blowfish.go               ← implementação da criptografia Blowfish
        ├── file_logger.go            ← implementação do logger em arquivo
        ├── lua_script_engine.go      ← execução de scripts Lua (gopher-lua)
        ├── memory_session_store.go   ← session store in-memory (dev)
        ├── redis_session_store.go    ← session store via Redis (produção)
        └── sqlite_db.go             ← SQLite: users, bans, atributos, schema DSL

external/
├── migrations/                       ← migrations SQL versionadas
│   └── 00000000000000_initial_schema.go
└── scripts/                          ← scripts Lua server-side
    └── echo.lua                      ← script exemplo
```

---

## O que é cada coisa

### Config

O pacote `config/` centraliza a leitura de variáveis de ambiente numa struct `ServerConfig`. Ele usa `os.LookupEnv` com fallbacks padrão para cada variável. O `main.go` chama `config.LoadServerConfig()` uma vez e passa os valores para as factories. Os defaults são baseados no `Multiuser.cfg` original do Shockwave Multiuser Server 3.0 (ex: `DEFAULT_USER_LEVEL=20`, `MAX_MESSAGE_SIZE=2097151`, `TCP_NO_DELAY=1`).

### Factory

O pacote `factory/` contém funções construtoras (`NewCipher`, `NewHandler`, `NewLogger`, `NewDatabase`, `NewSessionStore`, `NewScriptEngine`) que recebem um tipo (string vinda do config) e retornam a interface correspondente do domínio. Isso isola o `main.go` de conhecer as implementações concretas diretamente — ele só precisa saber o tipo desejado.

### Domain (Domínio)

É o **coração** do sistema. Aqui ficam as regras e estruturas que definem o que o MUSGoS **é**. No nosso caso:

- **`types/lingo/`** — os tipos de dados da linguagem Lingo (strings, inteiros, listas, prop-lists, etc.). Esses tipos existem independente de como o dado chegou ou para onde vai.

- **`types/smus/`** — a estrutura de uma mensagem MUS (`MUSMessage`). Sabe fazer o parsing dos bytes brutos em campos (subject, sender, recipients, conteúdo). Quando precisa descriptografar, ele **não sabe o que é Blowfish** — ele só pede um `ports.Cipher` e chama `.Decrypt()`.

- **`ports/`** — as **interfaces** que o domínio expõe. São os "contratos" que dizem: *"eu preciso de alguém que faça X, não me importa como"*.

### Ports (Portas)

Porta é só um nome bonito para **interface**. São os pontos de conexão entre o domínio e o mundo externo.

Temos sete:

#### `SessionStore` (porta outbound)
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
Centraliza o gerenciamento de conexões ativas, atributos efêmeros de sessão (por clientID) e membership de rooms/groups. Duas implementações: `MemorySessionStore` (in-memory, padrão para desenvolvimento) e `RedisSessionStore` (para produção, permite múltiplas instâncias compartilharem estado). Quando um cliente desconecta (`UnregisterConnection`), todos os dados associados (conexão, atributos, rooms) são limpos automaticamente.

#### `Cipher` (porta outbound)
```go
type Cipher interface {
    Encrypt(data []byte) []byte
    Decrypt(data []byte) []byte
}
```
O domínio (`mus_message.go`) precisa descriptografar conteúdo. Em vez de importar o Blowfish diretamente, ele recebe qualquer coisa que implemente `Cipher`. Hoje é Blowfish, amanhã poderia ser AES, ou um mock no teste.

#### `MessageHandler` (porta inbound)
```go
type MessageHandler interface {
    HandleRawMessage(clientID string, data []byte) ([]byte, error)
}
```
O TCP server precisa de alguém que processe os bytes que chegam pela rede. Em vez de conhecer o `SMUSHandler` diretamente, ele recebe qualquer coisa que implemente `MessageHandler`. Hoje é SMUS, mas poderia ser outro protocolo.

#### `Logger` (porta outbound)
```go
type Logger interface {
    Debug(msg string, fields ...map[string]interface{})
    Info(msg string, fields ...map[string]interface{})
    Warn(msg string, fields ...map[string]interface{})
    Error(msg string, fields ...map[string]interface{})
    Fatal(msg string, fields ...map[string]interface{})
}
```
Qualquer componente que precisa logar recebe um `ports.Logger`. Hoje é `FileLogger` (escreve em arquivo), mas poderia ser um logger para stdout, para um serviço externo, ou um mock nos testes.

#### `DBAdapter` (porta outbound)
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
Interface completa de persistência. Gerencia usuários (criação, autenticação com bcrypt), bans (por user/IP, temporários ou permanentes), atributos de aplicação e de jogador (armazenados como LValue via JSON), e operações de schema para migrations. Implementada via SQLite (`sqlite_db.go`).

#### `Migration` + `MigrationTracker` (portas outbound)
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
Contratos para o sistema de migrations. Cada migration tem um ID ordenável (timestamp) e um método `Up`. O `MigrationTracker` (implementado pelo próprio `DBAdapter`) registra quais já foram executadas. O `MigrationRunner` (service) orquestra a execução em ordem.

#### `ScriptEngine` (porta outbound)
```go
type ScriptEngine interface {
    HasScript(subject string) bool
    Execute(msg *ScriptMessage) (*ScriptResult, error)
}
```
Execução de scripts server-side. Quando uma mensagem chega, o handler verifica se existe um script para aquele subject e o executa. A interface é agnóstica ao protocolo — recebe um `ScriptMessage` genérico (Subject, SenderID, Content). Implementada via gopher-lua com VM sandboxed (sem acesso a `os`, `io`, `debug`).

### Adapters (Adaptadores)

Adaptadores são as **implementações concretas** que conectam o domínio ao mundo real. Existem dois tipos:

#### Inbound (Entrada) — "dados vindo para dentro do sistema"

São os adaptadores que **recebem** dados do mundo externo e os entregam ao domínio.

- **`tcp_server.go`** — abre uma porta TCP, aceita conexões, lê bytes da rede. Quando recebe dados, repassa para o `MessageHandler` (que ele conhece apenas pela interface). Configurável via `TCPServerConfig` (bind address, buffer size, TCP_NODELAY). Suporta graceful shutdown — ao receber SIGINT/SIGTERM, fecha todas as conexões ativas. É inbound porque é o **ponto de entrada** do sistema: o cliente Shockwave conecta aqui.

- **`smus_handler.go`** — recebe os bytes brutos do TCP server e usa o domínio (`smus.ParseMUSMessageWithDecryption`) para interpretar a mensagem. Roteia mensagens de Logon para o `LogonService`, e mensagens com recipient `system.script` para o `ScriptEngine` (o subject da mensagem é o nome do script). É inbound porque está do lado de "receber e processar" a requisição.

- **`mus/`** — sub-pacote com lógica específica do protocolo MUS:
  - **`logon.go`** — `LogonService` com 3 modos de autenticação configuráveis (`none`, `open`, `strict`). Extrai credenciais de `LList` ou `LPropList`, valida contra o banco, verifica bans. Atribui user level na sessão (`#userLevel`): em modo `strict`/`open` usa o nível do DB quando o usuário existe, senão usa `DEFAULT_USER_LEVEL`.
  - **`response.go`** — helpers para construção de respostas SMUS (`NewResponse`), usados pelo handler e futuros services.

- **`console.go`** — CLI interativo para administração do servidor. Suporta comandos como `create user <username> <password>`. Usa bcrypt para hash de senhas. Acessa o `DBAdapter` diretamente.

#### Outbound (Saída) — "o sistema acessando recursos externos"

São os adaptadores que o domínio **usa** para fazer coisas que ele não sabe (ou não quer saber) como fazer.

- **`blowfish.go`** — a implementação concreta da criptografia Blowfish. Implementa a interface `ports.Cipher`. É outbound porque é um **recurso** que o domínio consome.

- **`file_logger.go`** — a implementação concreta do logger em arquivo. Implementa a interface `ports.Logger`. Escreve logs formatados em arquivo e no stdout. É outbound porque logging é um **recurso de infraestrutura** que o sistema consome.

- **`sqlite_db.go`** — implementação completa do `ports.DBAdapter` via SQLite. Gerencia users (com bcrypt), bans (por user/IP, com expiração), atributos de aplicação e de jogador (LValues serializados como JSON), e operações de schema (CREATE TABLE, CREATE INDEX) para o sistema de migrations. Também implementa `MigrationTracker`.

- **`memory_session_store.go`** — session store in-memory com `sync.RWMutex`. Implementa `ports.SessionStore`. Ideal para desenvolvimento local — sem dependências externas, mas sem persistência entre restarts.

- **`redis_session_store.go`** — session store via Redis. Implementa `ports.SessionStore`. Gerencia conexões, atributos efêmeros e rooms usando estruturas Redis (HASH, SET) com key prefixing e TTL. Para produção e cenários multi-instância.

- **`lua_script_engine.go`** — implementação do `ports.ScriptEngine` via gopher-lua. Cria uma VM Lua fresca por execução, com libs inseguras removidas (`os`, `io`, `debug`). Registra o módulo `mus` com `getSender()`, `getContent()` e `response()`. Scripts ficam em `external/scripts/` com mapping 1:1 por subject. Timeout de execução configurável via `SCRIPT_TIMEOUT`.

---

## Como tudo se conecta

A "cola" acontece no `main.go` com ajuda do `config` e das `factories`. O config carrega as variáveis de ambiente, e as factories criam as instâncias concretas:

```go
cfg := config.LoadServerConfig()

// factory cria o logger (outbound) baseado no tipo configurado
gameLogger, _ := factory.NewLogger(cfg.LoggerType, cfg.ApplicationName, ...)

// factory cria o database (outbound) + migration runner
dbResult, _ := factory.NewDatabase(cfg.DatabaseType, cfg.DatabasePath, migrations.All)
dbResult.MigrationRunner.RunPending()

// factory cria o cipher (outbound) baseado no tipo configurado
cipher, _ := factory.NewCipher(cfg.CipherType, cfg.EncryptionKey)

// factory cria o script engine (outbound) baseado no path e timeout configurados
scriptEngine := factory.NewScriptEngine(cfg.ScriptsPath, gameLogger, cfg.ScriptTimeout)

// factory cria o handler (inbound), injetando dependências + defaultUserLevel
handler, _ := factory.NewHandler(cfg.Protocol, gameLogger, cipher, scriptEngine,
    dbResult.Adapter, sessionStore, cfg.AuthMode, cfg.DefaultUserLevel)

// factory cria o session store (outbound) baseado no tipo configurado
sessionStore, _ := factory.NewSessionStore(cfg.SessionStoreType, cfg.Redis)

// cria o servidor TCP (inbound) com config de rede (IP, buffer, NoDelay)
server := inbound.NewTCPServer(inbound.TCPServerConfig{
    Port: cfg.Port, ServerIP: cfg.ServerIP,
    MaxMessageSize: cfg.MaxMessageSize, TCPNoDelay: cfg.TCPNoDelay,
}, gameLogger, handler, sessionStore)

// console interativo para administração (usa DefaultUserLevel ao criar users)
console := inbound.NewConsole(dbResult.Adapter, gameLogger, os.Stdin, cfg.DefaultUserLevel)
```

Perceba que:
- O `main.go` não importa `outbound` diretamente — as factories fazem isso.
- Cada factory retorna a **interface** do domínio (`ports.Logger`, `ports.Cipher`, `ports.MessageHandler`, `ports.ScriptEngine`), nunca o tipo concreto.
- Os componentes se conhecem apenas pelos contratos, nunca pelos tipos concretos.

Isso é **inversão de dependência**: quem depende são os adaptadores (do domínio), nunca o contrário.

---

## Regra de dependência

A regra de ouro é que as setas de `import` **sempre apontam para dentro**:

```
adapters/inbound   ──importa──►  domain/ports
adapters/inbound   ──importa──►  domain/types
adapters/outbound  ──importa──►  domain/ports
domain/types/smus  ──importa──►  domain/ports
domain/types/smus  ──importa──►  domain/types/lingo
config/            ──importa──►  (nada do domain, só stdlib)
factory/           ──importa──►  adapters + domain/ports (resolve implementações)
cmd/main.go        ──importa──►  config + factory + adapters/inbound (é a cola)
```

O domínio **nunca** importa adapters, config ou factory. Adapters importam o domínio. As factories importam adapters e ports para montar as peças. O `main.go` importa config, factory, adapter inbound (TCP server, console) e as migrations externas.

---

## Resumo rápido

| Conceito | O que é | No MUSGoS |
|---|---|---|
| **Domain** | Lógica e tipos centrais do sistema | `types/lingo/`, `types/smus/`, `services/` |
| **Port** | Interface que define um contrato | `Cipher`, `MessageHandler`, `Logger`, `DBAdapter`, `SessionStore`, `ScriptEngine`, `Migration` |
| **Adapter Inbound** | Recebe dados do mundo externo | `TCPServer`, `SMUSHandler`, `Console` |
| **Adapter Outbound** | Provê capacidades ao domínio | `Blowfish`, `FileLogger`, `SQLiteDB`, `MemorySessionStore`, `RedisSessionStore`, `LuaScriptEngine` |
| **Config** | Carrega variáveis de ambiente | `ServerConfig`, `LoadServerConfig()` |
| **Factory** | Cria implementações concretas pelo tipo | `NewCipher()`, `NewHandler()`, `NewLogger()`, `NewDatabase()`, `NewSessionStore()`, `NewScriptEngine()` |
| **main.go** | Cola tudo usando config + factories | Injeção de dependência manual |

---

## Camada de Services

Em arquiteturas hexagonais tradicionais existe uma camada de "Application Service" com use cases como `LoginUseCase`, `SendMessageUseCase`, etc.

Servidores MUS são **script-driven**: o cliente Shockwave/Director envia scripts Lingo, e o servidor reage. Porém, existem mensagens padrão do protocolo MUS (como `Logon`, `Logoff`, `joinGroup`, `leaveGroup`) que envolvem lógica de negócio real — autenticação, gerenciamento de salas, persistência de estado.

Atualmente a camada `domain/services/` contém:

- **`MigrationRunner`** — orquestra a execução de migrations pendentes em ordem.

Lógica específica do protocolo MUS (como `LogonService` e helpers de resposta) vive em `adapters/inbound/mus/`, pois depende diretamente dos tipos SMUS e não é lógica de domínio pura.

Futuros services de domínio (como `Dispatcher`, `MovieManager`, etc.) serão adicionados em `domain/services/` quando envolverem lógica de negócio agnóstica ao protocolo.
