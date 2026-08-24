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
| `SPEC-001` Organização, unidades, modalidades e espaços | `implementing` (`001-A/B/D` validadas) | `ARCH-001` | estrutura física e contexto operacional |
| `SPEC-001/D` Serviços ofertáveis | `validated` | `SPEC-001/A-B` | catálogo simples reutilizável por plano e agenda |
| `SPEC-002` Pessoa, identidades e vínculos | `reviewed` | `ARCH-001`, `SPEC-001/A-B` | iniciativa de identidade humana dividida em slices |
| `SPEC-002/A` Fundação e cadastro manual | `validated` | `SPEC-001/A-B` | pessoa, backfill seguro e novo aluno compatível |
| `SPEC-003` Planos, matrículas, adicionais e direitos | `reviewed` | `ARCH-001`, `SPEC-001/B-D`, `SPEC-002` | iniciativa dividida em catálogo, composição, matrícula e acesso |
| `SPEC-003/A` Catálogo versionado de planos | `validated` | `SPEC-001/B-D`, `SPEC-002/A` | criação mobile, direitos base e publicação imutável |
| `SPEC-003/C` Matrícula em plano publicado | `validated` | `SPEC-002/A`, `SPEC-003/A` | adesão operacional simples, vigência e encerramento auditável |
| `SPEC-003/B` Adicionais e composição comercial | `validated` | `SPEC-003/A`, `SPEC-003/C` | adicionais na versão do plano (B1) e seleção na matrícula (B2) |

`001-D` e `002-A` estão integradas e homologadas manualmente no banco local. As
migrations finais são, respectivamente, `055_create_service_offerings.sql` e
`056_create_people_foundation.sql`. As slices `SPEC-003/A`, `SPEC-003/C` e
`SPEC-003/B` também estão integradas e homologadas localmente com
`057_create_membership_plan_catalog.sql`, `058_create_person_plan_enrollments.sql`
e `059_create_membership_addons_and_composition.sql`. As fundações de catálogo,
pessoas, planos e matrículas compostas estão concluídas. `001-C`, `002/B-C` e
`003/D` continuam adiadas ou em detalhamento.

## Onda 2 — trilhas paralelas

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-004/A` Fundação de contratos e emissão (`draft`) | `002/A`, `003/A-B-C` | template/versionamento, snapshot da matrícula, emissão e aceite por simulador local |
| `SPEC-004/B` Aceite eletrônico produtivo | contrato interno de `004/A` | qualificação das partes, provedor real, webhook/reconciliação e cópia |
| `SPEC-004/C` PAR-Q e formulários sensíveis | `002`, `003` | módulo `healthforms`, permissões, consentimento e retenção próprios |
| `SPEC-004/D` Ciclo jurídico posterior | `004/A-B`, contrato interno de `005` | aditivos, renovação, pausa/trancamento e cancelamento jurídico |
| `SPEC-005` Financeiro do aluno | `002`, `003`, contrato interno de `004/A` | cobrança, recorrência, Pix, boleto, cartão e conciliação |
| `SPEC-006` Agenda, profissionais e grade recorrente | `001/B-D`, contrato de direitos de `003` | horários, capacidade, profissional e sessões de aula/serviço |

`SPEC-004/A` possui spec executável em preparação, mas permanece `draft` e não
autoriza implementação. Depois de suas decisões bloqueadoras e portas estarem
aceitas, `004/B` pode ser detalhada. `004/C` pode ser especificada em paralelo
sobre `people` e `memberships`, porque pertence ao módulo independente
`healthforms`; a experiência pode ser coordenada, mas os dados não são
compartilhados. `004/D` depende das decisões jurídicas e financeiras
posteriores.

Depois que os contratos compartilhados de `004/A`, `005` e `006` estiverem
aceitos, agentes podem implementar slices com ownership separado.

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
