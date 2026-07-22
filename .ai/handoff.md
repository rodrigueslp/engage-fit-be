# EngageFit - Handoff de Contexto

Manual canônico de arquitetura e negócio: `docs/system-design.md`.

Guia operacional consolidado: `docs/application-readiness-guide.md`.

Atualizado em: 2026-07-21 (deploy Railway e validacao de producao)

## Checkpoint de campanhas e consulta de check-ins em 2026-07-21

- A lista de progresso da campanha deixou de cortar silenciosamente em 8 alunos e passou a ter paginacao local de 10 itens.
- Um aluno agora participa do progresso somente quando possui ao menos um check-in dentro das datas inclusivas da campanha e existe meta para sua origem.
- O recálculo substitui logicamente os snapshots em transacao, removendo progressos obsoletos; entregas pendentes de brinde tambem sao sincronizadas, sem apagar entregas ja realizadas.
- Nova tela `Check-ins` consulta um intervalo de datas e mostra quantidade, primeira e ultima presenca por aluno, com busca, plataforma, ordenacao e paginacao no frontend.
- Novo endpoint: `GET /api/v1/checkins/summary?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`; a agregacao ocorre no PostgreSQL e a API continua sem paginacao neste primeiro momento.

## Checkpoint de deploy no Railway em 2026-07-21

### Estado implantado

- Backend, frontend e PostgreSQL foram criados no projeto Railway exibido como `motivated-playfulness`, no mesmo ambiente `production`.
- Servicos usados: `engage-fit-api`, `engage-fit-web` e `Postgres`.
- Arquitetura efetiva: navegador -> `engage-fit-web` publico -> `/api` via Nginx/rede privada -> `engage-fit-api` privado -> `Postgres` privado.
- Dominio publico validado: `https://engage-fit-web-production.up.railway.app`.
- API e PostgreSQL permanecem sem dominio HTTP publico; a API e acessada pelo frontend usando `BACKEND_URL=http://${{engage-fit-api.RAILWAY_PRIVATE_DOMAIN}}:8080`.
- Frontend usa `PORT=8080`, nao define `VITE_API_BASE_URL` e usa `VITE_CSRF_COOKIE_NAME=engagefit_session_csrf`.
- Backend usa `PORT=8080`, `HTTP_HOST=0.0.0.0` e `DATABASE_URL=${{Postgres.DATABASE_URL}}`; segredos production foram configurados diretamente no Railway e nao foram registrados no Git.
- Healthcheck efetivo dos dois servicos ficou em `/health`, com timeout de 300 segundos. O backend atual tambem implementa `/health/live` e `/health/ready`, mas `/health` foi mantido como rota compativel entre revisoes.
- Features opcionais e efeitos externos permaneceram desligados durante a homologacao: WhatsApp, e-mail, automacao, treinos, LLM e envios reais.

### Migrations e release

- O pre-deploy `/usr/local/bin/engagefit-migrate up` falhou repetidamente antes de iniciar o container, sem stdout/stderr. O diagnostico automatico do Railway classificou a falha como erro transitorio de infraestrutura, mas chegou a afirmar incorretamente que as migrations ja haviam sido aplicadas.
- A evidencia real apareceu no startup da API: `relation "users" does not exist`, confirmando que o banco ainda estava vazio.
- Para inicializar o banco, foi usado temporariamente como Custom Start Command: `sh -c '/usr/local/bin/engagefit-migrate up && exec /usr/local/bin/engagefit-api'`.
- As 32 migrations foram aplicadas e a API passou a iniciar com o schema completo. O migrator e idempotente e usa historico/checksum/advisory lock.
- Estado operacional a conferir no Railway: deixar o Custom Start Command vazio depois da inicializacao e restaurar o pre-deploy quando o agendamento de containers estiver estavel. Enquanto o pre-deploy estiver desativado, nenhuma release com migration nova deve ser publicada sem executar explicitamente o migrator.

### Incidentes encontrados e correcoes

- O primeiro backend ativo era uma revisao antiga porque quatro commits locais ainda nao estavam no GitHub. Isso explicava `/api/v1/capabilities` ausente, `/health/live` ausente e comportamento antigo do setup. O backend foi enviado de `9170a69` ate `a562871` e depois recebeu os ajustes de CI/formato descritos abaixo.
- `GET /api/v1/setup/owner` retorna 404 por desenho; o onboarding usa `POST`. Na revisao nova, production tambem exige `OWNER_SETUP_ENABLED=true` e `X-Setup-Token` valido.
- O Nginx do frontend resolvia o dominio privado da API somente no startup. Cada redeploy do backend alterava seus IPs privados; o frontend continuava chamando IPs antigos e passava a responder 502/504 depois de esperas de 10/20 segundos.
- Correcao permanente no frontend, commit `d4650db`: resolver interno `[fd12::10]`, cache DNS de 10 segundos, `ipv6=off` e `proxy_pass` por variavel para forcar resolucao em runtime. A imagem Docker foi reconstruida e `nginx -t` passou.
- Depois da correcao, novos IPs privados do backend deixam de exigir restart/redeploy manual do frontend.
- Se `Serverless/App Sleeping` estiver habilitado na API, deve ser desabilitado para resposta previsivel em producao; confirmar esse toggle no Railway.

### CI corrigido e validado

- Frontend commit `81cee85`: readiness do E2E real passou de 30 para 90 segundos, usa `127.0.0.1` e detecta encerramento antecipado da API. A falha anterior era `curl` exit code 7 enquanto o segundo `go run` ainda compilava no runner frio.
- Backend commit `4172508`: corrigiu YAML invalido causado pelo `:` no comando inline de idempotencia das migrations e aplicou o mesmo readiness robusto ao smoke da API.
- Backend commit `8b5443b`: aplicou `gofmt` em `internal/adapters/whatsapp/provider_gateway.go`.
- CI do frontend passou no run `29884280635`: build, Playwright mockado e Playwright real com PostgreSQL/API.
- CI do backend passou no run `29884561040`: modulos, formato, vet, 32 migrations, idempotencia, PostgreSQL/race detector, smoke HTTP, binarios e scripts.
- Estado do codigo homologado antes desta atualizacao documental: backend `8b5443b`; frontend `81cee85`; ambos estavam sincronizados com `origin/main` e sem alteracoes pendentes.

### Homologacao funcional em producao

- `/health` do frontend respondeu `200` com `{"status":"ok"}`.
- O proxy publico `/api/v1/capabilities` chegou corretamente a API privada depois do deploy da revisao atual.
- A primeira academia e a conta owner foram criadas pelo onboarding protegido e o fluxo do owner foi testado manualmente no dominio de producao.
- A conta `PLATFORM_ADMIN`, criada/atualizada no startup a partir de `PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD`, foi autenticada pelo mesmo formulario de login e redirecionada para `#admin-messaging`; o fluxo administrativo tambem foi testado manualmente.
- Credenciais do owner, setup token, JWT, chave de criptografia e credenciais administrativas permanecem somente nas variaveis/controle do operador e nao devem entrar neste handoff.
- Acao de seguranca a confirmar imediatamente: depois do onboarding, definir `OWNER_SETUP_ENABLED=false` e remover ou selar `OWNER_SETUP_TOKEN`. Repetir o POST deve retornar 404. A confirmacao dessa desativacao nao apareceu nesta sessao.

