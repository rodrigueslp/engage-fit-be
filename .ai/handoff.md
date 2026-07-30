# EngageFit - Handoff de Contexto

Manual canônico de arquitetura e negócio: `docs/system-design.md`.

Guia operacional consolidado: `docs/application-readiness-guide.md`.

Atualizado em: 2026-07-30 (ativação consentida no WhatsApp sem telefone nas planilhas)

## Checkpoint de ativação consentida no WhatsApp em 2026-07-30

- Wellhub e TotalPass não fornecem telefone nos arquivos reais. O produto agora
  trata as planilhas como fonte de frequência e cria uma base de comunicação
  própria por adesão do aluno.
- A migration `041_create_contact_activations.sql` adiciona código público por
  academia, pedidos temporários de ativação e trilha versionada de
  consentimento/revogação.
- Nova página pública `#/activate/:code`: o aluno informa nome, plataforma e
  uma presença recente, aceita o texto explícito e segue para o WhatsApp sem
  digitar telefone.
- O número só é persistido depois que o aluno envia a mensagem. O webhook
  `POST /api/v1/webhooks/twilio/whatsapp` valida `X-Twilio-Signature` com a
  credencial efetiva do box antes de qualquer alteração.
- Correspondência única por box, origem, nome normalizado e data confirma
  automaticamente. Casos sem correspondência única entram em `needs_review`.
- A área `Ativação WhatsApp` mostra cobertura, QR público, convite individual
  para uso na recepção, fila de revisão e histórico recente.
- `SAIR`, `PARAR`, `CANCELAR` e `STOP` registram revogação e marcam o aluno
  como `opted_out`.
- Audiências de comunicação agora exigem `opted_in`; `unknown` deixou de ser
  tecnicamente contatável. O seed demo autoriza explicitamente seus alunos.
- Exportação e anonimização LGPD incluem a nova trilha. O comando de retenção
  remove ativações e eventos após o prazo de auditoria configurado.
- Configuração nova:
  `TWILIO_INBOUND_WEBHOOK_URL=https://www.engagefit.com.br/api/v1/webhooks/twilio/whatsapp`.
  O valor deve ser idêntico à URL cadastrada no sender Twilio para a assinatura
  validar. A configuração e o webhook ainda não foram aplicados em produção.
- Validações concluídas até este checkpoint: 41 migrations em PostgreSQL vazio,
  idempotência local, integração PostgreSQL do vínculo/consentimento,
  assinatura Twilio unitária, `go test ./...`, TypeScript e build Vite.

## Checkpoint de importação real e refinamento da retenção em 2026-07-30

Este é o checkpoint mais recente da V2 preparada para o piloto assistido do
CrossFit Alados. Em caso de conflito com o histórico abaixo, esta seção
prevalece. O proprietário autorizou preparar a V2 para produção porque a
operação do primeiro mês ficará nas mãos do próprio operador do EngageFit, mas
**esta sessão não executou merge nem deploy de produção**.

### Identidade dos alunos no TotalPass

- Foram analisados os arquivos reais de Wellhub e TotalPass fornecidos pelo
  usuário. O export de tokens do TotalPass contém `ID` e `Código`, mas ambos
  identificam o token/check-in, não o colaborador. O mesmo aluno recebe valores
  diferentes em cada visita.
- O bug fazia cada visita TotalPass criar um aluno. No arquivo analisado,
  Adriana Segatelli aparecia 16 vezes com 16 IDs diferentes.
- O parser agora reconhece o export de tokens pelas colunas `Colaborador` e
  `Validado em` e ignora `ID`/`Código` como identidade de aluno. Um
  `external_id` realmente estável continua sendo respeitado em outros formatos.
- Sem identificador, e-mail ou telefone estável, a identidade passa a ser o
  nome em minúsculas com espaços normalizados, sempre isolado por academia e
  origem. Limitação conhecida: duas pessoas diferentes com exatamente o mesmo
  nome completo na mesma academia/origem serão unificadas, pois o arquivo não
  oferece outro atributo para distingui-las.
- A migration `040_merge_totalpass_students_by_name.sql` corrige dados já
  importados. Ela:
  - consolida alunos TotalPass pelo nome normalizado dentro do box;
  - remove repetição da mesma visita antes de mover os check-ins;
  - preserva históricos de mensagens, e-mail, treino, retenção, auditoria e o
    estado manual de entregas de recompensa;
  - recalcula progresso de campanhas e sincroniza entregas pendentes;
  - preserva datas de início explicitamente confirmadas.
- Na primeira validação local, a consolidação reduziu 2.058 registros
  TotalPass para 211 pessoas; Adriana passou a ter um cadastro com 16
  check-ins. Depois da limpeza controlada e nova importação dos arquivos reais,
  o estado observado foi:
  - TotalPass: 206 alunos e 2.052 check-ins, de 30/06 a 30/07;
  - Wellhub: 55 alunos e 1.002 check-ins, de 01/04 a 29/07;
  - total: 261 cadastros e 3.054 check-ins.
- Existe pelo menos um mesmo nome presente nas duas plataformas. A
  consolidação entre Wellhub e TotalPass permanece fora deste recorte; o
  isolamento por origem continua intencional.

### Radar de retenção mais conservador

- O radar continua comparando 28 dias recentes com as quatro semanas
  anteriores e exige oito semanas de cobertura para comparar frequência.
- Uma rotina mínima de **4 check-ins no histórico** passou a ser obrigatória
  antes de classificar ausência ou queda como risco de retenção. Alunos com
  uma a três visitas ficam em `Pouco histórico`, com sinal
  `routine_insufficient`, e não entram na fila de ação.
- A queda percentual continua sendo calculada somente quando existem ao menos
  4 check-ins na janela anterior.
- Com `risk_inactive_days=7`, os limites explicados pela API são:
  - `Observar`: 5 dias sem presença ou queda mínima de 25%;
  - `Em queda`: 7 dias sem presença ou queda mínima de 50%;
  - `Atenção imediata`: 14 dias sem presença ou queda mínima de 75%.
- Na base real local, a nova rotina mínima reduziu:
  - `Precisam de ação`: de 35 para 25;
  - `Atenção imediata`: de 32 para 22.
- Todos os 25 casos restantes eram Wellhub e possuíam pelo menos quatro
  presenças. O TotalPass permaneceu em `Pouco histórico` porque seu arquivo
  cobre somente cerca de um mês.

### Explicação das regras dentro do produto

- Novo endpoint autenticado e permitido para `OWNER` e `COACH`:
  `GET /api/v1/retention/rules`.
- O endpoint é a fonte de verdade para datas das janelas, corte de histórico,
  rotina mínima e limites de ausência/queda. A tela não mantém uma segunda
  cópia desses números.
- O botão `Entenda os cálculos`, na página `Retenção`, abre um painel lateral
  com:
  - datas exatas dos dois períodos comparados;
  - requisito de oito semanas e quatro presenças;
  - limites de cada nível;
  - exemplos reais escolhidos deterministicamente do radar atual;
  - ressalva de que os sinais não preveem cancelamento.
- Os exemplos **não usam IA**. Eles são montados diretamente com check-ins,
  queda percentual, última presença e nível calculado. Isso evita custo,
  latência e explicações inventadas.
- O carregamento das regras é tolerante a uma API antiga durante rollout: se o
  endpoint ainda não estiver disponível, o radar continua carregando e o botão
  explicativo permanece desabilitado.

### Primeiros 30 dias com confiança explícita

- A planilha não contém data de matrícula. Antes deste ajuste, o primeiro
  check-in importado era tratado como início real; isso colocava 210 alunos em
  `Primeiros 30 dias`, incluindo todos os 206 TotalPass, apesar de o arquivo
  começar em 30/06.
- A jornada agora distingue:
  - `confirmed`: data informada manualmente pelo box ou recebida de integração;
  - `probable`: primeira presença inferida com pelo menos 56 dias anteriores
    cobertos pela mesma plataforma sem check-in daquele aluno;
  - desconhecido: cobertura anterior insuficiente; o aluno não aparece na
    jornada de primeiros 30 dias.
- A cobertura é calculada por box e origem a partir da menor data importada da
  plataforma. Isso evita usar os meses do Wellhub como cobertura do TotalPass.
- Com os arquivos atuais, a seção caiu de 210 para **4 inícios prováveis,
  todos Wellhub**, com 96 a 108 dias anteriores cobertos. Nenhum TotalPass é
  apresentado como novo.
- A aba foi renomeada para `Primeiros 30 dias confiáveis`. Cada linha mostra
  `Início provável` ou `Início confirmado`, a quantidade de dias cobertos
  antes da primeira presença e permite confirmar a data sugerida sem precisar
  alterá-la.
- O painel `Entenda os cálculos` também explica as tags:
  `Início confirmado`, `Início provável`, `Sem primeira presença`,
  `Início recente`, `Aguardando 2ª presença`, `Em formação de hábito` e
  `Rotina interrompida`. Os textos usam o limite real de inatividade retornado
  pelo backend.

### Estado local e validações deste checkpoint

- A base local preserva uma academia ativa e seu owner. Durante a sessão foi
  necessário refazer a limpeza: o estado correto para reteste manteve academia
  e usuários e removeu somente alunos, check-ins e importações; depois os dois
  arquivos reais foram importados novamente.
- Não registrar credenciais locais ou de produção neste handoff.
- Validações concluídas:
  - parser executado contra o XLSX real TotalPass: 16 linhas da Adriana sem ID
    transitório como identidade;
  - migration 040 aplicada no PostgreSQL local e reaplicada dentro de transação
    com rollback para conferir idempotência operacional;
  - `go test ./...`;
  - testes unitários da rotina mínima, janelas/regras e confiança do onboarding;
  - smoke autenticado em API temporária na porta 8081 para radar, regras e
    onboarding;
  - endpoint de onboarding retornando exatamente 4 Wellhub e zero TotalPass;
  - TypeScript sem erros;
  - build Vite de produção;
  - `git diff --check` nos dois repositórios.
- O build Vite mantém apenas o aviso já conhecido de chunk principal acima de
  500 kB; não é regressão deste checkpoint.
- Após atualizar o código local, reiniciar o backend é necessário para expor
  `/api/v1/retention/rules` e a nova resposta do onboarding.
- Commits deste checkpoint:
  - backend: o commit que contém esta seção;
  - frontend: `cc9bfe5` (`feat: explain retention rules and onboarding confidence`).

## Desenvolvimento V2 isolado do go-live

- A evolução V2 começou nas branches locais `v2/retention-foundation` do
  backend e frontend; não foi enviada nem implantada em produção.
- Primeiro recorte: radar explicável de mudança de frequência e registro de
  ações humanas, documentado em `.ai/v2-retention-foundation.md`.
- O radar compara duas janelas de 28 dias, distingue histórico insuficiente e
  evita apresentar o resultado como previsão de cancelamento.
- A migration `035_create_retention_interventions.sql`, endpoints de radar e
  intervenções e a página `Retenção` foram implementados.
- Retorno em 3, 7 e 14 dias é descrito como associação temporal, sem atribuir
  causalidade ao contato.
- Após validação de uso, o radar passou a separar o sinal de frequência do
  fluxo operacional. A tela abre em `Precisa de ação`; contatos realizados vão
  para `Em acompanhamento`, voltam como `Revisão vencida` na data informada e
  podem ser pausados ou encerrados. Assim, um aluno pode continuar crítico sem
  permanecer indevidamente na fila diária.
- O histórico individual de presença ganhou painel visual reutilizável no
  Radar e em `Check-ins`, com comparação das últimas oito semanas, calendário
  mensal navegável e detalhes por dia/origem. Usa o endpoint individual já
  existente e não adiciona migration.
- O painel `Histórico de frequência` permite navegar entre meses, selecionar
  uma data e conferir horário, origem e múltiplos check-ins no mesmo dia. As
  cores diferenciam Wellhub e TotalPass; a visão de oito semanas torna quedas e
  interrupções de rotina visualmente identificáveis.
- O fluxo foi validado manualmente pelo usuário no navegador com a base demo.
  Também foram conferidos o painel em desktop/mobile, a abertura a partir de
  `Retenção` e `Check-ins` e o estado real da Marina: sinal `critical`, mas
  acompanhamento `waiting_return` com data de revisão, fora da fila diária de
  ações pendentes.
- Validações técnicas da V2: `go test ./...`, teste PostgreSQL de isolamento
  entre tenants, TypeScript, build Vite, smoke autenticado do radar,
  navegação Playwright pelos dois pontos de entrada e `git diff --check`.
- O desenvolvimento local aplicou a migration 035; a segunda execução aplicou
  zero migrations.
- O conjunto permanece isolado da `main` nas branches
  `v2/retention-foundation` dos dois repositórios. Este checkpoint será
  versionado e enviado para essas branches remotas, sem merge nem deploy; a V1
  do go-live permanece inalterada.
- A continuidade da V2 implementou os itens priorizados até operação por
  coaches: painel de resultados, motivos estruturados, recomendações por
  regras, jornada dos primeiros 30 dias, entrada recorrente idempotente de
  arquivos e papel `COACH` com atribuição e allowlist de acesso.
- Foram adicionadas as migrations `036` a `039`. Elas passaram em PostgreSQL
  vazio (39 migrations), segunda execução com zero aplicações e testes reais
  de isolamento. Backend, TypeScript, build Vite e revisão visual
  desktop/mobile também passaram.
- A entrada recorrente é uma porta segura para conectores enviarem CSV/XLSX;
  não é integração nativa com Wellhub/TotalPass. Credenciais são exibidas
  somente na criação/rotação e persistidas como hash.
- A importação manual deixou de iniciar com Wellhub pré-selecionado. A origem
  agora deve ser escolhida explicitamente antes de liberar o envio, e a
  confirmação informa a plataforma usada. A mudança reduz a criação acidental
  de um segundo aluno quando o arquivo pertence a outra origem.
- O usuário validou no navegador a jornada dos primeiros 30 dias e a
  importação de um check-in para uma aluna TotalPass. Duas tentativas com a
  origem Wellhub selecionada criaram identidades separadas, como previsto pelo
  isolamento por plataforma; os registros de teste foram removidos e esse uso
  revelou a necessidade da seleção explícita descrita acima.
- A página de Retenção ganhou o primeiro manual de uso dentro do produto. O
  botão `Como usar` abre um painel lateral com oito tópicos, screenshots reais
  ampliáveis, rotina sugerida, explicação das filas, leitura dos sinais,
  histórico de frequência, registro de acompanhamento, primeiros 30 dias,
  resultados e boas práticas.
- O manual usa o componente reutilizável
  `src/components/help/ProductGuide.tsx`; o conteúdo específico fica em
  `src/features/help/guides/retentionGuide.ts` e as imagens em
  `public/help/retention`. O padrão pode ser replicado nas demais páginas sem
  acoplar o conteúdo ao componente. A primeira versão foi deliberadamente
  preparada e validada para desktop; versão editorial específica para mobile
  ficou para depois.
- O manual passou em TypeScript, build Vite e navegação Playwright: oito
  tópicos, troca de seção, ampliação de imagem, fechamento por `Escape` e
  ausência de overflow horizontal no viewport desktop.
- WhatsApp em duas vias e IA/score preditivo permanecem deliberadamente fora
  deste recorte.

## Checkpoint de go-live do CrossFit Alados em 2026-07-28

Este checkpoint é o retrato mais recente da preparação para o go-live previsto
para **01/08/2026**. Em caso de conflito com checkpoints históricos abaixo,
prevalece esta seção.

### Conclusão executiva

- O EngageFit está **próximo e tecnicamente apto para um piloto assistido** com
  o CrossFit Alados em 01/08/2026.
- Não foi identificado bloqueador técnico conhecido na aplicação, no deploy,
  no acesso ou na conexão dedicada de WhatsApp.
- O lançamento deve continuar restrito ao Alados, acompanhado manualmente e
  com automações agendadas desligadas no início.
- Os principais pontos ainda abertos são operacionais/jurídicos: revisar e
  assinar o termo do piloto, formalizar o aviso de privacidade e as
  responsabilidades do Alados, importar os dados reais quando o cliente estiver
  pronto e combinar a cobrança real.
