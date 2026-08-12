# SPEC-001/B — Catálogo de modalidades e espaços

- Status: `validated locally`
- Versão: `1.1`
- Data: 2026-08-12
- Depende de: `ARCH-001`, `SPEC-001/A`
- Migration reservada: `054`, sujeita à confirmação do integrador no merge
- Ownership: módulo `catalog`, com leitura mínima de `organizations`
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`

## 1. Resultado esperado

O owner da unidade atual consegue cadastrar, consultar, alterar, arquivar e
reativar modalidades e espaços usando o celular. O cadastro essencial exige
somente o nome. Modalidades nascem ativas na unidade atual; espaços podem ter
capacidade padrão opcional.

Esta slice cria o catálogo que planos e agenda reutilizarão. Ela não implementa
agenda, planos, segunda unidade ou troca de contexto.

## 2. Decisões de produto e proveniência

- **Benchmark — Next Fit:** modalidade representa a atividade física; preço,
  frequência, duração e quantidade de aulas pertencem ao contrato/plano.
- **Benchmark — Next Fit:** agenda referencia modalidade e local, mas suas
  regras de reserva pertencem à grade/contrato, não ao cadastro da modalidade.
- **Benchmark — Tecnofit:** modalidade aparece como cadastro auxiliar usado em
  contratos.
- **Inferência EngageFit:** modalidade pede somente nome; cor é opcional e
  recebe um padrão útil.
- **Inferência EngageFit:** espaço pede somente nome; capacidade padrão é
  opcional e poderá ser sobrescrita por uma sessão de aula futura.
- **Inferência EngageFit:** não haverá flags `utiliza agenda` ou `utiliza WOD`
  nesta entidade. A existência de agenda, treino ou direitos será determinada
  pelos módulos proprietários dessas regras.
- **Validação futura:** confirmar com o Alados se o termo de interface
  `Espaço` é mais claro que `Local` para box, sala, tatame e quadra.

Referências de benchmark:

- https://ajuda.nextfit.com.br/support/solutions/articles/69000554927-como-criar-uma-modalidade-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000840583-como-criar-uma-grade-de-hor%C3%A1rios-mista-contratos-e-servicos-
- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000698983-como-criar-uma-nova-modalidade-

## 3. Escopo

### Incluído

- leitura resumida da organização e da unidade atual;
- CRUD sem exclusão física de modalidades;
- ativação da modalidade na unidade atual;
- CRUD sem exclusão física de espaços da unidade atual;
- nomes normalizados e únicos no escopo correto;
- auditoria das mutações;
- feature flag e rollout compatível entre frontend e backend;
- tela `Estrutura` exclusiva do owner, mobile-first;
- leitura de catálogo pela role `COACH` via API para consumo futuro.

### Fora do escopo

- criar, editar, suspender ou arquivar unidades;
- timezone, endereço, CNPJ e contatos da unidade;
- seletor de unidade e acesso multiunidade;
- memberships e allowlist de unidade;
- planos, contratos, agenda, aulas, reservas e check-ins;
- WOD por modalidade;
- upload ou catálogo de ícones;
- exclusão física;
- importação automática de modalidades a partir de dados atuais;
- API pública para consultar eventos de auditoria;
- `Idempotency-Key` genérico para comandos internos de catálogo.

## 4. Regras e invariantes

1. Modalidade pertence exatamente a uma organização.
2. Espaço pertence exatamente a uma unidade (`box`).
3. A associação modalidade/unidade deve pertencer à mesma organização nos
   dois lados; essa regra deve ser protegida no banco e no repositório.
4. Na criação, a modalidade é associada como ativa à unidade da sessão na
   mesma transação.
5. O nome visível é armazenado com espaços externos removidos e espaços
   internos consecutivos reduzidos a um único espaço.
6. A chave de comparação do nome é a versão normalizada em minúsculas.
7. Nome de modalidade é único por organização usando a chave normalizada.
8. Nome de espaço é único por unidade usando a chave normalizada.
9. Nomes possuem de 1 a 80 caracteres após normalização.
10. Cor de modalidade é opcional; quando ausente, usar `#0F766E`. Quando
    informada, aceitar somente hexadecimal `#RRGGBB` e persistir em maiúsculas.
