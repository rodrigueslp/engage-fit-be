# Plano de Implementacao

Este arquivo acompanha o andamento macro do projeto.

O checklist detalhado fica em:

- `.ai/tasks.md`

---

## Fase 1: Estrutura Inicial do Backend

Status: concluida

Entregas:

- Projeto Go criado em `backend/`
- Arquitetura Hexagonal criada
- Entidades de dominio criadas
- Portas de repositorios e servicos criadas
- Casos de uso iniciais criados
- Router REST criado
- Handlers placeholder criados
- Models GORM iniciais criados
- Backend compilando com `go test ./...`

---

## Fase 2: Banco, Docker Compose e Migrations

Status: concluida parcialmente

Entregas:

- `docker-compose.yml` criado
- PostgreSQL configurado
- `.env.example` criado na raiz
- Migrations SQL criadas em `migrations`
- `Makefile` criado com comandos operacionais
- Checklist `.ai/tasks.md` atualizado
- `docker-compose config` validado

Pendente por ambiente:

- Subir PostgreSQL localmente
- Aplicar migrations com `make migrate-up`

Motivo:

- O Docker daemon nao esta rodando no ambiente atual.

Comandos para validar localmente:

```bash
docker-compose up -d postgres
make migrate-up
```

---

## Fase 3: Persistencia GORM

Status: concluida parcialmente

Objetivo:

- Implementar repositories reais com GORM
- Criar mappers entre models e dominio
- Garantir filtros por `box_id`
- Adicionar testes de repositorio

Entregas:

- Table names dos models alinhados as migrations
- Helper de geracao de UUID
- Mappers model <-> dominio
- Repositories GORM implementados:
  - Box
  - User
  - Student
  - Checkin
  - ImportHistory
  - Campaign
  - Reward
  - WhatsappSettings
  - Message
- Asserts estaticos de compatibilidade com interfaces
- Backend compilando com `go test ./...`

Pendente por ambiente:

- Validar repositories contra PostgreSQL real
- Adicionar testes de integracao de repositorio

Motivo:

- O Docker daemon nao estava rodando no ambiente durante a fase 2.

---

## Fase 4: Auth e Tenant Context

Status: concluida parcialmente

Objetivo:

- Implementar password hashing
- Implementar JWT real
- Implementar middleware de autenticacao
- Resolver `user_id` e `box_id` no contexto
- Criar seed ou fluxo inicial de owner

Entregas:

