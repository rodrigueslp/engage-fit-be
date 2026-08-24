# SPEC-006 — Agenda, profissionais e grade recorrente

- Status: `draft`
- Versão: `0.1`
- Data: 2026-08-24
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-003/A-B-C`
- Bloqueia: `SPEC-003/D`, `SPEC-007`, `SPEC-008`, partes de `SPEC-009`,
  `SPEC-010` e `SPEC-012`
- Ownership principal: módulo novo `scheduling`
- Colaboradores por contrato público: `organizations/access`, `catalog` e
  `memberships`
- Consumidores futuros: `reservations`, `attendance` e `athlete`
- Repositórios futuros envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration: número não definido; será reservado pelo integrador

Esta é uma spec-guia detalhada, ainda sem autorização de implementação. Cada
slice da seção 18 precisa de spec executável própria, revisão de integração e
estado `ready` antes de criar migration ou contrato público definitivo.

## 1. Objetivo e resultado esperado

O owner configura, pelo celular, os profissionais da unidade e uma grade
recorrente de aulas ou atendimentos. Cada item da grade escolhe exatamente uma
modalidade ou um serviço já existente, um espaço, um profissional responsável,
horário, duração, capacidade e modo de atendimento `group` ou `individual`.

Ao publicar a grade, o sistema cria sessões concretas, com data e intervalo de
tempo inequívocos. O owner vê e altera a agenda; o coach possui uma leitura
operacional clara. Exceções, mudanças e cancelamentos preservam histórico e
produzem contratos estáveis para reservas, presença e aplicativo do aluno.

O resultado termina antes da ocupação da sessão:

```text
profissional + grade recorrente
    -> preview sem conflito
    -> publicação
    -> sessões concretas e auditáveis
    -> leitura por owner/coach e portas públicas
```

Esta spec não reserva vaga, não cria lista de espera, não registra presença e
não decide se uma pessoa pode acessar ou consumir um direito.

## 2. Problema, evidências e proveniência

### 2.1 Problema operacional

**Existente:** `catalog` já mantém modalidades por organização, serviços
ofertáveis por organização, ativações por unidade e espaços por unidade. Esses
cadastros não possuem preço, profissional, duração ou horário.

**Existente:** `memberships` já preserva direitos versionados com alvo de
modalidade ou serviço, `delivery_mode=group|individual` e
`booking_required`. Matrícula, direito e preço não criam uma ocorrência no
calendário.

**Existente:** check-ins e o app do atleta usam modelos anteriores e não podem
ser convertidos silenciosamente em reserva ou presença de uma sessão.

**Entrada Alados — `REQ-SCHEDULE-01/02`:** a operação precisa representar
CrossFit, Spinning, Pilates, Recovery e atendimentos especializados com
horários, profissional, espaço e capacidade.

**Risco:** guardar horário no cadastro de modalidade/serviço ou tratar sessão
como direito do plano mistura catálogo, oferta comercial, agenda e consumo.
Isso impediria alterações auditáveis, criaria conflitos de tenancy e obrigaria
`SPEC-003/D` a inventar um contador antes de reservas e presença.

### 2.2 Decisões propostas

- **Inferência EngageFit:** profissional é uma entidade operacional de
  `scheduling`. Pode estar ligado a uma conta existente, mas a ligação não
  cria conta, papel ou permissão.
- **Inferência EngageFit:** uma série recorrente possui um único horário local
  e um ou mais dias da semana. Dois horários no mesmo dia são duas séries; isso
  mantém criação, conflito e alteração compreensíveis no celular.
- **Inferência EngageFit:** capacidade é sempre copiada explicitamente para a
  série e para a sessão. A capacidade padrão do espaço é apenas sugestão.
- **Inferência EngageFit:** sessões usam intervalo semiaberto
  `[starts_at, ends_at)`, permitindo que uma termine exatamente quando outra
  começa.
- **Inferência EngageFit:** recorrência inicial é semanal e estruturada, sem
  aceitar RRULE livre. Exceções representam alterações e datas especiais.
- **Inferência EngageFit:** publicar materializa uma janela futura limitada e
  um materializador idempotente a estende diariamente.
- **Inferência EngageFit:** arquivar ou renomear referência do catálogo não
  reescreve sessão já publicada; cada sessão mantém nomes de exibição em
  snapshot.
- **Validação futura:** terminologia, horizonte, feriados, conflito e regras de
  visibilidade ainda precisam ser validados com o Alados na seção 17.

### 2.3 Fontes

Nenhuma fonte externa adicional foi necessária para esta versão. O benchmark
oficial já registrado em `SPEC-001/B`, `SPEC-001/D`, `SPEC-003/A` e
`SPEC-003/C` sustenta a separação entre modalidade, serviço, espaço, plano,
profissional, grade e sessão. Esta spec registra como inferência toda decisão
nova do EngageFit e não copia fluxos ou modelos de concorrentes.

## 3. Escopo

### 3.1 Incluído

- profissional como entidade operacional, com nome mínimo e vínculo opcional
  a uma conta de owner/coach já existente;
- associação do profissional à unidade atual, status ativo/arquivado e leitura
  segura por owner/coach;
- grade semanal recorrente, intervalo de vigência e preview de ocorrências;
- exatamente um alvo por série/sessão: `modality` ou `service_offering`;
- espaço, unidade, horário local, timezone, duração, capacidade e profissional
  responsável explícitos;
- atendimento `group` ou `individual`;
- rascunho, publicação, alteração a partir de uma data e cancelamento da série;
- sessões concretas materializadas, sessão extra e exceção de uma ocorrência;
- alteração ou cancelamento de uma sessão futura;
- detecção concorrente de conflito de profissional e espaço;
- isolamento por organização e unidade;
- auditoria transacional e eventos públicos mínimos de sessão;
- experiência mobile-first para owner e leitura de agenda para coach;
- ports públicos estreitos entre catálogo, direitos, agenda, reservas,
  presença e atleta;
- feature flag, rollout aditivo e operação do materializador.

### 3.2 Fora do escopo

- reserva, cancelamento de reserva, lista de espera, no-show, reposição,
  antecipação, reagendamento de aluno ou consumo de vaga (`SPEC-007`);
- check-in, catraca, presença, liberação ou decisão final de acesso
  (`SPEC-003/D` e `SPEC-008`);
- saldo de sessões, pacote, frequência usada ou estorno de crédito;
- cobrança, pagamento, contrato, assinatura, aceite ou PAR-Q;
- integração real com Wellhub ou TotalPass;
- sessão que pertença simultaneamente a mais de uma unidade;
- calendário externo, videoconferência ou sincronização CalDAV/Google;
- comissão, folha, disponibilidade trabalhista ou prontuário clínico;
- cadastro de telefone, CPF, endereço, especialidade clínica, conselho
  profissional ou documento do profissional;
- criação/desativação de conta de acesso dentro de agenda;
- múltiplos responsáveis ou assistentes na mesma sessão;
- RRULE arbitrária, recorrência mensal e regra automática de feriados;
- notificações de alteração/cancelamento; mensagens pertencem a spec futura;
- exposição da agenda no app do aluno nesta entrega; somente o contrato de
  leitura é preparado.

## 4. Vocabulário

- **Modalidade:** atividade reutilizável do catálogo, como CrossFit ou
  Spinning.
- **Serviço ofertável:** oferta agendável do catálogo, como Recovery,
  avaliação ou massagem.
- **Espaço:** local físico de uma unidade, como área principal ou sala 1.
- **Profissional:** responsável operacional que pode conduzir uma sessão. Não
  é sinônimo de `Person`, `Student`, `User` ou `Coach`.
- **Conta vinculada:** referência opcional a um usuário operacional existente.
  Não é fonte de nome, permissão ou tenancy da sessão.
- **Série (`ScheduleSeries`):** regra semanal que produz ocorrências.
- **Grade:** conjunto das séries publicadas de uma unidade.
- **Sessão (`ScheduleSession`):** ocorrência concreta, com ID, data, começo,
  fim, capacidade, alvo, espaço e profissional próprios.
- **Sessão extra:** sessão concreta criada sem série para uma data específica.
- **Exceção:** mudança ou cancelamento manual de uma ocorrência gerada.
- **Capacidade:** limite configurado de participantes; não significa vagas
  livres, reservas confirmadas ou presenças.
- **Publicação:** transição que torna a série e suas sessões visíveis aos
  leitores autorizados.
- **Materialização:** criação idempotente das sessões concretas dentro do
  horizonte futuro.
- **Estado efetivo:** `upcoming`, `in_progress` ou `past`, calculado pelo
  relógio a partir de uma sessão `scheduled`.

Na UI, usar `Agenda`, `Grade`, `Profissionais`, `Aula` para modalidade e
`Atendimento` para serviço quando o contexto permitir. A API usa `series` e
`sessions` para manter contrato neutro.

## 5. Ownership e modelo de domínio

### 5.1 Propriedade por módulo

| Conceito | Módulo proprietário | Regra de colaboração |
|---|---|---|
| organização, unidade, usuário e papel | `organizations/access` | `scheduling` recebe contexto e valida conta por port público |
| modalidade, serviço e espaço | `catalog` | resolução por IDs e snapshots em port público |
| plano, matrícula e direitos | `memberships` | leitura estreita de termos; nunca join ou saldo em agenda |
| profissional, série, sessão, exceção | `scheduling` | tabelas, domínio, API e eventos próprios |
| reserva, fila, no-show e ocupação | `reservations` | consome sessão; não grava tabela de agenda |
| presença e validação de entrada | `attendance` | consome sessão/reserva/direito por ports |
| experiência autenticada do aluno | `athlete` | compõe leituras; não é fonte da agenda |
| documento/aceite | `contracts` | nenhuma dependência de `scheduling` nesta spec |

`scheduling` não consulta nem importa repositories, models GORM, DTOs HTTP ou
tabelas de `memberships`, `catalog`, `people` ou `contracts`. IDs externos são
referências opacas validadas por portas públicas. Snapshots de exibição não
transferem ownership do conceito externo.

### 5.2 Agregados

```text
Professional
  └── UnitAssignment[]

