# Funcionalidades

## Dashboard

Dashboard e a principal funcionalidade do sistema.

Indicadores do MVP:

- Total de alunos
- Total de check-ins
- Check-ins por plataforma
- Alunos elegiveis
- Alunos proximos da meta
- Alunos em risco
- Brindes pendentes
- Brindes entregues
- Campanhas ativas

Regras:

- Aluno proximo da meta: atingiu pelo menos 80% da meta da sua plataforma.
- Aluno em risco: esta ha 7 dias sem check-in.
- Aluno elegivel: atingiu a meta da campanha.

---

## Autenticacao

- Login
- JWT
- Usuario vinculado a box
- Perfil unico no MVP: `OWNER`

---

## Boxes

- Cadastro de box
- Vinculo de usuario owner ao box
- Isolamento multi-tenant por `box_id`

---

## Importacoes

- Upload Wellhub
- Upload TotalPass
- Upload XLSX
- Upload CSV
- Historico de importacoes
- Cadastro automatico de alunos
- Registro de check-ins
- Recalculo de progresso de campanhas ativas

---

## Alunos

- Cadastro automatico via importacao
- Listagem
- Busca
- Filtros
- Visualizacao de check-ins
- Status de campanha
- Status de risco

---

## Campanhas

- Criar campanha
- Editar campanha
- Encerrar campanha
- Listar campanhas
- Visualizar progresso

---

## Metas

- Meta por plataforma

Exemplo:

- Wellhub -> 12
- TotalPass -> 15

---

## Brindes

- Criar brinde
- Associar brinde a campanha
- Controlar quantidade
- Listar brindes pendentes
- Marcar brinde como entregue

---

## WhatsApp

WhatsApp faz parte do MVP.

Funcionalidades:

- Configurar Twilio WhatsApp
- Criar templates
- Disparo individual
- Disparo em massa
- Auditoria por destinatario

Publicos:

- Proximos da meta
- Meta atingida
- Inativos
- Todos os alunos

---

## Relatorios

- Elegiveis
- Brindes pendentes
- Frequencia mensal

---

## Fora do MVP

- Receita financeira
- Campo `revenue` em check-ins
- ERP financeiro
- Perfis alem de `OWNER`
- Integracao direta via API Wellhub/TotalPass
