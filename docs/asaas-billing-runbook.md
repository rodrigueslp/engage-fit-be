# EngageFit — operação do billing com Asaas

Atualizado em: 2026-07-23

Este runbook descreve a ativação, homologação e rotina do financeiro. Comece
sempre no sandbox do Asaas. Não use a chave de produção antes de concluir todos
os cenários de teste.

## Responsabilidades

- O Asaas cria cobranças, recebe o pagamento e mantém dados do meio de pagamento.
- O EngageFit mantém catálogo de planos, assinatura contratada, franquia de
  mensagens, espelho das cobranças e acesso da academia.
- O administrador da plataforma opera tudo em `Financeiro`.
- O owner consulta plano e cobranças em `Plano e cobranças`.
- Dados de cartão nunca passam pelo frontend, API ou banco do EngageFit.

## 1. Preparar o Asaas sandbox

1. Crie ou acesse uma conta sandbox em `https://sandbox.asaas.com`.
2. Gere uma chave de API em `Integrações > Chaves de API`.
3. Gere um token aleatório para autenticar o webhook:

   ```bash
   openssl rand -hex 32
   ```

4. Cadastre um webhook com:

   - URL: `https://SEU_DOMINIO/api/v1/webhooks/asaas`;
   - token de autenticação: o mesmo valor de `ASAAS_WEBHOOK_TOKEN`;
   - fila ativa;
   - eventos de pagamento habilitados, em especial criação, atualização,
     confirmação, recebimento, vencimento, estorno e chargeback.

O Asaas envia o token no header `asaas-access-token`. O EngageFit rejeita
requisições sem correspondência exata.

## 2. Configurar o backend

No Railway, configure no serviço `engage-fit-api`:

```env
FEATURE_BILLING_ENABLED=true
ASAAS_BASE_URL=https://api-sandbox.asaas.com/v3
ASAAS_API_KEY=<chave-do-sandbox>
ASAAS_WEBHOOK_TOKEN=<token-aleatorio-com-ao-menos-32-caracteres>
ASAAS_TIMEOUT_SECONDS=15
PRIVACY_RETENTION_BILLING_WEBHOOK_DAYS=365
```

Antes do deploy, aplique a migration `034_create_billing.sql` com:

```bash
/usr/local/bin/engagefit-migrate up
```

O frontend descobre a funcionalidade por `/api/v1/capabilities`; não precisa de
segredo nem variável própria.

## 3. Homologar ponta a ponta

1. Entre como `PLATFORM_ADMIN` e abra `Financeiro`.
2. Confira os planos iniciais e ajuste ou crie uma nova versão.
   Para boleto/Pix no teste, use ao menos R$ 10,00 para evitar rejeição pelos
   limites mínimos do Asaas.
3. Escolha uma academia e sincronize seus dados de cobrança.
4. Crie uma assinatura com vencimento próximo e forma `Cliente escolhe`.
5. Abra o cliente e a assinatura no Asaas sandbox e confirme que
   `externalReference` contém o identificador do EngageFit.
6. Simule o pagamento no sandbox.
7. Confirme no EngageFit:

   - assinatura `active`;
   - cobrança no histórico;
   - acesso liberado;
   - franquias de mensagens iguais ao plano.

8. Simule vencimento e confira o estado `past_due` e a data de tolerância.
9. Execute a conciliação e, após a tolerância, confira `suspended`, revogação de
   sessões e interrupção das automações.
10. Simule estorno/chargeback e confirme bloqueio imediato.
11. Conceda tolerância manual com motivo e confira o registro de auditoria.
12. Cancele uma assinatura de teste e confira cancelamento também no Asaas.

## 4. Conciliação periódica

Webhooks são o caminho principal. A conciliação cobre indisponibilidade,
eventos perdidos e divergências:

```bash
/usr/local/bin/engagefit-billing-reconcile
```

Agende a execução pelo menos uma vez por hora. O comando é idempotente, consulta
as cobranças das assinaturas e aplica bloqueios cuja tolerância terminou. Ele
pode ser executado como job/cron curto usando a mesma imagem e as mesmas
variáveis do backend; não precisa manter outro servidor permanentemente ativo.

Também existe a ação manual `Reconciliar Asaas` no painel administrativo.

## 5. Passagem para produção

Somente depois da homologação:

1. gere uma chave na conta Asaas de produção;
2. cadastre o webhook de produção com token novo;
3. altere:

   ```env
   ASAAS_BASE_URL=https://api.asaas.com/v3
   ASAAS_API_KEY=<chave-de-producao>
   ASAAS_WEBHOOK_TOKEN=<token-de-producao>
   ```

4. faça um piloto com uma única academia;
5. confirme cobrança, pagamento, webhook, conciliação, franquia e acesso;
6. mantenha alerta operacional para falhas do comando e eventos com status
   `failed`.

Não reutilize chaves ou tokens do sandbox em produção. Nunca registre esses
valores em Git, logs, screenshots ou handoffs.

## 6. Rotina administrativa

- Criar cliente e assinatura somente pelo painel EngageFit, evitando estados
  paralelos criados manualmente no Asaas.
- Criar nova versão de plano para mudar preço ou franquia já contratada.
- Usar tolerância manual apenas com justificativa verificável.
- Rodar conciliação após incidentes do Asaas ou do webhook.
- Conferir mensalmente receita recorrente, recebido, pendente e inadimplência.
- Executar a retenção em dry-run e depois conforme a política:

  ```bash
  make privacy-retention-dry-run
  make privacy-retention-apply
  ```

## 7. Diagnóstico

| Sintoma | Verificação |
|---|---|
| Financeiro não aparece | `FEATURE_BILLING_ENABLED=true` e novo deploy |
| Webhook retorna 401 | token do webhook e `ASAAS_WEBHOOK_TOKEN` |
| Webhook retorna 404 | capability desligada ou URL incorreta |
| Cliente não é criado | chave, ambiente/base URL e CPF/CNPJ |
| Assinatura rejeitada por valor | usar plano de ao menos R$ 10,00 no teste com boleto/Pix |
| Pagamento não aparece | webhook, depois conciliação manual |
| Academia continua bloqueada | status da cobrança mais recente e conciliação |
| Assinatura duplicada | repetir com a mesma `Idempotency-Key`; a UI já faz isso |

Falhas do provedor não expõem o corpo retornado pelo Asaas na resposta pública.
Use `request_id`, logs estruturados e o espelho de eventos no PostgreSQL.
Rejeições do Asaas geram o evento `billing_provider_request_failed`, com
`provider`, `operation`, `provider_status`, `provider_error_code` e a primeira
descrição normalizada e limitada. O payload bruto e as credenciais nunca são
registrados. Erros de validação do provedor retornam `422` ao painel com uma
mensagem segura; autenticação, limite de requisições e indisponibilidade usam
códigos públicos distintos.
