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

### 9. System Commands

**OpenSMUS:** `MUSDispatcher.handleSystemMsg()`

Comandos enviados para o recipient `"system"` são operações administrativas e de consulta. O `subject` da mensagem determina qual comando executar.

Principais categorias:

**Server:**
- `system.server.getVersion` — retorna a versão do servidor
- `system.server.getTime` — retorna o timestamp atual
- `system.server.getUserCount` — total de usuários conectados
- `system.server.getMovieCount` — total de movies ativos
- `system.server.getMovies` — lista de movies

**Movie:**
- `system.movie.getUserCount` — usuários no movie atual
- `system.movie.getGroups` — groups no movie atual
- `system.movie.getGroupCount` — total de groups

**Group:**
- `system.group.join` — entrar em um group
- `system.group.leave` — sair de um group
- `system.group.getUsers` — listar membros
- `system.group.getUserCount` — total de membros
- `system.group.setAttribute` / `getAttribute` — metadados do group

**User:**
- `system.user.getAddress` — IP do usuário
- `system.user.getGroups` — groups do usuário
- `system.user.delete` — desconectar usuário

Muitos desses comandos requerem verificação de **user level** — apenas administradores podem executar comandos destrutivos como `shutdown`, `disable`, `delete`.

**O que implementar:** Um handler de system commands que faz switch no subject e executa a operação correspondente. Para MVP, os mais importantes são `group.join`, `group.leave`, `group.getUsers`.

---

### 10. DB Dispatcher

**OpenSMUS:** `MUSDBDispatcher.java`

O protocolo MUS tem um sistema de banco de dados integrado. Clientes podem ler e escrever dados persistentes através de mensagens com subjects especiais:

**DBPlayer** (atributos por jogador por aplicação):
- `DBPlayer.getAttribute` — ler atributo do jogador
- `DBPlayer.setAttribute` — salvar atributo do jogador
- `DBPlayer.deleteAttribute` — apagar atributo
- `DBPlayer.getAttributeNames` — listar atributos

**DBApplication** (atributos globais da aplicação):
- `DBApplication.getAttribute/setAttribute/deleteAttribute/getAttributeNames`

**DBAdmin** (administração):
- `DBAdmin.createUser/deleteUser` — gerenciar usuários
- `DBAdmin.createApplication/deleteApplication` — gerenciar aplicações
- `DBAdmin.ban/revokeBan` — banimento

Nós já temos o SQLite adapter com suporte a application attributes e player attributes. O que falta é o **dispatcher** que recebe as mensagens DB e chama os métodos corretos do `DBAdapter`.

**O que implementar:** Um handler que interpreta subjects `DBPlayer.*`, `DBApplication.*`, `DBAdmin.*` e traduz para chamadas no `ports.DBAdapter`.

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

### 13. Idle Check

**OpenSMUS:** `MUSIdleCheck.java`

Um timer periódico que verifica a última atividade de cada conexão. Se um usuário ficar inativo por mais tempo que o limite configurado (idle timeout), ele é desconectado automaticamente.

Isso evita conexões fantasma (cliente crashou sem enviar disconnect, rede caiu, etc.) que consumiriam recursos indefinidamente.

No OpenSMUS, o `checkIdle()` é chamado periodicamente e compara `lastActivityTime` com o timestamp atual.

**O que implementar:** Uma goroutine com ticker que varre as conexões ativas e desconecta as inativas. O Redis session store já tem TTL nas conexões, mas o idle check no nível de aplicação é mais granular.

---

### 14. Lingo Types Faltantes

**OpenSMUS:** `LPoint, LRect, LColor, LDate, L3dVector, L3dTransform, LPicture`

Esses tipos existem no protocolo mas são raramente usados em cenários básicos:

- **LPoint** — coordenada (x, y) como dois inteiros de 32 bits
- **LRect** — retângulo (left, top, right, bottom) como quatro inteiros
- **LColor** — cor RGB como três bytes
- **LDate** — data/hora
- **LPicture** — imagem bitmap serializada
- **L3dVector** — vetor 3D (x, y, z) como três floats
- **L3dTransform** — matriz de transformação 3D (4x4)

Para um chat ou jogo 2D simples, esses tipos quase nunca aparecem. Mas aplicações 3D do Shockwave (como mundos virtuais) os usam extensivamente.

**O que implementar:** Structs que implementam `LValue` para cada tipo. Os mais prováveis de serem necessários primeiro são `LPoint` e `LRect`.