ScheduleSeries
  ├── RecurrenceWeekday[]
  └── ScheduleSession[] materializadas

ScheduleSession
  ├── origem: series | extra
  ├── configuração concreta
  ├── revisão
  └── cancelamento opcional
```

Uma série é a fonte da recorrência; a sessão é a fonte da ocorrência
publicada. Leitores futuros nunca recalculam uma sessão a partir da série.

### 5.3 Profissional e conta

- profissional pertence a uma organização e precisa estar ativo na unidade da
  série/sessão;
- `linked_user_id` é opcional, opaco e único por organização quando preenchido;
- o vínculo só aceita owner ou coach ativo e autorizado na unidade por meio do
  port de access;
- criar profissional não cria `User`, `Person` ou convite;
- vincular uma conta não concede mutação de agenda e não amplia a allowlist do
  usuário;
- o nome operacional é persistido em `scheduling`; a leitura não depende da
  conta continuar ativa;
- arquivamento é bloqueado enquanto houver série publicada capaz de gerar
  novas sessões ou sessão futura atribuída ao profissional.

## 6. Invariantes

### 6.1 Tenancy e referências

1. Toda entidade de agenda possui `organization_id`; toda série, sessão e
   associação operacional possui exatamente um `unit_id`.
2. Uma sessão pertence a uma única unidade. IDs de outra unidade/organização
   respondem como inexistentes.
3. `organization_id` e `unit_id` vêm da sessão e da rota validada, nunca do
   body como fonte de autorização.
4. Nesta fase, `unit_id` da rota deve ser igual a `users.box_id`. A evolução
   multiunidade depende de `SPEC-001/C` e allowlist explícita.
5. Profissional precisa estar ativo na mesma unidade da série/sessão.
6. Modalidade/serviço e espaço precisam ser resolvidos como ativos e
   habilitados na unidade ao criar rascunho, publicar, criar sessão extra ou
   alterar um campo que os referencie.
7. Uma mudança posterior no catálogo não apaga nem reescreve sessões
   publicadas. Novos comandos exibem alerta e exigem referências válidas.
8. Não existem joins de persistência entre tabelas de `scheduling` e tabelas
   de outro módulo.

### 6.2 Conceitos distintos

1. Cada série/sessão referencia exatamente um `target_kind` entre `modality`
   e `service_offering`, mais um único `target_id`.
2. `space_id`, `professional_id` e `target_id` nunca são intercambiáveis.
3. `delivery_mode` descreve a sessão; recorrência/cobrança de plano é outro
   conceito.
4. `booking_required` pertence ao direito de `memberships`, não à sessão.
5. Sessão não contém `plan_id`, `enrollment_id`, `person_id`, saldo, preço,
   reserva ou presença.

### 6.3 Horário, duração e conflito

1. Horário recorrente é persistido como hora local sem segundos, timezone IANA
   e dias da semana.
2. Sessão persiste `starts_at`/`ends_at` em `timestamptz`, além de data/hora
   local e timezone em snapshot.
3. Duração aceita inicialmente 5 a 1.440 minutos; `ends_at` é maior que
   `starts_at` e uma sessão não pode atravessar mais de um dia civil local.
4. Intervalos usam `[starts_at, ends_at)`: adjacência é permitida; qualquer
   interseção positiva é conflito.
5. Um profissional não pode ter duas sessões `scheduled` sobrepostas na mesma
   organização, inclusive em unidades diferentes.
6. Um espaço não pode ter duas sessões `scheduled` sobrepostas na mesma
   unidade.
7. Sessões canceladas não participam de conflito.
8. Não existe override de conflito nesta versão. A transação inteira falha com
   detalhes seguros dos recursos e ocorrências em conflito.
9. Preview e publicação aplicam o mesmo algoritmo; a publicação revalida sob
   locks e não confia no preview anterior.
10. Mutações adquirem advisory locks transacionais em ordem canônica para cada
    profissional e espaço envolvidos e reconsultam sobreposição antes do
    commit. Teste PostgreSQL concorrente é obrigatório.

### 6.4 Capacidade e modo de atendimento

1. Toda série e sessão possui capacidade explícita de 1 a 10.000.
2. O valor padrão do espaço apenas preenche o formulário; não é lido em tempo
   real depois da publicação.
3. `individual` exige capacidade exatamente 1.
4. `group` aceita capacidade de 1 a 10.000.
5. Reduzir capacidade futura não cancela nem mede reservas nesta spec. Quando
   `reservations` estiver habilitado, seu port de impacto precisa autorizar a
   mudança.
6. A API retorna `capacity`, nunca `available_capacity`, `remaining_spots` ou
   ocupação calculada.

### 6.5 Publicação

1. Rascunho não é visível ao coach, atleta ou portas públicas e não gera
   sessão persistida.
2. Publicação exige série válida, ao menos um dia da semana e uma ocorrência
   dentro do horizonte materializável.
3. Publicação materializa a janela inicial e grava série, sessões, auditoria,
   eventos e idempotência de forma atômica.
4. Se qualquer ocorrência conflitar ou falhar, nenhuma sessão da publicação é
   criada e a série permanece em rascunho.
5. Série publicada não volta a rascunho. Pode receber mudança efetiva futura
   ou cancelamento.
6. Toda sessão publicada possui revisão positiva e snapshot completo; nenhum
   consumidor precisa consultar catálogo/profissional para entender a
   ocorrência.

### 6.6 Recorrência e materialização

1. Recorrência inicial é semanal, com `interval_weeks=1`, uma hora local e um
   conjunto de dias `0..6` sem duplicidade.
2. `effective_from` é obrigatório; `effective_until` é opcional e inclusivo.
3. Uma série produz no máximo uma ocorrência por data local. Outra hora exige
   outra série.
4. Para ocorrência da série, a chave natural é
   `(series_id, occurrence_on)`, onde `occurrence_on` é a data original da
   regra, mesmo se a sessão for reagendada.
5. O materializador cria somente ocorrências até o horizonte configurado e
   nunca gera data anterior ao dia local da publicação.
6. Execução diária estende o horizonte. Repetição, múltiplas réplicas e retry
   não duplicam sessão por unique key, claim de run e lock por série.
7. Sessão marcada como exceção manual ou cancelada nunca é sobrescrita pelo
   materializador.
8. Alteração de série aplica somente a sessões futuras, não canceladas e sem
   override manual, a partir de `effective_on`.
9. Datas removidas pela nova regra são canceladas com origem
   `series_change`; datas adicionadas são criadas; datas correspondentes
   preservam o ID da sessão quando possível.
10. Horário local inexistente ou ambíguo por mudança de timezone nunca é
    deslocado silenciosamente. O preview/materializador retorna
    `invalid_local_time` e exige uma exceção explícita.

### 6.7 Exceções, alteração e cancelamento

1. Owner pode criar sessão extra ou alterar uma sessão futura. A alteração
   pode trocar horário, duração, alvo, espaço, profissional, capacidade e
   modo, sempre revalidando todos os invariantes.
2. Uma sessão alterada mantém ID, `occurrence_on`, origem e histórico; recebe
   `manual_override=true` e incrementa revisão.
3. Sessão iniciada ou passada é imutável por estas rotas.
4. Cancelamento exige motivo de 3 a 240 caracteres, ator, horário e revisão
   esperada; não existe exclusão física.
5. Cancelamento é final nesta spec. Reposição exige sessão extra com novo ID.
6. Cancelar série exige `effective_on`, mostra preview da quantidade afetada,
   impede novas materializações e cancela ocorrências futuras na mesma
   transação.
7. Séries/sessões com efeito já consumido por `reservations` ou `attendance`
   dependerão dos ports dessas specs. `scheduling` não altera suas tabelas.
8. Toda mutação grava auditoria e evento de sessão na mesma transação.

## 7. Fluxos do usuário

### 7.1 Cadastrar profissional

1. Owner abre `Agenda > Profissionais`.
2. Toca em `Adicionar profissional` e informa somente o nome.
3. Opcionalmente abre `Vincular conta` e seleciona uma conta operacional
   elegível da unidade.
4. Revisa a unidade e salva.
5. O profissional fica disponível para novas séries e sessões.

Erro de conta, conflito ou rede mantém o formulário recuperável. Arquivar
explica as dependências existentes e, se estiver em uso, oferece links/filtros
para as ocorrências que precisam de reassociação.

### 7.2 Criar e publicar grade

1. Owner abre `Agenda > Grade` e toca em `Novo horário recorrente`.
2. Escolhe `Aula`/modalidade ou `Atendimento`/serviço e o item do catálogo.
3. Escolhe espaço e profissional responsável.
4. Define `Em grupo` ou `Individual`, capacidade e duração.
5. Seleciona dias da semana, horário, início e fim opcional.
6. A tela mostra as próximas ocorrências e conflitos usando a mesma regra do
   backend.
7. Owner salva rascunho ou toca em `Publicar grade`.
8. A revisão informa quantidade e intervalo materializados e deixa explícito:
   `Publicar não reserva vagas nem avisa alunos`.

### 7.3 Consultar agenda

- mobile abre em `Hoje`, com troca curta de dia e opção `Semana`;
- card mostra hora, duração, modalidade/serviço, `Grupo`/`Individual`, espaço,
  capacidade total, profissional e estado;
- owner vê ações; coach recebe somente leitura;
- filtro `Minha agenda` aparece apenas quando a conta do coach está ligada a
  um profissional;
- não exibir preço, plano, matrícula, saldo ou “vagas restantes”.

### 7.4 Alterar

Ao editar sessão pertencente a série, o owner escolhe:

- `Somente esta sessão`: cria override manual;
- `Esta e as próximas`: abre preview de mudança da série a partir da data.

O preview informa sessões mantidas, alteradas, novas, canceladas e conflitos.
Conflito deixa o formulário aberto e destaca profissional/espaço e intervalo,
sem revelar dados de outro tenant.

### 7.5 Cancelar

- cancelamento de sessão pede motivo e confirma data, hora e profissional;
- cancelamento de série pede a primeira data afetada e mostra contagem;
- texto explica que reserva, presença e notificação não são tratadas por esta
  entrega;
- sessão cancelada permanece na agenda quando o filtro correspondente está
  ativo e no histórico/auditoria.

## 8. Experiência mobile-first e permissões

### 8.1 Navegação

- adicionar `PageKey` e hash `#/schedule`, condicionados a `scheduling`;
- item `Agenda` no grupo operacional para `OWNER` e `COACH`;
- owner vê seções `Agenda`, `Grade` e `Profissionais`;
- coach vê somente `Agenda`; hash direto para gestão é bloqueado também no
  backend;
