# SPEC-006/A — Fundação de profissionais da agenda

- Status: `ready`
- Versão: `1.0`
- Data: 2026-08-24
- Depende de: `ARCH-001`, `SPEC-006`
- Ownership principal: módulo novo `scheduling`
- Contrato colaborador: `organizations/access`, somente pelas portas desta spec
- Repositórios futuros envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration: número não reservado; o integrador define no merge

Esta é a única slice executável da `SPEC-006`. Ela não autoriza grade, série,
sessão, materialização, conflito, reserva, presença ou decisão de acesso.

## 1. Resultado esperado

O owner abre `Agenda > Profissionais` no celular, cadastra um profissional
informando somente o nome e, opcionalmente, vincula uma conta de owner ou coach
já ativa na unidade. Pode editar, desvincular, arquivar, reativar, exportar ou
anonimizar o cadastro dentro das regras desta spec.

O coach possui leitura dos profissionais ativos da unidade, sem qualquer ação
de mutação. Criar ou vincular um profissional não cria conta, não muda papel e
não concede permissão.

O ciclo vertical termina aqui:

```text
conta operacional opcional
    -> profissional ativo na unidade
    -> leitura owner/coach
    -> arquivamento, exportação e anonimização auditáveis
```

## 2. Decisões de produto e proveniência

- **Existente:** owner e coach já são papéis de `organizations/access`, com
  `users.box_id` como unidade autorizada nesta fase.
- **Existente:** coach possui leitura operacional em catálogo e retenção, sem
  acesso a configurações comerciais ou administrativas.
- **Inferência EngageFit:** profissional é identidade operacional de
  `scheduling`, não `Person`, `Student`, `User` nem `Coach`.
- **Inferência EngageFit:** nome é o único campo obrigatório. Não entram CPF,
  telefone, e-mail, endereço, conselho, especialidade clínica, comissão ou
  observação médica.
- **Inferência EngageFit:** contas de owner/coach são vínculos opcionais. O
  nome do profissional é persistido independentemente e não acompanha futuras
  alterações da conta.
- **Inferência EngageFit:** homônimos são permitidos e nunca são mesclados.
- **Decisão revisada:** coach lê os profissionais ativos, mas permanece sem
  mutação mesmo quando sua conta estiver vinculada.
- **Decisão revisada:** arquivamento preserva o nome; anonimização explícita
  neutraliza a identidade e não apaga o registro operacional.
- **Validação futura:** medir se o Alados precisa de campos não sensíveis como
  apelido operacional ou especialidade de agenda. Nenhum deles bloqueia esta
  slice.

## 3. Escopo

### Incluído

- módulo `scheduling` com domínio, ports, aplicação e persistência próprios;
- cadastro mínimo de profissional por organização e vínculo com a unidade
  autenticada;
- nome operacional normalizado, homônimos permitidos e revisão otimista;
- vínculo opcional e único com conta ativa de owner/coach na organização;
- listagem estreita de contas elegíveis por port de `organizations/access`;
- leitura dos profissionais ativos por owner e coach;
- edição do nome e vínculo, arquivamento e reativação pelo owner;
- idempotência em todos os comandos e auditoria transacional;
- exportação e anonimização explícita do profissional;
- capability `scheduling`, desligada por padrão;
- UI mobile-first `Agenda > Profissionais` para owner e coach;
- testes de autorização, concorrência, privacy e isolamento entre duas
  organizações/unidades.

### Fora do escopo

- modalidade, serviço, espaço, grade, série, sessão ou calendário;
- horário, duração, capacidade, modo individual/grupo ou conflito;
- disponibilidade, jornada, férias, feriado, remuneração ou comissão;
- especialidade clínica, conselho, prontuário, PAR-Q ou dado de saúde;
- criar, convidar, alterar papel, desativar ou redefinir senha de `User`;
- vincular profissional a `Person` ou tentar encontrá-lo por nome;
- múltiplas unidades acessíveis na mesma sessão;
- reserva, lista de espera, reposição, presença, check-in ou acesso;
- plano, matrícula, contrato, cobrança, saldo ou consumo;
- tela de agenda/grade vazia, série futura ou filtro funcional `Minha agenda`;
- notificação, integração externa ou efeito de rede;
- alteração de contratos compartilhados de `catalog`, `memberships` ou
  `people`.

