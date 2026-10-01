# SPEC-002/C — Identidades externas, revisão e consolidação segura

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-14
- Depende de: `SPEC-002/B`, experiência existente do atleta
- Migration: `063_complete_people_identity.sql`

## Resultado entregue

Uma pessoa pode possuir identidades verificadas Wellhub e TotalPass, únicas por
`(organization_id, provider, external_id)`. A migration converte os perfis
corporativos existentes sem associação por nome: conserva deterministicamente
o primeiro vínculo e cria revisão pendente para colisões.

Contas do atleta são ligadas automaticamente à pessoa apenas quando todos os
`athlete_student_links` confirmados dentro da organização apontam para uma
única pessoa. Mais de uma pessoa produz revisão, nunca merge automático.

O owner possui fila mobile de revisões e decide entre `pessoas distintas` e
`consolidar`. O merge humano é transacional, auditável e move vínculos de
unidade, `students`, identidades externas, endereço não conflitante e conta do
atleta. Ele é bloqueado por `person_merge_not_allowed` quando a origem possui
matrícula/contrato ou quando endereço/conta do atleta seriam ambíguos. A pessoa
origem permanece arquivada com `merged_into_person_id`, preservando
rastreabilidade.

## Contratos

- `POST /api/v1/units/:unit_id/people/:id/external-identities`
- `POST /api/v1/units/:unit_id/people/:id/merge`
- `GET /api/v1/units/:unit_id/people/identity-reviews`
- `POST /api/v1/units/:unit_id/people/identity-reviews/:review_id/resolve`

Conflitos geram `person_external_identity_conflict` e uma revisão pendente sem
ecoar o identificador. A exportação LGPD inclui identidades, unidades e conta;
anonimização final cria supressões, remove identidades/endereço e desvincula a
conta do atleta na mesma transação.

## Evidências

- migration em PostgreSQL demonstra backfill de colisão externa e de conta do
  atleta como duas revisões separadas, sem vínculo silencioso;
- teste de repositório demonstra identidade verificada, exportação,
  anonimização e merge seguro;
- Playwright cobre vínculo de identidade e descarte humano de revisão em
  360x800;
- suíte backend, typecheck e build foram aprovados localmente.
