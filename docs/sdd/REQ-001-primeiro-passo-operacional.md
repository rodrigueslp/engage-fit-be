# REQ-001 — Primeiro passo operacional para substituir o Tecnofit

- Status: `reviewed`
- Versão: `1.1`
- Data: 2026-08-24
- Origem: esboço de produto entregue pelo sócio no documento
  `Primeiro_Passo.pdf`
- Natureza: entrada de produto; não autoriza implementação isoladamente

## 1. Objetivo

Registrar de forma rastreável o primeiro recorte operacional solicitado pelo
Alados para que o EngageFit possa substituir o núcleo essencial do Tecnofit.
Este documento traduz o esboço de negócio em capacidades e restrições que serão
detalhadas nas specs executáveis.

O objetivo não é copiar telas ou o modelo interno do Tecnofit. O EngageFit deve
fechar as mesmas rotinas essenciais com menos campos, boa operação no celular e
estados separados para cadastro, matrícula, contrato, pagamento e acesso.

## 2. Capacidades solicitadas

| ID | Capacidade | Necessidade operacional |
|---|---|---|
| `REQ-CLIENT-01` | Cadastro de cliente | nome, telefone, CPF, e-mail e endereço, com entrada inicial simples |
| `REQ-DOC-01` | Contrato e PAR-Q | gerar/enviar depois do cadastro ou venda, por WhatsApp/e-mail |
| `REQ-DOC-02` | Assinatura eletrônica | assinatura auditável por Gov.br ou provedor equivalente |
| `REQ-SCHEDULE-01` | Modalidades e serviços | CrossFit, Spinning, Pilates, Recovery, corrida e atendimentos especializados |
| `REQ-SCHEDULE-02` | Grade e vagas | horários, profissional responsável, espaço e capacidade |
| `REQ-BOOKING-01` | Reserva e presença | aluno reserva/faz check-in conforme vaga e direito do plano |
| `REQ-PLAN-01` | Planos recorrentes | mensal, trimestral, semestral, renovação e grupos comerciais |
| `REQ-PLAN-02` | Pacotes e frequência | pacote de aulas, uma/duas aulas por semana, individual ou grupo |
| `REQ-PLAN-03` | Adicionais | plano base com serviços/modalidades opcionais somados ao preço |
| `REQ-PAY-01` | Cobrança online | cartão e recorrência por gateway seguro |
| `REQ-PAY-02` | Pagamento manual | baixa por maquininha, Pix ou dinheiro, com auditoria |
| `REQ-PAY-03` | Conciliação | relacionar cliente, matrícula/contrato, cobrança e situação financeira |
| `REQ-POS-01` | Quiosque | cadastrar produtos e registrar vendas manuais avulsas |
| `REQ-RET-01` | Retenção | identificar regularidade, atenção e risco por frequência configurável |
| `REQ-COMMS-01` | Comunicados | publicar notícias e avisos para o aplicativo do aluno |
| `REQ-COMMS-02` | Segmentação | unidade, modalidade, plano/grupo, prioridade, agenda e histórico |

## 3. Decisões de produto para detalhamento

### 3.1 Cadastro progressivo

O fluxo padrão continua começando com nome, WhatsApp e CPF. E-mail e endereço
ficam em `Mais informações` e só se tornam obrigatórios quando houver motivo
jurídico, financeiro, fiscal ou operacional identificado pela spec consumidora.
Isso atende o dado solicitado sem transformar o primeiro cadastro em uma ficha
extensa.

Cadastro, matrícula, contrato, situação financeira e acesso são estados
independentes. A interface pode apresentar uma jornada única, mas o banco não
deve colapsá-los em um único status de aluno.

### 3.2 Contrato e PAR-Q não são o mesmo dado

Contrato é documento jurídico vinculado aos termos de uma matrícula. PAR-Q é
questionário de saúde e pode conter dado pessoal sensível. Mesmo que ambos
sejam enviados juntos na experiência, terão armazenamento, permissões,
auditoria e retenção próprios.

A assinatura deve ficar atrás de um port substituível. Integração com Gov.br
ou provedor equivalente depende de contratação, documentação e homologação
externa; o núcleo local deve funcionar com simulador antes das credenciais.

### 3.3 Modalidade, serviço, espaço e sessão são conceitos distintos

- modalidade: atividade reutilizável, como CrossFit, Spinning ou Pilates;
- serviço: oferta agendável, como Recovery, avaliação, fisioterapia ou
  massagem;
- espaço: local físico, como área principal ou sala coringa;
- sessão: ocorrência em data/horário, com profissional e capacidade;
- profissional: responsável pela sessão, com ou sem conta de acesso.

