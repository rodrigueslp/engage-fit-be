# EngageFit — guia do que foi implementado e como operar

Atualizado em: 2026-07-20

Este é o guia operacional da preparação do EngageFit no nível da aplicação. O modelo completo de arquitetura e negócio está em `docs/system-design.md`; o `.ai/handoff.md` continua sendo o histórico técnico detalhado.

## O que você precisa saber primeiro

1. Existem dois tipos de usuário diferentes:
   - `OWNER`: dono de uma academia; só enxerga os dados da própria academia.
   - `PLATFORM_ADMIN`: administrador do EngageFit; enxerga academias, limites e conexões na tela `Admin > Governança WhatsApp`.
2. `owner@example.com` é somente a conta demo do CrossFit Alados. Ela não é administradora da plataforma.
3. O administrador da plataforma é criado pelas variáveis `PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD`.
4. Em produção, a API não inicia com JWT fraco, administrador ausente ou criptografia de dados ausente.
5. Migrations não são executadas automaticamente ao iniciar a API. Execute `engagefit-migrate up` antes de cada nova versão.
6. Envio real de WhatsApp e e-mail fica bloqueado por padrão em desenvolvimento.
7. A automação fica desligada por padrão e precisa de `AUTOMATION_WORKER_ENABLED=true` para processar agendas.
8. O comando `make demo-reset-seed` apaga todas as academias e recria a massa demo. Nunca deve ser usado em produção.
9. Logs, métricas e traces podem ser enviados por OpenTelemetry ao Grafana. Railway fica responsável pelas métricas da máquina/container.
10. Opt-out é efetivo: um aluno marcado como “Não contatar” não entra nos públicos de WhatsApp, e-mail ou Treino do dia.

## Visão geral do que foi entregue

### Segurança e acesso

- Criação de academia e owner em uma única transação.
- Endpoint de onboarding controlado por `OWNER_SETUP_ENABLED` e `OWNER_SETUP_TOKEN`.
- Rate limit de login e onboarding por IP e identidade normalizada.
- Limites de body, upload, linhas, colunas e conteúdo descomprimido de XLSX.
- JWT vinculado ao usuário, academia, papel e `auth_version` persistido no banco.
- Logout, troca de senha e redefinição administrativa revogam imediatamente tokens anteriores.
- Owner pode trocar a própria senha.
- Platform admin pode redefinir a senha do owner com motivo e auditoria.
- Separação entre rotas de owner e rotas de administrador.
- Correções de isolamento multitenant em alunos, campanhas, metas, progresso, brindes e históricos de destinatários.
- E-mail de login é normalizado, portanto diferenças entre maiúsculas/minúsculas e espaços não bloqueiam mais a autenticação.

Limitação conhecida: o rate limit é mantido na memória da API. Ele é adequado ao piloto com uma réplica; múltiplas réplicas exigirão Redis ou outro armazenamento compartilhado.

### Runtime e banco de dados

- Servidor HTTP com timeout de leitura, escrita, cabeçalho e conexão ociosa.
- Encerramento gracioso ao receber `SIGTERM` ou `SIGINT`.
- Worker também respeita o encerramento da aplicação.
- Pool PostgreSQL configurável.
- Verificação do banco no startup e na readiness.
- A variável `PORT`, usada por provedores como Railway, funciona como fallback de `HTTP_PORT`.
- Container final executa com usuário sem privilégios e contém certificados CA e dados de timezone.

Endpoints de saúde:

- `GET /health/live`: o processo está vivo.
- `GET /health/ready`: a aplicação está pronta e o PostgreSQL responde.
- `GET /health`: alias simples de liveness.

### Migrations versionadas

O migrator registra cada migration na tabela `schema_migrations`, incluindo checksum. Isso impede que um SQL já aplicado seja alterado silenciosamente.

Características:

- execução em ordem numérica;
- transação individual por migration;
- advisory lock contra duas releases migrando simultaneamente;
- segunda execução idempotente;
- recusa banco legado sem histórico, a menos que seja feito baseline consciente.

Comandos locais:

```bash
make migrate-status
make migrate-up
```

Em uma imagem de produção:

```bash
/usr/local/bin/engagefit-migrate up
/usr/local/bin/engagefit-api
```

O `baseline` não aplica SQL. Ele apenas declara que um banco legado já possui determinado schema. Nunca use em banco vazio ou sem conferir manualmente a estrutura:

```bash
make migrate-baseline VERSION=32
```

### Criptografia das credenciais