## 4. Vocabulário e permissões

- **Profissional:** responsável operacional que poderá conduzir sessões em
  slices futuras.
- **Conta vinculada:** referência opcional a owner/coach existente. Não é a
  identidade canônica do profissional.
- **Vínculo de unidade:** autorização para selecionar o profissional naquela
  unidade em uma grade futura; não é permissão de login.
- **Arquivado:** indisponível para novos usos, mas ainda identificado no
  histórico.
- **Anonimizado:** cadastro arquivado, sem conta e com nome neutro irreversível
  pela API desta slice.

| Ação | Owner | Coach | Platform admin |
|---|---:|---:|---:|
| listar profissionais ativos | sim | sim | não |
| incluir arquivados e ver vínculo de conta | sim | não | não |
| listar contas elegíveis | sim | não | não |
| criar, editar, arquivar e reativar | sim | não | não |
| exportar e anonimizar | sim | não | não |

## 5. Regras e invariantes

1. Profissional pertence a exatamente uma organização.
2. Nesta slice, todo profissional nasce com vínculo ativo somente na unidade da
   sessão; a API não aceita outras unidades no body.
3. `organization_id` e `unit_id` são derivados da sessão/rota validada. UUID de
   outro tenant retorna `404`.
4. Nome visível remove whitespace externo e reduz sequências internas a um
   espaço. Deve possuir de 1 a 100 caracteres Unicode após normalização.
5. `display_name_key` aplica lowercase Unicode à forma normalizada apenas para
   busca/ordenação. Não possui constraint única.
6. Dois profissionais podem ter o mesmo nome na mesma unidade ou organização.
   Nome nunca produz merge, sugestão de merge ou vínculo de conta.
7. Conta vinculada é opcional e precisa ser owner ou coach ativo, autorizado na
   unidade atual, confirmado pelo port de access no momento do commit.
8. Uma conta pode estar ligada a no máximo um profissional na organização. A
   defesa existe na aplicação e em índice único parcial.
9. Vincular conta não copia nome/e-mail e não altera papel, `box_id`,
   `auth_version`, sessão ou allowlist.
10. Desativação posterior da conta não arquiva nem renomeia o profissional. A
    leitura owner mostra `Conta indisponível`; novo uso da conta precisa ser
    revalidado em slices futuras.
11. `revision` começa em 1 e toda mutação efetiva a incrementa. Comando recebe
    `expected_revision` quando o recurso já existe.
12. Arquivar marca o profissional e seu vínculo de unidade como inativos. Não
    remove linha nem auditoria.
13. Reativar restaura o vínculo da unidade autenticada, desde que o cadastro
    não esteja anonimizado. Conta vinculada, se houver, é revalidada; conta
    inelegível deve ser desvinculada explicitamente antes da reativação. A
    reativação limpa `archived_at` e `archived_by_user_id`; o evento preserva o
    ciclo anterior.
14. Anonimização exige profissional arquivado, `linked_user_id=NULL` e nenhuma
    dependência futura informada pelo port interno de scheduling. Nesta slice,
    não há séries/sessões, portanto a checagem retorna zero por construção.
15. Anonimizar define `display_name='Profissional anonimizado'`, chave
    equivalente, `anonymized_at`, ator e revisão. Não existe endpoint de
    reversão.
16. Auditoria nunca armazena nome anterior/atual, e-mail ou dados da conta;
    registra somente IDs, campos alterados, estados e presença/ausência de
    vínculo.
17. Todos os comandos exigem `Idempotency-Key`. Replay idêntico retorna a
    resposta original; chave com outro fingerprint retorna conflito.
18. Mutação, vínculo de unidade, command request e auditoria pertencem à mesma
    transação.
19. Nenhuma operação cria série, sessão, reserva, presença, mensagem ou efeito
    externo.

## 6. Modelo de dados e migration

A migration é estritamente aditiva, sem seed, backfill ou número final nesta
spec. Não altera `users`, `persons`, `students`, catálogo, planos, matrículas,
check-ins ou contratos.