- uma unidade não mostra seletor. O nome da unidade aparece no cabeçalho.

### 8.2 Layout e acessibilidade

- fluxo principal validado em `360x800`, uma coluna e sem overflow horizontal;
- alvos de toque mínimos de 44 px;
- campos de hora/data usam teclado e controles nativos adequados;
- seleção de alvo distingue visualmente `Modalidade` e `Serviço`;
- capacidade padrão aparece como sugestão editável, nunca valor escondido;
- revisão precede publicação e alterações em lote;
- loading, vazio, conflito, erro, retry, sucesso e capability desligada possuem
  estados explícitos;
- drawers/modais preservam foco, labels, Escape e rolagem interna;
- cor não é o único indicador de estado ou conflito.

### 8.3 Permissões

| Ação | Owner | Coach | Platform admin |
|---|---:|---:|---:|
| ler agenda publicada da unidade | sim | sim | não |
| usar filtro `Minha agenda` | se vinculado | se vinculado | não |
| ler profissionais necessários à agenda | sim | sim | não |
| criar/editar/arquivar profissional | sim | não | não |
| criar/editar/publicar/cancelar série | sim | não | não |
| criar/alterar/cancelar sessão | sim | não | não |
| ver auditoria técnica | não nesta UI | não | não |

Conta vinculada não transforma coach em editor. Autorizações continuam vindo
do papel autenticado e da unidade permitida.

