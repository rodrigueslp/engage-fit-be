# EngageFit - Handoff de Contexto

Atualizado em: 2026-06-30

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
- Entidades centrais: `Box`, `User`, `Student`, `Checkin`, `Campaign`, `CampaignGoal`, `Reward`, `CampaignProgress`, `RewardDelivery`, `WhatsappSettings`, `MessageTemplate`, `MessageCampaign`, `MessageRecipient`, `EmailSettings`, `EmailTemplate`, `EmailCampaign`, `EmailRecipient`, `AutomationRun`, `AutomationSchedule`.
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
- Widgets do dashboard (`Campanhas ativas`, `Próximos da meta`, `Alunos em risco`, `Brindes pendentes`) usam paginacao client-side de 5 itens por pagina.
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
- E-mail personalizado e automacao diaria com agendamento/auditoria persistida ja foram implementados. Próximos focos devem partir das pendencias funcionais abaixo, nao da mensagem antiga de handoff.

## Estado atual implementado

### Backend

Diretorio: `engage-fit-be`

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
  - `SendMessageCampaignUseCase` aplica `risk_message_cooldown_days` para audiência `inactive`.
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
- Campanhas de mensagem vinculadas a campanha de meta:
  - `message_campaigns.campaign_id` referencia `campaigns.id`.
  - migration: `migrations/022_add_message_campaign_campaign_id.sql`.
  - audiências `almost_there`, `near_goal` e `achieved` usam a campanha vinculada, nao todas as campanhas ativas.
  - template context (variaveis `campaign_name`, `reward_name`, check-ins) usa a campanha vinculada.
- Preview de mensagem antes do envio:
  - `GET /api/v1/message-campaigns/:id/preview` retorna `body`, `total`, aluno exemplo e telefone.
  - usa a mesma renderizacao do envio real (`renderTemplate` + `templateValues`).
- Evolution API removida do produto:
  - providers suportados: `twilio` (padrao) e `meta_cloud`.
  - migration: `migrations/023_remove_evolution_provider.sql` normaliza configs antigas `evolution` para `twilio`.
  - removidos `evolution_client.go`, `docker-compose.evolution.yml` e targets `make evolution-*`.
- Automacao diaria operacional:
  - script `scripts/daily-automation.mjs` e target `make daily-automation`.
  - importa arquivo opcional (`DAILY_CHECKINS_FILE`), recalcula campanhas ativas, envia campanhas de mensagem vinculadas (`DAILY_SEND_MESSAGES=true`).
- E-mail personalizado:
  - provider `smtp` com modo seguro em development e provider `mock` para testes locais.
  - settings em `GET/PUT/POST /api/v1/email/settings` e `/test`.
  - templates em `/api/v1/email-templates` com assunto, corpo e variaveis iguais as campanhas WhatsApp.
  - campanhas em `/api/v1/email-campaigns` com `campaign_id`, audiência, preview, envio manual e auditoria por destinatario.
  - envio real local fica bloqueado por padrao; usar `EMAIL_ALLOW_REAL_SEND=true` ou provider `mock`.
- Automacao diaria como feature do produto:
  - tabelas `automation_runs` e `automation_schedules`.
  - endpoints de historico: `GET/POST /api/v1/automation/runs`, `GET/PATCH /api/v1/automation/runs/:id`.
  - endpoints de rotinas: `GET/POST /api/v1/automation/schedules`, `PUT/DELETE /api/v1/automation/schedules/:id`, `POST /api/v1/automation/schedules/:id/run`.
  - tela `Automacao` permite criar rotina com horario, dias da semana, modo, reenvio e status ativo/pausado.
  - modos: `full_daily`, `recalculate_only`, `send_almost_there`, `send_achieved`, `send_inactive`.
  - worker interno roda agendas quando `AUTOMATION_WORKER_ENABLED=true`; intervalo configuravel por `AUTOMATION_WORKER_INTERVAL_SECONDS`.
  - `scripts/daily-automation.mjs` permanece como alternativa operacional/CI e tambem registra `automation_runs`.

