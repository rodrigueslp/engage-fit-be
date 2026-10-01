# SPEC-014 — Comunicados e notificações

- Estado: `validated locally`
- Atualizada em: 2026-09-17
- Dependências: `SPEC-002`, `SPEC-003` e PWA atual
- Migration: `070_create_announcements.sql`

## Resultado implementado

O owner publica comunicados dentro do aplicativo do aluno, com prioridade,
agendamento e audiência segmentada. A publicação congela os destinatários
elegíveis para manter histórico estável; cada abertura na PWA registra o
primeiro e o último acesso e incrementa o contador de visualizações.

Este recorte implementa notificação in-app. Push, WhatsApp, e-mail e qualquer
outro efeito externo permanecem separados e desabilitados.

## Regras e contratos

- estados `draft`, `scheduled`, `published` e `archived`, sem edição depois da
  publicação;
- prioridades `info`, `important` e `urgent`;
- audiências: todos os atletas ativos da unidade, origem
  (`box_member`, `wellhub`, `totalpass`) ou plano ativo;
- audiência resolvida por vínculos explícitos conta/membership/student/person,
  nunca por busca de PII;
- owner-only em `/api/v1/units/:unit_id/announcements*`, com tenant e unidade
  validados pelo contexto autenticado;
- atleta autenticado lê `/api/v1/athlete/announcements` e registra abertura em
  `/api/v1/athlete/announcements/:id/view`;
- `FEATURE_ANNOUNCEMENTS_ENABLED` controla toda a superfície e fica desligada
  por padrão;
- exportação de privacidade inclui os recibos do titular; nenhum telefone,
  e-mail ou corpo de provider é duplicado nos destinatários.

## Experiência

- tela `Comunicados` mobile-first para rascunho, público, prioridade,
  publicação imediata/agendada, arquivamento e histórico com alcance/leitura;
- nova aba `Avisos` na PWA, filtrada pelo membership selecionado, com destaque
  visual por prioridade, estado não lido e conteúdo expansível;
- abrir o card registra o recibo sem depender de provider externo.

## Evidência local

- regras unitárias cobrem validação, publicação imediata e agendamento;
- integração PostgreSQL prova snapshot de audiência, isolamento pelo vínculo,
  leitura do atleta e contagem de visualizações;
- migrations `001..070` aplicadas em PostgreSQL vazio e segundo run sem novas
  aplicações;
- `go test ./...`, TypeScript e build Vite aprovados;
- Playwright focado cobre owner e atleta em `360x800` (`2 passed`).
- baseline Playwright completa aprovada (`52 passed`, `2 skipped`).

## Fora do escopo preservado

- Web Push, APNs/FCM, WhatsApp, SMS ou e-mail;
- envio para pessoa sem conta ativa na PWA;
- campanhas recorrentes, automações ou reuso da governança de custo de
  mensageria externa.
