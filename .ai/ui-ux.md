# UI / UX Guidelines

## Objetivo

O EngageFit deve transmitir a sensacao de um produto moderno, premium, simples e operacional.

O dashboard deve ser a principal tela do sistema.

A interface deve ajudar o dono do box a decidir rapidamente:

- Quem precisa de acao hoje
- Quem esta perto da meta
- Quem esta em risco
- Quais brindes precisam ser entregues
- Qual campanha esta ativa
- Qual mensagem deve ser enviada

---

## Referencias

O padrao visual deve ser inspirado em:

- Stripe
- Linear
- Vercel
- Supabase

Referencia secundaria:

- Notion

---

## Design Principles

- Clean
- Moderno
- Minimalista
- Dashboard first
- Mobile friendly
- Operacional
- Claro e escaneavel
- Poucas cores
- Hierarquia visual forte

Evitar:

- Visual de ERP antigo
- Interface poluida
- Muitas tabelas sem necessidade
- Excesso de cores
- Gradientes pesados
- Elementos decorativos sem funcao

---

## Stack Visual

Frontend deve utilizar:

- React
- Vite
- Tailwind
- shadcn/ui
- Lucide Icons
- Fonte Inter

Nao utilizar:

- Bootstrap
- Material UI
- Ant Design

---

## Layout

### Sidebar

Itens principais:

- Dashboard
- Campanhas
- Alunos
- Importacoes
- WhatsApp
- Relatorios
- Configuracoes

### Header

Deve exibir:

- Nome do box
- Usuario logado
- Acoes rapidas

Acoes rapidas sugeridas:

- Importar check-ins
- Criar campanha
- Enviar mensagem

---

## Dashboard

Prioridade visual:

1. KPIs
2. Campanhas ativas
3. Proximos da meta
4. Alunos em risco
5. Brindes pendentes

KPIs principais:

- Alunos ativos
- Check-ins do mes
- Elegiveis
- Proximos da meta
- Em risco
- Brindes pendentes

---

## Componentes

### KPI Cards

Devem ser compactos, claros e escaneaveis.

Exemplos:

- Alunos ativos
- Check-ins do mes
- Elegiveis
- Proximos da meta
- Alunos em risco
- Brindes pendentes

### Listas operacionais

Usar listas e tabelas simples para:

- Alunos proximos da meta
- Alunos em risco
- Brindes pendentes
- Historico de importacoes

### Badges

Usar badges para:

- Wellhub
- TotalPass
- Meta atingida
- Proximo da meta
- Em risco
- Brinde pendente
- Entregue

### Formularios

Formularios devem ser diretos e curtos.

Casos principais:

- Login
- Criar campanha
- Definir metas
- Criar brinde
- Configurar WhatsApp
- Criar template

---

## Cores

Base:

- Branco
- Cinza claro
- Cinza escuro

Destaque:

- Laranja

Uso:

- Laranja deve indicar acao principal, progresso e destaque de engajamento.
- Estados criticos devem usar cores funcionais com moderacao.

---

## Tipografia

Fonte:

- Inter

Diretrizes:

- Titulos claros
- Textos curtos
- Numeros dos KPIs com alta legibilidade
- Evitar blocos longos de texto na interface

---

## Responsividade

O sistema deve ser mobile friendly.

Prioridades no mobile:

- KPIs
- Acoes rapidas
- Listas de alunos que exigem acao
- Brindes pendentes

---

## Tom de Produto

O produto deve parecer:

- Moderno
- Confiavel
- Leve
- Profissional
- Focado em acao

O produto nao deve parecer:

- ERP legado
- Planilha sofisticada
- Sistema financeiro
- Painel excessivamente tecnico