- Backup automático/PITR do Railway foi conscientemente adiado por depender do
  plano Pro. Existe backup manual anterior à limpeza, mas isso não substitui uma
  política recorrente de backup.

### Produção, código e deploy

- Backend, frontend e PostgreSQL estão no ambiente `production` do Railway.
- Domínio principal validado: `https://www.engagefit.com.br`.
- A aplicação pública, o proxy para a API e os healthchecks responderam
  corretamente durante a conferência.
- Código implantado e CI confirmada como verde:
  - backend: `525bd21`;
  - frontend: `b6b790c`.
- O frontend possui a opção de desativar planos comerciais. Por padrão exibe
  somente planos ativos e oferece filtros para ativos, desativados ou todos.
- `OWNER_SETUP_ENABLED=false` foi confirmado e o setup token foi
  removido/selado.
- `Serverless/App Sleeping` está desligado na API.
- Custom Start Command está vazio.
- Pre-deploy configurado como
  `/usr/local/bin/engagefit-migrate up`; deploy validado com
  `migration complete: 0 applied`.
- A política de restart da API foi revisada no Railway.

### Observabilidade, disponibilidade e alertas

- OpenTelemetry está habilitado na API e os traces estão chegando ao Grafana
  Cloud. A própria tela de setup confirmou `Traces are being ingested
  properly`.
- Foi criado o synthetic check `engagefit-production-api` para:
  `GET https://www.engagefit.com.br/api/v1/capabilities`.
- Probe de São Paulo validada com HTTP 200.
- Foram configurados alertas do check para falhas, expiração de certificado TLS
  e latência.
- O contact point e a notification policy do Grafana foram configurados para
  encaminhar os alertas por e-mail.
- Monitorar nos primeiros dias se os alertas estão chegando de fato e fazer um
  teste controlado de notificação quando conveniente.
- Domínio, redirecionamento do apex para `www` e certificado TLS foram
  verificados. Também estão presentes CSP, `X-Frame-Options`,
  `X-Content-Type-Options`, política de referrer e permissions policy.
- O header HSTS ainda não está presente. Isso é uma melhoria de segurança, não
  um bloqueador imediato do piloto. Quando for habilitado, começar de forma
  gradual e não usar `includeSubDomains` ou `preload` até validar HTTPS em todos
  os subdomínios.

### Academias, dados de teste e backup manual

- As três academias usadas em homologação foram arquivadas:
  `Academia Produção Teste`, `Academia Teste` e `Billing Sandbox`.
- O CrossFit Alados é a única academia ativa.
- Os dados operacionais de teste do Alados foram removidos de produção para
  permitir início limpo:
  - 2 alunos;
  - 20 check-ins;
  - 2 importações;
  - 1 campanha de meta;
  - 1 campanha de mensagem;
  - 3 destinatários;
  - 2 disparos;
  - 2 buckets de consumo.
- Foram preservados o tenant, o owner, a conexão dedicada Twilio, os três
  templates aprovados e a política de mensageria.
- Antes da limpeza foi criado backup lógico em:
  `/home/luiz-paulo/workspace/engage-fit/production-backups/engagefit-production-pre-cleanup-20260728.dump`.
- SHA-256 do backup:
  `32640d8ed298bf4791803adca65ca08af582fe56b8ff834aa37c3e855932aec7`.
- O arquivo temporário que continha a credencial do banco foi removido.
- Não é necessário preservar histórico financeiro dos testes.
- Backup automático/PITR e teste recorrente de restore permanecem adiados até
  contratação do Railway Pro. Tratar isso como risco aceito do piloto e manter
  cópias dos arquivos de importação na origem.

### Acessos e identidade

- O acesso `PLATFORM_ADMIN` foi validado.
- E-mail administrativo correto: `lprodrigs@gmail.com`.
- O endereço antigo `improdrigs@gmail.com` está incorreto e não deve ser usado.
- Owner do CrossFit Alados validado com `aladoscrossfit@gmail.com`.
- Senhas e demais credenciais não devem ser registradas no Git ou neste
  handoff.

### Plano comercial do Alados

- Plano confirmado: `Piloto 500`.
- Duração: 90 dias.
- Mensalidade: **R$ 247,00**.
- Sem taxa de implantação.
- Franquia: 500 mensagens de WhatsApp por ciclo mensal.
- Limite: 50 mensagens por dia.
- Limite: 50 destinatários por disparo.
- Alerta de consumo: 80%.
- Tolerância após vencimento: 2 dias.
- Sem cobrança automática de excedentes e sem acúmulo de franquia.
- O documento histórico `.ai/commercial-pilot-alados.md` ainda contém a oferta
  antiga de R$ 297/300 mensagens e **não deve ser enviado ao cliente** sem
  atualização.
- A sincronização do cliente com o Asaas, a criação da assinatura e a primeira
  cobrança real foram adiadas por decisão do usuário. Os fluxos do Asaas já
  foram bastante homologados, mas a contratação real do Alados será feita
  depois.

### WhatsApp e automações

- A conexão dedicada Twilio do CrossFit Alados está salva e o teste de conexão
  foi validado pelo provedor.
- Os três templates `_2` permanecem aprovados e configurados:
  - `copy_engagefit_falta_pouco_2` ->
    `HX38b4be2ac29ab416ab94d06357280cf4`;
  - `copy_engagefit_meta_atingida_2` ->
    `HX75f8491a6c1e7f8446677b4ad13f493d`;
  - `copy_engagefit_sentimos_sua_falta_2` ->
    `HXa819f068a4fc87df4b75c1ade6be7a39`.
- O envio real controlado já foi recebido corretamente nos aparelhos de teste.
- `FEATURE_WHATSAPP_ENABLED=true` e `WHATSAPP_ALLOW_REAL_SEND=true` estão
  preparados para o uso controlado.
- `FEATURE_AUTOMATION_ENABLED=true`, mas
  `AUTOMATION_WORKER_ENABLED=false`.
- Não há rotinas automáticas cadastradas no Alados após a limpeza.
- Manter o worker desligado no início. Criar e revisar cada rotina junto com o
  Alados antes de habilitar execução agendada.
- O Alados deve revisar audiência, mensagem e preferências de contato antes de
  cada disparo e registrar `Autorizado`, `Não contatar` ou `Não informado`
  conforme a evidência disponível.

### Importação e operação inicial

- A interface de importação de Wellhub e TotalPass foi testada anteriormente.
- A importação dos arquivos reais foi deliberadamente adiada pelo usuário e
  será validada depois.
- Como a base foi limpa, o go-live operacional precisa começar com uma
  importação real revisada, conferindo:
  - origem correta;
  - quantidade de linhas, alunos e check-ins;
  - duplicidades e rejeições;
  - telefones antes de qualquer campanha;
  - preferência de contato;
  - período e metas da primeira campanha.
- Campanhas e automações devem ser criadas somente depois dessa conferência.

### Termo digital, privacidade e responsabilidades

- Foi criada a minuta
  [`.ai/termo-piloto-alados.md`](termo-piloto-alados.md), versão 0.1, para
  revisão da advogada do usuário.
- A minuta consolida escopo, prazo, valor, franquia, responsabilidades, regras
  de WhatsApp, confidencialidade, segurança, rescisão, tratamento de dados,
  fornecedores e assinatura eletrônica com trilha de auditoria.
- Enquadramento operacional proposto, ainda sujeito à revisão jurídica:
  - Alados como controlador por decidir finalidade, base legal, alunos,
    audiências e mensagens;
  - EngageFit como operador ao processar esses dados conforme as instruções do
    Alados.
- Para o piloto, a recomendação é assinar a versão final por plataforma de
  assinatura eletrônica que registre identidade, autenticação, data/hora,
  versão ou hash do documento, eventos, IP quando disponível e entregue uma
  cópia final às duas partes.
- Não armazenar no Git a versão final assinada, CPF, CNPJ completo ou outros
  dados pessoais dos representantes. O repositório deve conservar apenas a
  minuta com placeholders.
- Campos ainda pendentes na minuta:
  - razão social/nome, CPF ou CNPJ, endereço e representante das Partes;
  - forma e dia de vencimento;
  - canal e horário de suporte;
  - confirmação da janela de 30 dias para exportação após encerramento;
  - confirmação da rescisão com aviso de 7 dias e sem multa;
  - cidade/UF do foro;
  - revisão jurídica das cláusulas de privacidade, responsabilidade,
    transferência internacional e retenção.
- O Alados ainda precisa disponibilizar uma informação/aviso de privacidade aos
  alunos e definir internamente quem atende pedidos de acesso, correção,
  oposição, não contato e anonimização. Isso pode ser simples no piloto, mas
  deve existir antes de campanhas reais em escala.

### Pendências restantes classificadas

#### Necessárias antes do primeiro uso real

1. Receber a revisão jurídica, preencher e assinar o termo do piloto.
2. Definir e comunicar ao Alados o procedimento mínimo de privacidade e não
   contato dos alunos.
3. Importar e conferir o primeiro arquivo real.
4. Criar e revisar a primeira campanha, audiência e mensagem antes do envio.
5. Confirmar com o Alados a forma operacional de suporte e cobrança, ainda que
   a assinatura do Asaas seja criada posteriormente.

#### Riscos aceitos ou itens adiados

1. Backup/PITR automático e restore gerenciado do Railway, dependentes do plano
   Pro; existe apenas o backup lógico manual registrado acima.
2. Sincronização e cobrança real do Alados no Asaas.
3. Importação dos arquivos reais.
4. HSTS, a ser ativado gradualmente.
5. Automações agendadas, mantidas desligadas até homologação explícita.

#### Acompanhamento recomendado na primeira semana

1. Conferir diariamente Railway, Grafana, synthetic check, traces e alertas.
2. Acompanhar consumo e custo real da subconta Twilio.
3. Revisar manualmente os primeiros disparos e seus destinatários.
4. Conferir falhas de importação, duplicidades e preferências de contato.
5. Registrar problemas, decisões e feedback do Alados.
6. Fazer checkpoints de 30, 60 e 90 dias conforme o termo do piloto.

## Documento de visão futura de produto

Foi registrada a visão de evolução comercial e de produto em
[`docs/future-product-roadmap.md`](../docs/future-product-roadmap.md). O
documento reúne oportunidades futuras para aumentar frequência e retenção,
incluindo radar de risco, jornadas do ciclo de vida, central de ações da
equipe, medição de retorno/ROI, feedback, desafios, WhatsApp em duas vias,
integrações e usos responsáveis de IA. Também contém roadmap conceitual,
métricas para o piloto e frases para conversas com futuros donos de boxes.

Esse material é exploratório: não representa funcionalidades implementadas nem
compromissos de prazo. A validação com os primeiros boxes deve orientar a
priorização, especialmente sobre os sinais que antecedem afastamento e as
ações que efetivamente recuperam frequência.

## Pendências consolidadas após a sessão de 2026-07-24

### Billing Asaas

- O pagamento real em produção foi homologado ponta a ponta em 2026-07-24:
  criação da cobrança, pagamento, webhook, atualização financeira e liberação
  de acesso funcionaram corretamente.
- O binário de conciliação passou a usar validação mínima própria. O serviço
  Cron `engage-fit-billing-reconcile` foi criado no Railway usando somente
  PostgreSQL e as variáveis Asaas, sem JWT, credenciais do PLATFORM_ADMIN ou
  chaves de criptografia da API. A execução de validação iniciou o container,
  registrou `billing reconciliation completed`, encerrou com sucesso e voltou
  ao estado `Ready`. Confirmar que a agenda temporária `*/5 * * * *` foi
  substituída pela agenda horária definitiva `0 * * * *`.
- Durante a homologação em produção, a remoção manual de uma cobrança de boleto
  revelou que `PAYMENT_DELETED` tentava consultar novamente a cobrança já
  apagada no Asaas e deixava a projeção local como pendente. A correção passa a
  marcar a cobrança existente como `DELETED`, limpar links inválidos e concluir
  o webhook sem cancelar automaticamente a assinatura. Após o deploy, reentregar
  o evento falho no Asaas para corrigir a cobrança já afetada.
- O fluxo ponta a ponta já foi exercitado no Sandbox com boleto e Pix: criação
  de cliente, assinatura, cobrança, confirmação, atraso, tolerância manual,
  bloqueio financeiro, recuperação após pagamento e cancelamento.
- O teste de estorno do Pix foi solicitado com sucesso no Asaas para a cobrança
  de homologação. Ainda falta aguardar o processamento e confirmar no EngageFit
  o webhook `PAYMENT_REFUNDED`, a suspensão da assinatura, o bloqueio do
  acesso financeiro e a revogação das sessões do owner. Se o Sandbox não tiver
  saldo para devolver, criar/confirmar outra cobrança fictícia e repetir o
  estorno; as tarifas não são devolvidas pelo provedor.
- O cancelamento foi validado funcionalmente. A tela de login exibiu a mensagem
  genérica de assinatura pendente/vencida; melhoria opcional: diferenciar
  explicitamente `assinatura cancelada` de atraso ou tolerância vencida.
- O teste inicial com plano de R$ 1,00 falhou por regra de valor mínimo do
  Asaas. Para boleto e Pix, manter planos de homologação com pelo menos
  R$ 10,00 por cobrança.
- A melhoria de diagnóstico de erros do Asaas foi publicada no commit
  `ecff296` (`fix: expose safe Asaas billing errors`) e enviada para
  `origin/main`. O backend agora registra operação/status/código/descrição
  sanitizada com `request_id`, sem expor credenciais ou payload bruto.

### Próximos passos de billing

1. Confirmar no Railway que o Cron de conciliação ficou definitivamente em
   `0 * * * *` depois da execução de validação.
2. Reentregar o `PAYMENT_DELETED` da cobrança de boleto removida antes do deploy
   da correção e confirmar a projeção local como `DELETED`.
3. Concluir a conferência do estorno e registrar o resultado de
   `PAYMENT_REFUNDED`, bloqueio financeiro e revogação das sessões.
4. Monitorar logs, tempo e custo do serviço Cron durante os primeiros dias.
5. Manter o piloto restrito à primeira academia até validar esses cenários e a
   rotina operacional de suporte.

### Pendências operacionais do Railway

- Confirmar `OWNER_SETUP_ENABLED=false` e remover/selar o setup token.
- Confirmar `Serverless/App Sleeping` desligado no serviço da API.
- Confirmar Custom Start Command vazio e restaurar o pre-deploy
  `/usr/local/bin/engagefit-migrate up` quando o Railway estiver executando o
  container de release de forma confiável.
- Configurar backup/PITR do PostgreSQL e executar um restore em ambiente
  isolado.
- Configurar observabilidade, alertas e limites de custo do Railway.

### WhatsApp/Twilio - envios reais homologados

- Os três templates `_2` foram aprovados na subconta Twilio `Crossfit Alados`
  para conversas iniciadas pela empresa e pelo usuário. Mapeamento em uso:
  - `copy_engagefit_falta_pouco_2` ->
    `HX38b4be2ac29ab416ab94d06357280cf4`;
  - `copy_engagefit_meta_atingida_2` ->
    `HX75f8491a6c1e7f8446677b4ad13f493d`;
  - `copy_engagefit_sentimos_sua_falta_2` ->
    `HXa819f068a4fc87df4b75c1ade6be7a39`.
- O template `Falta pouco` foi enviado pelo EngageFit e recebido com conteúdo
  renderizado corretamente nos telefones de teste finais `4712` e `0429`.
- O arquivo local
  `/home/luiz-paulo/workspace/engage-fit/alados-teste-whatsapp-deborah-totalpass.csv`
  contém somente Deborah, telefone final `0429` e 10 check-ins entre 01/07 e
  10/07; foi validado no formato aceito pelo importador TotalPass.
- Ainda falta homologar manualmente os templates `Meta atingida` e `Sentimos sua
  falta`, cada um com audiência isolada, antes de habilitar rotinas automáticas.
- Manter `AUTOMATION_WORKER_ENABLED=false` até a homologação manual; habilitar
  o worker somente depois de revisar todas as agendas.