Credenciais dedicadas de WhatsApp e senhas SMTP são cifradas antes de chegar ao PostgreSQL com AES-256-GCM. A autenticação do ciphertext é vinculada à academia e ao campo, impedindo copiar um segredo cifrado para outro tenant.

Formato de configuração:

```env
DATA_ENCRYPTION_ACTIVE_KEY_ID=primary
DATA_ENCRYPTION_KEYS=primary:<chave-base64-de-32-bytes>
```

Para gerar uma chave:

```bash
openssl rand -base64 32
```

Para converter plaintext legado ou trocar a chave ativa:

```bash
make rotate-secrets
```

Rotação correta:

1. Adicione a chave nova sem remover a antiga do keyring.
2. Defina a nova como `DATA_ENCRYPTION_ACTIVE_KEY_ID`.
3. Execute `make rotate-secrets`.
4. Atualize todas as instâncias da API.
5. Só então remova a chave antiga.

Perder todas as chaves que cifraram os dados torna as credenciais irrecuperáveis. Elas precisam estar no gerenciador de segredos do ambiente e no procedimento de recuperação da empresa.

### Governança de WhatsApp

Cada academia possui política própria com:

- limite diário e mensal;
- limite por disparo;
- orçamento diário e mensal;
- custo unitário estimado;
- timezone;
- bloqueio administrativo.

Academias usando o número compartilhado também respeitam uma política global do EngageFit. A reserva de limite acontece em transação antes da chamada ao provedor, evitando estouro por envios concorrentes.

A tela administrativa mostra:

- academias cadastradas;
- quantas usam o número EngageFit;
- quantas usam conexão dedicada;
- consumo, limites e bloqueios;
- configuração efetiva da conexão de cada academia.

Modos de conexão:

- `platform`: usa conta, remetente e Content SIDs definidos em `WHATSAPP_PLATFORM_*` no backend.
- `dedicated`: usa a conta e o remetente próprios daquela academia, salvos cifrados no banco.

O owner vê a conexão efetiva, mas não recebe a credencial. Alterações sensíveis ficam com o platform admin.

### Automações e idempotência

As agendas podem executar:

- rotina diária completa;
- apenas recálculo;
- envio de “falta pouco”;
- envio de meta atingida;
- envio para alunos inativos.

O worker faz claim atômico no PostgreSQL. Mesmo com várias réplicas, apenas uma deve executar cada slot.

Estratégia de segurança: `at-most-once`. Se o processo morrer depois de chamar um provedor e antes de registrar o resultado, a aplicação não repete automaticamente um envio de resultado incerto. A execução fica para revisão manual.

- `AUTOMATION_STALE_RUN_MINUTES`: quando um run preso em `running` passa a ser considerado falho.
- `AUTOMATION_CATCHUP_WINDOW_MINUTES`: atraso curto ainda permitido após startup/deploy.
- Execuções manuais aceitam `Idempotency-Key`; o frontend gera uma chave por clique.

### Observabilidade

A aplicação produz:

- logs JSON estruturados em `stdout`;
- métricas HTTP, runtime Go, pool PostgreSQL e automações;
- traces HTTP, GORM, Twilio, Meta e OpenAI;
- correlação por `request_id`, `trace_id` e `span_id`.

Os logs não devem conter corpos de mensagem, senha, token, credencial, telefone, e-mail, URL concreta ou parâmetros SQL.

Stack local:

```bash
make observability-up

OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
make backend-run
```

Abra `http://localhost:3000`, entre com `admin` / `admin` e acesse `EngageFit - Visão geral`.

Para parar:

```bash
make observability-down
```

No Railway, a recomendação é:

- Railway: CPU, RAM, disco, rede, reinícios e logs imediatos.
- Grafana Cloud gratuito: logs de maior retenção, métricas, traces, dashboards e alertas.
- API: envio OTLP direto usando `OTEL_EXPORTER_OTLP_ENDPOINT` e `OTEL_EXPORTER_OTLP_HEADERS`.

### Privacidade e LGPD

Na tela `Alunos`, o owner pode:

- registrar “Autorizado”, “Não contatar” ou “Não informado”;
- exportar os dados de um aluno em JSON;
- anonimizar os dados com confirmação e motivo.

A exportação contém cadastro, check-ins, progresso e histórico de comunicações. A operação é auditada.

A anonimização:

- remove nome, e-mail e telefone;
- limpa destinos e erros dos históricos de comunicação;
- marca o aluno como opt-out;
- preserva check-ins e estatísticas de forma anônima;
- grava uma supressão hash para evitar que a mesma identidade seja recriada na próxima importação.

