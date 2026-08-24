# Prompt do agente implementador — SPEC-004/A

Você é o agente executor da
`SPEC-004/A — Fundação de contratos e emissão vinculada à matrícula` do
EngageFit.

A `SPEC-004/A` foi revisada e está em `ready` para a fundação local descrita
neste prompt. No início da execução, confirme que o documento canônico ainda
está em `ready`; se tiver voltado a `draft` ou sido substituído, não crie
código, migration ou endpoint e devolva a tarefa ao responsável pela spec.

## Contexto obrigatório

Trabalhe somente nos worktrees backend/frontend designados para esta slice.
Antes de alterar qualquer arquivo, leia integralmente:

1. o `AGENTS.md` da raiz do workspace;
2. `engage-fit-be/AGENTS.md`;
3. `engage-fit-fe/AGENTS.md`;
4. `engage-fit-be/.ai/handoff.md`, começando pelo checkpoint atual;
5. `engage-fit-be/docs/system-design.md`;
6. `engage-fit-be/docs/sdd/README.md`;
7. `engage-fit-be/docs/sdd/ARCH-001-modular-foundation.md`;
8. `engage-fit-be/docs/sdd/REQ-001-primeiro-passo-operacional.md`;
9. `SPEC-002/A`, `SPEC-003/A`, `SPEC-003/B`, `SPEC-003/C` e a guia
   `SPEC-003`;
10. a versão `ready` da `SPEC-004/A`.

Use a `SPEC-004/A` como único contrato executável. Inspecione profundamente os
padrões integrados antes de criar abstrações. Se uma mudança de contrato for
realmente necessária, atualize primeiro a própria spec, volte-a para revisão e
destaque a divergência; não a implemente silenciosamente.

## Resultado a entregar

Implemente uma slice vertical backend/frontend que permita ao owner:

- criar, editar, publicar, versionar, arquivar e reativar modelos de contrato;
- gerar uma prévia determinística para uma matrícula elegível;
- criar um rascunho de instância com snapshot comercial imutável;
- emitir a instância e acompanhar `draft -> pending_acceptance -> accepted`;
- descartar somente um rascunho ainda não emitido;
- gerar uma sessão de aceite pelo port e concluir o aceite no simulador local;
- consultar documento, estado e histórico no celular;
- exportar os dados contratuais da pessoa e limpar artefatos sintéticos do
  simulador conforme a política da spec.

## Fronteiras obrigatórias

- Crie o módulo `contracts` seguindo domínio -> aplicação -> ports -> adapters.
- Não importe GORM models, repositories concretos ou DTOs HTTP de outro
  módulo.
- Consuma matrícula/composição exclusivamente pela porta pública de
  `memberships` definida na spec.
- Consuma a pessoa exclusivamente pela porta pública de `people` definida na
  spec.
- Use somente `member_name` como dado da pessoa no schema 1. Não busque nem
  persista CPF, endereço, e-mail ou telefone nesta slice.
- O repositório de contracts não pode consultar `enrollments`,
  `enrollment_addons`, `membership_*`, `persons`, `students`, `billing_*`,
  check-ins, agenda ou reservas. FKs compostas documentadas são permitidas;
  consultas de negócio cruzadas não são.
- Persista `person_id`, `enrollment_id`, `template_version_id`, snapshots e
  hashes; não use `student_id` como identidade contratual.
- Não aceite preço, adicionais, vigência, nome do plano ou total editável do
  frontend.
- O snapshot deve preservar exatamente plano, versão, preço base, adicionais,
  total, moeda, ciclo, início, validade e resumos da matrícula.
- Renderer e hashes são determinísticos e versionados. Não use IA.
- Template publicado/retirado, documento, snapshots e hashes de instância são
  imutáveis no PostgreSQL e na aplicação.
- Mantenha uma única instância não descartada de `membership_terms` por
  matrícula e cubra concorrência em PostgreSQL real.
- Use revisão otimista, advisory lock/índice quando definidos, command request,
  evento e auditoria na mesma transação.
- Token de aceite deve ter alta entropia, ser persistido somente como hash e
  aparecer uma única vez na resposta de criação de sessão.
- Não grave token, URL concreta, documento, nome, payload, IP ou user-agent em
  logs, métricas ou auditoria.
- O adapter `local_simulator` não abre rede e deve ser impossível em
  `APP_ENV=production`.
- O simulador precisa dizer claramente que não produz assinatura eletrônica
  com validade jurídica.
- Não crie adapter real, webhook, polling, retry de provedor ou credencial.
- Atualize privacy export por port público; não acople o repository de privacy
  às tabelas de contracts.
- Implemente retenção do simulador em dry-run por padrão e `--apply` explícito,
  usando somente dados sintéticos nos testes.

## UX e compatibilidade

- Adicione `#/contracts` para modelos e integre instâncias à seção `Contrato`
  do painel de matrícula existente.