- `WHATSAPP_ALLOW_REAL_SEND=true` está sendo usado durante a homologação
  controlada; revisar a necessidade de mantê-lo ativo depois dos testes.
- O sender dedicado funciona, porém contatos que não salvaram o número ainda
  veem o telefone em vez do Business Display Name. A foto do perfil está sendo
  preparada. A verificação empresarial da Meta pediu um domínio próprio; a
  decisão foi adiar essa frente e não criar um site apenas para satisfazer o
  formulário. Antes de retomar, avaliar verificação por documentos/suporte
  Twilio e Meta.
- Melhoria recomendada antes de uso amplo: criar ação independente de
  `Disparo manual` com preview, quantidade/telefones mascarados, confirmação
  explícita e link para auditoria.

## Checkpoint de billing Asaas em 2026-07-23

### Implementação e homologação concluídas

- Billing foi implementado como contexto isolado no backend atual, usando
  `BillingGateway` e `BillingRepository`; não há segundo servidor permanente.
- Migration `034_create_billing.sql` adiciona planos versionados, clientes,
  assinaturas, cobranças, eventos idempotentes de webhook e a projeção
  `boxes.billing_access_blocked`.
- Asaas é fonte de verdade financeira; o EngageFit mantém contrato, franquia,
  histórico espelhado e acesso. Nenhum dado de cartão passa pelo sistema.
- Webhook público autenticado:
  `POST /api/v1/webhooks/asaas`, usando o header `asaas-access-token`.
- Clientes usam `box_id` como referência externa. Assinaturas exigem
  `Idempotency-Key`, evitando duplicação em retry inclusive após falha entre o
  provedor e a persistência local.
- Pagamento confirmado/recebido libera acesso; atraso inicia tolerância;
  tolerância vencida, estorno, chargeback e cancelamento bloqueiam acesso,
  revogam sessões e impedem novas automações. O estado administrativo da
  academia permanece independente.
- A assinatura aplica automaticamente as franquias de mensagens do plano. O
  preço é mensal único, com WhatsApp incluído nos limites; não há cobrança
  separada de WhatsApp.
- Termos comerciais de plano já contratado são imutáveis. Mudanças de preço,
  franquia ou tolerância exigem nova versão.
- Conciliação está disponível no painel e no binário
  `/usr/local/bin/engagefit-billing-reconcile`; deve ser agendada ao menos a
  cada hora.
- Frontend recebeu `Financeiro` para `PLATFORM_ADMIN` e `Plano e cobranças` para
  owner.
- Runbook completo: `docs/asaas-billing-runbook.md`. Arquitetura:
  `.ai/billing-design.md` e seção 14.1 de `docs/system-design.md`.
- Configuração permanece desligada por padrão com
  `FEATURE_BILLING_ENABLED=false`.
- Rejeições do Asaas retornam mensagem administrativa segura e geram o log
  estruturado `billing_provider_request_failed`, correlacionado por
  `request_id`, sem registrar payload bruto ou credenciais.
- Backend `6604ba9` e frontend `ded9c31` foram publicados e implantados no
  Railway. `FEATURE_BILLING_ENABLED=true` está efetivo e `/api/v1/capabilities`
  confirmou `billing: true`.
- A conta Sandbox possui webhook v3 ativo, fila ativa e envio sequencial para o
  endpoint público do EngageFit, autenticado com token próprio.
- Cliente, assinatura mensal por boleto, cobrança e confirmação de pagamento
  foram homologados ponta a ponta. O teste inicial de R$ 1,00 foi rejeitado
  pelo limite do Asaas; com plano de R$ 10,00 a cobrança foi criada e recebida
  corretamente.

### Antes de ativar em produção

1. Homologar vencimento, tolerância, suspensão, estorno, cancelamento e
   conciliação no sandbox.
2. Criar um job periódico usando o binário de conciliação.
3. Concluir a aprovação cadastral e habilitação dos meios de pagamento na conta
   Asaas real.
4. Criar chave e webhook exclusivos de produção.
5. Trocar para a URL e chave de produção e pilotar com uma
   academia.

## Checkpoint de onboarding Twilio e teste produtivo em 2026-07-23

### Estado concluido

- O ciclo de vida de academias foi implementado e publicado: criacao pelo `PLATFORM_ADMIN`, edicao de nome, suspensao, reativacao e arquivamento, com bloqueio de login/automacoes quando inativa e auditoria administrativa.
- Backend publicado no commit `7411e05` (`feat: add academy lifecycle management`) e frontend no commit `95fd575` (`feat: add academy lifecycle admin interface`). A migration `033_add_box_lifecycle.sql` acompanha o backend.
- A academia Crossfit Alados foi criada como tenant e sua configuracao passou a ser administrada em `Administracao -> Academias -> Crossfit Alados -> Conexao`.
- Foi criada a subconta Twilio `Crossfit Alados` dentro da conta principal. O sender WhatsApp `+55 18 99671-0587` foi migrado/conectado e chegou ao estado `Online` no Console da Twilio.
- A conexao dedicada do Alados foi salva no EngageFit usando a subconta, e o botao `Testar conexao` passou. Account SID/Auth Token permanecem somente no armazenamento cifrado e nao devem ser registrados neste handoff.
- A subconta compartilha automaticamente o saldo da conta Twilio principal; nao existe transferencia de credito. Recursos e relatorios de uso continuam separados por subconta, mas a cobranca reduz o saldo pai.
- `FEATURE_WHATSAPP_ENABLED=true` e necessario para expor a funcionalidade. `WHATSAPP_PLATFORM_ENABLED` controla apenas o numero compartilhado do EngageFit e nao interfere em academias configuradas como `dedicated`.
- A campanha de homologacao `Campanha do Testo` foi criada para 01/07/2026 a 31/07/2026, ativa, com meta observada de 12 check-ins.
- Foi criado o artefato local `/home/luiz-paulo/workspace/engage-fit/alados-teste-whatsapp-totalpass.csv`, aceito pelo importador como TotalPass. Ele contem um unico aluno de teste, telefone final `4712`, identificador exclusivo e 10 check-ins entre 01/07 e 10/07. Ainda confirmar a importacao antes de considerar o cenario ativo.

### Templates do Alados

- Content Templates criados dentro da subconta correta:
  - `engagefit_falta_pouco`;
  - `engagefit_meta_atingida`;
  - `engagefit_sentimos_sua_falta`.
- Em 22/07/2026 os tres templates ainda apareciam como `Under Review`. A elegibilidade `WhatsApp business initiated` ainda nao estava verde.
- Cada template possui um Content SID proprio `HX...` na subconta. Um Content SID da conta principal nao pode ser usado pela subconta.
- Depois da aprovacao real na Twilio, cadastrar cada `HX...` no card correspondente da tela WhatsApp e mudar o status manual para `Aprovado`. Enquanto estiver `Under Review`, usar `Pendente`; nao marcar como aprovado antecipadamente.
- Mapeamento:
  - `engagefit_falta_pouco` -> `Falta pouco para alcancar a meta`;
  - `engagefit_meta_atingida` -> `Voce atingiu a meta`;
  - `engagefit_sentimos_sua_falta` -> `Sentimos sua falta`.

### Passos pendentes para o primeiro envio real

1. Confirmar os tres templates como `Approved` na Twilio; para o primeiro teste basta `engagefit_falta_pouco`.
2. Cadastrar o `HX...` correspondente no EngageFit, selecionar `Aprovado` e salvar.
3. Importar `alados-teste-whatsapp-totalpass.csv` escolhendo a origem `TotalPass`; resultado esperado na primeira importacao: 1 aluno e 10 check-ins.
4. Abrir `Campanhas -> Campanha do Testo -> Participantes` e confirmar o aluno de teste em `10/12`, status `Proximo`.
5. Confirmar que nenhum outro aluno esta `Proximo`. O disparo atual envia para toda a audiencia elegivel, nao apenas para o primeiro aluno.
6. Na tela WhatsApp, clicar `Ativar mensagem` somente no template `Falta pouco`.
7. Confirmar no Railway `WHATSAPP_ALLOW_REAL_SEND=true` apenas quando destinatario, template e limites estiverem revisados. `FEATURE_WHATSAPP_ENABLED=true` tambem deve permanecer ativo.
8. Criar uma rotina pausada chamada `Teste manual - falta pouco`, modo `send_almost_there`, `allow_resend=false` e `enabled=false`. O botao `Executar` funciona mesmo com a rotina pausada e com `AUTOMATION_WORKER_ENABLED=false`.
9. Executar uma unica vez e conferir: total 1, enviados 1, falhas 0, destinatario final `4712`, `provider_message_sid`/status na auditoria e mensagem recebida no aparelho.
10. Manter `AUTOMATION_WORKER_ENABLED=false` ate revisar todas as agendas. Somente depois da homologacao alterar para `true`.

### Lacuna de produto/UX identificada

- A tela Automacao nao oferece uma acao independente de `Disparo manual`. Para executar manualmente hoje, o owner precisa criar antes uma rotina, mesmo que pausada, e entao usar `Executar`.
- O fluxo manual nao mostra previamente a lista/quantidade efetiva de destinatarios. Isso e especialmente perigoso em production porque uma campanha envia para toda a audiencia elegivel.
- Proxima melhoria recomendada: bloco `Disparo manual` com selecao de campanha e template, preview usando o endpoint de preview ja existente, total de destinatarios, telefones mascarados, bloqueio quando template/conexao estiverem indisponiveis, confirmacao explicita e link para auditoria.
- Antes dessa melhoria, toda homologacao deve usar tenant/campanha isolados e conferir a lista de participantes manualmente antes do clique.

### Configuracao de automacao recomendada durante a homologacao

```env
FEATURE_AUTOMATION_ENABLED=true
AUTOMATION_WORKER_ENABLED=false
AUTOMATION_WORKER_INTERVAL_SECONDS=60
AUTOMATION_STALE_RUN_MINUTES=120
AUTOMATION_CATCHUP_WINDOW_MINUTES=15
```

- `FEATURE_AUTOMATION_ENABLED` expoe a tela e os endpoints.
- `AUTOMATION_WORKER_ENABLED` liga execucao agendada; deve continuar falso no primeiro teste.
- O envio manual continua disponivel pela rotina pausada.
- Confirmar no Railway os valores efetivos; eles nao foram lidos diretamente do ambiente nesta sessao.

## Checkpoint de revisão completa de UX em 2026-07-22

- Resultado revisado e aprovado pelo usuário após a implementação completa das etapas propostas.
- O frontend recebeu uma revisão transversal de hierarquia visual, acessibilidade, navegação e responsividade, sem alterar contratos da API nem o schema do banco.
- O contraste da ação principal foi corrigido, controles móveis ganharam alvos maiores, cards ficaram mais leves e a navegação passou a ser agrupada em Visão geral, Operação, Engajamento e Gestão; no mobile, o menu agora abre em drawer.
- O Dashboard passou a priorizar o que exige atenção hoje, com resumo secundário compacto e menos blocos de mesmo peso.
- Campanhas agora abre pela listagem e separa Visão geral, Participantes e Ajustes; participantes mantêm paginação local e apresentação própria no mobile, e toda a configuração é salva por uma única ação.
- Brindes, Alunos, Check-ins e Relatórios ganharam visualizações responsivas que mantêm dados e ações essenciais visíveis sem depender de tabela horizontal; Alunos também recebeu filtros e paginação local.
- Importações ganhou upload por seleção/arrastar, origem explícita, retorno da última importação e histórico com data/status. WhatsApp ganhou indicadores de consumo, confirmação de envio e menos detalhes técnicos expostos.
- Automação, E-mail e Treino do dia passaram a separar configuração/operação de histórico; execuções de automação ficaram compactas e expansíveis. A arquitetura já segmentada de Configurações e Administração foi preservada.
- O fluxo E2E real foi atualizado para os novos controles. Após a primeira esteira apontar seletores ambíguos entre as versões mobile e desktop no DOM, os seletores foram tornados explícitos para os elementos visíveis. Validações executadas: TypeScript/build Vite, Playwright mockado, `git diff --check`, revisão visual desktop/mobile e os dois fluxos Playwright reais contra API/PostgreSQL temporários; owner e PLATFORM_ADMIN passaram.
- A correção dos seletores foi publicada no frontend pelo commit `3623924` (`test: target visible responsive elements in real e2e`). A execução `Frontend CI #6` (`29923876178`) concluiu com sucesso, incluindo o step `Real browser E2E with PostgreSQL`; o banco local temporário `boxengage_ux_e2e_20260722` usado na reprodução foi removido depois dos testes.
- Nenhuma migration é necessária para esta entrega.

## Checkpoint de campanhas e consulta de check-ins em 2026-07-21

- A lista de progresso da campanha deixou de cortar silenciosamente em 8 alunos e passou a ter paginacao local de 10 itens.
- Um aluno agora participa do progresso somente quando possui ao menos um check-in dentro das datas inclusivas da campanha e existe meta para sua origem.
- O recálculo substitui logicamente os snapshots em transacao, removendo progressos obsoletos; entregas pendentes de brinde tambem sao sincronizadas, sem apagar entregas ja realizadas.
- Nova tela `Check-ins` consulta um intervalo de datas e mostra quantidade, primeira e ultima presenca por aluno, com busca, plataforma, ordenacao e paginacao no frontend.
- Novo endpoint: `GET /api/v1/checkins/summary?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`; a agregacao ocorre no PostgreSQL e a API continua sem paginacao neste primeiro momento.

## Checkpoint de deploy no Railway em 2026-07-21

### Estado implantado

- Backend, frontend e PostgreSQL foram criados no projeto Railway exibido como `motivated-playfulness`, no mesmo ambiente `production`.
- Servicos usados: `engage-fit-api`, `engage-fit-web` e `Postgres`.
- Arquitetura efetiva: navegador -> `engage-fit-web` publico -> `/api` via Nginx/rede privada -> `engage-fit-api` privado -> `Postgres` privado.
- Dominio publico validado: `https://engage-fit-web-production.up.railway.app`.
- API e PostgreSQL permanecem sem dominio HTTP publico; a API e acessada pelo frontend usando `BACKEND_URL=http://${{engage-fit-api.RAILWAY_PRIVATE_DOMAIN}}:8080`.
- Frontend usa `PORT=8080`, nao define `VITE_API_BASE_URL` e usa `VITE_CSRF_COOKIE_NAME=engagefit_session_csrf`.
- Backend usa `PORT=8080`, `HTTP_HOST=0.0.0.0` e `DATABASE_URL=${{Postgres.DATABASE_URL}}`; segredos production foram configurados diretamente no Railway e nao foram registrados no Git.
- Healthcheck efetivo dos dois servicos ficou em `/health`, com timeout de 300 segundos. O backend atual tambem implementa `/health/live` e `/health/ready`, mas `/health` foi mantido como rota compativel entre revisoes.
- Features opcionais e efeitos externos permaneceram desligados durante a homologacao: WhatsApp, e-mail, automacao, treinos, LLM e envios reais.

### Migrations e release

- O pre-deploy `/usr/local/bin/engagefit-migrate up` falhou repetidamente antes de iniciar o container, sem stdout/stderr. O diagnostico automatico do Railway classificou a falha como erro transitorio de infraestrutura, mas chegou a afirmar incorretamente que as migrations ja haviam sido aplicadas.
- A evidencia real apareceu no startup da API: `relation "users" does not exist`, confirmando que o banco ainda estava vazio.
- Para inicializar o banco, foi usado temporariamente como Custom Start Command: `sh -c '/usr/local/bin/engagefit-migrate up && exec /usr/local/bin/engagefit-api'`.
- As 32 migrations foram aplicadas e a API passou a iniciar com o schema completo. O migrator e idempotente e usa historico/checksum/advisory lock.
- Estado operacional a conferir no Railway: deixar o Custom Start Command vazio depois da inicializacao e restaurar o pre-deploy quando o agendamento de containers estiver estavel. Enquanto o pre-deploy estiver desativado, nenhuma release com migration nova deve ser publicada sem executar explicitamente o migrator.

