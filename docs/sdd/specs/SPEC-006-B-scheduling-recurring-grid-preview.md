# SPEC-006/B — Grade recorrente em rascunho e preview da agenda

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-09-15
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-006`, `SPEC-006/A`
- Bloqueia: `SPEC-006/C`; informa contratos futuros de `SPEC-007` e
  `SPEC-008`, sem autorizá-las
- Ownership principal: módulo `scheduling`
- Colaborador obrigatório por contrato público: `catalog`
- Colaboradores sem chamada em runtime nesta slice: `memberships`,
  `reservations` e `attendance`
- Repositórios futuros envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration: número não reservado; será definido pelo integrador

Implementada localmente em 2026-09-15 como parte da execução integral da
SPEC-006. As decisões da seção 18 foram adotadas com os defaults conservadores
descritos neste documento. A implementação não realizou push, deploy,
alteração de produção nem integração externa.

## 1. Resultado esperado

O owner configura, pelo celular, horários semanais recorrentes da unidade. Cada
horário escolhe exatamente uma modalidade ou um serviço ofertável, um
profissional ativo, um modo `group|individual`, duração, capacidade, dias da
semana, hora local, timezone, vigência e, opcionalmente, um espaço.

O owner salva os itens como rascunho e visualiza ocorrências virtuais em um
intervalo explícito. O preview usa a mesma expansão e a mesma regra de
sobreposição para apontar conflitos de profissional e espaço. Ele deixa sempre
visível que nenhuma sessão foi publicada, nenhuma vaga existe e nenhum aluno
pode reservar ou fazer check-in a partir dessa prévia.

O ciclo vertical termina em configuração revisável:

```text
catálogo + profissional ativo
    -> série semanal em rascunho
    -> expansão virtual determinística
    -> conflitos e referências indisponíveis visíveis
    -> grade pronta para revisão, ainda não publicada
```

Não existem `ScheduleSession`, materialização, eventos públicos de sessão ou
agenda operacional nesta slice.

## 2. Problema, proveniência e decisões já herdadas

### 2.1 Problema operacional

- **Existente:** `catalog` mantém modalidades e serviços por organização,
  ativações por unidade e espaços por unidade. Duração, profissional, horário
  e capacidade não pertencem a esses cadastros.
- **Existente:** `SPEC-006/A` implementou profissionais próprios de
  `scheduling`, com vínculo ativo à unidade, conta opcional, tenancy,
  idempotência, revisão, auditoria e privacy.
- **Existente:** `memberships` descreve direitos sobre modalidade ou serviço,
  inclusive `delivery_mode` e `booking_required`, mas não cria calendário,
  sessão, reserva ou consumo.
- **Entrada Alados — `REQ-SCHEDULE-01/02`:** a operação precisa organizar
  horários, responsável, espaço e capacidade para aulas e atendimentos.
- **Risco:** tratar o preview como sessão publicada faria `reservations` e
  `attendance` consumirem uma intenção editável; tratar capacidade como vaga
  misturaria agenda e ocupação antes da `SPEC-007`.

### 2.2 Decisões herdadas da guia `SPEC-006`

- recorrência inicial semanal, sem RRULE livre;
- um horário local e um ou mais dias da semana por série;
- exatamente um alvo `modality|service_offering`;
- profissional obrigatório e espaço opcional;
- capacidade explícita, sendo a capacidade do espaço apenas sugestão;
- intervalo temporal semiaberto `[starts_at, ends_at)`;
- sobreposição de profissional e de espaço informado é conflito, sem override;
- timezone IANA e vigência por datas locais inclusivas;
- preview e futura publicação devem compartilhar a mesma função pura de
  expansão e conflito;
- grade não decide reserva, presença, cobrança, consumo ou acesso.

### 2.3 Propostas específicas desta slice

- rascunhos conflitantes podem ser salvos, porque ainda não produzem efeito
  operacional; o preview os marca como bloqueados para publicação futura;
- o preview da grade compara séries em rascunho não arquivadas; conflito com
  sessões publicadas será acrescentado pela `SPEC-006/C`, antes de qualquer
  publicação existir;
- timezone fica explícito e persistido em cada série, pois hoje não existe um
  timezone canônico na entidade de unidade;
- uma série em rascunho pode ser arquivada e reativada; não existe exclusão
  física nem status que simule publicação;
- nomes de catálogo, espaço e profissional não são copiados para a série em
  rascunho. A leitura resolve os rótulos e o estado atuais por ports; snapshots
  históricos pertencem à publicação da slice C.

Essas propostas são bloqueadores de promoção para `ready` e estão consolidadas
na seção 18.

## 3. Escopo

### 3.1 Incluído

- agregado `ScheduleSeries` limitado aos estados `draft|archived`;
- recorrência semanal com `interval_weeks=1`;
- um ou mais dias da semana, hora local sem segundos, timezone IANA e vigência;
- alvo tipado `modality|service_offering` resolvido pelo catálogo;
- profissional obrigatório da mesma organização, ativo na unidade;
- espaço opcional da unidade;
- modo `group|individual`, duração e capacidade explícitas;
- criação, edição, arquivamento e reativação de rascunho pelo owner;
- listagem/detalhe com saúde atual das referências;
- preview de uma proposta ainda não salva e preview da grade persistida;
- expansão virtual determinística, limites temporais e limite de volume;
- detecção de sobreposição de profissional e espaço entre rascunhos;
- idempotência dos comandos, revisão otimista e auditoria transacional;
- tenancy por organização/unidade e RBAC exclusivo do owner para grade;
- exportação de atribuições do profissional e retenção proposta;
- UX mobile-first dentro de `#/schedule`;
- capability própria para rollout independente da fundação de profissionais;
- matriz de testes de domínio, PostgreSQL, HTTP, frontend, concorrência,
  privacidade e regressão.

### 3.2 Explicitamente fora do escopo

- publicar série ou transformar preview em dado operacional;
- criar ou persistir sessão, occurrence row, agenda do dia ou calendário do
  coach/aluno;
- materializador, worker, run, reconciliação ou evento público de sessão;
- sessão extra, alteração de sessão, exceção, feriado, recesso, cancelamento
  ou reposição;
