# EngageFit - Handoff de Contexto

Atualizado em: 2026-06-26

## Papel do projeto

EngageFit e um sistema multi-tenant para boxes de CrossFit acompanharem check-ins de alunos vindos de Wellhub e TotalPass, calcularem metas mensais por plataforma, identificarem alunos proximos/aptos ao brinde e dispararem mensagens personalizadas. O foco inicial segue em boxes de CrossFit, mas a marca e o posicionamento deixam espaco para academias no geral.

## Stack aprovada

Backend:

- Go
- Gin
- GORM
- PostgreSQL
- Arquitetura Hexagonal

Frontend:

- React
- Vite
- TypeScript
- Tailwind
- componentes locais no padrao shadcn/ui
- Lucide Icons

## Decisoes principais

- Sistema multi-tenant desde o inicio.
- Entidades centrais: `Box`, `User`, `Student`, `Checkin`, `Campaign`, `CampaignGoal`, `Reward`, `CampaignProgress`, `RewardDelivery`, `WhatsappSettings`, `MessageTemplate`, `MessageCampaign`, `MessageRecipient`.
- MVP tem apenas perfil `OWNER`.
- Sem receita financeira no MVP.
- `Checkin` nao possui `revenue`.
- Aluno proximo da meta: atingiu pelo menos 80% da meta da plataforma.
- Aluno em risco: quantidade configuravel de dias sem check-in por box, default `7`.
- Dashboard e funcionalidade principal.
- Card `Brindes pendentes` no dashboard e apenas resumo operacional; a baixa de entrega fica na tela dedicada `Brindes`.
- Controle de brindes no MVP e baseado em `Reward.quantity` + `RewardDelivery`:
  - pendente quando aluno bate meta e ainda nao recebeu.
  - entregue quando usuario marca manualmente.
  - disponivel calculado como `quantity - delivered_deliveries`.
  - pendencias nao descontam estoque real ate serem entregues.
- Widgets do dashboard (`Campanhas ativas`, `Proximos da meta`, `Alunos em risco`, `Brindes pendentes`) usam paginacao client-side de 5 itens por pagina.
- Alunos em risco agora possuem acompanhamento:
  - `risk_status`: `active`, `observing`, `paused`, `not_interested`.
  - `risk_last_message_at`: ultima mensagem de risco enviada.
  - migration: `migrations/020_add_student_risk_tracking.sql`.
  - endpoint `PATCH /api/v1/students/:id/risk-status` permite mudar o status manualmente.
  - campanha de mensagem `inactive` nao envia para `paused`/`not_interested` e respeita cooldown configuravel apos a ultima mensagem de risco.
  - quando uma mensagem `inactive` e enviada com sucesso, o aluno passa para `observing`.
- Regras de risco configuraveis por box:
  - `boxes.risk_inactive_days`, default `7`.
  - `boxes.risk_message_cooldown_days`, default `14`.
  - migration: `migrations/021_add_box_risk_settings.sql`.
  - tela `Configuracoes > Regras de risco` permite alterar os dois valores.
- WhatsApp faz parte do MVP.
- Relatorios essenciais do MVP implementados:
  - elegiveis.
  - brindes pendentes.
  - frequencia mensal.
  - todos com visualizacao na UI, filtros client-side e exportacao CSV.
- Navegacao frontend usa hash (`#dashboard`, `#campaigns`, `#rewards`, etc.) para preservar a tela apos refresh. Logo da sidebar navega para `Dashboard`.

## Estado atual implementado

### Backend

Diretorio: `backend`

Implementado:

- Auth com JWT/bcrypt.
- Bootstrap owner via `POST /api/v1/setup/owner`.
- Multi-tenant por `box_id`.
- CRUD base de campanhas.
- Metas por plataforma em `campaign_goals`.
- Brindes em `rewards`.
- Importacao de check-ins CSV/XLSX.
- Parser ajustado para planilhas reais:
  - Wellhub com preambulo, `Data`, `Hora`, `Visitante`, `ID do Wellhub`.
  - TotalPass tokens com preambulo, `ID`, `Colaborador`, `Validado em`.
