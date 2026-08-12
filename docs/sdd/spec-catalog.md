# Catálogo inicial de especificações

Este catálogo organiza dependências e paralelização. Datas só serão atribuídas
depois que uma spec alcançar `ready` e tiver tamanho estimado pela
implementação.

## Onda 0 — direção compartilhada

| Documento | Estado | Resultado |
|---|---|---|
| `ARCH-001` | `reviewed` | fronteiras, tenancy, identidade e estratégia de evolução |

## Onda 1 — fundação operacional

| Spec | Estado | Dependências | Resultado |
|---|---|---|---|
| `SPEC-001` Organização, unidades, modalidades e espaços | `implementing` (`001-A` validada; `001-B` ready) | `ARCH-001` | estrutura física e contexto operacional |
| `SPEC-002` Pessoa, identidades e vínculos | `draft` | `ARCH-001`, `SPEC-001` | identidade humana e cadastro mínimo |
| `SPEC-003` Planos, matrículas e direitos | `draft` | `ARCH-001`, `SPEC-001`, `SPEC-002` | oferta e acesso versionados |

As implementações podem preparar packages e protótipos em paralelo, mas
migrations e contratos compartilhados seguem a ordem `001 → 002 → 003`.

## Onda 2 — trilhas paralelas

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-004` Contratos e aceite | `002`, `003` | documento versionado, assinatura, cancelamento e auditoria |
| `SPEC-005` Financeiro do aluno | `002`, `003`, contrato interno de `004` | cobrança, recorrência, Pix, boleto, cartão e conciliação |
| `SPEC-006` Agenda e grade recorrente | `001`, contrato de direitos de `003` | horários, capacidade, coach e sessões de aula |

Depois que os contratos das specs `004`, `005` e `006` estiverem aceitos, três
agentes podem implementá-las em paralelo com ownership separado.

## Onda 3 — experiência operacional

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-007` Reservas e lista de espera | `003`, `006` | reserva, cancelamento, reposição, no-show e fila |
| `SPEC-008` Check-in e presença | `003`, `006`, `007` | entrada mensalista e corporativa com decisão de acesso |
| `SPEC-010` Expansão da PWA do aluno | contratos públicos de `004` a `008` | agenda, plano, contrato, pagamento e avisos no app atual |

## Onda 4 — substituição e operação assistida

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-009` Operação do dia | `004` a `008` | tela Hoje com aulas, cobranças e pessoas que exigem ação |
| `SPEC-011` Migração do sistema anterior | modelos estabilizados | importação, reconciliação, ensaio e corte do Tecnofit |
| `SPEC-012` Wellhub e TotalPass | `006` a `008` + documentação externa | check-in, booking, reconciliação e homologação |

O desenho de `SPEC-011` começa antes da Onda 4 para influenciar campos e
identificadores. A execução final depende de exportações reais e modelos
estabilizados.

`SPEC-012` terá um núcleo neutro e simuladores antes das credenciais. Adapters
reais só serão fechados com documentação e homologação dos parceiros.

## Ownership sugerido quando houver paralelização

- agente de arquitetura/integração: contratos compartilhados, migrations e
  revisão de tenancy;
- agente de contratos: `contracts` e sua UI;
- agente financeiro: `memberbilling` e gateways;
- agente de agenda: `scheduling` e `reservations`;
- agente de experiência: PWA, operação do dia e fluxos mobile;
- agente de qualidade: testes E2E, fixtures, migração e reconciliação.

Ownership é de módulo, não de camada. O mesmo agente entrega domínio, API,
persistência e frontend de sua fatia para reduzir handoffs incompletos.