- reserva, cancelamento de reserva, lista de espera, no-show, ocupação ou
  `available_capacity`;
- presença, check-in, catraca, validação ou decisão de acesso;
- consulta de pessoa, matrícula, plano contratado, saldo ou direito para
  montar a grade;
- consumo, estorno ou contador de aulas;
- contrato, PAR-Q, cobrança, pagamento, comissão ou folha;
- notificação, WhatsApp, e-mail, push ou integração externa;
- preço, desconto ou regra comercial no rascunho;
- disponibilidade trabalhista, férias ou múltiplos responsáveis;
- RRULE, recorrência quinzenal/mensal, regra automática de feriado ou
  calendário externo;
- múltiplos espaços na mesma série;
- edição multiunidade ou seleção de unidade da `SPEC-001/C`;
- alteração de matrícula, check-in, `Student`, `Person`, conta do atleta ou
  dados legados.

## 4. Vocabulário e fronteira semântica

- **Grade:** conjunto de séries em rascunho não arquivadas da unidade.
- **Série recorrente (`ScheduleSeries`):** configuração semanal persistida,
  ainda não publicada.
- **Ocorrência virtual (`SchedulePreviewOccurrence`):** valor calculado em
  memória para uma data local; não possui ID de sessão nem é persistido.
- **Preview:** projeção somente leitura de ocorrências virtuais e problemas da
  grade em uma janela explícita.
- **Vigência:** intervalo de datas locais em que a regra semanal se aplica;
  `effective_until`, quando informado, é inclusivo.
- **Capacidade:** limite configurado que uma futura sessão poderá herdar. Não
  significa vaga total disponível, reserva ou presença.
- **Conflito:** interseção positiva de intervalos do mesmo profissional ou do
  mesmo espaço informado.
- **Referência indisponível:** alvo, espaço ou profissional que continua
  referenciado pelo ID, mas não está ativo/habilitado para novo uso.
- **Arquivado:** rascunho fora da grade ativa, preservado para histórico e
  possível reativação.

Na UI, `Grade` nunca recebe o título isolado `Agenda`. O preview usa o rótulo
`Prévia da grade — não publicada`. `Aula` representa modalidade,
`Atendimento` representa serviço e `Sessão` fica reservado à slice C.

## 5. Ownership e dependências de módulo

| Conceito | Owner | Contrato nesta slice |
|---|---|---|
| organização, unidade, usuário e papel | `organizations/access` | contexto autorizado já existente; nenhuma escrita em suas tabelas |
| modalidade, serviço, ativação e espaço | `catalog` | resolver estreito de referências atuais |
| profissional e vínculo com unidade | `scheduling` | leitura interna da fundação `SPEC-006/A` |
| série, recorrência, preview e conflito de rascunho | `scheduling` | domínio, tabelas, API e auditoria próprios |
| planos, matrículas e direitos | `memberships` | nenhuma chamada; apenas IDs de alvo semanticamente compatíveis |
| reserva, ocupação e lista de espera | `reservations` | nenhum contrato; preview não é consumível |
| presença e decisão de entrada | `attendance` | nenhum contrato; preview não autoriza acesso |
| experiência do aluno | `athlete` | nenhum contrato; rascunho não é exposto |

Regras de fronteira:

1. `scheduling` não importa repositories, models GORM, DTOs HTTP ou tabelas de
   `catalog`, `memberships`, `reservations`, `attendance` ou `athlete`.
2. O repository de série acessa apenas tabelas de `scheduling` e FKs internas
   do módulo. Referências externas são IDs opacos validados por ports.
3. Nenhum port de sessão publicada é criado nesta slice. Em especial, não
   existem `GetPublishedSession`, `ListPublishedSessions` ou
   `ListSessionEvents`.
4. `memberships`, `reservations` e `attendance` não recebem DTO de rascunho ou
   ocorrência virtual. A ausência desses contratos é intencional.
5. A slice C deverá consumir o mesmo motor puro de expansão, revalidar todas
   as referências e acrescentar sessões publicadas ao conjunto de conflitos.

## 6. Regras e invariantes

### 6.1 Tenancy e RBAC

1. Toda série possui exatamente um `organization_id` e um `unit_id`.
2. O ator, a organização e a unidade são derivados da sessão e da rota
   validada, nunca do body.
3. Nesta fase, `unit_id` da rota precisa ser igual a `users.box_id`. UUID de
   outra unidade/organização responde como `404`.
4. Somente `OWNER` cria, edita, arquiva, reativa, lista e pré-visualiza grade.
5. `COACH`, inclusive quando vinculado ao profissional da série, não lê
   rascunhos nem chama preview. Sua leitura dos profissionais ativos da slice A
   permanece inalterada.
6. `PLATFORM_ADMIN` não acessa a superfície tenant.
7. Unidade precisa estar ativa e autorizada para qualquer comando ou preview.
   Inativar/suspender a unidade não apaga rascunhos; enquanto ela estiver
   indisponível, a superfície não é acessível.

### 6.2 Alvo, profissional e espaço

1. Cada série referencia exatamente um `target_kind` entre `modality` e
   `service_offering`, acompanhado de um único `target_id`.
2. O alvo precisa pertencer à organização, estar `active` e habilitado na
   unidade ao criar, atualizar ou reativar o rascunho.
3. `professional_id` é obrigatório, pertence à organização e precisa possuir
   vínculo ativo com a unidade.
4. `space_id` é opcional. Quando informado, precisa existir, estar ativo e
   pertencer à mesma unidade.
5. Capacidade padrão do espaço pode preencher o formulário, mas a série sempre
   persiste sua própria `capacity`. Alterar o padrão depois não muda o
   rascunho.
6. `target_id`, `space_id` e `professional_id` não são intercambiáveis e não
   aceitam nomes livres como referência.
7. Renomear alvo, espaço ou profissional altera apenas o rótulo resolvido na
   próxima leitura. Não existe snapshot de nome na série B.

### 6.3 Recorrência, timezone e vigência

1. `interval_weeks` é fixo em `1` e não precisa ser enviado pelo cliente.
2. `weekdays` contém de 1 a 7 valores únicos no mapa `0=domingo`,
   `1=segunda`, ..., `6=sábado`. A API ordena o conjunto antes do fingerprint.