- Deduplicacao de check-ins:
  - chave: `box_id + source + student_id + checkin_date + checkin_time`
  - migration: `migrations/016_add_unique_checkins.sql`
  - `SaveMany` usa `ON CONFLICT DO NOTHING`
- Recalculo de progresso de campanha.
- WhatsApp settings.
- Templates de WhatsApp com variaveis:
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
- WhatsApp mock com `mock://local`.
- WhatsApp com provider configuravel:
  - `twilio` para Twilio WhatsApp, caminho comercial recomendado.
  - `meta_cloud` para Meta Cloud API oficial.
  - `evolution` mantido como legado/local.
  - migration: `migrations/017_add_whatsapp_provider.sql`
  - `instance_name` passa a representar `Phone number ID` quando provider for `meta_cloud`.
  - `instance_name` passa a representar `whatsapp:+...` ou `Messaging Service SID` quando provider for `twilio`.
  - `base_url` pode ficar vazio para Meta Cloud API; o backend usa `https://graph.facebook.com/v20.0`.
  - `base_url` pode ficar vazio para Twilio; o backend usa `https://api.twilio.com`.
- Templates de WhatsApp agora aceitam `content_sid`:
  - migration: `migrations/018_add_message_template_content_sid.sql`
  - usado pelo provider `twilio` como `ContentSid`.
  - variaveis Twilio enviadas como `ContentVariables`: `1=name`, `2=box_name`, `3=current_checkins`, `4=remaining_checkins`, `5=target_checkins`, `6=reward_name`, `7=platform`.
  - bug corrigido em `MessageTemplateModel`: `ContentSID` precisa de tag `gorm:"column:content_sid"`; sem isso o GORM tentava inserir em `content_s_id`.
- `SafeGateway`:
  - em `development`, envio real fica bloqueado por padrao.
  - para envio real local precisa:
    - `WHATSAPP_ALLOW_REAL_SEND=true`
    - `WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES=5511963834712,5518997980429` para permitir envio apenas para uma lista fechada.
    - ou `WHATSAPP_DEV_RECIPIENT_PHONE=55DDDNUMERO` para redirecionar tudo para um unico telefone.
  - quando `WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES` estiver preenchido, em development o backend envia para os proprios numeros permitidos e bloqueia qualquer outro destinatario.
- Teste real Twilio Sandbox validado:
  - Sandbox ativado no console da Twilio.
  - Numero de teste fez join enviando `join science-everyone` para `+1 415 523 8886`.
  - Remetente configurado no EngageFit como sandbox Twilio (`+14155238886` ou `whatsapp:+14155238886`).
  - Mensagens foram enviadas e recebidas no WhatsApp com sucesso para o numero do Luiz.
  - Erros anteriores resolvidos:
    - `63007 Twilio could not find a Channel with the specified From address`: sandbox ainda nao estava ativado/configurado.
    - `63031 Message cannot have the same To and From`: remetente estava configurado igual ao destinatario em tentativa anterior.
- Acompanhamento de alunos em risco:
  - `students` tem `risk_status` e `risk_last_message_at`.
  - `boxes` tem `risk_inactive_days` e `risk_message_cooldown_days`.
  - `PATCH /api/v1/students/:id/risk-status` atualiza status manual.
  - `Dashboard` usa `risk_inactive_days` para classificar aluno em risco.
  - `SendMessageCampaignUseCase` aplica `risk_message_cooldown_days` para audiencia `inactive`.
  - envio `inactive` bem-sucedido marca aluno como `observing` e grava `risk_last_message_at`.
- Controle de brindes:
  - `GET /api/v1/rewards/deliveries` lista entregas pendentes e entregues com campanha, aluno, telefone, brinde e status.
  - `GET /api/v1/rewards/pending-deliveries` lista apenas pendencias.
  - `PATCH /api/v1/reward-deliveries/:id/deliver` marca entrega como entregue com filtro por `box_id`.
  - `GET /api/v1/campaigns/:id/rewards` retorna contadores por brinde: `pending_deliveries`, `delivered_deliveries`, `available_quantity`.
