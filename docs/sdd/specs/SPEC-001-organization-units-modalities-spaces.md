# SPEC-001 — Organização, unidades, modalidades e espaços

- Status: `implementing`
- Versão: `0.5`
- Depende de: `ARCH-001`
- Bloqueia: `001/A-B` bloqueiam `SPEC-002`; `001/B-D` bloqueiam `SPEC-003` e
  agenda; `001/C` bloqueia permissões multiunidade

## 1. Resultado esperado

O proprietário consegue representar sua operação real sem configurar uma
estrutura administrativa complexa. Uma academia com uma unidade continua
simples; uma organização com várias unidades consegue compartilhar modalidades
e alternar o contexto operacional com segurança.

## 2. Problema

**Existente:** `Box` é simultaneamente tenant, academia e unidade. Isso atende
o CrossFit Alados atual, mas não representa uma organização com CrossFit,
musculação, spinning, espaços e unidades distintas.

**Benchmark:** Tecnofit anuncia gestão de multiunidades e multimodalidades. Os
produtos observados normalmente exigem cadastros administrativos extensos antes
de gerar valor.

**Inferência:** no EngageFit, a primeira unidade deve nascer pronta. Modalidades
e espaços serão cadastrados somente quando necessários para plano ou agenda.

**Entrada Alados — `REQ-001`:** a operação também vende serviços agendáveis,
como Recovery, fisioterapia, massagem e nutrição. Serviço não será representado
como modalidade ou espaço; uma slice aditiva criará o catálogo necessário antes
de planos e agenda dependerem dele.

## 3. Escopo

### Incluído

- organização com nome e status;
- uma ou mais unidades;
- modalidade reutilizável pela organização;
- ativação de modalidades por unidade;
- espaços físicos por unidade com capacidade padrão;
- seleção segura da unidade no produto;
- compatibilidade com `boxes` e sessões atuais;
- configuração mobile-first.

### Fora do escopo

- franquias, holdings e hierarquias com mais de dois níveis;
- consolidação financeira multiempresa;
- agenda e sessões de aula;
- feriados e horário de funcionamento detalhado;
- catracas e dispositivos físicos;
- permissões finais de gerente, recepção e financeiro.

## 4. Evidências e benchmark

- **Benchmark — Tecnofit:** apresenta multiunidades, multimodalidades, agenda e
  controle de presença como capacidades centrais.
- **Benchmark — Next Fit:** separa modalidade da oferta comercial; Cross
  Training, musculação e pilates são modalidades, enquanto planos determinam
  duração, frequência e preço.
- **Inferência EngageFit:** modalidade é catálogo da organização; espaço e
  capacidade pertencem à unidade; plano referencia modalidades, não textos
  livres duplicados.
- **Inferência EngageFit:** serviço é oferta agendável da organização, ativada
  por unidade; profissional, horário e vaga pertencem à agenda.

Referências:

- https://www.tecnofit.com.br/precos/
- https://ajuda.nextfit.com.br/support/solutions/articles/69000813031-como-criar-um-contrato-geral-

## 5. Regras e invariantes

1. Toda organização possui ao menos uma unidade não arquivada durante a
   operação normal.
2. Cada `box` pertence exatamente a uma organização.
3. O nome de uma unidade é único dentro da organização ignorando caixa e
   espaços externos.
4. Modalidade pertence à organização e pode estar ativa em várias unidades.
5. Espaço pertence a uma única unidade.
6. Capacidade padrão do espaço é positiva quando informada; uma aula pode
   sobrescrevê-la no futuro.
7. Arquivar modalidade ou espaço impede novo uso, sem apagar histórico.
8. Suspender uma unidade preserva dados e bloqueia novas operações conforme a
   política atual de `Box`.
9. Nenhum acesso multiunidade é concedido apenas porque o usuário pertence à
   mesma organização.
10. Timezone pertence à unidade; datas de negócio usam esse timezone.

## 6. Fluxos do usuário

### Primeiro acesso de uma organização existente

1. O owner entra normalmente.
2. O sistema mostra o nome atual da unidade e não exige configuração.
3. Em `Configurações > Estrutura`, o owner pode revisar nome, timezone,
   modalidades e espaços.
