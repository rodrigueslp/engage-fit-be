# SPEC-004/C — PAR-Q e formulários sensíveis

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-16
- Depende de: `SPEC-002`, `SPEC-003`

## Resultado implementado

`healthforms` é um módulo independente de contratos. O owner registra PAR-Q
versionado para matrícula vigente, com seis respostas objetivas, detalhe
opcional, consentimento explícito e indicação conservadora de revisão humana.
As respostas são cifradas com associated data por organização/submissão; lista
e auditoria não expõem respostas, e toda abertura detalhada grava evento.

Consentimento pode ser revogado e uma nova submissão ativa do mesmo schema
pode ser feita depois. Retenção padrão é 730 dias, com consulta dry-run/aplicação
no repositório, e o export do titular inclui respostas decifradas somente no
fluxo protegido de privacidade. A capability fica desligada por padrão.

## Aceite demonstrado

- owner-only, tenant/unidade, matrícula `scheduled|active`;
- consentimento, revogação, trilha de acesso e retenção própria;
- criptografia em repouso e ausência de dados de saúde em `contracts`;
- formulário mobile, estados de alerta, erro e leitura auditada.
