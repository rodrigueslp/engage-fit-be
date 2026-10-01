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
| `SPEC-001` Organização, unidades, modalidades e espaços | `validated` (`001-A/B/C/D`) | `ARCH-001` | estrutura física e contexto operacional multiunidade |
| `SPEC-001/D` Serviços ofertáveis | `validated` | `SPEC-001/A-B` | catálogo simples reutilizável por plano e agenda |
| `SPEC-002` Pessoa, identidades e vínculos | `validated locally` (`002-A/B/C`) | `ARCH-001`, `SPEC-001/A-B-C` | perfil humano unificado, edição, identidades externas e consolidação segura |
| `SPEC-002/A` Fundação e cadastro manual | `validated` | `SPEC-001/A-B` | pessoa, backfill seguro e novo aluno compatível |
| `SPEC-002/B` Perfil unificado e edição | `validated locally` | `SPEC-002/A`, `SPEC-001/C` | busca protegida, perfil, edição e vínculo multiunidade compatível |
| `SPEC-002/C` Identidades e consolidação | `validated locally` | `SPEC-002/B` | Wellhub/TotalPass, conta do atleta, revisão humana e merge seguro |
| `SPEC-003` Planos, matrículas, adicionais e direitos | `validated locally` (`003/A-B-C-D`) | `ARCH-001`, `SPEC-001/B-D`, `SPEC-002` | catálogo, composição, matrícula e acesso integrados |
| `SPEC-003/A` Catálogo versionado de planos | `validated` | `SPEC-001/B-D`, `SPEC-002/A` | criação mobile, direitos base e publicação imutável |
| `SPEC-003/C` Matrícula em plano publicado | `validated` | `SPEC-002/A`, `SPEC-003/A` | adesão operacional simples, vigência e encerramento auditável |
| `SPEC-003/B` Adicionais e composição comercial | `validated` | `SPEC-003/A`, `SPEC-003/C` | adicionais na versão do plano (B1) e seleção na matrícula (B2) |
| `SPEC-003/D` Decisão de acesso e contadores | `validated locally` | `SPEC-003/A-B-C`, `SPEC-006-008` | decisão central, saldo sem dupla contagem e transição fechada do legado |

`001-D` e `002-A` estão integradas e homologadas manualmente no banco local. As
migrations finais são, respectivamente, `055_create_service_offerings.sql` e
`056_create_people_foundation.sql`. As slices `SPEC-003/A`, `SPEC-003/C` e
`SPEC-003/B` também estão integradas e homologadas localmente com
`057_create_membership_plan_catalog.sql`, `058_create_person_plan_enrollments.sql`
e `059_create_membership_addons_and_composition.sql`. `003/D` foi validada com
`067_complete_membership_access.sql`. Catálogo, planos, matrículas compostas e
decisão de acesso estão concluídos. `001-C` foi validada
localmente com a migration `062_create_multi_unit_access.sql`; `002/B-C` foram
validadas localmente com `063_complete_people_identity.sql`.

## Onda 2 — trilhas paralelas

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| `SPEC-004/A` Fundação de contratos e emissão (`validated`) | `002/A`, `003/A-B-C` | template/versionamento, snapshot da matrícula, emissão e aceite por simulador local |
| [`SPEC-004/B` Aceite eletrônico e reconciliação](specs/SPEC-004-B-electronic-acceptance.md) — `validated locally` | contrato interno de `004/A` | qualificação cifrada, envelope provider-neutral e webhook idempotente; conector real segue desabilitado |
| [`SPEC-004/C` PAR-Q e formulários sensíveis](specs/SPEC-004-C-health-forms.md) — `validated locally` | `002`, `003` | módulo `healthforms`, respostas cifradas, consentimento, auditoria e retenção próprios |
| [`SPEC-004/D` Ciclo jurídico posterior](specs/SPEC-004-D-legal-lifecycle.md) — `validated locally` | `004/A-B`, contrato interno de `005` | aditivo, renovação, pausa/retomada e cancelamento auditáveis, sem efeito financeiro implícito |
| [`SPEC-005` Financeiro do aluno](specs/SPEC-005-member-billing.md) — `validated locally` | `002`, `003`, contrato interno de `004/A` | cobrança única/recorrente, Pix, boleto, cartão, baixa manual, estorno e conciliação por simulador local |
| [`SPEC-006` Agenda, profissionais e grade recorrente](specs/SPEC-006-scheduling-professionals-recurring-grid.md) — `validated locally` | `001/B-D`, contratos públicos de `003` | profissionais, grade, preview, publicação, sessões, materialização e exceções validados localmente |
| [`SPEC-006/A` Fundação de profissionais](specs/SPEC-006-A-scheduling-professionals.md) — `validated` | `ARCH-001`, `SPEC-006` | cadastro operacional, conta opcional, leitura do coach e privacidade |
| [`SPEC-006/B` Grade recorrente e preview](specs/SPEC-006-B-scheduling-recurring-grid-preview.md) — `validated locally` | `SPEC-001/B-D`, `SPEC-006/A` | séries semanais em rascunho, conflitos e ocorrências virtuais |