Retenção inicial:

| Dados | Prazo padrão |
|---|---:|
| Destinatários de mensagens/e-mails/treinos | 365 dias |
| Logs de geração por LLM | 90 dias |
| Execuções de automação | 180 dias |
| Importações e check-ins | 730 dias |
| Auditoria de privacidade | 1825 dias |
| Supressões de identidade | Sem expiração automática |

Primeiro simule:

```bash
make privacy-retention-dry-run
```

Somente depois de conferir as contagens:

```bash
make privacy-retention-apply
```

Esse segundo comando é destrutivo e os registros apagados só podem ser recuperados por backup.

Detalhes jurídicos e operacionais: `docs/privacy-runbook.md`.

### Testes e CI

Backend CI executa:

- verificação de módulos e formatação;
- `go vet`;
- 32 migrations em PostgreSQL 16 vazio;
- segunda execução idempotente das migrations;
- testes com race detector e integração PostgreSQL;
- smoke HTTP;
- build dos quatro binários operacionais;
- validação dos scripts Node.

Frontend CI executa `npm ci`, build TypeScript/Vite, Playwright mockado e Playwright real contra API/PostgreSQL com gateways externos desligados.

O smoke cobre:

- liveness e readiness;
- onboarding de duas academias;
- login e `/auth/me`;
- importação de aluno/check-in;
- campanha, meta, brinde, recálculo e dashboard;
- tentativas de acesso cruzado entre academias;
- opt-out, exportação, anonimização e bloqueio da reimportação;
- logout e invalidação do token.

Comandos locais principais:

```bash
cd engage-fit-be
go test ./...
go vet ./...

cd ../engage-fit-fe
npm run build
```

## Como iniciar o ambiente local

Em um terminal:

```bash
cd engage-fit-be
make up
make migrate-up
make backend-run
```

Em outro terminal:

```bash
cd engage-fit-fe
npm run dev
```

URLs usuais:

- frontend: `http://localhost:5173`;
- API: `http://localhost:8080`;
- readiness: `http://localhost:8080/health/ready`.

## Como recriar a demonstração do zero

Com PostgreSQL e API em execução:

```bash
cd engage-fit-be
make demo-reset-seed
```

Resultado esperado:

- academia `CrossFit Alados`;
- owner `owner@example.com`;
- senha `change-me`;
- duas campanhas ativas;
- alunos e check-ins TotalPass;
- metas, brindes e campanhas WhatsApp;
- e-mail mock;
- agenda de automação pausada;
- treino demo.

`demo-reset-seed` remove todas as academias existentes no banco apontado pelo `DATABASE_URL`. Confira o banco antes de executar.

Para apenas adicionar/verificar a demo sem limpar previamente:

```bash
make demo-seed
```

## Variáveis obrigatórias em produção

Esta é a configuração mínima que você precisa resolver antes de iniciar a API em `APP_ENV=production`:

```env
APP_ENV=production
DATABASE_URL=postgres://...
JWT_SECRET=<aleatório-com-ao-menos-32-caracteres>

PLATFORM_ADMIN_NAME=Administrador EngageFit
PLATFORM_ADMIN_EMAIL=admin@seudominio.com
PLATFORM_ADMIN_PASSWORD=<senha-forte>

OWNER_SETUP_ENABLED=false

DATA_ENCRYPTION_ACTIVE_KEY_ID=primary
DATA_ENCRYPTION_KEYS=primary:<base64-de-32-bytes>

AUTOMATION_WORKER_ENABLED=false
EMAIL_ALLOW_REAL_SEND=false
WHATSAPP_ALLOW_REAL_SEND=false
```

Depois, habilite cada capacidade de maneira consciente:

- onboarding público: somente com `OWNER_SETUP_ENABLED=true` e token de 32+ caracteres;
- Prometheus público: prefira não expor; se necessário, use Bearer token de 32+ caracteres;
- automação: habilite depois de homologar agendas e templates;
- e-mail real: habilite após testar remetente e destinatários;
- WhatsApp real: habilite após validar conta, remetente, templates, limites e opt-out;
- OpenAI: configure a chave apenas se Treino do dia com IA for utilizado.

## Ordem recomendada de uma release

1. CI verde no backend e frontend.
2. Backup/restore disponível no ambiente de destino.
3. Configurar segredos e variáveis da nova versão.
4. Construir/publicar a imagem.
5. Executar `engagefit-migrate up` como etapa única de release.
6. Iniciar a API.
7. Aguardar `/health/ready` retornar `200`.
8. Publicar/atualizar o frontend.
9. Executar um smoke sem envio real.
10. Observar erros, latência e readiness durante a estabilização.

