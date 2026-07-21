# Checklist de deploy no Railway

Este checklist comeca somente depois da prontidao da aplicacao. Ele separa decisoes e operacoes de infraestrutura que nao devem ser acopladas ao startup da API.

## Projeto e ambientes

- [ ] Criar projetos/ambientes de homologacao e producao.
- [ ] Escolher plano e regiao compativeis com latencia, custo e requisitos de dados.
- [ ] Criar servicos separados para API, frontend e PostgreSQL.
- [ ] Configurar dominios, DNS e TLS; habilitar HSTS somente depois da validacao HTTPS completa.
- [ ] Definir quem pode alterar variaveis, executar deploy e acessar dados.

## Build e release

- [ ] Configurar o backend com seu `Dockerfile` e healthcheck `/health/live`.
- [ ] Configurar o frontend com seu `Dockerfile` e healthcheck `/health`.
- [ ] Injetar `BUILD_VERSION`, `BUILD_COMMIT` e `BUILD_TIME` no build da API.
- [ ] Executar `engagefit-migrate` como etapa de release separada antes da nova API.
- [ ] Confirmar que o startup da API nao executa migrations nem seed.
- [ ] Definir estrategia de rollback e confirmar compatibilidade da migration com a versao anterior antes de reverter codigo.

## Configuracao e segredos

- [ ] Preencher as variaveis production do guia `docs/application-readiness-guide.md`.
- [ ] Gerar JWT, administrador, setup token, token Prometheus e chaves de criptografia fora do repositorio.
- [ ] Manter `OWNER_SETUP_ENABLED=false` depois do onboarding.
- [ ] Manter todas as `FEATURE_*` desligadas ate homologacao individual.
- [ ] Manter `WHATSAPP_ALLOW_REAL_SEND=false` e `EMAIL_ALLOW_REAL_SEND=false` ate homologacao dos provedores.
- [ ] Configurar `AUTH_COOKIE_SECURE=true`, origens CORS e proxies confiaveis de acordo com os dominios finais.
- [ ] Validar rotacao de segredos com o binario dedicado antes de remover uma chave antiga.

## Banco, backup e restore

- [ ] Habilitar PostgreSQL gerenciado, volume e politica de backup/PITR adequada ao plano.
- [ ] Fazer um restore completo em ambiente isolado e registrar RPO/RTO medidos.
- [ ] Confirmar pool de conexoes da API em relacao ao limite do plano.
- [ ] Agendar retencao primeiro em dry-run; habilitar `--apply` somente depois de revisar contagens.
- [ ] Documentar procedimento de incidente, exportacao e exclusao de dados.

## Validacao e observabilidade

- [ ] Confirmar `/health/live`, `/health/ready` e `/health/build` pela URL publica.
- [ ] Confirmar usuario sem privilegios e encerramento gracioso durante um redeploy.
- [ ] Confirmar logs JSON em `stdout` e busca por `request_id`.
- [ ] Configurar OTLP/Grafana Cloud e alertas externos para os sinais documentados.
- [ ] Executar smoke sem efeitos externos e depois homologar cada gateway separadamente.
- [ ] Confirmar que nenhum segredo, PII ou corpo de mensagem aparece em logs, metricas ou traces.

## Go-live e rollback

- [ ] Registrar versoes das imagens, commit, migrations aplicadas e responsavel pelo go-live.
- [ ] Fazer uma janela de validacao com owner e PLATFORM_ADMIN.
- [ ] Validar importacao, campanha, dashboard, brinde, privacidade, troca de senha e logout.
- [ ] Definir criterios objetivos de rollback e canal de comunicacao do incidente.
- [ ] Testar rollback de aplicacao em homologacao e confirmar que nao depende de desfazer migration destrutiva.
