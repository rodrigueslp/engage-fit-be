# SPEC-NNN — Título

- Status: `draft`
- Versão: `0.1`
- Responsáveis: Produto / Backend / Frontend
- Depende de: `ARCH-001`
- Bloqueia: listar specs dependentes

## 1. Resultado esperado

Descrever o resultado observável para usuário e negócio, sem começar pela
solução técnica.

## 2. Problema

Descrever como a operação funciona hoje, o custo da situação e por que ela é
prioritária.

## 3. Escopo

### Incluído

- item;
- item.

### Fora do escopo

- item;
- item.

## 4. Evidências e benchmark

Registrar fontes e separar comportamento observado de inferência. Não copiar
interface ou complexidade sem justificar o benefício.

## 5. Regras e invariantes

Enumerar regras que devem permanecer verdadeiras independentemente da UI ou da
implementação.

## 6. Fluxos do usuário

Detalhar caminho principal, estados vazios, erros, retomada e comportamento em
mobile.

## 7. Experiência e conteúdo

Definir hierarquia, ação primária, campos obrigatórios, abertura progressiva,
feedback, acessibilidade e comportamento responsivo.

## 8. Modelo de domínio e dados

Definir entidades, propriedade dos dados, estados, relacionamentos,
constraints, índices, histórico e estratégia de migração.

## 9. Contratos

Definir APIs, comandos, consultas, eventos, idempotência, erros e compatibilidade.

## 10. Autorização, privacidade e auditoria

Definir escopo de organização/unidade, papéis, PII, consentimento, trilha de
auditoria, exportação, anonimização e retenção.

## 11. Observabilidade e operação

Definir logs, métricas, alertas, suporte, reconciliação e procedimentos de
falha sem registrar segredos ou PII desnecessária.

## 12. Rollout e compatibilidade

Definir feature flags, backfill, convivência com o modelo atual, ordem de
deploy, rollback e remoção futura de compatibilidade.

## 13. Critérios de aceite

Usar cenários objetivos e verificáveis.

## 14. Testes mínimos

Listar testes unitários, PostgreSQL, HTTP, frontend, mobile/E2E, concorrência e
segurança exigidos.

## 15. Decisões abertas

Manter somente decisões que realmente impeçam a spec de chegar a `ready`.
