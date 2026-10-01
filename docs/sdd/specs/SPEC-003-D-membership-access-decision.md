# SPEC-003/D — Decisão de acesso, contadores e transição do legado

- Status: `validated locally`
- Data de validação: 2026-09-16
- Depende de: `SPEC-003/A-B-C`, `SPEC-006`, `SPEC-007`, `SPEC-008`

## Resultado implementado

O módulo `memberships` oferece uma decisão interna única para reserva e
check-in. Ela avalia organização, unidade, pessoa, vigência, modalidade ou
serviço, direito base ou adicional contratado, exigência de reserva, limite
diário, sessões por período e pacote total. O resultado contém razão estável,
matrícula, direito e saldo depois da ação quando calculável.

## Fonte de verdade de consumo

- reservas `confirmed` e `no_show` consomem a franquia da aula;
- fila e reserva cancelada não consomem;
- presença vinculada a uma reserva não é contada novamente;
- entrada geral conta `attendance_records` sem sessão;
- períodos usam o timezone IANA da unidade, semana iniciando na segunda-feira;
- `class_pack` usa toda a vigência da matrícula;
- reposição validada pela `SPEC-007` preserva a matrícula e ignora novo consumo.

## Compatibilidade

A migration `067_complete_membership_access.sql` marca uma coorte fechada de
mensalistas preexistentes, sem matrícula e com início conhecido, através de
`students.legacy_access_eligible`. Somente essa coorte recebe fallback de
check-in; novos cadastros não o herdam. A remoção futura do fallback depende de
a telemetria operacional chegar a zero.

## Validação local

- testes unitários de permissão, saldo, limite diário, reserva obrigatória,
  vigência, unidade, alvo e fallback;
- integração PostgreSQL do direito publicado até a decisão central;
- migration aplicada de `001` a `067` em banco vazio e segundo run sem novas
  aplicações;
- suítes backend e `go vet`, typecheck/build frontend e Playwright completo
  (`46 passed`, `2 skipped`), com revisão visual mobile; race detector ficou
  indisponível no ambiente sem CGO.
