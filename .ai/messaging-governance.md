# Governança de mensageria WhatsApp do EngageFit

Atualizado em: 2026-07-16

## Objetivo

Este documento é a referência funcional, técnica, operacional e de segurança para o controle de mensagens WhatsApp do EngageFit.

O objetivo da governança é permitir que diversas academias utilizem conexões compartilhadas ou dedicadas sem que uma delas consiga:

- consumir a franquia de outra academia;
- provocar picos de envio não planejados;
- gerar custos acima do orçamento aceito pelo EngageFit;
- disparar mensagens enquanto estiver administrativamente bloqueada;
- ocultar quem iniciou um envio, qual fluxo o originou ou qual conexão foi utilizada.

A regra central é:

> A conexão determina por onde a mensagem sai. A política determina se a mensagem pode sair.

Existe um único mecanismo de governança para os dois modos de conexão. Não há um sistema de limites separado para número compartilhado e número dedicado.

## Modos de conexão

### `platform`

- Usa a conta, credenciais, remetente e Content SIDs centrais do EngageFit.
- Diversas academias podem compartilhar o mesmo número.
- Cada academia mantém sua própria franquia e orçamento.
- Além dos limites individuais, todo disparo consome a política global da conexão compartilhada.
- Se a política global estiver bloqueada ou esgotada, nenhuma academia consegue enviar pelo número central.

### `dedicated`

- Usa conta, credenciais, remetente e Content SIDs dedicados à academia.
- Apenas a política individual da academia é aplicada.
- O custo ainda é controlado pelo EngageFit enquanto a conta Twilio e os créditos forem administrados pela operação da plataforma.
- A evolução recomendada é utilizar um subaccount Twilio por academia dedicada.

## Controles implementados

Cada política possui:

| Controle | Função |
|---|---|
| Limite diário | Restringe a quantidade aceita por dia e reduz risco de picos. |
| Limite mensal | Representa a franquia comercial da academia. |
| Limite por disparo | Impede campanhas acidentalmente grandes. |
| Custo estimado por mensagem | Reserva conservadora antes da chamada externa. |
| Orçamento diário | Protege contra custos excessivos no curto prazo. |
| Orçamento mensal | Limita a exposição financeira acumulada. |
| Percentual de alerta | Referência para alertas e destaque visual. |
| Timezone | Define a virada dos períodos diário e mensal. |
| Bloqueio administrativo | Kill switch para novos disparos. |

Valores monetários são persistidos em micros da moeda configurada. Um dólar equivale a `1.000.000` micros. Não são usados números de ponto flutuante no banco.

## Políticas existentes

### Política individual

Toda academia possui uma política `box`, independentemente do modo da conexão.

Novas academias recebem uma política conservadora automaticamente no primeiro acesso à governança:

- 100 mensagens por dia;
- 1.000 mensagens por mês;
- 100 destinatários por disparo;
- custo estimado de USD 0,10 por mensagem;
- orçamento diário estimado de USD 10;
- orçamento mensal estimado de USD 100;
- alerta em 80%;
- timezone `America/Sao_Paulo`.

Esses valores são proteções iniciais, não uma definição comercial definitiva. Devem ser revisados por plano e pelo custo real observado.

### Política global da plataforma

A conexão compartilhada possui uma política adicional `platform`:

- 1.000 mensagens por dia;
- 10.000 mensagens por mês;
- 250 destinatários por disparo;
- custo estimado de USD 0,10 por mensagem;
- orçamento diário estimado de USD 100;
- orçamento mensal estimado de USD 1.000.

Um envio pelo número compartilhado precisa ser autorizado simultaneamente pela política da academia e pela política global. Prevalece sempre o limite mais restritivo.

## Fluxo de um disparo

1. O caso de uso resolve a audiência efetiva.
2. Destinatários sem telefone são removidos.
3. O resolver determina se a conexão é `platform` ou `dedicated`.
4. A governança inicia uma transação no PostgreSQL.
5. A política individual e, quando aplicável, a global são bloqueadas para atualização.
6. Os buckets diário e mensal são criados ou bloqueados.
7. O sistema valida bloqueio, quantidade por disparo, quantidade diária, quantidade mensal e orçamento estimado.
8. Se qualquer regra falhar, um `message_dispatch` com status `blocked` e motivo é registrado. Nenhuma chamada externa ocorre.
9. Se autorizado, quantidade e custo são reservados atomicamente.
10. O envio segue para o gateway WhatsApp.
11. Cada sucesso síncrono armazena o identificador e status inicial devolvidos pelo provedor.
12. Ao final, a reserva é concluída: mensagens aceitas viram consumo; falhas síncronas liberam franquia e orçamento reservado.

As reservas impedem corrida entre campanhas manuais e automações. Se restarem 100 mensagens e dois processos tentarem reservar 100 ao mesmo tempo, apenas o primeiro poderá concluir a reserva.

## O que conta como consumo

Na implementação atual:

- `reserved`: envio autorizado, mas ainda em processamento;
- `accepted`: o provedor aceitou sincronamente a criação da mensagem;
- `failed`: falha ocorrida antes de o provedor aceitar a mensagem;
- mensagens bloqueadas não consomem franquia;
- falhas síncronas liberam a reserva;
- uma mensagem aceita continua contando mesmo que posteriormente termine como `undelivered`, até que exista conciliação confiável do custo real.

Essa abordagem é conservadora. Uma aceitação da Twilio pode gerar custo e não deve ser automaticamente estornada apenas porque a entrega final falhou.

## Disparos cobertos

O serviço central está integrado a:

- campanhas manuais de WhatsApp;
- campanhas executadas pela automação;
- mensagens do Treino do dia.

Novos fluxos WhatsApp devem obrigatoriamente depender de `MessagingGovernance`. Não é permitido chamar o `WhatsappGateway` diretamente a partir de uma nova feature.

## Rastreabilidade

Cada disparo registra:

- academia;
- usuário solicitante, quando manual;
- tipo e ID da origem (`message_campaign` ou `workout_draft`);
- modo de conexão efetivo;
- total de destinatários;
- quantidade reservada, aceita e falha;
- custo estimado;
- status e motivo de bloqueio;
- criação e conclusão.

Cada destinatário passa a registrar:

- `dispatch_id`;
- `provider_message_sid`;
- `provider_status` inicial;
- status local;
- erro síncrono;
- horário do envio.

Alterações de política geram `admin_audit_logs` contendo administrador, estado anterior, estado posterior, motivo, IP, alvo e horário.

## Controle de acesso

### `OWNER`

- Continua restrito ao próprio `box_id`.
- Pode consultar a própria franquia e consumo em `GET /api/v1/messaging/usage`.
- Não pode listar outras academias nem alterar limites.
- Recebe `403` ao tentar acessar `/api/v1/admin/*`.

### `PLATFORM_ADMIN`

- Não pertence a nenhuma academia; seu `box_id` é nulo.
- Acessa somente o plano administrativo autenticado.
- Não consegue usar endpoints tenant como `/api/v1/box`.
- Pode consultar e alterar políticas individuais e a política global.
- Toda alteração exige motivo e é auditada.

Um administrador é provisionado ou tem sua senha rotacionada na inicialização por:

```env
PLATFORM_ADMIN_NAME=Administrador EngageFit
PLATFORM_ADMIN_EMAIL=admin@dominio-seguro.com
PLATFORM_ADMIN_PASSWORD=uma-senha-longa-e-exclusiva
```

`PLATFORM_ADMIN_EMAIL` e `PLATFORM_ADMIN_PASSWORD` devem ser configurados juntos. Em produção, a senha deve vir do gerenciador de segredos do ambiente, nunca de arquivo versionado.

## API administrativa

| Método | Endpoint | Finalidade |
|---|---|---|
| `GET` | `/api/v1/admin/messaging/boxes` | Lista academias, conexão, política e consumo atual. |
| `GET` | `/api/v1/admin/messaging/boxes/:id/policy` | Consulta uma política individual. |
| `PUT` | `/api/v1/admin/messaging/boxes/:id/policy` | Altera limites, orçamento ou bloqueio individual. |
| `GET` | `/api/v1/admin/messaging/platform/policy` | Consulta a proteção global compartilhada. |
| `PUT` | `/api/v1/admin/messaging/platform/policy` | Altera ou bloqueia a conexão compartilhada. |
| `GET` | `/api/v1/admin/messaging/boxes/:id/whatsapp-settings` | Consulta a conexão mascarada da academia. |
| `PUT` | `/api/v1/admin/messaging/boxes/:id/whatsapp-settings` | Configura modo, provedor, remetente e credencial. |
| `POST` | `/api/v1/admin/messaging/boxes/:id/whatsapp-settings/test` | Testa dados salvos ou ainda não persistidos. |
| `GET` | `/api/v1/messaging/usage` | Permite ao owner consultar apenas o próprio uso. |

Exceder uma política retorna HTTP `429 Too Many Requests` e uma mensagem operacional. Falha do provedor continua retornando erro de gateway.

## Interface

A rota `#admin-messaging`, visível somente para `PLATFORM_ADMIN`, contém:

- consumo e orçamento global do número EngageFit;
- seletor de academia;
- identificação do modo compartilhado ou dedicado;
- consumo diário e mensal;
- estimativa financeira;
- edição de limites;
- gestão da conexão e credencial dedicada;
- kill switch;
- motivo obrigatório da alteração.

A tela WhatsApp do owner exibe uso diário, uso mensal, máximo por disparo e eventual bloqueio administrativo.

## Modelo de dados

### `messaging_policies`

Fonte das regras por academia e da proteção global.

### `messaging_usage_buckets`

Contadores transacionais diários e mensais. São a fonte rápida de autorização antes do envio.

### `message_dispatches`

Ledger de tentativas de disparo, inclusive bloqueadas.

### `message_recipients` e `workout_message_recipients`

Auditoria por destinatário, agora enriquecida com dispatch, SID e status do provedor.

