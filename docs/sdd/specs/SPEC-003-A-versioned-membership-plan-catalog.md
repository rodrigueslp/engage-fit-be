# SPEC-003/A — Catálogo versionado de planos

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-08-18
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-002/A`, `SPEC-003`
- Ownership: módulo novo `memberships`
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`
- Migration: `057_create_membership_plan_catalog.sql` (número provisório)

## 1. Resultado esperado

O owner cria e publica pelo celular um plano comercial como `CrossFit Livre`,
`Pilates 2x por semana` ou `Pacote de 10 aulas`. O fluxo exige somente nome,
preço, forma de cobrança e o que o plano permite. Limites de frequência ficam
em abertura progressiva e a revisão usa linguagem natural.

Uma versão publicada nunca é alterada. Quando o owner muda preço ou direitos,
o sistema clona a versão atual para um novo rascunho e publica uma nova versão
para vendas futuras. Esta slice não vende o plano para uma pessoa.

## 2. Evidências e decisões de produto

- **Existente:** modalidades, serviços e ativação por unidade já pertencem ao
  módulo `catalog`; planos devem referenciá-los por ID estável.
- **Existente:** `billing_plans` cobra o EngageFit da academia e não pode ser
  reutilizado para planos vendidos aos alunos.
- **Benchmark — Next Fit:** plano/contrato reúne modalidade, duração, preço e
  acesso padrão, sessões por semana ou pacote de aulas. A ferramenta expõe
  muitas restrições e decisões financeiras no mesmo cadastro.
- **Benchmark — Next Fit:** alterações em modalidades e frequência podem
  atingir contratos vigentes. O EngageFit evita esse efeito com versões
  imutáveis.
- **Benchmark — Tecnofit:** contratos representam mensalidades, pacotes e
  planos recorrentes; a venda acontece depois, no cadastro do aluno.
- **Benchmark — Tecnofit:** uma aula avulsa pode ser representada por contrato
  com quantidade de sessões e validade, mas o EngageFit preserva também o
  conceito separado de serviço ofertável.
- **Inferência EngageFit:** o cadastro padrão não mostra parcelamento,
  fidelidade, multa, templates jurídicos, horários ou gateway. Esses dados
  pertencem a contratos, financeiro e agenda.
- **Inferência EngageFit:** preço zero é ambíguo entre cortesia, bolsa, teste e
  falha de cadastro. Esta slice exige preço positivo; gratuidade será decidida
  junto da composição comercial.
- **Entrada Alados:** a fundação precisa representar mensalidade livre,
  Pilates com frequência semanal e pacote de aulas sem criar um tipo de plano
  diferente para cada modalidade.

Referências oficiais consultadas:

- https://ajuda.nextfit.com.br/support/solutions/articles/69000813031-como-criar-um-contrato-geral-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000851957-quais-s%C3%A3o-os-impactos-na-altera%C3%A7%C3%A3o-de-contratos-
- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000699531-como-efetuar-vendas-no-cadastro-do-aluno-contrato-servico-e-produto-
- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000415354-como-criar-um-contrato-de-aula-unica-para-agenda-de-turmas

## 3. Escopo

### Incluído

- catálogo de planos por organização;
- versão 1 em rascunho, publicação e clonagem para nova versão;
- preço comercial de referência em BRL;
- cobrança de referência única, mensal, trimestral, semestral ou anual;
- validade para planos de pagamento único;
- direitos base sobre modalidades e serviços ativos na unidade atual;
- acesso livre, sessões por período e pacote de aulas;
- resumo determinístico em linguagem natural;
- arquivamento e reativação do plano;
- auditoria transacional, idempotência e controle otimista;
- capability `memberships` desligada por padrão;
- página mobile-first `Planos` para owner.

### Fora do escopo

- adicionais, desconto, cupom, cortesia e preço zero (`SPEC-003/B`);
- matrícula, venda para pessoa, vigência concreta, pausa ou encerramento
  (`SPEC-003/C`);
- avaliação de acesso, saldo, consumo ou compatibilidade de check-in
  (`SPEC-003/D`);
