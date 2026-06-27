# Decisoes Arquiteturais

## ADR-001: Plataformas suportadas no MVP

O MVP suportara apenas:

- Wellhub
- TotalPass

---

## ADR-002: Formato de importacao

Importacoes serao realizadas atraves de:

- XLSX
- CSV

Motivo:

Os parceiros ja fornecem relatorios prontos nesses formatos.

---

## ADR-003: Metas por plataforma

Campanhas possuem metas diferentes por plataforma.

Exemplo:

- Wellhub -> 12 check-ins
- TotalPass -> 15 check-ins

---

## ADR-004: Dashboard como centro do produto

Dashboard e a principal funcionalidade do sistema.

O principal valor esta nos insights operacionais:

- Quem esta elegivel
- Quem esta proximo da meta
- Quem esta em risco
- Quais brindes estao pendentes
- Quem deve receber mensagem

---

## ADR-005: WhatsApp no MVP

WhatsApp faz parte do MVP.

O objetivo principal e aumentar frequencia atraves de campanhas e comunicacao ativa.

Integracao definida:

- Evolution API

---

## ADR-006: Padrao visual

Frontend deve seguir padrao visual moderno inspirado em:

- Stripe
- Linear
- Vercel
- Supabase

---

## ADR-007: Multi-tenant desde o inicio

O sistema sera multi-tenant desde o inicio.

Consequencias:

- Adicionar entidade `Box`.
- Todas as entidades operacionais principais devem ser isoladas por `box_id`.
- O usuario autenticado deve operar apenas dentro do seu box.

---

## ADR-008: Usuario e perfil do MVP

Adicionar entidade `User`.

O MVP tera apenas um perfil de usuario:

- `OWNER`

Nao havera RBAC complexo no MVP.

---

## ADR-009: Configuracao WhatsApp

Adicionar entidade `WhatsappSettings` para configuracao da Evolution API por box.

Essa entidade deve armazenar:

- URL base
- Nome da instancia
- Chave de API protegida
- Status ativo/inativo

---

## ADR-010: Auditoria de disparos

Adicionar entidade `MessageRecipient`.

Motivo:

Cada disparo de WhatsApp precisa registrar destinatarios, status e erros individuais.

---

## ADR-011: Receita financeira fora do MVP

Remover qualquer conceito de receita financeira do MVP.

Consequencias:

- `Checkin` nao deve possuir campo `revenue`.
- Relatorios financeiros nao fazem parte do MVP.
- O produto permanece focado em engajamento e retencao.

---

## ADR-012: Regra de aluno proximo da meta

Aluno proximo da meta e aquele que atingiu pelo menos 80% da meta da sua plataforma.

Formula:

```txt
current_checkins / target_checkins >= 0.8
```

---

## ADR-013: Regra de aluno em risco

Aluno em risco e aquele que esta ha 7 dias sem realizar check-in.

Formula conceitual:

```txt
today - last_checkin_date >= 7 dias
```

---

## ADR-014: Stack frontend

O frontend deve utilizar:

- React
- Vite
- Tailwind
- shadcn/ui
- Lucide Icons

---

## ADR-015: Stack backend

O backend deve utilizar:

- Go
- Gin
- GORM
- PostgreSQL
- Arquitetura Hexagonal
