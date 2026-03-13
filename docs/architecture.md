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
├── config/                        ← carrega variáveis de ambiente
│   └── config.go                  ← struct ServerConfig + LoadServerConfig()
│
├── factory/                       ← resolve qual implementação concreta usar
│   ├── cipher.go                  ← NewCipher() — escolhe o cipher pelo tipo
│   ├── handler.go                 ← NewHandler() — escolhe o handler pelo protocolo
│   └── logger.go                  ← NewLogger() — escolhe o logger pelo tipo
│
├── domain/                        ← núcleo, não depende de nada externo
│   ├── types/
│   │   ├── lingo/                 ← tipos do Lingo (LValue, LString, LInteger, etc.)
│   │   └── smus/                  ← tipos do protocolo SMUS (MUSMessage, headers)
│   └── ports/                     ← interfaces (contratos)
│       ├── cipher.go              ← interface Cipher
│       ├── handler.go             ← interface MessageHandler
│       └── logger.go              ← interface Logger + LogLevel
│
└── adapters/                      ← implementações concretas
    ├── inbound/                   ← adaptadores de ENTRADA
    │   ├── tcp_server.go          ← servidor TCP que aceita conexões
    │   └── smus_handler.go        ← processa mensagens SMUS recebidas
    └── outbound/                  ← adaptadores de SAÍDA
        ├── blowfish.go            ← implementação da criptografia Blowfish
        └── file_logger.go         ← implementação do logger em arquivo
```

---

## O que é cada coisa

### Config

O pacote `config/` centraliza a leitura de variáveis de ambiente numa struct `ServerConfig`. Ele usa `os.LookupEnv` com fallbacks padrão para cada variável. O `main.go` chama `config.LoadServerConfig()` uma vez e passa os valores para as factories.

### Factory

O pacote `factory/` contém funções construtoras (`NewCipher`, `NewHandler`, `NewLogger`) que recebem um tipo (string vinda do config) e retornam a interface correspondente do domínio. Isso isola o `main.go` de conhecer as implementações concretas diretamente — ele só precisa saber o tipo desejado.

### Domain (Domínio)

É o **coração** do sistema. Aqui ficam as regras e estruturas que definem o que o MUSGoS **é**. No nosso caso:

- **`types/lingo/`** — os tipos de dados da linguagem Lingo (strings, inteiros, listas, prop-lists, etc.). Esses tipos existem independente de como o dado chegou ou para onde vai.

- **`types/smus/`** — a estrutura de uma mensagem MUS (`MUSMessage`). Sabe fazer o parsing dos bytes brutos em campos (subject, sender, recipients, conteúdo). Quando precisa descriptografar, ele **não sabe o que é Blowfish** — ele só pede um `ports.Cipher` e chama `.Decrypt()`.

- **`ports/`** — as **interfaces** que o domínio expõe. São os "contratos" que dizem: *"eu preciso de alguém que faça X, não me importa como"*.

### Ports (Portas)

Porta é só um nome bonito para **interface**. São os pontos de conexão entre o domínio e o mundo externo.

Temos três:

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

### Adapters (Adaptadores)

Adaptadores são as **implementações concretas** que conectam o domínio ao mundo real. Existem dois tipos:

#### Inbound (Entrada) — "dados vindo para dentro do sistema"

São os adaptadores que **recebem** dados do mundo externo e os entregam ao domínio.

- **`tcp_server.go`** — abre uma porta TCP, aceita conexões, lê bytes da rede. Quando recebe dados, repassa para o `MessageHandler` (que ele conhece apenas pela interface). Ele é inbound porque é o **ponto de entrada** do sistema: o cliente Shockwave conecta aqui.

- **`smus_handler.go`** — recebe os bytes brutos do TCP server e usa o domínio (`smus.ParseMUSMessageWithDecryption`) para interpretar a mensagem. Ele é inbound porque está do lado de "receber e processar" a requisição. Ele também usa um `ports.Cipher` para a descriptografia, mas não sabe que é Blowfish.

#### Outbound (Saída) — "o sistema acessando recursos externos"

São os adaptadores que o domínio **usa** para fazer coisas que ele não sabe (ou não quer saber) como fazer.

- **`blowfish.go`** — a implementação concreta da criptografia Blowfish. Implementa a interface `ports.Cipher`. É outbound porque é um **recurso** que o domínio consome.

- **`file_logger.go`** — a implementação concreta do logger em arquivo. Implementa a interface `ports.Logger`. Escreve logs formatados em arquivo e no stdout. É outbound porque logging é um **recurso de infraestrutura** que o sistema consome.

---

## Como tudo se conecta

A "cola" acontece no `main.go` com ajuda do `config` e das `factories`. O config carrega as variáveis de ambiente, e as factories criam as instâncias concretas:

```go
cfg := config.LoadServerConfig()

// factory cria o logger (outbound) baseado no tipo configurado
gameLogger, _ := factory.NewLogger(cfg.LoggerType, cfg.ApplicationName, ...)

// factory cria o cipher (outbound) baseado no tipo configurado
cipher, _ := factory.NewCipher(cfg.CipherType, cfg.EncryptionKey)

// factory cria o handler (inbound), injetando logger e cipher pelas interfaces
handler, _ := factory.NewHandler(cfg.Protocol, gameLogger, cipher)

// cria o servidor TCP (inbound), injetando logger e handler pelas interfaces
server := inbound.NewTCPServer(cfg.Port, gameLogger, handler)
```

Perceba que:
- O `main.go` não importa `outbound` diretamente — as factories fazem isso.
- Cada factory retorna a **interface** do domínio (`ports.Logger`, `ports.Cipher`, `ports.MessageHandler`), nunca o tipo concreto.
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

O domínio **nunca** importa adapters, config ou factory. Adapters importam o domínio. As factories importam adapters e ports para montar as peças. O `main.go` importa config, factory e o adapter inbound (TCP server).

---

## Resumo rápido

| Conceito | O que é | No MUSGoS |
|---|---|---|
| **Domain** | Lógica e tipos centrais do sistema | `types/lingo/`, `types/smus/` |
| **Port** | Interface que define um contrato | `Cipher`, `MessageHandler`, `Logger` |
| **Adapter Inbound** | Recebe dados do mundo externo | `TCPServer`, `SMUSHandler` |
| **Adapter Outbound** | Provê capacidades ao domínio | `Blowfish`, `FileLogger` |
| **Config** | Carrega variáveis de ambiente | `ServerConfig`, `LoadServerConfig()` |
| **Factory** | Cria implementações concretas pelo tipo | `NewCipher()`, `NewHandler()`, `NewLogger()` |
| **main.go** | Cola tudo usando config + factories | Injeção de dependência manual |

---

## Camada de Services (planejado)

Em arquiteturas hexagonais tradicionais existe uma camada de "Application Service" com use cases como `LoginUseCase`, `SendMessageUseCase`, etc.

Servidores MUS são **script-driven**: o cliente Shockwave/Director envia scripts Lingo, e o servidor reage. Porém, existem mensagens padrão do protocolo MUS (como `Logon`, `Logoff`, `joinGroup`, `leaveGroup`) que envolvem lógica de negócio real — autenticação, gerenciamento de salas, persistência de estado.

Para essas responsabilidades, a camada `domain/services/` será adicionada, mantendo os adapters finos e a lógica de negócio no domínio.