4. Até uma segunda unidade existir, nenhum seletor permanente ocupa a tela.

### Adicionar modalidade

1. Owner toca em `Adicionar modalidade`.
2. Informa somente o nome; cor/ícone são opcionais.
3. Marca as unidades onde a modalidade existe, com a atual pré-selecionada.
4. Salva e recebe confirmação contextual.

### Adicionar espaço

1. Dentro de uma unidade, owner toca em `Adicionar espaço`.
2. Informa nome e, opcionalmente, capacidade padrão.
3. O espaço fica disponível para futuras grades de aula.

### Alternar unidade

1. Usuário com acesso a mais de uma unidade abre o seletor no cabeçalho.
2. Escolhe uma unidade permitida.
3. A rota e os dados deixam explícito o novo contexto.
4. A última unidade usada pode ser lembrada pelo frontend, mas é revalidada no
   backend em toda sessão.

## 7. Experiência

- Tela `Estrutura` abre com resumo da organização e cards das unidades.
- No celular, a ação primária é `Adicionar` e abre escolha entre modalidade,
  espaço e, quando permitido, unidade.
- Formulário de unidade exige nome e timezone; endereço, documento e contatos
  ficam em `Mais informações` e só se tornam obrigatórios quando outra spec os
  utilizar.
- Modalidades sugeridas (`CrossFit`, `Musculação`, `Spinning`) aceleram o
  cadastro, mas texto livre é permitido.
- Não expor `organization_id`, `box_id` ou termos de tenancy na interface.
- Estado vazio explica o benefício imediato: modalidades serão usadas em
  planos e agenda; espaços controlarão capacidade.

## 8. Modelo de dados proposto

### `organizations`

- `id`
- `name`
- `status`: `active`, `suspended`, `archived`
- timestamps e auditoria de mudança de status

### Alteração aditiva em `boxes`

- `organization_id` obrigatório após backfill;
- `timezone` com default temporário da configuração atual;
- opcionalmente `display_name`, se necessário sem substituir `name`.

`boxes` continua sendo a tabela física de unidades durante esta fase.

### `modalities`

- `id`, `organization_id`, `name`, `color`, `status`, timestamps;
- unicidade normalizada de nome dentro da organização.

### `unit_modalities`

- `unit_id`, `modality_id`, `active`, timestamps;
- chave única por unidade/modalidade.

### `spaces`

- `id`, `unit_id`, `name`, `default_capacity`, `status`, timestamps;
- unicidade normalizada de nome dentro da unidade.

### Acesso do usuário

A evolução usará `organization_user_memberships` para o papel na organização e
`user_unit_access` para a allowlist de unidades. Elas entram na Slice `001-C`.
Até lá, `users.box_id` permanece como contexto principal e concede acesso
somente à unidade já associada ao usuário.

## 9. Contratos HTTP propostos

- `GET /api/v1/organization`
- `PATCH /api/v1/organization`
- `GET /api/v1/units`
- `POST /api/v1/units`
- `GET /api/v1/units/:unit_id`
- `PATCH /api/v1/units/:unit_id`
- `GET/POST /api/v1/modalities`
- `PATCH /api/v1/modalities/:id`
- `PUT /api/v1/units/:unit_id/modalities/:modality_id`
- `GET/POST /api/v1/units/:unit_id/spaces`
- `PATCH /api/v1/units/:unit_id/spaces/:id`

Criação e alteração aceitam `Idempotency-Key` quando puderem ser repetidas por
cliente móvel após perda de conexão. Erros de nome duplicado retornam `409` com
código estável.

## 10. Autorização e auditoria

- owner pode alterar organização e estrutura;
- coach pode consultar somente unidades/modalidades/espaços permitidos;
- criação, suspensão e arquivamento geram evento administrativo com autor,
  motivo quando destrutivo e valores não sensíveis;
- platform admin não recebe acesso operacional implícito aos dados do aluno;
- toda rota com `unit_id` valida organização e allowlist do usuário.

## 11. Rollout e compatibilidade