11. Capacidade padrão é opcional; quando informada, deve ser inteira entre 1 e
    10.000.
12. Status permitido para modalidade e espaço: `active` ou `archived`.
13. Arquivar não apaga nem rompe associações. O recurso deixa de aparecer na
    listagem padrão e não pode ser usado em novas associações.
14. Reativar restaura o recurso e suas associações anteriores. A ativação
    efetiva da modalidade exige `modality.status = active` e
    `unit_modalities.active = true`.
15. Não existe `DELETE` HTTP nesta slice.
16. IDs de organização e unidade nunca são aceitos do corpo como fonte de
    autorização; o escopo vem da sessão e é validado novamente no repositório.
17. Conhecer um UUID de outra organização ou unidade retorna `404`, sem revelar
    a existência do recurso.
18. `PLATFORM_ADMIN` não ganha acesso operacional ao catálogo.
19. Nenhuma modalidade ou espaço será criado por backfill. Organizações atuais
    começam com estado vazio e sugestões apenas na interface.

### Normalização canônica de nome

A aplicação deve produzir `name_key` com uma única função compartilhada:

1. remover espaços externos;
2. substituir qualquer sequência de whitespace por um espaço comum;
3. aplicar lowercase Unicode;
4. rejeitar resultado vazio ou maior que 80 caracteres.

Exemplo: `"  Cross   Training "` vira nome `"Cross Training"` e chave
`"cross training"`.

## 5. Modelo de dados

### `modalities`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório, FK `organizations`, `ON DELETE RESTRICT` |
| `name` | varchar(80), nome normalizado para exibição |
| `name_key` | varchar(80), chave canônica |
| `color` | char(7), default `#0F766E` |
| `status` | `active` ou `archived` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- unique `(organization_id, name_key)`;
- unique `(id, organization_id)` para integridade composta;
- checks de nome não vazio, cor e status.

### `unit_modalities`

| Campo | Regra |
|---|---|
| `organization_id` | UUID obrigatório |
| `unit_id` | UUID obrigatório; representa `boxes.id` |
| `modality_id` | UUID obrigatório |
| `active` | boolean default `true` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- PK `(unit_id, modality_id)`;
- FK composta `(unit_id, organization_id)` para
  `boxes(id, organization_id)`;
- FK composta `(modality_id, organization_id)` para
  `modalities(id, organization_id)`;
- `boxes` recebe unique `(id, organization_id)` de forma aditiva.

### `spaces`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `unit_id` | UUID obrigatório, FK `boxes`, `ON DELETE RESTRICT` |
| `name` | varchar(80), nome normalizado para exibição |
| `name_key` | varchar(80), chave canônica |
| `default_capacity` | integer nullable, de 1 a 10.000 |
| `status` | `active` ou `archived` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- unique `(unit_id, name_key)`;
- unique `(id, unit_id)` para referências futuras;
- checks de nome, capacidade e status.

### `catalog_audit_events`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório |
| `unit_id` | UUID nullable |
| `actor_user_id` | UUID obrigatório |
| `action` | varchar(80) |
| `entity_type` | `modality`, `unit_modality` ou `space` |
| `entity_id` | UUID obrigatório |
| `before_data`, `after_data` | jsonb com dados não sensíveis |
| `created_at` | timestamptz obrigatório |

Eventos mínimos:

- `modality.created`, `modality.updated`, `modality.archived`,
  `modality.reactivated`;
- `unit_modality.activated`, `unit_modality.deactivated`;
- `space.created`, `space.updated`, `space.archived`,
  `space.reactivated`.

Criação/alteração e auditoria pertencem à mesma transação. Payloads não devem
conter dados pessoais nem objetos HTTP completos.

### Migration

O integrador reservou provisoriamente `054_create_catalog_structure.sql` para
esta branch, que depende da migration `053`. Se outra migration alcançar a
base antes do merge, o integrador renumera o arquivo e atualiza o teste do
migrator; o agente executor não escolhe outro número sozinho.

A migration é somente aditiva. Não há backfill de catálogo.

## 6. Contratos HTTP