- Password hashing com bcrypt
- JWT HMAC-SHA256 com `user_id`, `box_id`, `role`, `iat` e `exp`
- Middleware de autenticacao por Bearer token
- Tenant context com `box_id`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/login`
- `POST /api/v1/setup/owner`
- Wiring real no `cmd/api/main.go` com banco, repositories, services e use cases
- Backend compilando com `go test ./...`

Pendente por ambiente:

- Validar fluxo completo contra PostgreSQL real:
  - aplicar migrations
  - criar owner
  - login
  - acessar rota protegida com Bearer token

---

## Fase 5: Handlers e Casos de Uso Reais

Status: concluida parcialmente

Objetivo:

- Conectar handlers aos casos de uso
- Padronizar validacao de payloads
- Padronizar tratamento de erros
- Implementar endpoints por dominio

Entregas:

- Tratamento centralizado de erro HTTP
- Handlers conectados:
  - Auth
  - Setup owner
  - Box
  - Students basico
  - Imports leitura
  - Campaigns leitura e fechamento
  - Campaign goals leitura
  - Campaign progress leitura
  - Rewards leitura, pendencias e marcar entregue
  - Whatsapp settings
  - Message templates basico
  - Message campaigns leitura
  - Message recipients leitura
- Router usando construtores com dependencias
- `cmd/api/main.go` montando repositories, services e use cases principais
- Backend compilando com `go test ./...`

Ainda pendente:

- Upload/importacao real
- Criar/editar/remover campanhas
- Criar/editar/remover metas
- Recalculo real de progresso
- Endpoints de elegiveis/proximos da meta
- Criar/editar/remover brindes
- Atualizar/remover templates
- Criar/enviar campanhas de mensagem
- Testes HTTP/integracao contra banco

---

## Fase 6: Importacoes

Status: concluida parcialmente

Objetivo:

- Implementar parser Wellhub
- Implementar parser TotalPass
- Suportar CSV
- Suportar XLSX
- Criar alunos automaticamente
- Criar check-ins sem receita
- Recalcular progresso

Entregas:

- `POST /api/v1/imports` conectado a upload multipart
- Parser CSV com suporte a `,` e `;`
- Parser XLSX basico via OpenXML sem dependencia externa
- Mapeamento flexivel de colunas:
  - nome/name/aluno/student
  - email
  - telefone/phone/celular/whatsapp
  - data/date/checkin_date
  - hora/time/checkin_time
  - id/external_id/matricula
- Criacao automatica de alunos por identidade importada
- Criacao de check-ins sem campo financeiro
- Registro de `ImportHistory`
- Recalculo de progresso das campanhas ativas
- Testes unitarios do parser CSV
- Backend compilando com `go test ./...`

Pendente por ambiente:

- Validar importacao ponta a ponta contra PostgreSQL real
- Testar com arquivos reais Wellhub e TotalPass
- Ajustar aliases de colunas conforme arquivos reais

---

## Fase 7: Campanhas, Brindes e Dashboard

Status: concluida parcialmente

Objetivo:

- CRUD de campanhas
- Metas por plataforma
- Progresso de campanha
- Elegiveis
- Proximos da meta
- Alunos em risco
- Brindes pendentes e entregues
- KPIs do dashboard

Entregas:

- CRUD basico de campanhas
- Fechamento de campanha
- CRUD de metas por plataforma
- Recalculo de progresso de campanha
- Regra `near_goal >= 80%`
- Listagem de elegiveis
- Listagem de proximos da meta
- CRUD basico de brindes
- Controle de quantidade de brindes
- Geracao de entregas pendentes para alunos elegiveis durante recalc/import
- Marcar entrega como realizada
- Dashboard summary com KPIs
- Dashboard campanhas ativas
- Dashboard proximos da meta
- Dashboard alunos em risco por 7 dias sem check-in
- Dashboard brindes pendentes
- Backend compilando com `go test ./...`

Pendente por ambiente:

- Validar fluxo completo contra PostgreSQL real
- Testar com dados reais importados
- Ajustar consultas/performance conforme volume

---

## Fase 8: WhatsApp e Auditoria

Status: concluida parcialmente

Objetivo:

- Configurar Evolution API
- Templates
- Campanhas de mensagem
- Segmentacao
- MessageRecipient
- Auditoria de sucesso/falha

Entregas:

- Update/delete de templates
- Criacao de campanha de mensagem
- Consulta de campanha de mensagem
- Envio de campanha de mensagem
- Resolucao de publico:
  - `near_goal`
  - `achieved`
  - `inactive`
  - `all`
- Criacao de `MessageRecipient`
- Atualizacao individual de status:
  - `sent`
  - `failed`
- Registro de erro por destinatario
- Renderizacao simples de template:
  - `{{name}}`
  - `{{nome}}`
  - `{{email}}`
- Evolution API client via HTTP
- Backend compilando com `go test ./...`

Pendente por ambiente:

- Validar endpoints contra Evolution API real
- Confirmar formato exato esperado pela instancia Evolution API usada
- Validar disparos com numeros reais/teste

---

## Fase 9: Frontend

Status: concluida

Objetivo:

- React + Vite + Tailwind configurados
- Primitives locais em `components/ui` seguindo padrao shadcn/ui
- Lucide Icons configurado
- Layout principal com sidebar e header
- Dashboard first
- Telas operacionais iniciais

Entregue:

- Aplicacao em `frontend`
- Build de producao validado com `npm run build`
- Dev server iniciado em `http://127.0.0.1:5173`
- Integracao inicial com APIs REST do backend via client HTTP

Telas criadas:

- Login
- Dashboard
- Campanhas
- Alunos
- Importacoes
- WhatsApp
- Relatorios
- Configuracoes

---

## Fase 10: Campanhas como Fluxo Operacional

Status: concluida parcialmente

Objetivo:

- Tornar a tela de campanhas aderente ao fluxo real do box
- Permitir criar campanha com metas por plataforma
- Permitir definir brinde da campanha
- Exibir acompanhamento operacional da campanha

Entregue:

- Tela de campanhas reorganizada
- Criacao de campanha com:
  - nome
  - descricao
  - periodo
  - meta Wellhub
  - meta TotalPass
  - brinde
  - quantidade do brinde
- Integracao frontend com:
  - `POST /api/v1/campaigns`
  - `POST /api/v1/campaigns/:id/goals`
  - `POST /api/v1/campaigns/:id/rewards`
  - `GET /api/v1/campaigns/:id/goals`
  - `GET /api/v1/campaigns/:id/rewards`
  - `GET /api/v1/campaigns/:id/progress`
  - `POST /api/v1/campaigns/:id/recalculate-progress`
- Painel operacional com:
  - metas configuradas
  - brinde configurado
  - alunos que atingiram meta
  - alunos proximos da meta
  - progresso medio
  - lista inicial de progresso

Validacao:

- Frontend: `npm run build`
- Backend: `go test ./...`

Pendente:

- Implementar e-mail
- Implementar automacao diaria de importacao, recalculo e disparo

---

## Fase 11: Mensagens Personalizadas por Progresso

Status: concluida parcialmente

Objetivo:

- Permitir mensagens personalizadas com dados reais de progresso da campanha
- Evitar que o frontend dependa de UUID do aluno para exibir progresso

Entregue:

- `GET /api/v1/campaigns/:id/progress` agora retorna:
  - nome do aluno
  - e-mail
  - telefone
  - plataforma
  - check-ins atuais
  - meta
  - check-ins faltantes