Validacao atual:

```bash
cd engage-fit-be
go test ./...
```

Passando.

### Frontend

Diretorio: `engage-fit-fe`

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
  - editar campanha (nome, descricao, datas)
  - editar metas Wellhub/TotalPass
  - editar brinde (nome, descricao, quantidade)
  - encerrar e reativar campanha
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
  - campanhas de mensagem vinculadas a uma campanha de meta (`campaign_id`)
  - variaveis de template
  - preview renderizado da mensagem antes do envio (aluno exemplo + total de destinatarios)
  - botao enviar/reenviar campanha
  - retorno visual de `sent/total/failed` apos envio
  - auditoria do ultimo envio por destinatario, incluindo `error_message` da Twilio
- E-mail:
  - tela dedicada no menu lateral.
  - configuracao SMTP/mock com teste de credenciais.
  - templates com assunto/corpo e variaveis operacionais.
  - campanhas vinculadas a campanha de meta, preview renderizado, envio/reenviar e auditoria do ultimo envio.
- Automacao:
  - tela dedicada no menu lateral.
  - cria/pausa/remove rotinas automaticas com horario, dias, modo e permissao de reenvio.
  - botao `Executar` permite rodar uma rotina manualmente.
  - historico de execucoes diarias com status, importacao, campanhas recalculadas, mensagens enviadas, falhas e erros.
- Dashboard operacional:
  - atalhos para Brindes, Relatorios, WhatsApp e Automacao.
- Configuracoes:
  - Card `Regras de risco`:
    - Aluno em risco apos X dias sem check-in.
    - Reenviar mensagem de risco apos X dias.
  - formulario de provedor WhatsApp
  - Twilio WhatsApp como opcao comercial recomendada
  - Meta Cloud API mantida como opcao avancada/futura
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
cd engage-fit-fe
npm ci
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Passando.

Observacao: `engage-fit-fe/.npmrc` aponta ao registry publico (`registry.npmjs.org`). O `package-lock.json` foi corrigido para nao usar o registry privado Fury Cloud.

## Comandos principais

Subir banco do EngageFit:

```bash
make up
make migrate-up
```

Rodar backend:

```bash
cd engage-fit-be
make backend-run
```

`make backend-run` exporta `DATABASE_URL` automaticamente. Alternativa manual:

```bash
cp .env.example .env
go run ./cmd/api
```

Rodar frontend:

```bash
cd engage-fit-fe
npm install
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
- Cria templates e campanhas de mensagem vinculadas a campanha do mes para:
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
2. Enviar a campanha `Disparo teste - falta pouco` para validar a audiência dinamica.
3. Enviar a campanha `Disparo teste - aluno em risco` para validar `inactive`; Marina Risco deve virar `observing` e nao receber novo envio ate passar o cooldown configurado em `Configuracoes > Regras de risco`.
4. Importar `test-data/totalpass-checkins-hit-goal.csv` ou `test-data/totalpass-checkins-23-06-2026.xlsx` na tela de importacoes com fonte `TotalPass`.
5. Luiz, Deborah e Bruno ficam com 10/10 check-ins.
6. Configurar Twilio em `Configuracoes`, se ainda nao estiver configurado.
   - Para sandbox, usar remetente `+14155238886` ou `whatsapp:+14155238886`.
   - O WhatsApp do Luiz precisa entrar no sandbox enviando `join science-everyone` para `+1 415 523 8886` antes de receber mensagens.
7. Enviar ou reenviar a campanha `Disparo teste - meta atingida` na tela `WhatsApp`.

## WhatsApp comercial

Direcao atual: Twilio WhatsApp e o caminho principal para o MVP comercial.

Implementado nesta etapa:

- Provider `twilio` no backend.
- Cliente `TwilioClient` em `backend/internal/adapters/whatsapp/twilio_client.go`.
- `Content SID` em templates de mensagem.
- Provider `meta_cloud` tambem existe como opcao avancada/futura.
- Cliente `MetaCloudClient` em `backend/internal/adapters/whatsapp/meta_cloud_client.go`.
- Gateway roteador por provider em `backend/internal/adapters/whatsapp/provider_gateway.go`.
- Frontend de configuracoes permite escolher `Twilio WhatsApp` ou `Meta Cloud API`.
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

- Regra do WhatsApp/Twilio: mensagem livre so e confiavel dentro da janela de conversa de 24h apos o destinatario responder/iniciar conversa. Fora disso, usar template aprovado com `Content SID` (`HX...`).
- Sandbox: manter comportamento de teste atual, sem bloquear envio quando `content_sid` estiver vazio; o backend tenta enviar texto livre e registra o erro da Twilio se ela recusar (ex.: `63016`).
- Producao: validar com cliente antes de implementar. Direcao proposta: cada cliente cria modelos de mensagem usando apenas variaveis liberadas pelo sistema (`{{nome_aluno}}`, `{{nome_academia}}`, campanha, brinde etc.). O sistema converte isso para um template Twilio/WhatsApp e envia para aprovacao.
- Aprovacao pode ser operacional via Content Template Builder no console da Twilio no inicio, ou automatizada depois via Twilio Content API. A API cria o content template, dispara/acompanha aprovacao para WhatsApp e retorna/expoe o `Content SID` aprovado.
- Depois de aprovado, salvar `content_sid`, status de aprovacao, idioma e mapeamento de variaveis por tenant/template.
- Para envio comercial fora da janela de 24h, enviar via Twilio usando `ContentSid` e `ContentVariables`, nao `Body` livre.
- Remetente por cliente: possivel, mas cada tenant precisa ter seu proprio WhatsApp Sender/numero aprovado na Twilio/WABA ou uma configuracao equivalente. O backend deve escolher o remetente e credenciais pelo tenant.
- Garantir que as variaveis do template Twilio estejam na ordem esperada pelo provider.
- Decisao adiada: validar com cliente se aceitara fluxo de templates aprovados antes de implementar onboarding/aprovacao em producao.

Validacao apos mudanca:

```bash
cd backend
go test ./...

cd frontend
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Ambos passaram.

Automacao diaria:

```bash
cd engage-fit-be
DAILY_CHECKINS_SOURCE=totalpass \
DAILY_CHECKINS_FILE=test-data/totalpass-checkins-hit-goal.csv \
DAILY_SEND_MESSAGES=true \
make daily-automation
```

## Ideia futura: treino do dia com mensagens geradas por LLM

Feature proposta para evolucao futura: automacao inteligente de mensagens do treino do dia, especialmente para boxes de CrossFit.

Objetivo:

- Permitir que o box cadastre ou importe o treino/WOD do dia.
- Gerar mensagens personalizadas para alunos pelo WhatsApp com dicas contextuais do treino.
- Aumentar comparecimento, engajamento diario e percepcao de acompanhamento individual.
- Transformar o EngageFit de uma plataforma que reage a check-ins/metas em uma ferramenta que tambem ativa alunos antes do treino.

Exemplo de uso:

```txt
Fulano, hoje tem PR de back squat no CrossFit Alados.
Chegue alguns minutos antes para aquecer bem quadril, tornozelos e posterior.
Na hora das tentativas, priorize tecnica, respiracao e profundidade consistente.
Se estiver em duvida sobre carga, chama o coach antes de subir o peso.
Bora buscar esse PR com seguranca.
```

Desenho sugerido para MVP:

- Nova area `Treino do dia` ou `WOD`:
  - data.
  - titulo.
  - movimentos principais.
  - objetivo do treino (`forca`, `metcon`, `ginastico`, `cardio`, `PR`, `benchmark`, etc.).
  - observacoes do coach.
