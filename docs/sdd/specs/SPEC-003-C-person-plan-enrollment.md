# SPEC-003/C — Matrícula da pessoa em plano publicado

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-08-19
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-002/A`, `SPEC-003/A`
- Ownership principal: módulo `memberships`
- Contrato colaborador: módulo `people`, somente pela porta pública descrita
  nesta spec
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration esperada: `058_create_person_plan_enrollments.sql` (número
  provisório; o integrador confirma ou renumera)

## 1. Resultado esperado

O owner abre um aluno no celular, toca em `Matrícula`, escolhe um plano ativo
e publicado, revisa preço e termos, define a data de início e confirma. O
sistema registra uma matrícula operacional auditável para a `Person` daquele
aluno e preserva a versão exata do plano apresentada na revisão.

O painel mostra plano, vigência e estado atual. O owner também consegue
encerrar a matrícula com data e motivo. Nenhuma dessas ações gera contrato,
cobrança, pagamento, reserva, consumo, notificação ou autorização de acesso.

Esta slice fecha um ciclo pequeno e utilizável:

```text
aluno existente -> revisar plano -> matricular -> consultar -> encerrar
```

## 2. Problema e evidências

- **Existente:** `SPEC-002/A` criou `Person`, vínculo com unidade e
  `students.person_id`, mantendo `Student` como compatibilidade operacional.
- **Existente:** `SPEC-003/A` criou catálogo por unidade, versão publicada
  imutável, preço de referência e direitos sobre modalidades/serviços.
- **Existente:** `students.membership_started_at` alimenta retenção e não
  representa plano, termos, vigência ou histórico comercial. Esta slice não
  transforma esse campo na fonte de verdade de matrícula.
- **Benchmark — Tecnofit:** a venda de contrato parte do cadastro do aluno,
  permite escolher o contrato e ajustar o início antes de finalizar.
- **Benchmark — Next Fit:** a venda pode partir do perfil e permite início
  atual ou futuro; o produto concorrente também mistura ativação, financeiro,
  contrato e horários no mesmo fluxo.
- **Inferência EngageFit:** manter a descoberta `aluno -> plano`, mas chamar a
  ação de `Matricular` enquanto contrato e cobrança não existem. Isso evita
  comunicar uma venda financeira que o sistema ainda não executou.
- **Inferência EngageFit:** uma pessoa terá no máximo uma matrícula operacional
  aberta por unidade nesta primeira versão. Planos complementares serão
  representados por adicionais na `SPEC-003/B`, evitando duplicidade acidental.
- **Entrada Alados:** o primeiro passo precisa associar pessoas a planos de
  CrossFit, Pilates, Spinning e corrida preservando preço, frequência e
  duração escolhidos.

Referências oficiais consultadas:

- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000699531-como-efetuar-vendas-no-cadastro-do-aluno-contrato-servico-e-produto-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000554060
- https://ajuda.nextfit.com.br/support/solutions/articles/69000554077-como-cadastrar-um-aluno-cliente-geral-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000553628-como-criar-e-vender-um-contrato-padr%C3%A3o-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000554111-como-matricular-o-aluno-em-hor%C3%A1rios-fixos-

## 3. Escopo

### Incluído

- resolver um aluno elegível para a `Person` canônica pelo contrato público de
  `people`;
- listar o estado e o histórico de matrícula desse aluno na unidade atual;
- matricular em uma versão ainda publicada de plano ativo da unidade atual;
- data de início passada, atual ou futura dentro dos limites desta spec;
- cálculo determinístico de validade para planos de pagamento único;
- preservação da versão, preço e resumo de termos apresentados;
- no máximo uma matrícula aberta por pessoa/unidade;
- encerramento manual com motivo, data, revisão otimista e auditoria;
- reconciliação transacional de matrícula naturalmente expirada ao criar a
  próxima;
- idempotência dos comandos de criação e encerramento;
- export de privacidade com histórico comercial sem duplicar PII;
- capability `enrollments`, desligada por padrão;
- painel mobile-first acessado pela lista atual de alunos.

### Fora do escopo

- adicionais, desconto, cupom, cortesia ou alteração de preço (`SPEC-003/B`);
- documento contratual, assinatura, PAR-Q, fidelidade, multa, cancelamento
  jurídico, pausa ou trancamento (`SPEC-004`);
- cobrança, recorrência executada, parcelas, pagamento ou inadimplência
  (`SPEC-005`);
- grade, horário fixo, profissional, turma, vaga ou sessão (`SPEC-006`);
- reserva, saldo consumido, estorno ou lista de espera (`SPEC-007`);
- bloquear/liberar check-in e calcular direito de acesso (`SPEC-003/D` e
  `SPEC-008`);
- matrícula pelo app do aluno, venda online ou envio de mensagem;
- migração em massa dos mensalistas existentes;
- contexto multiunidade na mesma sessão (`SPEC-001/C`);
- Wellhub, TotalPass e qualquer efeito externo.

## 4. Vocabulário e estados

- **Pessoa (`Person`):** identidade humana canônica que recebe a matrícula.
- **Aluno (`Student`):** localizador de compatibilidade usado pela tela atual;
  não é a identidade persistida em `enrollments`.
- **Matrícula aberta:** registro com `lifecycle_status=active`, ainda não
  encerrado administrativamente.
- **Validade natural:** intervalo inclusivo entre `starts_on` e
  `valid_until`. Só planos `one_time` possuem `valid_until` nesta slice.
- **Estado efetivo:** projeção calculada na leitura; não é um segundo status
  mutável no banco.

Estados persistidos:

- `active`: matrícula aberta, mesmo quando o início ainda é futuro ou a
  validade natural já terminou;
- `ended`: encerrada por comando explícito ou reconciliada ao criar uma nova
  matrícula depois da expiração.

Estados efetivos retornados pela API, nesta ordem de precedência:

1. `ended` quando `lifecycle_status=ended`;
2. `scheduled` quando está aberta e `as_of < starts_on`;
3. `expired` quando está aberta, possui `valid_until` e
   `as_of > valid_until`;
4. `active` nos demais casos.

`active` significa somente adesão operacional vigente. Não significa
`contrato assinado`, `pago`, `adimplente` ou `acesso autorizado`.

## 5. Regras e invariantes

1. Toda matrícula pertence a uma organização, uma `Person`, uma unidade
   principal e exatamente uma versão de plano.
2. Organização e unidade são derivadas da sessão autenticada; IDs do payload
   nunca definem tenant.
3. O `unit_id` da rota deve ser a unidade autenticada. Alvo diferente retorna
   `404` para não revelar existência.
4. O `student_id` da rota é apenas localizador. O backend resolve uma pessoa
   ativa, não anonimizada, com vínculo ativo na mesma organização/unidade.
5. Matrícula persiste `person_id`, nunca CPF, telefone, e-mail, nome livre ou
   somente `student_id`.
6. O plano precisa estar `active`, pertencer à unidade atual e possuir a
   versão informada ainda em `published` no momento do commit.
7. O cliente envia `plan_id` e `plan_version_id`. Se uma nova versão foi
   publicada depois da revisão, a criação falha e exige nova revisão; o
   servidor não troca silenciosamente os termos.
8. Modalidades e serviços referenciados pelos direitos precisam continuar
   ativos na unidade para uma nova matrícula. Desativação posterior não altera
   a matrícula existente, mas bloqueia novas adesões até correção do plano ou
   catálogo.
9. Preço deve vir da versão imutável. Não há campo de preço, desconto ou total
   editável no comando.
10. Planos recorrentes (`monthly`, `quarterly`, `semiannual`, `annual`) não
    recebem fim artificial: `valid_until=NULL`. O ciclo é termo comercial; a
    execução da recorrência pertence a `memberbilling`.
11. Plano `one_time` calcula `valid_until` no servidor, de forma inclusiva, a
    partir da validade da versão.
12. Uma pessoa pode ter histórico ilimitado, mas no máximo uma matrícula
    aberta por unidade nesta slice.
13. Uma matrícula aberta com estado efetivo `scheduled` ou `active` impede
    outra criação e retorna conflito.
14. Ao criar uma nova matrícula quando a aberta está efetivamente `expired`, a
    mesma transação encerra a anterior com evento
    `enrollment.expiration_reconciled` e então cria a nova. A reconciliação usa
    `ended_on=valid_until`, `ended_at` igual ao instante do comando, motivo
    determinístico `Validade natural encerrada` e o owner do comando como ator.
15. Não há worker de mudança diária de status. O estado efetivo é calculado
    pelo relógio injetado; reconciliação persistida ocorre somente em comando.
16. Matrícula nunca é apagada por endpoint, arquivamento do plano ou nova
    versão. FKs usam `RESTRICT`.
17. Encerramento não altera a versão do plano nem cria cobrança, estorno,
    notificação ou bloqueio de check-in.
18. `membership_started_at` não é atualizado nesta slice. A matrícula torna-se
    fonte comercial; retenção continua usando o campo legado até uma spec de
    compatibilidade explícita.
19. Mensalistas existentes não recebem matrícula, preço ou vigência inferidos.
20. Matrículas não alteram `Student.source`, vínculo do app do atleta,
    check-ins ou regras atuais de acesso.

## 6. Datas e validade

Todas as datas HTTP usam `YYYY-MM-DD`. A referência inicial é
`America/Sao_Paulo`, já usada pelos fluxos operacionais atuais, até que unidade
tenha timezone próprio.

### Início

- UI sugere a data local atual;
- aceita no máximo 10 anos no passado para correção operacional;
- aceita no máximo 365 dias no futuro;
- fora desse intervalo retorna `invalid_start_date` sem gravar dados.

### Validade de pagamento único

O intervalo é inclusivo. O cálculo aplica a unidade em calendário e subtrai um
dia do limite exclusivo:

```text
1 day, início 19/08 -> valid_until 19/08
2 weeks, início 19/08 -> valid_until 01/09
1 month, início 19/08 -> valid_until 18/09
1 year, início 19/08/2026 -> valid_until 18/08/2027
```

Para meses e anos, usar aritmética de calendário, com testes para fim de mês e
ano bissexto. Não converter mês em 30 dias nem ano em 365 dias.

### Encerramento

- formulário sugere a data local atual;
- para matrícula já iniciada, `ended_on` deve ficar entre `starts_on` e hoje;
- uma matrícula futura pode ser encerrada antes de começar; nesse caso
  `ended_on` é hoje e o histórico preserva `starts_on` futuro;
- motivo normalizado é obrigatório, entre 3 e 240 caracteres;
- `valid_until` original não é sobrescrito; `ended_on` registra a interrupção
  administrativa separadamente.

## 7. Fluxos de produto

### Consultar e matricular

1. Owner abre `Alunos` no celular.
2. No card de uma pessoa não anonimizada, toca em `Matrícula`.
3. Um painel inferior carrega estado e histórico somente daquele aluno.
4. Sem matrícula aberta, o painel mostra `Matricular em um plano`.
5. Owner escolhe entre planos ativos com versão publicada, vendo nome, resumo
   e preço de referência.
6. Informa início, com hoje preenchido.
7. Revisão mostra aluno, plano, versão, preço, início e fim quando houver.
8. A revisão informa: `Esta ação não gera cobrança nem contrato.`
9. Owner toca uma vez em `Confirmar matrícula`.
10. Sucesso troca o painel para o resumo da matrícula, sem recarregar a página.

### Matrícula já aberta

- painel mostra badge efetiva `Agendada`, `Ativa` ou `Expirada`;
- mostra plano, preço de referência, resumo, início e validade;
- `Encerrar matrícula` é ação secundária destrutiva, protegida por confirmação;
- matrícula expirada oferece `Matricular novamente`; a reconciliação da
  anterior ocorre no backend junto da nova criação.

### Encerrar

1. Owner toca em `Encerrar matrícula`.
2. Informa data e motivo.
3. Confirma vendo que contrato/pagamento não serão alterados.
4. O painel mostra `Encerrada`, data, motivo e mantém o registro no histórico.

### Sem planos disponíveis

Mostrar empty state curto: `Publique um plano antes de matricular` e link para
`#/plans`. Rascunhos, arquivados e planos sem versão publicada não aparecem na
seleção. Plano publicado com aviso de modalidade/serviço desativado aparece
como indisponível, com orientação para corrigir o catálogo ou publicar nova
versão; não pode ser confirmado.