- Relatorios:
  - `GET /api/v1/reports/eligible-students`
  - `GET /api/v1/reports/pending-rewards`
  - `GET /api/v1/reports/monthly-frequency?month=YYYY-MM`
  - os tres endpoints aceitam `?format=csv`.
  - `eligible-students` usa `campaign_progresses` com joins em `campaigns`, `students` e `rewards`.
  - `pending-rewards` usa `reward_deliveries` pendentes enriquecidas com campanha/aluno/brinde.
  - `monthly-frequency` agrupa check-ins por aluno no periodo mensal.

Validacao atual:

```bash
cd backend
go test ./...
```

Passando.

### Frontend

Diretorio: `frontend`

Implementado:

- Login.
- Layout com sidebar/header.
- Marca atual: `EngageFit`.
  - Logo gerada e integrada em `frontend/public/engagefit-logo.png`.
  - Versao recortada sem margens em `frontend/public/engagefit-logo-cropped.png`.
  - Sidebar, login e favicon usam a versao recortada.
- Dashboard inicial.
  - Widgets com paginacao client-side de 5 itens.
  - `Brindes pendentes` mostra nome do aluno, nome do brinde e telefone como resumo; baixa operacional fica em `Brindes`.
  - `Alunos em risco` mostra ultima mensagem de risco, status de acompanhamento e permite alterar status manualmente.
- Campanhas com fluxo operacional:
  - criar campanha
  - meta Wellhub
  - meta TotalPass
  - brinde
  - indicadores de brinde por campanha: total, disponiveis, pendentes e entregues.
  - painel com progresso, faltantes, proximos e atingidos
  - botao recalcular
- Brindes:
  - tela dedicada no menu lateral.
  - busca por campanha, aluno, telefone e brinde.
  - filtro por status: pendentes, todas, entregues.
  - baixa manual com botao `Marcar entregue`.
  - historico de entregas permanece visivel em `Todas`/`Entregues`.
- Alunos.
- Importacoes.
- Relatorios:
  - tela dedicada no menu lateral.
  - relatorio de elegiveis com filtros por busca, campanha e plataforma.
  - relatorio de brindes pendentes com filtros por busca e campanha.
  - relatorio de frequencia mensal com filtro de mes, busca e plataforma.
  - CSV exporta o recorte filtrado na tela.
- Navegacao:
  - hash da URL preserva pagina atual apos refresh.
  - logo da sidebar volta para o dashboard.
  - icones diferenciados: `Campanhas` usa `Target`, `Brindes` usa `Gift`.
- WhatsApp:
  - templates
  - campanhas de mensagem
  - variaveis de template
  - botao enviar/reenviar campanha
  - retorno visual de `sent/total/failed` apos envio
  - auditoria do ultimo envio por destinatario, incluindo `error_message` da Twilio
- Configuracoes:
  - Card `Regras de risco`:
    - Aluno em risco apos X dias sem check-in.
    - Reenviar mensagem de risco apos X dias.
  - formulario de provedor WhatsApp
  - Twilio WhatsApp como opcao comercial recomendada
  - Meta Cloud API mantida como opcao avancada/futura
  - Evolution API mantida como opcao legado/desenvolvimento
  - Base URL
  - Phone number ID / Instancia
  - Access token / API key
  - Ativar WhatsApp
  - Testar conexao
  - Mostra se ha credencial salva e quando foi a ultima atualizacao.
  - Campo de credencial fica vazio por seguranca, mas salvar com ele vazio preserva a credencial ja cadastrada.
  - Botao de teste usa os dados atuais do formulario quando ha alteracoes nao salvas.

Validacao atual:

```bash
cd frontend
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Passando.

Observacao: nesta maquina, `npm run build` falhou por wrapper local quebrado em `node_modules/.bin/tsc` (`Cannot find module '../lib/tsc.js`). A validacao direta por TypeScript + Vite passou.

## Comandos principais

Subir banco do EngageFit:

```bash
make up
make migrate-up
```

Rodar backend:

```bash
cd backend
go run cmd/api/main.go
```

Rodar frontend:

```bash
cd frontend
npm run dev
```

Frontend local:

```txt
http://localhost:5173
```

Credenciais demo:

```txt
owner@example.com
change-me
```

Seed demo:

```bash
make demo-seed
```

Reset + seed demo:

```bash
make demo-reset-seed
```

O seed demo atual foi ajustado para teste controlado de WhatsApp:

- Cria campanha ativa com meta TotalPass `10`.
- Cria alunos/cenarios usando o telefone do Luiz (`5511963834712`):
  - Luiz: `9/10`, entra em `almost_there` (`Falta pouco`).
  - Deborah: `8/10`, entra em `almost_there` se ainda houver pelo menos 2 dias restantes na campanha.
  - Bruno Teste: `7/10`, fica fora de `almost_there` por estar abaixo de 80%.
  - Carla Teste: `10/10`, entra em `achieved` (`Meta atingida`).
  - Marina Risco: `3/10`, entra em `inactive` (`Aluno em risco`) por estar ha mais de 7 dias sem check-in.
- Cria templates e campanhas de mensagem para:
  - `almost_there` (`Disparo teste - falta pouco`)
  - `achieved` (`Disparo teste - meta atingida`)
  - `inactive` (`Disparo teste - aluno em risco`)
- Audience `almost_there` foi adicionada a constraint de `message_campaigns` pela migration `migrations/019_add_almost_there_message_audience.sql`.
- Nao configura WhatsApp mock e nao envia automaticamente, para nao sobrescrever configuracao Twilio real.
- `make demo-reset` preserva `boxes`, `users` e `whatsapp_settings`, entao a configuracao Twilio cadastrada pela UI nao precisa ser refeita a cada `make demo-reset-seed`.
- Validado em 2026-06-26: apos aplicar a migration `019`, `make demo-reset-seed` passou e o envio da campanha `Disparo teste - falta pouco` funcionou.
- Validado em 2026-06-26: migration `020` aplicada no banco local; `go test ./...`, `node --check scripts/demo-seed.mjs` e build frontend direto passaram apos acompanhamento de risco.
- Validado em 2026-06-26: migration `021` aplicada no banco local; `go test ./...` e build frontend direto passaram apos regras de risco configuraveis por box.
- Validado em 2026-06-26: controle de brindes, tela dedicada de brindes, relatorios essenciais, navegacao por hash e build frontend passaram com:
  - `cd backend && go test ./...`
  - `cd frontend && node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build`
  - smoke test autenticado dos endpoints de relatorio retornou `200`.

Planilha/CSV incremental para bater a meta:

- `test-data/totalpass-checkins-hit-goal.csv`
- `test-data/totalpass-checkins-23-06-2026.xlsx`

Ambos contem:

- Luiz recebe +1 check-in.
- Deborah recebe +2 check-ins.
- Bruno Teste recebe +3 check-ins.
- Apos importar um destes arquivos, Luiz, Deborah e Bruno ficam com `10/10`.

Fluxo de teste:

1. `make demo-reset-seed`
2. Enviar a campanha `Disparo teste - falta pouco` para validar a audiencia dinamica.
3. Enviar a campanha `Disparo teste - aluno em risco` para validar `inactive`; Marina Risco deve virar `observing` e nao receber novo envio ate passar o cooldown configurado em `Configuracoes > Regras de risco`.
4. Importar `test-data/totalpass-checkins-hit-goal.csv` ou `test-data/totalpass-checkins-23-06-2026.xlsx` na tela de importacoes com fonte `TotalPass`.
5. Luiz, Deborah e Bruno ficam com 10/10 check-ins.
6. Configurar Twilio em `Configuracoes`, se ainda nao estiver configurado.
   - Para sandbox, usar remetente `+14155238886` ou `whatsapp:+14155238886`.
   - O WhatsApp do Luiz precisa entrar no sandbox enviando `join science-everyone` para `+1 415 523 8886` antes de receber mensagens.
