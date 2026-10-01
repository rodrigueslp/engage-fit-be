# SPEC-010 — Expansão da PWA do aluno

- Estado: `validated locally`
- Atualizada em: 2026-09-26
- Dependências: contratos públicos de `SPEC-004` a `SPEC-008`
- Migrations: `075` (ator atleta na auditoria de reservas) e `076` (marcas pessoais lançadas manualmente)

## Resultado implementado

O PWA autenticado já usado para treino, resultado e PR passa a reunir a relação
operacional do aluno com cada unidade: agenda publicada, ocupação, reserva e
presença próprias, plano vigente, contrato, cobranças e avisos acionáveis.

A entrega não cria outro aplicativo nem replica estados. `athlete` compõe um
read model dos módulos donos e autoriza cada item exclusivamente pelo caminho
`athlete_account -> athlete_box_membership -> athlete_student_link -> person`.
Nome, telefone ou e-mail nunca são usados para descobrir dados do titular.

## Contrato e privacidade

- `GET /api/v1/athlete/portal` exige cookie e CSRF da sessão independente do
  atleta;
- a resposta contém uma coleção por membership/unidade ativa, para preservar o
  uso multiunidade já existente;
- agenda cobre as últimas 24 horas e os próximos 30 dias, somente para sessões
  publicadas/agendadas, e mostra contagem agregada de ocupação, reserva, posição
  na fila e presença da própria pessoa;
- plano e matrícula expõem somente snapshot comercial e vigência do titular;
- contrato expõe documento, declaração e estado do próprio titular, sem token
  de aceite, evidência ou payload de provider;
- financeiro expõe cobranças próprias e somente instruções allowlisted para o
  pagador; referências internas e payload bruto do gateway não saem da API;
- a inscrição em aula pelo atleta usa vínculo explícito, cookie e CSRF próprios,
  a decisão de elegibilidade e a fila do módulo `reservations`, chave de
  idempotência e auditoria com ator atleta; não expõe dados de outros alunos;
- o cancelamento exige reserva confirmada ou em espera da própria pessoa e
  sessão ainda futura; usa a mesma transição auditada e promove a fila quando
  uma vaga confirmada é liberada;
- PR manual é registrado por membership ativo em uma fonte própria e participa
  da escolha da melhor marca junto aos resultados de treino;
- o total de check-ins soma os registros da pessoa na unidade vinculada, sem
  usar nome, telefone ou e-mail para descobrir a identidade;
- contrato, baixa financeira, notificação e chamada externa continuam fora
  desta iniciativa.

## Experiência

- a navegação inferior apresenta `Agenda` e `Extras` sem remover treino,
  avisos, PRs ou perfil; `Plano e pagamentos` fica no menu do perfil;
- `Agenda` apresenta uma semana fixa de segunda a domingo, com navegação entre
  semanas e seleção de um dia por vez. As aulas aparecem em linhas compactas
  com horário, profissional e ocupação agregada; ao abrir uma aula, o aluno vê
  espaço, vagas e sua própria reserva/presença, podendo inscrever-se, entrar
  na lista de espera ou cancelar uma reserva futura com confirmação;
- `Plano` agrupa avisos, matrícula, documento contratual expansível e histórico
  financeiro com instruções de pagamento seguras;
- avisos são derivados de estados atuais: plano ausente/inativo/próximo do fim,
  contrato pendente, cobrança pendente/vencida e posição na lista de espera;
- estados de carregamento, vazio, erro e retry são explícitos;
- o seletor de unidade filtra treino e portal pelo mesmo membership.
- `Hoje` mostra somente o treino da data atual ou o próximo treino publicado;
  treino antigo permanece apenas no histórico;
- `PRs` oferece lançamento direto de carga, repetições ou tempo, além das
  sugestões derivadas de resultados;
- `Perfil` mostra o total de check-ins por unidade e deixa explícita a ação
  de verificação de e-mail e seu resultado;
- `Extras` concentra ferramentas locais: conversor kg/lb e cronômetro, AMRAP,
  EMOM e Tabata, deixando espaço para ferramentas futuras.

## Fora do escopo preservado

- gerar ou reutilizar link secreto de aceite contratual;
- efetuar baixa financeira ou acionar gateway real;
- publicar comunicados e push, que continuam na `SPEC-014`;
- check-in autônomo fora do fluxo curto já existente.

## Evidência local

- regras unitárias validam janela de leitura e avisos derivados;
- teste PostgreSQL executa todas as consultas sobre o schema migrado e prova
  que somente a pessoa explicitamente vinculada entra no read model;
- `go test ./...` e `go vet ./...` aprovados;
- TypeScript e build Vite aprovados;
- Playwright focado cobre convite, treino, agenda, plano, contrato, cobrança,
  avisos, PR e convite pelo owner (`5 passed`);
- baseline Playwright completa aprovada (`50 passed`, `2 skipped`);
- revisão visual em `360x800` confirmou cards, navegação e conteúdo sem
  overflow horizontal; o cabeçalho das novas abas foi ajustado após a revisão.

## Ajustes solicitados em 2026-09-26

As sete melhorias de agenda, inscrição, PR manual, treino atual, verificação de
e-mail, ferramentas de treino e check-ins por unidade foram implementadas sobre
a entrega validada. A inscrição e o PR ampliam o contrato original de leitura
conforme a solicitação atual do responsável pelo produto. A experiência da
agenda foi refinada com base nos prints de referência do Tecnofit fornecidos:
seletor semanal compacto, lista de horários e detalhe da aula. A lista nominal
de presença exibida no aplicativo de referência não foi reproduzida para
preservar a privacidade dos outros alunos; check-in autônomo continua fora do
escopo.
Validação dos ajustes: `go test ./...`, `go vet ./...`, build frontend,
Playwright completo (`63 passed`, `2 skipped`), teste PostgreSQL da reserva
com ator atleta e teste PostgreSQL de vínculo/PR manual. Reaplicação de
migrations no banco descartável: zero migrations pendentes.

## Refinamento solicitado após homologação

O aluno pode cancelar sua própria reserva confirmada ou posição na espera antes
do início da aula. A ação exige confirmação na interface, chave de
idempotência e revalidação do vínculo/reserva no backend. O menu inferior troca
`Plano` por `Extras`; plano, contrato e pagamentos continuam acessíveis pelo
perfil. Timer e conversor foram movidos de `Hoje` para `Extras`.
Validação local: `go test ./...`, `go vet ./...`, build Vite e Playwright
completo (`63 passed`, `2 skipped`); o teste móvel do aluno cobre cancelamento,
inscrição, `Extras` e acesso ao plano pelo perfil.