### Incidentes encontrados e correcoes

- O primeiro backend ativo era uma revisao antiga porque quatro commits locais ainda nao estavam no GitHub. Isso explicava `/api/v1/capabilities` ausente, `/health/live` ausente e comportamento antigo do setup. O backend foi enviado de `9170a69` ate `a562871` e depois recebeu os ajustes de CI/formato descritos abaixo.
- `GET /api/v1/setup/owner` retorna 404 por desenho; o onboarding usa `POST`. Na revisao nova, production tambem exige `OWNER_SETUP_ENABLED=true` e `X-Setup-Token` valido.
- O Nginx do frontend resolvia o dominio privado da API somente no startup. Cada redeploy do backend alterava seus IPs privados; o frontend continuava chamando IPs antigos e passava a responder 502/504 depois de esperas de 10/20 segundos.
- Correcao permanente no frontend, commit `d4650db`: resolver interno `[fd12::10]`, cache DNS de 10 segundos, `ipv6=off` e `proxy_pass` por variavel para forcar resolucao em runtime. A imagem Docker foi reconstruida e `nginx -t` passou.
- Depois da correcao, novos IPs privados do backend deixam de exigir restart/redeploy manual do frontend.
- Se `Serverless/App Sleeping` estiver habilitado na API, deve ser desabilitado para resposta previsivel em producao; confirmar esse toggle no Railway.

### CI corrigido e validado

- Frontend commit `81cee85`: readiness do E2E real passou de 30 para 90 segundos, usa `127.0.0.1` e detecta encerramento antecipado da API. A falha anterior era `curl` exit code 7 enquanto o segundo `go run` ainda compilava no runner frio.
- Backend commit `4172508`: corrigiu YAML invalido causado pelo `:` no comando inline de idempotencia das migrations e aplicou o mesmo readiness robusto ao smoke da API.
- Backend commit `8b5443b`: aplicou `gofmt` em `internal/adapters/whatsapp/provider_gateway.go`.
- CI do frontend passou no run `29884280635`: build, Playwright mockado e Playwright real com PostgreSQL/API.
- CI do backend passou no run `29884561040`: modulos, formato, vet, 32 migrations, idempotencia, PostgreSQL/race detector, smoke HTTP, binarios e scripts.
- Estado do codigo homologado antes desta atualizacao documental: backend `8b5443b`; frontend `81cee85`; ambos estavam sincronizados com `origin/main` e sem alteracoes pendentes.

### Homologacao funcional em producao

- `/health` do frontend respondeu `200` com `{"status":"ok"}`.
- O proxy publico `/api/v1/capabilities` chegou corretamente a API privada depois do deploy da revisao atual.
- A primeira academia e a conta owner foram criadas pelo onboarding protegido e o fluxo do owner foi testado manualmente no dominio de producao.
- A conta `PLATFORM_ADMIN`, criada/atualizada no startup a partir de `PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD`, foi autenticada pelo mesmo formulario de login e redirecionada para `#admin-messaging`; o fluxo administrativo tambem foi testado manualmente.
- Credenciais do owner, setup token, JWT, chave de criptografia e credenciais administrativas permanecem somente nas variaveis/controle do operador e nao devem entrar neste handoff.
- Acao de seguranca a confirmar imediatamente: depois do onboarding, definir `OWNER_SETUP_ENABLED=false` e remover ou selar `OWNER_SETUP_TOKEN`. Repetir o POST deve retornar 404. A confirmacao dessa desativacao nao apareceu nesta sessao.

### Proximos passos operacionais

1. Confirmar no Railway `OWNER_SETUP_ENABLED=false` e setup token removido/selado.
2. Confirmar `Serverless/App Sleeping` desligado em `engage-fit-api`.
3. Confirmar Custom Start Command vazio. Se ainda estiver executando migration + API, pode permanecer apenas como contingencia curta, pois e idempotente, mas nao e o desenho final.
4. Restaurar `/usr/local/bin/engagefit-migrate up` no pre-deploy quando o Railway voltar a iniciar o container de release de forma confiavel; validar `migration complete: 0 applied`.
5. Configurar backup/PITR do PostgreSQL e executar restore real em ambiente isolado antes de dados de clientes.
6. Manter todas as integracoes e envios reais desligados ate homologacao individual dos provedores.
7. Configurar observabilidade/alertas e limites de custo do Railway antes do piloto.

## Checkpoint de prontidao final da aplicacao em 2026-07-21

- Checklist executavel criado em `.ai/application-readiness-checklist.md`, separado por gates P0, P1 e P2.
- StatusCallback e demais evolucoes Twilio ficaram explicitamente fora desta fase, assim como Railway, dominio/TLS, banco gerenciado, backup/restore, Grafana Cloud e validacoes juridicas.
- Baseline local passou: modulos Go, formatacao, `go vet`, 32 migrations em PostgreSQL vazio, idempotencia das migrations, testes com integracoes PostgreSQL, smoke HTTP, quatro binarios, scripts Node, `npm ci`, build TypeScript/Vite e `git diff --check`.
- O smoke confirmou health/readiness, setup, auth, importacao, isolamento entre tenants, campanha, brinde, dashboard, privacidade, logout e revogacao. API encerrou graciosamente e os bancos temporarios foram removidos.
- Todos os gates P0/P1 foram concluidos: sessao HttpOnly/CSRF, CSP/headers, kill switches, metricas/alertas, erros com `request_id`, Playwright e integracoes PostgreSQL adicionais.
- Playwright passou contra API/PostgreSQL reais com gateways externos desligados: owner percorreu importacao, campanha, recalculo, dashboard, brinde, privacidade, troca de senha e logout; PLATFORM_ADMIN percorreu administracao e reset de senha.
- O teste de destinatarios encontrou e corrigiu o mapeamento GORM de `provider_message_sid`; a regressao esta coberta em PostgreSQL real.
- A revisão do manual encontrou e corrigiu a permissão de efeito externo: SMTP e WhatsApp reais agora exigem `*_ALLOW_REAL_SEND=true` também em production; mocks continuam liberados. Testes impedem regressão.
- Auditoria final reconstruiu as imagens, confirmou usuarios sem privilegios, build info/health, headers do frontend, `SIGTERM`, retencao dry-run e configuracao production fail-fast.
- O proximo trabalho e exclusivamente a fase de infraestrutura descrita em `docs/railway-deployment-checklist.md`, alem da Twilio que continua fora deste escopo.

## Checkpoint de transferencia de conhecimento em 2026-07-21

- `docs/system-design.md` foi criado como manual canonico para o proprietario tecnico e deve ser atualizado no mesmo pull request que alterar arquitetura, regra de negocio, seguranca, tenancy, privacidade, efeitos externos ou operacao.
- O manual documenta atores, arquitetura, pipeline HTTP, sessao, capabilities, modelo de dados, importacao, campanhas, progresso, brindes, risco, comunicacao, automacao, privacidade, criptografia, migrations, runtime, observabilidade, configuracao, API, testes, trade-offs e dividas conhecidas.
- A secao 26 contem uma trilha pratica de estudo e exercicios locais; a secao 27 funciona como verificacao de dominio do sistema; a secao 29 aponta os arquivos-fonte de cada assunto.
- A revisao cruzada entre documentacao e implementacao encontrou uma falha nos kill switches: production liberava SMTP e WhatsApp reais mesmo com as flags desligadas. Isso foi corrigido e coberto por testes.
- Commits deste fechamento: `b3245a2` (`fix: require explicit permission for real sends`) e `eb87a9f` (`docs: add canonical system engineering manual`).
- Validacao final: `go test ./...`, `go vet ./...`, `git diff --check` e conferencia dos links locais do manual passaram; os dois repositorios ficaram limpos.
- Nenhum push ou deploy foi executado. O proximo passo recomendado e percorrer a trilha de conhecimento localmente e depois executar o checklist do Railway, mantendo a integracao efetiva com Twilio fora do escopo por enquanto.

## Direcao concluida - prontidao da aplicacao antes do deploy

Decisao registrada em: 2026-07-20

Objetivo da proxima fase:

- Nao fazer o deploy agora.
- Resolver primeiro tudo que puder ser tratado no codigo e no desenho operacional da aplicacao.
- Manter infraestrutura desacoplada e adiar decisoes definitivas de dominio, TLS, banco gerenciado, backups do provedor e pipeline de deploy.
- Railway e o destino de deploy mais provavel, portanto a aplicacao deve funcionar bem em container efemero, usar configuracao por variaveis de ambiente, escrever logs em `stdout` e encerrar corretamente ao receber `SIGTERM`.
- A configuracao efetiva do numero/canal WhatsApp sera tratada separadamente e nao bloqueia o trabalho de prontidao da aplicacao.

### Bloqueadores de aplicacao antes de um piloto real

Checkpoint de seguranca em 2026-07-20:

- Criacao de box + owner passou a ser transacional; falha ao gravar o owner desfaz o box e o e-mail e verificado antes da criacao.
- `POST /api/v1/setup/owner` agora e controlado por `OWNER_SETUP_ENABLED`, fica desligado por padrao em production e exige `OWNER_SETUP_TOKEN` forte quando habilitado nesse ambiente.
- Configuracao production falha cedo com banco ausente, JWT fraco, administrador ausente ou limites HTTP invalidos.
- Login e setup possuem rate limit por IP e identidade normalizada/hasheada, com resposta `429` e `Retry-After`.
- Body JSON e upload possuem limites configuraveis; imports limitam arquivo, linhas, colunas e tamanho descomprimido das partes relevantes do XLSX.
- Usuario autenticado pode trocar a propria senha em `PUT /api/v1/auth/password` e na tela Configuracoes; minimo de 12 caracteres e confirmacao da senha atual.
- PLATFORM_ADMIN pode redefinir a senha do owner de uma academia em `PUT /api/v1/admin/boxes/:id/owner-password` e na aba `Acesso`; motivo obrigatorio e auditoria sem registrar a senha.
- Novas configuracoes: `HTTP_MAX_BODY_BYTES`, `IMPORT_MAX_UPLOAD_BYTES`, `LOGIN_RATE_LIMIT_*`, `SETUP_RATE_LIMIT_*`, `OWNER_SETUP_ENABLED` e `OWNER_SETUP_TOKEN`.
- Validacoes executadas: `go test ./...`, `node --check scripts/demo-seed.mjs`, TypeScript e build Vite.
- Ainda pendente para escala horizontal: tornar o rate limit compartilhado entre replicas; no piloto de uma replica o controle atual e efetivo. Ampliar os testes E2E segue na frente de qualidade.

Checkpoint de sessao/runtime em 2026-07-20:

- Migration `030_add_user_auth_version.sql` adiciona `users.auth_version`; aplicada no PostgreSQL local.
- JWT inclui `auth_version` e o middleware confirma usuario, tenant, papel e versao no banco em toda requisicao autenticada.
- Troca/redefinicao de senha e rotacao do PLATFORM_ADMIN incrementam `auth_version`, revogando tokens anteriores imediatamente.
- Logout agora incrementa `auth_version`; o frontend chama o endpoint antes de remover a sessao local.
- Smoke isolado confirmou: login `200`, token valido `200`, logout `204`, token antigo `401` e novo login `200`; tenant temporario removido.
- `TRUSTED_PROXIES` controla explicitamente quais proxies podem fornecer o IP do cliente; vazio nao confia em `X-Forwarded-For`.
- API usa `http.Server` com timeouts configuraveis e shutdown gracioso em `SIGINT`/`SIGTERM`.
- Worker recebe o contexto de encerramento, para de aceitar ticks e sinaliza conclusao antes do fim do processo.
- PostgreSQL possui pool configuravel, ping no startup e readiness com timeout.
- Endpoints: `/health/live`, `/health/ready` e alias `/health`; readiness retorna `503` quando PostgreSQL nao responde.
- A porta `PORT` do provedor e aceita como fallback de `HTTP_PORT`.
- Docker final executa como usuario `engagefit`, inclui CA certificates e `tzdata` e possui healthcheck dinamico.
- Validacao real confirmou liveness/readiness `200`, logs `shutdown_started`/`shutdown_completed`, build da imagem e usuario final sem privilegios.

Checkpoint de observabilidade em 2026-07-20:

- Backend instrumentado com OpenTelemetry para traces, metricas e logs via OTLP HTTP, habilitado por `OTEL_ENABLED` e configuracao padrao `OTEL_EXPORTER_OTLP_*`.
- Requests HTTP geram traces, metricas de volume/latencia/em andamento por rota normalizada e logs correlacionados por `request_id`, `trace_id` e `span_id`.
- Queries GORM e chamadas HTTP externas de Twilio, Meta e OpenAI participam dos traces; variaveis SQL nao entram nos spans.
- Runtime Go e pool do PostgreSQL expoem metricas sem labels de alta cardinalidade.
- Logs HTTP deixaram de registrar IP e URL concreta; `X-Request-ID` recebido e limitado a 128 caracteres seguros para evitar injecao em logs.
- `GET /metrics` existe para Prometheus, fica desligado por padrao em production e, se habilitado nesse ambiente, exige `PROMETHEUS_BEARER_TOKEN` com ao menos 32 caracteres.
- Stack local opcional em `docker-compose.observability.yml`: Grafana, Prometheus, Loki, Tempo e OpenTelemetry Collector, com datasources e painel `EngageFit - Visao geral` provisionados.
- Comandos: `make observability-up`, `make observability-down` e `make observability-logs`; instrucoes em `observability/README.md`.
- Railway pode manter metricas de CPU/RAM/disco/rede e a API pode enviar OTLP diretamente ao Grafana Cloud gratuito sem alteracao de codigo.
- Smoke ponta a ponta confirmou request no Prometheus, log no Loki sem IP/URL concreta, trace no Tempo e navegacao por `trace_id`; stack local foi desligada depois do teste e seus volumes foram preservados.
- Testes de middleware cobrem protecao Bearer do endpoint de metricas e saneamento de request ID; `go test ./...` passa.
- Ainda pendentes nesta frente: metricas de negocio/gateways, alertas provisionados e informacao de build; entram junto das automacoes/qualidade antes de encerrar a prontidao.

Checkpoint de migrations versionadas em 2026-07-20:

- O loop que reaplicava todos os SQLs foi substituido pelo migrator proprio em `migrations/migrator.go` e pelo binario `cmd/migrate`.
- `schema_migrations` registra versao, nome, SHA-256, horario e tempo de execucao; SQL aplicado nao pode ser alterado silenciosamente depois.
- Cada migration roda em transacao e um advisory lock do PostgreSQL impede duas releases de migrar simultaneamente.
- Comandos: `make migrate-up`, `make migrate-status` e, apenas para adotar banco legado ja conferido, `make migrate-baseline VERSION=N`.
- Banco nao vazio sem historico e recusado com instrucao de baseline; baseline e recusado em banco vazio e exige versao explicita.
- A imagem passa a incluir `/usr/local/bin/engagefit-migrate`; em release ele deve executar `up` separadamente antes da API. A API continua sem alterar schema no startup.
- Banco local foi conferido pelos marcos das migrations 028-030 e recebeu baseline ate 030. Nova execucao aplicou zero migrations.
- Smoke em banco PostgreSQL temporario real aplicou as 30 migrations, segunda execucao aplicou zero, confirmou tabelas finais e removeu o banco temporario.
- Testes unitarios cobrem ordenacao, checksum, filename e lacunas da sequencia; `go test ./...` passa.

Checkpoint de criptografia de credenciais em 2026-07-20:

- Credenciais dedicadas do WhatsApp (`api_key_encrypted`) e senhas SMTP (`password_encrypted`) agora sao cifradas no limite dos repositories antes de chegar ao PostgreSQL e decifradas somente para uso interno.
- Envelope versionado `enc:v1:<key_id>:<payload>` usa AES-256-GCM com nonce aleatorio e associated data vinculando tipo do segredo, `box_id` e campo; copiar ciphertext entre tenants/campos falha autenticacao.
- Keyring fica exclusivamente em `DATA_ENCRYPTION_KEYS` e a chave de escrita em `DATA_ENCRYPTION_ACTIVE_KEY_ID`; ambas sao obrigatorias em production.
- Runtime com criptografia configurada rejeita plaintext legado. Sem chaves, somente fora de production, a API preserva compatibilidade local e emite `data_encryption_disabled` em nivel warn.
- Binario `/usr/local/bin/engagefit-rotate-secrets` e `make rotate-secrets` convertem plaintext legado e recifram valores de chave antiga para a ativa dentro de uma unica transacao com locks de linha.
- Rotacao segura: adicionar chave nova + antiga ao keyring, tornar a nova ativa, executar o comando, atualizar todas as instancias e somente entao remover a antiga.
- Smoke em PostgreSQL temporario confirmou plaintext -> chave `old`, segunda execucao idempotente (`0` alteracoes) e `old` -> `new`, sem exibir os segredos; banco temporario removido.
- Testes cobrem round-trip, adulteracao de associated data, keyring invalido, leitura de chave antiga para rotacao e rejeicao de plaintext no runtime; `go test ./...` e `go vet ./...` passam.

Checkpoint de concorrencia/idempotencia das automacoes em 2026-07-20:

- Migration `031_add_automation_idempotency.sql` adiciona `schedule_id`, `scheduled_for` e `execution_key` unica por academia em `automation_runs`; aplicada localmente pelo novo migrator.
- Worker faz claim atomico do slot no PostgreSQL antes de qualquer efeito. Smoke com 20 claims concorrentes confirmou exatamente um vencedor; multiplas replicas podem manter o worker habilitado.
- Cada slot recebe chave deterministica `schedule:<id>:<horario>`. Repeticoes manuais aceitam `Idempotency-Key`; o frontend gera uma chave por clique e uma repeticao com a mesma chave retorna o run existente.
- Smoke HTTP confirmou primeira criacao `201`, replay `200`, mesmo `run.id` e `idempotent_replay=true`; registro temporario removido.
- `daily-automation.mjs` usa chave diaria por timezone (ou `DAILY_AUTOMATION_IDEMPOTENCY_KEY`) e encerra sem repetir import/recalculo/envio ao detectar replay.
- Estrategia e at-most-once por slot: se a instancia morrer com resultado externo incerto, o worker nao repete mensagens automaticamente. O run fica `running`, passa a `failed` apos `AUTOMATION_STALE_RUN_MINUTES` e exige revisao antes de uma nova chave manual.
- Janela `AUTOMATION_CATCHUP_WINDOW_MINUTES` permite recuperar atrasos curtos de startup/deploy sem executar rotinas horas depois do horario.
- Atualizacao administrativa da agenda nao sobrescreve mais `last_run_at`, eliminando corrida com o claim do worker.
- Nomes, modo, `HH:MM`, timezone IANA e dias da semana sao validados; entradas invalidas retornam `400`.
- Metricas OTel contam/duram runs por status, e logs estruturados registram conclusao/falha/stale sem PII.
- Logger GORM foi substituido por logger seguro: `record not found` nao gera ruido, SQL/parametros e erros brutos do PostgreSQL nao vao para logs; HTTP registra apenas contagem de erros, nao a mensagem bruta.
- `go test ./...`, `go vet ./...`, `node --check scripts/daily-automation.mjs`, TypeScript e build Vite passam.

Checkpoint de qualidade e CI em 2026-07-20:

- Backend e frontend agora possuem workflows GitHub Actions separados. Backend valida formato, modulos, `go vet`, migrations em PostgreSQL 16 vazio e idempotentes, testes com race detector/integracao, smoke HTTP, scripts e binarios; frontend executa instalacao reproduzivel, build TypeScript/Vite e Playwright mockado e real com PostgreSQL 16/API.
- Testes de autorizacao cobrem token com tenant/versao divergente e separacao OWNER/PLATFORM_ADMIN. Testes PostgreSQL cobrem claim concorrente de automacao e isolamento de alunos e recompensas entre duas academias.
- Auditoria durante os testes encontrou e corrigiu isolamento ausente em metas/progresso/brindes de campanha e nas listagens de destinatarios de WhatsApp, e-mail e Treino do dia. Recursos de outra academia passam a responder como nao encontrados.
- Login agora normaliza espacos e caixa do e-mail da mesma forma que o onboarding; o smoke capturou o caso em que owner criado com letra maiuscula nao conseguia entrar imediatamente.
- `scripts/smoke-api.mjs` cobre health/readiness, criacao de duas academias, login, `/auth/me`, campanha, meta, brinde, tentativas cruzadas entre tenants, recalculo, dashboard, logout e revogacao do token.
- Smoke completo passou em PostgreSQL temporario com todas as migrations; a API foi encerrada graciosamente e o banco temporario removido. `go test ./...`, `go vet ./...`, testes de integracao PostgreSQL e build frontend passam localmente.
- O race detector fica validado pelo runner Linux do CI; o ambiente local atual nao possui compilador C (`gcc`) para executar `go test -race`.

Checkpoint de privacidade/LGPD na aplicacao em 2026-07-20:

- Migration `032_add_student_privacy_controls.sql` adiciona estado/origem/data da preferencia de contato, data de anonimizacao, auditoria de privacidade e supressoes de identidade; aplicada no PostgreSQL local.
- A tela `Alunos` permite marcar `Autorizado`, `Nao contatar` ou `Nao informado`, exportar JSON e anonimizar com motivo e dupla confirmacao.
- `opted_out` e alunos anonimizados sao excluidos de todas as audiencias de WhatsApp, e-mail e Treino do dia. O envio de um rascunho antigo de treino volta a conferir a elegibilidade antes de reservar limite/chamar o gateway.
- Exportacao inclui cadastro, check-ins, progresso e historico de destinatarios e grava evento de auditoria sem duplicar PII na auditoria.
- Anonimizacao transacional limpa nome, e-mail, telefone e erros/destinos historicos, pausa contato, preserva metricas anonimas e registra hash de supressao. Reimportacoes posteriores da mesma identidade sao ignoradas.
- Binario `engagefit-privacy-retention` e comandos `make privacy-retention-dry-run`/`make privacy-retention-apply` tratam destinatarios, logs LLM, automacoes, importacoes/check-ins e auditoria com prazos configuraveis em `PRIVACY_RETENTION_*`; dry-run e o comportamento padrao.
- Politica inicial, fluxo de titular, responsabilidades propostas e procedimento de incidente estao em `docs/privacy-runbook.md`; classificacao controlador/operador, contrato, bases legais e textos publicos ainda exigem validacao juridica antes do go-live.
- Teste PostgreSQL confirma exportacao, anonimizacao, opt-out e supressao. Smoke HTTP completo confirma importacao, exportacao, anonimizacao e bloqueio da reimportacao em banco temporario com as 32 migrations; banco removido ao final.

Checkpoint de UX do owner e escopo inicial de producao em 2026-07-21:

- A pagina `Configuracoes` do owner foi reorganizada em tres secoes: `Alunos em risco`, `Acesso e seguranca` e `Integracao WhatsApp`, exibindo um assunto por vez em navegacao responsiva.
- Regras de risco passaram a usar textos mais diretos, explicacoes por campo e uma pre-visualizacao da regra efetiva antes de salvar.
- Troca de senha foi simplificada e mantem a exigencia de ao menos 12 caracteres, confirmacao e aviso de encerramento das sessoes abertas.
- Integracao WhatsApp do owner passou a mostrar somente resumo operacional, disponibilidade, tipo de conexao, remetente, ultima atualizacao e teste. Credenciais e campos tecnicos administrados pela plataforma nao aparecem mais como controles desabilitados.
- Os menus `Treino do dia` e `E-mail` foram ocultados temporariamente da navegacao do owner para a primeira publicacao em producao.
- As paginas, rotas e implementacoes de `Treino do dia` e `E-mail` continuam preservadas no codigo para reativacao futura; esta decisao altera apenas sua exposicao no menu neste momento.
- Validacoes executadas: TypeScript, build de producao Vite e `git diff --check`.

Seguranca e acesso:

1. Proteger ou remover de producao o endpoint publico `POST /api/v1/setup/owner`. O onboarding deve ser restrito ao administrador, usar convite/token de uso unico ou ficar explicitamente habilitado apenas em development.
2. Tornar a criacao de box + owner transacional para nunca deixar box orfao quando o usuario falhar.
3. Adicionar rate limit por IP/e-mail ao login, ao setup e aos endpoints de maior custo.
4. Adicionar limites de tamanho para body JSON, upload CSV/XLSX, quantidade de linhas importadas e campos de texto.
5. Implementar troca de senha pelo owner e um fluxo administrativo seguro de redefinicao. Avaliar recuperacao de senha por token quando houver e-mail produtivo.
6. Validar configuracao obrigatoria ao iniciar em `APP_ENV=production`: `JWT_SECRET` forte, credenciais administrativas, banco e demais segredos; a aplicacao deve falhar cedo se ainda usar defaults inseguros.
7. Revisar armazenamento do JWT no frontend, expiracao/logout e protecoes de browser (CSP e headers de seguranca) antes da exposicao publica.

Dados, segredos e migrations:

1. Substituir o loop SQL do `make migrate-up` por um migrator versionado, com tabela de controle e execucao unica de cada migration (ex.: `golang-migrate`, `goose` ou Atlas).
2. Criar um comando de migration apropriado para release, separado do start normal da API e dos comandos demo.
3. Implementar criptografia real para credenciais dedicadas de WhatsApp e senhas SMTP. Os campos atuais se chamam `*_encrypted`, mas hoje recebem o valor diretamente. A chave de criptografia deve ficar fora do PostgreSQL e permitir rotacao.
4. Garantir que segredos e PII nao aparecam em logs, auditorias, mensagens de erro ou dumps usados para suporte.
5. Configurar pool PostgreSQL, timeouts e verificacao de conectividade; preparar a aplicacao para conexoes limitadas do provedor.
6. Separar definitivamente comandos/dados demo dos fluxos de producao. `demo-reset-seed` nunca deve fazer parte de um deploy real.

Runtime e confiabilidade:

1. Trocar `router.Run` por `http.Server` com timeouts de leitura/escrita/idle, tratamento de `SIGTERM` e shutdown gracioso da API, telemetria e worker.
2. Adicionar endpoints separados de liveness e readiness; readiness deve verificar ao menos PostgreSQL e dependencias indispensaveis.
3. Ajustar a imagem final para usuario sem privilegios, certificados CA e `tzdata`.
4. Tornar automacoes seguras para mais de uma replica usando claim/lock transacional, idempotencia e protecao contra duas execucoes simultaneas da mesma rotina. Enquanto isso nao existir, documentar uma unica replica com worker.
5. Padronizar timeout, retry com backoff apenas quando seguro e idempotencia nas chamadas externas.
6. Implementar StatusCallback/webhook assinado da Twilio e atualizacao assincrona do status final. Hoje `sent` pode significar apenas que o provedor aceitou inicialmente uma mensagem que depois ficou `undelivered`.
7. Adicionar alertas operacionais para automacoes, importacoes e disparos com falha.

Qualidade e operacao:

1. Criar CI com `go test ./...`, verificacao das migrations, TypeScript e build Vite.
2. Adicionar testes de isolamento entre dois tenants, autorizacao OWNER/PLATFORM_ADMIN, onboarding, importacao/deduplicacao, limites de mensageria, automacoes e fluxos principais de campanha/brinde.
3. Adicionar pelo menos um smoke/E2E do login ate importacao, recalculo e consulta do dashboard.
4. Criar feature flags/configuracoes seguras para manter e-mail, automacao, LLM e envios reais desligados ate cada capacidade ser homologada.
5. Criar runbooks de erro comuns e deixar mensagens acionaveis na UI, sempre com `request_id` para suporte.

Privacidade/LGPD em nivel de aplicacao e processo:

1. [Processo/juridico] Formalizar a relacao controlador/operador, finalidades e bases legais; proposta tecnica registrada no runbook.
2. [Concluido na aplicacao] Exportacao, correcao via origem/reimportacao e exclusao por anonimizacao auditada.
3. [Concluido na aplicacao] Retencao definida e automatizavel por comando seguro; agendamento pertence a infraestrutura futura.
4. [Concluido na aplicacao] Opt-in/opt-out registrado e aplicado a todas as audiencias.
5. [Parcial] Procedimento de incidente documentado; politica publica, termos e contrato dependem de validacao juridica antes de dados reais.

### Direcao de observabilidade

Observabilidade passa a ser uma frente explicita de produto e operacao, nao uma tarefa deixada somente para a infraestrutura.

Arquitetura recomendada:

- Manter logs estruturados JSON via `slog` em `stdout`. Railway captura `stdout/stderr` e permite consulta imediata.
- Instrumentar a aplicacao Go com OpenTelemetry, usando OTLP e configuracao por ambiente para evitar dependencia de um fornecedor.
- Usar Railway para saude do container/maquina: CPU, memoria, disco, rede, deploys e logs de curta retencao.
- Usar Grafana Cloud no plano gratuito como primeira opcao para dashboards, alertas, metricas e traces de aplicacao. A aplicacao podera enviar OTLP diretamente no piloto ou passar por OpenTelemetry Collector/Grafana Alloy quando houver necessidade de processamento, batching ou multiplos destinos.
- Para desenvolvimento local, disponibilizar uma stack opcional via Docker Compose com Grafana + Prometheus + Loki + Tempo e Collector/Alloy, sem tornar essa stack requisito para rodar o EngageFit.
- Nao self-hostear toda a stack de observabilidade em producao inicialmente: isso adicionaria componentes, persistencia e manutencao justamente antes do primeiro piloto.

Telemetria minima da aplicacao:

- Logs HTTP: `request_id`, trace/span id, rota normalizada, metodo, status, latencia, ambiente, versao/release e replica.
- Logs de negocio: `box_id`, operacao, entidade, source type, resultado e erro normalizado, sem telefone, e-mail, token, corpo de mensagem ou credencial.
- Metricas HTTP: requests, erros, latencia e requests em andamento por rota normalizada.
- Metricas PostgreSQL: conexoes abertas/em uso/ociosas, espera por conexao e erros; evitar labels de alta cardinalidade.
- Metricas de gateways: chamadas, latencia, timeout e falha por provedor/operacao.
- Metricas de negocio/worker: imports concluídos/falhos, linhas processadas, automacoes executadas/falhas, mensagens autorizadas/bloqueadas/aceitas/falhas e geracoes LLM.
- Traces distribuidos da requisicao HTTP ate repository e chamadas externas, propagando contexto e `request_id`.
- Health endpoints: liveness, readiness e informacao de build sem expor segredo.

Dashboards/alertas iniciais:

1. Saude geral: disponibilidade, taxa de 5xx, p95/p99 de latencia e requests por minuto.
2. Runtime: CPU, memoria, disco, rede e reinicios da instancia.
3. PostgreSQL: disponibilidade, pool saturado, erros e queries lentas selecionadas.
4. Automacao: rotina atrasada, execucao com falha e ausencia de execucao esperada.
5. Mensageria: bloqueios por governanca, falhas do provider e divergencia entre aceito e entregue.
6. Alertas iniciais para API indisponivel, aumento de 5xx, readiness falha, automacao falha e consumo proximo do limite.

Cuidados de observabilidade:

- Nunca usar `student_id`, telefone, e-mail, `box_id` irrestrito, `request_id` ou URL crua como label de metrica de alta cardinalidade. IDs podem existir em logs/traces com acesso controlado, mas nao como dimensoes de metrica.
- Aplicar sampling em traces e reduzir logs de sucesso muito frequentes em producao.
- Definir retencao e controle de acesso tambem para a telemetria, pois logs podem virar uma nova fonte de vazamento de PII.
- Fazer flush da telemetria no shutdown gracioso antes de o Railway substituir a instancia.

Referencias de direcao:

- Railway Observability: `https://docs.railway.com/observability`
- Railway + ferramentas externas/OTLP: `https://docs.railway.com/guides/third-party-observability`
- OpenTelemetry: `https://opentelemetry.io/docs/`
- Grafana Cloud: `https://grafana.com/docs/grafana-cloud/introduction/`
- Grafana Loki: `https://grafana.com/docs/loki/latest/`

