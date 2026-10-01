# SPEC-009 — Operação do dia

- Estado: `validated locally`
- Atualizada em: 2026-09-16
- Dependências: `SPEC-004` a `SPEC-008`
- Migration: não aplicável; projeção somente leitura

## Resultado implementado

A página inicial passa a abrir a operação local do dia antes dos indicadores
históricos. Um único endpoint compõe, no timezone da unidade, aulas publicadas,
ocupação, fila, presenças, tentativas de acesso negadas, cobranças vencidas ou
com vencimento hoje e pessoas que aguardam aceite de contrato ou revisão de
PAR-Q.

O read model pertence ao dashboard e não grava nem redefine estados dos
módulos de origem. Agenda, reservas, presença, financeiro, contratos, saúde e
pessoas continuam donos dos respectivos registros. O endpoint aplica
`organization_id` e `unit_id` em todas as consultas e devolve somente nome e
referências necessárias para a ação, sem contato, respostas de saúde, conteúdo
contratual ou instruções de pagamento.

## Contrato e autorização

- `GET /api/v1/dashboard/today` usa a unidade selecionada no contexto autenticado;
- owner recebe agenda e todas as pendências;
- coach recebe somente agenda, ocupação, presença e acessos negados; cobranças,
  contrato e PAR-Q são removidos antes da resposta;
- o dia usa o timezone IANA da unidade e intervalo local semiaberto;
- não há feature flag própria ou efeito externo: a projeção depende das tabelas
  já entregues pelas SPECs 004–008.

## Experiência

- cards mobile mostram aulas, presenças, negativas e total de pendências;
- cada aula mostra horário, alvo, profissional, espaço, reservas/capacidade,
  fila e presentes;
- cobranças destacam valor aberto, vencimento e atraso;
- pessoas pendentes mostram somente a ação necessária;
- estados vazio, carregamento e erro são explícitos, e os atalhos levam às
  áreas donas da operação.

## Evidência local

- teste unitário demonstra o corte do dia no timezone da unidade;
- teste PostgreSQL executa a projeção no schema migrado e comprova que IDs de
  tenant/unidade inexistentes não retornam dados;
- suíte HTTP focada cobre a nova rota e autorização já aplicada ao dashboard;
- TypeScript e build Vite aprovados;
- Playwright em `360x800` cobre agenda, ocupação, cobrança, pessoa pendente,
  atalhos e ausência de overflow horizontal, com revisão visual.
- baseline Playwright completa aprovada com `49 passed` e `2 skipped`; respostas
  inválidas da projeção ficam contidas no estado de erro da seção Hoje.
