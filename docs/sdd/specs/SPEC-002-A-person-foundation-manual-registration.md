# SPEC-002/A — Fundação de pessoa e cadastro manual compatível

- Status: `validated locally`
- Versão: `1.1`
- Data: 2026-08-18
- Depende de: `ARCH-001`, `SPEC-001/A-B`
- Ownership: módulo novo `people`, com compatibilidade explícita com
  `students` e `privacy`
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration: número provisório escolhido no worktree e confirmado/renumerado
  pelo integrador no merge

## 1. Resultado esperado

O owner cadastra um novo aluno em poucos segundos pelo celular informando nome,
WhatsApp e CPF. E-mail e endereço são opcionais em `Mais informações`. A mesma
transação cria a pessoa, seu vínculo com a unidade e um `Student` local
compatível, portanto o aluno aparece imediatamente na base atual e continua
funcionando em convite, check-in, retenção e campanhas.

Esta slice introduz `Person` sem substituir nem reinterpretar silenciosamente
os dados atuais. Ela prepara a fundação para contratos, planos e financeiro.

## 2. Decisões e evidências

- **Existente:** `Student` é uma projeção por box/origem usada por check-ins,
  retenção, campanhas, privacidade e app do aluno.
- **Existente:** mesma pessoa em Wellhub e TotalPass pode ter dois `Student`s;
  merges automáticos entre origens foram evitados de propósito.
- **Benchmark — Next Fit:** cadastro do cliente antecede a venda do plano; a
  documentação recomenda e-mail/CPF para app e dados adicionais quando o
  pagamento online precisar deles.
- **Inferência EngageFit:** cadastro manual do mensalista exige apenas nome,
  WhatsApp e CPF; endereço e e-mail não bloqueiam a entrada operacional.
- **Inferência EngageFit:** possuir WhatsApp não concede consentimento. O novo
  `Student` nasce com `contact_status=unknown`.
- **Inferência EngageFit:** cada `Student` atual recebe uma pessoa própria no
  backfill. Nenhum nome, telefone ou conta do atleta provoca consolidação nesta
  slice.
- **Inferência EngageFit:** o fluxo novo faz dual-write apenas na criação. As
  leituras operacionais continuam em `students` até uma slice posterior
  entregar perfil unificado e migração gradual de consultas.

Referências de benchmark:

- https://ajuda.nextfit.com.br/support/solutions/articles/69000554077-como-cadastrar-um-aluno-cliente-geral-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000554985-como-cadastrar-ou-remover-o-cart%C3%A3o-do-cliente-

## 3. Escopo

### Incluído

- tabelas-base de pessoa, endereço e vínculo com unidade;
- cadastro manual de mensalista com idempotência;
- validação de CPF e telefone;
- CPF cifrado e lookup por HMAC dedicado;
- detecção de CPF exato e telefone potencialmente duplicado;
- criação transacional de `Person`, vínculo e `Student box_member`;
- backfill um-para-um de todos os `students` atuais;
- `students.person_id` opcional e exposto de forma aditiva;
- integração das novas tabelas ao privacy export e à anonimização existentes;
- auditoria sem PII;
- botão/formulário `Novo aluno` na tela atual de Alunos;
- capability independente e rollout desligado por padrão.

### Fora do escopo

- edição e perfil unificado da pessoa após o cadastro;
- consolidação de dois `Student`s na mesma pessoa;
- identidade externa Wellhub/TotalPass;
- vínculo com `AthleteAccount`;
- vínculo de pessoa existente a uma segunda unidade;
- merge, split ou reversão de merge;
- CRM, prospect, responsável, dependente ou menor sem CPF;
- contrato, PAR-Q, plano, matrícula, cobrança ou venda;
- contato opt-in automático;
- remoção ou obrigatoriedade de `students`;
- busca global por CPF via URL.

Esses itens ficam nas próximas slices da `SPEC-002` ou nas specs consumidoras.

## 4. Regras e invariantes

1. `Person` representa um ser humano no escopo de uma organização.
2. Cadastro manual exige nome, telefone e CPF válidos; registros de backfill
   podem não possuir telefone/CPF.