## 8. Experiência mobile-first

- não criar item novo no menu principal;
- integrar a ação na página atual `Alunos`;
- no card mobile, `Matrícula` fica junto das ações operacionais e abre painel
  de largura total; ações de privacidade continuam em abertura secundária;
- no desktop, adicionar coluna/ação compacta sem remover filtros existentes;
- o painel deve caber em `360x800`, permitir rolagem interna e não gerar
  overflow horizontal;
- somente uma ação primária por etapa;
- seleção de plano usa cards/radio legíveis, não um select com textos
  truncados;
- estados loading, vazio, erro recuperável, conflito e sucesso são explícitos;
- preservar foco, `aria-modal`, rótulos, fechamento por botão e Escape;
- erro de versão alterada retorna à revisão com termos recarregados;
- duplo toque e retry de rede reutilizam a mesma chave idempotente;
- alunos anonimizados não exibem a ação;
- coach não recebe ação nem preço no frontend.

Não carregar uma matrícula por aluno durante a listagem. O frontend só chama o
endpoint do aluno quando o painel é aberto, evitando N+1 e mudanças no contrato
de `GET /students`.

## 9. Modelo de domínio

Adicionar ao domínio `memberships`:

```text
Enrollment
  ID
  OrganizationID
  PersonID
  HomeUnitID
  PlanVersionID
  LifecycleStatus: active | ended
  StartsOn
  ValidUntil?
  EndedOn?
  EndedAt?
  EndedReason?
  Revision
  CreatedByUserID
  EndedByUserID?
  CreatedAt
  UpdatedAt
```