## 9. Modelo de dados proposto

Os nomes são propostas para as specs executáveis. Não fixam migration nem
autorizam schema.

### 9.1 `scheduling_professionals`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório |
| `display_name` | varchar(100), normalizado, obrigatório |
| `linked_user_id` | UUID opaco nullable |
| `status` | `active|archived` |
| `revision` | inteiro positivo |
| `created_by_user_id` | ator |
| `created_at`, `updated_at`, `archived_at` | timestamptz |

Constraints/índices:

- unique parcial `(organization_id, linked_user_id)` quando não nulo;
- unique `(id, organization_id)`;
- nome com 1–100 caracteres; status/revisão válidos;
- índice por organização, status e nome normalizado para listagem.

Nomes iguais são permitidos: homônimos não podem ser mesclados. A UI pode
mostrar sufixo operacional da conta/unidade sem transformá-lo em identidade.

### 9.2 `scheduling_professional_units`

- `organization_id`, `unit_id`, `professional_id`, `active`, timestamps;
- PK `(unit_id, professional_id)`;
- FK composta apenas para entidades internas de `scheduling` e para o contexto
  de organização/unidade aprovado pela arquitetura;
- nesta spec, a API cria somente vínculo com a unidade autenticada.

### 9.3 `scheduling_series`

Campos mínimos:

- `id`, `organization_id`, `unit_id`;
- `status`: `draft|published|cancelled`;
- `target_kind`: `modality|service_offering`;
- `target_id` e `target_name_snapshot`;
- `space_id` e `space_name_snapshot`;
- `professional_id` e `professional_name_snapshot`;
- `delivery_mode`: `group|individual`;
- `capacity`, `duration_minutes`;
- `local_start_time TIME`, `timezone VARCHAR(64)`;
- `interval_weeks=1`, `effective_from`, `effective_until` nullable;
- `revision`, `published_at`, `cancelled_at`, `cancelled_reason`;
- atores de criação/publicação/cancelamento e timestamps.

Constraints:

- exatamente um target tipado; snapshots não vazios;
- FKs compostas para profissional/unidade dentro do módulo;
- capacidade, duração, datas, timezone, estados e campos condicionais;
- unique `(id, organization_id, unit_id)` para referências internas.

### 9.4 `scheduling_series_weekdays`

- `organization_id`, `unit_id`, `series_id`, `weekday SMALLINT`;
- PK `(series_id, weekday)`;
- `weekday` entre 0 e 6;
- FK composta para série no mesmo tenant/unidade.

### 9.5 `scheduling_sessions`

Campos mínimos:

- `id`, `organization_id`, `unit_id`;
- `series_id` nullable; `origin=series|extra`;
- `occurrence_on` nullable e obrigatório quando `origin=series`;
- `status=scheduled|cancelled`;
- target, espaço, profissional, modo e snapshots equivalentes aos da série;
- `capacity`, `starts_at`, `ends_at`, `local_date`, `local_start_time`,
  `timezone`;
- `source_series_revision` nullable;
- `manual_override` boolean;
- `revision`, `published_at`;
- cancelamento, motivo, atores e timestamps.

Constraints/índices:

- unique `(series_id, occurrence_on)` quando `series_id` não é nulo;
- target tipado e snapshots obrigatórios;
- `ends_at > starts_at`, capacidade e `individual=1`;
- índices `(unit_id, starts_at)`, `(professional_id, starts_at)` e
  `(unit_id, space_id, starts_at)` filtrados/adequados a `scheduled`;
- FKs compostas internas impedem profissional/série de outro tenant;
- nenhuma FK, join ou trigger aponta para tabela de catálogo, memberships,
  people ou contracts.

### 9.6 `scheduling_command_requests`

Registra idempotência de `professional_create`, `series_create`,
`series_publish`, `series_change`, `series_cancel`, `session_create`,
`session_change` e `session_cancel`:

- unicidade `(organization_id, command, idempotency_key)`;
- chave de 8–128 caracteres na política já usada por memberships;
- fingerprint SHA-256 de payload canônico, incluindo unidade e alvo;
- IDs/revisões e resposta estável em JSONB;
- replay igual retorna o resultado original; outro payload retorna
  `idempotency_key_reused`.

PATCH de rascunho também usa `expected_revision`; a chave não substitui
controle otimista.

### 9.7 `scheduling_audit_events`

- UUID, organização, unidade, ator, ação, tipo/ID da entidade, revisão;
- `changed_fields`, `before_data`, `after_data`, motivo e timestamp;
- payloads minimizados: IDs, datas, capacidade, modo, status e códigos;
- não guardar request HTTP, telefone, e-mail, CPF, conteúdo clínico ou segredo;
- mutação e auditoria na mesma transação.

Eventos mínimos:

- `scheduling.professional.created|updated|archived|reactivated`;
- `scheduling.series.draft_created|updated|published|changed|cancelled`;
- `scheduling.session.published|changed|cancelled`;
- `scheduling.session.extra_created`.

### 9.8 `scheduling_session_events`

Change log público e transacional, diferente da auditoria humana:

- `id UUID`, `sequence BIGINT` monotônica e única;
- `organization_id`, `unit_id`, `session_id`, `session_revision`;
- `event_type`: `session_published|session_changed|session_cancelled`;
- snapshots estruturados `before_state`/`after_state` sem nomes/PII;
- `occurred_at` e `schema_version=1`.

Consumers não consultam esta tabela. Usam o port da seção 11 e processam
`event_id` idempotentemente. A tabela não cria broker nem autoriza efeitos de
reservations/attendance nesta spec.

### 9.9 `scheduling_materialization_runs`

- `id`, `execution_key`, `status=running|completed|failed`;
- janela, séries lidas, sessões criadas/mantidas, conflitos e `error_code`;
- início/fim e timestamps;
- unique `execution_key`, sem nomes ou payloads de agenda.

## 10. Estratégia de materialização

### 10.1 Horizonte

Proposta inicial: materializar 56 dias futuros, configurável entre 28 e 180
dias. O valor final depende da validação da seção 17; não é contrato pronto.

