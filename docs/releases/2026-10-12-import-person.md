# Release — pessoa e unidade nas importações

Data: 2026-10-12 (America/Sao_Paulo). Publicação autorizada explicitamente
pelo usuário e concluída no Railway, ambiente production.

## Código e deployments

- Backend: `5603d10531ee882e746101429b8b8a9559b236d8`.
- API: `ca0976ea-dd1d-4cae-a440-82772b531509`, `SUCCESS`.
- Job SaaS: `9f582eeb-99b3-45c0-8ca0-c431fa4d02b0`, `SUCCESS`, mesma fonte do backend.
- Frontend permanece em `f44b205`, deployment `75b5a773-2015-483e-9a28-0cf1bc636ce2`.
- Schema permanece nas 77 migrations existentes, sem nova migration.

Novos alunos importados recebem pessoa e relacionamento na unidade da sessão.
Reimportações completam alunos legados sem pessoa. O vínculo é transacional
com alunos, check-ins e derivados; IDs existentes, consentimentos, perfis
revisados e relações bloqueadas/inativas são preservados. Não há merge,
matrícula ou atribuição automática de modalidade/plano.

## Configuração

O deploy anterior `9ea4fa96-0b8e-4250-a0f4-6a2d37a6d4ac` havia falhado na
validação de dependências da Grade: o Catálogo estava habilitado pelo painel,
mas `FEATURE_CATALOG_ENABLED=false` no servidor. A variável foi ajustada para
`true`, alinhando o servidor à ativação feita pelo usuário. Profissionais,
Grade e materializador já estavam configurados como `true` e agora estão
ativos no novo processo. As demais flags, provedores e segredos foram
preservados. Somente os metadados de build também foram atualizados na API e
no job. O teste de prontidão configurado continua em `/health/ready`, timeout
300 segundos; o predeploy mantém o migrador com limites de lock e statement.

## Evidências

- 408 testes do backend com `-race -count=1` e PostgreSQL 18; nenhum falhou.
- `go vet ./...` e `git diff --check` passaram.
- As 77 migrations passaram em banco vazio; segundo run: zero aplicadas.
- Backup lógico anterior ao deploy: 1.220.174 bytes, SHA-256
  `1537c2f7b764654b6ffd2a0bcc0302d6c25ea4a6ee665dd5b3169108f62a517c`.
- Restore desse backup em PostgreSQL 18 isolado confirmou as contagens.
- Antes e depois: 433 alunos, 433 pessoas, 433 relacionamentos, 12.895 check-ins.
- Zero alunos não anonimizados sem pessoa ou relacionamento; zero divergências
  de organização entre aluno, unidade e pessoa.
- 18 verificações HTTP autenticadas existentes passaram antes/depois, incluindo
  dashboard, alunos, importações e configurações; autenticação por cookie passou.
- Seis rotas de gestão passaram depois: organização, unidades, modalidades,
  espaços, profissionais e séries da grade. As capabilities públicas confirmam
  `catalog`, `scheduling` e `scheduling_grid` ativos.
- Logs consultados da API sem linhas de erro; materializador inicializado.

A validação HTTP em produção usou leituras. O comportamento de criação,
reimportação, rollback, privacidade e isolamento foi exercitado com fixtures
sintéticas no banco isolado. Backups e snapshots com dados sensíveis estão
protegidos fora dos repositórios, em `production-backups/release-20261012-import-person`.

## Recuperação

A versão anterior é `9fc92ca`, deployment da API
`314d5c23-12ab-4da7-935b-68e87390b022`, e deployment do job
`2a69e863-c52c-4550-b942-817ffec9e99c`. O schema é compatível com ambas as
versões. Em caso de regressão, reimplantar a imagem anterior pelo Railway,
mantendo a configuração válida de dependências e o migrador que conhece as
77 migrations; não desfazer migrations nem restaurar o banco sobre gravações
novas apenas para reverter esta mudança de aplicação. O rollback do código
retira a normalização automática das importações futuras, mas mantém as
pessoas e relações já persistidas.