3. Nome visível remove whitespace externo e reduz sequências internas a um
   espaço; possui de 2 a 160 caracteres.
4. CPF aceita máscara na entrada, é reduzido a 11 dígitos e validado pelos dois
   dígitos verificadores. Sequências com todos os dígitos iguais são inválidas.
5. CPF válido é único por organização enquanto a pessoa não estiver
   anonimizada.
6. CPF não aparece em URL, log, métrica, auditoria, lista ou resposta comum.
7. `cpf_encrypted` usa cifra autenticada; `cpf_lookup_hash` usa HMAC-SHA-256
   com chave dedicada. SHA-256 simples de CPF é proibido.
8. Telefone é normalizado para E.164 brasileiro (`+55...`) em `persons`. A
   projeção legada `students.phone` recebe somente os dígitos (`55...`) para
   preservar as comparações atuais de self check-in e mensageria.
9. Telefone não é único: famílias podem compartilhar um número. Correspondência
   exata exige aviso e decisão explícita, nunca merge automático.
10. E-mail é opcional, trimado e normalizado em lowercase; quando presente
    precisa de sintaxe válida e no máximo 254 caracteres.
11. Endereço é opcional e não participa de deduplicação.
12. Criar pessoa, endereço opcional, vínculo, `Student`, auditoria e registro
    de idempotência é uma única transação.
13. O `Student` criado usa `source=box_member`,
    `external_id=person:<person_id>`, `contact_status=unknown`, risco ativo e
    início de vínculo/membership na data atual.
14. O telefone informado não produz evento de consentimento nem libera envios.
15. Toda pessoa criada manualmente possui exatamente um vínculo ativo com a
    unidade atual e um `Student box_member` ligado ao mesmo `person_id`.
16. Cada `Student` existente recebe uma pessoa exclusiva; o backfill não
    consolida por nome, telefone, e-mail, origem ou conta.
17. IDs de organização/unidade vêm da sessão. Nesta slice, `unit_id` da rota
    precisa ser igual ao `box_id` autenticado.
18. UUID de outro tenant retorna `404` sem revelar existência.
19. `PLATFORM_ADMIN` não recebe PII operacional.
20. Retry com a mesma chave e mesmo payload retorna o resultado original;
    mesma chave com payload diferente é conflito.

## 5. Modelo de dados

### `persons`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório, FK `organizations`, restrict |
| `name` | varchar(160) obrigatório |
| `phone_e164` | varchar(16) nullable |
| `email` | varchar(254) nullable |
| `cpf_encrypted` | text nullable |
| `cpf_lookup_hash` | varchar(80) nullable |
| `cpf_last2` | char(2) nullable, somente para máscara |
| `profile_status` | `active`, `archived`, `anonymized` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints/índices:

- unique `(id, organization_id)`;
- unique parcial `(organization_id, cpf_lookup_hash)` quando hash não é null;
- índice `(organization_id, lower(name))`;
- índice `(organization_id, phone_e164)`;
- checks de campos pareados: CPF cifrado, hash e últimos dígitos são todos
  nulos ou todos preenchidos;
- status anonimizado exige CPF/hash/últimos dígitos nulos.

Associated data da cifra: `person-cpf:<organization_id>:<person_id>`.
Resposta comum deriva apenas `cpf_masked` como `***.***.***-XX`; descriptografar
CPF fica restrito ao privacy export autorizado.

### `person_addresses`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `person_id`, `organization_id` | FKs compostas para `persons` |
| `address_type` | inicialmente `home` |
| `postal_code` | varchar(8), somente dígitos, nullable |
| `street` | varchar(160) nullable |
| `number` | varchar(30) nullable |
| `complement` | varchar(100) nullable |
| `district` | varchar(100) nullable |
| `city` | varchar(100) nullable |
| `state` | char(2) nullable, uppercase |
| `country` | char(2), default `BR` |
| `created_by_user_id`, `updated_by_user_id` | UUIDs obrigatórios para cadastro manual |
| `created_at`, `updated_at` | timestamptz obrigatórios |