### 6.1 `scheduling_professionals`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório |
| `display_name` | varchar(100) obrigatório |
| `display_name_key` | varchar(100) obrigatório, não único |
| `linked_user_id` | UUID opaco nullable, sem FK para tabela de access |
| `status` | `active|archived` |
| `revision` | inteiro positivo |
| `anonymized_at` | timestamptz nullable |
| `created_by_user_id`, `updated_by_user_id` | atores opacos obrigatórios |
| `archived_at`, `archived_by_user_id` | preenchidos em conjunto quando arquivado por comando |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints/índices mínimos:

- unique `(id, organization_id)`;
- unique parcial `(organization_id, linked_user_id)` quando não nulo;
- checks de nome, status, revisão e campos condicionais de anonimização;
- índice `(organization_id, status, display_name_key, id)`;
- FK `organization_id -> organizations(id) ON DELETE RESTRICT`; nenhuma query
  de scheduling consulta a tabela de organizações diretamente.

Anonimização usa o mesmo nome neutro para múltiplos registros; por isso nome e
chave deliberadamente não são únicos.

### 6.2 `scheduling_professional_units`

| Campo | Regra |
|---|---|
| `organization_id`, `unit_id`, `professional_id` | UUIDs obrigatórios |
| `active` | boolean obrigatório |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- PK `(unit_id, professional_id)`;
- FK composta `(professional_id, organization_id)` para
  `scheduling_professionals`;
- FK composta `(unit_id, organization_id)` para `boxes` usando a constraint já
  existente;
- nenhuma criação para outra unidade nesta slice.

### 6.3 `scheduling_command_requests`

Preparada para evolução do módulo, mas a migration desta slice aceita somente:

- `professional_create`;
- `professional_update`;
- `professional_archive`;
- `professional_reactivate`;
- `professional_anonymize`.

Campos: organização, unidade, comando, chave, fingerprint SHA-256 do payload
canônico, `professional_id`, resposta estável JSONB e timestamp. A unicidade é
`(organization_id, command, idempotency_key)`. Chaves seguem a política de
`memberships`: 8–128 caracteres em `[A-Za-z0-9._:-]`.

### 6.4 `scheduling_audit_events`

Campos mínimos: ID, organização, unidade, ator, ação, `entity_type`,
`entity_id`, revisão, `changed_fields`, `before_data`, `after_data`, motivo
normalizado opcional e timestamp.

Eventos desta slice:

- `scheduling.professional.created`;
- `scheduling.professional.updated`;
- `scheduling.professional.archived`;
- `scheduling.professional.reactivated`;
- `scheduling.professional.anonymized`.

`before_data`/`after_data` aceitam apenas `status`, `revision`,
`unit_assignment_active`, `linked_account_present` e `anonymized`. Nome e
`linked_user_id` não entram no JSON durável.

## 7. Contratos entre módulos

`scheduling` não importa repository, model GORM, DTO HTTP ou tabela de
`organizations/access`. O composition root injeta portas estreitas.

### 7.1 Contexto autorizado

```text
ResolveSchedulingUnitContext(authenticated_user_id, unit_id)
  -> organization_id, unit_id, unit_name, role: owner|coach, authorized
```

- deriva organização e unidade sem confiar no body;
- nesta fase autoriza somente `unit_id == users.box_id`;
- não retorna senha, sessão, e-mail ou objeto GORM.

### 7.2 Contas elegíveis

```text
ListProfessionalAccountCandidates(organization_id, unit_id)
  -> []{ user_id, display_name, role: owner|coach, active }

ResolveProfessionalAccount(organization_id, unit_id, user_id)
  -> { user_id, role: owner|coach, active, authorized_in_unit }
```

- listagem é usada somente pelo owner na abertura opcional `Vincular conta`;
- o comando revalida pelo resolver dentro da transação de aplicação;
- `display_name` da conta é apenas rótulo de seleção e não é persistido em
  scheduling nem copiado para o profissional;
- nenhuma das portas cria ou altera usuário.

### 7.3 Leitura futura por conta

```text
ResolveProfessionalForUser(organization_id, unit_id, user_id)
  -> professional_id?
```