- documento, assinatura, PAR-Q e cancelamento (`SPEC-004`);
- cobrança, parcelamento, Pix, boleto, cartão ou Asaas (`SPEC-005`);
- horário, profissional, capacidade, turma ou sessão (`SPEC-006`);
- reserva, reagendamento, antecipação, lista de espera ou no-show
  (`SPEC-007`);
- plano corporativo fictício para Wellhub ou TotalPass;
- multiunidade e seletor de unidade (`SPEC-001/C`);
- edição do catálogo de modalidades ou serviços dentro do formulário de plano.

## 4. Vocabulário

- **Plano:** identidade comercial, por exemplo `CrossFit Livre`.
- **Rascunho:** versão editável e ainda indisponível para matrícula.
- **Versão publicada:** fotografia imutável dos termos comerciais e direitos.
- **Direito base:** permissão versionada ligada a uma modalidade ou serviço.
- **Cobrança de referência:** periodicidade apresentada comercialmente; não
  gera título financeiro.
- **Validade:** tempo de uso previsto para um plano de pagamento único; não é
  uma matrícula concreta.

Na interface, usar `plano`, `publicar` e `nova versão`. Não usar `SKU`,
`entitlement`, `billing cycle` ou `mutation`.

## 5. Experiência mobile

### Lista

A navegação do owner recebe `Planos`, condicionada à capability
`memberships`. A página mostra:

- nome e status `Rascunho`, `Publicado` ou `Arquivado`;
- preço e cobrança em uma linha;
- resumo dos direitos em até duas linhas;
- versão publicada atual e indicação de rascunho pendente;
- uma ação primária coerente com o estado: `Continuar`, `Nova versão` ou
  `Reativar`.

Planos arquivados ficam ocultos por padrão e aparecem ao marcar `Mostrar
arquivados`. A lista usa cards no celular, não tabela comprimida.

### Criar plano

1. Owner toca em `Novo plano`.
2. Informa nome, preço e `Como é cobrado`.
3. Se escolher `Pagamento único`, informa quantidade e unidade de validade.
4. Seleciona ao menos uma modalidade ou serviço já ativo na unidade.
5. Cada item começa com o padrão `Acesso livre`.
6. `Limitar frequência` abre sessões por período ou pacote de aulas.
7. O sistema mostra uma revisão em linguagem natural.
8. Owner salva o rascunho ou toca em `Publicar plano`.

Descrição e regras avançadas ficam recolhidas em `Mais informações`. O
formulário não cria modalidade ou serviço e oferece link para `Estrutura`
quando o catálogo estiver vazio.

### Alterar plano publicado

1. Owner abre um plano publicado.
2. Toca em `Criar nova versão`.
3. O sistema clona os termos atuais para a próxima versão em rascunho.
4. Owner altera o rascunho e revisa um resumo das diferenças.
5. Ao publicar, a versão anterior permanece intacta e a nova vira a versão
   comercial atual.

Não existe botão `Editar` diretamente sobre uma versão publicada.

## 6. Regras e invariantes

1. Plano pertence a exatamente uma organização e registra a unidade de origem.
2. O contexto autorizado continua sendo a unidade `users.box_id`; nesta slice,
   somente a unidade de origem consulta e opera o plano.
3. Nome normalizado é único na organização, inclusive para plano arquivado.
4. Um plano possui no máximo um rascunho.
5. Números de versão começam em 1 e crescem sem lacunas por plano.
6. Uma versão publicada é imutável no banco e na aplicação.
7. Publicação exige preço positivo e ao menos um direito base válido.
8. Moeda é `BRL`; a API recebe e retorna centavos inteiros.
9. `billing_cycle` aceita `one_time`, `monthly`, `quarterly`, `semiannual` ou
   `annual`.
10. `one_time` exige validade; ciclos recorrentes não aceitam validade nesta
    slice.
11. Validade aceita 1–3650 dias, 1–520 semanas, 1–120 meses ou 1–10 anos.
12. Cada direito referencia exatamente uma modalidade ou um serviço.
13. O alvo precisa pertencer à organização, estar ativo e estar habilitado na
    unidade atual no momento da publicação.