3. `local_start_time` usa `HH:MM`, sem segundos.
4. `timezone` é obrigatório, usa nome IANA carregável pelo runtime e é
   persistido na série. Abreviações como `BRT`, offsets fixos como `-03:00` e
   timezone desconhecido são rejeitados.
5. A UI sugere `America/Sao_Paulo`, mas exibe o valor em abertura progressiva;
   o backend não o injeta silenciosamente quando estiver ausente.
6. `effective_from` é obrigatório. `effective_until` é opcional, inclusivo e,
   quando presente, não pode anteceder `effective_from`.
7. A recorrência produz no máximo uma ocorrência virtual por série/data local.
   Outro horário exige outra série.
8. Duração aceita inteiro de 5 a 1.440 minutos. O fim é calculado como instante
   inicial mais duração exata; uma ocorrência não pode avançar além do dia
   civil imediatamente seguinte no timezone da série.
9. A conversão de data + hora local para instante precisa encontrar exatamente
   um instante no timezone. Horário inexistente ou ambíguo por transição de
   fuso gera problema bloqueante `invalid_local_time`; nunca é deslocado ou
   escolhido silenciosamente.

### 6.4 Capacidade e modo

1. `delivery_mode` aceita somente `group|individual`.
2. `individual` exige `capacity=1`.
3. `group` exige capacidade inteira entre 1 e 10.000.
4. A resposta usa somente `capacity`. São proibidos `available_capacity`,
   `remaining_spots`, `reserved_count`, `waitlist_count` e equivalentes.
5. `booking_required` não é campo da série; continua termo de direito de
   `memberships`.

### 6.5 Expansão determinística

Para uma série e uma janela local inclusiva `[preview_from, preview_until]`:

1. validar estrutura e timezone;
2. calcular `from=max(preview_from, effective_from)`;
3. calcular `until=min(preview_until, effective_until)` quando houver fim;
4. percorrer datas em ordem crescente no calendário do timezone;
5. incluir somente datas cujo weekday pertença ao conjunto da série;
6. converter data/hora local em um único `starts_at`;
7. calcular `ends_at=starts_at+duration_minutes`;
8. criar occurrence key determinística:
   `series:<series_id>:<local_date>` para série salva ou
   `candidate:<payload_fingerprint>:<local_date>` para proposta não salva;
9. ordenar por `starts_at`, `ends_at`, `series_id/occurrence_key`;
10. calcular conflitos sem alterar a lista de ocorrências.

Mesmos dados persistidos, janela, versão de timezone IANA disponível no
runtime e referências resolvidas produzem a mesma resposta. O preview retorna
`preview_from`, `preview_until`, timezone, `series_revision` quando aplicável e
`generated_at`, deixando explícito o contexto do cálculo.

### 6.6 Limites de preview

1. A janela é informada por datas locais inclusivas e aceita de 1 a 56 dias.
2. A UI abre a grade persistida em 14 dias e permite ampliar até 56; o backend
   não mantém um default implícito.
3. Uma resposta aceita no máximo 2.000 ocorrências virtuais e 200 conflitos.
4. O limite é validado antes de retornar o corpo. Não existe resposta parcial
   ou truncada: excesso retorna `scheduling_preview_too_large`.
5. Um preview de proposta compara a proposta com todos os rascunhos ativos
   relevantes no mesmo intervalo, excluindo o próprio `series_id` em edição.
6. O preview da grade expande todos os rascunhos ativos da unidade e consulta
   também conflitos de profissional em outras unidades da mesma organização,
   quando essas séries existirem no futuro.
7. O endpoint limita body e campos pelos limites HTTP já existentes; não
   aceita lote arbitrário de séries enviado pelo cliente.

### 6.7 Conflitos e sobreposição

1. Intervalos são semiabertos: há conflito quando
   `candidate.starts_at < existing.ends_at` e
   `candidate.ends_at > existing.starts_at`.
2. Intervalos adjacentes, em que um termina exatamente no começo do outro,
   são permitidos.
3. O mesmo profissional não pode ocupar intervalos sobrepostos na organização,
   inclusive entre unidades.
4. O mesmo espaço informado não pode ocupar intervalos sobrepostos na unidade.
5. Série sem `space_id` não participa de conflito espacial e não usa valor
   sentinela.
6. Séries arquivadas são ignoradas como fonte de conflito.
7. Alvo, capacidade ou modo iguais/diferentes não alteram a regra de conflito.
8. A expansão identifica conflitos dentro do conjunto persistido e entre uma
   proposta e o conjunto persistido.
9. Salvar rascunho conflitante é permitido nesta proposta. A resposta do
   preview usa `ready_for_future_publication=false` e problemas bloqueantes;
   não existe comando de publicação nesta slice.
10. Não há override, confirmação para ignorar ou prioridade entre rascunhos.
11. A slice C deverá reexecutar o algoritmo sob locks e incluir sessões já
    publicadas; nenhum resultado da B será autorização para publicar.

### 6.8 Revisão, idempotência e transação

1. `revision` começa em 1 e incrementa uma vez por mutação efetiva.
2. Atualizar, arquivar e reativar exigem `expected_revision`.
3. Todos os comandos exigem `Idempotency-Key` de 8–128 caracteres conforme a
   política da slice A.
4. O fingerprint canônico inclui organização, unidade, comando, ID/revisão
   quando aplicável, target tipado, espaço nullable, profissional, modo,
   capacidade, duração, weekdays ordenados, hora, timezone, vigência e motivo
   normalizado quando houver.
5. Replay com mesmo fingerprint retorna ID/revisão/resposta originais sem nova
   auditoria. Reutilização com payload diferente retorna
   `idempotency_key_reused`.
6. Mutação, weekdays, command request e auditoria são confirmados na mesma
   transação.
7. Preview é somente leitura, não usa idempotency key, não incrementa revisão e
   não persiste ocorrência, conflito ou resultado.
8. Rascunhos concorrentes podem ambos ser salvos; o preview posterior precisa
   expor o conflito. Concorrência sobre o mesmo rascunho é resolvida por lock e
   revisão otimista.

## 7. Ciclo de vida e referências inativadas

### 7.1 Série

