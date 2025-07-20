# MUS Go Server

Um servidor implementado em Go para o protocolo SMUS (Shockwave Multi-User Server), compatível com aplicações Flash/Shockwave antigas.

## 🚧 Status do Projeto

**Work in Progress (WIP)** - Este projeto está em desenvolvimento ativo.

## 🛠️ Tecnologias Utilizadas

- **Go 1.21+** - Linguagem principal
- **Docker & Docker Compose** - Containerização e ambiente de desenvolvimento
- **TCP Sockets** - Comunicação de rede
- **Binary Protocol Parsing** - Decodificação do protocolo SMUS/MUS

## 📋 Pré-requisitos

- Go 1.21 ou superior
- Docker e Docker Compose (opcional, mas recomendado)
- Git

## 🚀 Como Executar

### Com Docker (Recomendado)

```bash
# Clone o repositório
git clone https://github.com/seu-usuario/fsos-server.git
cd fsos-server

# Execute com Docker Compose
docker-compose up --build

# O servidor estará rodando na porta 1199
```

### Sem Docker

```bash
# Clone o repositório
git clone https://github.com/seu-usuario/fsos-server.git
cd fsos-server

# Instale as dependências
go mod download

# Execute o servidor
go run cmd/gameserver/main.go

# O servidor estará rodando na porta 1199
```

## 🔧 Configuração

O servidor pode ser configurado através de variáveis de ambiente:

```bash
PORT=1199               # Porta do servidor (padrão: 1199)
LOG_LEVEL=DEBUG         # Nível de log: DEBUG, INFO, WARN, ERROR
ENVIRONMENT=development # Ambiente: development, production
```

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.


## 🙏 Agradecimentos

Agradecimento especial ao **Mauricio Piacentini** <mauricio@tabuleiro.com> por suas contribuições e insights sobre o protocolo SMUS e a comunidade Shockwave.