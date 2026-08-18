# SPEC-001/D — Catálogo de serviços ofertáveis

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-08-18
- Depende de: `ARCH-001`, `SPEC-001/A-B`
- Ownership: módulo `catalog`
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration integrada: `055_create_service_offerings.sql`

## 1. Resultado esperado

O owner da unidade atual consegue cadastrar, consultar, alterar, arquivar e
reativar serviços como Recovery, Avaliação física, Fisioterapia ou Nutrição
usando o celular. O cadastro inicial exige somente o nome e deixa descrição e
cor em abertura progressiva.

Esta slice entrega uma referência estável que planos e agenda poderão usar.
Ela não define preço, duração, profissional, disponibilidade, capacidade nem
regras de reserva.

## 2. Problema e decisões de produto

- **Existente:** `SPEC-001/B` já separa modalidade, associação da modalidade à
  unidade e espaço físico no módulo `catalog`.
- **Benchmark — Tecnofit:** a configuração de agenda de serviços trata nome,
  duração e quantidade de atendimentos simultâneos antes de associar
  profissionais e disponibilidades.
- **Benchmark — Next Fit:** o serviço pode ser vendido/consumido, enquanto a
  grade recebe modalidade, profissional, local, duração, capacidade e regras
  de reserva.
- **Inferência EngageFit:** o catálogo guarda somente a identidade do serviço.
  Duração, profissional, local, vagas e disponibilidade pertencem à agenda;
  preço, quantidade e validade pertencem ao plano, adicional ou venda.
- **Inferência EngageFit:** um serviço pertence à organização e é ativado por
  unidade, assim como uma modalidade. Isso evita duplicar “Avaliação física”
  quando a organização ganhar uma segunda unidade.
- **Validação futura:** confirmar no piloto se `Serviços` é o melhor termo para
  o owner e se Recovery deve aparecer como serviço, modalidade ou ambos em
  operações diferentes.

Referências de benchmark:

- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000739562-como-configurar-a-agenda-de-servicos-no-starter-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000555089
- https://ajuda.nextfit.com.br/support/solutions/articles/69000554927-como-criar-uma-modalidade-

## 3. Escopo

### Incluído

- CRUD sem exclusão física de serviços ofertáveis;
- associação e ativação do serviço na unidade atual;
- nome normalizado e único por organização;
- descrição curta e cor opcionais;
- auditoria transacional das mutações;
- API de leitura para `OWNER` e `COACH`;
- tela mobile-first dentro de `Estrutura` para `OWNER`;
- feature flag/capability independente e rollout compatível;
- testes de isolamento entre organizações e unidades.

### Fora do escopo

- duração, preço, moeda, pacotes, créditos ou validade;
- profissional, especialidade, comissão ou disponibilidade;
- espaço, capacidade, grade, sessão, reserva, fila ou check-in;
- venda de serviço e vínculo com plano;
- serviço individual versus coletivo;
- formulários, PAR-Q ou prontuário;
- segunda unidade, seletor de unidade ou membership multiunidade;
- exclusão física, importação e API pública de auditoria;
- alteração dos contratos de modalidade e espaço da `SPEC-001/B`.

## 4. Regras e invariantes

1. Serviço pertence exatamente a uma organização.
2. A ativação do serviço pertence a uma unidade da mesma organização.
3. Na criação, o serviço é associado como ativo à unidade da sessão na mesma
   transação.
4. Nome possui de 1 a 80 caracteres após normalização.
5. Nome visível remove whitespace externo e reduz sequências internas a um
   espaço; `name_key` aplica lowercase Unicode ao resultado.
6. `name_key` é único por organização, inclusive para serviço arquivado.
7. Descrição é opcional, remove whitespace externo e possui no máximo 280
   caracteres; ausente é persistida como string vazia.
8. Cor ausente usa `#0F766E`; quando informada aceita somente `#RRGGBB` e é
   persistida em maiúsculas.