O limite evita criar anos de sessões de uma série sem fim e mantém alterações
em lote pequenas. A série continua sendo a regra futura; a sessão é a fonte de
verdade dentro da janela publicada.

### 10.2 Publicação síncrona

Em uma única transação:

1. validar tenant, idempotência e revisão;
2. resolver referências por ports e congelar snapshots;
3. expandir datas locais até o horizonte;
4. converter para instantes com timezone e rejeitar horário ambíguo/inexistente;
5. adquirir locks canônicos de profissionais/espaços;
6. validar conflitos contra banco e dentro do lote;
7. publicar série e inserir sessões por natural key;
8. gravar auditoria, eventos e command request;
9. confirmar tudo ou nada.

### 10.3 Extensão diária

O runtime usa worker leve no monólito atual, sem serviço ou broker novo:

1. claim por `execution_key=materialize:<data UTC>`;
2. listar séries publicadas que alcançam a nova borda;
3. lock por série e processar cada uma em transação independente;
4. manter sessão existente, nunca sobrescrever override/cancelamento e criar
   apenas natural keys ausentes;
5. gravar run, métricas e erro normalizado.

Falha de uma série não reverte outras. Retry é seguro. Uma série bloqueada por
erro aparece ao owner como `Precisa de atenção`; não se desloca horário nem se
cria sessão parcial silenciosa.

### 10.4 Mudança de série

Preview é somente leitura. Aplicação reexecuta o cálculo sob lock e compara
`expected_revision`. A transação:

- preserva passado, sessão iniciada, overrides e cancelamentos;
- atualiza ocorrências correspondentes ainda automáticas;
- cancela datas removidas;
- cria datas novas;
- revalida conflitos e efeitos públicos;
- incrementa série/sessões e emite eventos.

O limite máximo de ocorrências por comando será definido pela spec executável;
payload ou intervalo excessivo retorna erro, não processamento parcial.

## 11. Contratos públicos entre módulos

Os tipos abaixo pertencem a packages públicos de aplicação/domínio, nunca a
handlers ou models GORM. Todos recebem organização/unidade explícitas e
retornam `not found` para referência adversária.

### 11.1 `organizations/access -> scheduling`

```text
ResolveSchedulingUnitContext(authenticated_user_id, unit_id)
  -> organization_id, unit_id, unit_name, timezone, role, authorized

ResolveProfessionalAccount(organization_id, unit_id, user_id)
  -> user_id, role: owner|coach, active, authorized_in_unit
```

O primeiro contrato deriva contexto e timezone sem aceitar organização do
body. O segundo valida somente o vínculo opcional da conta; não retorna nome,
e-mail, senha ou sessão e não concede permissão ao profissional. `scheduling`
não consulta `users` diretamente.

### 11.2 `catalog -> scheduling`

```text
ResolveSchedulingReferences(
  organization_id,
  unit_id,
  target { kind: modality|service_offering, id },
  space_id
) -> {
  target_id, target_kind, target_name, target_status, target_enabled_in_unit,
  space_id, space_name, space_status, default_capacity?
}
```

Uso: criar/alterar rascunho, publicar e criar/alterar sessão. O port não
retorna preço, agenda, GORM model ou catálogo completo.

### 11.3 `memberships -> scheduling/reservations/attendance/athlete`

```text
ResolveSchedulingEntitlements(
  organization_id,
  person_id,
  unit_id,
  target { kind, id },
  occurs_at
) -> []{
  enrollment_id,
  entitlement_id,
  booking_required,
  delivery_mode?,
  access_type
}
```

Este contrato pode informar que um direito exige reserva. Nesta spec:

- `scheduling` não precisa chamá-lo para materializar sessão;
- se uma facade de agenda o consultar, usa apenas termos declarativos;
- o retorno não é decisão de reserva/acesso e não inclui saldo confiável;
- `scheduling` nunca decrementa, reserva ou estorna consumo;
- `SPEC-003/D`, `SPEC-007` e `SPEC-008` precisam definir conjuntamente a fonte
  de verdade para contagem antes de ampliar o retorno.

### 11.4 `scheduling -> reservations/attendance/athlete`

```text
GetPublishedSession(organization_id, unit_id, session_id)
  -> SessionSnapshotV1

ListPublishedSessions(organization_id, unit_id, from, to, filters)
  -> []SessionSnapshotV1
```

`SessionSnapshotV1` contém:

- `session_id`, `revision`, `organization_id`, `unit_id`;
- `target_kind`, `target_id`, `target_name`;
- `space_id`, `space_name`;
- `professional_id`, `professional_name`;
- `starts_at`, `ends_at`, `timezone`, dados locais;
- `delivery_mode`, `capacity`, `status`, `origin`.

Não contém pessoa, plano, preço, matrícula, `booking_required`, ocupação,
vagas restantes, reserva ou presença.

### 11.5 Eventos de mudança

```text
ListSessionEvents(organization_id, unit_id, after_sequence, limit)
  -> events[], next_sequence
```

- contrato `SessionEventV1` é versionado e sem PII;
- reservations/attendance processam `event_id` idempotentemente;
- todo comando futuro desses módulos revalida a revisão/estado atual da sessão
  antes de criar efeito;
- cancelamento da sessão bloqueia imediatamente nova reserva/check-in pela
  leitura atual, mesmo que um consumidor ainda reconcilie efeitos anteriores;
- política de cancelar reservas existentes, repor crédito ou notificar aluno
  pertence a `SPEC-007/008/014`.

### 11.6 `reservations -> scheduling` e composição de leitura

`reservations` será dono de ocupação e poderá publicar futuramente:

```text
GetSessionReservationSummary(organization_id, unit_id, session_ids)
  -> confirmed_count, waitlist_count
```

Esse port não entra na persistência de `scheduling`. Uma facade de API pode
compor agenda + resumo quando `SPEC-007` existir. Até lá, a resposta de agenda
expõe apenas capacidade total.

### 11.7 `attendance` e `athlete`

- `attendance` resolve a sessão publicada, depois combina reserva, matrícula e
  política de acesso em seus próprios casos de uso;
- `athlete` lista sessões pelo port de scheduling e, quando existirem, compõe
  elegibilidade/reserva por ports de memberships/reservations;
- nenhum desses consumidores lê tabelas de agenda;
- `scheduling` não consulta conta do atleta, resultado de treino ou presença.

## 12. Contratos HTTP propostos

Todas as rotas ficam sob `/api/v1/units/:unit_id/scheduling`, exigem capability
`scheduling`, sessão tenant e CSRF em mutações.

### 12.1 Profissionais

- `GET /professionals?include_archived=false` — owner/coach;
- `POST /professionals` — owner + `Idempotency-Key`;
- `PATCH /professionals/:professional_id` — owner, payload parcial e
  `expected_revision`;
- `POST /professionals/:professional_id/archive` — owner;
- `POST /professionals/:professional_id/reactivate` — owner.

Criação mínima:

```json
{
  "display_name": "Marina",
  "linked_user_id": null
}
```

### 12.2 Séries

