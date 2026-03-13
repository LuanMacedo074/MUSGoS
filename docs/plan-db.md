# Database Adapter + Infraestrutura de Resposta

## Contexto

O servidor hoje parseia mensagens SMUS mas nunca responde ao cliente (`HandleRawMessage` retorna `nil, nil`). Para suportar operacoes de banco de dados, precisamos de 3 coisas: serializar mensagens de resposta, rotear comandos por subject, e o adapter de banco em si. Comecar com backend in-memory (zero dependencias externas).

## Etapa 1 — Serializar tipos Lingo (`GetBytes()`)

Os seguintes tipos nao implementam `GetBytes()` (herdam do `BaseLValue` que retorna `[]byte{}`):

| Arquivo | Mudanca |
|---------|---------|
| `internal/types/lingo/l_void.go` | `GetBytes()` → retorna `[2 bytes tipo]` |
| `internal/types/lingo/l_integer.go` | `GetBytes()` → retorna `[2 tipo][4 valor]` |
| `internal/types/lingo/l_string.go` | `GetBytes()` → retorna `[2 tipo][4 len][string][padding]` |
| `internal/types/lingo/l_symbol.go` | `GetBytes()` → retorna `[2 tipo][4 len][string][padding]` |
| `internal/types/lingo/l_float.go` | `GetBytes()` → retorna `[2 tipo][8 float64]` |
| `internal/types/lingo/l_list.go` | `GetBytes()` → retorna `[2 tipo][4 count][elements...]` |

`LPropList` ja tem `GetBytes()` implementado (l_prop_list.go:106). Ele chama `GetBytes()` nos filhos, entao precisa que os tipos acima funcionem primeiro.

Tambem exportar `addElement` → `AddElement` em `l_prop_list.go:23` para construir PropLists de resposta.

## Etapa 2 — Serializar MUSMessage

Adicionar metodos de escrita (inverso do parsing que ja existe):

| Arquivo | Mudanca |
|---------|---------|
| `internal/types/smus/mus_msg_header_string.go` | Adicionar `WriteBytes(buf *bytes.Buffer)` |
| `internal/types/smus/mus_msg_header_string_list.go` | Adicionar `WriteBytes(buf *bytes.Buffer)` |
| `internal/types/smus/mus_message.go` | Adicionar `GetBytes() []byte` — monta: `[header 0x7200][4 contentSize][4 errCode][4 timestamp][subject][senderID][recptID][content]` |

## Etapa 3 — Router de comandos

Novo arquivo: `internal/handlers/router.go`

```go
type CommandHandler func(clientID string, msg *smus.MUSMessage) (*smus.MUSMessage, error)

type Router struct {
    routes map[string]CommandHandler
}
```

- `Register(subject, handler)` — registra handler por subject exato
- `Route(subject)` — busca handler por subject (map lookup O(1))

Modificar `internal/handlers/smus_handler.go`:
- Adicionar campo `router *Router` na struct
- Alterar `NewSMUSHandler` para receber `*Router`
- No `HandleRawMessage`, apos parsear a mensagem, chamar `router.Route(msg.Subject.Value)` e retornar `response.GetBytes()`

Novo arquivo: `internal/handlers/response.go`
- `BuildResponse(req, errCode, content)` — constroi MUSMessage de resposta (swap sender/recipient, subject mantido, sender = "system")
- `BuildErrorResponse(req, err)` — resposta de erro

## Etapa 4 — Interface do database adapter

Novo arquivo: `internal/database/database.go`

```go
type DBAdapter interface {
    // DBAdmin
    CreateApplication(appName string) error
    DeleteApplication(appName string) error

    // DBApplication (dados globais da app)
    SetApplicationAttribute(appName, attrName string, value lingo.LValue) error
    GetApplicationAttribute(appName, attrName string) (lingo.LValue, error)
    GetApplicationAttributeNames(appName string) ([]string, error)
    DeleteApplicationAttribute(appName, attrName string) error

    // DBPlayer (persistente por userID)
    SetPlayerAttribute(appName, userID, attrName string, value lingo.LValue) error
    GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error)
    GetPlayerAttributeNames(appName, userID string) ([]string, error)
    DeletePlayerAttribute(appName, userID, attrName string) error

    // DBUser (sessao, por clientID)
    SetUserAttribute(clientID, attrName string, value lingo.LValue) error
    GetUserAttribute(clientID, attrName string) (lingo.LValue, error)
    GetUserAttributeNames(clientID string) ([]string, error)
    DeleteUserAttribute(clientID, attrName string) error

    Close() error
}
```

