# Checklist de prontidao da aplicacao

Atualizado em: 2026-07-21

## Objetivo e criterio de conclusao

Deixar backend e frontend prontos para a preparacao do deploy no Railway, com comportamento seguro em production, operacao diagnosticavel e fluxos principais cobertos localmente.

Esta fase termina quando todos os itens `P0` e `P1` estiverem concluidos e a auditoria final passar. Cada item so deve ser marcado como concluido depois da execucao da evidencia indicada.

## Fora do escopo desta fase

- StatusCallback, entrega final, custo real e demais evolucoes da Twilio.
- Liberacao da WABA e homologacao de numero, credenciais e Content SIDs.
- Projeto, plano, regiao e configuracao de servicos no Railway.
- Dominio, DNS e TLS definitivos.
- Backup, PITR e restore do PostgreSQL gerenciado.
- Grafana Cloud e alertas externos.
- Validacao juridica de politica de privacidade, termos e contrato.

## P0 - linha de base e seguranca

### Baseline reproduzivel

- [x] `go mod verify` passa.
- [x] Todos os arquivos Go estao formatados.
- [x] `go vet ./...` passa.
- [x] As 32 migrations sobem em PostgreSQL vazio.
- [x] Uma segunda execucao das migrations aplica zero mudancas.
- [x] `go test ./...` passa com os testes PostgreSQL habilitados.
- [x] O smoke HTTP completo passa sem envio externo real.
- [x] Os quatro binarios operacionais compilam.
- [x] Os scripts Node passam em `node --check`.
- [x] `npm ci` e o build TypeScript/Vite passam.
- [x] `git diff --check` passa nos dois repositorios.

Evidencia: comandos equivalentes aos workflows em `.github/workflows/ci.yml`, executados localmente e registrados no handoff.

Execucao local em 2026-07-21: todos os itens acima passaram. As migrations e os testes de integracao usaram bancos PostgreSQL temporarios exclusivos, removidos depois da validacao. O smoke confirmou health/readiness, setup, auth, importacao, isolamento entre tenants, campanha, brinde, dashboard, privacidade, logout e revogacao do token; a API encerrou graciosamente.

### Seguranca do navegador e da sessao

- [x] Registrar a decisao de armazenamento/transporte da sessao e seu modelo de ameaca.
- [x] Remover o JWT persistente de `localStorage` ou justificar formalmente sua manutencao com mitigacoes.
- [x] Configurar cookies, CSRF e CORS de forma coerente caso a sessao migre para cookie.
- [x] Adicionar CSP apropriada ao frontend.
- [x] Adicionar `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy` e protecao contra framing.
- [x] Definir HSTS somente no ponto HTTPS apropriado e documentar a divisao de responsabilidade com o Railway.
- [x] Garantir logout, expiracao, revogacao e troca de senha no novo fluxo.
- [x] Testar origens permitidas e rejeitadas.

Evidencia: testes HTTP automatizados, build frontend e smoke de login/logout/troca de senha.

Execucao local em 2026-07-21: JWT removido do armazenamento do frontend; sessao em cookie `HttpOnly`, CSRF duplo, CORS por allowlist e Bearer preservado para scripts. Smoke real confirmou cookie, atributos, `403` sem CSRF, sucesso com CSRF, logout/revogacao e Bearer. Frontend recebeu imagem production com Nginx sem privilegios, healthcheck e headers; imagem construida e inspecionada por HTTP. Decisao e modelo de ameaca em `docs/session-security.md`.

### Capacidades e kill switches

- [x] Definir flags explicitas para WhatsApp, e-mail, automacao, Treino do dia e geracao LLM.
- [x] Manter separadas as flags de disponibilidade e as permissoes de efeito externo real.
- [x] Fazer o backend rejeitar de forma segura uma capacidade desabilitada.
- [x] Expor ao frontend somente um resumo seguro das capacidades habilitadas.
- [x] Ocultar navegacao e impedir acesso por URL quando a capacidade estiver desabilitada.
- [x] Validar defaults conservadores em `APP_ENV=production`.

Evidencia: testes de configuracao/router, testes de UI e smoke com combinacoes de flags.

Execucao local em 2026-07-21: flags separadas de permissoes de envio real, dependencias de configuracao validadas, middleware de bloqueio coberto por teste e endpoint `/api/v1/capabilities` sem dados sensiveis. Smoke com todas as flags desligadas retornou `404` para e-mail, automacao, workouts e mensagens. Frontend filtra menus, bloqueia hash direto e passou no build; a cobertura de navegador sera consolidada na gate Playwright.

## P1 - operacao, diagnostico e qualidade

### Observabilidade restante

- [x] Expor informacao de build: versao, commit e horario de build, sem segredos.
- [x] Adicionar metricas de importacao e linhas processadas/falhas.
- [x] Adicionar metricas de automacao por modo, resultado e stale run.
- [x] Adicionar metricas dos gateways SMTP e OpenAI por operacao/resultado/latencia.
- [x] Manter metricas WhatsApp apenas sobre aceite/falha sincrona enquanto callbacks estiverem fora do escopo.
- [x] Provisionar alertas locais para API indisponivel, 5xx, readiness, automacao e importacao com falha.
- [x] Atualizar o dashboard local com os novos sinais.
- [x] Confirmar ausencia de PII e labels de alta cardinalidade.