### `admin_audit_logs`

Histórico imutável das mudanças administrativas.

## Operação segura recomendada

### Antes de ativar uma academia

1. Definir o modo de conexão.
2. Definir franquia comercial mensal.
3. Definir limite diário inferior à franquia mensal.
4. Definir limite por disparo compatível com a audiência normal.
5. Definir custo unitário de forma conservadora.
6. Definir orçamento abaixo do prejuízo máximo aceitável.
7. Fazer disparo controlado para destinatário autorizado.
8. Verificar ledger e identificador do provedor.

### Ao alterar um plano

- Registrar no motivo o plano, aprovação ou chamado relacionado.
- Nunca elevar simultaneamente todos os limites sem revisar o orçamento financeiro.
- Comparar o custo médio real recente com a estimativa unitária.
- Manter margem para variação cambial e recategorização de templates pela Meta.

### Em incidente

1. Bloquear a academia se o problema estiver isolado.
2. Bloquear a política global se houver risco na conexão compartilhada.
3. Desativar `WHATSAPP_PLATFORM_ENABLED` se for necessário cortar a infraestrutura central.
4. Revogar a API key da Twilio se houver suspeita de vazamento.
5. Suspender o subaccount dedicado, quando existir.
6. Preservar dispatches, recipients e auditoria para investigação.
7. Reativar somente após identificar causa e ajustar limites.

## Segurança das credenciais Twilio

Estado atual:

- credenciais centrais ficam em variáveis de ambiente;
- credenciais dedicadas ainda usam a estrutura legada de `whatsapp_settings`;
- credenciais nunca são devolvidas integralmente pela API;
- o campo legado se chama `APIKeyEncrypted`, mas a garantia de criptografia forte e gestão de chaves precisa ser revisada antes da produção ampla.

Evolução obrigatória recomendada:

1. Mover credenciais para um secret manager.
2. Persistir apenas `secret_ref` no banco.
3. Usar API keys restritas e revogáveis no lugar do Auth Token principal.
4. Rotacionar chaves por conexão.
5. Manter credenciais e modo de conexão editáveis somente por `PLATFORM_ADMIN`; owners possuem apenas leitura e teste da conexão já salva.
6. Exigir MFA para todos os administradores da plataforma.

## Subaccounts Twilio

Recomendação:

- número compartilhado: manter na conta/conexão central e separar consumo por academia no ledger do EngageFit;
- número dedicado: criar um subaccount Twilio por academia sempre que o onboarding permitir;
- guardar o SID do subaccount na futura entidade de conexão;
- criar Usage Triggers financeiros no subaccount como segunda camada de alerta;
- suspender somente o subaccount afetado em caso de inadimplência ou incidente.

Subaccounts melhoram isolamento e conciliação, mas não criam saldo separado: o custo continua chegando à conta principal. Eles complementam, não substituem, os bloqueios do EngageFit.

## Pendências e próximas fases

### Prioridade alta

1. Implementar `StatusCallback` da Twilio com validação da assinatura oficial.
2. Atualizar recipients para `queued`, `sent`, `delivered`, `undelivered`, `failed` e `read`.
3. Criar job de conciliação para obter `price` e `price_unit` quando disponíveis.
4. Alimentar `actual_cost_micros` sem substituir o custo estimado até a conciliação estar completa.
5. Alertar administrador em 70%, 85%, 95% e 100% dos limites.
6. Alertar reservas antigas que indiquem processo interrompido.
7. Criar tela de dispatches e auditoria administrativa.

### Prioridade média

1. Criar entidade explícita `messaging_connections` para conexões centrais e dedicadas.
2. Automatizar criação e suspensão de subaccounts Twilio.
3. Versionar políticas e permitir vigência futura por plano.
4. Implementar exceção pontual de limite com aprovação e expiração.
5. Adicionar métricas e alertas por taxa de falha, `63051`, template rejeitado e remetente inválido.

## Decisões que não devem ser revertidas sem revisão

- Não confiar apenas em Usage Triggers da Twilio: eles são assíncronos.
- Não contabilizar academias do número compartilhado apenas pelo relatório agregado da Twilio.
- Não fazer checagem de limite somente na UI.
- Não contar destinatários e enviar sem reserva transacional.
- Não permitir envio parcial silencioso quando a campanha exceder o saldo.
- Não liberar reserva de mensagem aceita apenas porque ela terminou `undelivered`.
- Não usar ponto flutuante para valores financeiros persistidos.
- Não expor credenciais centrais ou dedicadas aos owners.
- Não criar novos fluxos WhatsApp fora do serviço central de governança.

## Arquivos principais

- `migrations/029_create_messaging_governance.sql`
- `internal/domain/messaging_governance.go`
- `internal/app/messaging/governance.go`
- `internal/app/platformadmin/platform_admin_usecases.go`
- `internal/adapters/persistence/postgres/repositories/messaging_governance_repository.go`
- `internal/adapters/http/handlers/messaging_governance_handler.go`
- `engage-fit-fe/src/pages/admin/MessagingGovernancePage.tsx`
