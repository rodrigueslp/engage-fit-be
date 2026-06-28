# Dominio

## Box

Representa um box de CrossFit cliente do EngageFit.

O sistema sera multi-tenant desde o inicio. Todas as entidades operacionais principais devem pertencer a um `Box`.

Relacionamentos principais:

- Um box possui usuarios
- Um box possui alunos
- Um box possui importacoes
- Um box possui campanhas
- Um box possui templates e campanhas de mensagem
- Um box possui configuracao de WhatsApp

---

## User

Representa um usuario autenticado da plataforma.

No MVP existira apenas um perfil:

- `OWNER`

O usuario `OWNER` administra um box, acessa o dashboard, cria campanhas, importa check-ins, controla brindes, configura WhatsApp e dispara mensagens.

---

## Student

Aluno identificado atraves das importacoes.

Origens suportadas no MVP:

- Wellhub
- TotalPass

Um aluno pertence a um box.

No MVP, a identidade do aluno sera baseada na combinacao de box, origem e identificador externo quando disponivel. Quando nao houver identificador externo, o sistema podera usar dados importados como nome, email e telefone conforme regra do parser.

---

## Checkin

Representa um check-in realizado por um aluno.

Um check-in pertence a:

- Um box
- Um aluno
- Uma origem: Wellhub ou TotalPass
- Uma importacao

A entidade `Checkin` nao possui campo de receita financeira no MVP.

---

## ImportHistory

Representa uma importacao de arquivo XLSX ou CSV.

Registra:

- Box
- Nome do arquivo
- Origem
- Total de registros
- Data da importacao

---

## Campaign

Campanha de incentivo criada por um box.

Exemplos:

- Camiseta Junho
- Garrafa Julho
- Mochila Agosto

Uma campanha possui:

- Periodo
- Status ativo/inativo
- Metas por plataforma
- Brindes
- Progresso dos alunos

---

## CampaignGoal

Define a meta de check-ins por plataforma dentro de uma campanha.

Exemplo:

- Wellhub -> 12 check-ins
- TotalPass -> 15 check-ins

Campanhas possuem metas diferentes por plataforma.

---

## Reward

Brinde associado a uma campanha.

Exemplos:

- Camiseta
- Garrafa
- Mochila
- Shake

Um brinde possui quantidade planejada/disponivel e pode gerar entregas para alunos elegiveis.

---

## CampaignProgress

Representa o progresso de um aluno em uma campanha.

Contem:

- Check-ins atuais
- Meta aplicavel
- Percentual de progresso
- Indicacao de meta atingida
- Indicacao de aluno proximo da meta

Regra definitiva:

- Aluno proximo da meta e aquele que atingiu pelo menos 80% da meta da sua plataforma.

---

## RewardDelivery

Representa a entrega de um brinde para um aluno.

Controla:

- Brinde
- Aluno
- Status de entrega
- Data de entrega

---

## MessageTemplate

Template de mensagem usado em disparos via WhatsApp.

Pertence a um box.

---

## MessageCampaign

Representa um disparo de mensagens para um publico segmentado.

Publicos do MVP:

- Proximos da meta
- Meta atingida
- Inativos
- Todos os alunos

---

## MessageRecipient

Representa a auditoria individual de cada destinatario de uma campanha de mensagem.

Registra:

- Campanha de mensagem
- Aluno
- Telefone usado
- Status do envio
- Erro, quando houver
- Data/hora de envio

---

## WhatsappSettings

Configuracao do provedor WhatsApp para um box.

Registra:

- URL base
- Nome da instancia
- Chave de API protegida
- Status ativo/inativo

---

## Regras Transversais

### Multi-tenant

Toda consulta operacional deve ser filtrada por `box_id`.

### Aluno proximo da meta

Aluno com progresso maior ou igual a 80% da meta da sua plataforma.

### Aluno em risco

Aluno ha 7 dias sem realizar check-in.

### Receita financeira

Receita financeira esta fora do MVP.

`Checkin` nao deve possuir campo `revenue`.