- nova série nasce `draft`;
- arquivar exige motivo de 3–240 caracteres e a retira da grade ativa;
- reativar revalida todas as referências e a revisão;
- reativação não é permitida enquanto alvo, profissional ou espaço informado
  estiver indisponível;
- não existe `DELETE`, publicação, cancelamento ou retorno de `archived` para
  um estado diferente de `draft` nesta slice;
- arquivar série não arquiva profissional, catálogo ou espaço.

### 7.2 Profissional

- arquivar profissional na slice A continua permitido mesmo quando ele está em
  rascunhos; não há sessão publicada a proteger;
- rascunhos ativos permanecem armazenados e passam a apresentar
  `professional_unavailable` no detalhe/preview;
- reativar o profissional e seu vínculo torna o rascunho elegível novamente,
  desde que as demais referências estejam válidas;
- anonimização do profissional é bloqueada enquanto existir série `draft`
  ativa atribuída a ele; o owner precisa reatribuir ou arquivar a série;
- não existe nome do profissional copiado na série, auditoria ou command
  request.

### 7.3 Modalidade e serviço

- arquivar o item ou desativá-lo na unidade não altera nem apaga rascunhos;
- detalhe e preview retornam `target_unavailable` e o rótulo atual resolvido;
- reativação no catálogo restaura a validade sem regravar a série;
- renomear muda o rótulo resolvido, não a identidade `target_id`;
- desligar a capability `service_catalog` oculta novos serviços da seleção e
  marca rascunho de serviço como indisponível para preview de prontidão, sem
  apagá-lo.

### 7.4 Espaço

- arquivar o espaço mantém a referência e produz `space_unavailable`;
- remover o espaço do rascunho é uma edição válida e elimina conflito espacial
  futuro;
- reativação restaura a referência; alteração de `default_capacity` nunca
  reescreve `series.capacity`;
- renomear altera somente o rótulo atual da leitura.

### 7.5 Unidade

- suspender, arquivar ou bloquear operacionalmente a unidade impede a
  resolução de contexto e qualquer rota da grade;
- séries e auditoria permanecem armazenadas, sem cascade de status;
- reativar a unidade restaura o acesso conforme sessão/RBAC; cada preview
  revalida também catálogo e profissionais;
- a slice não cria rotina administrativa para transferir série a outra unidade.

## 8. Fluxos do usuário e UX mobile-first

### 8.1 Navegação

- manter hash `#/schedule` e item `Agenda` já entregues pela slice A;
- owner passa a ver as seções `Grade` e `Profissionais`;
- coach continua vendo somente `Profissionais` nesta slice, sem tab vazia de
  agenda ou grade;
- `Grade` só aparece quando `scheduling_grid=true`;
- cabeçalho mostra unidade atual e badge permanente `Somente rascunho`.

### 8.2 Criar horário recorrente

1. Owner toca em `Adicionar horário`.
2. Escolhe `Aula` (modalidade) ou `Atendimento` (serviço) e o alvo ativo.
3. Seleciona profissional e espaço opcional.
4. Define `Em grupo` ou `Individual`, capacidade e duração.
5. Seleciona dias, hora local, início e fim opcional.
6. `Fuso horário` aparece em `Opções de calendário`, preenchido visualmente com
   `America/Sao_Paulo`, mas precisa fazer parte do payload confirmado.
7. Toca em `Revisar prévia`; a tela mostra ocorrências, avisos e conflitos.
8. Toca em `Salvar rascunho`. A confirmação repete:
   `Nenhuma sessão foi publicada e alunos ainda não podem reservar`.

O formulário é recuperável após conflito, revisão antiga ou falha de rede.
Conflito não apaga dados nem impede salvar rascunho nesta proposta.

### 8.3 Preview da grade

- abre agrupado por data local, com 14 dias explícitos;
- oferece ampliar até 56 dias sem paginação temporal escondida;
- cada ocorrência virtual mostra hora, duração, alvo, modo, capacidade total,
  profissional e espaço quando houver;
- não mostra preço, plano, matrícula, aluno, vaga restante ou botão de reserva;
- conflito informa recurso e os dois intervalos, com ação `Editar horário`;
- referência inativa informa exatamente o cadastro a corrigir;
- banner e empty state usam `Prévia — nenhuma sessão publicada`;
- nenhuma ocorrência virtual recebe link ou ID com aparência de sessão real.

### 8.4 Editar, arquivar e reativar

- editar abre os valores persistidos e exclui a própria série do comparativo;
- revisão antiga preserva o formulário e oferece recarregar a versão atual;
- arquivar pede motivo e explica que o rascunho sai da grade, sem afetar
  cadastros;
- `Mostrar arquivados` fica desligado por padrão;
- reativar mostra validação das referências antes da confirmação;
- não existe ação `Publicar`, `Cancelar aula`, `Criar sessão` ou `Abrir vagas`.

### 8.5 Responsividade e acessibilidade

- fluxo principal em uma coluna, validado em `360x800`, sem overflow
  horizontal;
- alvos de toque de no mínimo 44 px;
- labels e descrição de erro associados aos campos;
- seletores de data/hora com teclado e controles adequados;
- weekdays operáveis por teclado e leitor de tela;
- foco previsível, Escape e rolagem interna em drawer/modal;
- estado/conflito não depende apenas de cor;
- loading, vazio, erro, retry, sucesso, conflito, referência inativa e flag
  desligada possuem estados próprios;
- a UI não mostra UUID, `box_id`, tenant, RRULE ou termos de banco.

## 9. Modelo de domínio e persistência proposta

Os nomes abaixo são proposta desta versão `draft`. Não fixam migration nem
autorizam schema.

### 9.1 Agregado `ScheduleSeries`

```text
ScheduleSeries
  ├── scope: organization_id + unit_id
  ├── target: kind + opaque ID
  ├── resources: professional_id + optional space_id
  ├── delivery: mode + capacity + duration
  ├── recurrence: weekdays + local time + timezone + validity
  ├── lifecycle: draft | archived
  └── revision + actors + timestamps

SchedulePreviewOccurrence (não persistida)
  └── occurrence_key + local/instant interval + resolved labels + issues
```

