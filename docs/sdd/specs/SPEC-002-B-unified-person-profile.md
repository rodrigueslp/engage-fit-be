# SPEC-002/B — Perfil unificado e migração gradual de leitura

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-14
- Depende de: `SPEC-002/A`, `SPEC-001/C`
- Migration: `063_complete_people_identity.sql`

## Resultado entregue

Owner e coach consultam uma lista de pessoas da unidade, com busca protegida
por nome, telefone ou CPF. A lista agrupa os perfis operacionais de origem e
expõe apenas contato e CPF mascarados. O perfil detalhado reúne cadastro,
unidades, perfis `Student`, conta do aluno e histórico. Coach recebe projeção
minimizada, sem CPF, endereço, identidade externa ou conta vinculada.

Owner edita nome, nome preferido, WhatsApp, e-mail, CPF, endereço e estado
cadastral com revisão otimista. Alterações de contato e nome atualizam as
projeções `students` ligadas na mesma transação, mantendo campanhas, retenção,
check-in e app compatíveis durante a migração gradual. CPF continua cifrado e
indexado por HMAC; telefone compartilhado exige revisão explícita.

Um CPF encontrado em outra unidade da organização não cria pessoa nova: o
owner pode criar o vínculo e um `Student box_member` compatível na unidade
selecionada. A autorização continua baseada na unidade ativa da sessão e em sua
allowlist.

## Contratos

- `GET /api/v1/units/:unit_id/people`
- `POST /api/v1/units/:unit_id/people/search`
- `GET /api/v1/units/:unit_id/people/:id`
- `PATCH /api/v1/units/:unit_id/people/:id`
- `POST /api/v1/units/:unit_id/people/:id/link`

Erros estáveis adicionados: `person_not_found`, `person_revision_conflict` e
`person_unit_already_linked`. CPF e telefone usados em busca permanecem no body,
nunca na URL.

## Evidências

- testes de serviço/repositório cobrem busca por CPF mascarada, edição,
  propagação compatível, revisão obsoleta e concorrência;
- teste HTTP cobre edição do owner e minimização do coach;
- Playwright em 360x800 cobre busca, perfil e edição;
- migration 063 passou em PostgreSQL vazio e segundo migrator sem aplicação.