### Proximos passos operacionais

1. Confirmar no Railway `OWNER_SETUP_ENABLED=false` e setup token removido/selado.
2. Confirmar `Serverless/App Sleeping` desligado em `engage-fit-api`.
3. Confirmar Custom Start Command vazio. Se ainda estiver executando migration + API, pode permanecer apenas como contingencia curta, pois e idempotente, mas nao e o desenho final.
4. Restaurar `/usr/local/bin/engagefit-migrate up` no pre-deploy quando o Railway voltar a iniciar o container de release de forma confiavel; validar `migration complete: 0 applied`.
5. Configurar backup/PITR do PostgreSQL e executar restore real em ambiente isolado antes de dados de clientes.
6. Manter todas as integracoes e envios reais desligados ate homologacao individual dos provedores.
7. Configurar observabilidade/alertas e limites de custo do Railway antes do piloto.

## Checkpoint de prontidao final da aplicacao em 2026-07-21

- Checklist executavel criado em `.ai/application-readiness-checklist.md`, separado por gates P0, P1 e P2.
- StatusCallback e demais evolucoes Twilio ficaram explicitamente fora desta fase, assim como Railway, dominio/TLS, banco gerenciado, backup/restore, Grafana Cloud e validacoes juridicas.
- Baseline local passou: modulos Go, formatacao, `go vet`, 32 migrations em PostgreSQL vazio, idempotencia das migrations, testes com integracoes PostgreSQL, smoke HTTP, quatro binarios, scripts Node, `npm ci`, build TypeScript/Vite e `git diff --check`.
- O smoke confirmou health/readiness, setup, auth, importacao, isolamento entre tenants, campanha, brinde, dashboard, privacidade, logout e revogacao. API encerrou graciosamente e os bancos temporarios foram removidos.
- Todos os gates P0/P1 foram concluidos: sessao HttpOnly/CSRF, CSP/headers, kill switches, metricas/alertas, erros com `request_id`, Playwright e integracoes PostgreSQL adicionais.
- Playwright passou contra API/PostgreSQL reais com gateways externos desligados: owner percorreu importacao, campanha, recalculo, dashboard, brinde, privacidade, troca de senha e logout; PLATFORM_ADMIN percorreu administracao e reset de senha.
- O teste de destinatarios encontrou e corrigiu o mapeamento GORM de `provider_message_sid`; a regressao esta coberta em PostgreSQL real.
- A revisão do manual encontrou e corrigiu a permissão de efeito externo: SMTP e WhatsApp reais agora exigem `*_ALLOW_REAL_SEND=true` também em production; mocks continuam liberados. Testes impedem regressão.
- Auditoria final reconstruiu as imagens, confirmou usuarios sem privilegios, build info/health, headers do frontend, `SIGTERM`, retencao dry-run e configuracao production fail-fast.
- O proximo trabalho e exclusivamente a fase de infraestrutura descrita em `docs/railway-deployment-checklist.md`, alem da Twilio que continua fora deste escopo.

## Checkpoint de transferencia de conhecimento em 2026-07-21

- `docs/system-design.md` foi criado como manual canonico para o proprietario tecnico e deve ser atualizado no mesmo pull request que alterar arquitetura, regra de negocio, seguranca, tenancy, privacidade, efeitos externos ou operacao.
- O manual documenta atores, arquitetura, pipeline HTTP, sessao, capabilities, modelo de dados, importacao, campanhas, progresso, brindes, risco, comunicacao, automacao, privacidade, criptografia, migrations, runtime, observabilidade, configuracao, API, testes, trade-offs e dividas conhecidas.
- A secao 26 contem uma trilha pratica de estudo e exercicios locais; a secao 27 funciona como verificacao de dominio do sistema; a secao 29 aponta os arquivos-fonte de cada assunto.
- A revisao cruzada entre documentacao e implementacao encontrou uma falha nos kill switches: production liberava SMTP e WhatsApp reais mesmo com as flags desligadas. Isso foi corrigido e coberto por testes.
- Commits deste fechamento: `b3245a2` (`fix: require explicit permission for real sends`) e `eb87a9f` (`docs: add canonical system engineering manual`).
- Validacao final: `go test ./...`, `go vet ./...`, `git diff --check` e conferencia dos links locais do manual passaram; os dois repositorios ficaram limpos.
- Nenhum push ou deploy foi executado. O proximo passo recomendado e percorrer a trilha de conhecimento localmente e depois executar o checklist do Railway, mantendo a integracao efetiva com Twilio fora do escopo por enquanto.

## Direcao concluida - prontidao da aplicacao antes do deploy

Decisao registrada em: 2026-07-20

Objetivo da proxima fase:

- Nao fazer o deploy agora.
- Resolver primeiro tudo que puder ser tratado no codigo e no desenho operacional da aplicacao.
- Manter infraestrutura desacoplada e adiar decisoes definitivas de dominio, TLS, banco gerenciado, backups do provedor e pipeline de deploy.
- Railway e o destino de deploy mais provavel, portanto a aplicacao deve funcionar bem em container efemero, usar configuracao por variaveis de ambiente, escrever logs em `stdout` e encerrar corretamente ao receber `SIGTERM`.
- A configuracao efetiva do numero/canal WhatsApp sera tratada separadamente e nao bloqueia o trabalho de prontidao da aplicacao.

### Bloqueadores de aplicacao antes de um piloto real

Checkpoint de seguranca em 2026-07-20:

- Criacao de box + owner passou a ser transacional; falha ao gravar o owner desfaz o box e o e-mail e verificado antes da criacao.
- `POST /api/v1/setup/owner` agora e controlado por `OWNER_SETUP_ENABLED`, fica desligado por padrao em production e exige `OWNER_SETUP_TOKEN` forte quando habilitado nesse ambiente.
- Configuracao production falha cedo com banco ausente, JWT fraco, administrador ausente ou limites HTTP invalidos.
- Login e setup possuem rate limit por IP e identidade normalizada/hasheada, com resposta `429` e `Retry-After`.
- Body JSON e upload possuem limites configuraveis; imports limitam arquivo, linhas, colunas e tamanho descomprimido das partes relevantes do XLSX.
- Usuario autenticado pode trocar a propria senha em `PUT /api/v1/auth/password` e na tela Configuracoes; minimo de 12 caracteres e confirmacao da senha atual.
- PLATFORM_ADMIN pode redefinir a senha do owner de uma academia em `PUT /api/v1/admin/boxes/:id/owner-password` e na aba `Acesso`; motivo obrigatorio e auditoria sem registrar a senha.
- Novas configuracoes: `HTTP_MAX_BODY_BYTES`, `IMPORT_MAX_UPLOAD_BYTES`, `LOGIN_RATE_LIMIT_*`, `SETUP_RATE_LIMIT_*`, `OWNER_SETUP_ENABLED` e `OWNER_SETUP_TOKEN`.
- Validacoes executadas: `go test ./...`, `node --check scripts/demo-seed.mjs`, TypeScript e build Vite.
- Ainda pendente para escala horizontal: tornar o rate limit compartilhado entre replicas; no piloto de uma replica o controle atual e efetivo. Ampliar os testes E2E segue na frente de qualidade.

