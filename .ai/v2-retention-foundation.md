# EngageFit V2 — fundação de retenção

Status: recorte aprovado para implementação inicial na branch
`v2/retention-foundation`.

Este documento descreve a primeira evolução da V2. Ela não faz parte do
go-live inicial do CrossFit Alados e não deve ser implantada em produção antes
de homologação específica.

## Objetivo

Transformar os check-ins já importados em uma fila diária, explicável e
acionável de alunos que merecem atenção. O primeiro ciclo precisa responder:

1. quem precisa de atenção;
2. quais sinais justificam a classificação;
3. qual ação humana foi realizada;
4. se houve retorno depois da intervenção.

O produto continua sendo uma camada de inteligência e retenção sobre os
sistemas já usados pela academia. Este recorte não tenta substituir CRM,
financeiro, agenda ou controle de acesso.

## Primeiro vertical slice

### Radar de retenção

O radar compara duas janelas consecutivas de 28 dias. Neste primeiro recorte,
as datas são fechadas em UTC, como o dashboard atual. A evolução deve adotar o
timezone operacional da academia quando essa configuração passar a existir no
tenant:

- janela recente: hoje e os 27 dias anteriores;
- janela de referência: os 28 dias imediatamente anteriores.

Para cada aluno não anonimizado, calcular:

- data do primeiro e do último check-in;
- dias desde o último check-in;
- check-ins na janela recente;
- check-ins na janela de referência;
- média semanal recente e anterior;
- percentual de queda, quando existir base anterior suficiente;
- sinais explicáveis que levaram à classificação.

Classificações iniciais:

- `healthy`: frequência estável e sem sinal relevante;
- `attention`: primeiro sinal de queda ou aproximação do limite de inatividade;
- `at_risk`: limite atual de inatividade atingido ou queda relevante;
- `critical`: ausência prolongada ou queda muito acentuada;
- `recovered`: houve intervenção registrada e um check-in posterior dentro da
  janela de acompanhamento.

As regras devem ser determinísticas, testadas e retornadas pela API com códigos
de motivo e textos legíveis. Não haverá modelo preditivo ou LLM nesta etapa.

### Regras iniciais

Usar `boxes.risk_inactive_days` como referência configurável:

- `critical`:
  - dias sem check-in >= `max(14, risk_inactive_days * 2)`; ou
  - queda >= 75%, desde que a janela anterior tenha ao menos 4 check-ins;
- `at_risk`:
  - dias sem check-in >= `risk_inactive_days`; ou
  - queda >= 50%, desde que a janela anterior tenha ao menos 4 check-ins;
- `attention`:
  - dias sem check-in >= `ceil(risk_inactive_days * 0.7)`; ou
  - queda >= 25%, desde que a janela anterior tenha ao menos 4 check-ins;
- `healthy`: nenhum dos sinais anteriores;
- `recovered`: substitui a classificação corrente quando existir uma
  intervenção concluída e ao menos um check-in posterior a ela, ocorrido em até
  14 dias.

Alunos sem check-in suficiente continuam visíveis, mas o motivo deve deixar
claro que ainda não existe histórico para medir queda. O sistema não deve
inventar precisão.

### Central de ações

Uma intervenção registra uma ação humana relacionada a um aluno:

- canal: `whatsapp`, `phone`, `in_person` ou `other`;
- status: `planned`, `completed` ou `cancelled`;
- resultado: `contacted`, `no_response`, `follow_up`, `paused`,
  `not_interested` ou `other`;
- data planejada e data de conclusão;
- observação curta opcional;
- usuário responsável.

O primeiro frontend permitirá:

- filtrar o radar por classificação e buscar aluno;
- operar filas separadas para ações pendentes, acompanhamentos em andamento,
  retornos e casos pausados/encerrados;
- entender os sinais sem abrir outra tela;
- registrar uma intervenção;
- consultar o histórico recente;
- destacar retorno observado depois de uma ação.
- abrir o histórico de frequência com tendência das últimas oito semanas,
  calendário mensal navegável e detalhes dos check-ins de cada dia.

O painel de frequência é reutilizado no Radar e na tela `Check-ins`. Ele
consome `GET /api/v1/students/:id/checkins`; não exige nova tabela nem
migration. As cores distinguem Wellhub e TotalPass e um dia pode exibir mais de
um check-in.

O sinal de frequência e o estado operacional são independentes. Um aluno pode
continuar `critical` enquanto está `waiting_return`: a frequência ainda exige
atenção, mas a equipe já realizou o contato.

Estados operacionais:

- `needs_action`: sinal relevante sem acompanhamento;
- `waiting_return`: contato realizado e data de revisão ainda não atingida;
- `follow_up_due`: não houve retorno e a revisão venceu;
- `paused`: pausa registrada até a data informada;
- `closed`: aluno informou não ter interesse;
- `recovered`: presença observada em até 14 dias após a ação;
- `none`: nenhuma ação necessária.

Sem uma data informada, um contato concluído recebe janela padrão de sete dias.
Retornos ficam destacados por até 30 dias, evitando que o aluno permaneça para
sempre na fila de recuperados.

Não haverá atribuição entre múltiplos funcionários nesta primeira entrega,
porque o produto ainda possui somente o papel operacional `OWNER`.

## Segundo vertical slice — operação e aprendizado

Implementado na continuidade da branch `v2/retention-foundation`:

- `GET /api/v1/retention/summary` apresenta ações concluídas, fila atual,
  retornos observados em 3/7/14 dias, mediana válida e distribuições por motivo,
  canal e resultado;
- intervenções possuem `reason_code` estruturado e recomendação operacional
  determinística no radar;
- a jornada dos primeiros 30 dias usa `membership_started_at`, diferenciando
  data inferida pela primeira presença de data confirmada manualmente;
- entradas recorrentes aceitam os arquivos Wellhub/TotalPass existentes com
  credencial armazenada como hash e batch idempotente;
- o papel `COACH` possui allowlist operacional, sessão revogada ao ser
  desativado e pode receber atribuição de acompanhamentos;
- owner continua exclusivo para cobrança, integrações, importações,
  configurações, campanhas, comunicação, automação e privacidade.

Migrations deste slice: `036` a `039`.

## Contrato técnico proposto

Nova tabela tenant-scoped `retention_interventions`:

- UUID;
- `box_id`, `student_id` e `created_by_user_id`;
- canal, status e resultado;
- `planned_for`, `completed_at`, observação;
- timestamps;
- índices por academia/status/data e aluno/data;
- cascade a partir de academia e aluno;
- usuário responsável preservado de acordo com as regras atuais de usuários.

Endpoints iniciais:

- `GET /api/v1/retention/radar`;
- `GET /api/v1/students/:id/retention-interventions`;
- `POST /api/v1/students/:id/retention-interventions`;
- `PATCH /api/v1/retention-interventions/:id`.

O radar será calculado no PostgreSQL a partir de agregações de check-ins, sem
persistir um score opaco. Persistência de snapshots só será adicionada quando
houver necessidade comprovada de histórico ou escala.

## Segurança, tenancy e privacidade

- toda consulta e mutação deve filtrar por `box_id`;
- aluno de outra academia deve responder como não encontrado;
- aluno anonimizado não aparece no radar;
- opt-out não remove o aluno do radar, pois a equipe ainda pode realizar uma
  abordagem presencial, mas a UI deve indicar que contato eletrônico não está
  autorizado;
- observações não devem receber dados de saúde, diagnóstico ou informação
  financeira detalhada;
- exportação e anonimização do aluno devem incluir ou sanitizar as
  intervenções;
- logs não devem conter nome, telefone, observação ou texto de motivo livre.

## Medição inicial

Para cada intervenção concluída, apresentar:

- retorno em até 3, 7 e 14 dias;
- data do primeiro check-in posterior;
- tempo até o retorno;
- classificação atual.

Isso representa associação temporal, não causalidade. A interface não deve
afirmar que a intervenção causou o retorno.

## Fora do primeiro slice

- score preditivo ou uso de IA;
- integração nativa com APIs de check-ins;
- WhatsApp em duas vias;
- dados financeiros, renovação e cancelamento;
- testes A/B e grupos de controle;
- automações de envio baseadas no novo radar.

A porta recorrente implementada não é uma integração nativa com as APIs da
Wellhub ou TotalPass. Conectores externos ainda precisam obter o arquivo na
origem e enviá-lo ao EngageFit.

Esses itens continuam no roadmap, mas dependem de histórico e aprendizado com
os primeiros clientes.

## Critérios de aceite

1. A mesma base e data de referência sempre produzem a mesma classificação.
2. Toda classificação diferente de `healthy` possui ao menos um motivo
   explicável.
3. Consultas e mutações são isoladas entre duas academias em teste PostgreSQL.
4. Uma intervenção concluída pode ser associada a um retorno em 3/7/14 dias.
5. Aluno anonimizado não aparece; opt-out aparece com restrição de canal.
6. A UI funciona em desktop e mobile sem tabela horizontal obrigatória.
7. Migration nova passa em banco vazio, segunda execução aplica zero e o fluxo
   existente continua verde.