- Não faça uma requisição de contrato por aluno na listagem.
- Trate capabilities ausentes como `false`.
- Mantenha uma ação primária por estado e abertura progressiva no editor.
- Cubra loading, vazio, erro, conflito, sessão expirada, replay, draft,
  aguardando, aceito e descartado.
- Preserve foco, Escape, labels, teclado e alvos touch; valide em `360x800` sem
  overflow horizontal.
- Renderize Markdown com allowlist segura e sem HTML, script, iframe ou recurso
  remoto.
- Aplique os limites aprovados: título de 1 a 160 caracteres, corpo Markdown
  de 1 a 50.000 e declaração de aceite de 10 a 1.000. Bloqueie também URLs
  executáveis.
- Permita que o owner escreva o conteúdo, sem seed jurídico e sem apresentar
  qualquer modelo como fornecido, validado ou juridicamente adequado pelo
  EngageFit.
- Mostre antes da emissão: `Esta etapa não gera cobrança, não altera a
  matrícula e não libera acesso.`
- Não chame o resultado local de assinatura digital/eletrônica válida.
- Frontend/backend devem poder ser publicados em qualquer ordem com as flags
  desligadas.

## Proibições de escopo

Não implemente:

- PAR-Q, formulário ou dado de saúde;
- cobrança do aluno ou alteração de `billing` SaaS;
- assinatura eletrônica de provedor real ou Gov.br;
- mensagem, e-mail, push ou qualquer envio real;
- pausa, trancamento, aditivo, renovação, distrato ou cancelamento jurídico;
- agenda, profissional, aula, vaga, reserva ou lista de espera;
- consumo, saldo, catraca, check-in ou decisão de acesso;
- mudança de status, datas ou termos da matrícula;
- contrato fictício de Wellhub/TotalPass;
- multiunidade ou seletor de contexto;
- PDF, object storage ou anexos;
- backfill de contratos ou template jurídico automático.

Não faça push, merge, deploy, migration de produção ou habilitação de flag
externa. Não altere o banco compartilhado `boxengage`. Não use dados ou
credenciais reais. Preserve qualquer arquivo ou mudança preexistente do
usuário, especialmente `.ai/acordo-fundadores-engagefit.md` e
`docs/proposta-engagefit-guilherme.html` quando estiverem presentes.

## Migration, configuração e rollout

- O integrador define o número final da migration. Use somente um número
  provisório explicitamente registrado no relatório.
- Não modifique migrations publicadas.
- A migration deve ser aditiva, sem seed, backfill ou alteração semântica de
  dados existentes.
- Use bancos PostgreSQL temporários com nomes inequívocos para banco vazio,
  segundo run, upgrade desde a migration integrada anterior, integração,
  concorrência e HTTP real. Remova todos ao terminar.
- Implemente as flags e validações de startup exatamente como fechadas na spec.
- Mantenha todas desligadas por padrão e não habilite produção nesta slice.

## Verificação obrigatória

Execute todos os testes da spec e, no mínimo:

### Backend

- `gofmt` nos arquivos Go alterados;
- `go test -count=1 ./...` com PostgreSQL real quando aplicável;
- `go vet ./...`;
- race detector em Linux/CGO com PostgreSQL real;
- migrations completas em banco vazio;
- segundo migrator com zero aplicações;
- upgrade desde a migration integrada anterior, comprovando zero backfill;
- constraints, FKs compostas, triggers, tenant e rollback;
- corridas reais de instância, emissão, sessão e aceite;
- testes arquiteturais de fronteira;
- privacy export e retenção dry-run/apply;
- `git diff --check`.

### Frontend

- TypeScript;
- build Vite;
- Playwright focado no fluxo completo em `360x800`;
- teste do simulador somente quando a capability local estiver ativa;
- regressão completa;
- revisão visual mobile e desktop;
- `git diff --check`.

Não atualize `docs/system-design.md` nem marque a spec como `validated` antes
de toda a implementação e validação estarem concluídas. Entregue commits
separados de backend e frontend.

## Relatório final

Informe:

1. resultado funcional entregue;
2. hashes dos commits backend/frontend;
3. número e nome provisório da migration;
4. comandos/testes e resultados;
5. evidências de banco vazio, segundo run e upgrade sem backfill;
6. evidências de imutabilidade, concorrência, idempotência e rollback;
7. revisão explícita de tenancy e fronteiras com memberships/people/privacy;
8. evidência de ausência dos módulos e efeitos proibidos;
9. validação mobile e compatibilidade de rollout;
10. checks não executados e motivo;
11. riscos, decisões restantes e divergências da spec;
12. confirmação de remoção dos bancos temporários;
13. confirmação de que não houve push, merge, deploy, banco compartilhado,
    produção, flag externa, provider real ou dados reais.
