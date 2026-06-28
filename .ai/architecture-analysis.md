# Blueprint Tecnico Final do EngageFit

Este documento consolida a arquitetura final validavel antes da implementacao.

Fontes consideradas:

- `product-vision.md`
- `domain.md`
- `database.md`
- `architecture.md`
- `features.md`
- `tasks.md`
- `decisions.md`
- `ui-ux.md`

---

## 1. Modelo Relacional Final

### `boxes`

Representa um box de CrossFit cliente da plataforma.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `name` | VARCHAR | Nome do box |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Relacionamentos:

- Um box possui muitos usuarios
- Um box possui muitos alunos
- Um box possui muitas importacoes
- Um box possui muitas campanhas
- Um box possui uma configuracao de WhatsApp
- Um box possui templates e campanhas de mensagem

---

### `users`

Usuario autenticado da plataforma.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `name` | VARCHAR | |
| `email` | VARCHAR | Unico |
| `password_hash` | VARCHAR | |
| `role` | VARCHAR | Apenas `OWNER` no MVP |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Regras:

- O MVP possui apenas o perfil `OWNER`.
- Toda acao autenticada deve respeitar o `box_id` do usuario.

---

### `students`

Aluno identificado via importacoes.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `name` | VARCHAR | |
| `email` | VARCHAR | Opcional |
| `phone` | VARCHAR | Opcional |
| `source` | VARCHAR | `wellhub` ou `totalpass` |
| `external_id` | VARCHAR | ID externo quando existir |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Indices recomendados:

- `(box_id, source, external_id)`
- `(box_id, name)`
- `(box_id, phone)`

---

### `import_histories`

Historico de importacao de arquivos.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `filename` | VARCHAR | |
| `source` | VARCHAR | `wellhub` ou `totalpass` |
| `total_records` | INTEGER | |
| `imported_at` | TIMESTAMP | |

---

### `checkins`

Check-in realizado por aluno.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `student_id` | FK | `students.id` |
| `checkin_date` | DATE | |
| `checkin_time` | TIME | Opcional |
| `source` | VARCHAR | `wellhub` ou `totalpass` |
| `import_history_id` | FK | `import_histories.id` |
| `created_at` | TIMESTAMP | |

Regras:

- Nao possui campo `revenue`.
- Receita financeira esta fora do MVP.

Indices recomendados:

- `(box_id, checkin_date)`
- `(box_id, student_id, checkin_date)`
- `(box_id, source, checkin_date)`

---

### `campaigns`

Campanha de incentivo.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `name` | VARCHAR | |
| `description` | TEXT | |
| `start_date` | DATE | |
| `end_date` | DATE | |
| `active` | BOOLEAN | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

---

### `campaign_goals`

Meta por plataforma dentro da campanha.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `campaign_id` | FK | `campaigns.id` |
| `source` | VARCHAR | `wellhub` ou `totalpass` |
| `target_checkins` | INTEGER | |

Constraint recomendada:

- Unico: `(campaign_id, source)`

---

### `rewards`

Brinde associado a uma campanha.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `campaign_id` | FK | `campaigns.id` |
| `name` | VARCHAR | |
| `description` | TEXT | |
| `quantity` | INTEGER | Quantidade planejada/disponivel |

---

### `campaign_progresses`

Progresso do aluno em uma campanha.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `campaign_id` | FK | `campaigns.id` |
| `student_id` | FK | `students.id` |
| `current_checkins` | INTEGER | |
| `target_checkins` | INTEGER | |
| `progress_percentage` | DECIMAL | Percentual de progresso |
| `achieved` | BOOLEAN | Meta atingida |
| `near_goal` | BOOLEAN | Progresso maior ou igual a 80% |
| `updated_at` | TIMESTAMP | |

Constraint recomendada:

- Unico: `(campaign_id, student_id)`

Regras:

- `achieved = current_checkins >= target_checkins`
- `near_goal = progress_percentage >= 80`

---

### `reward_deliveries`

Entrega de brinde para aluno.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `reward_id` | FK | `rewards.id` |
| `student_id` | FK | `students.id` |
| `delivered` | BOOLEAN | |
| `delivered_at` | TIMESTAMP | Opcional |

Constraint recomendada:

- Unico: `(reward_id, student_id)`

---

