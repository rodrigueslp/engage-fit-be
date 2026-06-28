# Banco de Dados

Banco: PostgreSQL

O modelo sera multi-tenant desde o inicio. Todas as entidades operacionais principais devem possuir `box_id` direta ou indiretamente.

---

# Box

id
name
created_at
updated_at

---

# User

id
box_id
name
email
password_hash
role
created_at
updated_at

Regras:

- `role` no MVP aceita apenas `OWNER`.
- `email` deve ser unico.
- Usuario pertence a um box.

---

# Student

id
box_id
name
email
phone
source
external_id
created_at
updated_at

Regras:

- `source` aceita `wellhub` ou `totalpass`.
- Aluno pertence a um box.
- Indice recomendado: `box_id`, `source`, `external_id`.

---

# ImportHistory

id
box_id
filename
source
total_records
imported_at

Regras:

- `source` aceita `wellhub` ou `totalpass`.
- Importacao pertence a um box.

---

# Checkin

id
box_id
student_id
checkin_date
checkin_time
source
import_history_id
created_at

Regras:

- Check-in pertence a um box.
- Check-in pertence a um aluno.
- Check-in pertence a uma importacao.
- `source` aceita `wellhub` ou `totalpass`.
- Nao existe campo `revenue` no MVP.
- Receita financeira esta fora do MVP.

---

# Campaign

id
box_id
name
description
start_date
end_date
active
created_at
updated_at

Regras:

- Campanha pertence a um box.
- Campanha possui metas por plataforma.

---

# CampaignGoal

id
campaign_id
source
target_checkins

Regras:

- `source` aceita `wellhub` ou `totalpass`.
- Deve existir no maximo uma meta por campanha e plataforma.
- Exemplo: Wellhub -> 12, TotalPass -> 15.

---

# Reward

id
campaign_id
name
description
quantity

Regras:

- Brinde pertence a uma campanha.
- `quantity` representa a quantidade planejada/disponivel para controle operacional.

---

# CampaignProgress

id
campaign_id
student_id
current_checkins
target_checkins
progress_percentage
achieved
near_goal
updated_at

Regras:

- Deve existir no maximo um progresso por campanha e aluno.
- `achieved` e verdadeiro quando `current_checkins >= target_checkins`.
- `near_goal` e verdadeiro quando o aluno atingiu pelo menos 80% da meta da sua plataforma.
- O progresso deve ser recalculado apos importacoes relevantes.

---

# RewardDelivery

id
reward_id
student_id
delivered
delivered_at

Regras:

- Deve existir no maximo uma entrega por brinde e aluno.
- Entregas pendentes possuem `delivered = false`.
- Entregas realizadas possuem `delivered = true` e `delivered_at` preenchido.

---

# WhatsappSettings

id
box_id
base_url
instance_name
api_key_encrypted
enabled
created_at
updated_at

Regras:

- Configuracao pertence a um box.
- Chave da API deve ser armazenada de forma protegida.
- Twilio WhatsApp faz parte do MVP; Meta Cloud API fica como opcao oficial futura.

---

# MessageTemplate

id
box_id
name
content
created_at
updated_at

Regras:

- Template pertence a um box.

---

# MessageCampaign

id
box_id
name
audience
template_id
sent_at
created_at

Regras:

- Campanha de mensagem pertence a um box.
- `audience` aceita:
  - `near_goal`
  - `achieved`
  - `inactive`
  - `all`

---

# MessageRecipient

id
message_campaign_id
student_id
phone
status
error_message
sent_at
created_at

Regras:

- Registra auditoria individual dos disparos de WhatsApp.
- `status` aceita:
  - `pending`
  - `sent`
  - `failed`

---

# Regras de Segmentacao

## Aluno proximo da meta

Aluno com `progress_percentage >= 80`.

## Aluno em risco

Aluno que esta ha 7 dias sem realizar check-in.

## Meta atingida

Aluno com `current_checkins >= target_checkins`.

---

# Campos Removidos do MVP

## Checkin.revenue

Removido definitivamente do MVP.

Motivo:

- O produto nao e ERP.
- Receita financeira nao faz parte do escopo inicial.
- O foco do MVP e engajamento, frequencia, campanhas, brindes e comunicacao.