7. Enviar ou reenviar a campanha `Disparo teste - meta atingida` na tela `WhatsApp`.

## WhatsApp comercial

Direcao atual: para MVP comercial, substituir o caminho principal de QR Code/Evolution API por Twilio WhatsApp.

Implementado nesta etapa:

- Provider `twilio` no backend.
- Cliente `TwilioClient` em `backend/internal/adapters/whatsapp/twilio_client.go`.
- `Content SID` em templates de mensagem.
- Provider `meta_cloud` tambem existe como opcao avancada/futura.
- Cliente `MetaCloudClient` em `backend/internal/adapters/whatsapp/meta_cloud_client.go`.
- Gateway roteador por provider em `backend/internal/adapters/whatsapp/provider_gateway.go`.
- Frontend de configuracoes permite escolher `Twilio WhatsApp`, `Meta Cloud API` ou `Evolution API`.
- Frontend de WhatsApp permite enviar/reenviar campanhas de mensagem.
- README documenta configuracao comercial.

Configuracao esperada:

```txt
Provedor: Twilio WhatsApp
Base URL: https://api.twilio.com
Remetente WhatsApp ou Messaging Service SID: whatsapp:+14155238886 ou MG...
Account SID:Auth Token: AC...:<auth-token>
Ativar WhatsApp: marcado
```

Para disparo comercial em massa:

- Criar/aprovar templates na Twilio.
- Preencher `Content SID` (`HX...`) no template do EngageFit.
- Garantir que as variaveis do template Twilio estejam na ordem documentada no README.

Validacao apos mudanca:

```bash
cd backend
go test ./...

cd frontend
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Ambos passaram.

## Evolution API local

Foi criado:

- `docker-compose.evolution.yml`
- targets:
  - `make evolution-up`
  - `make evolution-down`
  - `make evolution-logs`
  - `make evolution-ps`

Config local:

```txt
Evolution API: http://localhost:8081
API key: boxengage-local-key
Instancia tentada: crossfit-alados
```

Subir:

```bash
make evolution-up
```

Logs:

```bash
make evolution-logs
```

Criar instancia tentado:

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

Resposta recebida:

```json
{
  "instance": {
    "instanceName": "crossfit-alados",
    "integration": "WHATSAPP-BAILEYS",
    "status": "connecting"
  },
  "qrcode": {
    "count": 0
  }
}
```

`connectionState` retorna:

```json
{
  "instance": {
    "instanceName": "crossfit-alados",
    "state": "connecting"
  }
}
```

Problema atual:

- A Evolution sobe corretamente.
- Migrations da Evolution aplicam com sucesso.
- Instancia e criada.
- Mas QR Code nao e gerado: sempre retorna `{ "count": 0 }`.
- Tentativa de `/instance/connect/crossfit-alados` tambem retorna `{ "count": 0 }`.
- Tentativa sugerida de pareamento por numero ainda nao resolveu.

Hipoteses para o proximo chat:

- A imagem/config atual da Evolution API v2.2.3 pode exigir outra variavel/env para QR/pairing.
- Pode ser necessario usar outro endpoint de QR/pairing da versao exata.
- Pode ser melhor subir uma versao diferente da Evolution ou usar painel/manager oficial.
- Pode ser preciso limpar volumes:

```bash
docker-compose -f docker-compose.evolution.yml down -v
make evolution-up
```

Mas isso ainda nao resolveu no chat anterior.

## Pendencias funcionais

Prioridade alta:

1. Resolver conexao real da Evolution API local.
2. Adicionar na UI de WhatsApp:
   - preview da mensagem renderizada
3. Associar `MessageCampaign` a uma campanha de meta especifica, em vez de resolver por todas as campanhas ativas.
4. Melhorar gestao de campanhas:
   - editar campanha/metas/brindes pela UI.
   - encerrar/reativar campanha pela UI.
   - lidar melhor com multiplas campanhas ativas ao mesmo tempo.
5. Implementar e-mail personalizado.
6. Implementar automacao diaria:
   - importar check-ins do dia anterior
   - recalcular campanhas
   - enviar WhatsApp/e-mail
   - auditar resultados

Prioridade media:

1. Relatorios avancados:
   - filtros server-side/paginacao se volume crescer.
   - historico de brindes entregues por periodo.
   - relatorio de conversao de mensagens por campanha.
2. UX mobile/responsiva para tabelas grandes.
3. Testes automatizados para use cases de relatorio e controle de brindes.
4. Dashboard com atalhos para telas operacionais (`Brindes`, `Relatorios`, `WhatsApp`).
5. Melhorar seed/demo para cobrir multiplas campanhas com brindes simultaneos.

## Arquivos importantes

Docs:

- `.ai/product-vision.md`
- `.ai/domain.md`
- `.ai/database.md`
- `.ai/architecture.md`
- `.ai/features.md`
- `.ai/tasks.md`
- `.ai/decisions.md`
- `.ai/ui-ux.md`
- `.ai/implementation-plan.md`
- `.ai/handoff.md`

Backend:

- `backend/cmd/api/main.go`
- `backend/internal/app/imports/import_checkins_usecase.go`
- `backend/internal/adapters/parsers/parser.go`
- `backend/internal/adapters/persistence/postgres/repositories/checkin_repository.go`
- `backend/internal/app/messages/message_usecases.go`
- `backend/internal/app/reports/report_usecases.go`
- `backend/internal/adapters/whatsapp/evolution_client.go`
- `backend/internal/adapters/whatsapp/twilio_client.go`
- `backend/internal/adapters/whatsapp/provider_gateway.go`
- `backend/internal/adapters/whatsapp/safe_gateway.go`
- `backend/internal/config/config.go`
- `backend/internal/adapters/http/handlers/reports_handler.go`
- `backend/internal/adapters/persistence/postgres/repositories/campaign_repository.go`
- `backend/internal/adapters/persistence/postgres/repositories/reward_repository.go`
- `migrations/016_add_unique_checkins.sql`

Frontend:

- `frontend/src/pages/campaigns/CampaignsPage.tsx`
- `frontend/src/pages/rewards/RewardsPage.tsx`
- `frontend/src/pages/reports/ReportsPage.tsx`
- `frontend/src/pages/imports/ImportsPage.tsx`
- `frontend/src/pages/whatsapp/WhatsappPage.tsx`
- `frontend/src/pages/settings/SettingsPage.tsx`
- `frontend/src/features/api/endpoints.ts`

Infra/dev:

- `Makefile`
- `docker-compose.yml`
- `docker-compose.evolution.yml`
- `scripts/demo-seed.mjs`
- `test-data/README.md`
- `test-data/totalpass-checkins-hit-goal.csv`

## Orientacao para iniciar novo chat

Mensagem sugerida:

```txt
Leia `.ai/handoff.md` e continue a partir dele. O estado atual do EngageFit esta documentado ali. O seed controlado cria a audiencia dinamica `almost_there` (`Falta pouco`) com Luiz 9/10, Deborah 8/10, Bruno 7/10 e Carla 10/10, todos usando o telefone do Luiz (`5511963834712`). Importar `test-data/totalpass-checkins-hit-goal.csv` ou `test-data/totalpass-checkins-23-06-2026.xlsx` como TotalPass leva Luiz, Deborah e Bruno a 10/10 para testar `Meta atingida`. Controle de brindes ja tem tela dedicada com baixa manual e relatorios essenciais ja existem com filtros e CSV. O proximo foco sugerido e melhorar a UI de WhatsApp com preview de mensagem e associar `MessageCampaign` a uma campanha de meta especifica.
```