- Motor de template WhatsApp passou a suportar:
  - `{{name}}`
  - `{{nome}}`
  - `{{email}}`
  - `{{phone}}`
  - `{{telefone}}`
  - `{{source}}`
  - `{{platform}}`
  - `{{plataforma}}`
  - `{{box_name}}`
  - `{{campaign_name}}`
  - `{{reward_name}}`
  - `{{current_checkins}}`
  - `{{checkins}}`
  - `{{target_checkins}}`
  - `{{goal_checkins}}`
  - `{{remaining_checkins}}`
  - `{{faltam_checkins}}`
- Tela de WhatsApp agora inicia template com mensagem operacional baseada em progresso
- Tela de WhatsApp permite inserir variaveis de template

Validacao:

- Frontend: `npm run build`
- Backend: `go test ./...`

Pendente:

- Associar campanha de mensagem a uma campanha de meta especifica
- Implementar e-mail personalizado
- Implementar automacao diaria de importacao, recalculo e disparo

---

## Fase 12: Ambiente Demo de Desenvolvimento

Status: concluida

Objetivo:

- Permitir testar o fluxo principal sem depender de dados reais
- Permitir testar auditoria de WhatsApp sem Evolution API real

Entregue:

- `make demo-seed`
- Script `scripts/demo-seed.mjs`
- Massa demo criada via API publica/autenticada
- Owner demo:
  - `owner@example.com`
  - `change-me`
- Box demo `CrossFit Alados`
- Campanha demo do mes atual
- Metas:
  - Wellhub: 12 check-ins
  - TotalPass: 10 check-ins
- Brinde demo
- Importacao CSV sintética para Wellhub e TotalPass
- Recalculo de progresso da campanha
- WhatsApp mock usando `mock://local`
- Template demo com variaveis de progresso
- Campanha de mensagem demo para alunos proximos da meta
- Auditoria de destinatarios em `message_recipients`
- Importacao automatica das planilhas reais quando presentes:
  - Wellhub: `/Users/luizr/Downloads/c01c014d-c88b-4578-a3b9-ffdd24dd3aae.xlsx`
  - TotalPass: `/Users/luizr/Downloads/22-06-2026_tokens.xlsx`

Validacao:

- `node --check scripts/demo-seed.mjs`
- Backend: `go test ./...`
- Frontend: `npm run build`

Como usar:

1. Subir Postgres e aplicar migrations.
2. Subir backend.
3. Rodar `make demo-seed` na raiz.
4. Entrar no frontend com `owner@example.com` / `change-me`.

---

## Fase 13: Importacao de Planilhas Reais

Status: concluida

Objetivo:

- Ajustar parser para os arquivos reais Wellhub e TotalPass fornecidos

Entregue:

- Deteccao automatica da linha de cabecalho mesmo com preambulo
- Suporte Wellhub:
  - `Data`
  - `Hora`
  - `Visitante`
  - `ID do Wellhub`
- Suporte TotalPass tokens:
  - `ID`
  - `Colaborador`
  - `Validado em`
- Suporte a data e hora no mesmo campo
- Normalizacao de coluna `TIME` do Postgres na persistencia
- Testes de parser com exemplos dos formatos reais

Validacao:

- Backend: `go test ./...`

---

## Fase 14: Idempotencia de Importacao

Status: concluida

Objetivo:

- Impedir que a mesma planilha gere check-ins duplicados
- Permitir reimportacao segura de arquivos repetidos ou parcialmente repetidos

Decisao:

- A chave natural de um check-in e:
  - `box_id`
  - `source`
  - `student_id`
  - `checkin_date`
  - `checkin_time`

Entregue:

- Migration `016_add_unique_checkins.sql`
- Indice unico `idx_checkins_unique_visit`
- `SaveMany` usa `ON CONFLICT DO NOTHING`
- Importacao retorna quantidade de check-ins realmente inseridos
- `make demo-reset` garante o indice em bases locais ja existentes

Comportamento:

- Importar a mesma planilha novamente nao duplica check-ins.
- Importar planilha parcialmente repetida insere apenas check-ins novos.
- Recalculo de campanha usa a base deduplicada.

Validacao:

- Backend: `go test ./...`
- Makefile: `make -n demo-reset-seed`

---

## Fase 15: Envio Real Seguro em Desenvolvimento

Status: concluida

Objetivo:

- Permitir teste real de WhatsApp sem risco de disparar para alunos

Decisao:

- Em `development`, envio real e bloqueado por padrao.
- Para habilitar envio real local:
  - `WHATSAPP_ALLOW_REAL_SEND=true`
  - `WHATSAPP_DEV_RECIPIENT_PHONE=<numero_de_teste>`
- Mesmo que a campanha tenha varios alunos, todos os envios sao redirecionados para `WHATSAPP_DEV_RECIPIENT_PHONE`.
- `mock://` continua funcionando sem envio real.
- Em `production`, nao ha override local.

Entregue:

- `SafeGateway` para WhatsApp
- Variaveis em `.env`
- Composicao do gateway protegida no `cmd/api/main.go`
- Tela de configuracao da Evolution API em Configuracoes
- Botao de teste de conexao com Evolution API

Validacao:

- Backend: `go test ./...`
- Frontend: `npm run build`