- unique `(person_id, address_type)` nesta versão;
- nenhum endereço vazio é criado;
- campos parcialmente informados são aceitos, mas CEP e UF, se presentes,
  precisam ser válidos estruturalmente;
- endereço nunca aparece na lista atual de alunos.

### `person_unit_relationships`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id`, `person_id`, `unit_id` | tenancy composta |
| `relationship_status` | `active`, `inactive`, `blocked` |
| `started_at` | timestamptz obrigatório |
| `ended_at` | timestamptz nullable |
| `status_reason` | varchar(280), default vazio |
| `created_at`, `updated_at` | timestamptz obrigatórios |

- unique `(person_id, unit_id)`;
- FKs compostas garantem que pessoa e unidade sejam da mesma organização;
- o status não representa contrato, pagamento nem autorização de entrada.

### Compatibilidade com `students`

Adicionar `students.person_id UUID NULL REFERENCES persons(id) ON DELETE SET
NULL` e índice por `person_id`. O domínio, model e DTO de `Student` recebem
`PersonID`/`person_id` opcionais sem remover campos existentes.

Backfill determinístico:

1. para cada `student`, criar `Person` com `persons.id = students.id` e
   organização obtida pelo `box`;
2. copiar nome/e-mail e telefone somente quando normalizáveis; CPF fica vazio;
3. pessoa de aluno já anonimizado nasce `anonymized`; demais nascem `active`;
4. criar um vínculo com `relationship.id = student.id`, unidade do aluno e
   início em `membership_started_at`, ou `student.created_at` como fallback;
5. preencher `students.person_id = students.id`;
6. não criar endereços nem eventos de auditoria para o backfill.

A migration deve reconciliar:

- quantidade de `students` sem `person_id` igual a zero imediatamente após o
  backfill inicial;
- uma pessoa e um vínculo por student existente;
- nenhuma alteração em IDs, origem, external ID, check-ins ou consentimento.

`person_id` permanece nullable porque imports e ativações concorrentes ainda
podem usar caminhos legados depois do rollout. Torná-lo obrigatório pertence a
outra slice.

### `person_creation_requests`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id`, `unit_id`, `actor_user_id` | escopo/auditoria |
| `idempotency_key` | varchar(128) |
| `request_fingerprint` | HMAC do payload canônico; nunca CPF/telefone bruto |
| `person_id`, `student_id` | resultado persistido |
| `created_at` | timestamptz obrigatório |

- unique `(organization_id, unit_id, idempotency_key)`;
- chave aceita 8–128 caracteres em `[A-Za-z0-9._:-]`;
- replay do mesmo fingerprint retorna o mesmo resultado;
- fingerprint diferente retorna `409 idempotency_key_reused`.

### `people_audit_events`

Campos: `id`, `organization_id`, `unit_id`, `actor_user_id`, `action`,
`person_id`, `student_id` nullable, `metadata` jsonb e `created_at`.

Eventos mínimos: `person.created`, `person.address_added`,
`person.unit_linked`. Metadata contém apenas status, campos presentes e IDs;
nunca nome, e-mail, telefone, CPF ou endereço.

### Migration e concorrência entre agentes

A branch parte da migration `054`. O executor usa provisoriamente o próximo
número no próprio worktree. Se a `SPEC-001/D` usar o mesmo número, isso é
esperado: o integrador renumera a segunda migration na ordem de merge e roda
os testes de banco vazio, upgrade e idempotência. O agente não coordena números
alterando o outro worktree.

## 6. Proteção de CPF e configuração

Reutilizar o port `SecretCipher` para cifra autenticada, sem acessar chave no
repositório. Criar uma responsabilidade pequena de lookup HMAC no módulo
`people`, alimentada por `PERSON_LOOKUP_HMAC_KEY`, base64 de exatamente 32
bytes. O segredo deve ser estável, separado do JWT e das chaves AES e guardado
com backup; rotação exige uma migration/reindexação futura explícita.

Quando `FEATURE_PEOPLE_FOUNDATION_ENABLED=true`, a configuração falha no
startup se faltar:

- `DATA_ENCRYPTION_ACTIVE_KEY_ID`;
- `DATA_ENCRYPTION_KEYS`;
- `PERSON_LOOKUP_HMAC_KEY`.

O módulo de pessoas nunca usa `PlaintextSecretCipher`, inclusive em
development. Testes injetam chaves próprias e não reutilizam segredo real.

## 7. Contratos HTTP

Rotas sob `/api/v1`, sessão tenant e capability `people`. `unit_id` precisa ser
o `box_id` da sessão. Somente `OWNER` acessa os endpoints desta slice.

### Verificação de duplicidade

`POST /units/:unit_id/people/duplicate-check`

Body:

```json
{
  "cpf": "123.456.789-09",
  "phone": "(11) 99999-9999"
}
```

Ao menos um campo é obrigatório. CPF e telefone ficam no body, nunca em URL.

Resposta `200`:

```json
{
  "cpf_match": null,
  "phone_matches": [
    {
      "person_id": "person-uuid",
      "name": "Maria S.",
      "phone_masked": "+55 11 *****-9999",
      "relationship_status": "active"
    }
  ]
}
```

Resultados pertencem somente à organização e unidade autorizadas. A API limita
phone matches a 5, não retorna CPF, e-mail ou endereço e não busca outra
organização.

### Criar pessoa/aluno

`POST /units/:unit_id/people`

Headers:

- `Idempotency-Key`: obrigatório, 8–128 caracteres seguros.

Body mínimo:

```json
{
  "name": "Maria da Silva",
  "phone": "(11) 99999-9999",
  "cpf": "123.456.789-09"
}
```

Body com abertura progressiva e decisão de telefone:

```json
{
  "name": "Maria da Silva",
  "phone": "(11) 99999-9999",
  "cpf": "123.456.789-09",
  "email": "maria@example.com",
  "address": {
    "postal_code": "01234567",
    "street": "Rua Exemplo",
    "number": "10",
    "complement": "Sala 2",
    "district": "Centro",
    "city": "São Paulo",
    "state": "SP"
  },
  "phone_duplicate_review": {
    "decision": "create_separate",
    "matched_person_id": "existing-person-uuid",
    "reason": "Telefone compartilhado pela família"
  }
}
```

O bloco de revisão só é aceito quando o match informado ainda corresponde ao
telefone normalizado; motivo possui de 5 a 280 caracteres. CPF igual nunca
pode ser sobrescrito pela revisão.

Resposta `201` para criação e replay idêntico:

```json
{
  "person_id": "person-uuid",
  "student_id": "student-uuid",
  "name": "Maria da Silva",
  "phone": "+5511999999999",
  "email": "maria@example.com",
  "cpf_masked": "***.***.***-09",
  "relationship_status": "active",
  "contact_status": "unknown",
  "created_at": "2026-08-18T20:00:00Z"
}
```

Replay inclui `Idempotency-Replayed: true`; primeira execução pode omitir o
header ou enviá-lo como `false`.

### Compatibilidade de `Student`

`GET /students` e `GET /students/:id` mantêm o contrato existente e adicionam:

```json
{ "person_id": "person-uuid" }
```

Campo vazio é omitido. Nenhuma rota atual troca `student_id` por `person_id`.

### Erros estáveis

| HTTP | Código | Situação |
|---|---|---|
| 400 | `invalid_people_input` | nome, telefone, e-mail, endereço ou JSON inválido |
| 400 | `person_cpf_invalid` | CPF ausente ou inválido |
| 400 | `invalid_idempotency_key` | header ausente/fora do padrão |
| 404 | `unit_not_found` | unidade ausente ou fora da sessão |
| 409 | `person_cpf_conflict` | CPF já pertence a pessoa da organização |
| 409 | `person_possible_duplicate` | telefone igual sem revisão válida |
| 409 | `idempotency_key_reused` | mesma chave com payload diferente |

Conflitos nunca ecoam CPF/telefone bruto. Constraint errors são traduzidos e
uma corrida de dois requests com o mesmo CPF produz exatamente uma pessoa.

## 8. Privacidade, autorização e auditoria