A projeção de leitura inclui a identidade imutável da versão:

```text
EnrollmentTerms
  PlanID
  PlanVersionID
  VersionNumber
  Name
  Description
  PriceCents
  Currency
  BillingCycle
  Summary
```

Não duplicar esses termos na tabela de matrícula. A FK para a versão imutável é
o snapshot. A resposta é montada por join/projeção dentro de `memberships`.

## 10. Contrato público com `people`

`memberships` não pode consultar diretamente `students`, `people` ou
`person_unit_relationships`. Criar uma porta pequena, consumida pela aplicação
de matrículas e implementada pelo adapter/repositório de `people`:

```text
ResolveEnrollmentSubject(ctx, organization_id, unit_id, student_id)
  -> person_id
  -> display_name
  -> eligible
  -> ineligible_reason
```

O resolver deve, na mesma organização/unidade:

- localizar `Student` não anonimizado com `person_id` preenchido;
- localizar `Person` ativa;
- confirmar vínculo ativo da pessoa com a unidade;
- não retornar CPF, telefone, e-mail ou endereço;
- mapear ausência/inaptidão para erro estável sem vazar outro tenant.

Executar a resolução dentro do contexto transacional nos comandos. Testes de
arquitetura/repositório devem demonstrar que o repositório de matrículas não
possui joins ou queries diretas nas tabelas de `people`.