Checkpoint de sessao/runtime em 2026-07-20:

- Migration `030_add_user_auth_version.sql` adiciona `users.auth_version`; aplicada no PostgreSQL local.
- JWT inclui `auth_version` e o middleware confirma usuario, tenant, papel e versao no banco em toda requisicao autenticada.
- Troca/redefinicao de senha e rotacao do PLATFORM_ADMIN incrementam `auth_version`, revogando tokens anteriores imediatamente.
- Logout agora incrementa `auth_version`; o frontend chama o endpoint antes de remover a sessao local.
- Smoke isolado confirmou: login `200`, token valido `200`, logout `204`, token antigo `401` e novo login `200`; tenant temporario removido.
- `TRUSTED_PROXIES` controla explicitamente quais proxies podem fornecer o IP do cliente; vazio nao confia em `X-Forwarded-For`.
- API usa `http.Server` com timeouts configuraveis e shutdown gracioso em `SIGINT`/`SIGTERM`.
- Worker recebe o contexto de encerramento, para de aceitar ticks e sinaliza conclusao antes do fim do processo.
- PostgreSQL possui pool configuravel, ping no startup e readiness com timeout.
- Endpoints: `/health/live`, `/health/ready` e alias `/health`; readiness retorna `503` quando PostgreSQL nao responde.
- A porta `PORT` do provedor e aceita como fallback de `HTTP_PORT`.
- Docker final executa como usuario `engagefit`, inclui CA certificates e `tzdata` e possui healthcheck dinamico.
- Validacao real confirmou liveness/readiness `200`, logs `shutdown_started`/`shutdown_completed`, build da imagem e usuario final sem privilegios.

Checkpoint de observabilidade em 2026-07-20:

- Backend instrumentado com OpenTelemetry para traces, metricas e logs via OTLP HTTP, habilitado por `OTEL_ENABLED` e configuracao padrao `OTEL_EXPORTER_OTLP_*`.
- Requests HTTP geram traces, metricas de volume/latencia/em andamento por rota normalizada e logs correlacionados por `request_id`, `trace_id` e `span_id`.
- Queries GORM e chamadas HTTP externas de Twilio, Meta e OpenAI participam dos traces; variaveis SQL nao entram nos spans.
- Runtime Go e pool do PostgreSQL expoem metricas sem labels de alta cardinalidade.
- Logs HTTP deixaram de registrar IP e URL concreta; `X-Request-ID` recebido e limitado a 128 caracteres seguros para evitar injecao em logs.
- `GET /metrics` existe para Prometheus, fica desligado por padrao em production e, se habilitado nesse ambiente, exige `PROMETHEUS_BEARER_TOKEN` com ao menos 32 caracteres.
- Stack local opcional em `docker-compose.observability.yml`: Grafana, Prometheus, Loki, Tempo e OpenTelemetry Collector, com datasources e painel `EngageFit - Visao geral` provisionados.
- Comandos: `make observability-up`, `make observability-down` e `make observability-logs`; instrucoes em `observability/README.md`.
- Railway pode manter metricas de CPU/RAM/disco/rede e a API pode enviar OTLP diretamente ao Grafana Cloud gratuito sem alteracao de codigo.
- Smoke ponta a ponta confirmou request no Prometheus, log no Loki sem IP/URL concreta, trace no Tempo e navegacao por `trace_id`; stack local foi desligada depois do teste e seus volumes foram preservados.
- Testes de middleware cobrem protecao Bearer do endpoint de metricas e saneamento de request ID; `go test ./...` passa.
- Ainda pendentes nesta frente: metricas de negocio/gateways, alertas provisionados e informacao de build; entram junto das automacoes/qualidade antes de encerrar a prontidao.

Checkpoint de migrations versionadas em 2026-07-20:

- O loop que reaplicava todos os SQLs foi substituido pelo migrator proprio em `migrations/migrator.go` e pelo binario `cmd/migrate`.
- `schema_migrations` registra versao, nome, SHA-256, horario e tempo de execucao; SQL aplicado nao pode ser alterado silenciosamente depois.
- Cada migration roda em transacao e um advisory lock do PostgreSQL impede duas releases de migrar simultaneamente.
- Comandos: `make migrate-up`, `make migrate-status` e, apenas para adotar banco legado ja conferido, `make migrate-baseline VERSION=N`.
- Banco nao vazio sem historico e recusado com instrucao de baseline; baseline e recusado em banco vazio e exige versao explicita.
- A imagem passa a incluir `/usr/local/bin/engagefit-migrate`; em release ele deve executar `up` separadamente antes da API. A API continua sem alterar schema no startup.
- Banco local foi conferido pelos marcos das migrations 028-030 e recebeu baseline ate 030. Nova execucao aplicou zero migrations.
- Smoke em banco PostgreSQL temporario real aplicou as 30 migrations, segunda execucao aplicou zero, confirmou tabelas finais e removeu o banco temporario.
- Testes unitarios cobrem ordenacao, checksum, filename e lacunas da sequencia; `go test ./...` passa.

Checkpoint de criptografia de credenciais em 2026-07-20:

- Credenciais dedicadas do WhatsApp (`api_key_encrypted`) e senhas SMTP (`password_encrypted`) agora sao cifradas no limite dos repositories antes de chegar ao PostgreSQL e decifradas somente para uso interno.
- Envelope versionado `enc:v1:<key_id>:<payload>` usa AES-256-GCM com nonce aleatorio e associated data vinculando tipo do segredo, `box_id` e campo; copiar ciphertext entre tenants/campos falha autenticacao.
- Keyring fica exclusivamente em `DATA_ENCRYPTION_KEYS` e a chave de escrita em `DATA_ENCRYPTION_ACTIVE_KEY_ID`; ambas sao obrigatorias em production.
- Runtime com criptografia configurada rejeita plaintext legado. Sem chaves, somente fora de production, a API preserva compatibilidade local e emite `data_encryption_disabled` em nivel warn.
- Binario `/usr/local/bin/engagefit-rotate-secrets` e `make rotate-secrets` convertem plaintext legado e recifram valores de chave antiga para a ativa dentro de uma unica transacao com locks de linha.
- Rotacao segura: adicionar chave nova + antiga ao keyring, tornar a nova ativa, executar o comando, atualizar todas as instancias e somente entao remover a antiga.
- Smoke em PostgreSQL temporario confirmou plaintext -> chave `old`, segunda execucao idempotente (`0` alteracoes) e `old` -> `new`, sem exibir os segredos; banco temporario removido.
- Testes cobrem round-trip, adulteracao de associated data, keyring invalido, leitura de chave antiga para rotacao e rejeicao de plaintext no runtime; `go test ./...` e `go vet ./...` passam.

Checkpoint de concorrencia/idempotencia das automacoes em 2026-07-20:

- Migration `031_add_automation_idempotency.sql` adiciona `schedule_id`, `scheduled_for` e `execution_key` unica por academia em `automation_runs`; aplicada localmente pelo novo migrator.
- Worker faz claim atomico do slot no PostgreSQL antes de qualquer efeito. Smoke com 20 claims concorrentes confirmou exatamente um vencedor; multiplas replicas podem manter o worker habilitado.
- Cada slot recebe chave deterministica `schedule:<id>:<horario>`. Repeticoes manuais aceitam `Idempotency-Key`; o frontend gera uma chave por clique e uma repeticao com a mesma chave retorna o run existente.
- Smoke HTTP confirmou primeira criacao `201`, replay `200`, mesmo `run.id` e `idempotent_replay=true`; registro temporario removido.
- `daily-automation.mjs` usa chave diaria por timezone (ou `DAILY_AUTOMATION_IDEMPOTENCY_KEY`) e encerra sem repetir import/recalculo/envio ao detectar replay.
- Estrategia e at-most-once por slot: se a instancia morrer com resultado externo incerto, o worker nao repete mensagens automaticamente. O run fica `running`, passa a `failed` apos `AUTOMATION_STALE_RUN_MINUTES` e exige revisao antes de uma nova chave manual.
- Janela `AUTOMATION_CATCHUP_WINDOW_MINUTES` permite recuperar atrasos curtos de startup/deploy sem executar rotinas horas depois do horario.
- Atualizacao administrativa da agenda nao sobrescreve mais `last_run_at`, eliminando corrida com o claim do worker.
- Nomes, modo, `HH:MM`, timezone IANA e dias da semana sao validados; entradas invalidas retornam `400`.
- Metricas OTel contam/duram runs por status, e logs estruturados registram conclusao/falha/stale sem PII.
- Logger GORM foi substituido por logger seguro: `record not found` nao gera ruido, SQL/parametros e erros brutos do PostgreSQL nao vao para logs; HTTP registra apenas contagem de erros, nao a mensagem bruta.
- `go test ./...`, `go vet ./...`, `node --check scripts/daily-automation.mjs`, TypeScript e build Vite passam.

Checkpoint de qualidade e CI em 2026-07-20:

- Backend e frontend agora possuem workflows GitHub Actions separados. Backend valida formato, modulos, `go vet`, migrations em PostgreSQL 16 vazio e idempotentes, testes com race detector/integracao, smoke HTTP, scripts e binarios; frontend executa instalacao reproduzivel, build TypeScript/Vite e Playwright mockado e real com PostgreSQL 16/API.
- Testes de autorizacao cobrem token com tenant/versao divergente e separacao OWNER/PLATFORM_ADMIN. Testes PostgreSQL cobrem claim concorrente de automacao e isolamento de alunos e recompensas entre duas academias.
- Auditoria durante os testes encontrou e corrigiu isolamento ausente em metas/progresso/brindes de campanha e nas listagens de destinatarios de WhatsApp, e-mail e Treino do dia. Recursos de outra academia passam a responder como nao encontrados.
- Login agora normaliza espacos e caixa do e-mail da mesma forma que o onboarding; o smoke capturou o caso em que owner criado com letra maiuscula nao conseguia entrar imediatamente.
- `scripts/smoke-api.mjs` cobre health/readiness, criacao de duas academias, login, `/auth/me`, campanha, meta, brinde, tentativas cruzadas entre tenants, recalculo, dashboard, logout e revogacao do token.
- Smoke completo passou em PostgreSQL temporario com todas as migrations; a API foi encerrada graciosamente e o banco temporario removido. `go test ./...`, `go vet ./...`, testes de integracao PostgreSQL e build frontend passam localmente.
- O race detector fica validado pelo runner Linux do CI; o ambiente local atual nao possui compilador C (`gcc`) para executar `go test -race`.

Checkpoint de privacidade/LGPD na aplicacao em 2026-07-20:

- Migration `032_add_student_privacy_controls.sql` adiciona estado/origem/data da preferencia de contato, data de anonimizacao, auditoria de privacidade e supressoes de identidade; aplicada no PostgreSQL local.
- A tela `Alunos` permite marcar `Autorizado`, `Nao contatar` ou `Nao informado`, exportar JSON e anonimizar com motivo e dupla confirmacao.
- `opted_out` e alunos anonimizados sao excluidos de todas as audiencias de WhatsApp, e-mail e Treino do dia. O envio de um rascunho antigo de treino volta a conferir a elegibilidade antes de reservar limite/chamar o gateway.
- Exportacao inclui cadastro, check-ins, progresso e historico de destinatarios e grava evento de auditoria sem duplicar PII na auditoria.
- Anonimizacao transacional limpa nome, e-mail, telefone e erros/destinos historicos, pausa contato, preserva metricas anonimas e registra hash de supressao. Reimportacoes posteriores da mesma identidade sao ignoradas.
- Binario `engagefit-privacy-retention` e comandos `make privacy-retention-dry-run`/`make privacy-retention-apply` tratam destinatarios, logs LLM, automacoes, importacoes/check-ins e auditoria com prazos configuraveis em `PRIVACY_RETENTION_*`; dry-run e o comportamento padrao.
- Politica inicial, fluxo de titular, responsabilidades propostas e procedimento de incidente estao em `docs/privacy-runbook.md`; classificacao controlador/operador, contrato, bases legais e textos publicos ainda exigem validacao juridica antes do go-live.
- Teste PostgreSQL confirma exportacao, anonimizacao, opt-out e supressao. Smoke HTTP completo confirma importacao, exportacao, anonimizacao e bloqueio da reimportacao em banco temporario com as 32 migrations; banco removido ao final.

Checkpoint de UX do owner e escopo inicial de producao em 2026-07-21:

- A pagina `Configuracoes` do owner foi reorganizada em tres secoes: `Alunos em risco`, `Acesso e seguranca` e `Integracao WhatsApp`, exibindo um assunto por vez em navegacao responsiva.
- Regras de risco passaram a usar textos mais diretos, explicacoes por campo e uma pre-visualizacao da regra efetiva antes de salvar.
- Troca de senha foi simplificada e mantem a exigencia de ao menos 12 caracteres, confirmacao e aviso de encerramento das sessoes abertas.
- Integracao WhatsApp do owner passou a mostrar somente resumo operacional, disponibilidade, tipo de conexao, remetente, ultima atualizacao e teste. Credenciais e campos tecnicos administrados pela plataforma nao aparecem mais como controles desabilitados.
- Os menus `Treino do dia` e `E-mail` foram ocultados temporariamente da navegacao do owner para a primeira publicacao em producao.
- As paginas, rotas e implementacoes de `Treino do dia` e `E-mail` continuam preservadas no codigo para reativacao futura; esta decisao altera apenas sua exposicao no menu neste momento.
- Validacoes executadas: TypeScript, build de producao Vite e `git diff --check`.

Seguranca e acesso:

1. Proteger ou remover de producao o endpoint publico `POST /api/v1/setup/owner`. O onboarding deve ser restrito ao administrador, usar convite/token de uso unico ou ficar explicitamente habilitado apenas em development.
2. Tornar a criacao de box + owner transacional para nunca deixar box orfao quando o usuario falhar.
3. Adicionar rate limit por IP/e-mail ao login, ao setup e aos endpoints de maior custo.
4. Adicionar limites de tamanho para body JSON, upload CSV/XLSX, quantidade de linhas importadas e campos de texto.
5. Implementar troca de senha pelo owner e um fluxo administrativo seguro de redefinicao. Avaliar recuperacao de senha por token quando houver e-mail produtivo.
6. Validar configuracao obrigatoria ao iniciar em `APP_ENV=production`: `JWT_SECRET` forte, credenciais administrativas, banco e demais segredos; a aplicacao deve falhar cedo se ainda usar defaults inseguros.
7. Revisar armazenamento do JWT no frontend, expiracao/logout e protecoes de browser (CSP e headers de seguranca) antes da exposicao publica.

