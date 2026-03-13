# Infraestrutura de Resposta SMUS

## Contexto

O servidor hoje parseia mensagens SMUS mas nunca responde ao cliente (`HandleRawMessage` retorna `nil, nil`). Para fechar o ciclo request/response, precisamos: serializar tipos Lingo, serializar mensagens SMUS, e rotear comandos por subject.

## Etapa 1 — Serializar tipos Lingo (`GetBytes()`)

Os seguintes tipos nao implementam `GetBytes()` (herdam do `BaseLValue` que retorna `[]byte{}`):

| Arquivo | Mudanca |
|---------|---------|
| `internal/domain/types/lingo/l_void.go` | `GetBytes()` → retorna `[2 bytes tipo]` |
| `internal/domain/types/lingo/l_integer.go` | `GetBytes()` → retorna `[2 tipo][4 valor]` |
| `internal/domain/types/lingo/l_string.go` | `GetBytes()` → retorna `[2 tipo][4 len][string][padding]` |
| `internal/domain/types/lingo/l_symbol.go` | `GetBytes()` → retorna `[2 tipo][4 len][string][padding]` |
| `internal/domain/types/lingo/l_float.go` | `GetBytes()` → retorna `[2 tipo][8 float64]` |
| `internal/domain/types/lingo/l_list.go` | `GetBytes()` → retorna `[2 tipo][4 count][elements...]` |

`LPropList` ja tem `GetBytes()` implementado (l_prop_list.go:106). Ele chama `GetBytes()` nos filhos, entao precisa que os tipos acima funcionem primeiro.

Tambem exportar `addElement` → `AddElement` em `l_prop_list.go:23` para construir PropLists de resposta.

## Etapa 2 — Serializar MUSMessage

Adicionar metodos de escrita (inverso do parsing que ja existe):

| Arquivo | Mudanca |
|---------|---------|
| `internal/domain/types/smus/mus_msg_header_string.go` | Adicionar `WriteBytes(buf *bytes.Buffer)` |
| `internal/domain/types/smus/mus_msg_header_string_list.go` | Adicionar `WriteBytes(buf *bytes.Buffer)` |
| `internal/domain/types/smus/mus_message.go` | Adicionar `GetBytes() []byte` — monta: `[header 0x7200][4 contentSize][4 errCode][4 timestamp][subject][senderID][recptID][content]` |

## Etapa 3 — Router de comandos + Response helpers

Novo arquivo: `internal/domain/services/router.go`

```go
type CommandHandler func(clientID string, msg *smus.MUSMessage) (*smus.MUSMessage, error)

type Router struct {
    routes map[string]CommandHandler
}
```

- `Register(subject, handler)` — registra handler por subject exato
- `Route(subject)` — busca handler por subject (map lookup O(1))

Novo arquivo: `internal/domain/services/response.go`
- `BuildResponse(req, errCode, content)` — constroi MUSMessage de resposta (swap sender/recipient, subject mantido, sender = "system")
- `BuildErrorResponse(req, err)` — resposta de erro

## Integracao no SMUSHandler

Modificar `internal/adapters/inbound/smus_handler.go`:
- Adicionar campo `router *Router` na struct
- Alterar `NewSMUSHandler` para receber `*Router`
- No `HandleRawMessage`, apos parsear a mensagem, chamar `router.Route(msg.Subject.Value)` e retornar `response.GetBytes()`

## Wiring

- `internal/factory/handler.go` — criar router, passar para SMUSHandler
- `cmd/gameserver/main.go` — wiring atualizado

## Arquivos novos (2)

- `internal/domain/services/router.go`
- `internal/domain/services/response.go`

## Arquivos modificados (11)

- `internal/domain/types/lingo/l_void.go` — GetBytes()
- `internal/domain/types/lingo/l_integer.go` — GetBytes()
- `internal/domain/types/lingo/l_string.go` — GetBytes()
- `internal/domain/types/lingo/l_symbol.go` — GetBytes()
- `internal/domain/types/lingo/l_float.go` — GetBytes()
- `internal/domain/types/lingo/l_list.go` — GetBytes()
- `internal/domain/types/lingo/l_prop_list.go` — exportar AddElement
- `internal/domain/types/smus/mus_msg_header_string.go` — WriteBytes()
- `internal/domain/types/smus/mus_msg_header_string_list.go` — WriteBytes()
- `internal/domain/types/smus/mus_message.go` — GetBytes()
- `internal/adapters/inbound/smus_handler.go` — integrar router
- `internal/factory/handler.go` — wiring do router
- `cmd/gameserver/main.go` — wiring

## Verificacao

1. `go build ./...` — compilacao sem erros
2. Testes unitarios para GetBytes() round-trip (serialize → parse → comparar)
3. Teste manual: enviar mensagem SMUS e verificar que o servidor responde com bytes validos