14. Arquivar/desativar depois um item de catálogo não reescreve versões já
    publicadas; a UI mostra aviso e versões futuras precisam corrigir o alvo.
15. `open_access` não possui quantidade nem período; pode ter limite diário.
16. `sessions_per_period` exige quantidade e período `day`, `week` ou `month`.
17. `class_pack` exige quantidade total e não aceita período.
18. Quantidade e limite diário ficam entre 1 e 10000.
19. `delivery_mode` é opcional e aceita `group` ou `individual`.
20. `booking_required` é apenas um termo do direito; não cria agenda ou
    reserva.
21. Criar plano, publicar e criar nova versão são idempotentes.
22. Atualizar rascunho exige revisão esperada para impedir sobrescrita entre
    abas.
23. Arquivamento nunca apaga plano, versão, direito ou auditoria.
24. Nenhuma operação desta slice cria matrícula, cobrança ou efeito externo.

## 7. Modelo de dados e migration

A migration provisória seguinte à `056` é aditiva e não altera
`billing_plans`, `students`, `persons`, check-ins ou catálogo.

### `membership_plans`

Campos:

| Campo | Regra |
|---|---|
| `id` | UUID primário |
| `organization_id` | FK `organizations`, obrigatório |
| `home_unit_id` | FK composta à unidade da sessão que criou o plano |
| `name`, `name_key` | 1–100 caracteres, normalização do catálogo |
| `status` | `active` ou `archived` |
| `created_at`, `updated_at` | timestamptz |

Constraints mínimas:

- unique `(organization_id, name_key)`;
- unique `(id, organization_id)`;
- FK `(home_unit_id, organization_id)` para `boxes`;
- status e tamanhos validados no PostgreSQL.

`name` representa a identidade comercial atual. Na versão 1 em rascunho, sua
edição atualiza também o plano. Em versões posteriores, o plano mantém o nome
publicado até o novo rascunho ser publicado. Cada versão preserva o próprio
nome.

### `membership_plan_versions`

Campos:

| Campo | Regra |
|---|---|
| `id`, `organization_id`, `plan_id` | UUIDs e FK composta ao plano |
| `version_number` | inteiro positivo |
| `revision` | inteiro positivo para controle otimista do rascunho |
| `status` | `draft`, `published` ou `retired` |
| `name` | fotografia de 1–100 caracteres |
| `description` | até 500 caracteres, vazia por padrão |
| `price_cents` | inteiro positivo |
| `currency` | sempre `BRL` |
| `billing_cycle` | enum definido na seção 6 |
| `validity_count`, `validity_unit` | exigidos somente para `one_time` |
| `published_at` | obrigatório apenas quando publicada |
| `created_at`, `updated_at` | timestamptz |

Constraints e índices:

- unique `(plan_id, version_number)`;
- unique `(id, organization_id)`;
- índice único parcial de um `draft` por plano;
- checks cruzados entre ciclo e validade;
- checks de preço, moeda, tamanho e status.

Publicar a nova versão marca a versão publicada anterior como `retired`. Isso
não significa exclusão nem invalidação de matrículas futuras que a referenciem.

### `membership_plan_entitlements`

Campos:

| Campo | Regra |
|---|---|
| `id`, `organization_id`, `plan_version_id` | UUIDs e FK composta à versão |
| `modality_id` | nullable, FK composta ao catálogo |
| `service_offering_id` | nullable, FK composta ao catálogo |
| `access_type` | `open_access`, `sessions_per_period`, `class_pack` |
| `quantity` | conforme tipo de acesso |
| `period_unit` | `day`, `week`, `month` quando aplicável |
| `daily_limit` | opcional |
| `delivery_mode` | nullable, `group` ou `individual` |
| `booking_required` | boolean, false por padrão |
| `position` | inteiro não negativo para ordem da UI |
| timestamps | criação/alteração |

Uma check constraint exige exatamente um alvo. Outras constraints validam as
combinações de tipo, quantidade e período. FKs compostas impedem referência a
catálogo de outra organização.

### `membership_plan_command_requests`

Registra idempotência para `create`, `publish` e `new_version`:

- `organization_id`, `command`, `idempotency_key` formam a unicidade;
- fingerprint SHA-256 do payload canônico, que não contém segredo nem PII;
- `plan_id`, `plan_version_id` e resposta estável;
- chave entre 8 e 128 caracteres em `[A-Za-z0-9._:-]`;
- mesma chave/payload retorna o resultado anterior;
- mesma chave/payload diferente retorna `409 idempotency_key_reused`.

### `membership_plan_audit_events`

Campos mínimos: `id`, `organization_id`, `unit_id`, `actor_user_id`, `action`,
`plan_id`, `plan_version_id` nullable, `before_data`, `after_data` e
`created_at`.

Eventos:

- `membership_plan.draft_created`;
- `membership_plan.draft_updated`;
- `membership_plan.published`;
- `membership_plan.new_version_created`;
- `membership_plan.archived`;
- `membership_plan.reactivated`.

Auditoria e mudança pertencem à mesma transação. Não há PII nesta slice.

### Imutabilidade no banco

Triggers impedem `UPDATE`/`DELETE` de versão publicada ou retirada e
`INSERT`/`UPDATE`/`DELETE` de seus direitos. A única transição permitida em uma
versão publicada é `published -> retired`, executada pela publicação da versão
seguinte.

Não há backfill nem seed. Dados atuais continuam intactos.

## 8. Contratos internos e fronteiras

O módulo `memberships` segue domínio → aplicação → ports → adapters e não
importa GORM models ou repository concreto de `catalog`.

Contrato público mínimo do catálogo:

```text
ResolvePlanReferences(organization_id, unit_id, modality_ids, service_ids)
  -> referências com tipo, nome, status organizacional e ativação na unidade
```

O adapter pode usar PostgreSQL, mas o contrato e os testes pertencem à
fronteira pública do catálogo. A persistência de memberships mantém FKs
compostas como defesa adicional de tenancy.

O resumo textual é função pura do domínio e não chama IA:

```text
CrossFit Livre · acesso livre · R$ 189/mês
Pilates · 2 sessões por semana · R$ 260/mês
Pilates · pacote de 10 aulas por 90 dias · R$ 500 pagamento único
```

## 9. Contratos HTTP

Todas as rotas ficam sob `/api/v1`, exigem sessão owner, CSRF nos comandos,
tenant da sessão e capability `memberships`.

### Listar

`GET /membership-plans?include_archived=false`

Retorna uma lista resumida com plano, estado, versão publicada atual, rascunho
pendente, preço, resumo e alertas de referências indisponíveis.

### Detalhar

`GET /membership-plans/:id`

Retorna a versão publicada atual e o rascunho, se existir. Histórico completo
de versões pode ficar nesta resposta; não criar endpoint genérico de auditoria.

### Criar plano e versão 1 em rascunho

`POST /membership-plans`

Header obrigatório: `Idempotency-Key`.

```json
{
  "name": "Pilates 2x",
  "description": "",
  "price_cents": 26000,
  "billing_cycle": "monthly",
  "validity_count": null,
  "validity_unit": null,
  "entitlements": [
    {
      "modality_id": "uuid",
      "service_offering_id": null,
      "access_type": "sessions_per_period",
      "quantity": 2,
      "period_unit": "week",
      "daily_limit": 1,
      "delivery_mode": "group",
      "booking_required": true
    }
  ]
}
```

Cria plano e versão 1 `draft` na mesma transação.

### Atualizar rascunho

`PATCH /membership-plans/:id/draft`

Recebe o payload completo editável e `expected_revision`. A lista de direitos
substitui atomicamente a lista anterior. Sucesso incrementa `revision`.

### Publicar

`POST /membership-plans/:id/publish`

Header obrigatório: `Idempotency-Key`. O payload informa `draft_revision`.
Revalida referências, regras, preço e concorrência antes de publicar.

### Criar nova versão

`POST /membership-plans/:id/versions`

Header obrigatório: `Idempotency-Key`. Clona a versão publicada atual para um
rascunho de número seguinte. Se já houver rascunho, retorna o existente apenas
quando a mesma chave representar o mesmo comando; caso contrário retorna
`409 membership_plan_draft_exists`.

### Arquivar e reativar

