# MUSGoS — Roadmap para MVP

Análise comparativa com [OpenSMUS 1.02](https://sourceforge.net/p/opensmus/code/HEAD/tree/tags/1.02/src/net/sf/opensmus/) para identificar o que falta até um servidor mínimo operável.

## O que já funciona

| Componente | Arquivo | OpenSMUS equivalente |
|---|---|---|
| TCP Server | `tcp_server.go` | `MUSServer.java` |
| Message Parsing | `mus_message.go` | `MUSMessage.java` |
| Blowfish Cipher | `blowfish.go` | `MUSBlowfish.java` / `MUSBlowfishCypher.java` |
| Lingo Types (core) | `lingo/*.go` | `LInteger, LString, LSymbol, LList, LPropList, LFloat, LVoid, LMedia` |
| Database (persistência) | `sqlite_db.go` | `MUSSQLConnection.java` |
| Session Store (memory) | `memory_session_store.go` | — (in-memory no OpenSMUS) |
| Session Store (Redis) | `redis_session_store.go` | — |
| Console | `console.go` | — |
| Schema DSL | `ports/schema.go` | — |
| Lingo JSON Codec | `lingo/codec.go` | — |
| Lua Script Engine | `lua_script_engine.go` | `ServerSideScript.java` |
| Message Queue (memory/redis/rabbitmq) | `memory_queue.go`, `redis_queue.go`, `rabbitmq_queue.go` | — |
| Logging | `file_logger.go` | `MUSLog.java` |
| Config | `config.go` | `MUSServerProperties.java` |
| Migrations | `migration_runner.go` | — |
| Error Codes | `mus_error_code.go` | `MUSErrorCode.java` |
| Users & Bans (DB) | `sqlite_db.go` + migration | `MUSSQLConnection.java` (user/ban tables) |
| #All Encryption | `mus_message.go` | `MUSMessage.java` (full-packet decrypt) |
| Movie (Room) Manager | `movie.go` | `MUSMovie.java` |
| Group Manager | `group.go` | `MUSGroup.java` |
| Connection Pool | `conn_pool.go` | — |
| Message Dispatcher | `mus/dispatcher.go` | `MUSDispatcher.java` |
| Message Sender | `mus/sender.go` | `MUSUser.send()` + `MUSGroup.deliver()` |

## O que falta

### Prioridade CRÍTICA — sem isso o servidor não opera

| # | Componente | OpenSMUS equivalente | Descrição |
|---|---|---|---|
| ~~1~~ | ~~**Error Codes**~~ | ~~`MUSErrorCode.java`~~ | ~~Implementado em `mus_error_code.go`~~ |
| ~~2~~ | ~~**Response Builder**~~ | ~~`MUSMessage.send()`~~ | ~~Serializar `MUSMessage` de volta para bytes~~ ✅ FEITO |
| ~~3~~ | ~~**Logon Handler**~~ | ~~`MUSLogonMessage.java`~~ | ~~Processar logon: extrair movieID, userID, password; validar; responder~~ ✅ FEITO |
| ~~4~~ | ~~**Movie (Room) Manager**~~ | ~~`MUSMovie.java`~~ | ~~Gerenciar movies — criar, adicionar/remover usuários, listar groups~~ ✅ FEITO |
| ~~5~~ | ~~**Group Manager**~~ | ~~`MUSGroup.java`~~ | ~~`@AllUsers` auto-join, join/leave, broadcast~~ ✅ FEITO |
| ~~6~~ | ~~**Message Dispatcher**~~ | ~~`MUSDispatcher.java`~~ | ~~Roteamento central por recipient~~ ✅ FEITO (`mus/dispatcher.go`) |
| ~~7~~ | ~~**Group Messaging**~~ | ~~`MUSGroup.deliver()`~~ | ~~Broadcast para todos os membros de um group~~ ✅ FEITO (`mus/sender.go`) |
| ~~8~~ | ~~**User-to-User Messaging**~~ | ~~`MUSUser.send()`~~ | ~~Envio direto entre usuários~~ ✅ FEITO (`mus/sender.go`) |

### Prioridade MÉDIA — servidor funciona sem, mas é incompleto

| # | Componente | OpenSMUS equivalente | Descrição |
|---|---|---|---|
| 9 | **System Commands** | `MUSDispatcher.handleSystemMsg()` | `system.server.*`, `system.group.*`, `system.user.*` |
| 10 | **DB Dispatcher** | `MUSDBDispatcher.java` | Comandos `DBPlayer.*`, `DBApplication.*`, `DBAdmin.*` |
| ~~11~~ | ~~**User Send Queue**~~ | ~~`MUSUserSendQueue.java`~~ | ~~Fila assíncrona por usuário~~ ✅ Substituído pelo sistema de queue genérico (memory/redis/rabbitmq) |
| ~~12~~ | ~~**Group Send Queue**~~ | ~~`MUSGroupSendQueue.java`~~ | ~~Fila assíncrona por group~~ ✅ Substituído pelo sistema de queue genérico |
| 13 | **Idle Check** | `MUSIdleCheck.java` | Desconexão de usuários inativos |
| 14 | **Lingo Types faltantes** | `LPoint, LRect, LColor, LDate, L3dVector, L3dTransform, LPicture` | Tipos raramente usados |

### Prioridade BAIXA — nice to have

| # | Componente | OpenSMUS equivalente | Descrição |
|---|---|---|---|
| ~~15~~ | ~~**Server-side scripting (fundação)**~~ | ~~`ServerSideScript.java`, `MUSScriptMap.java`~~ | ~~ScriptEngine + LValue↔Lua + echo.lua (APIs DB pendentes)~~ |
| 16 | UDP support | `MUSUDPListener.java` | Transporte UDP para baixa latência |
| 17 | Email sending | `MUSEmail.java` | Envio de emails SMTP |
| 18 | Kill timers | `MUSKillServerTimer.java`, `MUSKillUserTimer.java` | Timers de shutdown/desconexão |
| 19 | User levels / permissions | user level cache no `MUSDispatcher` | Controle de acesso por nível (DB pronto, falta enforcement nos System Commands — item 9) |
| 20 | Ban system | `MUSDBDispatcher.ban/revokeBan` | Banimento de usuários (DB pronto, verificação no logon implementada nos modos `strict`/`open`) |

## Fluxo de conexão (referência OpenSMUS)

```
Cliente Shockwave                         Servidor
       |                                      |
       |──── TCP connect ────────────────────►│
       │                                      │ registra conexão
       │──── Logon message (encrypted) ──────►│
       │                                      │ decrypt com Blowfish
       │                                      │ extrai movieID, userID, password
       │                                      │ valida credenciais
       │                                      │ cria/encontra Movie
       │                                      │ adiciona User ao Movie
       │                                      │ auto-join @AllUsers group
       │◄──── Logon reply (success/error) ────│
       │                                      │
       │──── message (subject, recipient) ───►│
       │                                      │ Dispatcher analisa recipient:
       │                                      │   "system.*"  → system command
       │                                      │   "@GroupName" → group broadcast
       │                                      │   "userName"   → user-to-user
       │                                      │   "@MovieName" → cross-movie
       │◄──── response / broadcast ───────────│
       │                                      │
       │──── disconnect ─────────────────────►│
       │                                      │ remove de groups
       │                                      │ remove de movie
       │                                      │ cleanup sessão
```

## Ordem de implementação sugerida (MVP)

```
1. Error Codes ──────────── ✅ FEITO (mus_error_code.go)
   Users & Bans DB ────────── ✅ FEITO (migration + CRUD no sqlite_db.go)
   Session Store ───────────── ✅ FEITO (memory + Redis)
   Console ─────────────────── ✅ FEITO (create user)
   Lua Scripting (fundação) ── ✅ FEITO (ScriptEngine + LValue↔Lua + mus.publish)
   Message Queue ────────────── ✅ FEITO (memory/redis/rabbitmq + registry + factory)
2. Response Builder ─────── ✅ FEITO (LValue.GetBytes() + MUSMessage.GetBytes())
3. Logon Handler ────────── ✅ FEITO (LogonService com 3 modos: none/open/strict)
4. Movie + Group Manager ── ✅ FEITO (movie sessions + group sessions)
5. Message Dispatcher ───── ✅ FEITO (Dispatcher roteia por primeiro recipient)
6. Group Messaging ──────── ✅ FEITO (Sender broadcast via ConnPool)
7. User-to-User ─────────── ✅ FEITO (Sender direto via ConnPool)
```

Resultado: cliente conecta → autentica → entra em sala → troca mensagens.

Para explicação detalhada de cada item, veja [human-read-to-do.md](human-read-to-do.md).