Todas as rotas abaixo ficam sob `/api/v1`, sessão tenant e capability
`catalog`. Respostas de erro usam o envelope padrão com `request_id`.

### Organização atual

`GET /organization`

Roles: `OWNER`, `COACH`.

Resposta `200`:

```json
{
  "id": "organization-uuid",
  "name": "CrossFit Alados",
  "status": "active",
  "current_unit": {
    "id": "unit-uuid",
    "name": "CrossFit Alados",
    "status": "active"
  }
}
```

Esta slice não oferece `PATCH /organization`.

### Modalidades

`GET /modalities?include_archived=false&unit_id=<current-unit-id>`

- roles: `OWNER`, `COACH`;
- `unit_id` é opcional e assume a unidade da sessão;
- nesta slice, qualquer `unit_id` diferente do `box_id` da sessão retorna
  `404`;
- ordenação: ativas primeiro, depois `name_key` ascendente.

Item de resposta:

```json
{
  "id": "modality-uuid",
  "name": "Cross Training",
  "color": "#0F766E",
  "status": "active",
  "active_in_unit": true,
  "created_at": "2026-08-11T20:00:00Z",
  "updated_at": "2026-08-11T20:00:00Z"
}
```

`POST /modalities`

- role: `OWNER`;
- body: `{ "name": "Cross Training", "color": "#0F766E" }`;
- associa automaticamente a unidade da sessão;
- retorna `201` com o item.

`PATCH /modalities/:id`

- role: `OWNER`;
- body parcial: `{ "name"?, "color"?, "status"? }`;
- ao menos um campo deve ser informado;
- retorna `200` com o item atualizado.

`PUT /units/:unit_id/modalities/:modality_id`

- role: `OWNER`;
- body: `{ "active": true }`;
- nesta slice, `unit_id` deve ser a unidade da sessão;
- ativar modalidade arquivada retorna `409`;
- retorna `200` com o item de modalidade no contexto da unidade.

### Espaços

`GET /units/:unit_id/spaces?include_archived=false`

- roles: `OWNER`, `COACH`;
- nesta slice, `unit_id` deve ser a unidade da sessão;
- ordenação: ativos primeiro, depois `name_key` ascendente.

Item de resposta:

```json
{
  "id": "space-uuid",
  "unit_id": "unit-uuid",
  "name": "Área principal",
  "default_capacity": 20,
  "status": "active",
  "created_at": "2026-08-11T20:00:00Z",
  "updated_at": "2026-08-11T20:00:00Z"
}
```

`POST /units/:unit_id/spaces`

- role: `OWNER`;
- body: `{ "name": "Área principal", "default_capacity": 20 }`;
- `default_capacity` pode ser omitida ou `null`;
- retorna `201` com o item.

`PATCH /units/:unit_id/spaces/:id`

- role: `OWNER`;
- body parcial: `{ "name"?, "default_capacity"?, "status"? }`;
- `default_capacity: null` remove a capacidade padrão;
- ao menos um campo deve ser informado;
- retorna `200` com o item atualizado.

### Erros estáveis

| HTTP | Código | Situação |
|---|---|---|
| 400 | `invalid_catalog_input` | body, nome, cor, capacidade ou status inválido |
| 404 | `organization_not_found` | organização da sessão ausente |
| 404 | `unit_not_found` | unidade ausente ou fora do escopo |
| 404 | `modality_not_found` | modalidade ausente ou fora da organização |
| 404 | `space_not_found` | espaço ausente ou fora da unidade |
| 409 | `modality_name_conflict` | nome normalizado já existe na organização |
| 409 | `space_name_conflict` | nome normalizado já existe na unidade |
| 409 | `modality_archived` | tentativa de ativar modalidade arquivada |

Constraint errors conhecidas devem ser traduzidas para esses códigos; SQL e
nomes internos de constraints não chegam ao cliente.

### Retry e duplicidade

`Idempotency-Key` não entra nesta slice. POSTs possuem identidade natural por
nome normalizado e constraint única. O frontend desabilita submissão durante o
request; se receber conflito após uma perda de resposta, recarrega a lista e
mostra o recurso existente. PATCH/PUT expressam o estado desejado e são
repetíveis.

## 7. Autorização e isolamento