Essa separação evita colocar preço, agenda e profissional no cadastro simples
da modalidade.

### 3.4 Planos e adicionais são versionados

Plano é oferta versionada. A matrícula preserva a versão e os adicionais
escolhidos no momento da venda. Alterar preço, frequência ou composição não
muda silenciosamente contratos existentes.

`Recorrente` descreve renovação/cobrança, não uma modalidade. Pacote de aulas,
sessões por semana e acesso livre continuam sendo direitos diferentes.

### 3.5 Financeiro do aluno é separado do billing SaaS

O módulo atual `billing` cobra o EngageFit das academias. Cobranças de alunos
pertencem a `memberbilling`, mesmo que ambos usem Asaas. Podem ser reutilizados
padrões técnicos de gateway, idempotência, webhook e conciliação; entidades,
tabelas, rotas e permissões permanecem distintas.

Pagamentos manuais são eventos financeiros de primeira classe, com valor,
forma, data, responsável e motivo. Eles não podem ser representados apenas por
uma alteração livre de status.

### 3.6 Quiosque começa sem ERP

O primeiro recorte registra produto, venda, itens, total e pagamento. Estoque
avançado, emissão fiscal, fornecedores, compras e comissão ficam fora até
validação operacional.

### 3.7 Retenção existente será evoluída, não substituída

O radar atual já compara frequência, classifica severidade e organiza ações
humanas. Planos e agenda fornecerão frequência esperada e situação de matrícula
como novos sinais; a lógica atual e seu histórico serão preservados durante a
migração de `Student` para `Person`.

### 3.8 Comunicado não é campanha de WhatsApp

Comunicados possuem conteúdo publicado, período de exibição, prioridade,
audiência e histórico no aplicativo. WhatsApp, e-mail e push podem distribuir
ou avisar sobre a publicação, mas não são sua fonte de verdade.

## 4. Rastreabilidade para o SDD

| Requisito | Destino | Estado em 2026-08-24 |
|---|---|---|
| Cadastro | `SPEC-002/A` | fundação e cadastro manual implementados e validados |
| Modalidades e espaços | `SPEC-001/A-B` | implementadas e validadas localmente |
| Serviços ofertáveis | `SPEC-001/D` | implementada e validada; antecede planos completos |
| Planos, pacotes e adicionais | `SPEC-003/A-D` | catálogo `003/A`, matrícula `003/C` e adicionais `003/B` validados; acesso `003/D` adiado |
| Fundação de contrato e emissão | `SPEC-004/A` | spec `ready` para implementação local, sem uso produtivo |
| Aceite eletrônico produtivo | `SPEC-004/B` | planejada; depende de provedor e revisão jurídica |
| PAR-Q e formulários sensíveis | `SPEC-004/C` (`healthforms`) | planejada e separada de contratos |
| Financeiro do aluno | `SPEC-005` | planejada |
| Agenda, profissionais e vagas | `SPEC-006` | planejada |
| Reservas | `SPEC-007` | planejada |
| Check-in e decisão de acesso | `SPEC-008` | planejada; existe fluxo legado compatível |
| Retenção | módulo atual + `SPEC-009` | base implementada; integração futura |
| Quiosque | `SPEC-013` | novo escopo planejado |
| Comunicados e notificações | `SPEC-014` | novo escopo planejado |

## 5. Validações futuras com o Alados

- em qual momento e-mail e endereço passam a ser obrigatórios;
- se contrato e PAR-Q são enviados ao cadastrar ou somente ao vender plano;
- provedor aceitável de assinatura e necessidade real de Gov.br;
- quem pode visualizar respostas do PAR-Q e por quanto tempo;
- diferença operacional entre modalidade e serviço na linguagem do time;
- se Pilates individual usa vaga, agenda exclusiva ou ambos;
- regra comercial final de Recovery;
- quais adicionais podem ser combinados e se possuem desconto;
- tratamento de plano família e plano vizinhos;
- gateway financeiro e modelo de repasse/conta da academia;
- necessidade de estoque no quiosque e emissão fiscal;
- quais comunicados exigem push e quais apenas aparecem no aplicativo.

Nenhuma dessas validações exige descartar `Organization`, modalidades, espaços,
retenção, comunicação ou check-ins já implementados.

O fatiamento da `SPEC-004` preserva a separação desta entrada: a slice A trata
template, versão, instância, snapshot da matrícula e simulador local; a B
tratará aceite eletrônico produtivo; a C pertencerá a `healthforms`; e a D
ficará reservada ao ciclo jurídico posterior, coordenado com o financeiro do
aluno. O estado `ready` da slice A autoriza somente sua fundação local; não
autoriza aceite eletrônico produtivo, PAR-Q nem ciclo jurídico posterior.
