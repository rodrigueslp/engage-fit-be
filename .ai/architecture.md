# Arquitetura

## Visao Geral

O EngageFit sera uma aplicacao web multi-tenant composta por:

- Backend em Go
- API REST com Gin
- Persistencia com GORM
- Banco PostgreSQL
- Frontend em React com Vite
- UI com Tailwind, shadcn/ui e Lucide Icons
- Autenticacao com JWT
- Upload de arquivos XLSX e CSV
- Integracao WhatsApp via Twilio WhatsApp
- Deploy local/inicial com Docker Compose

Arquitetura do backend:

- Hexagonal Architecture

---

## Principios Arquiteturais

- Multi-tenant desde o inicio
- Dashboard first
- Separacao clara entre dominio, casos de uso e adaptadores
- Regras de negocio fora dos handlers HTTP
- Persistencia acessada por interfaces de repositorio
- Integracoes externas atras de portas/interfaces
- UI moderna, simples e operacional
- MVP focado em engajamento, nao ERP

---

## Backend

Tecnologias definitivas:

- Go
- Gin
- GORM
- PostgreSQL
- JWT
- Hexagonal Architecture

Camadas:

- Domain: entidades e regras centrais
- Application: casos de uso
- Ports: interfaces de repositorios e servicos externos
- Adapters: HTTP, PostgreSQL, parsers, Twilio WhatsApp e Meta Cloud API
- Config: configuracao de ambiente

---

## Banco

Banco definitivo:

- PostgreSQL

Caracteristicas:

- Modelo relacional
- Multi-tenant com `box_id`
- Auditoria basica com `created_at` e `updated_at`
- Sem campo de receita financeira no MVP
- Check-ins sem `revenue`

---

## Frontend

Tecnologias definitivas:

- React
- Vite
- Tailwind
- shadcn/ui
- Lucide Icons
- Inter

Direcao:

- Dashboard como tela principal
- Sidebar fixa em desktop
- Navegacao simples
- Componentes reutilizaveis
- Tabelas apenas quando agregam valor operacional

---

## Autenticacao

Autenticacao:

- JWT

Perfil do MVP:

- `OWNER`

Regras:

- Usuario pertence a um box.
- Todas as consultas autenticadas devem respeitar o `box_id` do usuario.
- Nao havera RBAC complexo no MVP.

---

## Uploads

Formatos suportados:

- XLSX
- CSV

Origens suportadas:

- Wellhub
- TotalPass

Cada origem tera parser especifico.

---

## WhatsApp

Integracao:

- Twilio WhatsApp

Faz parte do MVP.

Componentes necessarios:

- `WhatsappSettings`
- `MessageTemplate`
- `MessageCampaign`
- `MessageRecipient`

Objetivo:

- Enviar mensagens para alunos proximos da meta, alunos com meta atingida, alunos inativos e todos os alunos.

---

## Regras de Segmentacao

### Proximo da meta

Aluno que atingiu pelo menos 80% da meta da sua plataforma.

### Em risco

Aluno que esta ha 7 dias sem realizar check-in.

### Elegivel

Aluno que atingiu a meta da campanha.

---

## Deploy

MVP:

- Docker Compose

Servicos esperados:

- Backend
- Frontend
- PostgreSQL

---

## Integracoes Futuras

- Wellhub API
- TotalPass API
- Resend
- Perfis adicionais de usuario
- Regras avancadas de risco
- Deduplicacao avancada de alunos entre plataformas