Dados, segredos e migrations:

1. Substituir o loop SQL do `make migrate-up` por um migrator versionado, com tabela de controle e execucao unica de cada migration (ex.: `golang-migrate`, `goose` ou Atlas).
2. Criar um comando de migration apropriado para release, separado do start normal da API e dos comandos demo.
3. Implementar criptografia real para credenciais dedicadas de WhatsApp e senhas SMTP. Os campos atuais se chamam `*_encrypted`, mas hoje recebem o valor diretamente. A chave de criptografia deve ficar fora do PostgreSQL e permitir rotacao.
4. Garantir que segredos e PII nao aparecam em logs, auditorias, mensagens de erro ou dumps usados para suporte.
5. Configurar pool PostgreSQL, timeouts e verificacao de conectividade; preparar a aplicacao para conexoes limitadas do provedor.
6. Separar definitivamente comandos/dados demo dos fluxos de producao. `demo-reset-seed` nunca deve fazer parte de um deploy real.

Runtime e confiabilidade:

1. Trocar `router.Run` por `http.Server` com timeouts de leitura/escrita/idle, tratamento de `SIGTERM` e shutdown gracioso da API, telemetria e worker.
2. Adicionar endpoints separados de liveness e readiness; readiness deve verificar ao menos PostgreSQL e dependencias indispensaveis.
3. Ajustar a imagem final para usuario sem privilegios, certificados CA e `tzdata`.
4. Tornar automacoes seguras para mais de uma replica usando claim/lock transacional, idempotencia e protecao contra duas execucoes simultaneas da mesma rotina. Enquanto isso nao existir, documentar uma unica replica com worker.
5. Padronizar timeout, retry com backoff apenas quando seguro e idempotencia nas chamadas externas.
6. Implementar StatusCallback/webhook assinado da Twilio e atualizacao assincrona do status final. Hoje `sent` pode significar apenas que o provedor aceitou inicialmente uma mensagem que depois ficou `undelivered`.
7. Adicionar alertas operacionais para automacoes, importacoes e disparos com falha.

Qualidade e operacao:

1. Criar CI com `go test ./...`, verificacao das migrations, TypeScript e build Vite.
2. Adicionar testes de isolamento entre dois tenants, autorizacao OWNER/PLATFORM_ADMIN, onboarding, importacao/deduplicacao, limites de mensageria, automacoes e fluxos principais de campanha/brinde.
3. Adicionar pelo menos um smoke/E2E do login ate importacao, recalculo e consulta do dashboard.
4. Criar feature flags/configuracoes seguras para manter e-mail, automacao, LLM e envios reais desligados ate cada capacidade ser homologada.
5. Criar runbooks de erro comuns e deixar mensagens acionaveis na UI, sempre com `request_id` para suporte.

Privacidade/LGPD em nivel de aplicacao e processo:

1. [Processo/juridico] Formalizar a relacao controlador/operador, finalidades e bases legais; proposta tecnica registrada no runbook.
2. [Concluido na aplicacao] Exportacao, correcao via origem/reimportacao e exclusao por anonimizacao auditada.
3. [Concluido na aplicacao] Retencao definida e automatizavel por comando seguro; agendamento pertence a infraestrutura futura.
4. [Concluido na aplicacao] Opt-in/opt-out registrado e aplicado a todas as audiencias.
5. [Parcial] Procedimento de incidente documentado; politica publica, termos e contrato dependem de validacao juridica antes de dados reais.

### Direcao de observabilidade

Observabilidade passa a ser uma frente explicita de produto e operacao, nao uma tarefa deixada somente para a infraestrutura.

Arquitetura recomendada:

- Manter logs estruturados JSON via `slog` em `stdout`. Railway captura `stdout/stderr` e permite consulta imediata.
- Instrumentar a aplicacao Go com OpenTelemetry, usando OTLP e configuracao por ambiente para evitar dependencia de um fornecedor.
- Usar Railway para saude do container/maquina: CPU, memoria, disco, rede, deploys e logs de curta retencao.
- Usar Grafana Cloud no plano gratuito como primeira opcao para dashboards, alertas, metricas e traces de aplicacao. A aplicacao podera enviar OTLP diretamente no piloto ou passar por OpenTelemetry Collector/Grafana Alloy quando houver necessidade de processamento, batching ou multiplos destinos.
- Para desenvolvimento local, disponibilizar uma stack opcional via Docker Compose com Grafana + Prometheus + Loki + Tempo e Collector/Alloy, sem tornar essa stack requisito para rodar o EngageFit.
- Nao self-hostear toda a stack de observabilidade em producao inicialmente: isso adicionaria componentes, persistencia e manutencao justamente antes do primeiro piloto.

Telemetria minima da aplicacao:

- Logs HTTP: `request_id`, trace/span id, rota normalizada, metodo, status, latencia, ambiente, versao/release e replica.
- Logs de negocio: `box_id`, operacao, entidade, source type, resultado e erro normalizado, sem telefone, e-mail, token, corpo de mensagem ou credencial.
- Metricas HTTP: requests, erros, latencia e requests em andamento por rota normalizada.
- Metricas PostgreSQL: conexoes abertas/em uso/ociosas, espera por conexao e erros; evitar labels de alta cardinalidade.
- Metricas de gateways: chamadas, latencia, timeout e falha por provedor/operacao.
- Metricas de negocio/worker: imports concluídos/falhos, linhas processadas, automacoes executadas/falhas, mensagens autorizadas/bloqueadas/aceitas/falhas e geracoes LLM.
- Traces distribuidos da requisicao HTTP ate repository e chamadas externas, propagando contexto e `request_id`.
- Health endpoints: liveness, readiness e informacao de build sem expor segredo.

Dashboards/alertas iniciais:

1. Saude geral: disponibilidade, taxa de 5xx, p95/p99 de latencia e requests por minuto.
2. Runtime: CPU, memoria, disco, rede e reinicios da instancia.
3. PostgreSQL: disponibilidade, pool saturado, erros e queries lentas selecionadas.
4. Automacao: rotina atrasada, execucao com falha e ausencia de execucao esperada.
5. Mensageria: bloqueios por governanca, falhas do provider e divergencia entre aceito e entregue.
6. Alertas iniciais para API indisponivel, aumento de 5xx, readiness falha, automacao falha e consumo proximo do limite.

Cuidados de observabilidade:

- Nunca usar `student_id`, telefone, e-mail, `box_id` irrestrito, `request_id` ou URL crua como label de metrica de alta cardinalidade. IDs podem existir em logs/traces com acesso controlado, mas nao como dimensoes de metrica.
- Aplicar sampling em traces e reduzir logs de sucesso muito frequentes em producao.
- Definir retencao e controle de acesso tambem para a telemetria, pois logs podem virar uma nova fonte de vazamento de PII.
- Fazer flush da telemetria no shutdown gracioso antes de o Railway substituir a instancia.

Referencias de direcao:

- Railway Observability: `https://docs.railway.com/observability`
- Railway + ferramentas externas/OTLP: `https://docs.railway.com/guides/third-party-observability`
- OpenTelemetry: `https://opentelemetry.io/docs/`
- Grafana Cloud: `https://grafana.com/docs/grafana-cloud/introduction/`
- Grafana Loki: `https://grafana.com/docs/loki/latest/`

### Itens deliberadamente adiados para a fase de infraestrutura/deploy

- Escolha final de plano/regiao do Railway.
- Dominio, DNS, TLS e roteamento definitivo frontend/API.
- PostgreSQL gerenciado, politica concreta de backup/PITR e teste de restore no provedor.
- Pipeline de deploy, ambientes Railway e estrategia de rollback de release.
- Alertas de custo e limites do provedor.
- Configuracao produtiva do numero/canal WhatsApp.

Esses itens continuam obrigatorios antes do go-live, mas nao devem interromper a conclusao da prontidao no nivel da aplicacao.

## Feature concluída - governança de limites e custos do WhatsApp

Implementada em: 2026-07-16

- Documento de referência: `.ai/messaging-governance.md`.
- Migration `029_create_messaging_governance.sql` aplicada no PostgreSQL local.
- Políticas individuais por academia com limite diário, mensal, por disparo, orçamento diário/mensal, estimativa unitária, timezone e bloqueio administrativo.
- Política global adicional para todas as academias que usam o número compartilhado do EngageFit.
- Reserva transacional antes do envio impede estouro concorrente e bloqueia antes da chamada à Twilio.
- Campanhas manuais, automações e Treino do dia usam a mesma governança.
- `message_dispatches` registra envios autorizados/bloqueados; buckets diários/mensais registram consumo e reservas.
- Resposta do gateway agora preserva `provider_message_sid` e status inicial da Twilio/Meta por destinatário.
- Novo papel `PLATFORM_ADMIN`, sem `box_id`, isolado dos endpoints tenant.
- Administrador provisionado/rotacionado por `PLATFORM_ADMIN_NAME`, `PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD`.
- Área frontend `#admin-messaging` permite gerenciar política global e individual com auditoria e motivo obrigatório.
- Conta, remetente, credencial e modo da conexão dedicada agora são editados na área administrativa; owner possui somente leitura e teste da conexão salva.
- Owner visualiza sua franquia na tela WhatsApp, mas não pode alterar limites nem acessar rotas administrativas.
- Smoke test confirmou bloqueio antes do gateway, isolamento do admin e `403` para owner nas rotas administrativas; dados temporários removidos.
- Validações: `go test ./...`, TypeScript e build Vite passando.
- Próximas camadas: StatusCallback assinado da Twilio, conciliação de custo real, alertas, tela de dispatches/auditoria, secret manager e subaccounts para conexões dedicadas.

## Status operacional mais recente - WhatsApp/Twilio producao

Data: 2026-07-16

Objetivo atual do produto:

- Deixar o canal oficial de WhatsApp produtivado para o EngageFit.
- Manter um unico numero central do EngageFit como opcao padrao, enviando mensagens "em nome do box" no corpo da mensagem.
- Permitir que uma academia use sua propria conta Twilio, WABA, numero e templates quando precisar de uma conexao dedicada.
- Usar a conta verificada do primeiro cliente durante o piloto inicial de aproximadamente tres meses, evitando depender agora da verificacao empresarial do EngageFit.
- Tratar numero proprio por academia como capacidade core ja implementada e possivel diferencial de plano premium no futuro.

## Feature concluida - conexao WhatsApp compartilhada ou dedicada

Implementada em: 2026-07-16

- Backend, frontend, migration, documentacao e testes concluidos.
- API `GET/PUT /api/v1/whatsapp/settings` agora aceita e retorna `connection_mode` com valores `platform` ou `dedicated`.
- `platform` e o default para novas academias e usa exclusivamente as credenciais/remetente/Content SIDs definidos nas variaveis `WHATSAPP_PLATFORM_*` do backend.
- `dedicated` usa os dados persistidos em `whatsapp_settings` para o tenant autenticado.
- Campanhas oficiais, automacoes que enviam campanhas e mensagens do Treino do dia usam o resolver central de configuracao antes de chamar o gateway WhatsApp.
- No modo dedicado, a tela WhatsApp permite cadastrar o Content SID e o status de aprovacao de cada um dos tres templates oficiais na conta Twilio da academia.
- Migration `028` foi aplicada no PostgreSQL local. O registro existente foi preservado como `dedicated`, com credencial e remetente anteriores intactos.
- A alternancia `dedicated -> platform -> dedicated` foi validada pela API sem perder a credencial dedicada.
- Validacoes executadas com sucesso:
  - `go test ./...`
  - TypeScript (`tsc -b`)
  - build Vite
  - smoke test autenticado de configuracoes e previews dos tres templates.
- Estado local ao finalizar: `connection_mode=dedicated`; a conexao compartilhada esta indisponivel ate as variaveis `WHATSAPP_PLATFORM_*` serem preenchidas.
- Proximo passo do piloto: substituir na conexao dedicada as credenciais/remetente atuais pelos dados da conta Twilio verificada do primeiro cliente e cadastrar os tres Content SIDs aprovados nessa conta.

Decisao de arquitetura/comercial:

- Nao usar WhatsApp Web, Evolution API ou servicos nao oficiais para producao.
- Caminho recomendado e oficial: Twilio WhatsApp integrado a WABA/Meta.
- Conexao WhatsApp agora possui dois modos por academia:
  - `platform` (padrao para novas academias): usa a conta Twilio e o numero compartilhado do EngageFit, configurados somente por variaveis de ambiente do backend.
  - `dedicated`: usa a conta, credenciais, numero e Content SIDs proprios da academia; preparado para virar opcao premium.
- Migration `migrations/028_add_whatsapp_connection_mode.sql` adiciona `whatsapp_settings.connection_mode`. Configuracoes existentes e ativas migram para `dedicated` para preservar o comportamento; academias sem registro usam `platform` por padrao.
- A conexao compartilhada e configurada por `WHATSAPP_PLATFORM_ENABLED`, `WHATSAPP_PLATFORM_BASE_URL`, `WHATSAPP_PLATFORM_TWILIO_SENDER`, `WHATSAPP_PLATFORM_TWILIO_ACCOUNT_SID`, `WHATSAPP_PLATFORM_TWILIO_AUTH_TOKEN` e os tres `WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_*`.
- O backend resolve a conexao efetiva antes dos envios de campanhas e Treino do dia. Credenciais compartilhadas nunca sao retornadas pela API nem duplicadas em `whatsapp_settings`.
- A tela `Configuracoes > Integracao WhatsApp` permite escolher `Numero do EngageFit` ou `Numero proprio da academia`, informa a disponibilidade da conexao compartilhada e so exibe credenciais no modo dedicado.
- Content SIDs pertencem a conta Twilio: no modo `platform`, os tres SIDs vem do ambiente central; ao usar `dedicated`, os templates precisam ser aprovados na conta da academia e seus SIDs/status podem ser cadastrados na tela WhatsApp daquele tenant.
- O produto deve controlar internamente limites por plano, volume por box, logs/auditoria, cooldown e elegibilidade de destinatarios.
- A integracao com Twilio deve ser tratada como infraestrutura central do EngageFit, nao como configuracao individual obrigatoria de cada box no MVP.