## 11. Persistência e migration

### `enrollments`

Campos mínimos:

- `id UUID PRIMARY KEY`;
- `organization_id UUID NOT NULL`;
- `person_id UUID NOT NULL`;
- `home_unit_id UUID NOT NULL`;
- `plan_version_id UUID NOT NULL`;
- `lifecycle_status VARCHAR(16) NOT NULL`;
- `starts_on DATE NOT NULL`;
- `valid_until DATE NULL`;
- `ended_on DATE NULL`;
- `ended_at TIMESTAMPTZ NULL`;
- `ended_reason VARCHAR(240) NULL`;
- `revision INTEGER NOT NULL DEFAULT 1`;
- `created_by_user_id UUID NOT NULL`;
- `ended_by_user_id UUID NULL`;
- timestamps.

Constraints/FKs:

- FK composta `(person_id, organization_id)` para `people`;
- FK composta `(home_unit_id, organization_id)` para `boxes`;
- FK composta `(plan_version_id, organization_id)` para
  `membership_plan_versions`;
- FKs de atores para `users`, todas `ON DELETE RESTRICT`;
- status somente `active|ended`;
- revisão positiva;
- campos de encerramento todos nulos quando `active`;
- `ended_at`, `ended_on`, `ended_reason` e `ended_by_user_id` todos preenchidos
  quando `ended`;