É uma porta pública de `scheduling`, preparada para o filtro `Minha agenda`.
Nesta slice não existem sessões e o frontend não exibe o filtro.

Não há colaboração com `catalog`, `memberships`, `people`, `contracts`,
`reservations`, `attendance` ou `athlete` nesta slice.

## 8. Serviço de aplicação e concorrência

### Criar

Em uma transação:

1. resolver contexto e exigir owner;
2. validar chave/fingerprint e procurar replay;
3. normalizar nome;
4. se houver conta, resolver elegibilidade e adquirir advisory lock por
   `(organization, linked_user_id)`;
5. criar profissional, vínculo ativo da unidade, auditoria e command request;
6. retornar DTO persistido.

Corrida com a mesma conta deixa um vencedor e um
`scheduling_professional_account_conflict`, nunca dois vínculos.

### Atualizar

- valida `expected_revision`, nome e conta opcional;
- `linked_user_id: null` desvincula a conta;
- conta nova é revalidada e lockada antes do commit;
- substitui apenas campos enviados e incrementa revisão uma vez;
- recurso anonimizado não aceita atualização.

### Arquivar/reativar

- comandos exigem revisão esperada e motivo de 3–240 caracteres;
- arquivar desativa o vínculo da unidade e preserva conta/nome;
- reativar valida conta vinculada e restaura o vínculo;
- replay não incrementa revisão novamente.

### Anonimizar

- exige revisão, motivo de 5–240 caracteres, status arquivado e conta já
  desvinculada;
- locka profissional e dependências futuras de scheduling;
- troca nome por valor neutro, grava instante/ator, incrementa revisão, audita
  sem nome e persiste replay na mesma transação;
- não apaga linhas nem chama `people` ou outro módulo.

Relógio deve ser injetável. Não espalhar `time.Now()` entre handler, domínio e
repository.

## 9. Contratos HTTP

Todas as rotas ficam sob `/api/v1/units/:unit_id/scheduling`, exigem capability
`scheduling`, sessão tenant e CSRF em mutações.

### Listar profissionais

```http
GET /professionals?include_archived=false&query=
```

- owner recebe ativos/arquivados conforme filtro e resumo do vínculo;
- coach recebe somente ativos, sem `linked_user_id`, auditoria ou ações;
- ordenação por `display_name_key`, depois ID; `query` normalizada possui no
  máximo 100 caracteres;
- resposta não contém e-mail, telefone, CPF, papel de `Person` ou unidade
  diferente da atual.

### Listar contas elegíveis

```http
GET /professional-account-candidates
```

Somente owner. Retorna `user_id`, rótulo, papel e se já está vinculada. A rota
compõe o port de access e o vínculo atual sem expor e-mail ou credencial.

### Criar

```http
POST /professionals
Idempotency-Key: scheduling-professional.<uuid>
```

```json
{
  "display_name": "Marina",
  "linked_user_id": null
}
```

Retorna `201`; replay retorna `200` com o mesmo ID/revisão.

### Atualizar

```http
PATCH /professionals/:professional_id
Idempotency-Key: scheduling-professional-update.<uuid>
```

```json
{
  "display_name": "Marina Souza",
  "linked_user_id": "uuid-or-null",
  "expected_revision": 1
}
```

Ao menos um campo mutável é obrigatório. Retorna `200`.

### Arquivar e reativar

```http
POST /professionals/:professional_id/archive
POST /professionals/:professional_id/reactivate
```

Ambos exigem `Idempotency-Key`, `expected_revision` e motivo.

### Exportar e anonimizar

```http
GET  /professionals/:professional_id/privacy-export
POST /professionals/:professional_id/anonymize
```

- export owner retorna JSON com cadastro, unidade, vínculo de conta por ID,
  estado e eventos de auditoria sem dados da conta;
- anonimização exige `Idempotency-Key`, `expected_revision` e motivo;
- nenhum endpoint aceita `organization_id`, `created_by` ou status arbitrário.

## 10. Erros estáveis

