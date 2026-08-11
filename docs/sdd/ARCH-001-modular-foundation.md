# ARCH-001 — Fundação modular do EngageFit Gestão

- Status: `reviewed`
- Versão: `0.2`
- Data: 2026-08-11
- Escopo: backend, frontend, PostgreSQL e integrações

## 1. Decisão executiva

O EngageFit continuará com um repositório de backend, um repositório de
frontend, uma API principal e um PostgreSQL. O backend evoluirá como monólito
modular; não serão criados microserviços ou novos repositórios para os módulos
de gestão neste ciclo.

`boxes` continuará sendo a unidade física e o limite de compatibilidade do
sistema atual. Uma organização será adicionada acima das unidades. O termo
`box` poderá permanecer internamente durante a transição, enquanto produto e
novas APIs usam `organização` e `unidade`.

A identidade humana, as identidades de origem e o vínculo comercial serão
separados:

```text
Person
├── participa de uma ou mais Unit
├── pode autenticar por AthleteAccount
├── possui zero ou mais ExternalIdentity
└── possui zero ou mais Enrollment
```

Essa separação permite que uma pessoa seja mensalista, Wellhub e TotalPass sem
duplicar seu histórico humano nem realizar merge silencioso.

## 2. Contexto atual

**Existente:** `Box` é o tenant e quase todas as tabelas operacionais usam
`box_id`. `Student` representa ao mesmo tempo o aluno na unidade e sua origem
(`wellhub`, `totalpass` ou `box_member`). `AthleteAccount` é uma conta global
ligada aos registros de `Student` por convites e memberships do aplicativo.

**Existente:** o backend já segue ports and adapters, possui migrations
versionadas, isolamento por `box_id`, feature flags, criptografia de segredos,
auditoria e testes PostgreSQL.

**Risco:** substituir essas estruturas em uma única migration colocaria
retenção, campanhas, check-ins, privacidade e o app do aluno em risco. A
evolução será aditiva e com período de convivência.

## 3. Objetivos arquiteturais

- permitir organização com uma ou mais unidades;
- preservar todos os fluxos atuais durante a evolução;
- dar propriedade clara de dados a cada módulo;
- permitir agentes paralelos sem contratos implícitos;
- manter transações locais simples para matrícula, acesso e pagamento;
- deixar integrações externas substituíveis e testáveis;
- suportar frontend mobile-first sem criar uma segunda aplicação de gestão;
- permitir extração futura de módulos sem pagar agora o custo de microserviços.

## 4. Não objetivos

- renomear imediatamente todas as ocorrências de `box_id`;
- criar broker, event streaming, Redis ou service mesh;
- criar banco por organização;
- criar um backend separado para o aplicativo do aluno;
- generalizar o produto para qualquer setor fora de negócios fitness;
- desenhar desde já estoque, POS, avaliações físicas ou comissões avançadas.

## 5. Módulos e propriedade

| Módulo | Responsabilidade | Dados principais |
|---|---|---|
| `organizations` | organização, unidade e contexto selecionado | `organizations`, `boxes` |
| `catalog` | modalidades e espaços | modalidades, vínculos com unidade, espaços |
| `people` | pessoa, identificadores, contato e participação na unidade | pessoas e relacionamentos |
| `memberships` | planos, matrículas e direitos de acesso | planos, versões, matrículas |
| `contracts` | documentos, versões, aceite, trancamento e cancelamento | contratos e eventos |
| `memberbilling` | cobrança da academia para o aluno | clientes financeiros, cobranças e conciliação |
| `scheduling` | grade, aula, capacidade e profissionais | séries e sessões de aula |
| `reservations` | reserva, lista de espera, cancelamento e no-show | reservas e eventos |
| `attendance` | presença e validação de acesso | check-ins e sessões públicas |
| `integrations` | conectores Wellhub, TotalPass, acesso e pagamentos | configurações, inbox e reconciliação |
| `retention` | sinais, filas, intervenções e resultados | radar e intervenções |
| `athlete` | autenticação e experiência do aluno | conta, sessão, perfil e resultados |
| `billing` | cobrança do EngageFit para a academia | planos SaaS, assinaturas e invoices |

`billing` e `memberbilling` são contextos diferentes mesmo quando usam o mesmo
provedor. Tipos, rotas, tabelas e permissões não podem ser compartilhados por
conveniência.

## 6. Dependências permitidas

```text
organizations ─┬─> catalog
               ├─> people
               └─> permissões

people + catalog ─> memberships ─┬─> contracts
                                 ├─> memberbilling
                                 ├─> scheduling/reservations
                                 └─> attendance

attendance + reservations + memberbilling ─> retention
athlete consome contratos públicos dos módulos operacionais
integrations adapta provedores aos contratos de scheduling e attendance
```