- `valid_until IS NULL OR valid_until >= starts_on`;
- índice parcial único `(organization_id, person_id, home_unit_id)` onde
  `lifecycle_status='active'`;
- índices de leitura por pessoa/unidade e por versão.

### `enrollment_command_requests`

- escopo por organização, comando e `Idempotency-Key`;
- comandos permitidos `create|end`;
- fingerprint SHA-256 do payload canônico incluindo unidade e alvo;
- `enrollment_id`, resposta estável em JSONB e timestamp;
- chave entre 8 e 128 caracteres com a mesma política da `SPEC-003/A`.

### `enrollment_events`

- organização, unidade, pessoa, matrícula, ator, ação e timestamp;
- `before_data` e `after_data` JSONB sem PII;
- `reason` normalizado quando aplicável;
- ações iniciais:
  - `enrollment.created`;
  - `enrollment.ended`;
  - `enrollment.expiration_reconciled`.

Eventos e command request pertencem à mesma transação da mutação. Rollback de
qualquer parte reverte tudo.

### Compatibilidade da migration

- migration somente aditiva e sem backfill;
- não alterar dados em `students`, `people`, check-ins ou tabelas de billing;
- tabelas novas ficam vazias ao atualizar um banco real da `057`;
- não modificar migrations publicadas `001..057`;
- número `058` é provisório até a integração.

## 12. Serviço de aplicação e concorrência

### Criar matrícula

Em uma única transação:

1. derivar organização pela unidade autenticada;
2. validar chave e procurar replay idempotente;
3. resolver a pessoa elegível pela porta de `people`;
4. adquirir lock transacional estável para
   `(organization, person, unit)`;
5. carregar matrícula aberta, se houver;
6. se efetivamente expirada, encerrá-la e auditar reconciliação; se agendada ou
   ativa, retornar `active_enrollment_exists`;
7. carregar/lockar plano e versão informados;
8. confirmar plano ativo, unidade, versão ainda publicada e alvos de catálogo
   ativos;
9. calcular `valid_until` a partir da versão;
10. criar matrícula, evento e command request;
11. retornar os termos efetivamente persistidos.

O lock de aplicação não substitui o índice único. Corridas simultâneas devem
produzir uma matrícula e um conflito/replay previsível, nunca duas matrículas.

### Encerrar matrícula

Em uma única transação:

1. validar escopo, idempotência, data, motivo e `expected_revision`;
2. lockar a matrícula por organização/unidade;
3. rejeitar revisão divergente ou matrícula já encerrada por chave diferente;
4. preencher campos de encerramento e incrementar revisão;
5. criar evento e command request;
6. retornar a matrícula encerrada.

Relógio e timezone devem ser injetáveis nos testes. Não usar `time.Now()`
espalhado em domínio, repositório e handler.

## 13. Contratos HTTP

Todos os endpoints são autenticados, limitados a owner e protegidos pela
capability `enrollments`.

### Consultar por aluno

```http
GET /api/v1/units/:unit_id/students/:student_id/enrollments
```

Resposta `200`:

```json
{
  "current": null,
  "history": []
}
```

`current` contém a matrícula aberta, inclusive quando efetivamente agendada ou
expirada. `history` contém encerradas em ordem decrescente.

### Criar

```http
POST /api/v1/units/:unit_id/students/:student_id/enrollments
Idempotency-Key: enrollment.<uuid>
```

```json
{
  "plan_id": "uuid",
  "plan_version_id": "uuid",
  "starts_on": "2026-08-19"
}
```

Retorna `201` na criação e `200` em replay idempotente. A resposta contém:

- IDs da matrícula, pessoa, plano e versão;
- número da versão e termos de leitura;
- `lifecycle_status`, `effective_status`, `starts_on`, `valid_until`;
- campos de encerramento quando existirem;
- revisão e timestamps.

### Consultar por ID

```http
GET /api/v1/units/:unit_id/enrollments/:enrollment_id
```

### Encerrar

```http
POST /api/v1/units/:unit_id/enrollments/:enrollment_id/end
Idempotency-Key: enrollment-end.<uuid>
```

```json
{
  "ended_on": "2026-08-19",
  "reason": "Solicitação do aluno",
  "expected_revision": 1
}
```