9. Status permitido: `active` ou `archived`.
10. Arquivar não apaga o serviço nem a associação. Ele some da listagem padrão
    e não pode ser ativado enquanto estiver arquivado.
11. Reativar restaura o serviço e preserva a associação anterior. O estado
    efetivo exige `service_offerings.status = active` e
    `unit_service_offerings.active = true`.
12. Não existe `DELETE` HTTP nesta slice.
13. `organization_id` e `unit_id` do corpo nunca são fonte de autorização.
14. Nesta slice, todo `unit_id` de rota/query deve ser igual ao `box_id` da
    sessão; outro UUID retorna `404`.
15. Conhecer ID de outro tenant nunca revela sua existência.
16. `PLATFORM_ADMIN` não recebe acesso operacional ao catálogo.

A implementação deve reutilizar a função canônica de normalização de nome do
catálogo criada na `SPEC-001/B`, sem uma segunda versão local.

## 5. Modelo de dados

### `service_offerings`

| Campo | Regra |
|---|---|
| `id` | UUID PK |
| `organization_id` | UUID obrigatório, FK `organizations`, `ON DELETE RESTRICT` |
| `name` | varchar(80), nome normalizado para exibição |
| `name_key` | varchar(80), chave canônica |
| `description` | varchar(280), default vazio |
| `color` | char(7), default `#0F766E` |
| `status` | `active` ou `archived` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- unique `(organization_id, name_key)`;
- unique `(id, organization_id)` para integridade composta;
- checks de nome, descrição, cor e status.

### `unit_service_offerings`

| Campo | Regra |
|---|---|
| `organization_id` | UUID obrigatório |
| `unit_id` | UUID obrigatório; representa `boxes.id` |
| `service_offering_id` | UUID obrigatório |
| `active` | boolean default `true` |
| `created_at`, `updated_at` | timestamptz obrigatórios |

Constraints:

- PK `(unit_id, service_offering_id)`;
- FK composta `(unit_id, organization_id)` para
  `boxes(id, organization_id)`;
- FK composta `(service_offering_id, organization_id)` para
  `service_offerings(id, organization_id)`.

### Auditoria existente

Ampliar `catalog_audit_events.entity_type` de forma aditiva para aceitar
`service_offering` e `unit_service_offering`, preservando os três valores já
existentes. A migration deve substituir a check constraint pelo nome conhecido
e não recriar a tabela.

Eventos mínimos:

- `service_offering.created`, `.updated`, `.archived`, `.reactivated`;
- `unit_service_offering.activated`, `.deactivated`.

Criação/alteração e auditoria pertencem à mesma transação. `before_data` e
`after_data` contêm apenas os campos do catálogo, nunca sessão ou request HTTP.

### Migration e concorrência entre agentes

A branch parte da migration `054`. O executor usa provisoriamente o próximo
número disponível no próprio worktree e atualiza o teste do migrator. Como a
`SPEC-002/A` pode criar outra migration em paralelo, o integrador reserva o
número final na ordem de merge, renomeia uma delas se necessário e executa as
duas suítes de upgrade. Essa renumeração não altera o contrato desta spec.

Não há backfill nem seed obrigatório. As sugestões existem apenas na UI.

## 6. Contratos HTTP

Todas as rotas ficam sob `/api/v1`, sessão tenant e capabilities `catalog` e
`service_catalog`. Erros usam o envelope atual com `request_id`.

### Listar

`GET /service-offerings?include_archived=false&unit_id=<current-unit-id>`

- roles: `OWNER`, `COACH`;
- `unit_id` é opcional e assume a unidade da sessão;
- outro `unit_id` retorna `404 unit_not_found`;
- ordenação: ativos primeiro, depois `name_key` ascendente.

Resposta `200`:

```json
[
  {
    "id": "service-offering-uuid",
    "name": "Recovery",
    "description": "Sessão de recuperação",
    "color": "#0F766E",
    "status": "active",
    "active_in_unit": true,
    "created_at": "2026-08-18T20:00:00Z",
    "updated_at": "2026-08-18T20:00:00Z"
  }
]
```

