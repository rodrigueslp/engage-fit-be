# EngageFit Gestão — Spec-Driven Development

Este diretório é a fonte de verdade das evoluções planejadas para o EngageFit
Gestão. Ele complementa `docs/system-design.md`, que continua descrevendo o
sistema efetivamente implantado.

Uma spec aprovada descreve o comportamento que queremos construir. O
`system-design.md` só deve ser atualizado quando esse comportamento estiver
implementado e validado.

## Objetivo

Evoluir o EngageFit para substituir o núcleo operacional de sistemas como
Tecnofit e Next Fit, com uma experiência mais simples, mobile-first e orientada
à produtividade, preservando retenção e relacionamento como diferenciais.

Não buscamos paridade de catálogo. Cada capacidade concorrente deve ser
classificada como:

- essencial para substituição;
- útil, mas adiável;
- atendida por integração;
- deliberadamente fora do produto.

## Princípios de produto

1. O fluxo principal deve funcionar bem em um celular e com poucos toques.
2. Nenhum campo é obrigatório sem uso operacional, financeiro, jurídico ou de
   segurança identificável.
3. O sistema oferece bons padrões antes de oferecer configuração.
4. Complexidade avançada aparece por abertura progressiva.
5. Dinheiro, contratos, identidade, acesso e comunicação são auditáveis.
6. Dados são informados uma vez e reutilizados pelos módulos autorizados.
7. Integrações externas ficam atrás de contratos internos e feature flags.
8. Uma inferência pode ser implementada, mas deve ser identificada e barata de
   revisar.
9. O EngageFit fecha ciclos operacionais; não acumula telas sem uma ação clara.
10. Compatibilidade e migração fazem parte da funcionalidade, não são tarefas
    deixadas para o final.

## Proveniência das decisões

Requisitos e decisões relevantes devem usar uma destas marcações:

- **Existente:** comprovado no código ou na operação atual do EngageFit.
- **Benchmark:** observado em documentação, demonstração ou material público.
- **Inferência:** solução proposta para o EngageFit sem evidência direta.
- **Validação futura:** hipótese que deve ser verificada na operação do Alados.
- **Dependência externa:** comportamento condicionado a provedor ou parceiro.

Benchmark explica o problema e oferece referência; não obriga a copiar fluxo,
texto, estrutura visual ou modelo de dados de outro produto.

## Ciclo de uma spec

| Estado | Significado |
|---|---|
| `draft` | aberta para decisões e ajustes |
| `reviewed` | produto e arquitetura revisados |
| `ready` | contratos e critérios permitem implementação |
| `implementing` | existe trabalho em andamento |
| `validated` | código, testes e documentação foram validados |
| `superseded` | substituída por uma versão ou decisão posterior |

Uma spec em `draft` não autoriza migrations ou contratos públicos definitivos.

## Ordem inicial

1. `ARCH-001` — fundação modular e evolução segura.
2. `SPEC-001` — organização, unidades, modalidades e espaços.
3. `SPEC-001/D` e `SPEC-002/A` — integradas, automatizadas e homologadas
   manualmente, sem habilitação em produção.
4. `SPEC-003/A` — catálogo versionado de planos — integrada e homologada.
5. `SPEC-003/C` — matrícula operacional da pessoa em um plano publicado —
   integrada e homologada localmente.
6. `SPEC-003/B` — adicionais versionados e composição comercial (B1/B2) —
   integrada e homologada localmente.
7. `SPEC-004/A` — fundação de contratos e emissão vinculada à matrícula —
   revisada e `ready` para implementação local, sem aceite eletrônico
   produtivo, PAR-Q ou dados reais.
8. Financeiro do aluno e agenda podem ser detalhados em paralelo quando seus
   contratos internos e as fundações aplicáveis estiverem aceitos.

`SPEC-001/C`, de contexto multiunidade, pode ser executada depois da fundação
operacional de uma unidade. As letras identificam slices da mesma spec; não
obrigam que C seja implementada antes de D quando as dependências registradas
permitirem.

## Regras para agentes paralelos

- Cada agente trabalha em worktree/branch própria e declara os módulos sob sua
  responsabilidade.
- Alterações em contratos compartilhados exigem atualização da spec antes do
  merge.
- Um agente não acessa diretamente tabelas pertencentes a outro módulo sem um
  contrato registrado.
- Números finais de migrations são reservados pelo integrador no momento do
  merge; specs não fixam o próximo número.
- Mudanças devem ser aditivas sempre que possível e preservar o rollout da API
  e do frontend em ordens diferentes.
- Todo efeito externo exige idempotência, auditoria, timeout e feature flag.
- O agente integrador revisa migrations, rotas, tenancy, eventos e alterações
  concorrentes antes do merge.

## Definition of Done de uma implementação

- critérios de aceite automatizados quando tecnicamente possível;
- testes unitários e de integração proporcionais ao risco;
- isolamento entre organizações/unidades testado;
- migration em PostgreSQL vazio e idempotência do migrator verificadas;
- compatibilidade com dados existentes demonstrada;
- TypeScript e build do frontend aprovados quando aplicável;
- fluxo mobile principal coberto;
- observabilidade, auditoria, privacidade e erros revisados;
- feature flag e rollout definidos;
- `docs/system-design.md` e handoff atualizados após validação.

## Documentos

- [Template de spec](spec-template.md)
- [Catálogo e ondas de execução](spec-catalog.md)
- [REQ-001 — Primeiro passo operacional](REQ-001-primeiro-passo-operacional.md)
- [ARCH-001 — Fundação modular](ARCH-001-modular-foundation.md)
- [SPEC-001 — Organização e estrutura](specs/SPEC-001-organization-units-modalities-spaces.md)
- [SPEC-001/B — Modalidades e espaços](specs/SPEC-001-B-modalities-spaces.md)
- [SPEC-001/D — Serviços ofertáveis](specs/SPEC-001-D-service-offerings.md)
- [SPEC-002 — Pessoa, identidades e vínculos](specs/SPEC-002-person-identities-relationships.md)
- [SPEC-002/A — Fundação e cadastro manual](specs/SPEC-002-A-person-foundation-manual-registration.md)
- [SPEC-003 — Planos, matrículas e acesso](specs/SPEC-003-plans-enrollments-entitlements.md)
- [SPEC-003/A — Catálogo versionado de planos](specs/SPEC-003-A-versioned-membership-plan-catalog.md)
- [SPEC-003/B — Adicionais e composição comercial](specs/SPEC-003-B-versioned-addons-composition.md)
- [SPEC-003/C — Matrícula em plano publicado](specs/SPEC-003-C-person-plan-enrollment.md)
- [SPEC-004/A — Fundação de contratos e emissão](specs/SPEC-004-A-contract-foundation-enrollment-issuance.md)
- [Prompt de implementação da SPEC-004/A](prompts/SPEC-004-A-implementation.md)
- [SPEC-006 — Agenda, profissionais e grade recorrente](specs/SPEC-006-scheduling-professionals-recurring-grid.md)