Retorna `200`, incluindo replay.

### Erros estáveis

| HTTP | `code` | Situação |
|---|---|---|
| 400 | `invalid_request` | JSON, UUID ou data malformada |
| 404 | `enrollment_subject_not_found` | aluno/pessoa/vínculo inexistente ou fora do escopo |
| 404 | `enrollment_not_found` | matrícula inexistente ou de outro escopo |
| 409 | `active_enrollment_exists` | já existe matrícula aberta não expirada |
| 409 | `plan_version_changed` | versão revisada deixou de ser a publicada |
| 409 | `enrollment_revision_conflict` | encerramento com revisão antiga |
| 409 | `idempotency_key_reused` | mesma chave com fingerprint diferente |
| 409 | `enrollment_already_ended` | novo comando sobre matrícula encerrada |
| 422 | `enrollment_subject_ineligible` | pessoa inativa/anonimizada ou vínculo inativo |
| 422 | `plan_unavailable` | plano arquivado, sem publicação ou de outra unidade |
| 422 | `plan_targets_inactive` | modalidade/serviço não pode ser vendido agora |
| 422 | `invalid_start_date` | início fora da janela permitida |
| 422 | `invalid_end` | data ou motivo de encerramento inválido |

Capability desligada retorna o envelope atual de `feature_disabled`. Coach e
platform admin recebem a política atual de autorização sem detalhes de preço.

## 14. Capability, autorização e rollout

Adicionar:

```text
FEATURE_ENROLLMENTS_ENABLED=false
capabilities.enrollments=false
```

Regras de configuração:

- `FEATURE_ENROLLMENTS_ENABLED=true` exige
  `FEATURE_MEMBERSHIPS_ENABLED=true`;
- exige também `FEATURE_PEOPLE_FOUNDATION_ENABLED=true` e suas chaves válidas;
- nasce `false` em todos os ambientes, inclusive desenvolvimento;
- frontend trata capability ausente como `false` para permitir deploy em
  qualquer ordem;
- middleware cobre todas as rotas `/units/.../enrollments` e
  `/units/.../students/.../enrollments`;
- owner consulta preço e gerencia; coach e platform admin não acessam esta UI
  ou API.

Ordem segura:

1. aplicar migration com flag desligada;
2. publicar backend capaz de responder `enrollments=false`;
3. publicar frontend que tolera campo ausente;
4. ativar primeiro em ambiente/local controlado;
5. homologar criar, conflito, expiração e encerramento;
6. ativar por configuração somente após aceite.

Rollback operacional é desligar `FEATURE_ENROLLMENTS_ENABLED`. Não remover
tabelas nem dados. Esta execução não autoriza push, deploy ou ativação de flag.

## 15. Privacidade, segurança e auditoria

- matrícula armazena somente IDs, datas e termos comerciais; não duplica PII;
- logs não incluem nome, CPF, telefone, e-mail, motivo livre ou payload JSON;
- logs estruturados usam ação, organização, unidade, ator, matrícula, plano,
  versão, resultado e duração;
- export de privacidade do aluno passa a incluir suas matrículas e eventos
  comerciais pelo contrato público de leitura de `memberships`, sem query
  direta do módulo de estudantes nas tabelas novas;
- anonimização não apaga matrícula nem versão por exigência de auditoria;
  como a `Person` é anonimizada in-place, a FK continua válida sem PII;
- aluno anonimizado não pode abrir nova matrícula nem consultar o painel pelo
  localizador de aluno;
- anonimização não encerra automaticamente uma matrícula, pois isso seria uma
  decisão comercial fora do pedido de privacidade; o registro deixa de conter
  identidade direta e permanece auditável;
- motivo de encerramento é dado operacional minimizado e não aparece em logs;
- todas as queries de matrícula filtram organização e unidade antes do ID;
- respostas adversárias de outro tenant são indistinguíveis de inexistência.

Para enriquecer o export, compor uma pequena porta de leitura de
`memberships` no caso de uso de privacidade. Não introduzir dependência do
repositório de privacidade sobre tabelas de matrícula.

## 16. Observabilidade

Métricas sem labels de alta cardinalidade:

- `engagefit.enrollments.commands{command,result}`;
- `engagefit.enrollments.command.duration{command}`;
- `engagefit.enrollments.conflicts{reason}`;
- `engagefit.enrollments.expiration_reconciled`.

Logs de comando devem distinguir criação, replay, conflito, encerramento e
rollback. Falha ao gravar auditoria ou idempotência deve abortar a transação.

Não criar dashboard, alerta externo ou job nesta slice. Apenas expor sinais no
padrão de observabilidade já usado em `memberships`.

## 17. Compatibilidade e limites arquiteturais

- `memberships` continua um módulo do monólito modular;
- `people -> memberships` não ganha dependência de domínio; a colaboração usa
  portas pequenas no composition root;
- repositório de matrícula não lê tabelas de `people` diretamente;
- `billing_plans` e módulo `billing` não são importados nem alterados;
- `memberbilling` não é criado nesta slice;
- handlers não contêm regras de validade, tenant ou transição;
- frontend não calcula `valid_until` como fonte de verdade;
- nenhuma mutation atual de student/check-in é interceptada;
- não atualizar `docs/system-design.md` com comportamento futuro durante a
  implementação. Atualizá-lo somente depois de código e validação concluídos.

## 18. Critérios de aceite

### Produto

- owner encontra `Matrícula` no card/linha do aluno sem item novo no menu;
- uma pessoa válida pode ser matriculada em plano ativo/publicado da unidade;
- revisão explicita que não há cobrança nem contrato;
- plano mensal de CrossFit gera matrícula sem `valid_until`;
- pacote de 10 aulas com validade de 30 dias preserva a versão e calcula fim
  inclusivo;
- versão 1 permanece na matrícula depois da publicação da versão 2;
- tentativa com versão revisada não matricula silenciosamente na nova versão;
- matrícula futura aparece como `Agendada`;
- matrícula naturalmente vencida aparece como `Expirada`;
- nova matrícula após expiração encerra e audita a anterior na mesma transação;
- owner encerra matrícula com data/motivo e mantém histórico;
- aluno anonimizado não pode ser matriculado;
- nenhuma cobrança, contrato, mensagem, reserva ou check-in é criado/alterado.

### Segurança e dados

- organização/unidade/pessoa/plano adversários não vazam existência;
- corrida de duas criações deixa exatamente uma matrícula aberta;
- replay igual retorna o mesmo ID; chave reutilizada não executa novamente;
- auditoria e command request sofrem rollback junto da matrícula;
- plano arquivado ou alvo de catálogo inativo bloqueia nova matrícula;
- plano/versão arquivado ou substituído depois da matrícula não altera o
  histórico;
- migration não cria matrículas para dados legados;
- privacy export contém histórico e nenhum novo campo duplica PII;
- capability e autorização bloqueiam backend mesmo com chamada manual.

### Experiência

- fluxo principal funciona em Chromium `360x800` sem overflow horizontal;
- loading, vazio, erro, conflito, sucesso, agendada, ativa, expirada e encerrada
  possuem estados visuais testados;
- foco e teclado funcionam no painel;
- duplo toque não cria duplicidade;
- frontend com backend antigo esconde a ação por capability ausente.

## 19. Testes obrigatórios

### Domínio e aplicação

- matriz de cálculo de dias, semanas, meses, anos, fim de mês e ano bissexto;
- projeção de `effective_status` com relógio determinístico;
- validação de datas e motivo;
- criação recorrente e one-time;
- plano/unidade/versão/target inválidos;
- versão trocada entre revisão e confirmação;
- matrícula aberta, agendada, expirada e encerrada;
- reconciliação de expirada + nova criação com rollback integral;
- idempotência igual/diferente em create e end;
- revisão otimista e transições inválidas;
- concorrência real deixando uma única matrícula aberta;
- `membership_started_at` e check-ins inalterados.

### PostgreSQL e HTTP

- migration `001..058` em PostgreSQL vazio;
- segunda execução aplica zero migrations;
- upgrade real `057 -> 058` com pessoas, vínculos, alunos e planos publicados,
  comprovando zero backfill;
- constraints, FKs compostas, índice parcial e rollback de eventos;
- isolamento de organização e unidade com referências adversárias;
- teste que inspeciona/garante ausência de query cruzada no repositório de
  matrícula;