- Botao `Gerar mensagens` usando LLM.
- Gerar variacoes por publico:
  - todos os alunos.
  - alunos em risco.
  - alunos proximos da meta/check-ins.
  - alunos frequentes.
- Preview obrigatorio antes do envio, pelo menos na primeira versao.
- Reutilizar infraestrutura atual de WhatsApp:
  - templates/campanhas.
  - preview renderizado.
  - auditoria por destinatario.
  - automacao/agendamento no futuro.

Cuidados de produto e compliance:

- Nao vender como "IA livre mandando qualquer coisa"; posicionar como `automacao inteligente do treino do dia`.
- LLM deve gerar texto dentro de guardrails:
  - mensagens curtas.
  - tom motivacional e pratico.
  - sem prescricao individual de dieta.
  - sem orientacao medica.
  - sem prometer resultado fisico.
  - dicas tecnicas gerais e seguras.
  - recomendar falar com o coach em caso de duvida sobre carga, dor ou adaptacao.
- Para nutricao, usar apenas orientacao generica, por exemplo: chegar bem alimentado/hidratado e seguir plano de nutricionista quando houver.
- Para tecnica, usar orientacoes seguras e gerais, nunca diagnostico individual.
- Considerar aprovacao manual pelo coach/owner antes do envio no MVP.

Restricao importante de WhatsApp:

- Fora da janela de conversa de 24h, mensagem livre gerada por LLM pode falhar ou nao ser permitida comercialmente.
- Caminho seguro:
  - usar template aprovado para iniciar conversa, por exemplo: `Oi {{1}}, hoje tem treino importante no {{2}}. Quer receber dicas rapidas para mandar bem? Responda QUERO.`
  - apos resposta do aluno, enviar a mensagem livre dentro da janela de 24h.
  - alternativa: criar templates aprovados por categoria de treino com variaveis controladas.
- Integracao futura deve respeitar `ContentSid`/templates aprovados para producao.

Possiveis entidades futuras:

- `Workout`
- `WorkoutMovement`
- `WorkoutMessageDraft`
- `WorkoutMessageCampaign`
- `LLMGenerationLog`

Possiveis configuracoes por box:

- tom da comunicacao.
- tamanho maximo da mensagem.
- assinatura/padrao de encerramento.
- habilitar/desabilitar mencoes de nutricao generica.
- aprovacao obrigatoria antes de envio.
- horarios permitidos de envio.

## Pendencias funcionais

Prioridade alta:

1. Relatorios avancados e filtros server-side quando o volume crescer.
2. Operacionalizar automacao em ambiente real:
   - decidir se o worker interno (`AUTOMATION_WORKER_ENABLED=true`) sera o caminho principal em producao ou se havera cron/CI externo como fallback.
   - definir fonte automatica dos check-ins do dia anterior.
   - configurar observabilidade/alertas para `automation_runs` com falha.
3. Avaliar o sistema inteiro e padronizar logs/telemetria pensando em observabilidade futura:
   - definir convencao unica de logs estruturados por camada (HTTP, use cases, gateways externos, jobs/worker, scripts operacionais).
   - garantir `request_id`/correlation id atravessando handlers, casos de uso e chamadas externas.
   - padronizar campos: tenant/box_id, usuario quando aplicavel, entidade, operacao, status, latencia, erro bruto e erro normalizado.
   - definir niveis (`debug`, `info`, `warn`, `error`) e evitar vazamento de segredos/PII sensivel.
   - preparar caminho para coletor futuro (ex.: OpenTelemetry, Loki, Datadog ou similar), metricas e alertas.
4. Lidar melhor com multiplas campanhas ativas simultaneas na operacao apos testes com clientes reais.

Prioridade media:

1. Renomear a entidade `Box` no dominio/UI para um conceito mais generico antes de expandir para academias convencionais. O sistema nasceu focado em boxes de CrossFit, mas o posicionamento atual e fitness/academias em geral; avaliar nomes como `Academy`, `Gym`, `Business`, `Unit` ou `Organization`. A migracao deve preservar `box_id` ou planejar renomeacao gradual de banco/API/frontend sem quebrar dados existentes.
2. Testes automatizados para use cases de relatorio, controle de brindes, mensagens e e-mail.
3. Relatorios avancados:
   - filtros server-side/paginacao se volume crescer.
   - historico de brindes entregues por periodo.
   - relatorio de conversao de mensagens por campanha.
4. UX mobile/responsiva para tabelas grandes.
5. Testes de integracao de repositories contra PostgreSQL real.
6. Refinar Dashboard com atalhos conforme feedback de uso.
7. Melhorar seed/demo com mais cenarios de conversao de mensagens e e-mail.
8. Avaliar e implementar `Treino do dia` com mensagens geradas por LLM, respeitando guardrails e restricoes comerciais do WhatsApp.

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

- `engage-fit-be/cmd/api/main.go`
- `engage-fit-be/internal/app/imports/import_checkins_usecase.go`
- `engage-fit-be/internal/app/messages/message_usecases.go`
- `engage-fit-be/internal/app/email/email_usecases.go`
- `engage-fit-be/internal/app/automation/automation_usecases.go`
- `engage-fit-be/internal/app/automation/worker.go`
- `engage-fit-be/internal/adapters/whatsapp/twilio_client.go`
- `engage-fit-be/internal/adapters/whatsapp/provider_gateway.go`
- `engage-fit-be/internal/adapters/whatsapp/safe_gateway.go`
- `engage-fit-be/internal/adapters/email/smtp_gateway.go`
- `engage-fit-be/migrations/022_add_message_campaign_campaign_id.sql`
- `engage-fit-be/migrations/023_remove_evolution_provider.sql`
- `engage-fit-be/migrations/024_create_email_and_automation.sql`
- `engage-fit-be/migrations/025_create_automation_schedules.sql`

Frontend:

- `engage-fit-fe/src/pages/campaigns/CampaignsPage.tsx`
- `engage-fit-fe/src/pages/whatsapp/WhatsappPage.tsx`
- `engage-fit-fe/src/pages/email/EmailPage.tsx`
- `engage-fit-fe/src/pages/automation/AutomationPage.tsx`
- `engage-fit-fe/src/pages/settings/SettingsPage.tsx`
- `engage-fit-fe/src/features/api/endpoints.ts`
- `engage-fit-fe/.npmrc`

Infra/dev:

- `engage-fit-be/Makefile` (`docker compose`, `make backend-run` com `DATABASE_URL`, `make daily-automation`)
- `engage-fit-be/docker-compose.yml`
- `engage-fit-be/scripts/demo-seed.mjs`
- `engage-fit-be/scripts/daily-automation.mjs`
- `engage-fit-be/migrations/024_create_email_and_automation.sql`
- `engage-fit-be/migrations/025_create_automation_schedules.sql`
- `engage-fit-be/test-data/totalpass-checkins-hit-goal.csv`

## Orientacao para iniciar novo chat

Mensagem sugerida:

```txt
Leia `.ai/handoff.md` e continue a partir dele. WhatsApp ja tem preview renderizado, campanhas de mensagem vinculadas a `campaign_id`, edicao de campanhas/metas/brindes na UI, Evolution API removida (Twilio + Meta Cloud), e-mail personalizado implementado, e automacao diaria implementada com `automation_runs`, `automation_schedules`, worker interno e alternativa operacional via `make daily-automation`. Próximo foco sugerido: operacionalizar automacao em ambiente real, definir fonte automatica de check-ins do dia anterior, adicionar alertas para falhas e evoluir relatorios/filtros server-side.
```
