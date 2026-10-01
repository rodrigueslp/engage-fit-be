# SPEC-007 — Reservas, fila, falta e reposição

- Estado: `validated locally`
- Depende de: `SPEC-002`, `SPEC-003/A-C`, `SPEC-006`
- Migration: `065_create_reservations.sql`

## Resultado implementado

O módulo `reservations` é dono da ocupação das sessões publicadas. Owner e
coach da unidade podem reservar uma pessoa com matrícula e direito aplicáveis,
cancelar, operar a fila FIFO, registrar falta depois do início e usar uma falta
uma única vez como reposição. `scheduling` continua dono da capacidade e do
estado da sessão; `memberships` continua dono dos termos comerciais.

## Decisões conservadoras

- reserva só é criada antes do início e para sessão `scheduled`;
- capacidade é serializada por advisory lock da sessão; ao lotar, a reserva
  entra na fila sem consumir o direito;
- cancelamento de uma confirmação promove a primeira pessoa ainda elegível;
  entradas que perderam elegibilidade são canceladas de forma auditável;
- `confirmed` e `no_show` consomem limites; `waitlisted` e `cancelled` não;
- períodos usam o timezone da sessão, semana iniciando na segunda-feira;
- pacote conta todo o histórico da matrícula/direito; `daily_limit` é aplicado
  em conjunto com a regra principal;
- falta pode originar exatamente uma reposição ativa, vinculada à mesma pessoa
  e matrícula, sem consumir novamente a franquia; cancelar a reposição devolve
  esse crédito;
- cancelamento de sessão ou ocorrência pela agenda cancela confirmações e fila
  na mesma transação, sem promover nem enviar mensagens;
- notificações pertencem à `SPEC-014`; presença positiva pertence à `SPEC-008`.

## Contratos HTTP

- `POST /api/v1/units/:unit_id/reservations`;
- `POST /api/v1/units/:unit_id/reservations/:reservation_id/cancel`;
- `POST /api/v1/units/:unit_id/reservations/:reservation_id/no-show`;
- `GET /api/v1/units/:unit_id/reservations/summaries?session_ids=`;
- `GET /api/v1/units/:unit_id/scheduling/sessions/:session_id/reservations`;
- `GET /api/v1/units/:unit_id/people/:person_id/reservations?from=&to=`.

Mutações exigem `Idempotency-Key`, contexto tenant/unidade e CSRF. IDs de outro
tenant/unidade retornam `404`. A capability `FEATURE_RESERVATIONS_ENABLED`
depende de grade, pessoas e matrículas. Nenhum endpoint causa efeito externo.

## Persistência e privacidade

`reservations` guarda referências para sessão, pessoa, matrícula e direito,
estado, revisão e eventual origem de reposição. Índices parciais impedem duas
reservas ativas da mesma pessoa/sessão e dois usos ativos do mesmo crédito.
`reservation_command_requests` sustenta idempotência e
`reservation_audit_events` registra somente IDs, ação, razão e timestamps, sem
nome, contato ou payload HTTP. O export do titular inclui suas reservas; o nome
é sempre resolvido do perfil atual, não duplicado na tabela.

## Evidência de aceitação

- teste de aplicação cobre confirmação, replay idempotente, fila, promoção,
  falta e reposição;
- migration completa aplica em PostgreSQL vazio e o segundo run não aplica nada;
- suíte backend, vet, frontend typecheck/build e Playwright mobile cobrem a
  integração e operação da fila;
- flags permanecem desligadas por default e nenhum deploy/push é parte da spec.
