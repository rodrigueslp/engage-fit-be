# Release de produção — 2026-10-09

## Escopo e configuração

Publicar o código acumulado dos dois repositórios e atualizar o banco de 52
para 77 migrations. Preservar WhatsApp, e-mail, automação, treinos e billing
conforme a configuração já homologada; manter as funcionalidades novas, os
simuladores e o materializador de agenda desligados nesta primeira publicação.
O worker de automação continua desligado. A ativação dos módulos novos é uma
operação posterior, com suas dependências e provedores revisados.

Versões anteriores: API e reconciliação `4aad802`; frontend `c46b4fa`.
O release usa backend antes do frontend e healthcheck da API `/health/ready`.

## Evidências do ensaio

- Backup lógico de produção restaurado em PostgreSQL 18.6 isolado, sem
  acesso externo das APIs nem execução de envios reais ou billing.
- Todas as primeiras 52 migrations possuem o mesmo checksum do código.
- As 25 migrations restantes aplicam e uma segunda execução aplica zero.
- Comparação de contagens e hashes das colunas legadas em 51 tabelas sem
  alteração de conteúdo. `updated_at` de boxes e students é excluído porque
  os backfills acionam seus triggers de atualização.
- Backfill reconciliado: quatro organizações/unidades; 424 alunos, pessoas e
  vínculos; quatro usuários operacionais ativos com memberships e acessos.
  Sem organização/pessoa ausente; controle de acesso permanece `disabled`;
  sem matrículas, cobranças de alunos ou overrides criados pelo release.
- API antiga e nova preservam 18 consultas autenticadas para os usuários
  operacionais disponíveis, incluindo cookies. Na comparação, campos novos
  são projetados fora e a ordenação de coleções é normalizada.
- Smoke com dados fictícios valida importação, campanhas, brindes,
  isolamento entre unidades, privacidade, troca de senha e logout.
- Regressão Playwright: 64 testes passaram; os dois testes contra API real
  foram executados separadamente e passaram, com o fluxo do owner em 360×800.
- Backend: 399 testes passaram com `-race -count=1` e PostgreSQL 18;
  `go vet ./...`, TypeScript e build Vite passaram. O build mantém o aviso
  conhecido de bundle JavaScript acima de 500 kB.

O primeiro ensaio reproduziu `SQLSTATE 0A000` em uma conexão antiga com
consultas preparadas após adicionar colunas. O ensaio repetido com
`default_query_exec_mode=exec` preservou as respostas da API anterior durante
o upgrade. Essa opção precisa estar ativa na API antiga antes das migrations.

Três fixtures de integração receberam telefone fictício conforme a migration
073; o teste de commerce deixou de fixar a última migration em 71. A integração
também revelou um filtro de presença que convertia `date` implicitamente no
PostgreSQL: o código agora usa `timestamp` explícito antes de `AT TIME ZONE`,
com regressão para 23:30 em São Paulo no dia UTC seguinte.
Os testes de convite e self-check-in agora usam transação com rollback e
hashes únicos para permitir repetir a suíte no mesmo banco isolado.

## Backup e recuperação

O plano Hobby recusou backup nativo e agendamento, que exigem Pro. Não houve
alteração de plano. Backups lógicos e snapshots de configuração permanecem
fora dos repositórios, em diretório local com permissão 0700 e arquivos 0600:
`production-backups/release-20261009/`. Não versionar esse diretório nem seus
logs: eles contêm dados pessoais e configurações de produção.

Guardar `rehearsal.dump`, `predeploy.dump`, `postdeploy.dump` e seus metadados de checksum/data.
A restauração foi validada por `pg_restore --exit-on-error --no-owner --no-acl`
em banco vazio isolado. Este backup é pontual; não existe uma política nova de
backup automático ou PITR. Um restore de produção precisa interromper as
escritas e considerar os registros posteriores ao backup.

## Procedimento do release

1. Validar as rotas atuais em produção com consultas de leitura.
2. Configurar `default_query_exec_mode=exec` sem mudar credenciais, flags ou
   provedores; redeploy da imagem atual antes do upgrade; aguardar readiness.
3. Fixar flags novas em `false` sem disparar deploy e proteger o predeploy com
   `PGOPTIONS="-c lock_timeout=5s -c statement_timeout=120s"`; timeout total
   300 segundos. Um bloqueio de migration deve abortar o novo deploy.
4. Gerar novo backup imediatamente antes da publicação e verificar restore.
5. Publicar backend; aguardar migrations e readiness; validar rotas e
   reconciliação do banco antes de publicar frontend.
