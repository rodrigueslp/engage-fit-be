# EngageFit Backend

Manual canônico de arquitetura, regras de negócio e manutenção: [`docs/system-design.md`](docs/system-design.md).

Guia consolidado de segurança, operação, testes, observabilidade, privacidade e preparação para produção: [`docs/application-readiness-guide.md`](docs/application-readiness-guide.md).

Documentação da governança de limites e custos do WhatsApp: [`.ai/messaging-governance.md`](.ai/messaging-governance.md).

Backend Go do EngageFit seguindo Arquitetura Hexagonal.

Este repositorio concentra o codigo da API e tambem a infraestrutura local do projeto (Docker, Makefile, scripts de demo e dados de teste). A pasta `.ai/` documenta o planejamento do produto completo — backend e frontend — para manter o contexto unificado enquanto os deploys continuam separados.

## Estrutura do repositorio

```txt
cmd/api                         # entrypoint HTTP
internal/                       # dominio, casos de uso, adapters
migrations/                     # migrations SQL
scripts/                        # seed demo local
test-data/                      # fixtures de importacao e WhatsApp
.ai/                            # planejamento do produto (backend + frontend)
Makefile                        # atalhos locais (postgres, migrations, demo)
docker-compose.yml              # Postgres local
```

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
internal/adapters/whatsapp      # Twilio e Meta Cloud API
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

No diretorio do backend:

```bash
docker-compose up -d postgres
make migrate-up
```

O comando usa o migrator versionado embutido no binario. Ele mantem a tabela
`schema_migrations`, valida checksum, aplica cada arquivo uma unica vez dentro
de transacao e usa advisory lock do PostgreSQL para impedir execucao concorrente.

```bash
make migrate-status
```

Para um banco preexistente que recebeu os SQLs pelo processo antigo, verifique
primeiro o schema e adote explicitamente o historico ate a versao confirmada:

```bash
make migrate-baseline VERSION=30
```

Nunca use `baseline` em banco vazio ou sem confirmar a versao real. Em release,
execute `/usr/local/bin/engagefit-migrate up` como comando separado antes de
iniciar `/usr/local/bin/engagefit-api`; a API nao altera o schema no startup.

## Criptografia de credenciais

Credenciais dedicadas do WhatsApp e senhas SMTP sao persistidas em envelopes
AES-256-GCM. Configure uma chave ativa e o keyring fora do PostgreSQL:

```bash
openssl rand -base64 32

DATA_ENCRYPTION_ACTIVE_KEY_ID=primary
DATA_ENCRYPTION_KEYS=primary:<base64-de-32-bytes>
```

Em `production` essas variaveis sao obrigatorias. Com chave configurada, a API
recusa valores legados em plaintext. Para adotar um banco antigo, configure as
variaveis e execute o comando de rotacao antes de iniciar a API:

```bash
make rotate-secrets
```

Para trocar a chave, inclua a nova e a antiga no keyring, marque a nova como
ativa, execute `make rotate-secrets` e mantenha ambas durante a atualizacao de
todas as instancias. Apos a rotacao confirmar zero pendencias, a chave antiga
pode ser removida. O ciphertext e vinculado ao tenant e ao campo por associated
data, portanto nao pode ser copiado validamente para outro registro.

As migrations ficam em:

```txt
migrations/
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

Em operacao normal, novas academias devem ser criadas pela conta `PLATFORM_ADMIN` em `Administracao > Academias`. O endpoint `/api/v1/setup/owner` fica reservado ao bootstrap inicial e pode permanecer desabilitado em producao.

O ciclo de vida administrativo usa os estados:

- `active`: login, sessoes, envios e automacoes liberados;
- `suspended`: bloqueia login e sessoes do owner e impede o worker de executar automacoes, preservando todos os dados;
- `archived`: estado terminal no painel, destinado a encerramento com retencao e auditoria.

Criacao, edicao, suspensao, reativacao, arquivamento e redefinicao de senha exigem `PLATFORM_ADMIN`; as operacoes sensiveis registram motivo, administrador, IP e valores anterior/posterior em `admin_audit_logs`.

Rota protegida:

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <token>"
```

## Massa demo

Com backend e Postgres rodando, no diretorio do backend:

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
- Templates e campanhas de WhatsApp demo vinculadas a campanha do mes
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

O sistema oferece duas origens de conexão por academia:

- `Número do EngageFit` (padrão): usa a conta Twilio e o remetente compartilhado configurados no ambiente do backend.
- `Número próprio da academia`: armazena e usa o provedor, remetente e credenciais dedicados daquele tenant.

Configure a conexão compartilhada do EngageFit no `.env` do backend:

```env
WHATSAPP_PLATFORM_ENABLED=true
WHATSAPP_PLATFORM_BASE_URL=https://api.twilio.com
WHATSAPP_PLATFORM_TWILIO_SENDER=whatsapp:+5511000000000
WHATSAPP_PLATFORM_TWILIO_ACCOUNT_SID=AC...
WHATSAPP_PLATFORM_TWILIO_AUTH_TOKEN=...
WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_ALMOST_THERE=HX...
WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_GOAL_REACHED=HX...
WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_WE_MISS_YOU=HX...
```

As credenciais compartilhadas nunca são retornadas pela API nem gravadas por academia. Para uma conexão dedicada, selecione `Número próprio da academia` em `Configurações` e informe:

```txt
Provedor: Twilio WhatsApp
Base URL: https://api.twilio.com
Remetente WhatsApp ou Messaging Service SID: whatsapp:+14155238886 ou MG...
Account SID:Auth Token: AC...:<auth-token>
Ativar WhatsApp: marcado
```

Cada conta Twilio possui seus próprios Content SIDs. Ao usar o número dedicado de uma academia, os templates oficiais também precisam ser criados/aprovados nessa conta e seus respectivos SIDs cadastrados no EngageFit.

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

O provider `meta_cloud` continua disponivel como opcao avancada/futura. Evolution API foi removida do produto.

## Envio real em desenvolvimento

Para receber mensagens reais apenas no seu celular em desenvolvimento, configure `.env`:

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

## Automacao diaria

A rotina operacional diaria pode ser chamada por cron, GitHub Actions ou outro scheduler:

```bash
DAILY_CHECKINS_SOURCE=totalpass \
DAILY_CHECKINS_FILE=/caminho/checkins-do-dia.csv \
DAILY_SEND_MESSAGES=true \
make daily-automation
```

Comportamento:

- se `DAILY_CHECKINS_FILE` estiver configurado, importa o arquivo como `wellhub` ou `totalpass`;
- recalcula todas as campanhas ativas;
- quando `DAILY_SEND_MESSAGES=true`, envia campanhas de mensagem vinculadas a campanhas ativas;
- campanhas de mensagem ja enviadas sao ignoradas, a menos que `DAILY_RESEND_MESSAGES=true`;
- o resultado e impresso no console para auditoria operacional.

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

## Privacidade e retencao

- A tela `Alunos` permite registrar preferencia de contato, exportar os dados de um aluno e anonimiza-lo com confirmacao e motivo.
- Opt-out e anonimizacao removem o aluno das audiencias de WhatsApp, e-mail e Treino do dia; a anonimizacao tambem impede recriacao pela mesma identidade importada.
- `make privacy-retention-dry-run` mostra o que expirou sem excluir. `make privacy-retention-apply` aplica os prazos configurados por `PRIVACY_RETENTION_*`.
- O procedimento completo e os pontos que ainda exigem validacao juridica estao em `docs/privacy-runbook.md`.

## Estado Atual

A aplicacao possui API e frontend completos para execucao local e preparacao do deploy: PostgreSQL com migrations versionadas, repositorios reais, importacao CSV/XLSX, campanhas/metas/brindes, dashboard/relatorios, privacidade, sessao segura, administracao, automacao opcional e observabilidade.

Os gates P0/P1 e a auditoria local ficam registrados em `.ai/application-readiness-checklist.md`. O deploy e operacoes do provedor devem seguir `docs/railway-deployment-checklist.md`.

Continuam deliberadamente fora da prontidao atual: StatusCallback/entrega final/custo real da Twilio, configuracao da infraestrutura Railway, backup/restore gerenciado, alertas externos e validacoes juridicas. Nenhum gateway real e chamado sem capacidade e permissao explicitas.


## E-mail personalizado

O EngageFit tambem possui disparos de e-mail no mesmo modelo operacional do WhatsApp: configuracao de provedor, templates com variaveis, campanhas vinculadas a uma campanha de meta, preview e auditoria por destinatario.

Para desenvolvimento local, use o provider `mock` na tela `E-mail`. Para SMTP real em `APP_ENV=development`, o envio fica bloqueado por padrao e exige:

```bash
EMAIL_ALLOW_REAL_SEND=true
EMAIL_DEV_RECIPIENT_EMAIL=seu-email@exemplo.com # opcional: redireciona todos os envios locais
```

## Auditoria da automacao diaria

O comando `make daily-automation` registra uma execucao em `automation_runs`, finalizando com status, arquivo importado, campanhas recalculadas, mensagens enviadas e falhas. O historico fica disponivel na tela `Automacao` e nos endpoints `/api/v1/automation/runs`.


### Worker interno de automacao

A tela `Automacao` permite configurar rotinas de produto com horario, dias da semana e modo de execucao. Para o backend executar essas rotinas automaticamente, habilite o worker no ambiente da API:

```bash
AUTOMATION_WORKER_ENABLED=true
AUTOMATION_WORKER_INTERVAL_SECONDS=60
AUTOMATION_STALE_RUN_MINUTES=120
AUTOMATION_CATCHUP_WINDOW_MINUTES=15
```

Com o worker desligado, as rotinas continuam configuraveis e podem ser executadas manualmente pelo botao `Executar` na tela `Automacao`.

O claim de cada agenda/minuto e atomico no PostgreSQL, portanto varias replicas
podem manter o worker habilitado sem executar o mesmo slot duas vezes. Cada run
possui `execution_key` unica; chamadas manuais e o frontend enviam
`Idempotency-Key`. Runs que permanecem `running` alem do timeout sao marcadas
como `failed` e geram log operacional.

O script `daily-automation.mjs` usa por padrao a chave `daily:AAAA-MM-DD` no
timezone configurado. Se uma execucao com essa chave ja existe, nenhuma etapa e
repetida. Depois de uma falha parcial, revise o historico antes de tentar de
novo e forneca conscientemente outra chave em
`DAILY_AUTOMATION_IDEMPOTENCY_KEY`. Essa escolha privilegia nao duplicar
mensagens quando o resultado do provedor ficou incerto.