### Criar

`POST /service-offerings`

- role: `OWNER`;
- body:

```json
{
  "name": "Recovery",
  "description": "Sessão de recuperação",
  "color": "#0F766E"
}
```

- `description` e `color` podem ser omitidos;
- associa automaticamente à unidade da sessão;
- retorna `201` com o item.

### Alterar/arquivar/reativar

`PATCH /service-offerings/:id`

- role: `OWNER`;
- body parcial: `{ "name"?, "description"?, "color"?, "status"? }`;
- ao menos um campo deve ser informado;
- `description: ""` remove a descrição;
- retorna `200` com o item atualizado.

### Ativar/desativar na unidade

`PUT /units/:unit_id/service-offerings/:service_offering_id`

- role: `OWNER`;
- body: `{ "active": true }`;
- `unit_id` deve ser a unidade da sessão;
- ativar serviço arquivado retorna `409`;
- retorna `200` com o item no contexto da unidade.

### Erros estáveis

| HTTP | Código | Situação |
|---|---|---|
| 400 | `invalid_catalog_input` | body, nome, descrição, cor ou status inválido |
| 404 | `unit_not_found` | unidade ausente ou fora do escopo |
| 404 | `service_offering_not_found` | serviço ausente ou fora da organização |
| 409 | `service_offering_name_conflict` | nome normalizado já existe |
| 409 | `service_offering_archived` | tentativa de ativar serviço arquivado |

Constraints conhecidas devem ser traduzidas; SQL e nomes de constraints não
chegam ao cliente.

### Retry

`Idempotency-Key` não entra nesta slice. O POST tem identidade natural por
nome normalizado e constraint única. O frontend bloqueia duplo clique; após
perda de resposta seguida de `409`, recarrega a lista e exibe o item existente.
PATCH e PUT expressam estado desejado e são repetíveis.

## 7. Autorização e fronteiras

- casos de uso permanecem em `internal/app/catalog`;
- domínio e ports do catálogo podem ser ampliados sem criar módulo `services`
  genérico, nome que conflita conceitualmente com ports de infraestrutura;
- consultas filtram simultaneamente ID e organização/unidade;
- operações compostas usam o transaction manager atual;
- GET permite `OWNER` e `COACH`; mutações usam `middleware.Owner()`;
- nenhum módulo de agenda ou plano consulta as novas tabelas nesta slice;
- consumidores futuros devem usar port/contrato do catálogo, não GORM direto.

## 8. Feature flag e rollout

Adicionar `FEATURE_SERVICE_CATALOG_ENABLED`:

- default `true` em development/test e `false` em production;
- só pode estar `true` quando `FEATURE_CATALOG_ENABLED=true`;
- exposta em `GET /api/v1/capabilities` como `service_catalog`;
- campo ausente é tratado como `false` pelo frontend;
- rotas retornam `404 capability_disabled` quando qualquer flag necessária
  estiver desligada.

O flag separado permite publicar frontend novo contra backend antigo sem fazer
requests para endpoints inexistentes, mesmo se `catalog` já estiver habilitado.

Sequência:

1. aplicar migration;
2. publicar backend com `service_catalog` desligado em produção;
3. publicar frontend em qualquer ordem;
4. executar smoke autenticado de listagem e ciclo de vida;
5. habilitar a capability;
6. observar taxa de erro e latência das rotas.

Rollback operacional: desligar `FEATURE_SERVICE_CATALOG_ENABLED`. As tabelas
aditivas permanecem e o catálogo anterior continua funcionando.

## 9. Experiência mobile-first

- manter `#/structure` e o item de navegação existentes;
- adicionar a tab `Serviços` depois de `Modalidades` e `Espaços`, somente com
  `service_catalog === true`;
- resumo da tela passa a mencionar modalidades, espaços e serviços apenas
  quando a capability estiver ativa;
