# Catálogo inicial de especificações

Este catálogo organiza dependências e paralelização. Datas só serão atribuídas
depois que uma spec alcançar `ready` e tiver tamanho estimado pela
implementação.

## Onda 0 — direção compartilhada

| Documento | Estado | Resultado |
|---|---|---|
| `REQ-001` | `reviewed` | primeiro escopo operacional consolidado com o Alados |
| `ARCH-001` | `reviewed` | fronteiras, tenancy, identidade e estratégia de evolução |

## Onda 1 — fundação operacional

| Spec | Estado | Dependências | Resultado |
|---|---|---|---|
| `SPEC-001` Organização, unidades, modalidades e espaços | `implementing` (`001-A/B` validadas localmente) | `ARCH-001` | estrutura física e contexto operacional |
| `SPEC-001/D` Serviços ofertáveis | `ready` | `SPEC-001/A-B` | catálogo simples reutilizável por plano e agenda |
| `SPEC-002` Pessoa, identidades e vínculos | `reviewed` | `ARCH-001`, `SPEC-001/A-B` | iniciativa de identidade humana dividida em slices |
| `SPEC-002/A` Fundação e cadastro manual | `validated locally` | `SPEC-001/A-B` | pessoa, backfill seguro e novo aluno compatível |
| `SPEC-003` Planos, matrículas, adicionais e direitos | `draft` | `ARCH-001`, `SPEC-001/B-D`, `SPEC-002` | oferta, composição e acesso versionados |

`001-D` continua pronta para implementação e `002-A` foi validada localmente
em seu worktree. As duas podem usar o mesmo número provisório de migration; o
integrador reserva os números finais na ordem de merge e repete os testes de
upgrade. A slice `001-C` de multiunidade pode ser adiada no piloto de uma
unidade; `001-D` deve estar integrada antes de planos referenciarem serviços.

## Onda 2 — trilhas paralelas

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-004` Contratos, PAR-Q e aceite | `002`, `003` | documentos e formulários separados, assinatura, cancelamento e auditoria |
| `SPEC-005` Financeiro do aluno | `002`, `003`, contrato interno de `004` | cobrança, recorrência, Pix, boleto, cartão e conciliação |
| `SPEC-006` Agenda, profissionais e grade recorrente | `001/B-D`, contrato de direitos de `003` | horários, capacidade, profissional e sessões de aula/serviço |

Depois que os contratos das specs `004`, `005` e `006` estiverem aceitos, três
agentes podem implementá-las em paralelo com ownership separado.

`SPEC-004` é uma iniciativa de experiência, não autorização para misturar
dados: contratos pertencem a `contracts`; respostas do PAR-Q pertencem a
`healthforms`, com permissões e retenção próprias.

## Onda 3 — experiência operacional

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-007` Reservas e lista de espera | `003`, `006` | reserva, cancelamento, reposição, no-show e fila |
| `SPEC-008` Check-in e presença | `003`, `006`, `007` | entrada mensalista e corporativa com decisão de acesso |
| `SPEC-010` Expansão da PWA do aluno | contratos públicos de `004` a `008` | agenda, plano, contrato, pagamento e avisos no app atual |
| `SPEC-013` Quiosque e vendas avulsas | `002`, ports financeiros de `005` | produtos e registro manual de vendas sem ERP avançado |
| `SPEC-014` Comunicados e notificações | `002`, `003`, PWA atual | publicações segmentadas, prioridade, agenda e histórico |

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
