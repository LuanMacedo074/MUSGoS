# Server-Side Scripting — Lua via gopher-lua

> **Status:** Fundação implementada. ScriptEngine funcional com `getSender()`, `getContent()`, `response()`, `publish()`, `sendMessage()`. APIs de DB pendentes.

## Contexto

O MUS original suporta server-side scripting. Precisamos de uma forma extensível para que qualquer pessoa possa escrever lógica customizada no servidor sem alterar o código Go.

**Decisão:** Lua via [gopher-lua](https://github.com/yuin/gopher-lua) — VM pura em Go, sem CGo, sandboxed, padrão da indústria para scripting em game servers.
---

## Fluxo

```
Cliente envia mensagem SMUS
        │
        ▼
   Handler parseia a mensagem
        │
        ▼
   ScriptEngine procura scripts/{subject}.lua
        │
   ┌────┴────┐
   │ existe  │ não existe
   ▼         ▼
 Executa   Fluxo normal
 o script  (mensagens de sistema:
   │        Logon, Logoff, etc.)
   ▼
 Script processa, usa APIs do servidor
   │
   ▼
 Retorna resultado
   │
   ▼
 Service monta resposta SMUS → Cliente
```

### Exemplo concreto

1. Cliente envia mensagem com subject `buy-item`
2. Handler parseia a mensagem SMUS
3. `ScriptEngine` procura `scripts/buy-item.lua`
4. Se existe, executa o script passando os dados (sender, content, etc.)
5. O script processa usando APIs expostas pelo servidor
6. O script retorna o resultado
7. O service monta a mensagem de resposta e devolve pro client

---

## Estrutura

```
external/scripts/                ← pasta de scripts, configurável via SCRIPTS_PATH
├── echo.lua                     ← script exemplo (subject "echo")
├── buy-item.lua                 ← subject "buy-item" → buy-item.lua
└── ...

internal/domain/ports/
└── script_engine.go             ← interface ScriptEngine + ScriptMessage/ScriptResult

internal/domain/types/lingo/
└── lua_convert.go               ← conversão bidirecional LValue ↔ Lua

internal/adapters/outbound/
└── lua_script_engine.go         ← implementação com gopher-lua
```

- Subject da mensagem → nome do arquivo `.lua` (mapping 1:1)
- Se não existe script pro subject, segue o fluxo normal

---

## API exposta para os scripts

O ScriptEngine registra funções Go no ambiente Lua. Os scripts chamam essas funções para interagir com o servidor:

```lua
-- exemplo: scripts/buy-item.lua

local sender = mus.getSender()
local content = mus.getContent()

local itemName = content:get("itemName")
local gold = mus.db.getPlayerAttribute(sender, "gold")

if gold >= 100 then
    mus.db.setPlayerAttribute(sender, "gold", gold - 100)
    mus.db.setPlayerAttribute(sender, itemName, true)
    return mus.response({ success = true, item = itemName })
else
    return mus.response({ success = false, error = "not enough gold" })
end
```

APIs implementadas:
- ✅ `mus.getSender()` — ID do cliente que enviou
- ✅ `mus.getContent()` — conteúdo da mensagem (LValue convertido para tabela Lua)
- ✅ `mus.response(table)` — monta resposta para o client
- ✅ `mus.publish(topic, content)` — publica mensagem na message queue
- ✅ `mus.sendMessage(recipientID, subject, content)` — envia mensagem para outro client via `MessageSender` (valida que recipientID não é vazio)

APIs planejadas:
- `mus.db.getPlayerAttribute(userID, attr)` — lê atributo do jogador
- `mus.db.setPlayerAttribute(userID, attr, value)` — grava atributo do jogador
- `mus.db.getApplicationAttribute(attr)` — lê atributo global da app
- `mus.db.setApplicationAttribute(attr, value)` — grava atributo global

---

## Considerações

- **Segurança**: gopher-lua roda sandboxed — os scripts não têm acesso ao filesystem ou rede diretamente, só às APIs que expusermos
- **Performance**: gopher-lua é rápido o suficiente para game server scripting. Se necessário, pool de VMs Lua para concorrência
- **Hot reload**: possibilidade de recarregar scripts sem reiniciar o servidor (futuro)
- **Erros**: scripts com erro retornam erro pro client sem derrubar o servidor