| HTTP | Código | Situação |
|---:|---|---|
| 400 | `invalid_request` | JSON ou UUID malformado |
| 404 | `unit_not_found` | unidade ausente/fora do escopo |
| 404 | `scheduling_professional_not_found` | profissional ausente/outro tenant |
| 409 | `scheduling_revision_conflict` | revisão antiga |
| 409 | `scheduling_professional_account_conflict` | conta já ligada na organização |
| 409 | `scheduling_professional_archived` | mutação incompatível com arquivado |
| 409 | `scheduling_professional_anonymized` | tentativa de reativar/editar anonimizado |
| 409 | `idempotency_key_reused` | chave com outro payload/comando |
| 422 | `scheduling_professional_invalid` | nome ou combinação inválida |
| 422 | `scheduling_professional_account_unavailable` | conta inativa, papel inválido ou fora da unidade |
| 422 | `scheduling_professional_anonymization_blocked` | conta/dependência futura impede anonimização |

Erros incluem `request_id`. Constraints conhecidas são traduzidas; SQL, nome
de constraint, nome de profissional e dados da conta não entram na mensagem de
log ou erro técnico.

## 11. Experiência mobile-first

### Navegação

- adicionar `PageKey`/hash `#/schedule` condicionado a `scheduling`;
- item `Agenda` no grupo operacional para owner e coach;
- nesta slice a página abre diretamente na seção `Profissionais` e não mostra
  tabs vazias de Agenda/Grade;
- owner vê `Adicionar profissional`, filtros e ações;
- coach vê lista somente leitura com explicação curta;
- nome da unidade atual aparece no cabeçalho, sem seletor multiunidade.

### Fluxo owner

1. tocar em `Adicionar profissional`;
2. informar somente `Nome`;
3. opcionalmente abrir `Vincular conta` e escolher owner/coach elegível;
4. salvar uma única vez e ver o card criado;
5. menu secundário oferece editar, arquivar, reativar, exportar e anonimizar
   quando permitido.

Requisitos:

- uma coluna em `360x800`, sem overflow horizontal;
- alvos de toque de no mínimo 44 px;
- conta opcional em abertura progressiva;
- homônimos exibidos sem alerta falso de duplicidade;
- estados loading, vazio, erro, conflito, retry e sucesso explícitos;
- ações destrutivas mostram nome, consequência e confirmação;
- anonimização explica irreversibilidade e exige motivo;
- drawer/modal preserva foco, labels, Escape e rolagem interna;
- nenhuma tela exibe UUID, `box`, tenant, CPF, e-mail ou termos técnicos.

## 12. Privacidade e retenção

- `display_name` é o único dado pessoal obrigatório novo;
- logs, métricas, auditoria e command requests não repetem o nome;
- exportação é auditada e retorna somente dados pertencentes a scheduling;
- arquivamento preserva o nome para operação/histórico e pode ser revertido;
- anonimização explícita é irreversível pela API e usa valor neutro;
- não há job automático de retenção nesta slice;
- prazo/base de retenção automática exige revisão jurídica futura;
- slices B–D devem chamar a operação interna de anonimização para neutralizar
  snapshots históricos de sessão na mesma transação, sem apagar horários;
- não inferir nem reutilizar política de anonimização de `people`.

## 13. Capability, rollout e compatibilidade

Adicionar `FEATURE_SCHEDULING_ENABLED`, exposta como `scheduling`:

- default `false` em development e production;
- não exige `catalog`, `memberships`, `people` ou `SPEC-004/A` nesta slice;
- frontend trata campo ausente como `false`;
- backend bloqueia todas as rotas quando desligada;
- `SCHEDULING_MATERIALIZER_ENABLED` não existe/é ignorada nesta slice, pois não
  há sessões.

Ordem segura:

1. reservar número da migration no merge;
2. aplicar migration aditiva sem seed/backfill;
3. publicar backend com capability desligada;
4. publicar frontend tolerante a capability ausente;
5. validar contratos de access com fakes e adapter real;
6. habilitar apenas em ambiente controlado;
7. homologar owner/coach e privacy em mobile;
8. ativação de produção depende de decisão separada.

Rollback operacional desliga `scheduling` e pode reverter binários. Schema e
dados permanecem. Nenhum fluxo existente passa a depender dos profissionais.

## 14. Observabilidade

Logs estruturados:

- `scheduling_professional_command_completed|failed` com comando, resultado,
  organização, unidade, IDs, revisão, duração e `request_id`;