6. Publicar frontend; validar HTTPS, proxy autenticado, assets/PWA, serviços
   e configuração efetiva. Registrar commits e deployments finais abaixo.

## Rollback

Não executar down migrations nem restaurar o backup para uma falha apenas de
aplicação. O banco expandido é compatível com a API anterior, conforme ensaio.
O migrador original de `4aad802` não reconhece migrations 053–077: um rollback
direto que execute esse predeploy falha.

Foi preparado, fora do repositório, `legacy-backend/` com o código da API
`4aad802` e os 77 arquivos SQL imutáveis. Seu migrador compilado aceita o banco
77 e aplica zero alterações. Em incidente, manter a conexão `exec`, publicar
esse pacote na API ou restaurar a imagem antiga com um predeploy que utilize
o migrador atual. Reverter frontend para a imagem `c46b4fa` se necessário.
Validar readiness e as consultas autenticadas antes de concluir a recuperação.

## Resultado da publicação

Publicado e validado em 2026-10-09, concluindo as verificações por volta de
23:22 America/Sao_Paulo (2026-10-10 02:22 UTC).

| Serviço | Commit publicado | Deployment Railway |
| --- | --- | --- |
| API | `9fc92ca06bdcd9b269e5fff12b27580eb0e13dd0` | `314d5c23-12ab-4da7-935b-68e87390b022` |
| Reconciliação SaaS | `9fc92ca06bdcd9b269e5fff12b27580eb0e13dd0` | `2a69e863-c52c-4550-b942-817ffec9e99c` |
| Frontend | `f44b205a5e69ad2072e2bdb760d3f5949d61fc37` | `75b5a773-2015-483e-9a28-0cf1bc636ce2` |

Todos os deployments terminaram em `SUCCESS`. O job agendado registrou
conclusão normal. As 77 migrations possuem checksum correto e os registros
revisados da API e do job não mostraram erro de aplicação. As 18 consultas
autenticadas passaram antes do release, depois do backend com frontend antigo
e depois do frontend novo; sessões por cookie continuam aceitas.

Durante o rollout, duas importações normais acrescentaram nove alunos e 771
check-ins desde o backup final anterior ao deploy. A comparação por ID com o
backup confirmou todos os registros anteriores, com conteúdo de alunos,
check-ins e importações preservado. As demais diferenças foram apenas
timestamps de billing e recálculo de campanhas. Por isso, o hash integral das
tabelas sob escrita mudou; a comparação não foi tratada como perda de dados.

A conferência final encontrou 433 alunos, 12.895 check-ins, 433 pessoas e
vínculos, quatro organizações/unidades e quatro usuários operacionais com
acesso migrado. Nenhuma pessoa/organização ausente, matrícula nova, cobrança de
aluno, override ou controle físico ativado. Flags e permissões de envio já
homologadas foram preservadas; módulos novos e workers continuam desligados.

O JavaScript publicado corresponde byte a byte ao build validado
(`index-bcjDHT65.js`); manifest, service worker e favicon também correspondem.
HTTPS, saúde pública e login em 360×800 foram verificados, sem erro 5xx ou
overflow horizontal nesse teste. Não foram executadas importações sintéticas,
redefinições de senha ou disparos de provedores em produção.

Backups anteriores e posteriores ao upgrade tiveram restauração completa e
checksum verificados. O backup posterior, com esquema 77 e os novos registros,
foi criado em 2026-10-10 02:22:04 UTC. Seus metadados protegidos são a referência
para recuperação, junto com as chaves de criptografia salvas separadamente.

O autodeploy da API e do job observa `cmd/**`, `internal/**`, `migrations/**`,
`go.mod`, `go.sum`, `Dockerfile` e `.dockerignore`. Alterações apenas nesta
documentação não reiniciam produção. Mudanças de variáveis continuam exigindo
controle explícito do deploy. O frontend mantém seu autodeploy existente.

Limitações: backup nativo/agendado e PITR não foram habilitados no Hobby;
o rollback de aplicação foi ensaiado na cópia isolada, sem reverter o release
saudável em produção. Recursos novos precisam de ativação e configuração
próprias antes do uso, especialmente pessoas, contratos e cobranças de alunos.

## Referências operacionais

- [Railway: healthchecks](https://docs.railway.com/deployments/healthchecks)
- [Railway: predeploy](https://docs.railway.com/deployments/pre-deploy-command)
- [Railway: watch paths](https://docs.railway.com/builds/build-configuration)
- [pgx: cache de consultas e alterações de esquema](https://github.com/jackc/pgx/wiki/Automatic-Prepared-Statement-Caching)