Um módulo pode chamar um caso de uso ou port público de outro módulo. Não pode
importar repositories concretos, models GORM ou DTOs HTTP pertencentes ao
outro contexto.

Operações que exigem consistência imediata e pertencem ao mesmo processo usam
transação PostgreSQL. Efeitos externos são executados depois de persistir um
claim idempotente e um estado auditável. Não será introduzido broker nesta fase.

## 7. Organização e tenancy

### Decisões

1. `Organization` é o limite comercial e de identidade da academia/rede.
2. `Box` passa a representar uma `Unit` física durante a compatibilidade.
3. Dados de agenda, presença e operação continuam pertencendo a uma unidade.
4. Pessoa e catálogo reutilizável podem pertencer à organização.
5. Toda consulta de unidade continua filtrando explicitamente `box_id`.
6. Acesso multiunidade exige vínculo explícito do usuário com organização e
   unidades permitidas; conhecer um UUID não concede acesso.

As rotas atuais continuam inferindo `box_id` da sessão. Novas rotas
multiunidade devem explicitar a unidade no caminho, por exemplo
`/api/v1/units/:unit_id/people`. O middleware valida o acesso antes do handler.
Isso evita um cabeçalho implícito e mantém logs e testes mais legíveis.

## 8. Identidade

`Person` representa o ser humano. `Student` será mantido temporariamente como
perfil de origem/frequência da unidade e receberá referência para `Person`.

Regras:

- CPF exato pode identificar uma pessoa dentro da organização;
- identificador estável do provedor identifica somente dentro de
  `(organization, provider)` ou do escopo exigido pelo provedor;
- telefone e nome produzem sugestão/revisão, não merge global automático;
- import sem identificador estável preserva o isolamento atual por unidade e
  origem;
- uma conta do aluno pode ser vinculada à pessoa sem depender de um contrato;
- exclusão e exportação de privacidade percorrem pessoa e perfis vinculados.

## 9. API e compatibilidade

- contratos novos usam recursos e nomes do novo domínio;
- endpoints atuais permanecem durante o rollout;
- adapters de compatibilidade podem montar a resposta antiga a partir do novo
  modelo;
- frontend e backend devem poder ser publicados em qualquer ordem dentro da
  janela documentada;
- erros seguem o envelope atual com código estável e `request_id`;
- comandos financeiros, contratuais e de integração exigem idempotência.

Não haverá uma migração ampla de rotas somente para padronização estética.

## 10. Frontend

O frontend permanece uma aplicação Vite. Novas funcionalidades são organizadas
por feature e carregadas por rota quando possível. Separação de bundle é uma
otimização de entrega, não motivo para novo repositório.

Cada feature deve concentrar tipos, API, validações, componentes e regras de
apresentação. Páginas compõem features; não devem se tornar a única camada de
regra do cliente.

Princípios:

- ação principal alcançável e visível no celular;
- formulários curtos, em uma coluna e com teclado apropriado;
- filtros essenciais primeiro;
- tabelas recebem representação em cards no mobile;
- telas detalhadas usam resumo + ações e abertura progressiva;
- contexto da unidade é visível; se houver uma unidade, o seletor não ocupa a
  navegação principal.

## 11. Migrations e rollout

1. Alterações começam aditivas: tabelas e colunas novas, sem remover as atuais.
2. Backfills são determinísticos, observáveis e reiniciáveis ou transacionais.
3. Dual read pode ser usado por período curto; dual write só com regra explícita
   de fonte de verdade.
4. Constraints são adicionadas depois de backfill validado quando necessário.
5. Remoções exigem telemetria provando que o caminho legado não é mais usado.
6. Cada rollout informa contagens de reconciliação e procedimento de rollback.

Agentes não escolhem antecipadamente o número da migration. O integrador
reserva a sequência no merge para evitar colisões.

## 12. Decisões que permanecem abertas

- conjunto final de papéis operacionais além de owner e coach;
- se uma pessoa poderá compartilhar CPF/representante em cenários de menores;
- política de acesso entre organizações distintas usando a mesma conta;
- provedor financeiro definitivo do aluno e modelo de subcontas;
- estratégia final de sincronização de agenda de Wellhub e TotalPass.

Essas decisões não impedem as specs `001` a `003`, desde que seus contratos
preservem os pontos de extensão descritos.

## 13. Critérios para aprovação

- specs `001` a `003` usam este glossário sem criar entidades concorrentes;
- a transição mantém check-ins, retenção, campanhas e app do aluno funcionais;
- cada tabela nova possui módulo proprietário e escopo de tenancy definido;
- contratos permitem testes de isolamento entre duas organizações e unidades;
- nenhum novo serviço/repositório é necessário para a primeira onda;
- billing SaaS e cobrança do aluno permanecem inequivocamente separados.