- `OWNER` executa cadastro e duplicate check; `COACH` continua usando apenas a
  lista operacional atual, sem CPF/endereço;
- feature gate deve reconhecer `/api/v1/units/:id/people...` sem bloquear
  outras rotas de unidade;
- handlers não registram body e mensagens de erro não interpolam PII;
- logs estruturados usam request/actor/tenant e resultado, nunca campos
  pessoais;
- métricas não usam IDs ou PII como labels;
- privacy export de `/students/:id/privacy-export` passa a incluir a pessoa e
  endereço ligados. O CPF completo pode aparecer nesse arquivo autorizado,
  pois é exportação do titular, mas nunca em resposta comum;
- anonimização de `/students/:id/anonymize` remove nome, telefone, e-mail, CPF,
  lookup hash e endereço da pessoa ligada na mesma transação quando não existe
  outro `Student` não anonimizado ligado a ela;
- se houver outro vínculo ativo inesperado, anonimiza somente o student e
  registra evento de auditoria sem apagar a PII necessária ao outro perfil;
- falha de cifra, auditoria ou privacy update aborta a transação.

## 9. Compatibilidade com fluxos existentes

- o novo cadastro deve aparecer em `GET /students` logo após sucesso;
- convite do app recebe o `student_id` criado e continua igual;
- check-in manual e self check-in usam o `Student box_member` criado;
- retenção lê `membership_started_at` e `membership_started_source=manual`;
- campanhas não podem contatar o aluno até opt-in explícito;
- imports Wellhub/TotalPass, ativações e contas do atleta continuam sem mudança
  funcional;
- caminhos legados que criarem `Student` depois da migration podem deixar
  `person_id` nulo. Deve existir métrica/log agregado para medir esse débito,
  sem falhar o fluxo atual;
- não adicionar joins obrigatórios de `persons` às consultas quentes atuais.

## 10. Feature flag e rollout

Adicionar `FEATURE_PEOPLE_FOUNDATION_ENABLED`:

- default `false` em todos os ambientes até as chaves serem configuradas;
- exposta como `people` em `GET /api/v1/capabilities`;
- campo ausente/falso esconde o botão e bloqueia as rotas com
  `404 capability_disabled`;
- não altera `GET /students` nem os fluxos antigos quando desligada;
- frontend novo funciona com backend antigo porque trata `people` ausente como
  false.

Sequência:

1. configurar e guardar as chaves sem habilitar a feature;
2. aplicar migration e reconciliar contagens do backfill;
3. publicar backend com flag false;
4. publicar frontend em qualquer ordem;
5. em staging, habilitar e executar cadastro, retry, conflito, export e
   anonimização;
6. habilitar no piloto e observar erros/latência/quantidade de students sem
   pessoa.

Rollback operacional: desligar a flag. Dados criados permanecem compatíveis
com `students`; não executar down migration nem apagar pessoas.

## 11. Experiência mobile-first

- na tela `Alunos`, mostrar botão `Novo aluno` somente para owner e
  `capabilities.people === true`;
- em mobile o botão fica visível próximo ao título, não depois da tabela;
- abrir modal/bottom sheet de uma coluna;
- primeira visão contém apenas `Nome`, `WhatsApp` e `CPF`;
- máscaras são visuais; validação final é do backend;
- `Mais informações` revela e-mail e endereço;
- antes do POST, executar duplicate check; CPF existente bloqueia e oferece
  fechar/voltar para a lista, sem criar nada;
- telefone existente mostra candidato mascarado e ações `Cancelar` e
  `Criar outra pessoa`. A segunda exige motivo curto;
- usar chave idempotente nova por intenção de cadastro e preservá-la durante
  retries do mesmo formulário;
- no sucesso, fechar formulário, recarregar a lista atual, destacar o novo
  aluno e mostrar `Aluno cadastrado`;
- não marcar autorização de contato e não exibir checkbox pré-selecionado;
- estados loading, erro, conflito e sucesso não dependem apenas de cor;
- 360x800 não possui scroll horizontal e ações têm alvo mínimo de 44px.

## 12. Critérios de aceite