- `GET /series?status=&from=&to=` — owner; coach não lê rascunho;
- `GET /series/:series_id` — owner; coach somente se publicada;
- `POST /series/preview` — owner, preview sem persistência;
- `POST /series` — owner, cria rascunho idempotente;
- `PATCH /series/:series_id/draft` — owner + revisão;
- `POST /series/:series_id/publish` — owner + idempotência/revisão;
- `POST /series/:series_id/changes/preview` — owner;
- `POST /series/:series_id/changes` — owner + idempotência/revisão;
- `POST /series/:series_id/cancel-preview` — owner;
- `POST /series/:series_id/cancel` — owner + idempotência/revisão.

Payload de recorrência usa IDs, nunca nomes livres como referência:

```json
{
  "target": { "kind": "modality", "id": "uuid" },
  "space_id": "uuid",
  "professional_id": "uuid",
  "delivery_mode": "group",
  "capacity": 20,
  "duration_minutes": 60,
  "weekdays": [1, 3, 5],
  "local_start_time": "18:30",
  "timezone": "America/Sao_Paulo",
  "effective_from": "2026-09-01",
  "effective_until": null
}
```

### 12.3 Sessões

- `GET /sessions?from=<instant>&to=<instant>&professional_id=&target_kind=&target_id=&status=`
  — owner/coach, intervalo máximo definido pela slice;
- `GET /sessions/:session_id` — owner/coach;
- `POST /sessions` — owner, cria sessão extra idempotente;
- `POST /sessions/:session_id/change` — owner + idempotência/revisão;
- `POST /sessions/:session_id/cancel` — owner + idempotência/revisão.

Não existe `DELETE`. Listagem não aceita `person_id`, `plan_id` ou status
financeiro.

### 12.4 Erros estáveis

| HTTP | Código | Situação |
|---:|---|---|
| 400 | `invalid_request` | JSON, UUID, data ou hora malformada |
| 404 | `unit_not_found` | unidade ausente ou fora do escopo |
| 404 | `scheduling_professional_not_found` | profissional inexistente/outro tenant |
| 404 | `schedule_series_not_found` | série inexistente/outro tenant |
| 404 | `schedule_session_not_found` | sessão inexistente/outro tenant |
| 409 | `scheduling_revision_conflict` | edição com revisão antiga |
| 409 | `scheduling_conflict` | sobreposição de profissional ou espaço |
| 409 | `scheduling_professional_in_use` | arquivamento com agenda futura |
| 409 | `schedule_series_already_published` | publicação/transição inválida |
| 409 | `schedule_session_already_started` | tentativa de mudar passado/em andamento |
| 409 | `schedule_session_cancelled` | mutação incompatível com cancelamento |
| 409 | `idempotency_key_reused` | mesma chave com outro fingerprint |
| 409 | `scheduling_downstream_impact` | consumer futuro bloqueia alteração |
| 422 | `scheduling_input_invalid` | combinação de recorrência/capacidade/duração inválida |
| 422 | `scheduling_target_unavailable` | modalidade/serviço inválido ou inativo |
| 422 | `scheduling_space_unavailable` | espaço inválido ou inativo |
| 422 | `scheduling_professional_unavailable` | profissional arquivado/não vinculado à unidade |
| 422 | `invalid_local_time` | horário local inexistente ou ambíguo |
| 422 | `scheduling_range_too_large` | preview/listagem/comando excede limite |

Erros incluem `request_id`; conflito pode retornar IDs, recurso e intervalos
necessários para correção, sem SQL, constraint, PII ou dados de outro tenant.

## 13. Auditoria, privacidade e segurança

### 13.1 Auditoria

- criar, ligar/desligar conta, arquivar e reativar profissional são auditados;
- criar/editar/publicar/alterar/cancelar série é auditado;
- criar/alterar/cancelar sessão é auditado;
- publicação em lote produz um evento de auditoria da série e eventos públicos
  por sessão;
- motivo livre não entra em logs/métricas; fica no evento durável com acesso
  operacional apropriado;
- falha de auditoria ou event log reverte a mutação.

### 13.2 Privacidade

- o único dado pessoal novo obrigatório é `professional.display_name`;
- não armazenar telefone, e-mail, CPF, endereço, salário, comissão, laudo,
  especialidade clínica ou observação médica;
- snapshots de nome permitem agenda histórica, mas devem participar de export
  e eventual anonimização do profissional;
- anonimização futura exige primeiro não haver atribuição futura: o comando
  desvincula conta e troca nome/snapshots históricos por valor neutro, sem
  apagar horário/auditoria;
- auditoria pública por sessão usa IDs e não repete nome;
- retenção específica do nome histórico precisa de validação jurídica/produto
  antes de `ready`; não inferir a política de `people` ou de contratos;
- não incluir agenda de profissional no privacy export de aluno apenas porque
  ele futuramente reservou uma sessão. Cada módulo exporta sua parte via port.

### 13.3 Segurança

- backend deriva ator e tenant da sessão validada;
- coach permanece leitura mesmo quando é o profissional responsável;
- referências de outro tenant retornam `404`;
- limites de intervalo, tamanho do lote e quantidade de dias impedem expansão
  de recorrência não controlada;
- idempotência, revisão e locks impedem duplo toque e corrida;
- nenhum endpoint desta spec é público/anônimo.

## 14. Observabilidade e operação

### 14.1 Logs

Eventos estruturados mínimos:

- `scheduling_command_completed|failed` com comando, resultado, IDs, revisão,
  unidade, duração e `request_id`;
- `scheduling_materialization_completed|failed` com janela, contagens,
  `error_code` e duração;
- `scheduling_conflict_detected` com tipo de recurso e intervalos, sem nomes.

Não registrar payload completo, motivo livre, nome do profissional, nome do
aluno futuro, dados clínicos ou resposta bruta do PostgreSQL.

### 14.2 Métricas

- `engagefit.scheduling.commands{command,result}`;
- `engagefit.scheduling.conflicts{resource_type}`;
- `engagefit.scheduling.sessions.materialized{result}`;
- `engagefit.scheduling.materialization.duration`;
- `engagefit.scheduling.materialization.lag`;
- `engagefit.scheduling.series.blocked`.

IDs, nomes, request IDs, séries e sessões não viram labels.

### 14.3 Alertas e diagnóstico

- alerta quando o lag se aproxima da borda do horizonte;
- alerta após execuções consecutivas falhas ou série bloqueada;
- painel operacional mostra última materialização e quantidade agregada de
  séries com atenção;
- suporte usa `request_id`, `run_id`, `series_id` e `session_id`, nunca pede
  planilha, token ou PII;
- comando de reconciliação dry-run lista apenas contagens e IDs e pode
  rematerializar natural keys ausentes sem alterar overrides/cancelamentos.

## 15. Feature flag, rollout e compatibilidade

### 15.1 Capability

Adicionar `FEATURE_SCHEDULING_ENABLED`, exposta como `scheduling`:

- default `false` em development e production até homologação da slice;
- exige `FEATURE_CATALOG_ENABLED=true`;
- serviços só aparecem quando `service_catalog=true`; modalidade continua
  possível sem serviço;
