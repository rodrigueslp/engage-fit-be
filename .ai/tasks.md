# Tasks

## Documentacao e Blueprint

- [x] Definir multi-tenant desde o inicio
- [x] Adicionar entidade Box
- [x] Adicionar entidade User
- [x] Definir perfil unico `OWNER`
- [x] Adicionar WhatsappSettings
- [x] Adicionar MessageRecipient
- [x] Remover receita financeira do MVP
- [x] Remover `Checkin.revenue`
- [x] Definir regra de proximo da meta como 80%
- [x] Definir regra de aluno em risco como 7 dias sem check-in
- [x] Consolidar stack backend
- [x] Consolidar stack frontend

---

## Infraestrutura

- [x] Criar estrutura backend Go
- [x] Configurar PostgreSQL
- [x] Configurar Docker Compose
- [x] Configurar migrations
- [x] Configurar variaveis de ambiente

---

## Backend

- [x] Criar estrutura hexagonal
- [x] Criar camada de dominio
- [x] Criar camada de casos de uso
- [x] Criar portas de repositorio
- [x] Criar adaptadores HTTP
- [x] Criar adaptadores PostgreSQL com GORM
- [x] Criar tratamento centralizado de erros

---

## Handlers HTTP

- [x] Conectar Auth handlers
- [x] Conectar Setup owner handler
- [x] Conectar Box handlers
- [x] Conectar Students handlers basicos
- [x] Conectar Imports handlers de leitura
- [x] Conectar Campaigns handlers de leitura/fechamento
- [x] Conectar Rewards handlers de leitura/entrega
- [x] Conectar Whatsapp settings handlers
- [x] Conectar Message templates handlers basicos
- [x] Conectar Message campaigns handlers de leitura
- [x] Conectar Imports upload real
- [x] Conectar Campaigns create/update/delete
- [x] Conectar Campaign goals create/update/delete
- [x] Conectar Campaign progress recalculation
- [x] Conectar Campaign eligible/near-goal endpoints
- [x] Conectar Rewards create/update/delete
- [x] Conectar Message templates update/delete
- [x] Conectar Message campaigns create/get/send

---

## Persistencia GORM

- [x] Criar mappers entre models e dominio
- [x] Implementar BoxRepository
- [x] Implementar UserRepository
- [x] Implementar StudentRepository
- [x] Implementar CheckinRepository
- [x] Implementar ImportHistoryRepository
- [x] Implementar CampaignRepository
- [x] Implementar RewardRepository
- [x] Implementar WhatsappSettingsRepository
- [x] Implementar MessageRepository
- [x] Adicionar asserts de compatibilidade com interfaces
- [ ] Validar repositories contra PostgreSQL real
- [ ] Adicionar testes de repositorio

---

## Frontend

- [x] Configurar React
- [x] Configurar Vite
- [x] Configurar Tailwind
- [x] Configurar primitives locais no padrao shadcn/ui
- [x] Configurar Lucide Icons
- [x] Criar layout principal
- [x] Criar sidebar
- [x] Criar header
- [x] Criar dashboard inicial
- [x] Criar tela de login
- [x] Criar tela de campanhas
- [x] Criar tela de alunos
- [x] Criar tela de importacoes
- [x] Criar tela de WhatsApp
- [x] Melhorar tela de campanhas com metas por plataforma
- [x] Integrar criacao de metas Wellhub e TotalPass
- [x] Integrar criacao de brinde da campanha
- [x] Exibir painel operacional de campanha
- [x] Retornar nome do aluno diretamente no progresso de campanha
- [x] Criar templates com variaveis de campanha
- [x] Criar seed demo para desenvolvimento
- [x] Criar modo mock de WhatsApp para desenvolvimento
- [x] Criar tela de configuracao da Evolution API
- [x] Suportar XLSX real Wellhub com preambulo
- [x] Suportar XLSX real TotalPass tokens com preambulo
- [x] Ajustar seed demo para usar planilhas reais quando disponiveis
- [x] Impedir duplicacao de check-ins na reimportacao da mesma planilha
- [x] Proteger envio real de WhatsApp em desenvolvimento com numero override
- [ ] Implementar e-mail personalizado
- [ ] Implementar automacao diaria de importacao e disparos

---

## Auth

- [x] Login
- [x] JWT
- [x] Middleware de autenticacao
- [x] Resolver box do usuario autenticado
- [x] Perfil unico `OWNER`

---

## Boxes e Users

- [x] Criar entidade Box
- [x] Criar entidade User
- [x] Criar seed ou fluxo inicial de owner
- [x] Garantir isolamento por `box_id`

---

## Students

- [x] Cadastro automatico
- [x] Busca
- [x] Filtros
- [x] Detalhe do aluno
- [x] Historico de check-ins

---

## Imports

- [x] Parser Wellhub
- [x] Parser TotalPass
- [x] Upload XLSX
- [x] Upload CSV
- [x] Historico de importacao
- [x] Criacao automatica de alunos
- [x] Criacao de check-ins sem receita
- [x] Recalculo de progresso

---

## Campaigns

- [x] CRUD campanhas
- [x] Encerrar campanha
- [x] CRUD metas
- [x] Recalcular progresso
- [x] Listar elegiveis
- [x] Listar proximos da meta

---

## Rewards

- [x] CRUD brindes
- [x] Controle de quantidade
- [x] Controle de entrega
- [x] Listar brindes pendentes
- [x] Marcar brinde como entregue

---

## Dashboard

- [x] KPIs principais
- [x] Campanhas ativas
- [x] Proximos da meta
- [x] Elegiveis
- [x] Alunos em risco
- [x] Brindes pendentes

---

## WhatsApp

- [x] Entidade WhatsappSettings
- [x] Configuracao Evolution API
- [x] Templates
- [x] Campanhas de mensagem
- [x] MessageRecipient para auditoria
- [x] Disparo segmentado
- [x] Registro de falhas de envio

---

## Relatorios

- [x] Exportar elegiveis
- [x] Exportar brindes pendentes
- [x] Relatorio de frequencia mensal

---

## Regras de Negocio

- [x] Proximo da meta: progresso >= 80%
- [x] Em risco: 7 dias sem check-in
- [x] Elegivel: check-ins >= meta
- [x] Todas as consultas operacionais filtradas por `box_id`
