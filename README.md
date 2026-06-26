# EngageFit Backend

Backend Go do EngageFit seguindo Arquitetura Hexagonal.

## Stack

- Go
- Gin
- GORM
- PostgreSQL
- JWT

## Estrutura

```txt
cmd/api                         # entrypoint HTTP
internal/domain                 # entidades e regras centrais
internal/app                    # casos de uso
internal/ports/repositories     # contratos de persistencia
internal/ports/services         # contratos de servicos externos
internal/adapters/http          # router, middleware e handlers
internal/adapters/persistence   # Postgres/GORM
internal/adapters/parsers       # parsers Wellhub/TotalPass
internal/adapters/whatsapp      # Evolution API
internal/adapters/security      # JWT e senha
internal/adapters/reports       # exportadores
internal/config                 # configuracao
migrations                      # migrations futuras
tests                           # testes e fixtures
```

## Rodando localmente

```bash
cp .env.example .env
go run ./cmd/api
```

## Banco local

Na raiz do projeto:

```bash
docker-compose up -d postgres
make migrate-up
```

As migrations ficam em:

```txt
backend/migrations
```

## Validacao

```bash
go test ./...
```

## Bootstrap inicial

Depois de aplicar migrations:

```bash
curl -X POST http://localhost:8080/api/v1/setup/owner \
  -H 'Content-Type: application/json' \
  -d '{
    "box_name": "Meu Box",
    "owner_name": "Owner",
    "owner_email": "owner@example.com",
    "password": "change-me"
  }'
```

Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "owner@example.com",
    "password": "change-me"
  }'
```

Rota protegida:

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <token>"
```

## Massa demo

Com backend e Postgres rodando, na raiz do projeto:

```bash
make demo-seed
```

Para limpar a base local e recriar a massa demo:

```bash
make demo-reset-seed
```

Esse comando executa `TRUNCATE ... CASCADE` nas tabelas do EngageFit antes de rodar o seed. Use somente em ambiente local/desenvolvimento.

O seed cria:

- Box demo `CrossFit Alados`
- Owner demo
- Campanha do mes
- Metas Wellhub e TotalPass
- Brinde
- Alunos e check-ins por importacao XLSX/CSV
- Progresso recalculado
- Configuracao WhatsApp em modo `mock://`
- Template e campanha de WhatsApp demo
- Auditoria de destinatarios em `message_recipients`

Credenciais:

```txt
owner@example.com
change-me
```

Variaveis opcionais:

```bash
API_BASE_URL=http://localhost:8080 \
DEMO_OWNER_EMAIL=owner@example.com \
DEMO_OWNER_PASSWORD=change-me \
DEMO_WELLHUB_FILE=/Users/luizr/Downloads/c01c014d-c88b-4578-a3b9-ffdd24dd3aae.xlsx \
DEMO_TOTALPASS_FILE=/Users/luizr/Downloads/22-06-2026_tokens.xlsx \
make demo-seed
```

Por padrao, o seed tenta importar:

- Wellhub: `/Users/luizr/Downloads/c01c014d-c88b-4578-a3b9-ffdd24dd3aae.xlsx`
- TotalPass: `/Users/luizr/Downloads/22-06-2026_tokens.xlsx`

Se esses arquivos nao existirem, ele usa CSV sintético.

## WhatsApp comercial

Para um MVP comercial, prefira Twilio WhatsApp em vez de uma conexao por QR Code/WhatsApp Web. A Twilio usa a WhatsApp Business Platform por baixo e reduz o atrito operacional de onboarding, sender, envio e templates.

Configuracao em `Configuracoes`:

```txt
Provedor: Twilio WhatsApp
Base URL: https://api.twilio.com
Remetente WhatsApp ou Messaging Service SID: whatsapp:+14155238886 ou MG...
Account SID:Auth Token: AC...:<auth-token>
Ativar WhatsApp: marcado
```

Templates:

- Crie/aprove templates no console da Twilio.
- Cadastre o `Content SID` (`HX...`) no template do EngageFit.
- As variaveis enviadas para a Twilio seguem esta ordem:
  - `1`: `name`
  - `2`: `box_name`
  - `3`: `current_checkins`
  - `4`: `remaining_checkins`
  - `5`: `target_checkins`
  - `6`: `reward_name`
  - `7`: `platform`

O provider `meta_cloud` continua disponivel como opcao avancada/futura. O provider `evolution` fica mantido para legado/desenvolvimento local.

## Evolution API local

Para testar WhatsApp real em desenvolvimento, suba a Evolution API local:

```bash
make evolution-up
```

Servicos:

- Evolution API: `http://localhost:8081`
- API key local: `boxengage-local-key`

Ver status:

```bash
make evolution-ps
```

Logs:

```bash
make evolution-logs
```

Parar:

```bash
make evolution-down
```

Criar instancia WhatsApp:

```bash
curl -X POST http://localhost:8081/instance/create \
  -H "Content-Type: application/json" \
  -H "apikey: boxengage-local-key" \
  -d '{
    "instanceName": "crossfit-alados",
    "qrcode": true,
    "integration": "WHATSAPP-BAILEYS"
  }'
```

Conectar por QR Code:

```bash
curl http://localhost:8081/instance/connect/crossfit-alados \
  -H "apikey: boxengage-local-key"
```

Validar estado:

```bash
curl http://localhost:8081/instance/connectionState/crossfit-alados \
  -H "apikey: boxengage-local-key"
```

Configurar no EngageFit em `Configuracoes`:

```txt
Base URL: http://localhost:8081
Instancia: crossfit-alados
API key: boxengage-local-key
Ativar WhatsApp: marcado
```

Para receber mensagens reais apenas no seu celular em desenvolvimento, configure `backend/.env`:

```env
WHATSAPP_ALLOW_REAL_SEND=true
WHATSAPP_DEV_RECIPIENT_PHONE=55DDDNUMERO
```

Para testar com uma lista fechada de numeros reais, sem redirecionar todos para um unico telefone:

```env
WHATSAPP_ALLOW_REAL_SEND=true
WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES=5511963834712,5518997980429
```

Reinicie o backend apos alterar o `.env`.

## Importacao de check-ins

Upload CSV ou XLSX:

```bash
curl -X POST http://localhost:8080/api/v1/imports \
  -H "Authorization: Bearer <token>" \
  -F "source=wellhub" \
  -F "file=@/caminho/relatorio.csv"
```

Origens aceitas:

- `wellhub`
- `totalpass`

Colunas reconhecidas:

- `nome`, `name`, `aluno`, `student`
- `visitante` no padrao Wellhub
- `colaborador` no padrao TotalPass
- `email`
- `telefone`, `phone`, `celular`, `whatsapp`
- `data`, `date`, `checkin_date`
- `validado em` no padrao TotalPass
- `hora`, `time`, `checkin_time`
- `id`, `external_id`, `matricula`, `id do wellhub`, `codigo`

## Estado Atual

Esta estrutura inicial define:

- Entidades de dominio aprovadas
- Contratos de repositorio e servicos
- Casos de uso principais
- Rotas REST finais
- Adaptadores base

Ainda nao implementa:

- Migrations
- Persistencia real dos repositorios
- Handlers com casos de uso conectados
- Parsers reais de XLSX/CSV
- JWT real
- Integracao real com Evolution API
