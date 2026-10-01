# SPEC-004/B — Aceite eletrônico e reconciliação

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-16
- Depende de: `SPEC-004/A`

## Resultado implementado

O módulo `contracts` qualifica o signatário, cifra nome, e-mail e documento,
registra um envelope provider-neutral e reconcilia callbacks autenticados e
idempotentes. Um evento `accepted` revalida a matrícula, persiste sessão e
evidência minimizada, move a instância para `accepted` e grava evento durável
na mesma transação. Payload bruto e PII nunca entram em logs ou eventos.

Providers são allowlisted por `CONTRACT_SIGNATURE_PROVIDERS`; o webhook exige
`CONTRACT_SIGNATURE_WEBHOOK_TOKEN`. `local_signature_simulator` permite teste
sem rede e é recusado em produção. Um conector concreto, credenciais, cópia
externa do documento e homologação jurídica continuam como ativação
operacional específica do provider, não como efeito desta entrega local.

## Aceite demonstrado

- tenancy composta e `404` seguro;
- qualificação cifrada com associated data por envelope;
- deduplicação por `(organization, provider, provider_event_id)`;
- aceite externo preserva hash, declaração, pessoa e documento exatos;
- feature flag desligada por padrão e nenhum efeito de rede.