`SPEC-004` está `validated locally`: A preserva a fundação; B acrescenta
qualificação cifrada e reconciliação provider-neutral; C mantém PAR-Q no
módulo isolado `healthforms`; D registra o ciclo jurídico sem efeitos
financeiros implícitos. A migration integrada é
`069_complete_contract_experience.sql`. Produção continua sem provider,
credenciais ou chancela jurídica: o simulador não abre rede e é proibido em
produção, e a ativação concreta exige escolha e homologação do fornecedor.

`SPEC-005` está `validated locally` com a migration
`068_create_member_billing.sql`. O módulo `memberbilling` materializa cobranças
únicas e recorrentes, baixa manual, conciliação e estorno, mantendo ledger,
idempotência, tenancy e privacidade. O painel permite valores parciais, mostra o
ledger por parcela e inclui homologação visual de reenvio idempotente somente em
desenvolvimento local. Pix, boleto e cartão usam somente o simulador local;
gateway, credenciais e efeitos reais continuam desabilitados.

Depois que os contratos compartilhados de `004/A`, `005` e `006` estiverem
aceitos, agentes podem implementar slices com ownership separado.

`SPEC-006` foi executada com quatro slices sequenciais: profissionais (`006/A`),
grade e preview (`006/B`), publicação, materialização e cancelamento mínimo
(`006/C`) e exceções/alterações (`006/D`). `006/A` está `validated`; `006/B-D`
estão `validated locally`. B preserva seu documento executável e C-D foram
derivadas desta guia. A iniciativa termina antes de reserva, ocupação e
presença, que continuam pertencendo às specs seguintes.

`SPEC-004` é uma iniciativa de experiência, não autorização para misturar
dados: contratos pertencem a `contracts`; respostas do PAR-Q pertencem a
`healthforms`, com permissões e retenção próprias.

## Onda 3 — experiência operacional

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| [`SPEC-007` Reservas e lista de espera](specs/SPEC-007-reservations-waitlist.md) — `validated locally` | `002`, `003`, `006` | reserva elegível, cancelamento, reposição, no-show, ocupação e fila FIFO |
| [`SPEC-008` Check-in e presença](specs/SPEC-008-attendance-access.md) — `validated locally` | `003`, `006`, `007` | entrada mensalista e corporativa com decisão explicável, presença e auditoria |
| [`SPEC-010` Expansão da PWA do aluno](specs/SPEC-010-athlete-pwa-expansion.md) — `validated locally` | contratos públicos de `004` a `008` | agenda, plano, contrato, pagamento e avisos no app atual |
| [`SPEC-013` Quiosque e vendas avulsas](specs/SPEC-013-commerce-kiosk.md) — `validated locally` | `002`, ports financeiros de `005` | produtos e registro manual de vendas sem ERP avançado |
| [`SPEC-014` Comunicados e notificações](specs/SPEC-014-announcements-notifications.md) — `validated locally` | `002`, `003`, PWA atual | publicações segmentadas, prioridade, agenda e histórico |

## Onda 4 — substituição e operação assistida

| Spec planejada | Depende principalmente de | Resultado |
|---|---|---|
| [`SPEC-009` Operação do dia](specs/SPEC-009-today-operations.md) — `validated locally` | `004` a `008` | tela Hoje com aulas, ocupação, presenças, cobranças e pessoas que exigem ação |
| `SPEC-011` Migração do sistema anterior | modelos estabilizados | importação, reconciliação, ensaio e corte do Tecnofit |
| `SPEC-012` Wellhub e TotalPass | `006` a `008` + documentação externa | check-in, booking, reconciliação e homologação |

O desenho de `SPEC-011` começa antes da Onda 4 para influenciar campos e
identificadores. A execução final depende de exportações reais e modelos
estabilizados.

`SPEC-012` terá um núcleo neutro e simuladores antes das credenciais. Adapters
reais só serão fechados com documentação e homologação dos parceiros.

`SPEC-009` está `validated locally` sem migration própria. O dashboard compõe
um read model tenant/unidade sobre os contratos implementados de `004` a `008`;
owner vê todas as pendências e coach recebe somente operação de agenda/acesso.

`SPEC-010` está `validated locally`. O PWA compõe agenda diária,
reserva/presença próprias, plano, contrato, cobranças, check-ins e avisos por
vínculo explícito da conta do atleta. Os ajustes de 2026-09-26 acrescentam
inscrição em aula com auditoria/idempotência, PR manual, seleção correta do
treino atual e ferramentas de treino, usando as migrations `075` e `076`.

`SPEC-013` está `validated locally` com a migration
`071_create_commerce.sql`. O módulo `commerce` mantém produtos por unidade e
registra vendas imutáveis com itens, preço congelado, total calculado no
servidor e pagamento manual reconciliado. Não movimenta estoque, não emite
documento fiscal e não chama gateway.

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
