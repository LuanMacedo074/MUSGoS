# MUSGoS

Servidor MUS (Multiuser Server) escrito em Go, compatível com clientes Macromedia Shockwave/Director que usam o protocolo SMUS.

## Requisitos

- Go 1.24+

## Início rápido

```bash
cp .env.example .env    # configure as variáveis
make run                # inicia o servidor
```

## Comandos

```bash
make test               # roda todos os testes
make test-v             # testes com output verbose
make test-cover         # testes com relatório de cobertura
make test-run T=Nome    # roda teste específico por nome
make build              # compila para bin/gameserver
make run                # executa o servidor
```

## Configuração

Via variáveis de ambiente (veja `.env.example`):

| Variável | Default | Descrição |
|---|---|---|
| `APPLICATION_NAME` | `SMUS-SERVER` | Nome da aplicação |
| `PORT` | `1199` | Porta TCP do servidor |
| `LOG_LEVEL` | `INFO` | Nível de log (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `LOGGER_TYPE` | `file` | Tipo de logger |
| `LOG_PATH` | `logs` | Diretório dos arquivos de log |
| `ENVIRONMENT` | `development` | Ambiente de execução |
| `CIPHER_TYPE` | `blowfish` | Tipo de criptografia |
| `ENCRYPTION_KEY` | — | Chave de criptografia |
| `PROTOCOL` | `smus` | Protocolo de comunicação |

## Arquitetura

O projeto usa **arquitetura hexagonal** (ports & adapters). O domínio define interfaces (ports), e as implementações concretas (adapters) são injetadas via factories.

```
internal/
├── config/              ← variáveis de ambiente
├── factory/             ← resolve implementações concretas
├── domain/
│   ├── types/
│   │   ├── lingo/       ← tipos Lingo (LValue, LString, LInteger, etc.)
│   │   └── smus/        ← protocolo SMUS (MUSMessage, headers)
│   └── ports/           ← interfaces (Cipher, MessageHandler, Logger)
└── adapters/
    ├── inbound/         ← TCP server, SMUS handler
    └── outbound/        ← Blowfish cipher, file logger
```

Regra de dependência: as setas de import sempre apontam para o domínio. Adapters dependem do domínio, nunca o contrário.

Documentação detalhada em [`docs/architecture.md`](docs/architecture.md).

## Testes

Os testes ficam em `_tests/` (diretório com underscore, ignorado pelo `go test ./...`). Rode com `make test`.

```
_tests/
├── testutil/            ← mocks compartilhados
├── config/              ← testes de configuração
├── domain/              ← testes dos tipos e ports
├── factory/             ← testes das factories
└── adapters/            ← testes dos adapters
```

## Créditos

A implementação do Blowfish é baseada no [OpenSMUS](https://github.com/piacentini/OpenSMUS) de Mauricio Piacentini, licenciado sob MIT.