### 9.2 `scheduling_series`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id`, `unit_id` | UUIDs obrigatórios |
| `status` | `draft|archived` |
| `target_kind` | `modality|service_offering` |
| `target_id` | UUID opaco obrigatório |
| `professional_id` | UUID obrigatório, FK interna composta |
| `space_id` | UUID opaco nullable |
| `delivery_mode` | `group|individual` |
| `capacity` | 1–10.000; individual exige 1 |
| `duration_minutes` | 5–1.440 |
| `local_start_time` | `TIME` sem segundos |
| `timezone` | varchar(64), IANA validado na aplicação |
| `effective_from` | `DATE` obrigatório |
| `effective_until` | `DATE` nullable e inclusivo |
| `revision` | inteiro positivo |
| `created_by_user_id`, `updated_by_user_id` | atores opacos obrigatórios |
| `archived_at`, `archived_by_user_id` | preenchidos em conjunto no status arquivado |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints/índices mínimos:

- unique `(id, organization_id, unit_id)`;
- FK composta `(professional_id, organization_id)` para
  `scheduling_professionals` e vínculo da unidade validado pela aplicação;
- FK composta `(unit_id, organization_id)` para o contexto de unidade já
  usado pela slice A;
- checks de status, alvo, modo/capacidade, duração, minuto exato, timezone não
  vazio, vigência, revisão e campos de arquivamento;
- índice `(organization_id, unit_id, status, effective_from, id)`;
- índice `(organization_id, professional_id, status)` para conflitos e
  privacy;
- índice `(organization_id, unit_id, space_id, status)` parcial quando espaço
  não for nulo;
- nenhuma FK ou trigger para tabelas de catálogo, memberships, reservations ou
  attendance.

### 9.3 `scheduling_series_weekdays`

| Campo | Regra |
|---|---|
| `organization_id`, `unit_id`, `series_id` | UUIDs obrigatórios |
| `weekday` | smallint entre 0 e 6 |

Constraints:

- PK `(series_id, weekday)`;
- FK composta para série na mesma organização/unidade com `ON DELETE RESTRICT`;
- uma série nunca fica sem weekday após commit.

### 9.4 Evolução de `scheduling_command_requests`

A migration `061` atualmente especializa o registro em `professional_id`. A
evolução proposta preserva todas as linhas e replays da slice A:

- adicionar `series_id` nullable com FK composta para a nova série;
- tornar `professional_id` nullable, preservando sua FK;
- ampliar `command` com `series_create`, `series_update`, `series_archive` e
  `series_reactivate`;
- check exige exatamente `professional_id` para comandos de profissional ou
  exatamente `series_id` para comandos de série;
- manter unique `(organization_id, command, idempotency_key)` e política de
  chave/fingerprint existentes;
- nunca editar a migration `061` já integrada localmente.

### 9.5 Evolução de `scheduling_audit_events`

O evento atual possui `entity_type/entity_id`, mas a FK de `entity_id` aponta
somente a profissionais. A migration futura precisa preservar a defesa de FK
sem apagar eventos existentes:

- adicionar colunas nullable `professional_id` e `series_id`;
- preencher `professional_id=entity_id` nos eventos atuais;
- substituir a FK genérica atual por FKs tipadas;
- exigir exatamente uma referência tipada e igualdade com `entity_id`;
- aceitar `entity_type=professional|series` de forma coerente com a referência;
- manter consultas e exportação da slice A compatíveis.

Eventos novos:

- `scheduling.series.created`;
- `scheduling.series.updated`;
- `scheduling.series.archived`;
- `scheduling.series.reactivated`.

`before_data`/`after_data` podem conter IDs opacos, status, revisão, modo,
capacidade, duração, weekdays, horário, timezone e vigência. Não contêm nomes,
payload HTTP, dados de aluno, preço, motivo livre, e-mail, telefone ou CPF.

### 9.6 Compatibilidade da migration

- estritamente evolutiva sobre a `061`, sem seed e sem séries de backfill;
- não altera profissionais, catálogo, planos, matrículas, contratos, check-ins
  ou app do atleta;
- dados existentes de profissionais, command requests e auditoria preservam
  contagens e contratos;
- banco vazio aplica toda a sequência; upgrade desde `061` começa com grade
  vazia; segunda execução aplica zero migrations;
- número final pertence ao integrador e produção continua em `052` até decisão
  separada.

## 10. Contratos públicos e internos

### 10.1 `organizations/access -> scheduling`

Reutilizar sem ampliar a superfície pública da slice A:

```text
ResolveSchedulingUnitContext(authenticated_user_id, unit_id)
  -> organization_id, unit_id, unit_name, role, authorized
```

O adapter deve considerar o estado operacional atual da unidade ao definir
`authorized`. Timezone não vem deste contrato na proposta inicial; ele pertence
à série até existir uma decisão canônica de timezone da unidade.

### 10.2 `catalog -> scheduling`

Contrato obrigatório, pertencente ao package público de aplicação/domínio do
catálogo, sem DTO HTTP ou GORM:

```text
ResolveSchedulingDraftReferences(
  organization_id,
  unit_id,
  target { kind: modality|service_offering, id },
  space_id?
) -> {
  target { id, kind, display_name, status, enabled_in_unit },
  space? { id, display_name, status, default_capacity }
}
```

Regras:

- o resolver consegue devolver recurso arquivado/desativado para diagnóstico,
  mas referências de outro tenant retornam `not found`;
- comando exige ativo/habilitado; leitura/preview de rascunho existente inclui
  saúde e rótulo atuais;
- descrição, cor, preço, plano, GORM model e histórico de auditoria do catálogo
  não atravessam o port;
- `default_capacity` é sugestão e não regra;
- o frontend pode usar endpoints de catálogo existentes para preencher
  seletores, mas o comando sempre revalida no backend pelo port.

### 10.3 `scheduling` interno — profissional

O caso de uso resolve `Professional` pelo repository/serviço do próprio módulo
e exige:

```text
ResolveActiveProfessionalForSeries(organization_id, unit_id, professional_id)
  -> professional_id, active_in_unit, anonymized
```

Não cria contrato com `people` ou `users`. Nome é resolvido para a resposta
autorizada, mas não entra na série, preview persistido, log ou auditoria.

### 10.4 `memberships`