- owner cadastra aluno com três campos essenciais pelo celular;
- e-mail/endereço opcionais são persistidos quando informados;
- criação produz pessoa, vínculo, Student e auditoria atomicamente;
- aluno aparece na lista e funciona nos fluxos atuais por `student_id`;
- cadastro nasce sem consentimento de contato;
- retry idêntico não cria duplicata;
- mesmo CPF na organização cria uma única pessoa e retorna conflito seguro;
- mesmo telefone alerta e permite cadastro separado com motivo;
- CPF fica cifrado, lookup usa HMAC e nenhuma resposta comum expõe o valor;
- todos os students existentes recebem pessoa própria sem merge;
- aluno importado sem CPF continua funcional;
- tenant adversário não lê nem deduz correspondência por UUID/CPF;
- privacy export e anonimização cobrem pessoa/endereço;
- flag desligada preserva integralmente a experiência atual;
- frontend novo e backend antigo são compatíveis.

## 13. Testes mínimos

### Domínio/aplicação

- CPF válido/inválido, repetido e sequência uniforme;
- normalização de nome, telefone e e-mail;
- HMAC determinístico sem usar hash simples e cifra com associated data;
- criação mínima, com endereço e com rollback em cada falha;
- idempotência igual/diferente e dois requests concorrentes;
- telefone duplicado sem revisão, revisão inválida e criação separada;
- nenhum evento de consentimento na criação.

### Persistência/migration

- banco vazio e upgrade real da `054` com fixtures legadas;
- backfill de homônimos, telefones iguais, origens diferentes e anonimizado;
- uma pessoa por student, sem alterar IDs/check-ins/origens;
- FKs compostas e isolamento entre duas organizações;
- unique de CPF sob concorrência;
- migrator idempotente e número final reconciliado na integração;
- privacy export e anonymization com endereço/CPF.

### HTTP/configuração

- role owner, coach e platform admin;
- unit ID diferente e UUID de outro tenant;
- capability desligada;
- startup falha com feature ligada e qualquer chave ausente;
- body/erros/logs não vazam CPF;
- contratos e códigos estáveis.

### Frontend/regressão

- typecheck e build;
- capability ausente/falsa esconde botão;
- formulário mínimo, abertura progressiva e máscaras;
- conflitos de CPF/telefone, retry e sucesso;
- Playwright 360x800;
- convite, privacidade e lista atual continuam funcionando;
- `go test ./...`, `go vet ./...` e suíte frontend completa.

## 14. Próximas slices da SPEC-002

- `SPEC-002/B`: perfil unificado, edição e migração gradual de leitura;
- `SPEC-002/C`: identidades externas, revisão e consolidação segura;
- vínculo multiunidade depende de `SPEC-001/C`;
- vínculo de `AthleteAccount` e política de merge exigem contrato próprio antes
  de implementação.

## 15. Evidências de validação

- migration provisória `055_create_people_foundation.sql` validada em banco
  PostgreSQL vazio, upgrade real a partir da `054` populada e segundo run com
  zero migrations aplicadas;
- backfill reconciliado um-para-um com homônimos, telefones iguais, origens
  diferentes e student anonimizado, sem alterar origem, external ID ou
  consentimento;
- normalização, CPF válido/inválido, HMAC dedicado, cifra autenticada com
  associated data, idempotência, rollback, concorrência de CPF, isolamento e
  auditoria cobertos por testes;
- privacy export inclui CPF autorizado/endereço e anonimização remove PII e
  endereço na mesma transação quando a pessoa não é compartilhada;
- contratos HTTP cobrem capability, roles, unit adversária, erros estáveis e
  ausência de CPF bruto nas respostas;
- frontend validado com capability ausente, cadastro mínimo, abertura
  progressiva, telefone compartilhado, retry com a mesma chave, sucesso e
  viewport `360x800` sem scroll horizontal;
- `go test ./...`, suíte PostgreSQL completa, `go vet ./...`, race detector em
  Linux/CGO, TypeScript, build Vite, Playwright completo e `git diff --check`
  permanecem como gates antes do merge.