- handlers nunca recebem `organization_id` confiável do cliente;
- o `organization_id` vem do box autenticado;
- consultas de modalidade filtram simultaneamente por `id` e
  `organization_id`;
- consultas de espaço filtram simultaneamente por `id` e `unit_id`;
- a rota com `unit_id` valida primeiro que ele corresponde ao `box_id` da
  sessão; a `SPEC-001/C` substituirá essa igualdade por allowlist explícita;
- GETs permitem owner e coach; mutações usam `middleware.Owner()`;
- platform admin continua separado pelo grupo `/admin`;
- testes usam UUIDs conhecidos de duas organizações para provar que IDs de
  outra organização não podem ser lidos, alterados ou associados.

## 8. Backend e fronteiras

- casos de uso em `internal/app/catalog`;
- contratos públicos em `internal/ports/repositories`, sem expor GORM;
- entidades/enums de catálogo permanecem no domínio conforme o padrão atual;
- persistência em `internal/adapters/persistence/postgres`;
- handlers e DTOs não contêm regra de normalização ou tenancy;
- composição explícita em `cmd/api/main.go`;
- operações compostas e auditoria são transacionais;
- nenhum módulo existente consulta diretamente as novas tabelas nesta slice.

## 9. Feature flag e rollout

Adicionar `FEATURE_CATALOG_ENABLED`:

- default `true` em development/test;
- default `false` em production;
- exposta em `GET /api/v1/capabilities` como `catalog`;
- rotas desta spec retornam `404 capability_disabled` quando desligada;
- frontend trata campo ausente como `false`, permitindo frontend novo com
  backend antigo.

Sequência:

1. aplicar migration;
2. publicar backend com capability desligada em produção;
3. publicar frontend em qualquer ordem;
4. executar smoke test autenticado e reconciliações;
5. habilitar capability;
6. observar erros HTTP e uso da tela.

Rollback operacional: desligar `FEATURE_CATALOG_ENABLED` e voltar frontend/API
se necessário. As tabelas aditivas permanecem; a migration não possui down
automático e nenhum dado legado depende do catálogo.

## 10. Experiência mobile-first

### Navegação

- criar `PageKey` `structure` e hash `#/structure`;
- item `Estrutura` no grupo `Gestão`, somente para `OWNER` e capability
  `catalog === true`;
- acesso direto sem capability ou como coach redireciona para uma página
  permitida;
- não alterar a tela atual de Configurações.

### Tela `Estrutura`

- cabeçalho: `Estrutura` e resumo `Organize modalidades e espaços da sua
  unidade`;
- mostrar organização e unidade atual em resumo compacto;
- não mostrar seletor ou card de múltiplas unidades;
- tabs ou segmented control `Modalidades` e `Espaços`;
- uma ação primária `Adicionar modalidade` ou `Adicionar espaço`, conforme a
  seção ativa;
- em viewport móvel, formulário em uma coluna e ações com alvo mínimo de 44px;
- desktop pode usar grid de cards, sem tabela como única representação.

### Modalidade

- formulário inicial mostra somente `Nome`;
- `Cor` fica em abertura progressiva `Personalizar` com um pequeno conjunto de
  opções e valor padrão;
- sugestões não persistidas: `Cross Training`, `Musculação`, `Funcional`,
  `Pilates` e `Spinning`;
- selecionar sugestão apenas preenche o campo;
- card mostra nome, marcador de cor e estado;
- menu secundário permite editar, arquivar e reativar;
- arquivamento exige confirmação curta e explica que o histórico será
  preservado.

### Espaço

- formulário mostra `Nome` e `Capacidade padrão (opcional)`;
- helper: `Você poderá ajustar a capacidade em cada aula.`;
- sugestões não persistidas: `Área principal`, `Sala 1`, `Tatame` e `Quadra`;
- card mostra capacidade somente quando informada;
- menu secundário permite editar, arquivar e reativar.

### Estados

- loading com descrição contextual;
- empty state com benefício e ação principal;
- erro com mensagem segura e `Tentar novamente`;
- sucesso próximo à ação, sem depender apenas de toast efêmero;
- `Mostrar arquivados` fica desligado por padrão;
- ao receber conflito de nome, focar o campo e explicar onde o nome já existe;
- nenhuma tela exibe UUID, `box`, tenant ou termos técnicos.