Nenhuma porta de memberships é necessária. Em particular, a slice não chama
`ResolveSchedulingEntitlements` e não pergunta se uma pessoa possui plano,
matrícula, direito, `booking_required` ou saldo.

A compatibilidade futura é somente semântica:

- target usa o mesmo par tipado `kind+id` do catálogo;
- `delivery_mode` usa o mesmo vocabulário `group|individual`;
- uma divergência entre grade e direito não é erro de rascunho, porque não há
  pessoa nem matrícula no fluxo.

### 10.5 `reservations`

Nenhum contrato é criado ou consumido. Ocorrência virtual não possui
`session_id`, revisão de sessão, status publicado ou ocupação e não pode ser
reservada. `GetSessionReservationSummary` permanece fora até `SPEC-007`.

### 10.6 `attendance`

Nenhum contrato é criado ou consumido. Preview não é fonte de presença,
check-in ou decisão de entrada. `attendance` só poderá resolver
`SessionSnapshot` publicado depois da slice C e das specs autorizadas.

### 10.7 Contrato de preview

Tipos `ScheduleSeriesDraftV1`, `SchedulePreviewOccurrenceV1`,
`SchedulePreviewIssueV1` pertencem a `scheduling` e são privados à API owner
nesta slice. O sufixo V1 protege a futura UI, mas não os transforma em eventos
ou contratos consumíveis por outros módulos.

## 11. Contratos HTTP propostos

Todas as rotas ficam sob `/api/v1/units/:unit_id/scheduling`, exigem sessão
tenant, `OWNER`, capabilities `scheduling` e `scheduling_grid`, além de CSRF em
mutações. Coach recebe `404`/rota não autorizada conforme o padrão existente,
sem revelar rascunhos.

### 11.1 Listar e detalhar

```http
GET /series?include_archived=false
GET /series/:series_id
```

- ordenação: ativas antes de arquivadas, depois horário local, primeiro
  weekday e ID;
- resposta inclui revisão, campos persistidos e `reference_health` atual;
- nomes atuais aparecem apenas na resposta autorizada;
- não inclui sessão, ocupação, aluno, plano, preço ou audit log.

### 11.2 Preview de proposta

```http
POST /series/preview
```

```json
{
  "series_id": null,
  "target": { "kind": "modality", "id": "uuid" },
  "space_id": "uuid-or-null",
  "professional_id": "uuid",
  "delivery_mode": "group",
  "capacity": 20,
  "duration_minutes": 60,
  "weekdays": [1, 3, 5],
  "local_start_time": "18:30",
  "timezone": "America/Sao_Paulo",
  "effective_from": "2026-09-01",
  "effective_until": null,
  "preview_from": "2026-09-01",
  "preview_until": "2026-10-26"
}
```

- `series_id` é opcional e serve somente para excluir a versão atual no
  preview de edição; precisa pertencer à unidade;
- não persiste nada e retorna `200` mesmo com conflitos/referências
  indisponíveis estruturalmente resolvíveis;
- input malformado, tenant adversário ou limite excedido retorna erro;
- resposta contém `occurrences`, `issues`, `ready_for_future_publication`,
  janela e fingerprint, nunca `session_id`.

### 11.3 Preview da grade persistida

```http
GET /preview?from=2026-09-01&until=2026-09-14
```

- expande todas as séries `draft` da unidade;
- retorna conflitos, referências indisponíveis e ordenação determinística;
- não aceita status de aluno, `person_id`, plano ou filtros de reserva;
- séries arquivadas não entram.

### 11.4 Criar e atualizar rascunho

```http
POST  /series
PATCH /series/:series_id
Idempotency-Key: scheduling-series.<uuid>
```

Body de criação contém os campos recorrentes do preview, sem janela de
preview. Atualização é substituição completa da configuração mutável e inclui
`expected_revision`; isso evita semântica ambígua de arrays parciais de
weekdays. Retorna `201`/`200`; replay de criação retorna `200`.

Salvar não executa preview implicitamente e não falha por sobreposição nesta
proposta. Referências, timezone, capacidade, duração e vigência precisam ser
válidos no commit.

### 11.5 Arquivar e reativar

```http
POST /series/:series_id/archive
POST /series/:series_id/reactivate
```

Ambos exigem `Idempotency-Key`, `expected_revision` e motivo de 3–240
caracteres. Reativação revalida referências; não existe `DELETE`.

### 11.6 Erros estáveis

| HTTP | Código | Situação |
|---:|---|---|
| 400 | `invalid_request` | JSON, UUID, data ou hora malformada |
| 404 | `unit_not_found` | unidade ausente, inativa ou fora do escopo |
| 404 | `schedule_series_not_found` | série ausente/outro tenant |
| 409 | `scheduling_revision_conflict` | revisão antiga |
| 409 | `schedule_series_archived` | edição incompatível com arquivada |
| 409 | `idempotency_key_reused` | chave com outro fingerprint |
| 422 | `scheduling_series_invalid` | recorrência, modo, capacidade, duração ou vigência inválida |
| 422 | `scheduling_target_unavailable` | modalidade/serviço inexistente, arquivado ou desativado |
| 422 | `scheduling_space_unavailable` | espaço informado inexistente/arquivado |
| 422 | `scheduling_professional_unavailable` | profissional arquivado, anonimizado ou fora da unidade |
| 422 | `invalid_local_time` | horário local inexistente ou ambíguo |
| 422 | `scheduling_preview_range_invalid` | janela ausente, invertida ou acima de 56 dias |
| 422 | `scheduling_preview_too_large` | ocorrências/conflitos excedem o limite |

Sobreposição válida retorna issue `professional_overlap|space_overlap` em
preview, não erro HTTP. Erros incluem `request_id`; SQL, constraint, nome de
recurso de outro tenant, payload completo e PII não aparecem.

Os códigos `*_unavailable` da tabela bloqueiam criação, atualização e
reativação. Em preview, uma referência existente no mesmo tenant que ficou
arquivada/desativada é convertida em issue bloqueante e a resposta continua
`200`; referência inexistente ou adversária não é revelada.

## 12. Privacidade, exportação, retenção e segurança

### 12.1 Minimização e exportação

- a série não armazena nome, telefone, e-mail, CPF, endereço, dado clínico,
  remuneração ou observação livre;