- frontend trata campo ausente como `false`;
- rotas bloqueiam no backend quando desligada.

`SCHEDULING_MATERIALIZER_ENABLED` controla apenas a extensão do horizonte:

- publicação nova é bloqueada se o materializador não estiver operacional no
  ambiente, salvo modo explícito de teste;
- leitura continua disponível para diagnóstico;
- desligar materializador não cancela nem apaga sessões já publicadas.

### 15.2 Ordem segura

1. criar specs executáveis das slices e obter revisão de integração;
2. aplicar migration aditiva, sem seed/backfill e sem escolher número antes do
   merge;
3. publicar backend com capability desligada;
4. publicar frontend que tolera capability ausente;
5. validar ports com adapters fake/no-op para módulos ainda ausentes;
6. habilitar local/homologação, cadastrar dados fictícios e publicar grade;
7. confirmar duas extensões do horizonte, idempotência e conflitos;
8. homologar owner/coach em mobile;
9. ativação de produção depende de decisão separada, backup/restore e
   observabilidade.

### 15.3 Rollback

- desligar `scheduling` impede novas mutações/leitura de produto;
- desligar materializador interrompe extensão, sem alterar sessões;
- schema e dados aditivos permanecem;
- frontend/backend podem voltar em qualquer ordem;
- nenhum check-in, matrícula, plano, contrato ou fluxo legado passa a depender
  de agenda durante esta spec.

### 15.4 Compatibilidade futura

- ao habilitar `reservations`, startup deve exigir o adapter real para eventos
  e impacto de sessão definido pela `SPEC-007`;
- ao habilitar `attendance`, check-in/presença revalida sessão atual e não usa
  snapshot antigo como autorização;
- não ativar `SPEC-003/D` com consumo definitivo antes de `SPEC-007/008`
  escolherem contagem, reserva, presença e estorno.

## 16. Critérios de aceite e matriz de testes

### 16.1 Critérios de produto

- owner cadastra profissional com apenas nome em `360x800`;
- profissional pode existir sem conta e pode ser ligado a owner/coach elegível
  sem ganhar permissão;
- owner cria série semanal usando modalidade ou serviço e espaço já existentes;
- atendimento individual força capacidade 1; grupo aceita capacidade
  explícita;
- preview mostra próximas datas, timezone e conflitos;
- publicação cria sessões concretas e atômicas dentro do horizonte;
- coach lê agenda publicada, mas não rascunho, preço ou ações de mutação;
- owner altera somente uma sessão sem modificar as demais;
- owner altera esta e próximas com preview e preserva overrides;
- owner cria sessão extra e cancela sessão/série com histórico;
- uma sessão cancelada não some e não participa de conflito;
- agenda nunca mostra vagas restantes antes da `SPEC-007`;
- nenhum fluxo cria reserva, presença, cobrança, contrato, consumo ou mensagem.

### 16.2 Critérios de arquitetura e dados

- scheduling não importa/consulta tabela ou repository de catalog,
  memberships, people ou contracts;
- ports públicos possuem testes de contrato e tipos sem GORM/HTTP;
- IDs adversários de outra organização/unidade não vazam existência;
- corrida de publicação/criação preserva um único conjunto e detecta conflito;
- natural key impede duplicação por materializador/retry/múltiplas réplicas;
- override e cancelamento sobrevivem a novas materializações;
- publicação/mudança/auditoria/evento/idempotência são atômicos;
- sessão mantém snapshot após renome/arquivamento de catálogo/profissional;
- evento `SessionEventV1` é versionado, ordenável e sem PII;
- migration é aditiva, vazia em upgrade e não possui número fixado pela spec.

### 16.3 Matriz mínima

| Camada | Cenários obrigatórios |
|---|---|
| domínio | recorrência semanal, intervalo semiaberto, capacidade, individual/grupo, estados efetivos, timezone/DST |
| aplicação | preview/publicação, mudança em lote, override, extra, cancelamento, replay, revisão e rollback |
| PostgreSQL | constraints, FKs internas, unique occurrence, locks concorrentes, tenant e auditoria/eventos atômicos |
| materializador | primeira janela, extensão, duas réplicas, retry, falha parcial entre séries, override/cancelamento preservados |
| ports | fakes de catalog/access/memberships, SessionSnapshotV1, cursor/evento e ausência de DTO/GORM estrangeiro |
| HTTP | owner/coach/admin, CSRF, capability, range limit, erros estáveis, IDs adversários e request_id |
| frontend | profissionais, grade, preview, agenda, mudança/cancelamento, loading/empty/error/conflict em `360x800` |
| acessibilidade | foco, Escape, labels, toque, teclado de data/hora e estado sem depender de cor |
| privacidade | logs sem nomes/PII, export/anonymization do profissional e snapshots minimizados |
| regressão | catálogo, planos, matrículas, check-ins, retenção, workout e app do atleta inalterados |

Comandos futuros seguem o baseline dos repositórios e incluem PostgreSQL real,
race detector no ambiente suportado, migrations em banco vazio/upgrade,
TypeScript, build e Playwright focado/completo.

## 17. Decisões abertas para validar com o Alados

As propostas abaixo mantêm a direção técnica, mas impedem a spec-guia de virar
`ready` sem revisão:

| Decisão | Proposta inicial | Impacta |
|---|---|---|
| termo na operação | `Agenda`, `Grade`, `Aula` e `Atendimento` | UX de todas as slices |
| profissional sem conta | permitido; somente nome | slice A |
| vínculo de conta | uma conta por profissional/organização; owner ou coach | slice A |
| visibilidade do coach | ler toda agenda da unidade + filtro `Minha agenda` | slices A/B |
| espaço obrigatório | sim, inclusive atendimento individual | slice B |
| responsável | exatamente um profissional | slice B |
| individual | capacidade obrigatória 1 | slice B |
| recorrência | semanal; dois horários exigem duas séries | slice B |
| horizonte | 56 dias, configurável 28–180 | slices B/C e operação |
| série sem fim | permitida com materialização limitada | slice B |
| duração máxima | até 24h sem cruzar mais de um dia local | slice B |
| conflito | bloqueio absoluto, sem override | slices B/C |
| feriado | nenhuma regra automática; owner cria exceção/cancelamento | slice C |
| cancelamento | final; reposição cria sessão extra | slice C |
| alteração em lote | preserva overrides e muda só ocorrências futuras | slice C |
| catálogo arquivado | sessões existentes continuam pelo snapshot; novos comandos bloqueiam | slices B/C |
| exibição ao aluno | adiada para SPEC-010 após reservations | contrato público |
| retenção do nome profissional | definir prazo/base antes de anonimização definitiva | slice A/privacidade |

Perguntas concretas para a validação:

1. Pilates individual representa horários disponíveis de capacidade 1 ou
   atendimentos já combinados fora do sistema?
2. Recovery precisa aparecer como modalidade, serviço ou ambos conforme a
   oferta, sem o sistema inferir a escolha?