## 11. Critérios de aceite

### Backend

- [ ] migration funciona em banco vazio e sobre schema `053` com dados;
- [ ] segunda execução do migrator aplica zero migrations;
- [ ] criar modalidade também cria associação ativa com a unidade na mesma
  transação;
- [ ] falha na associação ou auditoria reverte a criação;
- [ ] nomes normalizados duplicados retornam `409` no escopo correto;
- [ ] mesmo nome pode existir em organizações/unidades diferentes;
- [ ] banco rejeita associação modalidade/unidade de organizações diferentes;
- [ ] owner pode mutar e coach pode apenas ler;
- [ ] UUID de outra organização/unidade retorna `404` em leitura e mutação;
- [ ] arquivar e reativar preservam IDs e histórico;
- [ ] capability desligada bloqueia todas as novas rotas;
- [ ] login, box, alunos, check-ins e app do atleta continuam funcionando.

### Frontend

- [ ] frontend antigo continua funcionando com backend novo;
- [ ] frontend novo esconde `Estrutura` quando `catalog` está ausente ou false;
- [ ] owner adiciona modalidade informando apenas nome;
- [ ] owner adiciona espaço sem informar capacidade;
- [ ] editar, arquivar, mostrar arquivados e reativar funcionam;
- [ ] conflitos e falhas de rede deixam formulário recuperável;
- [ ] coach e platform admin não veem nem acessam a tela;
- [ ] fluxo funciona em viewport `390x844` sem rolagem horizontal;
- [ ] foco, labels, teclado e botões permitem operação touch e teclado.

## 12. Testes mínimos e comandos

Backend:

- unitários: normalização, validação, status e casos de uso;
- integração PostgreSQL: constraints, transações e isolamento entre dois
  tenants;
- HTTP: contratos, roles, CSRF, códigos de erro e feature gate;
- regressão completa e race detector no ambiente suportado.

```bash
gofmt -w <arquivos-go-alterados>
go test ./...
go vet ./...
git diff --check
```

Frontend:

- teste Playwright focado em `tests/structure.spec.ts` com API mockada;
- fluxo de modalidade e espaço em viewport móvel;
- teste de capability ausente/desligada e role coach;
- build completo e inspeção visual da tela mobile.

```bash
node node_modules/typescript/lib/tsc.js -b
node node_modules/vite/bin/vite.js build
node node_modules/@playwright/test/cli.js test tests/structure.spec.ts
git diff --check
```

## 13. Entrega e limites do agente

O agente entrega commits separados por repositório, ambos na branch
`feature/spec-001-catalog-structure`, com:

- migration e implementação backend;
- contratos e tela frontend;
- testes e documentação do comportamento implementado;
- relatório de comandos executados, checks não executados e riscos restantes.

O agente não deve:

- implementar `SPEC-001/C`;
- criar segunda unidade ou seletor;
- mudar autenticação/JWT;
- renomear `boxes` ou remover `users.box_id`;
- implementar agenda, plano ou contrato;
- fazer push, merge, deploy ou habilitar production;
- modificar os arquivos não commitados do repositório principal.

Se um contrato desta spec precisar mudar, o agente deve atualizar este
documento e destacar a decisão no handoff da execução antes de codificar o
contrato divergente.

## 14. Evidências de validação

- migration `054` validada em banco vazio e no upgrade de schema `053`
  populado, com segunda execução sem migrations pendentes;
- constraints PostgreSQL e repositories validados entre duas organizações,
  incluindo rejeição de associação e auditoria com unidade de outro tenant;
- criação de modalidade, associação com unidade e auditoria cobertas pela mesma
  transação, inclusive rollback forçado por falha de auditoria;
- contratos HTTP, feature gate, leitura por coach, mutação exclusiva do owner,
  CSRF e ocultação de UUIDs externos cobertos pela suíte;
- tela validada em viewport `390x844`, incluindo criação, conflito, retry,
  edição, remoção de capacidade, arquivamento e reativação;
- suíte completa, race detector, build e regressões Playwright permanecem como
  gates obrigatórios de CI antes do merge.
