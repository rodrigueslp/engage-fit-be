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
3. `SPEC-002` — pessoa, identidades e vínculos.
4. `SPEC-003` — planos, matrículas e direitos de acesso.
5. Contratos, financeiro do aluno e agenda podem avançar em paralelo quando as
   três primeiras specs estiverem `ready`.

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
- [ARCH-001 — Fundação modular](ARCH-001-modular-foundation.md)
- [SPEC-001 — Organização e estrutura](specs/SPEC-001-organization-units-modalities-spaces.md)
- [SPEC-002 — Pessoa, identidades e vínculos](specs/SPEC-002-person-identities-relationships.md)
- [SPEC-003 — Planos, matrículas e acesso](specs/SPEC-003-plans-enrollments-entitlements.md)