- `POST /membership-plans/:id/archive`
- `POST /membership-plans/:id/reactivate`

Arquivar plano com rascunho é permitido, mas o rascunho não pode ser publicado
até a reativação.

## 10. Erros estáveis

| Código | HTTP | Uso |
|---|---:|---|
| `membership_plan_not_found` | 404 | plano inexistente ou outro tenant |
| `membership_plan_name_conflict` | 409 | nome normalizado já usado |
| `membership_plan_invalid` | 422 | combinação de campos inválida |
| `membership_plan_target_invalid` | 422 | modalidade/serviço inválido ou indisponível |
| `membership_plan_draft_not_found` | 409 | comando exige rascunho |
| `membership_plan_draft_exists` | 409 | nova versão concorrente |
| `membership_plan_revision_conflict` | 409 | rascunho alterado em outra aba |
| `membership_plan_already_published` | 409 | publicação inválida |
| `membership_plan_archived` | 409 | publicação em plano arquivado |
| `idempotency_key_reused` | 409 | chave reaproveitada com outro comando/payload |

Validação de campo retorna detalhes seguros no envelope atual e sempre inclui
`request_id`. IDs de outro tenant respondem `404`.

## 11. Autorização e tenancy

- somente `OWNER` lista preço, cria, altera, publica, arquiva e reativa;
- `COACH` não acessa estas rotas comerciais; a leitura operacional futura
  virá de `SPEC-003/D` sem preço;
- `PLATFORM_ADMIN` não herda acesso a dados comerciais da academia;
- `organization_id` e `unit_id` nunca vêm do corpo;
- toda query filtra organização derivada da unidade autenticada;
- referências adversárias de outra organização ou unidade retornam erro seguro;
- testes usam duas organizações e pelo menos duas unidades.

## 12. Capability e rollout

Adicionar `FEATURE_MEMBERSHIPS_ENABLED`, padrão `false` em development e
production. `GET /capabilities` expõe somente `memberships: boolean`.

Ordem compatível:

1. aplicar a migration com a flag desligada;
2. publicar backend com a flag desligada;
3. publicar frontend, que trata capability ausente como `false`;
4. executar smoke autenticado;
5. habilitar em ambiente de homologação;
6. homologar criação, publicação, nova versão e arquivamento;
7. habilitar produção somente por decisão separada.

Rollback operacional desliga a flag. As tabelas aditivas permanecem e nenhum
fluxo legado depende delas. A API não executa migration no startup.

## 13. Observabilidade, auditoria e privacidade

- logs estruturados usam ação, IDs, organização/unidade, versão, resultado e
  `request_id`;
- não registrar payload integral, nome/descrição livre ou corpo da auditoria
  em logs;
- métricas mínimas: comandos por resultado, conflitos, publicação e latência;
- não existe efeito externo, segredo ou PII nesta slice;
- descrição do plano ainda é conteúdo do tenant e deve respeitar limites;
- auditoria preserva preço e direitos por serem termos comerciais, não dados
  pessoais.

## 14. Critérios de aceite

- owner cria rascunho com poucos campos em `360x800`, sem overflow horizontal;
- plano simples mensal exige nome, preço e ao menos um item do catálogo;
- modalidades e serviços ativos aparecem com distinção visual clara;
- `Acesso livre` funciona sem abrir configurações avançadas;
- Pilates 2x semanal e pacote de 10 aulas são representáveis;
- resumo textual corresponde deterministicamente ao payload;
- publicação torna versão e direitos imutáveis no PostgreSQL;
- nova versão clona a anterior sem alterá-la;
- primeira versão mantém preço/direitos depois da publicação da segunda;
- concorrência de edição não perde alterações silenciosamente;
- comandos idempotentes repetidos retornam o mesmo resultado;
- plano de outra organização e alvo de outra unidade não vazam existência;
- coach e platform admin não veem preço nem acessam a página;
- capability ausente ou falsa esconde navegação e bloqueia API;
- arquivamento e reativação preservam histórico;
- `billing_plans`, students, check-ins e fluxos atuais não mudam.

## 15. Testes obrigatórios

### Backend