- `professional_id` é dado pseudônimo e a atribuição de horário entra na
  privacy export do profissional;
- ampliar `GET /professionals/:id/privacy-export` com
  `draft_series_assignments`, contendo IDs, unidade, alvo tipado, recorrência,
  intervalo, estado e timestamps, sem nomes de catálogo/conta;
- preview não é persistido e, portanto, não entra em exportação;
- não existe exportação de aluno, pois nenhum aluno participa da slice;
- exportação continua exclusiva do owner e gera auditoria conforme o padrão da
  fundação de profissionais.

### 12.2 Anonimização

- nenhuma série contém snapshot de nome; por isso não há texto a reescrever;
- anonimização de profissional é bloqueada enquanto houver série `draft`
  ativa atribuída a ele;
- séries arquivadas podem preservar o ID pseudônimo e passam a resolver o nome
  neutro da slice A;
- anonimização não apaga horários, IDs ou auditoria e não chama `people`.

### 12.3 Retenção proposta

- série `draft` ativa permanece enquanto configurada pela unidade;
- série arquivada e sua auditoria tornam-se elegíveis a expurgo após 1.825
  dias, removendo primeiro a auditoria e depois a série para preservar FKs;
- command requests de série tornam-se elegíveis após 365 dias;
- preview e conflitos não são persistidos;
- o comando de retenção deve operar em dry-run por padrão, emitir apenas
  contagens/IDs e preservar eventos ainda dentro do prazo;
- prazos são proposta de produto/privacidade e precisam de confirmação antes
  de `ready`; não são herdados silenciosamente de students ou contracts.

### 12.4 Segurança

- nenhum endpoint é público/anônimo;
- backend deriva tenant/ator, aplica CSRF e limites de body;
- IDs adversários retornam `404`;
- motivo livre não entra em log/métrica;
- timezone e expansão possuem limites contra uso excessivo;
- preview não executa efeito externo nem cria claim operacional;
- logs e traces não contêm nomes, payload completo ou lista de ocorrências.

## 13. Observabilidade e operação

### 13.1 Logs

- `scheduling_series_command_completed|failed`: comando, resultado,
  organização, unidade, series ID, revisão, duração e `request_id`;
- `scheduling_preview_completed|failed`: modo `candidate|grid`, janela,
  contagens, issues por tipo, duração e `request_id`;
- `scheduling_draft_conflict_detected`: tipo de recurso e contagem, sem nomes
  ou intervalos completos em labels/log de alta frequência;
- não registrar recorrência completa, weekdays, motivo, nomes ou response body.

### 13.2 Métricas

- `engagefit.scheduling.series.commands{command,result}`;
- `engagefit.scheduling.preview.requests{mode,result}`;
- `engagefit.scheduling.preview.duration{mode}`;
- `engagefit.scheduling.preview.issues{issue_type}`;
- gauge agregado de séries draft/archived por ambiente, sem IDs como labels.

### 13.3 Operação e falhas

- não existe worker, heartbeat de materialização ou alerta de horizonte na B;
- falha de preview não altera rascunhos;
- suporte usa `request_id` e `series_id`, nunca pede nome do profissional,
  payload completo ou credencial;
- reconciliação futura pode conferir séries sem weekdays, referências e
  contagens em dry-run, mas não cria sessão nem corrige dados automaticamente.

## 14. Feature flags, rollout e compatibilidade

### 14.1 Capabilities propostas

Manter `FEATURE_SCHEDULING_ENABLED`/`scheduling` para a fundação existente e
adicionar:

```text
FEATURE_SCHEDULING_GRID_ENABLED -> scheduling_grid
```

Regras:

- default `false` em development, test e production;
- exige `FEATURE_SCHEDULING_ENABLED=true` e
  `FEATURE_CATALOG_ENABLED=true` no startup;
- alvo de serviço só aparece/é considerado disponível quando
  `service_catalog=true`;
- frontend trata campo ausente como `false`;
- backend aplica os gates, independentemente de menu/hash;
- não existe flag de materializador nesta slice;
- a flag não autoriza efeito externo, publicação ou produção.

### 14.2 Ordem segura futura

1. obter confirmação das decisões e promover a spec para `ready`;
2. criar prompt separado somente depois dessa promoção;
3. implementar em worktree/branch própria, sem escolher número final de
   migration;
4. aplicar migration aditiva sobre `061` em banco descartável;
5. publicar backend com `scheduling_grid=false`;
6. publicar frontend tolerante a capability ausente;
7. validar ports de catálogo e regressão da slice A;
8. habilitar apenas em ambiente local/homologação;
9. homologar rascunhos e preview em `360x800` com dados sintéticos;
10. qualquer ativação de produção depende de decisão separada, backup/restore,
    migrations `053+` e revisão do estado real de produção.

### 14.3 Rollback

- desligar `scheduling_grid` oculta e bloqueia grade/preview, preservando a
  tela de profissionais quando `scheduling` continuar ativa;
- schema e rascunhos aditivos permanecem;
- frontend/backend podem voltar em qualquer ordem;
- nenhum fluxo legado, matrícula, check-in, contrato, reserva ou presença
  passa a depender da grade;
- não existe sessão para cancelar ou reconciliar.

## 15. Critérios de aceite

### 15.1 Produto e UX

- owner cria rascunho semanal por modalidade ou serviço em `360x800`;
- profissional é obrigatório, espaço é opcional e capacidade do espaço é só
  sugestão;
- individual força capacidade 1; grupo aceita capacidade explícita;
- owner escolhe dias, hora, timezone, início e fim opcional;
- preview não salvo e preview da grade usam janela explícita e mesma expansão;
- ocorrências são virtuais e a UI afirma que não foram publicadas;
- horários adjacentes não conflitam; sobreposição positiva de profissional ou
  espaço aparece como bloqueio;
- conflito não impede salvar rascunho nesta proposta;
- owner edita com revisão, arquiva, filtra arquivados e reativa após revalidação;
- coach/platform admin não leem grade nem preview;
- nenhum card mostra vagas restantes, preço, plano, aluno ou ação de reserva;
- loading, vazio, erro, retry, conflito e referência inativa são recuperáveis.

### 15.2 Domínio e arquitetura