### `whatsapp_settings`

Configuracao do provedor WhatsApp por box.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `base_url` | VARCHAR | URL do provedor WhatsApp |
| `instance_name` | VARCHAR | |
| `api_key_encrypted` | TEXT | Chave protegida |
| `enabled` | BOOLEAN | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Constraint recomendada:

- Unico: `(box_id)`

---

### `message_templates`

Template de mensagem.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `name` | VARCHAR | |
| `content` | TEXT | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

---

### `message_campaigns`

Disparo de mensagens para publico segmentado.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `box_id` | FK | `boxes.id` |
| `name` | VARCHAR | |
| `audience` | VARCHAR | `near_goal`, `achieved`, `inactive`, `all` |
| `template_id` | FK | `message_templates.id` |
| `sent_at` | TIMESTAMP | Opcional |
| `created_at` | TIMESTAMP | |

---

### `message_recipients`

Auditoria individual dos disparos.

| Campo | Tipo | Observacoes |
|---|---|---|
| `id` | UUID / BIGINT | PK |
| `message_campaign_id` | FK | `message_campaigns.id` |
| `student_id` | FK | `students.id` |
| `phone` | VARCHAR | Snapshot do telefone usado |
| `status` | VARCHAR | `pending`, `sent`, `failed` |
| `error_message` | TEXT | Opcional |
| `sent_at` | TIMESTAMP | Opcional |
| `created_at` | TIMESTAMP | |

---

## 2. Estrutura Final do Backend

```txt
backend/
  cmd/
    api/
      main.go

  internal/
    domain/
      box.go
      user.go
      student.go
      checkin.go
      import_history.go
      campaign.go
      campaign_goal.go
      campaign_progress.go
      reward.go
      reward_delivery.go
      whatsapp_settings.go
      message_template.go
      message_campaign.go
      message_recipient.go

    app/
      auth/
        login_usecase.go
        get_current_user_usecase.go

      boxes/
        create_box_usecase.go
        get_box_usecase.go

      students/
        list_students_usecase.go
        get_student_usecase.go
        list_student_checkins_usecase.go

      imports/
        import_checkins_usecase.go
        list_imports_usecase.go
        get_import_usecase.go

      campaigns/
        create_campaign_usecase.go
        update_campaign_usecase.go
        close_campaign_usecase.go
        list_campaigns_usecase.go
        get_campaign_usecase.go
        upsert_campaign_goal_usecase.go
        recalculate_campaign_progress_usecase.go
        list_campaign_progress_usecase.go

      rewards/
        create_reward_usecase.go
        update_reward_usecase.go
        list_rewards_usecase.go
        list_pending_reward_deliveries_usecase.go
        mark_reward_delivered_usecase.go

      dashboard/
        get_dashboard_summary_usecase.go
        list_active_campaigns_usecase.go
        list_near_goal_students_usecase.go
        list_at_risk_students_usecase.go
        list_pending_rewards_usecase.go

      whatsapp/
        get_whatsapp_settings_usecase.go
        update_whatsapp_settings_usecase.go
        test_whatsapp_settings_usecase.go

      messages/
        create_message_template_usecase.go
        update_message_template_usecase.go
        list_message_templates_usecase.go
        create_message_campaign_usecase.go
        send_message_campaign_usecase.go
        list_message_recipients_usecase.go

      reports/
        eligible_students_report_usecase.go
        pending_rewards_report_usecase.go
        monthly_frequency_report_usecase.go

    ports/
      repositories/
        box_repository.go
        user_repository.go
        student_repository.go
        checkin_repository.go
        import_history_repository.go
        campaign_repository.go
        reward_repository.go
        whatsapp_settings_repository.go
        message_repository.go

      services/
        token_service.go
        password_service.go
        checkin_file_parser.go
        whatsapp_gateway.go
        report_exporter.go

    adapters/
      http/
        router.go
        middleware/
          auth_middleware.go
          tenant_middleware.go
          error_middleware.go

        handlers/
          auth_handler.go
          boxes_handler.go
          students_handler.go
          imports_handler.go
          campaigns_handler.go
          rewards_handler.go
          dashboard_handler.go
          whatsapp_handler.go
          messages_handler.go
          reports_handler.go

        dto/
          auth_dto.go
          boxes_dto.go
          students_dto.go
          imports_dto.go
          campaigns_dto.go
          rewards_dto.go
          dashboard_dto.go
          whatsapp_dto.go
          messages_dto.go
          reports_dto.go

      persistence/
        postgres/
          db.go
          models/
            box_model.go
            user_model.go
            student_model.go
            checkin_model.go
            import_history_model.go
            campaign_model.go
            campaign_goal_model.go
            campaign_progress_model.go
            reward_model.go
            reward_delivery_model.go
            whatsapp_settings_model.go
            message_template_model.go
            message_campaign_model.go
            message_recipient_model.go

          repositories/
            box_gorm_repository.go
            user_gorm_repository.go
            student_gorm_repository.go
            checkin_gorm_repository.go
            import_history_gorm_repository.go
            campaign_gorm_repository.go
            reward_gorm_repository.go
            whatsapp_settings_gorm_repository.go
            message_gorm_repository.go

      parsers/
        wellhub_csv_parser.go
        wellhub_xlsx_parser.go
        totalpass_csv_parser.go
        totalpass_xlsx_parser.go

      whatsapp/
        twilio_client.go

      security/
        jwt_service.go
        bcrypt_password_service.go

      reports/
        csv_exporter.go

    config/
      config.go
      env.go

  migrations/

  tests/
    integration/
    fixtures/

  Dockerfile
  go.mod
  go.sum
```

