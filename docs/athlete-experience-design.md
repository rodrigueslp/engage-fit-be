# Experiência do aluno — direção de produto e arquitetura

Estado: ciclo completo implementado localmente em 2026-08-06; ainda não publicado.

## Princípios

- O box continua sendo o tenant e o aluno acessa conteúdo por meio de um
  vínculo explícito com ele.
- A operação do coach deve ser mínima. Para publicar o treino, data e texto
  livre são suficientes; classificação e enriquecimento são automáticos.
- Nenhuma classificação automática exige aprovação do coach para aparecer ao
  aluno. Quando a IA produzir recomendações individualizadas, elas devem ser
  explicáveis, conservadoras e apresentadas como apoio, não prescrição médica.
- O texto original do box é sempre preservado. Estrutura, movimentos e formatos
  são derivados versionados que podem ser recalculados.
- Identidade global nunca é inferida apenas por nome ou por um score de IA.

## Produto proposto

A experiência mobile/PWA do aluno usa o mesmo `engage-fit-be` e um frontend
próprio. O fluxo principal é:

1. o box cola e publica o treino;
2. o aluno vinculado vê o treino original e sua organização automática;
3. o EngageFit acrescenta explicação, pacing e faixas de carga baseadas em
   dados confirmados do atleta;
4. o aluno registra resultado, escala, carga e esforço percebido;
5. o histórico detecta possíveis PRs e pede confirmação;
6. resultados posteriores alimentam comparações e evolução pessoal.

O MVP implementado cobre publicação text-first, conta e sessão próprias do
atleta, convite individual, vínculo explícito com o box, feed de treinos
publicados, histórico e PWA mobile-first.

## Identidade e vínculo com o box

`students` continua representando o registro operacional isolado por
`(box, source)`. A conta global não substitui esses registros. O modelo atual
separa:

- `athlete_accounts`: conta pertencente à pessoa;
- `athlete_box_memberships`: vínculo explícito entre conta e box;
- `athlete_student_links`: ligação auditável entre o vínculo e um ou mais
  registros `students` de Wellhub, TotalPass ou mensalista;
- `athlete_invitations`: tokens individuais, opacos, armazenados somente como
  hash, com validade de sete dias e uso único;
- `athlete_sessions`: sessões opacas independentes da autenticação do box.

Uma conta pode possuir vínculos com vários boxes e, dentro de um box, ligar
mais de um registro de origem. PRs e diário pessoal pertencem à conta; treino,
check-ins e notas operacionais permanecem no contexto do box.

Antes de uma reivindicação verificada, o sistema possui candidatos, não uma
pessoa globalmente deduplicada. Formas previstas de vínculo, em ordem de
confiança:

1. convite individual emitido a partir de um `student_id` (implementado);
2. código curto presencial emitido pelo box;
3. ativação genérica usando box, origem, nome, presença recente e canal
   verificado, reaproveitando `contact_activation_requests`;
4. revisão humana somente quando o vínculo de identidade for ambíguo.

O e-mail escolhido pelo atleta é único para login, mas não é usado para inferir
que dois cadastros operacionais são a mesma pessoa. Um novo box só é ligado à
conta existente quando o atleta recebe o convite daquele `student_id` e informa
a senha da conta. Nome nunca é chave global. Verificação de e-mail/passkey e
fluxos reversíveis de merge continuam como endurecimento futuro.

## Experiência PWA implementada

- rota pública `#/athlete/invite/:token` para onboarding contextualizado pelo
  nome do aluno e do box;
- rota `#/athlete/login` e área autenticada `#/athlete`;
- home mobile com treino em destaque, blocos classificados, formatos e texto
  original preservado;
- histórico de treinos publicados e perfil com todos os boxes vinculados;
- navegação inferior, safe areas, estados de carregamento/vazio/erro, animação
  respeitando `prefers-reduced-motion` e ação nativa de instalação;
- `manifest.webmanifest` e service worker com cache conservador apenas do shell;
  chamadas de API permanecem network-only para não exibir treino privado
  obsoleto após logout.
- seleção explícita do box quando a conta possui mais de um vínculo;
- detalhe de treino e registro flexível por treino/bloco com RX, scaled ou
  adaptado, tempo, rounds/reps, carga, distância, calorias, conclusão, RPE e
  notas pessoais;
- histórico pessoal combina treino publicado e resultado efetivamente salvo;
- cargas, repetições e tempos elegíveis geram possíveis PRs; o atleta confirma
  quais são oficiais e uma edição do resultado recalcula a melhor marca;
- pacing e faixas de carga usam regras determinísticas e somente PRs
  confirmados. Faixas são referência explicável, nunca prescrição;
- explicação contextual sob demanda usa OpenAI quando `FEATURE_LLM_ENABLED` e
  credencial estão ativos. A entrada contém o cálculo determinístico e a saída
  fica em cache por atleta/treino/hash; qualquer falha usa explicação `rules-v1`;
- recuperação de senha invalida sessões anteriores, e verificação de e-mail usa
  tokens opacos, de uso único e validade curta enviados pela configuração SMTP
  do primeiro box ativo;
- convite do owner possui ação direta `wa.me`, além de cópia do link.

## Endurecimentos posteriores

- passkeys e segundo fator são evolução opcional; e-mail verificado e
  recuperação já estão cobertos;
- notificações push e modo offline de dados privados permanecem fora do escopo
  por exigirem política própria de consentimento, revogação e cache seguro.