- ação primária da tab: `Adicionar serviço`;
- formulário inicial exibe somente `Nome`;
- abertura `Detalhar` exibe `Descrição` e `Cor`;
- sugestões não persistidas: `Recovery`, `Avaliação física`, `Fisioterapia`,
  `Nutrição` e `Massagem`;
- cards mostram nome, descrição quando houver e estado na unidade;
- arquivar/reativar e ativar/desativar seguem o padrão visual já usado nas
  duas tabs existentes;
- alvos de toque possuem no mínimo 44px; em 360px não há scroll horizontal;
- loading, vazio, erro, conflito e sucesso possuem feedback visível;
- nenhuma tela solicita duração, preço, vagas, profissional ou local.

## 10. Observabilidade

- logs estruturados incluem ação, `organization_id`, `unit_id`, `actor_user_id`
  e `service_offering_id`, sem payload HTTP completo;
- métricas seguem as métricas HTTP já existentes, sem labels por ID/nome;
- auditoria é a fonte durável de mutações;
- falha de auditoria falha a transação, sem recurso criado parcialmente.

## 11. Critérios de aceite

- owner cria um serviço informando somente o nome em viewport móvel;
- serviço nasce ativo na unidade atual;
- owner edita, arquiva, reativa, ativa e desativa o serviço;
- coach consegue listar por API, mas não mutar;
- nomes equivalentes após normalização retornam conflito;
- UUID de outro tenant retorna `404` em leitura e mutação;
- serviço não possui campos de agenda ou preço;
- tabs de modalidade e espaço não regridem;
- frontend novo com backend antigo esconde `Serviços`;
- flag desligada não expõe UI nem rota;
- migration funciona em banco vazio e em upgrade da `054`.

## 12. Testes mínimos

### Backend

- normalização compartilhada de nome, descrição, cor e status;
- ciclo de vida e associação transacional;
- conflito de nome e ativação de arquivado;
- autorização `OWNER`/`COACH`/`PLATFORM_ADMIN`;
- isolamento com duas organizações e UUIDs conhecidos;
- tradução de constraints para códigos estáveis;
- auditoria de todos os comandos e rollback quando ela falha;
- capability desligada e dependência de `catalog` validada;
- migration em banco vazio, upgrade da `054` e migrator idempotente;
- `go test ./...` e `go vet ./...`.

### Frontend

- typecheck/build;
- tab ausente com capability ausente/falsa;
- criação mínima e abertura progressiva;
- edição e ciclo arquivar/reativar/ativar/desativar;
- loading, vazio, erro e conflito;
- Playwright em 360x800 sem overflow horizontal;
- regressão das tabs Modalidades e Espaços.

## 13. Decisões deliberadamente adiadas

- serviço individual/coletivo e duração padrão: `SPEC-006`;
- preço, pacote, adicional e consumo: `SPEC-003`/`SPEC-013`;
- profissional e disponibilidade: `SPEC-006`;
- reserva, fila e check-in: `SPEC-007/008`;
- ativação real em outra unidade: `SPEC-001/C`.

## 14. Evidências de validação

- migration `055_create_service_offerings.sql` validada em banco
  PostgreSQL vazio, no upgrade isolado de `054` e em segunda execução sem
  migrations pendentes;
- constraints compostas, conflito normalizado, transações, rollback por falha
  de auditoria e isolamento entre duas organizações validados com PostgreSQL;
- contratos HTTP validados para owner, leitura por coach, bloqueio de platform
  admin, CSRF, capabilities combinadas e códigos estáveis;
- ciclo criar, editar, arquivar, reativar, ativar e desativar coberto no backend
  e no Playwright, com auditoria de todos os comandos;
- frontend validado em `360x800`, sem overflow horizontal, incluindo abertura
  progressiva, conflito recuperável e compatibilidade com capability ausente;
- suíte Go completa, `go vet`, race detector em container Linux, TypeScript,
  build Vite e suíte Playwright completa aprovados localmente.