1. Criar uma organização para cada `box` existente.
2. Vincular o `box` à organização em backfill transacional.
3. Preservar login e rotas atuais usando a unidade original.
4. Ativar a nova tela somente após reconciliação `boxes sem organização = 0`.
5. Não renomear tabelas nem remover `users.box_id` neste rollout.

Para o CrossFit Alados, o resultado inicial será uma organização e uma unidade
com o mesmo nome. Ajustes de nomenclatura são feitos depois pelo owner.

## 12. Critérios de aceite

- banco existente recebe organização sem perda ou mudança de tenant;
- owner de uma unidade continua usando todos os fluxos atuais;
- owner multiunidade vê somente unidades autorizadas;
- modalidade pode ser compartilhada entre duas unidades;
- espaço nunca aparece em unidade de outra organização;
- unidade arquivada não aceita nova configuração operacional;
- interface de uma única unidade não exibe complexidade multiunidade;
- fluxo completo funciona em viewport móvel sem rolagem horizontal.

## 13. Testes mínimos

- migrations em banco vazio, backfill e segunda execução do migrator;
- isolamento entre duas organizações com IDs conhecidos;
- allowlist entre duas unidades da mesma organização;
- nomes duplicados normalizados;
- suspensão/arquivamento sem exclusão histórica;
- API e fluxo Playwright mobile de modalidade e espaço;
- regressão de login, dashboard, alunos, check-ins e app do atleta atuais.

## 14. Decisões fechadas para implementação

- `organization_user_memberships` registra papel na organização e
  `user_unit_access` limita as unidades permitidas;
- platform admin e owner da organização poderão criar unidade; a ação do owner
  poderá ser condicionada ao plano SaaS em evolução posterior;
- CNPJ, endereço completo e dados financeiros não entram nesta spec. Serão
  solicitados apenas quando contratos, emissão ou pagamentos os exigirem;
- a primeira implementação será dividida nas slices abaixo.

### Slice `001-A` — Fundação compatível

Status: `validated locally` em 2026-08-11.

- criar `organizations`;
- adicionar `boxes.organization_id`;
- criar uma organização determinística para cada box existente;
- garantir que novos boxes também nasçam com organização;
- mapear `organization_id` no domínio e persistência;
- preservar login, rotas, autorização e interface atuais;
- não criar endpoints nem frontend novos;
- validar backfill, integridade e criação transacional de box + owner.

Evidências: migrations `001–053` em banco vazio, segunda execução sem migrations
pendentes e teste PostgreSQL automatizado do backfill transacional sobre schema
`052` populado. A suíte também cobre criação/rollback atômico, insert do binário
legado e isolamento atual por `box_id`. A validação de CI permanece necessária
antes do merge.

### Slice `001-B` — Modalidades e espaços

Status: `validated locally` em 2026-08-12.

- implementar catálogo, APIs e configuração mobile;
- preservar a experiência de uma unidade sem seletor;
- contrato executivo detalhado em
  [`SPEC-001-B-modalities-spaces.md`](SPEC-001-B-modalities-spaces.md).

### Slice `001-C` — Acesso e contexto multiunidade

- implementar memberships de usuário, allowlist e seleção de unidade;
- testar duas unidades da mesma organização e duas organizações distintas.

Esta slice pode ser adiada enquanto o piloto operar uma única unidade. O
backend continua usando `users.box_id` como contexto autorizado até sua
implementação.

### Slice `001-D` — Catálogo de serviços ofertáveis

Status: [`ready`](SPEC-001-D-service-offerings.md) em 2026-08-18,
incorporada a partir de `REQ-001`.

- distinguir serviço de modalidade, espaço e sessão de agenda;
- permitir nome, descrição curta e arquivamento sem exigir preço ou horário;
- ativar serviços por unidade sem duplicar o catálogo da organização;
- deixar profissional, duração, preço, capacidade e recorrência para seus
  módulos proprietários;
- fornecer referências estáveis para direitos de plano e agenda;
- executar pelos contratos, tenancy, UX e rollout da spec detalhada.

As letras identificam slices, não uma ordem obrigatória. `001-D` pode ser
detalhada e executada antes de `001-C`, desde que permaneça limitada à unidade
autorizada por `users.box_id`.
