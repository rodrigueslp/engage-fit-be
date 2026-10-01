# SPEC-004/D — Ciclo jurídico posterior

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-16
- Depende de: `SPEC-004/A-B`, contrato interno de `SPEC-005`

## Resultado implementado

O módulo `contracts` registra ações jurídicas imutáveis de `amendment`,
`renewal`, `pause`, `resume` e `cancellation`, sempre vinculadas a uma matrícula,
pessoa e instância aceita do mesmo tenant. Cada ação preserva data efetiva,
período opcional, motivo, termos e instrução financeira, além de evento de
auditoria e revisão.

A entrega usa `financial_instruction.mode=manual_review` como default seguro:
registrar um fato jurídico não cancela matrícula, parcela, reserva ou acesso
silenciosamente. Integrações posteriores podem consumir a instrução por porta
pública e reconciliar os módulos explicitamente. A UI mobile mostra histórico
e cria novas ações somente quando existe contrato aceito.

## Aceite demonstrado

- FKs compostas de organização e unidade;
- somente contrato aceito da própria matrícula;
- uma pausa aberta por matrícula, datas e motivo validados;
- nenhum efeito financeiro ou operacional implícito.