- owner permitido; coach e platform admin bloqueados;
- capability, CSRF, cookies/bearer e envelopes de erro;
- privacy export e anonimização com matrícula existente;
- `go test ./...`, `go vet ./...` e `go test -race -count=1 ./...` em ambiente
  Linux com PostgreSQL real.

### Frontend

- capability ausente/falsa esconde ação;
- aluno anonimizado sem ação;
- empty state sem plano e link para `#/plans`;
- seleção/revisão/criação com preço BRL e datas;
- visualização agendada, ativa e expirada;
- conflito de matrícula e versão alterada recuperáveis;
- encerramento exige data, motivo e confirmação;
- replay/duplo toque não duplica;
- Playwright focado em `360x800` e regressão completa;
- TypeScript, build Vite e `git diff --check`.

## 20. Não objetivos e proibições para o agente executor

- não implementar `SPEC-003/B`, `SPEC-003/D` ou qualquer parte de `004..008`;
- não adicionar addons, pausa, parcelas, gateway, webhook ou assinatura;
- não mudar o acesso/check-in legado com base na matrícula;
- não inventar matrícula para mensalista existente;
- não alterar migration publicada;
- não tocar no banco local compartilhado `boxengage`;
- não usar dados reais do Alados;
- não fazer push, deploy, migration de produção ou habilitar feature flag;
- não acessar workspace de outra spec;
- não introduzir dependência direta em tabelas de outro módulo.

## 21. Definition of Done

- implementação vertical backend/frontend em commits separados;
- nenhuma divergência silenciosa desta spec;
- migration provisória declarada ao integrador;
- todos os testes da seção 19 executados ou impedimentos registrados;
- revisão explícita de tenancy, transação, idempotência, concorrência,
  privacidade e rollout;
- revisão visual mobile e ausência de overflow;
- worktrees limpos;
- documentação/status alterados para `validated locally` e evidências anexadas
  somente após validação;
- `docs/system-design.md` atualizado somente após implementação validada;
- nenhum push, merge, deploy, banco compartilhado ou flag externa alterados.

## 22. Decisões deliberadamente adiadas

- múltiplas matrículas abertas e composição por adicionais: `SPEC-003/B`;
- pausa/trancamento e deslocamento de vigência: `SPEC-004`;
- cancelamento jurídico e reflexo financeiro: `SPEC-004/005`;
- renovação e cobrança recorrente: `SPEC-005`;
- consumo de sessões/pacotes e decisão de acesso: `SPEC-003/D/007/008`;
- atualização da retenção a partir da matrícula: spec de compatibilidade após
  homologação operacional;
- exibição no app do aluno: `SPEC-010`;
- importação dos contratos/matrículas do Tecnofit: `SPEC-011`.

Essas decisões não impedem esta slice porque matrícula preserva pessoa, unidade
e versão imutável sem afirmar estados dos módulos futuros.

## 23. Evidências de validação local

Validação concluída em 2026-08-19, sem divergência do contrato desta spec:

- PostgreSQL temporário exclusivo aplicou `001..058` em banco vazio; a segunda
  execução aplicou zero migrations;
- teste de upgrade real `057 -> 058` com pessoa, vínculo, aluno e plano
  publicado confirmou zero backfill nas três tabelas novas;
- constraints, FKs compostas, índice parcial, rollback de evento, isolamento,
  idempotência, revisão, expiração reconciliada, versão substituída,
  arquivamento, anonimização e corrida real passaram;
- `go test -count=1 ./...` e `go vet ./...` passaram com PostgreSQL real;
- `go test -race -count=1 ./...` passou em contêiner Linux com CGO e PostgreSQL
  real, pois o host não possui compilador C;
- TypeScript e build Vite passaram;
- Playwright focado passou 4 cenários de matrícula, incluindo `360x800`, foco,
  Escape, duplo toque, conflitos, estados e encerramento;
- regressão Playwright passou 29 testes; 2 fluxos de API real permaneceram
  ignorados por serem opt-in;
- revisão visual em `360x800` confirmou painel sem overflow horizontal;
- o banco compartilhado `boxengage`, produção e feature flags não foram
  alterados. A migration `058` permanece provisória para o integrador.