Valores armazenados diretamente como `lingo.LValue` — sem conversao intermediaria.

## Etapa 5 — Implementacao in-memory

Novo arquivo: `internal/database/memory.go`

```go
type MemoryDB struct {
    mu           sync.RWMutex
    applications map[string]map[string]lingo.LValue              // app → attr → value
    players      map[string]map[string]map[string]lingo.LValue   // app → userID → attr → value
    users        map[string]map[string]lingo.LValue              // clientID → attr → value
}
```

Thread-safe com `sync.RWMutex`. Dados nao sobrevivem restart (futuro: SQLite adapter).

## Etapa 6 — Command handlers

Cada grupo de comandos em seu proprio arquivo, registrado via `Register*Handlers(router, db, logger)`:

| Arquivo | Subjects |
|---------|----------|
| `internal/handlers/db_player.go` | `system.DBPlayer.setAttribute`, `.getAttribute`, `.getAttributeNames`, `.deleteAttribute` |
| `internal/handlers/db_application.go` | `system.DBApplication.setAttribute`, `.getAttribute`, `.getAttributeNames`, `.deleteAttribute` |
| `internal/handlers/db_user.go` | `system.DBUser.setAttribute`, `.getAttribute`, `.getAttributeNames`, `.deleteAttribute` |
| `internal/handlers/db_admin.go` | `system.DBAdmin.createApplication`, `.deleteApplication` |

Padrao de cada handler:
1. Extrair dados do `msg.MsgContent` (LPropList para set, LList/LString para get)
2. Chamar metodo do `DBAdapter`
3. Construir resposta com `BuildResponse()`

## Etapa 7 — Wiring no main.go

```go
db := database.NewMemoryDB()
router := handlers.NewRouter()
handlers.RegisterDBPlayerHandlers(router, db, gameLogger)
handlers.RegisterDBApplicationHandlers(router, db, gameLogger)
handlers.RegisterDBUserHandlers(router, db, gameLogger)
handlers.RegisterDBAdminHandlers(router, db, gameLogger)
smusHandler := handlers.NewSMUSHandler(gameLogger, blowfish, router)
```

## Arquivos novos (6)

- `internal/database/database.go`
- `internal/database/memory.go`
- `internal/handlers/router.go`
- `internal/handlers/response.go`
- `internal/handlers/db_player.go`
- `internal/handlers/db_application.go`
- `internal/handlers/db_user.go`
- `internal/handlers/db_admin.go`

## Arquivos modificados (9)

- `internal/types/lingo/l_void.go` — GetBytes()
- `internal/types/lingo/l_integer.go` — GetBytes()
- `internal/types/lingo/l_string.go` — GetBytes()
- `internal/types/lingo/l_symbol.go` — GetBytes()
- `internal/types/lingo/l_float.go` — GetBytes()
- `internal/types/lingo/l_list.go` — GetBytes()
- `internal/types/lingo/l_prop_list.go` — exportar AddElement
- `internal/types/smus/mus_msg_header_string.go` — WriteBytes()
- `internal/types/smus/mus_msg_header_string_list.go` — WriteBytes()
- `internal/types/smus/mus_message.go` — GetBytes()
- `internal/handlers/smus_handler.go` — integrar router
- `cmd/gameserver/main.go` — wiring

## Verificacao

1. `go build ./...` — compilacao sem erros
2. Teste manual: enviar mensagem SMUS com subject `system.DBPlayer.setAttribute` e verificar que o servidor responde com bytes validos
3. Futuro: testes unitarios para GetBytes() round-trip (serialize → parse → comparar) e para cada comando DB
