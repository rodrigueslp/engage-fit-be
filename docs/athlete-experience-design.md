# Experiência do aluno — direção de produto e arquitetura

Estado: MVP navegável implementado em 2026-08-05.

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

## Próximos recortes

1. Registro de resultado e modelo de PRs confirmados/estimados.
2. Verificação adicional de identidade (e-mail/passkey) e recuperação de senha.
3. Compartilhamento do convite diretamente pelo WhatsApp, sem exigir copiar o
   link manualmente.
4. Enriquecimento personalizado inspirado no WodScope, usando primeiro
   cálculos determinísticos e LLM para explicação contextual.
