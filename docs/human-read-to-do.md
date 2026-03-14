# MUSGoS — O que falta e por quê (explicado)

Guia de leitura sobre cada componente que ainda precisa ser implementado, comparado com o [OpenSMUS 1.02](https://sourceforge.net/p/opensmus/code/HEAD/tree/tags/1.02/src/net/sf/opensmus/).

---

## Prioridade CRÍTICA

Sem esses itens, nenhum cliente Shockwave/Director consegue se conectar e trocar mensagens.

---

### ~~1. Error Codes~~ ✅ FEITO

Implementado em `internal/domain/types/smus/mus_error_code.go` — ~54 constantes do protocolo MUS.

---

### ~~2. Response Builder~~ ✅ FEITO

Implementado: `GetBytes()` em todos os tipos Lingo (`LVoid`, `LInteger`, `LString`, `LSymbol`, `LFloat`, `LList`, `LPropList`), `WriteBytes()` em `MUSMsgHeaderString` e `MUSMsgHeaderStringList`, `GetBytes()` em `MUSMessage`, e helpers de resposta em `internal/adapters/inbound/mus/response.go`.

---

### ~~3. Logon Handler~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/logon.go` com 3 modos de autenticação configuráveis via `AUTH_MODE`:
- **`none`** — aceita qualquer conexão sem validação, atribui `DEFAULT_USER_LEVEL`
- **`open`** — aceita qualquer usuário, mas verifica bans. Se o usuário existe no DB, usa o nível do DB; senão, usa `DEFAULT_USER_LEVEL`
- **`strict`** — exige usuário cadastrado no banco, verifica senha (bcrypt) e bans. Usa o nível do DB

Suporta extração de credenciais de `LList` (legado) e `LPropList` (moderno). Integrado ao `SMUSHandler` via factory com parâmetros `authMode` e `defaultUserLevel`. Ao completar o logon, armazena `#userLevel` como atributo de sessão (`LInteger`) para uso futuro pelo dispatcher.

---

### ~~4. Movie (Room) Manager~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/movie.go` — `MovieManager` gerencia movies (salas) sob demanda. Cria movie automaticamente quando o primeiro usuário entra, remove quando o último sai. Auto-join `@AllUsers` group no JoinMovie. Integrado ao `SystemService` para join automático durante o Logon.

---

### ~~5. Group Manager~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/group.go` — `GroupManager` gerencia groups dentro de movies. `@AllUsers` é criado automaticamente pelo MovieManager. Join/leave via SessionStore rooms. Prefixo `@` distingue groups de usuários no roteamento.

---

### ~~6. Message Dispatcher~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/dispatcher.go`. Roteia pelo primeiro recipient da mensagem:
- `"System"` → `SystemService` (Logon, futuros system commands)
- `"system.script"` → `ScriptEngine` (subject = nome do script)
- `"@GroupName"` → `Sender.SendMessage()` broadcast para o group
- `"userName"` → `Sender.SendMessage()` envio direto

O `SMUSHandler` delega toda a lógica de roteamento para o Dispatcher.

---

### ~~7. Group Messaging (Broadcast)~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/sender.go` — `deliverToGroup()`. Serializa a mensagem uma única vez com o groupRef como recipient (conforme spec OpenSMUS), depois entrega a todos os membros do group via `ConnectionWriter`. Membros são resolvidos pelo SessionStore (room `movie:{movieID}:group:{groupName}`).

---

### ~~8. User-to-User Messaging~~ ✅ FEITO

Implementado em `internal/adapters/inbound/mus/sender.go` — `SendMessage()`. Quando o recipientID não começa com `@`, envia diretamente via `ConnectionWriter.WriteToClient()`. O ConnPool (`conn_pool.go`) gerencia o mapeamento clientID→conn com per-conn write mutex para thread safety.

---

## Prioridade MÉDIA

O servidor funciona sem esses itens, mas fica incompleto para uso real.

---

### ~~9. System Commands~~ ✅ FEITO

**OpenSMUS:** `MUSDispatcher.handleSystemMsg()`

Implementado em `system_service_*.go` com handler map (`map[string]handlerFunc`) para roteamento por subject. Todos os comandos system do protocolo MUS estão implementados:

- **Server:** `getVersion`, `getTime`, `getUserCount`, `getMovieCount`, `getMovies`
- **Movie:** `getUserCount`, `getGroups`, `getGroupCount`
- **Group:** `join`, `leave`, `getUsers`, `getUserCount`, `setAttribute`/`getAttribute`/`deleteAttribute`/`getAttributeNames`
- **User:** `getAddress`, `getGroups`, `delete` (com cleanup de sessão — `LeaveAllRooms` + `UnregisterConnection` antes do disconnect)

Permissões via `checkCommandLevel` com deny-by-default: comandos não mapeados no `commandLevels` são rejeitados.

---

### ~~10. DB Dispatcher~~ ✅ FEITO

**OpenSMUS:** `MUSDBDispatcher.java`

Implementado via `handleDBCommand` helper genérico em `system_service.go`, com handlers em `system_service_db_*.go`:

- **DBPlayer:** `getAttribute`, `setAttribute`, `deleteAttribute`, `getAttributeNames`
- **DBApplication:** `getAttribute`, `setAttribute`, `deleteAttribute`, `getAttributeNames`
- **DBAdmin:** `createUser` (bcrypt), `deleteUser`, `createApplication`, `deleteApplication`, `getUserCount`, `ban`, `revokeBan`

Cada handler usa o helper genérico que faz: check permissions → parse proplist → extract fields → execute action → map errors. `dbErrorCode` mapeia erros de domínio para códigos do protocolo MUS (`ErrUserNotFound` → `ErrDatabaseUserIDNotFound`, etc.) usando `errors.Is`.

---

### 11. User Send Queue

**OpenSMUS:** `MUSUserSendQueue.java`

Quando o servidor precisa enviar uma mensagem para um usuário, escrever diretamente no socket TCP pode bloquear (se o buffer do socket estiver cheio, o cliente estiver lento, etc.). A send queue resolve isso:

- Mensagens são colocadas numa fila (channel em Go)
- Uma goroutine dedicada consome a fila e escreve no socket
- Se a fila encher, mensagens antigas são descartadas (ou o usuário é desconectado)

Isso evita que um usuário lento trave o envio para outros.

**O que implementar:** Um channel buffered por conexão, com uma goroutine writer. Relativamente simples em Go graças a goroutines e channels nativos.

---

### 12. Group Send Queue

**OpenSMUS:** `MUSGroupSendQueue.java`

Similar ao User Send Queue, mas para broadcast de grupo. Quando uma mensagem é enviada para um group, em vez de serializar e enviar para cada membro de forma síncrona (bloqueante), a mensagem vai para uma fila do group.

Uma goroutine dedicada consome a fila e distribui para as send queues individuais de cada membro.

**O que implementar:** Channel buffered por group com goroutine de distribuição.

---

### ~~13. Idle Check~~ ✅ FEITO

**OpenSMUS:** `MUSIdleCheck.java`

Implementado em `internal/adapters/inbound/idle_checker.go`:
- `IdleChecker` com goroutine + ticker que varre conexões periodicamente
- `LastActivityAt` adicionado ao `ConnectionInfo` e atualizado após cada mensagem processada com sucesso no TCP server
- `UpdateLastActivity(clientID)` adicionado à interface `SessionStore` e implementado em ambos os stores (memory e Redis)
- Configurável via `IDLE_TIMEOUT` env var (em segundos, 0 = desabilitado)
- Intervalo de verificação = timeout/2 (mínimo 30s)
- Ao detectar conexão idle: `LeaveAllRooms` → `UnregisterConnection` → `DisconnectClient`

---

### ~~14. Lingo Types Faltantes~~ ✅ FEITO

**OpenSMUS:** `LPoint, LRect, LColor, LDate, L3dVector, L3dTransform, LPicture`

Todos os 7 tipos implementados em `internal/domain/types/lingo/`:

- **LPicture** (Vt=5) — dados binários com prefixo de 4 bytes de comprimento
- **LPoint** (Vt=8) — coordenadas `LocH, LocV` como LValues tipados (LInteger ou LFloat), parsados recursivamente via `FromRawBytes`
- **LRect** (Vt=9) — `Left, Top, Right, Bottom` como LValues tipados, parsados recursivamente
- **LColor** (Vt=18) — RGB (3 bytes + 1 padding)
- **LDate** (Vt=19) — 8 bytes opacos
- **L3dVector** (Vt=22) — `X, Y, Z` como float32 big-endian
- **L3dTransform** (Vt=23) — matriz 4x4 de float32 (16 elementos, 64 bytes)

Integração completa: `FromRawBytes` switch, `codec.go` (JSON marshal/unmarshal), `lua_convert.go` (conversão bidirecional Lua).

---

## Prioridade BAIXA

Nice to have. O servidor opera sem eles.

---

### ~~15. Server-Side Scripting~~ ✅ FEITO

**OpenSMUS:** `ServerSideScript.java`, `MUSScriptMap.java`

Sistema de scripting Lua completo:

- **`ports.ScriptEngine`** — interface agnóstica ao protocolo (`HasScript`, `Execute`)
- **`LuaScriptEngine`** — implementação com gopher-lua, VM sandboxed (sem `os`/`io`/`debug`)
- **Conversão bidirecional** LValue ↔ Lua (`lua_convert.go`)
- **APIs disponíveis nos scripts:** `mus.getSender()`, `mus.getContent()`, `mus.response()`, `mus.publish()`, `mus.sendMessage()`
- **APIs de banco (`lua_db_module.go`):** `mus.db.getPlayerAttribute`, `mus.db.setPlayerAttribute`, `mus.db.createUser` (com bcrypt), `mus.db.ban`, `mus.db.revokeBan`, etc.
- **Query builder (`lua_db_module.go`):** `mus.db.table("name"):where(...):get()` para queries arbitrárias em tabelas customizadas
- **APIs de servidor (`lua_server_module.go`):** informações e operações do servidor
- **Roteamento** — scripts são invocados quando o `recipient` da mensagem é `"system.script"` (padrão OpenSMUS). O `subject` da mensagem é usado como nome do script.

**Evolução futura:**
- Event hooks (`userLogOn`, `userLogOff`, `groupJoin`, `groupLeave`)
- Hot reload de scripts sem restart
- Pool de VMs Lua para performance

---

### ~~16. UDP Support~~ ✅ FEITO

**OpenSMUS:** `MUSUDPListener.java`

Implementado em `internal/adapters/inbound/udp_server.go`:
- `UDPServer` com `net.ListenUDP`, loop de leitura, e dispatch para o mesmo `MessageHandler` do TCP
- Stateless — não usa ConnPool. Cliente faz logon via TCP primeiro, depois pode enviar mensagens via UDP
- Configurável via `UDP_PORT` env var (vazio = desabilitado)
- Respostas enviadas de volta via `WriteToUDP` para o endereço do remetente

---

### ~~17. Email Sending~~ ✅ FEITO (interface only)

**OpenSMUS:** `MUSEmail.java`

Implementado:
- **Port:** `ports.EmailSender` interface + `ports.EmailMessage` struct em `internal/domain/ports/email.go`
- **Handler:** `system.server.sendEmail` em `system_service_server.go` — extrai campos do PropList (sender, recipient, subject, SMTPhost, data), delega para `EmailSender`
- **Permissão:** level 80 (admin)
- **Sem implementação concreta** — `emailSender` é passado como `nil` no factory. Para ativar, basta implementar `ports.EmailSender` e injetar via factory.

---

### ~~18. Kill Timers~~ ✅ FEITO

**OpenSMUS:** `MUSKillServerTimer.java`, `MUSKillUserTimer.java`

Implementado:
- **Port:** `ports.TimerManager` interface em `internal/domain/ports/timer.go`
- **Implementação:** `inbound.TimerManager` em `internal/adapters/inbound/timer_manager.go` — usa `time.AfterFunc`, mutex-protegido
- **Server kill timer:** `SetServerKillTimer(minutes)` → `time.AfterFunc` que envia SIGTERM ao servidor
- **User kill timer:** `SetUserKillTimer(clientID, minutes)` → `time.AfterFunc` que faz `LeaveAllRooms` + `UnregisterConnection` + `DisconnectClient`
- **4 comandos:** `system.server.setKillTimer`, `system.server.cancelKillTimer`, `system.user.setKillTimer`, `system.user.cancelKillTimer` (todos level 80)
- **Cleanup:** `Stop()` chamado no shutdown cancela todos os timers pendentes

---

### 21. Cache ✅ FEITO

Sistema de cache genérico com duas implementações:

- **Port:** `ports.Cache` interface em `internal/domain/ports/cache.go` — `Get`, `Set` (com TTL), `Delete`, `Exists`, `Close`
- **Memory:** `internal/adapters/outbound/memory_cache.go` — in-memory com suporte a TTL, mutex-protegido, cópias isoladas (sem mutation leak)
- **Redis:** `internal/adapters/outbound/redis_cache.go` — Redis-backed com key prefix isolado, `NewRedisCacheWithClient` para testes
- **Factory:** `internal/factory/cache.go` — `NewCache(cacheType, redisCfg)` seleciona implementação
- **Config isolada:** `CACHE_TYPE` (memory/redis) + Redis independente (`CACHE_REDIS_HOST`, `CACHE_REDIS_PORT`, `CACHE_REDIS_PASSWORD`, `CACHE_REDIS_DB=2`, `CACHE_REDIS_KEY_PREFIX=musgoc`)
- **Wired** em `main.go` — inicializado e disponível para uso futuro (scripts, session store, rate limiting, etc.)

---

### ~~19. User Levels / Permissions~~ ✅ FEITO

**OpenSMUS:** cache de user levels no `MUSDispatcher`

Implementado end-to-end:
- ✅ Tabela `users` com `user_level` — `CreateUser`, `UpdateUserLevel`
- ✅ `DEFAULT_USER_LEVEL` configurável via env var (default: 20)
- ✅ User level atribuído na sessão (`#userLevel`) durante o logon
- ✅ Console usa `DEFAULT_USER_LEVEL` ao criar usuários
- ✅ `checkCommandLevel` com deny-by-default: comandos devem estar mapeados no `commandLevels` (map configurável), senão são rejeitados
- ✅ `COMMAND_LEVELS` configurável via env var

---

### ~~20. Ban System~~ ✅ FEITO

**OpenSMUS:** `MUSDBDispatcher.ban/revokeBan`

Implementado end-to-end:
- ✅ Tabela `bans` com suporte a ban por user_id e/ou IP, expiração temporal e ban permanente
- ✅ CRUD: `CreateBan`, `GetActiveBanByUserID`, `GetActiveBanByIP`, `RevokeBan`
- ✅ Verificação no logon nos modos `strict` e `open`
- ✅ Comandos `DBAdmin.ban` e `DBAdmin.revokeBan` via protocolo MUS
- ✅ `mus.db.ban()` e `mus.db.revokeBan()` via Lua scripts

---

## Ordem de Implementação Sugerida (MVP)

```
1. Error Codes ──────────── ✅ FEITO (mus_error_code.go)
   Users & Bans DB ────────── ✅ FEITO (migration + CRUD no sqlite_db.go)
   Session Store ───────────── ✅ FEITO (memory + Redis)
   Console ─────────────────── ✅ FEITO (create user)
   Lua Scripting (fundação) ── ✅ FEITO (ScriptEngine + LValue↔Lua)
2. Response Builder ─────── ✅ FEITO (LValue.GetBytes() + MUSMessage.GetBytes())
3. Logon Handler ────────── ✅ FEITO (LogonService com 3 modos: none/open/strict)
4. Movie + Group Manager ── ✅ FEITO (movie sessions + group sessions)
5. Message Dispatcher ───── ✅ FEITO (Dispatcher roteia por primeiro recipient)
6. Group Messaging ──────── ✅ FEITO (Sender broadcast via ConnPool)
7. User-to-User ─────────── ✅ FEITO (Sender direto via ConnPool)
```

Resultado: cliente conecta → autentica → entra em sala → troca mensagens.