- normalização e resumo textual no domínio;
- matriz completa de combinações válidas/inválidas de direitos;
- transações, rollback de auditoria, idempotência e revisão otimista;
- publicação, retirada da versão anterior e clonagem;
- trigger de imutabilidade para versão e direitos;
- unicidade e concorrência de número de versão;
- isolamento entre organizações/unidades e referências adversárias;
- owner permitido; coach/platform admin bloqueados;
- capability, CSRF, erros e envelopes HTTP;
- migration em PostgreSQL vazio, segundo run com zero aplicações;
- upgrade real a partir da `056` com dados de catálogo e pessoas;
- regressão PostgreSQL completa, `go test`, `go vet` e race detector Linux.

### Frontend

- capability ausente/falsa esconde `Planos`;
- empty state leva ao cadastro;
- criação mínima e abertura progressiva;
- seleção de modalidade e serviço;
- acesso livre, sessões semanais e pacote;
- máscara monetária sem perda de centavos;
- erro recuperável, revisão de publicação e conflito de revisão;
- nova versão, arquivamento, filtro de arquivados e reativação;
- Playwright focado em `360x800` e regressão completa;
- TypeScript, build Vite e `git diff --check`.

## 16. Decisões deliberadamente adiadas

- adicionais compartilhados ou por versão: `SPEC-003/B`;
- cortesia, bolsa, desconto, remoção de adicional e preço zero: `SPEC-003/B`;
- uma ou várias matrículas simultâneas: `SPEC-003/C`;
- status inicial antes de contrato/pagamento: `SPEC-003/C`;
- pausa, trancamento e deslocamento de vigência: `SPEC-003/C`/`SPEC-004`;
- fonte de verdade de consumo e estorno: `SPEC-003/D`/`SPEC-007`;
- situação financeira e bloqueio externo: `SPEC-005`;
- horários promocionais, reagendamento e antecipação: `SPEC-006/007`;
- forma comercial definitiva de Recovery: `SPEC-003/B` com validação Alados.

Nenhuma dessas decisões impede o catálogo versionado porque a versão preserva
os termos base e os pontos de extensão sem inventar comportamento operacional.

## 17. Definition of Done da slice

- implementação vertical backend/frontend em commits separados;
- nenhuma divergência silenciosa desta spec;
- migration provisória declarada ao integrador;
- todos os testes da seção 15 executados ou impedimentos registrados;
- revisão explícita de tenancy, imutabilidade, idempotência e rollout;
- revisão visual mobile e ausência de overflow;
- `docs/system-design.md`, status da spec e evidências atualizados somente após
  implementação e validação;
- nenhum push, merge, deploy, migration de produção ou habilitação de flag pelo
  agente executor.

## 18. Evidências da execução local

Validação concluída em 2026-08-18 nos worktrees dedicados da SPEC-003/A:

- migration completa em PostgreSQL temporário vazio: 57 arquivos aplicados;
- segundo `engagefit-migrate up` no mesmo banco: `0 applied`;
- teste de upgrade aplica `001`–`056`, semeia catálogo e people, aplica apenas
  `057`, confirma as cinco tabelas vazias e remove o banco temporário;
- tentativas SQL diretas de alterar ou excluir versão publicada/retirada e seus
  direitos foram rejeitadas pelos triggers; `published -> retired` foi aceita;
- `TEST_DATABASE_URL=... go test ./...`: passou, incluindo integração real,
  tenant adversário, rollback de auditoria, idempotência/replay estável,
  conflito de revisão, publicação e clonagem;
- `go vet ./...`: passou;
- `go test -race -count=1 ./...`: passou em `golang:1.25-bookworm`, pois o host
  local não possui compilador C;
- TypeScript e build Vite: passaram;
- Playwright focado: 4 testes passaram; regressão: 25 passaram e 2 fluxos reais
  opt-in preexistentes foram ignorados por não haver stack E2E real habilitada;
- revisão visual em Chromium `360x800`: cards e ações legíveis, sem overflow
  horizontal;
- capability permanece desligada por padrão e nenhuma ação externa é criada.

Não houve divergência de contrato. A migration `057` continua provisória para
confirmação ou renumeração pelo integrador.