Observacao: migrations ainda nao devem ser geradas nesta etapa.

---

## 3. Estrutura Final do Frontend

```txt
frontend/
  src/
    app/
      App.tsx
      router.tsx
      providers.tsx

    pages/
      login/
        LoginPage.tsx

      dashboard/
        DashboardPage.tsx
        components/
          KpiGrid.tsx
          ActiveCampaigns.tsx
          NearGoalStudents.tsx
          AtRiskStudents.tsx
          PendingRewards.tsx

      campaigns/
        CampaignsPage.tsx
        CampaignDetailsPage.tsx
        CampaignFormPage.tsx
        components/
          CampaignForm.tsx
          CampaignGoalEditor.tsx
          CampaignProgressTable.tsx
          RewardEditor.tsx

      students/
        StudentsPage.tsx
        StudentDetailsPage.tsx
        components/
          StudentsFilters.tsx
          StudentsTable.tsx
          StudentCheckins.tsx
          StudentCampaignStatus.tsx

      imports/
        ImportsPage.tsx
        ImportUploadPage.tsx
        components/
          UploadDropzone.tsx
          ImportHistoryTable.tsx

      whatsapp/
        WhatsappSettingsPage.tsx
        MessageTemplatesPage.tsx
        MessageCampaignsPage.tsx
        components/
          WhatsappSettingsForm.tsx
          TemplateForm.tsx
          AudienceSelector.tsx
          MessagePreview.tsx
          MessageRecipientsTable.tsx

      reports/
        ReportsPage.tsx

      settings/
        SettingsPage.tsx

    components/
      layout/
        AppLayout.tsx
        Sidebar.tsx
        Header.tsx
        PageHeader.tsx

      ui/
        # shadcn/ui components

      common/
        EmptyState.tsx
        LoadingState.tsx
        ErrorState.tsx
        DataTable.tsx
        SourceBadge.tsx
        StatusBadge.tsx
        KpiCard.tsx

    features/
      auth/
        auth.api.ts
        auth.types.ts
        auth.store.ts

      dashboard/
        dashboard.api.ts
        dashboard.types.ts

      students/
        students.api.ts
        students.types.ts

      imports/
        imports.api.ts
        imports.types.ts

      campaigns/
        campaigns.api.ts
        campaigns.types.ts

      rewards/
        rewards.api.ts
        rewards.types.ts

      whatsapp/
        whatsapp.api.ts
        whatsapp.types.ts

      messages/
        messages.api.ts
        messages.types.ts

      reports/
        reports.api.ts
        reports.types.ts

    lib/
      api.ts
      auth.ts
      dates.ts
      formatters.ts
      constants.ts

    styles/
      globals.css

  public/

  index.html
  package.json
  vite.config.ts
  tailwind.config.ts
  components.json
```

Observacao: componentes React ainda nao devem ser gerados nesta etapa.

---