### Itens deliberadamente adiados para a fase de infraestrutura/deploy

- Escolha final de plano/regiao do Railway.
- Dominio, DNS, TLS e roteamento definitivo frontend/API.
- PostgreSQL gerenciado, politica concreta de backup/PITR e teste de restore no provedor.
- Pipeline de deploy, ambientes Railway e estrategia de rollback de release.
- Alertas de custo e limites do provedor.
- Configuracao produtiva do numero/canal WhatsApp.

Esses itens continuam obrigatorios antes do go-live, mas nao devem interromper a conclusao da prontidao no nivel da aplicacao.

## Feature concluída - governança de limites e custos do WhatsApp

Implementada em: 2026-07-16

- Documento de referência: `.ai/messaging-governance.md`.
- Migration `029_create_messaging_governance.sql` aplicada no PostgreSQL local.
- Políticas individuais por academia com limite diário, mensal, por disparo, orçamento diário/mensal, estimativa unitária, timezone e bloqueio administrativo.
- Política global adicional para todas as academias que usam o número compartilhado do EngageFit.
- Reserva transacional antes do envio impede estouro concorrente e bloqueia antes da chamada à Twilio.
- Campanhas manuais, automações e Treino do dia usam a mesma governança.
- `message_dispatches` registra envios autorizados/bloqueados; buckets diários/mensais registram consumo e reservas.
- Resposta do gateway agora preserva `provider_message_sid` e status inicial da Twilio/Meta por destinatário.
- Novo papel `PLATFORM_ADMIN`, sem `box_id`, isolado dos endpoints tenant.
- Administrador provisionado/rotacionado por `PLATFORM_ADMIN_NAME`, `PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD`.
- Área frontend `#admin-messaging` permite gerenciar política global e individual com auditoria e motivo obrigatório.
- Conta, remetente, credencial e modo da conexão dedicada agora são editados na área administrativa; owner possui somente leitura e teste da conexão salva.
- Owner visualiza sua franquia na tela WhatsApp, mas não pode alterar limites nem acessar rotas administrativas.
- Smoke test confirmou bloqueio antes do gateway, isolamento do admin e `403` para owner nas rotas administrativas; dados temporários removidos.
- Validações: `go test ./...`, TypeScript e build Vite passando.
- Próximas camadas: StatusCallback assinado da Twilio, conciliação de custo real, alertas, tela de dispatches/auditoria, secret manager e subaccounts para conexões dedicadas.

## Status operacional mais recente - WhatsApp/Twilio producao

Data: 2026-07-16

Objetivo atual do produto:

- Deixar o canal oficial de WhatsApp produtivado para o EngageFit.
- Manter um unico numero central do EngageFit como opcao padrao, enviando mensagens "em nome do box" no corpo da mensagem.
- Permitir que uma academia use sua propria conta Twilio, WABA, numero e templates quando precisar de uma conexao dedicada.
- Usar a conta verificada do primeiro cliente durante o piloto inicial de aproximadamente tres meses, evitando depender agora da verificacao empresarial do EngageFit.
- Tratar numero proprio por academia como capacidade core ja implementada e possivel diferencial de plano premium no futuro.

## Feature concluida - conexao WhatsApp compartilhada ou dedicada

Implementada em: 2026-07-16

- Backend, frontend, migration, documentacao e testes concluidos.
- API `GET/PUT /api/v1/whatsapp/settings` agora aceita e retorna `connection_mode` com valores `platform` ou `dedicated`.
- `platform` e o default para novas academias e usa exclusivamente as credenciais/remetente/Content SIDs definidos nas variaveis `WHATSAPP_PLATFORM_*` do backend.
- `dedicated` usa os dados persistidos em `whatsapp_settings` para o tenant autenticado.
- Campanhas oficiais, automacoes que enviam campanhas e mensagens do Treino do dia usam o resolver central de configuracao antes de chamar o gateway WhatsApp.
- No modo dedicado, a tela WhatsApp permite cadastrar o Content SID e o status de aprovacao de cada um dos tres templates oficiais na conta Twilio da academia.
- Migration `028` foi aplicada no PostgreSQL local. O registro existente foi preservado como `dedicated`, com credencial e remetente anteriores intactos.
- A alternancia `dedicated -> platform -> dedicated` foi validada pela API sem perder a credencial dedicada.
- Validacoes executadas com sucesso:
  - `go test ./...`
  - TypeScript (`tsc -b`)
  - build Vite
  - smoke test autenticado de configuracoes e previews dos tres templates.
- Estado local ao finalizar: `connection_mode=dedicated`; a conexao compartilhada esta indisponivel ate as variaveis `WHATSAPP_PLATFORM_*` serem preenchidas.
- Proximo passo do piloto: substituir na conexao dedicada as credenciais/remetente atuais pelos dados da conta Twilio verificada do primeiro cliente e cadastrar os tres Content SIDs aprovados nessa conta.

Decisao de arquitetura/comercial:

- Nao usar WhatsApp Web, Evolution API ou servicos nao oficiais para producao.
- Caminho recomendado e oficial: Twilio WhatsApp integrado a WABA/Meta.
- Conexao WhatsApp agora possui dois modos por academia:
  - `platform` (padrao para novas academias): usa a conta Twilio e o numero compartilhado do EngageFit, configurados somente por variaveis de ambiente do backend.
  - `dedicated`: usa a conta, credenciais, numero e Content SIDs proprios da academia; preparado para virar opcao premium.
- Migration `migrations/028_add_whatsapp_connection_mode.sql` adiciona `whatsapp_settings.connection_mode`. Configuracoes existentes e ativas migram para `dedicated` para preservar o comportamento; academias sem registro usam `platform` por padrao.
- A conexao compartilhada e configurada por `WHATSAPP_PLATFORM_ENABLED`, `WHATSAPP_PLATFORM_BASE_URL`, `WHATSAPP_PLATFORM_TWILIO_SENDER`, `WHATSAPP_PLATFORM_TWILIO_ACCOUNT_SID`, `WHATSAPP_PLATFORM_TWILIO_AUTH_TOKEN` e os tres `WHATSAPP_PLATFORM_TWILIO_CONTENT_SID_*`.
- O backend resolve a conexao efetiva antes dos envios de campanhas e Treino do dia. Credenciais compartilhadas nunca sao retornadas pela API nem duplicadas em `whatsapp_settings`.
- A tela `Configuracoes > Integracao WhatsApp` permite escolher `Numero do EngageFit` ou `Numero proprio da academia`, informa a disponibilidade da conexao compartilhada e so exibe credenciais no modo dedicado.
- Content SIDs pertencem a conta Twilio: no modo `platform`, os tres SIDs vem do ambiente central; ao usar `dedicated`, os templates precisam ser aprovados na conta da academia e seus SIDs/status podem ser cadastrados na tela WhatsApp daquele tenant.
- O produto deve controlar internamente limites por plano, volume por box, logs/auditoria, cooldown e elegibilidade de destinatarios.
- A integracao com Twilio deve ser tratada como infraestrutura central do EngageFit, nao como configuracao individual obrigatoria de cada box no MVP.

Estado Twilio/Meta:

- Conta Twilio saiu do trial e foi feito upgrade/pagamento.
- Sandbox Twilio ja havia sido validado com sucesso antes: mensagens enviadas e recebidas no WhatsApp do Luiz usando o sandbox.
- Sender oficial criado no Twilio:
  - numero: `+55 11 5217-0912`
  - formato backend: `+551152170912`
  - display name: `EngageFit`
  - WABA ID exibido na Twilio/Meta: `1361712461954280`
  - Meta Business/Portfolio ID: `1002668215888321`
- O sender oficial consegue receber mensagens inbound do celular do Luiz.
- O sender oficial nao consegue enviar mensagens outbound nem responder mensagens inbound.
- Erro confirmado na Twilio:
  - status final: `undelivered`
  - erro: `63051`
  - significado operacional: WABA/sender bloqueado/restrito pela Meta.
- Na Meta Business Support foi localizado o ativo correto:
  - `Contas do WhatsApp > EngageFit`
  - status: `Conta com restricao`
  - restricoes exibidas:
    - `You can not start conversations with customers`
    - `You can not respond to messages from customers`
    - `You can not have phone numbers added to it`
- A propria tela da Meta informa que, para comecar a enviar mensagens, o provedor de solucoes de negocios precisara verificar a empresa.
- Foi aberta a tela `Pedir analise` para contestar a restricao da WABA. Texto recomendado em ingles:

```text
This WhatsApp Business Account was recently created for the EngageFit product through the official Twilio integration.

We are still in the initial technical setup and pilot phase. We have not sent bulk messages, spam, abusive automation, or used the account in any improper way. The phone number +55 11 5217-0912 is already connected and can receive messages from users, but the account is currently blocked from starting conversations and replying to customer messages.

In Twilio, outbound messages fail with error 63051. I am requesting a review of the restriction on the EngageFit WhatsApp Business Account, WABA ID 1361712461954280, so we can complete the setup and test messaging in compliance with WhatsApp Business policies.
```

Leitura atual:

- Nao e problema de codigo, template ou Content SID.
- O backend/Twilio conseguem criar mensagens, mas a Meta derruba a entrega de forma assincrona por causa da restricao da WABA.
- Enquanto o erro `63051` continuar, nao adianta insistir em novos testes reais pelo EngageFit, pois a mensagem sera aceita inicialmente pela API da Twilio e depois ficara `undelivered`.
- Um piloto sem verificacao empresarial poderia existir apenas se a WABA estivesse em conformidade. Como a WABA esta restrita, o piloto real esta bloqueado ate a Meta/Twilio liberarem a conta.
- Nao preencher CPF como se fosse CNPJ/registro empresarial na verificacao da empresa. Isso pode gerar inconsistencia na verificacao.
- Para o produto real, a tendencia e precisar formalizar empresa e concluir verificacao empresarial. MEI provavelmente nao e a melhor opcao para SaaS/desenvolvimento de software; avaliar ME/SLU/Simples com contador.

Templates oficiais Twilio criados/aprovados:

- `engagefit_falta_pouco`
  - Content SID: `HX0a74da5635b2401c1b0ce1769aaea1ac`
  - idioma: `pt_BR`
  - tipo: texto
- `engagefit_meta_atingida`
  - Content SID: `HX63d54262966db42c1641c32ab64b11c9`
  - idioma: `pt_BR`
  - tipo: texto
- `engagefit_sentimos_sua_falta`
  - Content SID: `HX198c7dcaf71ae42a733719eee86d5aa5`
  - idioma: `pt_BR`
  - tipo: texto
- Os tres templates aparecem como elegiveis/aprovados para:
  - `WhatsApp business initiated`
  - `WhatsApp user initiated`

Configuracao local/backend ja alinhada:

- `whatsapp_settings` local aponta para provider `twilio`, enabled `true`, sender `+551152170912`.
- `.env` local relevante:
  - `WHATSAPP_ALLOW_REAL_SEND=true`
  - `WHATSAPP_DEV_RECIPIENT_PHONE=+5511963834712`
  - `AUTOMATION_WORKER_ENABLED=true`
  - `EMAIL_ALLOW_REAL_SEND=true`
- A configuracao local existente foi migrada como `connection_mode=dedicated`; a conexao compartilhada aparece indisponivel enquanto as novas variaveis `WHATSAPP_PLATFORM_*` nao forem preenchidas.
- Nao expor credenciais Twilio/OpenAI em handoff, logs ou respostas.
- `scripts/demo-seed.mjs` configura os tres templates oficiais via API como `APPROVED`.
- `Makefile demo-reset` foi ajustado para limpar tambem tabelas de e-mail e automacao, preservando configuracoes.
- `internal/app/messages/message_usecases.go` foi ajustado para a ordem correta de variaveis Twilio:
  - `1`: `student_name`
  - `2`: `box_name`
  - `3`: `current_checkins`
  - `4`: `missing_checkins`
  - `5`: `target_checkins`
  - `6`: `reward_name`
  - `7`: `platform_name`
- `internal/domain/whatsapp_template_catalog.go` foi alinhado aos textos aprovados na Twilio.
- Testes executados e passando:

```bash
cd engage-fit-be
node --check scripts/demo-seed.mjs
go test ./...
make demo-reset-seed
```

Cenario demo atual para testes controlados:

- Frontend: `http://localhost:5173/#/whatsapp`
- Backend: `http://localhost:8080`
- Login demo:
  - email: `owner@example.com`
  - senha: `change-me`
- Campanha demo: `Brinde do mes 07/2026`
- Destinatario de desenvolvimento: `+55 11 96383-4712`
- Audiencias do seed:
  - `Meta atingida`: 1 mensagem
  - `Aluno em risco`: 1 mensagem
  - `Falta pouco`: 2 mensagens
- Enquanto a WABA estiver restrita, os envios reais devem falhar com `63051` mesmo que a API do backend retorne sucesso inicial.

Proximos passos obrigatorios:

1. Aguardar/acompanhar a analise da Meta para a WABA `1361712461954280`.
2. Abrir ticket na Twilio se a analise da Meta nao resolver:

```text
My WhatsApp Business Account is restricted in Meta Business Support.

WABA ID: 1361712461954280
Sender: +55 11 5217-0912
Twilio error: 63051
Meta restrictions shown:
- You can not start conversations with customers
- You can not respond to messages from customers
- You can not have phone numbers added to it

Meta says that my business solution provider needs to verify my business before I can start sending messages. Could you please verify what is required on Twilio side and help unlock or verify this WABA?
```

3. Se a Meta/Twilio exigirem verificacao empresarial, formalizar a empresa antes de insistir na WABA atual.
4. Depois que a WABA for destravada, repetir um teste real minimo:
   - inbound do celular para o sender oficial;
   - resposta livre dentro da janela de 24h;
   - envio de um template aprovado pelo EngageFit;
   - verificar status final na Twilio como `delivered` ou equivalente.

Pendencias tecnicas apos desbloqueio:

- Implementar webhook/status callback da Twilio para atualizar `message_recipients` de forma assincrona.
- Hoje o sistema pode marcar `sent` quando a Twilio aceita a criacao da mensagem, mesmo se a Meta depois retornar `undelivered`.
- A UI deve expor falhas posteriores de entrega com erro Twilio, especialmente `63051`, `undelivered` e rejeicoes de template/sender.
- Desenhar limites por plano somente depois do canal estar estavel:
  - limite mensal por box;
  - limite diario por box;
  - cooldown por aluno/audiencia;
  - bloqueio de disparo duplicado por campanha;
  - auditoria por destinatario;
  - possivel plano premium para numero dedicado por box.

## Papel do projeto

EngageFit e um sistema multi-tenant para boxes de CrossFit acompanharem check-ins de alunos vindos de Wellhub e TotalPass, calcularem metas mensais por plataforma, identificarem alunos proximos/aptos ao brinde e dispararem mensagens personalizadas. O foco inicial segue em boxes de CrossFit, mas a marca e o posicionamento deixam espaco para academias no geral.

## Stack aprovada

Backend:

- Go
- Gin
- GORM
- PostgreSQL
- Arquitetura Hexagonal

Frontend:

- React
- Vite
- TypeScript
- Tailwind
- componentes locais no padrao shadcn/ui
- Lucide Icons

## Decisoes principais

- Sistema multi-tenant desde o inicio.
- Entidades centrais: `Box`, `User`, `Student`, `Checkin`, `Campaign`, `CampaignGoal`, `Reward`, `CampaignProgress`, `RewardDelivery`, `WhatsappSettings`, `MessageTemplate`, `MessageCampaign`, `MessageRecipient`, `EmailSettings`, `EmailTemplate`, `EmailCampaign`, `EmailRecipient`, `AutomationRun`, `AutomationSchedule`, `Workout`, `WorkoutMessageDraft`, `WorkoutMessageRecipient`, `LLMGenerationLog`.
- MVP tem apenas perfil `OWNER`.
- Sem receita financeira no MVP.
- `Checkin` nao possui `revenue`.
- Aluno proximo da meta: atingiu pelo menos 80% da meta da plataforma.
- Aluno em risco: quantidade configuravel de dias sem check-in por box, default `7`.
- Dashboard e funcionalidade principal.
- Card `Brindes pendentes` no dashboard e apenas resumo operacional; a baixa de entrega fica na tela dedicada `Brindes`.
- Controle de brindes no MVP e baseado em `Reward.quantity` + `RewardDelivery`:
  - pendente quando aluno bate meta e ainda nao recebeu.
  - entregue quando usuario marca manualmente.
  - disponivel calculado como `quantity - delivered_deliveries`.
  - pendencias nao descontam estoque real ate serem entregues.