3. Existe aula sem espaço físico cadastrado ou com dois espaços simultâneos?
4. Um horário precisa de dois coaches/assistentes no primeiro recorte?
5. O coach pode ver toda a grade ou apenas sessões pelas quais é responsável?
6. O Alados publica grade com quantas semanas de antecedência?
7. Há recorrência quinzenal, mensal ou mudança sazonal indispensável?
8. Em feriados, o padrão deve manter, cancelar ou pedir revisão da ocorrência?
9. O owner precisa reverter cancelamento ou criar uma reposição atende melhor
   a auditoria operacional?
10. Quais motivos de cancelamento devem ser estruturados além do texto curto?

## 18. Fatiamento recomendado

As slices são sequenciais e verticais. Nenhuma está autorizada enquanto não
existir seu documento executável `ready`.

### `SPEC-006/A` — Fundação de profissionais

Entrega:

- domínio/tabelas/ports de profissional e vínculo de unidade/conta;
- CRUD sem exclusão, auditoria, privacy e capability;
- UI `Agenda > Profissionais` mobile;
- leitura mínima por coach e filtro futuro `mine` preparado;
- nenhuma série ou sessão.

Gate para `ready`: decisões de nome, conta, visibilidade e retenção aceitas;
contrato de access revisado; nenhum campo clínico/comissão.

### `SPEC-006/B` — Grade recorrente em rascunho e preview

Entrega:

- série semanal, dias, horário, duração, capacidade, alvo, espaço e responsável;
- resolução por port de catalog, rascunho e preview determinístico;
- conflito contra agenda publicada existente;
- UI `Grade`, revisão e estados mobile;
- ainda não publica nem persiste sessões.

Valor vertical: owner prepara e revisa sua grade real sem expor sessão
incompleta. Gate: recorrência, espaço, responsável, capacidade, timezone,
horizonte e conflito validados.

### `SPEC-006/C` — Publicação, materialização e agenda

Entrega:

- sessões concretas, publicação atômica, idempotência e eventos V1;
- materializador diário, run/reconciliação e observabilidade;
- leitura `Hoje/Semana` por owner/coach e ports de consumidores;
- sessão extra;
- cancelamento mínimo e auditável de sessão futura ou série a partir de uma
  data, com motivo, para nunca deixar o owner preso a uma publicação incorreta;
- sem reagendar ou editar configuração de sessão/série publicada.

Gate: contratos SessionSnapshot/Event revisados por integradores de
reservations/attendance/athlete; horizonte e operação homologados.

### `SPEC-006/D` — Exceções e alterações futuras

Entrega:

- override de uma sessão e mudança `esta e próximas`;
- preview de impacto, preservação de IDs/revisões e eventos;
- integração do cancelamento já existente com os contratos de impacto de
  módulos que venham a ser habilitados;
- adapters neutros/contratos de impacto para módulos ainda desligados;
- UX de conflito, confirmação e histórico;
- matriz de compatibilidade com `SPEC-007/008` antes de habilitá-las.

Gate: política de feriado, cancelamento/reposição e responsabilidades entre
reservations/attendance/notifications revisadas.

### Ordem e paralelização

`A` pode ser detalhada primeiro. `B` depende de `A`; `C` depende de `B`; `D`
depende de `C`. Frontend e backend de uma mesma slice permanecem com o mesmo
owner de módulo. Não paralelizar migrations ou contratos dessas slices entre
agentes sem integrador, pois todos alteram `scheduling`.

## 19. Dependências explícitas e riscos

- `SPEC-003/D`: não pode considerar agenda como contador. Receberá consumo de
  reservas/presenças somente depois de fonte de verdade conjunta.
- `SPEC-007`: será dona de reserva, ocupação, espera, no-show, reposição e
  efeitos de cancelamento sobre vaga/crédito.
- `SPEC-008`: será dona de presença/check-in e decisão final de entrada,
  consumindo sessão e reserva por contrato.
- `SPEC-010`: poderá expor agenda e reservas no app depois de contratos aceitos;
  SPEC-006 não altera hoje o PWA.
- `SPEC-012`: conectores externos adaptarão agenda/reserva sem escrever tabelas
  de scheduling diretamente.
- `SPEC-004/A` em detalhamento paralelo não é premissa. Esta spec não consulta
  contrato/aceite nem supõe mudanças ainda não aprovadas.

Riscos principais:

- materializar pouco e deixar a agenda sem datas futuras;
- materializar demais e tornar mudanças caras;
- conflito concorrente sem lock de recurso;
- usar snapshot desatualizado como autorização;
- tratar capacidade como ocupação;
- deixar cancelamento de sessão decidir estorno/consumo de outro módulo;
- coletar PII desnecessária de profissional;
- criar dependência circular por repository/tabela compartilhada.

## 20. Proibições para o futuro agente implementador

O agente executor deve:

- implementar somente a slice explicitamente autorizada e em estado `ready`;
- trabalhar em worktree/branch própria e declarar ownership do módulo;
- atualizar a slice antes de qualquer divergência de contrato;
- usar somente ports públicos estreitos para módulos colaboradores;
- entregar commits separados de backend/frontend quando ambos mudarem;
- preservar mudanças alheias e os arquivos
  `.ai/acordo-fundadores-engagefit.md` e
  `docs/proposta-engagefit-guilherme.html`.

O agente executor não deve:

- implementar diretamente a partir desta spec-guia em `draft`;
- escolher ou reservar número de migration;
- consultar tabelas/repositories/models GORM de `catalog`, `memberships`,
  `people` ou `contracts`;
- adicionar `plan_id`, `enrollment_id`, `person_id`, preço, saldo, reserva,
  presença ou vagas restantes às tabelas de agenda;
- decidir elegibilidade pessoal, consumir direito ou alterar contador;
- implementar `SPEC-003/D`, `SPEC-007`, `SPEC-008` ou parte não autorizada de
  `SPEC-010/012/014`;
- alterar rotas legadas de check-in, workout ou athlete;
- inventar multiunidade, RRULE genérica, calendário externo, vídeo,
  comissão, folha ou prontuário;
- criar, migrar ou mesclar profissionais a `Person` por nome/telefone;
- editar migration publicada, usar banco compartilhado para testes
  destrutivos ou tocar produção;
- fazer push, merge, deploy ou habilitar feature flag.

## 21. Gates para promover a spec

Antes de `reviewed`:

- decisões da seção 17 discutidas com o Alados;
- revisão de arquitetura dos ports e do ownership de eventos;
- revisão de privacy para nome/snapshot do profissional;
- confirmação de que `SPEC-004/A` não introduziu dependência relevante.

Antes de qualquer slice `ready`:

- documento executável próprio com payloads/constraints finais;
- migration sem número final e plano de upgrade/rollback;
- limites de lote/horizonte fechados;
- testes concorrentes e de tenancy definidos com fixtures de duas organizações;
- contrato público revisado pelos donos dos módulos consumidores;
- prompt do executor restringindo explicitamente as demais slices.

Esta versão permanece `draft` até revisão de integração; criar o arquivo não
autoriza implementação, migration, push, deploy ou feature flag.
