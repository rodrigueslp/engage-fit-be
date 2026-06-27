# EngageFit

## Visao

EngageFit e uma plataforma multi-tenant de engajamento e retencao para boxes de CrossFit, com caminho natural para atender academias no geral.

O objetivo e transformar relatorios de check-ins da Wellhub e TotalPass em acoes praticas que aumentem a frequencia dos alunos, melhorem a retencao e ajudem gestores de unidades fitness a executar campanhas de incentivo com controle de metas, brindes e comunicacao via WhatsApp.

O sistema nao e um ERP.

O sistema e uma plataforma de engajamento baseada em dashboard, campanhas, metas, brindes, importacoes e automacoes de mensagem.

---

## Posicionamento

O EngageFit deve ser simples, moderno e orientado a decisao.

O dashboard e a principal funcionalidade do sistema e deve responder rapidamente:

- Quantos alunos existem no box?
- Quantos check-ins ocorreram no periodo?
- Como os check-ins se distribuem entre Wellhub e TotalPass?
- Quais alunos atingiram a meta?
- Quais alunos estao proximos da meta?
- Quais alunos estao em risco?
- Quantos brindes estao pendentes?
- Quem deve receber mensagem hoje?

---

## Problema

Boxes recebem relatorios de check-ins diariamente ou periodicamente.

Esses dados normalmente sao pouco aproveitados porque ficam em planilhas, sem visao consolidada, sem segmentacao e sem acao operacional clara.

Gestores nao conseguem responder facilmente:

- Quantos alunos atingiram a meta?
- Quantos alunos estao proximos?
- Quem esta em risco de abandono?
- Quantos brindes preciso entregar?
- Quem deve receber mensagem hoje?
- Qual campanha esta performando melhor?

---

## Solucao

Importar relatorios da Wellhub e TotalPass e transformar os dados em:

- Dashboard operacional
- Campanhas de incentivo
- Metas por plataforma
- Controle de brindes
- Relatorios
- Templates de mensagem
- Disparos de WhatsApp via Evolution API
- Auditoria dos disparos

---

## Publico

### Primario

- Donos de box

### Futuro

- Gestores de academias
- Coaches
- Times administrativos

No MVP havera apenas um perfil de usuario: `OWNER`.

---

## Objetivos do MVP

- Operar em modelo multi-tenant desde o inicio
- Cadastrar boxes e usuarios donos
- Importar check-ins da Wellhub e TotalPass via XLSX e CSV
- Criar campanhas com metas diferentes por plataforma
- Identificar alunos elegiveis, proximos da meta e em risco
- Controlar brindes pendentes e entregues
- Automatizar comunicacao via WhatsApp
- Registrar auditoria dos disparos
- Disponibilizar relatorios essenciais

---

## Fora do Escopo do MVP

- ERP financeiro
- Controle de mensalidades
- Controle financeiro de receita
- Campo de receita em check-ins
- Perfis avancados alem de `OWNER`
- Integracao direta com APIs Wellhub e TotalPass
- Email transacional via Resend

---

## Regras de Negocio Chave

- Aluno proximo da meta e aquele que atingiu pelo menos 80% da meta da sua plataforma na campanha.
- Aluno em risco e aquele que esta ha 7 dias sem realizar check-in.
- Campanhas podem ter metas diferentes por plataforma.
- WhatsApp faz parte do MVP.
- O dashboard e a tela central do produto.