- série possui exatamente um alvo e um profissional ativo da unidade;
- timezone IANA, vigência, weekdays, duração e capacidade seguem os limites;
- expansão rejeita horário DST inexistente/ambíguo sem deslocamento silencioso;
- scheduling acessa catálogo apenas pelo port definido;
- nenhuma porta de memberships/reservations/attendance é chamada ou criada;
- não existe tabela/entidade de sessão ou ocorrência persistida;
- nomes externos não são persistidos na série;
- renome/arquivo/inativação seguem a seção 7 sem cascade destrutivo;
- UUID adversário de outra organização/unidade não vaza existência;
- migration preserva integralmente profissionais, commands e auditoria da A.

### 15.3 Consistência, privacy e segurança

- replay idêntico preserva ID/revisão e chave divergente não executa;
- revisão antiga nunca sobrescreve série;
- falha de weekday/auditoria/command request reverte a mutação;
- comandos concorrentes no mesmo rascunho deixam um vencedor por revisão;
- rascunhos concorrentes diferentes podem ser salvos e aparecem em conflito;
- export do profissional inclui atribuições sem nomes de catálogo/conta;
- anonimização é bloqueada por série ativa e não exige apagar nome da série;
- logs, métricas, auditoria e command requests não contêm nomes/PII;
- capability ausente/falsa esconde UI e bloqueia API;
- nenhum efeito externo ou decisão de acesso é executado.

## 16. Matriz de testes obrigatória

| Camada | Cenários mínimos |
|---|---|
| domínio | normalização, weekday 0..6, vigência inclusiva, modo/capacidade, duração, timezone IANA, DST ambíguo/inexistente e intervalos semiabertos |
| expansão | limites de datas, effective bounds, occurrence key, ordenação estável, 14/56 dias e cap de volume |
| conflitos | profissional na unidade e entre unidades, espaço somente na unidade, sem espaço, adjacência, arquivados ignorados e múltiplos conflitos determinísticos |
| aplicação | create/update/archive/reactivate, reference health, replay, revisão, rollback, relógio injetável e preview sem persistência |
| ports | contrato de catalog com ativo/arquivado/desativado/adversário; ausência de DTO/GORM e ausência de ports memberships/reservations/attendance |
| PostgreSQL | constraints, weekdays, FKs internas, extensão compatível de command/audit, transações, locks e isolamento com duas organizações/unidades |
| migration | banco vazio, upgrade exato desde `061` com dados de profissionais/commands/audit, contagens preservadas e segunda execução zero |
| HTTP | owner/coach/admin, cookie/Bearer, CSRF, capabilities, ranges, volume, erros estáveis, request_id e UUID adversário |
| frontend | criação/edição/arquivo/reativação, previews, conflitos, referências inativas, capability ausente e formulário recuperável |
| mobile/a11y | `360x800`, sem overflow, toque 44 px, foco, Escape, labels, weekday/date/time por teclado e estado sem depender de cor |
| privacidade | export de atribuições, anonimização bloqueada, retenção dry-run proposta e ausência de nomes/PII em logs/auditoria |
| regressão | profissionais A, catálogo, planos, matrículas, contratos, check-ins, retenção, workout e PWA inalterados |

Quando autorizada no futuro, a implementação deverá executar o baseline dos
dois repositórios, PostgreSQL real, race detector em ambiente suportado,
migrations vazio/upgrade/idempotência, TypeScript, build, Playwright focado e
regressão completa. Esta seção não autoriza executar esses trabalhos agora.

## 17. Proibições para um futuro executor

O futuro agente, somente depois de `ready`, não deve:

- implementar `SPEC-006/C-D`, `SPEC-003/D`, `SPEC-007`, `SPEC-008` ou
  `SPEC-010+`;
- criar sessão, publicação, materializador, cancelamento, exceção ou evento de
  sessão;
- consultar tabelas/repositories/models de catálogo ou memberships;
- adicionar pessoa, plano, matrícula, preço, saldo, reserva, ocupação,
  presença ou decisão de acesso à série/preview;
- transformar preview em contrato público de reservations/attendance/athlete;
- alterar roles ou dar mutação ao coach;
- usar nome como FK/identidade, copiar PII ou criar dados clínicos/financeiros;
- implementar RRULE, feriado, calendário externo ou multiunidade;
- tocar fluxos legados, produção, banco compartilhado ou integrações externas;
- editar migrations publicadas, escolher número final, fazer push, merge,
  deploy ou habilitar flags.

## 18. Decisões adotadas na implementação

As propostas abaixo foram adotadas na entrega local por serem os defaults
conservadores coerentes com a guia `SPEC-006`:

1. **Conflitos em rascunho:** permitir salvar séries sobrepostas e tratá-las
   como bloqueio no preview, deixando o bloqueio transacional definitivo para
   a publicação da slice C.
2. **Fonte do timezone:** persistir timezone IANA por série, com
   `America/Sao_Paulo` apenas como sugestão visível, sem adicionar agora um
   timezone canônico à unidade.
3. **Janela de preview:** UI inicial de 14 dias, API com janela explícita de no
   máximo 56 dias, até 2.000 ocorrências e 200 conflitos, sem resposta parcial.
4. **Ciclo do rascunho:** usar somente `draft|archived`, sem exclusão física,
   status `published/cancelled` ou cascade quando referências externas forem
   inativadas.
5. **Flags:** criar `scheduling_grid` separada de `scheduling`, para que a
   fundação de profissionais já validada possa permanecer ativa enquanto a
   grade continua desligada.
6. **Retenção:** manter rascunho ativo enquanto configurado, conservar série
   arquivada e auditoria por 1.825 dias e tornar command request elegível a
   expurgo após 365 dias.
7. **Privacidade do profissional:** permitir arquivamento mesmo com rascunho
   ativo, mas bloquear anonimização até reatribuir ou arquivar todas as séries
   ativas.
8. **Fronteiras futuras:** não criar contrato algum com `memberships`,
   `reservations` ou `attendance` nesta slice; a B expõe preview somente ao
   owner e a C define os primeiros contratos de sessão publicada.

Eventuais ajustes futuros devem preservar compatibilidade com séries e sessões
já persistidas e usar migration aditiva.
