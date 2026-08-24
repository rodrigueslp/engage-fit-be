# Prompt do agente implementador — SPEC-006/A

Você é o agente executor da
`SPEC-006/A — Fundação de profissionais da agenda` do EngageFit.

A SPEC-006 é uma guia `reviewed` e a SPEC-006/A é a única slice `ready`.
Antes de criar código, migration ou endpoint, confirme esses status. Se a
documentação tiver voltado a `draft` ou sido substituída, interrompa e devolva
a tarefa ao responsável pela spec.

## Contexto obrigatório

Trabalhe somente nos worktrees backend/frontend designados para esta slice.
Antes de alterar qualquer arquivo, leia integralmente:

1. os `AGENTS.md` da raiz, backend e frontend;
2. `engage-fit-be/.ai/handoff.md`, começando pelo checkpoint atual;
3. `docs/system-design.md`, `docs/sdd/README.md`, `spec-catalog.md`,
   `REQ-001` e `ARCH-001`;
4. `SPEC-001/B`, `SPEC-001/D`, `SPEC-003/A`, `SPEC-003/B`, `SPEC-003/C`,
   a guia `SPEC-006` e a `SPEC-006/A`;
5. este prompt.

Use a SPEC-006/A como único contrato executável. A guia não autoriza as slices
B, C ou D. Inspecione os padrões integrados antes de criar abstrações. Se uma
mudança de contrato for indispensável, atualize primeiro a SPEC-006/A,
retorne-a à revisão e registre a divergência; nunca a implemente silenciosamente.

## Resultado a entregar

Entregue uma slice vertical backend/frontend para que o owner, em
`Agenda > Profissionais`, crie um profissional com apenas nome e opcionalmente
vincule uma conta ativa owner/coach da unidade. Ele pode editar, desvincular,
arquivar, reativar, exportar e anonimizar. Homônimos são válidos e jamais
produzem merge. O coach só lista profissionais ativos da unidade, sem ações.

Inclua:

- módulo `scheduling` na direção domínio -> aplicação -> ports -> adapters;
- migration aditiva de número provisório, sem seed ou backfill;
- isolamento explícito por organização e unidade, revisão otimista,
  idempotência e auditoria transacional;
- portas estreitas para contexto autorizado e contas elegíveis de
  `organizations/access`;
- exportação e anonimização irreversível, sem nome, motivo ou dados de conta
  em logs, métricas, auditoria ou command requests;
- capability `scheduling`, desligada por padrão, e frontend tolerante ao campo
  ausente;
- experiência mobile-first de owner e leitura mobile para coach.

## Fronteiras obrigatórias

- `scheduling` não importa nem consulta tabelas, models GORM, repositories
  concretos ou DTOs HTTP de `organizations/access`, `catalog`, `memberships`,
  `people` ou `contracts`.
- O composition root injeta os ports públicos. IDs externos são opacos.
- Organização e unidade vêm do contexto/rota autorizados; body nunca escolhe
  tenant ou unidade. UUID fora do escopo retorna `404`.
- Uma conta pode vincular no máximo um profissional por organização. Defenda
  isso na aplicação, com advisory lock e índice parcial PostgreSQL.
- O vínculo não copia nome/e-mail e não altera papel, permissões, `box_id`,
  sessão ou conta. Não crie papel, usuário ou multiunidade.
- Cada mutação, auditoria e command request ocorre na mesma transação; falha
  em qualquer parte reverte tudo. Todo comando exige `Idempotency-Key`.
- Preserve os dados permitidos pela SPEC-006/A: nome é a única PII nova.

## Rotas e UX

Implemente somente as rotas sob
`/api/v1/units/:unit_id/scheduling`, protegidas por capability, sessão tenant e
CSRF em mutações: listar profissionais, listar contas elegíveis, criar,
atualizar, arquivar, reativar, exportar e anonimizar, conforme os contratos e
erros estáveis da SPEC-006/A.

Adicione `#/schedule` condicionado à capability. Nesta slice a página abre em
Profissionais, sem tabs vazias de agenda/grade. Em `360x800`, mantenha coluna
única, alvos de 44 px, conta opcional por disclosure, estados de loading/vazio/
erro/conflito/retry/sucesso e confirmação explícita de ações destrutivas. Não
exiba UUIDs, tenant, e-mail, CPF ou jargão técnico. Preserve foco, Escape,
labels e rolagem interna de modal/drawer.

## Proibições de escopo

Não implemente grade, série, sessão, calendário, materializador, exceção,
espaço, capacidade, conflito, reserva, fila, presença, check-in, catraca,
consumo, acesso, contrato, PAR-Q, cobrança, mensagem, integração externa,
multiunidade, conta/papel/permissão, especialidade, remuneração ou PII além do
nome. Não altere contratos de `catalog`, `memberships`, `people` ou `contracts`.

Não faça push, merge, deploy, migration de produção, habilitação de flag ou
acesso ao banco compartilhado. Preserve mudanças e arquivos preexistentes do
usuário.

## Migration, rollout e validação

O integrador escolhe o número final da migration. Registre o número provisório;
não tente reservar ou alterar migration concorrente. Mantenha router,
capabilities, navegação, system-design e handoff na sua própria branch, pois o
integrador resolverá conflitos documentais.

Antes de atualizar `system-design.md`, status da spec e handoff, execute e
registre:

- `gofmt`, `go test -count=1 ./...`, `go vet ./...`, `git diff --check`;
- race detector em Linux/CGO com PostgreSQL real, ou o impedimento objetivo;
- migration em PostgreSQL vazio, segunda execução e upgrade desde a migration
  059, sem backfill;
- testes PostgreSQL de constraints, duas organizações/unidades, concorrência
  do vínculo, idempotência, revisão, auditoria, exportação e anonimização;
- TypeScript, build Vite, Playwright focado em `360x800`, regressão completa e
  revisão visual mobile.

Entregue commits locais separados para backend e frontend. No relatório final,
informe hashes, migration provisória, comandos/resultados, impedimentos,
evidências de isolamento/fronteiras/privacidade e a confirmação de que não
houve push, merge, deploy, banco compartilhado, produção ou flag habilitada.