---

## Prioridade BAIXA

Nice to have. O servidor opera sem eles.

---

### ~~15. Server-Side Scripting (fundação)~~ ✅ FEITO

**OpenSMUS:** `ServerSideScript.java`, `MUSScriptMap.java`

A fundação do sistema de scripting Lua está implementada:

- **`ports.ScriptEngine`** — interface agnóstica ao protocolo (`HasScript`, `Execute`)
- **`LuaScriptEngine`** — implementação com gopher-lua, VM sandboxed (sem `os`/`io`/`debug`)
- **Conversão bidirecional** LValue ↔ Lua (`lua_convert.go`)
- **APIs disponíveis nos scripts:** `mus.getSender()`, `mus.getContent()`, `mus.response()`, `mus.publish()`, `mus.sendMessage()`
- **Roteamento** — scripts são invocados quando o `recipient` da mensagem é `"system.script"` (padrão OpenSMUS). O `subject` da mensagem é usado como nome do script. O handler busca `external/scripts/{subject}.lua` e o executa.
- **Script exemplo:** `external/scripts/echo.lua`

**O que falta (evolução futura):**
- APIs de banco (`mus.db.getPlayerAttribute`, `mus.db.setPlayerAttribute`, etc.)
- Event hooks (`userLogOn`, `userLogOff`, `groupJoin`, `groupLeave`)
- Hot reload de scripts sem restart
- Pool de VMs Lua para performance

---

### 16. UDP Support

**OpenSMUS:** `MUSUDPListener.java`

Além de TCP, o protocolo MUS suporta UDP para mensagens de baixa latência (posição de jogadores, animações). O cliente negocia a porta UDP durante o logon.

UDP é útil para dados onde perder um pacote é aceitável (posição atualiza a cada frame), mas a maioria das aplicações funciona apenas com TCP.

---

### 17. Email Sending

**OpenSMUS:** `MUSEmail.java`

O comando `system.server.sendEmail` permite enviar emails SMTP. Usado para recuperação de senha, notificações, etc. O conteúdo da mensagem é um PropList com campos: sender, recipient, subject, SMTPhost, data.

Funcionalidade nicho — a maioria dos deployments não usa.

---

### 18. Kill Timers

**OpenSMUS:** `MUSKillServerTimer.java`, `MUSKillUserTimer.java`

Timers agendados para:
- **Kill server** — desligar o servidor após X tempo (manutenção programada)
- **Kill user** — desconectar um usuário após X tempo (punição temporária, timeout de sessão)

Implementação simples com `time.AfterFunc` em Go.

---

### 19. User Levels / Permissions (DB ✅, sessão ✅, enforcement pendente)

**OpenSMUS:** cache de user levels no `MUSDispatcher`

Cada usuário tem um nível numérico (0-100). Comandos system verificam o nível antes de executar:
- Nível 20+ pode ver informações do servidor
- Nível 80+ pode executar comandos administrativos
- Nível 100 é superadmin

O nível é armazenado no banco e cacheado em memória para performance. O OpenSMUS permite configurar o nível mínimo por comando.

**Status:**
- ✅ Tabela `users` com `user_level` — `CreateUser`, `UpdateUserLevel` implementados
- ✅ `DEFAULT_USER_LEVEL` configurável via env var (default: 20, mesmo do original Multiuser.cfg)
- ✅ User level atribuído na sessão (`#userLevel`) durante o logon:
  - `none` → `DEFAULT_USER_LEVEL`
  - `open` → nível do DB se o usuário existe, senão `DEFAULT_USER_LEVEL`
  - `strict` → nível do DB
- ✅ Console usa `DEFAULT_USER_LEVEL` ao criar usuários
- ❌ Enforcement nos System Commands (verificar `#userLevel` da sessão antes de executar comandos) — depende de System Commands (item 9)

---

### 20. Ban System (DB ✅, verificação no logon ✅)

**OpenSMUS:** `MUSDBDispatcher.ban/revokeBan`

Permite banir usuários por IP ou nome, com duração configurável. Bans são verificados durante o logon — se o usuário está banido, a conexão é recusada com erro.

**Status:** Tabela `bans` com suporte a ban por user_id e/ou IP, expiração temporal e ban permanente. CRUD implementado (`CreateBan`, `GetActiveBanByUserID`, `GetActiveBanByIP`, `RevokeBan`). Verificação no logon implementada nos modos `strict` e `open` do `LogonService`.

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