- Widgets do dashboard (`Campanhas ativas`, `Próximos da meta`, `Alunos em risco`, `Brindes pendentes`) usam paginacao client-side de 5 itens por pagina.
- Alunos em risco agora possuem acompanhamento:
  - `risk_status`: `active`, `observing`, `paused`, `not_interested`.
  - `risk_last_message_at`: ultima mensagem de risco enviada.
  - migration: `migrations/020_add_student_risk_tracking.sql`.
  - endpoint `PATCH /api/v1/students/:id/risk-status` permite mudar o status manualmente.
  - campanha de mensagem `inactive` nao envia para `paused`/`not_interested` e respeita cooldown configuravel apos a ultima mensagem de risco.
  - quando uma mensagem `inactive` e enviada com sucesso, o aluno passa para `observing`.
- Regras de risco configuraveis por box:
  - `boxes.risk_inactive_days`, default `7`.
  - `boxes.risk_message_cooldown_days`, default `14`.
  - migration: `migrations/021_add_box_risk_settings.sql`.
  - tela `Configuracoes > Regras de risco` permite alterar os dois valores.
- WhatsApp faz parte do MVP.
- Relatorios essenciais do MVP implementados:
  - elegiveis.
  - brindes pendentes.
  - frequencia mensal.
  - todos com visualizacao na UI, filtros client-side e exportacao CSV.
- Navegacao frontend usa hash (`#dashboard`, `#campaigns`, `#rewards`, etc.) para preservar a tela apos refresh. Logo da sidebar navega para `Dashboard`.
- E-mail personalizado, automacao diaria com agendamento/auditoria persistida e `Treino do dia` com rascunhos gerados por LLM ja foram implementados. Próximos focos devem partir das pendencias funcionais abaixo, nao da mensagem antiga de handoff.

## Estado atual implementado

### Backend

Diretorio: `engage-fit-be`

Implementado:

- Auth com JWT/bcrypt.
- Bootstrap owner via `POST /api/v1/setup/owner`.
- Multi-tenant por `box_id`.
- CRUD base de campanhas.
- Metas por plataforma em `campaign_goals`.
- Brindes em `rewards`.
- Importacao de check-ins CSV/XLSX.
- Parser ajustado para planilhas reais:
  - Wellhub com preambulo, `Data`, `Hora`, `Visitante`, `ID do Wellhub`.
  - TotalPass tokens com preambulo, `ID`, `Colaborador`, `Validado em`.
- Deduplicacao de check-ins:
  - chave: `box_id + source + student_id + checkin_date + checkin_time`
  - migration: `migrations/016_add_unique_checkins.sql`
  - `SaveMany` usa `ON CONFLICT DO NOTHING`
- Recalculo de progresso de campanha.
- WhatsApp settings.
- Templates de WhatsApp com variaveis:
  - `{{name}}`
  - `{{nome}}`
  - `{{email}}`
  - `{{phone}}`
  - `{{telefone}}`
  - `{{source}}`
  - `{{platform}}`
  - `{{plataforma}}`
  - `{{box_name}}`
  - `{{campaign_name}}`
  - `{{reward_name}}`
  - `{{current_checkins}}`
  - `{{checkins}}`
  - `{{target_checkins}}`
  - `{{goal_checkins}}`
  - `{{remaining_checkins}}`
  - `{{faltam_checkins}}`
- WhatsApp mock com `mock://local`.
- WhatsApp com provider configuravel:
  - `twilio` para Twilio WhatsApp, caminho comercial recomendado.
  - `meta_cloud` para Meta Cloud API oficial.
  - migration: `migrations/017_add_whatsapp_provider.sql`
  - `instance_name` passa a representar `Phone number ID` quando provider for `meta_cloud`.
  - `instance_name` passa a representar `whatsapp:+...` ou `Messaging Service SID` quando provider for `twilio`.
  - `base_url` pode ficar vazio para Meta Cloud API; o backend usa `https://graph.facebook.com/v20.0`.
  - `base_url` pode ficar vazio para Twilio; o backend usa `https://api.twilio.com`.
- Templates de WhatsApp agora aceitam `content_sid`:
  - migration: `migrations/018_add_message_template_content_sid.sql`
  - usado pelo provider `twilio` como `ContentSid`.
  - variaveis Twilio enviadas como `ContentVariables`: `1=name`, `2=box_name`, `3=current_checkins`, `4=remaining_checkins`, `5=target_checkins`, `6=reward_name`, `7=platform`.
  - bug corrigido em `MessageTemplateModel`: `ContentSID` precisa de tag `gorm:"column:content_sid"`; sem isso o GORM tentava inserir em `content_s_id`.
- `SafeGateway`:
  - em `development`, envio real fica bloqueado por padrao.
  - para envio real local precisa:
    - `WHATSAPP_ALLOW_REAL_SEND=true`
    - `WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES=5511963834712,5518997980429` para permitir envio apenas para uma lista fechada.
    - ou `WHATSAPP_DEV_RECIPIENT_PHONE=55DDDNUMERO` para redirecionar tudo para um unico telefone.
  - quando `WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES` estiver preenchido, em development o backend envia para os proprios numeros permitidos e bloqueia qualquer outro destinatario.
- Teste real Twilio Sandbox validado:
  - Sandbox ativado no console da Twilio.
  - Numero de teste fez join enviando `join science-everyone` para `+1 415 523 8886`.
  - Remetente configurado no EngageFit como sandbox Twilio (`+14155238886` ou `whatsapp:+14155238886`).
  - Mensagens foram enviadas e recebidas no WhatsApp com sucesso para o numero do Luiz.
  - Erros anteriores resolvidos:
    - `63007 Twilio could not find a Channel with the specified From address`: sandbox ainda nao estava ativado/configurado.
    - `63031 Message cannot have the same To and From`: remetente estava configurado igual ao destinatario em tentativa anterior.
- Acompanhamento de alunos em risco:
  - `students` tem `risk_status` e `risk_last_message_at`.
  - `boxes` tem `risk_inactive_days` e `risk_message_cooldown_days`.
  - `PATCH /api/v1/students/:id/risk-status` atualiza status manual.
  - `Dashboard` usa `risk_inactive_days` para classificar aluno em risco.
  - `SendMessageCampaignUseCase` aplica `risk_message_cooldown_days` para audiência `inactive`.
  - envio `inactive` bem-sucedido marca aluno como `observing` e grava `risk_last_message_at`.
- Controle de brindes:
  - `GET /api/v1/rewards/deliveries` lista entregas pendentes e entregues com campanha, aluno, telefone, brinde e status.
  - `GET /api/v1/rewards/pending-deliveries` lista apenas pendencias.
  - `PATCH /api/v1/reward-deliveries/:id/deliver` marca entrega como entregue com filtro por `box_id`.
  - `GET /api/v1/campaigns/:id/rewards` retorna contadores por brinde: `pending_deliveries`, `delivered_deliveries`, `available_quantity`.
- Relatorios:
  - `GET /api/v1/reports/eligible-students`
  - `GET /api/v1/reports/pending-rewards`
  - `GET /api/v1/reports/monthly-frequency?month=YYYY-MM`
  - os tres endpoints aceitam `?format=csv`.
  - `eligible-students` usa `campaign_progresses` com joins em `campaigns`, `students` e `rewards`.
  - `pending-rewards` usa `reward_deliveries` pendentes enriquecidas com campanha/aluno/brinde.
  - `monthly-frequency` agrupa check-ins por aluno no periodo mensal.
- Campanhas de mensagem vinculadas a campanha de meta:
  - `message_campaigns.campaign_id` referencia `campaigns.id`.
  - migration: `migrations/022_add_message_campaign_campaign_id.sql`.
  - audiências `almost_there`, `near_goal` e `achieved` usam a campanha vinculada, nao todas as campanhas ativas.
  - template context (variaveis `campaign_name`, `reward_name`, check-ins) usa a campanha vinculada.
- Preview de mensagem antes do envio:
  - `GET /api/v1/message-campaigns/:id/preview` retorna `body`, `total`, aluno exemplo e telefone.
  - usa a mesma renderizacao do envio real (`renderTemplate` + `templateValues`).
- Evolution API removida do produto:
  - providers suportados: `twilio` (padrao) e `meta_cloud`.
  - migration: `migrations/023_remove_evolution_provider.sql` normaliza configs antigas `evolution` para `twilio`.
  - removidos `evolution_client.go`, `docker-compose.evolution.yml` e targets `make evolution-*`.
- Automacao diaria operacional:
  - script `scripts/daily-automation.mjs` e target `make daily-automation`.
  - importa arquivo opcional (`DAILY_CHECKINS_FILE`), recalcula campanhas ativas, envia campanhas de mensagem vinculadas (`DAILY_SEND_MESSAGES=true`).
- E-mail personalizado:
  - provider `smtp` com modo seguro em development e provider `mock` para testes locais.
  - settings em `GET/PUT/POST /api/v1/email/settings` e `/test`.
  - templates em `/api/v1/email-templates` com assunto, corpo e variaveis iguais as campanhas WhatsApp.
  - campanhas em `/api/v1/email-campaigns` com `campaign_id`, audiência, preview, envio manual e auditoria por destinatario.
  - envio real local fica bloqueado por padrao; usar `EMAIL_ALLOW_REAL_SEND=true` ou provider `mock`.
- Automacao diaria como feature do produto:
  - tabelas `automation_runs` e `automation_schedules`.
  - endpoints de historico: `GET/POST /api/v1/automation/runs`, `GET/PATCH /api/v1/automation/runs/:id`.
  - endpoints de rotinas: `GET/POST /api/v1/automation/schedules`, `PUT/DELETE /api/v1/automation/schedules/:id`, `POST /api/v1/automation/schedules/:id/run`.
  - tela `Automacao` permite criar rotina com horario, dias da semana, modo, reenvio e status ativo/pausado.
  - modos: `full_daily`, `recalculate_only`, `send_almost_there`, `send_achieved`, `send_inactive`.
  - worker interno roda agendas quando `AUTOMATION_WORKER_ENABLED=true`; intervalo configuravel por `AUTOMATION_WORKER_INTERVAL_SECONDS`.
  - `scripts/daily-automation.mjs` permanece como alternativa operacional/CI e tambem registra `automation_runs`.
- Treino do dia com mensagens geradas por LLM:
  - migration: `migrations/026_create_workouts.sql`.
  - entidades/tabelas: `workouts`, `workout_message_drafts`, `workout_message_recipients`, `llm_generation_logs`.
  - adapter OpenAI via `OPENAI_API_KEY`, `OPENAI_MODEL` e `OPENAI_TIMEOUT_SECONDS`; default de modelo: `gpt-4.1-mini`.
  - prompt com guardrails: mensagem curta, pratica, segura, sem dieta individual, sem orientacao medica, sem promessa de resultado e recomendando falar com o coach em caso de dor/duvida/adaptacao.
  - endpoints: `GET/POST /api/v1/workouts`, `GET/PUT/DELETE /api/v1/workouts/:id`, `GET/POST /api/v1/workouts/:id/message-drafts`, `PUT /api/v1/workout-message-drafts/:id`, `POST /api/v1/workout-message-drafts/:id/approve`, `POST /api/v1/workout-message-drafts/:id/send`, `GET /api/v1/workout-message-drafts/:id/recipients`.
  - envio WhatsApp exige rascunho aprovado manualmente (`approved`) e usa texto livre aprovado pelo owner; restricoes comerciais do WhatsApp/Twilio aparecem como falhas auditadas por destinatario.
  - audiencias reutilizadas: `all`, `inactive`, `almost_there`, `near_goal`, `achieved`; as audiencias de progresso exigem `campaign_id`.
  - `make demo-reset` limpa as tabelas novas e `scripts/demo-seed.mjs` cria um WOD demo sem chamar OpenAI.

Validacao atual:

```bash
cd engage-fit-be
go test ./...
```

Passando.

### Frontend

Diretorio: `engage-fit-fe`

Implementado:

- Login.
- Layout com sidebar/header.
- Marca atual: `EngageFit`.
  - Logo gerada e integrada em `frontend/public/engagefit-logo.png`.
  - Versao recortada sem margens em `frontend/public/engagefit-logo-cropped.png`.
  - Sidebar, login e favicon usam a versao recortada.
- Dashboard inicial.
  - Widgets com paginacao client-side de 5 itens.
  - `Brindes pendentes` mostra nome do aluno, nome do brinde e telefone como resumo; baixa operacional fica em `Brindes`.
  - `Alunos em risco` mostra ultima mensagem de risco, status de acompanhamento e permite alterar status manualmente.
- Campanhas com fluxo operacional:
  - criar campanha
  - meta Wellhub
  - meta TotalPass
  - brinde
  - indicadores de brinde por campanha: total, disponiveis, pendentes e entregues.
  - painel com progresso, faltantes, proximos e atingidos
  - botao recalcular
  - editar campanha (nome, descricao, datas)
  - editar metas Wellhub/TotalPass
  - editar brinde (nome, descricao, quantidade)
  - encerrar e reativar campanha
- Brindes:
  - tela dedicada no menu lateral.
  - busca por campanha, aluno, telefone e brinde.
  - filtro por status: pendentes, todas, entregues.
  - baixa manual com botao `Marcar entregue`.
  - historico de entregas permanece visivel em `Todas`/`Entregues`.
- Alunos.
- Importacoes.
- Relatorios:
  - tela dedicada no menu lateral.
  - relatorio de elegiveis com filtros por busca, campanha e plataforma.
  - relatorio de brindes pendentes com filtros por busca e campanha.
  - relatorio de frequencia mensal com filtro de mes, busca e plataforma.
  - CSV exporta o recorte filtrado na tela.
- Navegacao:
  - hash da URL preserva pagina atual apos refresh.
  - logo da sidebar volta para o dashboard.
  - icones diferenciados: `Campanhas` usa `Target`, `Brindes` usa `Gift`.
- WhatsApp:
  - templates
  - campanhas de mensagem vinculadas a uma campanha de meta (`campaign_id`)
  - variaveis de template
  - preview renderizado da mensagem antes do envio (aluno exemplo + total de destinatarios)
  - botao enviar/reenviar campanha
  - retorno visual de `sent/total/failed` apos envio
  - auditoria do ultimo envio por destinatario, incluindo `error_message` da Twilio
- E-mail:
  - tela dedicada no menu lateral.
  - configuracao SMTP/mock com teste de credenciais.
  - templates com assunto/corpo e variaveis operacionais.
  - campanhas vinculadas a campanha de meta, preview renderizado, envio/reenviar e auditoria do ultimo envio.
- Automacao:
  - tela dedicada no menu lateral.
  - cria/pausa/remove rotinas automaticas com horario, dias, modo e permissao de reenvio.
  - botao `Executar` permite rodar uma rotina manualmente.
  - historico de execucoes diarias com status, importacao, campanhas recalculadas, mensagens enviadas, falhas e erros.
- Treino do dia:
  - tela dedicada no menu lateral.
  - cadastro de WOD/treino com data, titulo, objetivo, movimentos principais e observacoes do coach.
  - geracao de rascunho por IA para audiencias existentes.
  - edicao do texto, aprovacao manual obrigatoria e envio pelo WhatsApp.
  - auditoria do ultimo envio por destinatario, incluindo falhas do provider.
- Dashboard operacional:
  - atalhos para Brindes, Relatorios, WhatsApp e Automacao.