Se a migration falhar, não inicie a nova API. Se a API falhar depois da migration, reverta a versão da aplicação somente após confirmar que a migration é compatível com a versão anterior.

## Problemas comuns

### A API não inicia em produção

Leia o primeiro erro do processo. A aplicação falha cedo quando falta banco, JWT forte, platform admin ou chave de criptografia.

### Readiness retorna `503`

O PostgreSQL não está respondendo dentro do timeout. Verifique `DATABASE_URL`, rede, limite de conexões e estado do banco.

### Owner recebe `401` depois de trocar a senha

É esperado: todos os tokens anteriores foram revogados. Faça login novamente.

### Setup de owner retorna `404`

`OWNER_SETUP_ENABLED` está desligado. Em produção, isso é o comportamento seguro padrão.

### Setup retorna `401`

O header `X-Setup-Token` não corresponde a `OWNER_SETUP_TOKEN`.

### Login/setup retorna `429`

O rate limit foi atingido. Aguarde o valor de `Retry-After`. Reiniciar a API limpa o limitador em memória, mas isso não deve ser usado como solução operacional.

### Migration informa banco não vazio sem histórico

Não force. Confira a estrutura real e só use baseline se o banco já possuir exatamente as migrations declaradas.

### Credencial não pode ser decifrada

Não altere ou remova chaves às cegas. Recoloque no keyring a chave indicada no envelope e execute a rotação corretamente.

### Automação ficou em `running`

Ela não será repetida automaticamente por segurança. Depois do período de stale, revise possíveis efeitos externos e faça nova execução manual com uma nova chave.

### Mensagem aparece como enviada, mas não chegou

Hoje o sistema registra primeiro a aceitação do provedor. A confirmação final assinada via StatusCallback da Twilio ainda é uma evolução pendente.

## O que ainda não está resolvido por código

Antes do go-live ainda será necessário decidir/configurar:

- projeto, plano e região no Railway;
- domínio, DNS e TLS;
- PostgreSQL gerenciado, backups, PITR e teste de restauração;
- ambientes e estratégia de rollback/deploy;
- configuração real do número/conta WhatsApp e Content SIDs;
- Grafana Cloud e alertas externos;
- políticas de acesso aos painéis e logs;
- política de privacidade, termos e contrato de tratamento validados juridicamente;
- procedimento organizacional de incidente;
- homologação com dados e usuários do CrossFit Alados.

Evoluções de aplicação recomendadas depois desta fase:

- StatusCallback assinado da Twilio para distinguir aceito, entregue e não entregue;
- reconciliação do custo real do provedor;
- rate limit compartilhado quando houver múltiplas réplicas;
- recuperação de senha por token/e-mail;
- regressao visual e validacao em aparelhos/navegadores adicionais;
- paginação/filtros server-side quando o volume crescer.

## Checklist antes de entregar ao CrossFit Alados

- [ ] CI verde nos dois repositórios.
- [ ] Backup e restauração testados.
- [ ] Todas as 32 migrations aplicadas.
- [ ] Platform admin com senha exclusiva e forte.
- [ ] Senha demo `change-me` substituída.
- [ ] Onboarding público desligado.
- [ ] Chaves de criptografia armazenadas com segurança.
- [ ] Número, credenciais e três Content SIDs homologados.
- [ ] Limites diários, mensais, por disparo e orçamento configurados.
- [ ] Opt-in/opt-out e abordagem jurídica validados.
- [ ] Envio de teste feito somente para destinatários autorizados.
- [ ] Automação inicialmente desligada ou agendas revisadas uma a uma.
- [ ] Grafana/Railway recebendo sinais e alertas básicos ativos.
- [ ] Smoke pós-release concluído sem envio real.
- [ ] Responsável operacional sabe consultar logs pelo `request_id`.

## Onde encontrar mais detalhes

- Manual canônico de engenharia: `docs/system-design.md`.
- Histórico técnico completo: `.ai/handoff.md`.
- Checklist separado da infraestrutura: `docs/railway-deployment-checklist.md`.
- Governança WhatsApp: `.ai/messaging-governance.md`.
- Observabilidade local: `observability/README.md`.
- Privacidade/LGPD: `docs/privacy-runbook.md`.
- Variáveis disponíveis: `.env.example`.
- Comandos operacionais: `Makefile`.