## 4. Lista Final de APIs REST

Base: `/api/v1`

### Auth

```txt
POST   /auth/login
POST   /auth/logout
GET    /auth/me
```

### Setup

Endpoint publico para bootstrap inicial do primeiro box e owner:

```txt
POST   /setup/owner
```

### Boxes

```txt
GET    /box
PUT    /box
```

### Dashboard

```txt
GET    /dashboard/summary
GET    /dashboard/active-campaigns
GET    /dashboard/near-goal-students
GET    /dashboard/at-risk-students
GET    /dashboard/pending-rewards
```

### Students

```txt
GET    /students
GET    /students/:id
GET    /students/:id/checkins
GET    /students/:id/campaign-progress
```

Filtros:

```txt
source
search
campaign_id
achieved
near_goal
inactive
page
limit
```

### Imports

```txt
POST   /imports
GET    /imports
GET    /imports/:id
```

Upload:

```txt
POST /imports
Content-Type: multipart/form-data

fields:
- file
- source: wellhub | totalpass
```

### Campaigns

```txt
GET    /campaigns
POST   /campaigns
GET    /campaigns/:id
PUT    /campaigns/:id
PATCH  /campaigns/:id/close
DELETE /campaigns/:id
```

### Campaign Goals

```txt
GET    /campaigns/:campaignId/goals
POST   /campaigns/:campaignId/goals
PUT    /campaigns/:campaignId/goals/:goalId
DELETE /campaigns/:campaignId/goals/:goalId
```

### Campaign Progress

```txt
GET    /campaigns/:campaignId/progress
POST   /campaigns/:campaignId/recalculate-progress
GET    /campaigns/:campaignId/eligible-students
GET    /campaigns/:campaignId/near-goal-students
```

### Rewards

```txt
GET    /campaigns/:campaignId/rewards
POST   /campaigns/:campaignId/rewards
PUT    /rewards/:id
DELETE /rewards/:id
GET    /rewards/pending-deliveries
PATCH  /reward-deliveries/:id/deliver
```

### WhatsApp Settings

```txt
GET    /whatsapp/settings
PUT    /whatsapp/settings
POST   /whatsapp/settings/test
```

### Message Templates

```txt
GET    /message-templates
POST   /message-templates
GET    /message-templates/:id
PUT    /message-templates/:id
DELETE /message-templates/:id
```

### Message Campaigns

```txt
GET    /message-campaigns
POST   /message-campaigns
GET    /message-campaigns/:id
POST   /message-campaigns/:id/send
GET    /message-campaigns/:id/recipients
```

### Reports

```txt
GET    /reports/eligible-students
GET    /reports/pending-rewards
GET    /reports/monthly-frequency
```

Formatos:

```txt
?format=json
?format=csv
```

---

## 5. Fluxos Principais do Sistema

### Login

1. Usuario acessa a aplicacao.
2. Envia email e senha.
3. Backend valida credenciais.
4. Backend retorna JWT com referencia ao usuario e box.
5. Frontend armazena sessao.
6. Usuario acessa o dashboard do seu box.

---

### Importacao de Check-ins

1. Usuario acessa Importacoes.
2. Seleciona origem: Wellhub ou TotalPass.
3. Envia arquivo XLSX ou CSV.
4. Backend cria `ImportHistory`.
5. Parser da origem processa o arquivo.
6. Sistema cria ou atualiza alunos do box.
7. Sistema cria check-ins sem receita financeira.
8. Sistema recalcula progresso das campanhas ativas.
9. Dashboard passa a refletir os novos dados.

---

### Criacao de Campanha

1. Usuario cria campanha com nome, descricao e periodo.
2. Define metas por plataforma.
3. Associa brindes a campanha.
4. Campanha fica ativa.
5. Sistema calcula progresso dos alunos conforme check-ins existentes.

---

### Acompanhamento de Dashboard

1. Dashboard carrega indicadores principais.
2. Sistema exibe campanhas ativas.
3. Sistema lista alunos proximos da meta.
4. Sistema lista alunos em risco.
5. Sistema lista brindes pendentes.
6. Usuario decide a proxima acao operacional.

---

### Controle de Brindes