- Configuracoes:
  - Card `Regras de risco`:
    - Aluno em risco apos X dias sem check-in.
    - Reenviar mensagem de risco apos X dias.
  - formulario de provedor WhatsApp
  - Twilio WhatsApp como opcao comercial recomendada
  - Meta Cloud API mantida como opcao avancada/futura
  - Base URL
  - Phone number ID / Instancia
  - Access token / API key
  - Ativar WhatsApp
  - Testar conexao
  - Mostra se ha credencial salva e quando foi a ultima atualizacao.
  - Campo de credencial fica vazio por seguranca, mas salvar com ele vazio preserva a credencial ja cadastrada.
  - Botao de teste usa os dados atuais do formulario quando ha alteracoes nao salvas.

Validacao atual:

```bash
cd engage-fit-fe
npm ci
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Passando.

Observacao: `engage-fit-fe/.npmrc` aponta ao registry publico (`registry.npmjs.org`). O `package-lock.json` foi corrigido para nao usar o registry privado Fury Cloud.

## Comandos principais

Subir banco do EngageFit:

```bash
make up
make migrate-up
```

Rodar backend:

```bash
cd engage-fit-be
make backend-run
```

`make backend-run` exporta `DATABASE_URL` automaticamente. Alternativa manual:

```bash
cp .env.example .env
go run ./cmd/api
```

Rodar frontend:

```bash
cd engage-fit-fe
npm install
npm run dev
```

Frontend local:

```txt
http://localhost:5173
```

Credenciais demo:

```txt
owner@example.com
change-me
```

Seed demo:

```bash
make demo-seed
```

Reset + seed demo:

```bash
make demo-reset-seed
```

O seed demo atual foi ajustado para teste controlado de WhatsApp:

- Cria campanha ativa com meta TotalPass `10`.
- Cria alunos/cenarios usando o telefone do Luiz (`5511963834712`):
  - Luiz: `9/10`, entra em `almost_there` (`Falta pouco`).
  - Deborah: `8/10`, entra em `almost_there` se ainda houver pelo menos 2 dias restantes na campanha.
  - Bruno Teste: `7/10`, fica fora de `almost_there` por estar abaixo de 80%.
  - Carla Teste: `10/10`, entra em `achieved` (`Meta atingida`).
  - Marina Risco: `3/10`, entra em `inactive` (`Aluno em risco`) por estar ha mais de 7 dias sem check-in.
- Cria templates e campanhas de mensagem vinculadas a campanha do mes para:
  - `almost_there` (`Disparo teste - falta pouco`)
  - `achieved` (`Disparo teste - meta atingida`)
  - `inactive` (`Disparo teste - aluno em risco`)
- Audience `almost_there` foi adicionada a constraint de `message_campaigns` pela migration `migrations/019_add_almost_there_message_audience.sql`.
- Nao configura WhatsApp mock e nao envia automaticamente, para nao sobrescrever configuracao Twilio real.
- `make demo-reset` preserva `boxes`, `users` e `whatsapp_settings`, entao a configuracao Twilio cadastrada pela UI nao precisa ser refeita a cada `make demo-reset-seed`.
- Validado em 2026-06-26: apos aplicar a migration `019`, `make demo-reset-seed` passou e o envio da campanha `Disparo teste - falta pouco` funcionou.
- Validado em 2026-06-26: migration `020` aplicada no banco local; `go test ./...`, `node --check scripts/demo-seed.mjs` e build frontend direto passaram apos acompanhamento de risco.
- Validado em 2026-06-26: migration `021` aplicada no banco local; `go test ./...` e build frontend direto passaram apos regras de risco configuraveis por box.
- Validado em 2026-06-26: controle de brindes, tela dedicada de brindes, relatorios essenciais, navegacao por hash e build frontend passaram com:
  - `cd backend && go test ./...`
  - `cd frontend && node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build`
  - smoke test autenticado dos endpoints de relatorio retornou `200`.

Planilha/CSV incremental para bater a meta:

- `test-data/totalpass-checkins-hit-goal.csv`
- `test-data/totalpass-checkins-23-06-2026.xlsx`

Ambos contem:

- Luiz recebe +1 check-in.
- Deborah recebe +2 check-ins.
- Bruno Teste recebe +3 check-ins.
- Apos importar um destes arquivos, Luiz, Deborah e Bruno ficam com `10/10`.

Fluxo de teste:

1. `make demo-reset-seed`
2. Enviar a campanha `Disparo teste - falta pouco` para validar a audiência dinamica.
3. Enviar a campanha `Disparo teste - aluno em risco` para validar `inactive`; Marina Risco deve virar `observing` e nao receber novo envio ate passar o cooldown configurado em `Configuracoes > Regras de risco`.
4. Importar `test-data/totalpass-checkins-hit-goal.csv` ou `test-data/totalpass-checkins-23-06-2026.xlsx` na tela de importacoes com fonte `TotalPass`.
5. Luiz, Deborah e Bruno ficam com 10/10 check-ins.
6. Configurar Twilio em `Configuracoes`, se ainda nao estiver configurado.
   - Para sandbox, usar remetente `+14155238886` ou `whatsapp:+14155238886`.
   - O WhatsApp do Luiz precisa entrar no sandbox enviando `join science-everyone` para `+1 415 523 8886` antes de receber mensagens.
7. Enviar ou reenviar a campanha `Disparo teste - meta atingida` na tela `WhatsApp`.

## WhatsApp comercial

Direcao atual: Twilio WhatsApp e o caminho principal para o MVP comercial.

Implementado nesta etapa:

- Provider `twilio` no backend.
- Cliente `TwilioClient` em `backend/internal/adapters/whatsapp/twilio_client.go`.
- `Content SID` em templates de mensagem.
- Provider `meta_cloud` tambem existe como opcao avancada/futura.
- Cliente `MetaCloudClient` em `backend/internal/adapters/whatsapp/meta_cloud_client.go`.
- Gateway roteador por provider em `backend/internal/adapters/whatsapp/provider_gateway.go`.
- Frontend de configuracoes permite escolher `Twilio WhatsApp` ou `Meta Cloud API`.
- Frontend de WhatsApp permite enviar/reenviar campanhas de mensagem.
- README documenta configuracao comercial.

Configuracao esperada:

```txt
Provedor: Twilio WhatsApp
Base URL: https://api.twilio.com
Remetente WhatsApp ou Messaging Service SID: whatsapp:+14155238886 ou MG...
Account SID:Auth Token: AC...:<auth-token>
Ativar WhatsApp: marcado
```

Para disparo comercial em massa:

- Regra do WhatsApp/Twilio: mensagem livre so e confiavel dentro da janela de conversa de 24h apos o destinatario responder/iniciar conversa. Fora disso, usar template aprovado com `Content SID` (`HX...`).
- Sandbox: manter comportamento de teste atual, sem bloquear envio quando `content_sid` estiver vazio; o backend tenta enviar texto livre e registra o erro da Twilio se ela recusar (ex.: `63016`).
- Producao: validar com cliente antes de implementar. Direcao proposta: cada cliente cria modelos de mensagem usando apenas variaveis liberadas pelo sistema (`{{nome_aluno}}`, `{{nome_academia}}`, campanha, brinde etc.). O sistema converte isso para um template Twilio/WhatsApp e envia para aprovacao.
- Aprovacao pode ser operacional via Content Template Builder no console da Twilio no inicio, ou automatizada depois via Twilio Content API. A API cria o content template, dispara/acompanha aprovacao para WhatsApp e retorna/expoe o `Content SID` aprovado.
- Depois de aprovado, salvar `content_sid`, status de aprovacao, idioma e mapeamento de variaveis por tenant/template.
- Para envio comercial fora da janela de 24h, enviar via Twilio usando `ContentSid` e `ContentVariables`, nao `Body` livre.
- Remetente por cliente: possivel, mas cada tenant precisa ter seu proprio WhatsApp Sender/numero aprovado na Twilio/WABA ou uma configuracao equivalente. O backend deve escolher o remetente e credenciais pelo tenant.
- Garantir que as variaveis do template Twilio estejam na ordem esperada pelo provider.
- Decisao adiada: validar com cliente se aceitara fluxo de templates aprovados antes de implementar onboarding/aprovacao em producao.

Validacao apos mudanca:

```bash
cd backend
go test ./...

cd frontend
node node_modules/typescript/lib/tsc.js -b && node node_modules/vite/bin/vite.js build
```

Ambos passaram.

Automacao diaria:

```bash
cd engage-fit-be
DAILY_CHECKINS_SOURCE=totalpass \
DAILY_CHECKINS_FILE=test-data/totalpass-checkins-hit-goal.csv \
DAILY_SEND_MESSAGES=true \
make daily-automation
```

## Treino do dia com mensagens geradas por LLM

Implementado em 2026-07-02 como MVP operacional com aprovacao manual antes do envio.

Fluxo atual:

- Owner cadastra o treino/WOD do dia.
- Owner escolhe audiencia (`all`, `inactive`, `almost_there`, `near_goal`, `achieved`).
- Backend gera rascunho com OpenAI usando guardrails de seguranca.
- Owner revisa/edita o texto e aprova manualmente.
- Sistema envia pelo WhatsApp usando texto livre aprovado e registra auditoria por destinatario.

Restricao importante de WhatsApp:

- O MVP envia texto livre aprovado pelo owner. Fora da janela de conversa de 24h, Twilio/Meta podem bloquear a mensagem; o erro fica registrado em `workout_message_recipients.error_message`.
- Proximo passo comercial recomendado: evoluir para templates aprovados/opt-in antes de usar em producao em massa.

Futuras evolucoes:

- Automacao/agendamento de geracao/envio de rascunhos do treino.
- Configuracoes por box para tom, tamanho maximo, assinatura, horarios permitidos e exigencia de aprovacao.
- Templates aprovados por categoria de treino ou fluxo de opt-in para respeitar WhatsApp fora da janela de 24h.
- Audiencia de alunos frequentes, caso a operacao valide a regra de frequencia.

## Pendencias funcionais

Prioridade alta:

1. Relatorios avancados e filtros server-side quando o volume crescer.
2. Operacionalizar automacao em ambiente real:
   - decidir se o worker interno (`AUTOMATION_WORKER_ENABLED=true`) sera o caminho principal em producao ou se havera cron/CI externo como fallback.
   - definir fonte automatica dos check-ins do dia anterior.
   - configurar observabilidade/alertas para `automation_runs` com falha.
3. Avaliar o sistema inteiro e padronizar logs/telemetria pensando em observabilidade futura:
   - definir convencao unica de logs estruturados por camada (HTTP, use cases, gateways externos, jobs/worker, scripts operacionais).
   - garantir `request_id`/correlation id atravessando handlers, casos de uso e chamadas externas.
   - padronizar campos: tenant/box_id, usuario quando aplicavel, entidade, operacao, status, latencia, erro bruto e erro normalizado.
   - definir niveis (`debug`, `info`, `warn`, `error`) e evitar vazamento de segredos/PII sensivel.
   - preparar caminho para coletor futuro (ex.: OpenTelemetry, Loki, Datadog ou similar), metricas e alertas.
4. Lidar melhor com multiplas campanhas ativas simultaneas na operacao apos testes com clientes reais.

Prioridade media:

1. Renomear a entidade `Box` no dominio/UI para um conceito mais generico antes de expandir para academias convencionais. O sistema nasceu focado em boxes de CrossFit, mas o posicionamento atual e fitness/academias em geral; avaliar nomes como `Academy`, `Gym`, `Business`, `Unit` ou `Organization`. A migracao deve preservar `box_id` ou planejar renomeacao gradual de banco/API/frontend sem quebrar dados existentes.
2. Testes automatizados para use cases de relatorio, controle de brindes, mensagens e e-mail.
3. Relatorios avancados:
   - filtros server-side/paginacao se volume crescer.
   - historico de brindes entregues por periodo.
   - relatorio de conversao de mensagens por campanha.
4. UX mobile/responsiva para tabelas grandes.
5. Testes de integracao de repositories contra PostgreSQL real.
6. Refinar Dashboard com atalhos conforme feedback de uso.
7. Melhorar seed/demo com mais cenarios de conversao de mensagens e e-mail.
8. Evoluir `Treino do dia` para fluxo comercial com templates aprovados/opt-in, agendamento e configuracoes por box.

## Arquivos importantes

Docs:

- `docs/system-design.md` (fonte canonica de arquitetura, negocio e manutencao)
- `docs/application-readiness-guide.md` (guia operacional da aplicacao)
- `docs/railway-deployment-checklist.md` (proxima fase de infraestrutura)
- `.ai/product-vision.md`
- `.ai/domain.md`
- `.ai/database.md`
- `.ai/architecture.md`
- `.ai/features.md`
- `.ai/tasks.md`
- `.ai/decisions.md`
- `.ai/ui-ux.md`
- `.ai/implementation-plan.md`
- `.ai/handoff.md`

Backend:

- `engage-fit-be/cmd/api/main.go`
- `engage-fit-be/internal/app/imports/import_checkins_usecase.go`
- `engage-fit-be/internal/app/messages/message_usecases.go`
- `engage-fit-be/internal/app/email/email_usecases.go`
- `engage-fit-be/internal/app/automation/automation_usecases.go`
- `engage-fit-be/internal/app/automation/worker.go`
- `engage-fit-be/internal/app/workouts/workout_usecases.go`
- `engage-fit-be/internal/adapters/whatsapp/twilio_client.go`
- `engage-fit-be/internal/adapters/whatsapp/provider_gateway.go`
- `engage-fit-be/internal/adapters/whatsapp/safe_gateway.go`
- `engage-fit-be/internal/adapters/email/smtp_gateway.go`
- `engage-fit-be/internal/adapters/llm/openai_generator.go`
- `engage-fit-be/migrations/022_add_message_campaign_campaign_id.sql`
- `engage-fit-be/migrations/023_remove_evolution_provider.sql`
- `engage-fit-be/migrations/024_create_email_and_automation.sql`
- `engage-fit-be/migrations/025_create_automation_schedules.sql`
- `engage-fit-be/migrations/026_create_workouts.sql`

Frontend:

- `engage-fit-fe/src/pages/campaigns/CampaignsPage.tsx`
- `engage-fit-fe/src/pages/whatsapp/WhatsappPage.tsx`
- `engage-fit-fe/src/pages/email/EmailPage.tsx`
- `engage-fit-fe/src/pages/automation/AutomationPage.tsx`
- `engage-fit-fe/src/pages/workouts/WorkoutsPage.tsx`
- `engage-fit-fe/src/pages/settings/SettingsPage.tsx`
- `engage-fit-fe/src/features/api/endpoints.ts`
- `engage-fit-fe/.npmrc`

Infra/dev:

- `engage-fit-be/Makefile` (`docker compose`, `make backend-run` com `DATABASE_URL`, `make daily-automation`)
- `engage-fit-be/docker-compose.yml`
- `engage-fit-be/scripts/demo-seed.mjs`
- `engage-fit-be/scripts/daily-automation.mjs`
- `engage-fit-be/migrations/024_create_email_and_automation.sql`
- `engage-fit-be/migrations/025_create_automation_schedules.sql`
- `engage-fit-be/migrations/026_create_workouts.sql`
- `engage-fit-be/test-data/totalpass-checkins-hit-goal.csv`

## Orientacao para iniciar novo chat

Mensagem sugerida:

```txt
Leia `.ai/handoff.md` e use `docs/system-design.md` como fonte canonica da arquitetura e das regras de negocio. Backend, frontend e PostgreSQL ja estao implantados no Railway production. Codigo homologado: backend `8b5443b` e frontend `81cee85`, com CI verde; pode haver commit documental posterior somente para este handoff. Dominio publico: `https://engage-fit-web-production.up.railway.app`; API e banco usam rede privada. Owner e PLATFORM_ADMIN foram homologados. Antes de nova evolucao, confirme no Railway: `OWNER_SETUP_ENABLED=false`, setup token removido/selado, Serverless desligado na API, Custom Start vazio e estrategia de pre-deploy restaurada/validada. Nao publique migrations novas enquanto o pre-deploy estiver desativado. Mantenha integracoes e envios reais desligados ate homologacao explicita.
```