Evidencia: testes de metricas e smoke na stack local Prometheus/Loki/Tempo/Grafana com falhas controladas.

Execucao local em 2026-07-21: `/health/build` e `engagefit_application_info` confirmados com valores injetados; metricas de importacao, automacao, SMTP, OpenAI e aceite sincrono WhatsApp adicionadas com enums limitados. Teste de coleta passou. `promtool` validou sete regras, Prometheus carregou todas com health `ok` e o alerta de heartbeat ausente entrou em `firing`. Dashboard JSON validado e stack local reprovisionada.

### Erros acionaveis e suporte

- [x] Padronizar envelope HTTP de erro com codigo estavel, mensagem segura e `request_id`.
- [x] Propagar o `request_id` para erros apresentados pelo frontend.
- [x] Exibir um codigo copiavel para suporte sem revelar detalhes internos.
- [x] Normalizar erros de PostgreSQL, importacao, automacao, SMTP, OpenAI e WhatsApp.
- [x] Garantir que logs e respostas nao exponham SQL, segredos, telefone, e-mail ou corpo de mensagem.
- [x] Criar runbook de falhas comuns e diagnostico por `request_id`.

Evidencia: testes de handlers/middleware e cenarios locais de falha controlada.

Execucao local em 2026-07-21: todas as respostas de erro HTTP usam `code`, `message` segura e `request_id`; teste confirma correlacao com `X-Request-ID`. Gateways deixaram de devolver erro bruto. O cliente React preserva codigo/ID e acrescenta o identificador de suporte a mensagem. Runbook criado em `docs/operations-runbook.md`; testes Go, vet e build frontend passaram.

### Testes de navegador e PostgreSQL

- [x] Adicionar Playwright ao frontend.
- [x] Cobrir login, importacao, recalculo, dashboard, campanha, brinde e logout.
- [x] Cobrir troca de senha e revogacao da sessao anterior.
- [x] Cobrir opt-out, exportacao e anonimizacao de aluno.
- [x] Cobrir fluxo administrativo critico do PLATFORM_ADMIN.
- [x] Cobrir capacidades desabilitadas e erro com `request_id`.
- [x] Adicionar testes PostgreSQL para relatorios, brindes e destinatarios ainda nao cobertos.
- [x] Executar o E2E sem SMTP, OpenAI ou WhatsApp reais.

Evidencia: suite E2E reproduzivel localmente e adicionada ao CI.

Execucao local em 2026-07-21: Playwright/Chromium passou no cenario isolado de sessao, capacidades e erro de suporte e em dois cenarios contra API e PostgreSQL reais. O fluxo owner cobre importacao, campanha, metas, recalculo, dashboard, entrega de brinde, opt-out, exportacao, anonimizacao, troca de senha, novo login e logout; o fluxo de plataforma cobre acesso administrativo e reset de senha. Todas as capacidades externas permaneceram desligadas. Teste PostgreSQL adicional validou relatorios de elegiveis/frequencia, entrega de brinde e ciclo de persistencia de destinatario; ele tambem detectou e protege a regressao do mapeamento `provider_message_sid`. A CI do frontend instala Chromium e reproduz os testes mockados e reais.

## P2 - recomendados, sem bloquear a primeira publicacao

- [ ] Filtros e paginacao server-side quando o volume justificar.
- [ ] Relatorio de historico de brindes entregues.
- [ ] Relatorio de conversao de mensagens.
- [ ] Rate limit compartilhado antes de usar mais de uma replica.
- [ ] Recuperacao de senha por token/e-mail quando houver e-mail transacional homologado.
- [ ] Definir a fonte automatica de check-ins antes de automatizar a ingestao.
- [ ] Refinar tabelas em aparelhos moveis reais.

## Auditoria final antes da fase Railway

- [x] Todos os itens P0 e P1 estao concluidos com evidencia.
- [x] CI de backend e frontend reproduz o que passou localmente.
- [x] Configuracao production falha cedo com valores ausentes ou inseguros.
- [x] Imagem final roda como usuario sem privilegios e encerra em `SIGTERM`.
- [x] Liveness, readiness e informacao de build respondem como documentado.
- [x] Nenhum fluxo local chama gateway externo sem habilitacao explicita.
- [x] Retencao roda em dry-run por padrao.
- [x] Migrations e rotacao de segredos continuam separadas do startup da API.
- [x] Handoff e guia operacional refletem o comportamento final.
- [x] Foi criado um checklist separado para Railway, deploy, backup/restore e rollback.

Evidencia final em 2026-07-21: auditoria repetida em PostgreSQL 16 vazio aplicou as 32 migrations, confirmou segunda execucao com zero mudancas e passou toda a suite Go com integracoes. O ambiente local nao possui CGo/GCC, portanto o race detector permanece na CI Linux; a suite equivalente sem `-race` passou. Smoke HTTP, scripts Node, quatro binarios, build Vite, Playwright mockado/real e `git diff --check` passaram. As imagens `engagefit-api:audit` e `engagefit-web:audit` foram reconstruidas; API executou como `engagefit`, frontend como UID 101, health/build e headers foram inspecionados, e `SIGTERM` registrou `shutdown_started` e `shutdown_completed`. A retencao informou `mode=dry-run`; production invalida encerrou antes de abrir o servidor. Bancos e containers temporarios foram removidos. O passo seguinte esta isolado em `docs/railway-deployment-checklist.md`.