1. Aluno atinge a meta da campanha.
2. Sistema marca progresso como `achieved`.
3. Aluno aparece como elegivel.
4. Sistema cria ou apresenta pendencia de entrega.
5. Usuario marca brinde como entregue.
6. Sistema registra `delivered = true` e `delivered_at`.

---

### Disparo de WhatsApp

1. Usuario configura Twilio WhatsApp em `WhatsappSettings`.
2. Usuario cria template de mensagem.
3. Usuario cria campanha de mensagem.
4. Usuario escolhe publico:
   - `near_goal`
   - `achieved`
   - `inactive`
   - `all`
5. Sistema resolve destinatarios pelo `box_id`.
6. Sistema cria registros em `MessageRecipient`.
7. Sistema envia mensagens via Twilio WhatsApp.
8. Sistema atualiza status individual como `sent` ou `failed`.

---

### Relatorios

1. Usuario acessa Relatorios.
2. Escolhe tipo de relatorio:
   - Elegiveis
   - Brindes pendentes
   - Frequencia mensal
3. Sistema aplica filtros do box autenticado.
4. Usuario visualiza ou exporta em CSV.

---

## 6. Entidades do Dominio

### Box

Tenant principal do sistema.

### User

Usuario autenticado. No MVP, sempre `OWNER`.

### Student

Aluno importado da Wellhub ou TotalPass.

### Checkin

Registro de presenca do aluno, sem informacao financeira.

### ImportHistory

Historico de processamento de arquivos.

### Campaign

Campanha de incentivo criada pelo box.

### CampaignGoal

Meta de check-ins por plataforma.

### CampaignProgress

Progresso do aluno em uma campanha.

### Reward

Brinde de uma campanha.

### RewardDelivery

Entrega de brinde para aluno.

### WhatsappSettings

Configuracao do provedor WhatsApp por box.

### MessageTemplate

Template reutilizavel de mensagem.

### MessageCampaign

Campanha de envio de mensagens.

### MessageRecipient

Auditoria individual dos destinatarios e status de envio.

---

## 7. Casos de Uso Principais

### Autenticacao

- Login do owner
- Obter usuario atual
- Resolver box do usuario autenticado

### Box

- Consultar dados do box
- Atualizar dados do box

### Importacoes

- Importar arquivo Wellhub
- Importar arquivo TotalPass
- Listar historico de importacoes
- Consultar detalhes da importacao

### Alunos

- Criar aluno automaticamente via importacao
- Listar alunos
- Buscar alunos
- Filtrar alunos por origem, campanha e status
- Consultar detalhe do aluno
- Consultar check-ins do aluno

### Campanhas

- Criar campanha
- Editar campanha
- Encerrar campanha
- Listar campanhas
- Consultar campanha
- Definir metas por plataforma
- Recalcular progresso
- Listar elegiveis
- Listar proximos da meta

### Brindes

- Criar brinde
- Editar brinde
- Listar brindes de campanha
- Listar entregas pendentes
- Marcar entrega como realizada

### Dashboard

- Obter resumo de KPIs
- Listar campanhas ativas
- Listar alunos proximos da meta
- Listar alunos em risco
- Listar brindes pendentes

### WhatsApp

- Consultar configuracao do provedor WhatsApp
- Atualizar configuracao do provedor WhatsApp
- Testar configuracao do provedor WhatsApp
- Criar template
- Editar template
- Listar templates
- Criar campanha de mensagem
- Enviar campanha de mensagem
- Auditar destinatarios e status

### Relatorios

- Gerar relatorio de elegiveis
- Gerar relatorio de brindes pendentes
- Gerar relatorio de frequencia mensal

---

## 8. Regras Definitivas

- Sistema multi-tenant desde o inicio.
- Toda entidade operacional deve respeitar isolamento por box.
- MVP possui apenas usuario `OWNER`.
- Dashboard e a principal funcionalidade.
- WhatsApp faz parte do MVP.
- Twilio WhatsApp sera usado para WhatsApp.
- Receita financeira esta fora do MVP.
- `Checkin` nao possui campo `revenue`.
- Aluno proximo da meta: progresso maior ou igual a 80%.
- Aluno em risco: 7 dias sem check-in.
- Frontend: React, Vite, Tailwind, shadcn/ui, Lucide Icons.
- Backend: Go, Gin, GORM, PostgreSQL, Arquitetura Hexagonal.
