# SPEC-008 — Check-in, presença e decisão de acesso

- Estado: `validated locally`
- Atualizada em: 2026-09-16
- Dependências: `SPEC-002`, `SPEC-003`, `SPEC-006`, `SPEC-007`

## Resultado entregue

A recepção agora decide e registra a entrada de uma pessoa conhecida na unidade
com um motivo explícito. A operação aceita mensalistas com matrícula vigente e
perfis locais Wellhub/TotalPass com vínculo ativo. Quando uma sessão é
selecionada, exige sessão publicada dentro da janela de 60 minutos antes até
120 minutos depois e reserva `confirmed`; fila de espera, cancelamento, ausência
de reserva ou sessão indisponível negam a entrada.

A decisão é sempre persistida, inclusive quando negativa. Somente uma decisão
positiva cria `attendance_record` e espelha um check-in na frequência legada.
Presenças são únicas por pessoa/dia na entrada geral ou pessoa/sessão em aula;
retries usam `Idempotency-Key` e fingerprint transacional.

## Contratos implementados

- `POST /api/v1/units/:unitId/attendance/admissions` — owner/coach, com
  `Idempotency-Key`, `person_id` e `session_id` opcional;
- `GET /api/v1/units/:unitId/attendance/today` — decisões positivas e negativas
  do dia local da unidade;
- capability `attendance`, desligada por padrão e dependente de pessoas,
  matrículas, grade, reservas e suas flags;
- migration `066_create_attendance_access.sql` com decisões, presenças,
  comandos idempotentes, FKs compostas, índices de unicidade e auditoria;
- export de privacidade inclui as decisões e presenças da pessoa.

## Limites deliberados

- Wellhub/TotalPass são avaliados pelo vínculo local já associado à pessoa. Não
  há chamada, confirmação ou efeito no provedor; isso pertence à `SPEC-012`.
- Uma aula sempre exige reserva confirmada; a portaria não cria walk-in nem
  ignora lista de espera/capacidade.
- A nova operação convive com importação, check-in manual e QR legados. Apenas
  admissões positivas usam `entry_method=access_decision` no histórico.
- Catraca física, biometria, QR corporativo e notificações permanecem fora.

## Critérios demonstrados

- mensalista vigente é liberado e mensalista sem matrícula é negado;
- vínculo corporativo ativo é liberado para entrada geral;
- vínculo inativo é negado;
- sessão libera somente reserva confirmada, nega espera ou ausência de reserva;
- tentativa negativa fica auditada sem criar presença;
- repetição do mesmo comando não duplica decisão ou presença;
- comando diferente no mesmo dia/sessão reaproveita a presença já registrada;
- tenant/unidade são aplicados na autorização, resolução e FKs;
- painel mobile exibe pessoa, sessão opcional, resultado explicável e histórico
  do dia para owner/coach.

## Evidência local

- `go test ./...` e `go vet ./...`;
- migrations `001..066` em PostgreSQL vazio e segundo run sem novas aplicações;
- teste PostgreSQL específico de decisão, presença, espelho de frequência,
  idempotência e deduplicação;
- TypeScript, build Vite e Playwright mobile em `360x800`.