Estado Twilio/Meta:

- Conta Twilio saiu do trial e foi feito upgrade/pagamento.
- Sandbox Twilio ja havia sido validado com sucesso antes: mensagens enviadas e recebidas no WhatsApp do Luiz usando o sandbox.
- Sender oficial criado no Twilio:
  - numero: `+55 11 5217-0912`
  - formato backend: `+551152170912`
  - display name: `EngageFit`
  - WABA ID exibido na Twilio/Meta: `1361712461954280`
  - Meta Business/Portfolio ID: `1002668215888321`
- O sender oficial consegue receber mensagens inbound do celular do Luiz.
- O sender oficial nao consegue enviar mensagens outbound nem responder mensagens inbound.
- Erro confirmado na Twilio:
  - status final: `undelivered`
  - erro: `63051`
  - significado operacional: WABA/sender bloqueado/restrito pela Meta.
- Na Meta Business Support foi localizado o ativo correto:
  - `Contas do WhatsApp > EngageFit`
  - status: `Conta com restricao`
  - restricoes exibidas:
    - `You can not start conversations with customers`
    - `You can not respond to messages from customers`
    - `You can not have phone numbers added to it`
- A propria tela da Meta informa que, para comecar a enviar mensagens, o provedor de solucoes de negocios precisara verificar a empresa.
- Foi aberta a tela `Pedir analise` para contestar a restricao da WABA. Texto recomendado em ingles:

```text
This WhatsApp Business Account was recently created for the EngageFit product through the official Twilio integration.

We are still in the initial technical setup and pilot phase. We have not sent bulk messages, spam, abusive automation, or used the account in any improper way. The phone number +55 11 5217-0912 is already connected and can receive messages from users, but the account is currently blocked from starting conversations and replying to customer messages.

In Twilio, outbound messages fail with error 63051. I am requesting a review of the restriction on the EngageFit WhatsApp Business Account, WABA ID 1361712461954280, so we can complete the setup and test messaging in compliance with WhatsApp Business policies.
```

Leitura atual:

- Nao e problema de codigo, template ou Content SID.
- O backend/Twilio conseguem criar mensagens, mas a Meta derruba a entrega de forma assincrona por causa da restricao da WABA.
- Enquanto o erro `63051` continuar, nao adianta insistir em novos testes reais pelo EngageFit, pois a mensagem sera aceita inicialmente pela API da Twilio e depois ficara `undelivered`.
- Um piloto sem verificacao empresarial poderia existir apenas se a WABA estivesse em conformidade. Como a WABA esta restrita, o piloto real esta bloqueado ate a Meta/Twilio liberarem a conta.
- Nao preencher CPF como se fosse CNPJ/registro empresarial na verificacao da empresa. Isso pode gerar inconsistencia na verificacao.
- Para o produto real, a tendencia e precisar formalizar empresa e concluir verificacao empresarial. MEI provavelmente nao e a melhor opcao para SaaS/desenvolvimento de software; avaliar ME/SLU/Simples com contador.

Templates oficiais Twilio criados/aprovados:

- `engagefit_falta_pouco`
  - Content SID: `HX0a74da5635b2401c1b0ce1769aaea1ac`
  - idioma: `pt_BR`
  - tipo: texto
- `engagefit_meta_atingida`
  - Content SID: `HX63d54262966db42c1641c32ab64b11c9`
  - idioma: `pt_BR`
  - tipo: texto
- `engagefit_sentimos_sua_falta`
  - Content SID: `HX198c7dcaf71ae42a733719eee86d5aa5`
  - idioma: `pt_BR`
  - tipo: texto
- Os tres templates aparecem como elegiveis/aprovados para:
  - `WhatsApp business initiated`
  - `WhatsApp user initiated`

Configuracao local/backend ja alinhada:

- `whatsapp_settings` local aponta para provider `twilio`, enabled `true`, sender `+551152170912`.
- `.env` local relevante:
  - `WHATSAPP_ALLOW_REAL_SEND=true`
  - `WHATSAPP_DEV_RECIPIENT_PHONE=+5511963834712`
  - `AUTOMATION_WORKER_ENABLED=true`
  - `EMAIL_ALLOW_REAL_SEND=true`
- A configuracao local existente foi migrada como `connection_mode=dedicated`; a conexao compartilhada aparece indisponivel enquanto as novas variaveis `WHATSAPP_PLATFORM_*` nao forem preenchidas.
- Nao expor credenciais Twilio/OpenAI em handoff, logs ou respostas.
- `scripts/demo-seed.mjs` configura os tres templates oficiais via API como `APPROVED`.
- `Makefile demo-reset` foi ajustado para limpar tambem tabelas de e-mail e automacao, preservando configuracoes.
- `internal/app/messages/message_usecases.go` foi ajustado para a ordem correta de variaveis Twilio:
  - `1`: `student_name`
  - `2`: `box_name`
  - `3`: `current_checkins`
  - `4`: `missing_checkins`
  - `5`: `target_checkins`
  - `6`: `reward_name`
  - `7`: `platform_name`
- `internal/domain/whatsapp_template_catalog.go` foi alinhado aos textos aprovados na Twilio.
- Testes executados e passando:

```bash
cd engage-fit-be
node --check scripts/demo-seed.mjs
go test ./...
make demo-reset-seed
```

Cenario demo atual para testes controlados:

- Frontend: `http://localhost:5173/#/whatsapp`
- Backend: `http://localhost:8080`
- Login demo:
  - email: `owner@example.com`
  - senha: `change-me`
- Campanha demo: `Brinde do mes 07/2026`
- Destinatario de desenvolvimento: `+55 11 96383-4712`
- Audiencias do seed:
  - `Meta atingida`: 1 mensagem
  - `Aluno em risco`: 1 mensagem
  - `Falta pouco`: 2 mensagens
- Enquanto a WABA estiver restrita, os envios reais devem falhar com `63051` mesmo que a API do backend retorne sucesso inicial.

Proximos passos obrigatorios:

1. Aguardar/acompanhar a analise da Meta para a WABA `1361712461954280`.
2. Abrir ticket na Twilio se a analise da Meta nao resolver:

```text
My WhatsApp Business Account is restricted in Meta Business Support.

WABA ID: 1361712461954280
Sender: +55 11 5217-0912
Twilio error: 63051
Meta restrictions shown:
- You can not start conversations with customers
- You can not respond to messages from customers
- You can not have phone numbers added to it

Meta says that my business solution provider needs to verify my business before I can start sending messages. Could you please verify what is required on Twilio side and help unlock or verify this WABA?
```

3. Se a Meta/Twilio exigirem verificacao empresarial, formalizar a empresa antes de insistir na WABA atual.
4. Depois que a WABA for destravada, repetir um teste real minimo:
   - inbound do celular para o sender oficial;
   - resposta livre dentro da janela de 24h;
   - envio de um template aprovado pelo EngageFit;
   - verificar status final na Twilio como `delivered` ou equivalente.

Pendencias tecnicas apos desbloqueio:

- Implementar webhook/status callback da Twilio para atualizar `message_recipients` de forma assincrona.
- Hoje o sistema pode marcar `sent` quando a Twilio aceita a criacao da mensagem, mesmo se a Meta depois retornar `undelivered`.
- A UI deve expor falhas posteriores de entrega com erro Twilio, especialmente `63051`, `undelivered` e rejeicoes de template/sender.
- Desenhar limites por plano somente depois do canal estar estavel:
  - limite mensal por box;
  - limite diario por box;
  - cooldown por aluno/audiencia;
  - bloqueio de disparo duplicado por campanha;
  - auditoria por destinatario;
  - possivel plano premium para numero dedicado por box.

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
- Entidades centrais: `Box`, `User`, `Student`, `Checkin`, `Campaign`, `CampaignGoal`, `Reward`, `CampaignProgress`, `RewardDelivery`, `WhatsappSettings`, `MessageTemplate`, `MessageCampaign`, `MessageRecipient`, `EmailSettings`, `EmailTemplate`, `EmailCampaign`, `EmailRecipient`, `AutomationRun`, `AutomationSchedule`, `Workout`, `WorkoutMessageDraft`, `WorkoutMessageRecipient`, `LLMGenerationLog`.
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
- E-mail personalizado, automacao diaria com agendamento/auditoria persistida e `Treino do dia` com rascunhos gerados por LLM ja foram implementados. Próximos focos devem partir das pendencias funcionais abaixo, nao da mensagem antiga de handoff.

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
- Treino do dia com mensagens geradas por LLM:
  - migration: `migrations/026_create_workouts.sql`.
  - entidades/tabelas: `workouts`, `workout_message_drafts`, `workout_message_recipients`, `llm_generation_logs`.
  - adapter OpenAI via `OPENAI_API_KEY`, `OPENAI_MODEL` e `OPENAI_TIMEOUT_SECONDS`; default de modelo: `gpt-4.1-mini`.
  - prompt com guardrails: mensagem curta, pratica, segura, sem dieta individual, sem orientacao medica, sem promessa de resultado e recomendando falar com o coach em caso de dor/duvida/adaptacao.
  - endpoints: `GET/POST /api/v1/workouts`, `GET/PUT/DELETE /api/v1/workouts/:id`, `GET/POST /api/v1/workouts/:id/message-drafts`, `PUT /api/v1/workout-message-drafts/:id`, `POST /api/v1/workout-message-drafts/:id/approve`, `POST /api/v1/workout-message-drafts/:id/send`, `GET /api/v1/workout-message-drafts/:id/recipients`.
  - envio WhatsApp exige rascunho aprovado manualmente (`approved`) e usa texto livre aprovado pelo owner; restricoes comerciais do WhatsApp/Twilio aparecem como falhas auditadas por destinatario.
  - audiencias reutilizadas: `all`, `inactive`, `almost_there`, `near_goal`, `achieved`; as audiencias de progresso exigem `campaign_id`.
  - `make demo-reset` limpa as tabelas novas e `scripts/demo-seed.mjs` cria um WOD demo sem chamar OpenAI.

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
- Treino do dia:
  - tela dedicada no menu lateral.
  - cadastro de WOD/treino com data, titulo, objetivo, movimentos principais e observacoes do coach.
  - geracao de rascunho por IA para audiencias existentes.
  - edicao do texto, aprovacao manual obrigatoria e envio pelo WhatsApp.
  - auditoria do ultimo envio por destinatario, incluindo falhas do provider.
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

## Treino do dia com mensagens geradas por LLM

Implementado em 2026-07-02 como MVP operacional com aprovacao manual antes do envio.

Fluxo atual:

- Owner cadastra o treino/WOD do dia.
- Owner escolhe audiencia (`all`, `inactive`, `almost_there`, `near_goal`, `achieved`).
- Backend gera rascunho com OpenAI usando guardrails de seguranca.
- Owner revisa/edita o texto e aprova manualmente.
- Sistema envia pelo WhatsApp usando texto livre aprovado e registra auditoria por destinatario.

Restricao importante de WhatsApp:

- O MVP envia texto livre aprovado pelo owner. Fora da janela de conversa de 24h, Twilio/Meta podem bloquear a mensagem; o erro fica registrado em `workout_message_recipients.error_message`.
- Proximo passo comercial recomendado: evoluir para templates aprovados/opt-in antes de usar em producao em massa.

Futuras evolucoes:

- Automacao/agendamento de geracao/envio de rascunhos do treino.
- Configuracoes por box para tom, tamanho maximo, assinatura, horarios permitidos e exigencia de aprovacao.
- Templates aprovados por categoria de treino ou fluxo de opt-in para respeitar WhatsApp fora da janela de 24h.
- Audiencia de alunos frequentes, caso a operacao valide a regra de frequencia.

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
8. Evoluir `Treino do dia` para fluxo comercial com templates aprovados/opt-in, agendamento e configuracoes por box.

## Arquivos importantes

Docs:

- `docs/system-design.md` (fonte canonica de arquitetura, negocio e manutencao)
- `docs/application-readiness-guide.md` (guia operacional da aplicacao)
- `docs/railway-deployment-checklist.md` (proxima fase de infraestrutura)
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
- `engage-fit-be/internal/app/workouts/workout_usecases.go`
- `engage-fit-be/internal/adapters/whatsapp/twilio_client.go`
- `engage-fit-be/internal/adapters/whatsapp/provider_gateway.go`
- `engage-fit-be/internal/adapters/whatsapp/safe_gateway.go`
- `engage-fit-be/internal/adapters/email/smtp_gateway.go`
- `engage-fit-be/internal/adapters/llm/openai_generator.go`
- `engage-fit-be/migrations/022_add_message_campaign_campaign_id.sql`
- `engage-fit-be/migrations/023_remove_evolution_provider.sql`
- `engage-fit-be/migrations/024_create_email_and_automation.sql`
- `engage-fit-be/migrations/025_create_automation_schedules.sql`
- `engage-fit-be/migrations/026_create_workouts.sql`

Frontend:

- `engage-fit-fe/src/pages/campaigns/CampaignsPage.tsx`
- `engage-fit-fe/src/pages/whatsapp/WhatsappPage.tsx`
- `engage-fit-fe/src/pages/email/EmailPage.tsx`
- `engage-fit-fe/src/pages/automation/AutomationPage.tsx`
- `engage-fit-fe/src/pages/workouts/WorkoutsPage.tsx`
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
- `engage-fit-be/migrations/026_create_workouts.sql`
- `engage-fit-be/test-data/totalpass-checkins-hit-goal.csv`

## Orientacao para iniciar novo chat

Mensagem sugerida:

```txt
Leia `.ai/handoff.md` e use `docs/system-design.md` como fonte canonica da arquitetura e das regras de negocio. Backend, frontend e PostgreSQL ja estao implantados no Railway production. Codigo homologado: backend `8b5443b` e frontend `81cee85`, com CI verde; pode haver commit documental posterior somente para este handoff. Dominio publico: `https://engage-fit-web-production.up.railway.app`; API e banco usam rede privada. Owner e PLATFORM_ADMIN foram homologados. Antes de nova evolucao, confirme no Railway: `OWNER_SETUP_ENABLED=false`, setup token removido/selado, Serverless desligado na API, Custom Start vazio e estrategia de pre-deploy restaurada/validada. Nao publique migrations novas enquanto o pre-deploy estiver desativado. Mantenha integracoes e envios reais desligados ate homologacao explicita.
```