- nunca registrar nome, motivo, dados de conta ou payload completo.

Métricas sem labels de alta cardinalidade:

- `engagefit.scheduling.professionals.commands{command,result}`;
- `engagefit.scheduling.professionals.conflicts{reason}`;
- `engagefit.scheduling.professionals.active` como observable gauge agregado.

Auditoria é a fonte durável das mutações. Não criar dashboard, alerta externo,
worker ou integração nesta slice.

## 15. Critérios de aceite

### Produto e experiência

- owner cadastra profissional informando somente nome em `360x800`;
- nome homônimo é aceito e produz outro ID;
- owner edita nome, vincula/desvincula conta, arquiva e reativa;
- conta owner/coach ativa pode ser vinculada sem alterar papel/permissão;
- mesma conta não pode ser vinculada a dois profissionais da organização;
- coach lista ativos, mas não vê nem executa qualquer mutação;
- platform admin não acessa a superfície tenant;
- exportação e anonimização funcionam com confirmação e motivo;
- capability ausente/falsa esconde navegação e bloqueia API.

### Arquitetura, dados e segurança

- scheduling não importa/consulta repository, model ou tabela de access;
- nenhum contrato de catalog, memberships ou people é criado/alterado;
- UUID adversário de outra organização/unidade retorna `404`;
- corrida de vínculo da mesma conta deixa exatamente um vencedor;
- revisão antiga não sobrescreve edição;
- replay idêntico preserva ID/revisão e chave reutilizada não executa;
- falha de auditoria/command request reverte toda a mutação;
- migration vazia em upgrade não cria profissional para usuário/pessoa atual;
- logs/auditoria não contêm nome, e-mail, telefone, CPF ou motivo livre;
- anonimização não apaga ID, unidade ou eventos e não pode ser revertida;
- nenhuma série, sessão, reserva, presença, consumo ou efeito externo é criado.

## 16. Testes obrigatórios

### Backend

- domínio: normalização Unicode/whitespace, limites, homônimos, estados e
  anonimização;
- aplicação: create/update/archive/reactivate/anonymize, revisão,
  idempotência, rollback e relógio;
- ports: fakes e contrato do contexto/contas, sem DTO HTTP ou GORM estrangeiro;
- PostgreSQL: constraints, índice parcial da conta, transação, concorrência e
  isolamento com duas organizações/unidades;
- HTTP: owner/coach/platform admin, CSRF, cookie/Bearer, capability, erros,
  UUIDs adversários e `request_id`;
- privacy: export, nome ausente de logs/auditoria e anonimização irreversível;
- migration em banco vazio, upgrade desde o `main` integrado e segunda
  execução com zero migrations;
- `gofmt`, `go test ./...`, `go vet ./...`, `git diff --check` e race detector
  no ambiente Linux suportado.

### Frontend

- capability ausente/falsa;
- lista read-only do coach e bloqueio de hash/ações;
- criação mínima, homônimos e vínculo opcional de conta;
- edição, desvínculo, arquivamento, filtro, reativação, export e anonimização;
- conflitos de revisão/conta e falha de rede recuperáveis;
- foco, Escape, labels, toque e ausência de overflow em `360x800`;
- TypeScript, build Vite, Playwright focado e regressão completa;
- `git diff --check`.

## 17. Definition of Done e limites do executor

O executor entrega uma slice vertical em commits separados de backend e
frontend, atualiza `docs/system-design.md` e o status/evidências somente depois
de implementação e validação e informa migration provisória ao integrador.

O executor não deve:

- implementar `SPEC-006/B`, C ou D;
- criar série, sessão, materializador, espaço, conflito ou calendário;
- alterar autorização de owner/coach ou criar conta;
- adicionar campo clínico, financeiro, comissão ou PII além do nome;
- consultar tabelas/repositories/models de outro módulo;
- mudar contratos de `catalog`, `memberships`, `people` ou `contracts`;
- tocar check-ins, retenção, workout, athlete, produção ou banco compartilhado;
- fazer push, merge, deploy ou habilitar feature flag.

Qualquer divergência de contrato exige atualizar esta spec antes do código e
destacar a decisão no handoff.
