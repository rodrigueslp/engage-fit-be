# EngageFit — arquitetura de billing

Atualizado em: 2026-07-23

## Objetivo

O billing do EngageFit é um contexto isolado dentro do backend existente. O
Asaas processa dinheiro e mantém os dados do meio de pagamento; o EngageFit
mantém planos, assinaturas, faturas espelhadas, acesso e franquias.

## Fontes de verdade

| Assunto | Fonte de verdade |
|---|---|
| Pagamento confirmado, estorno e chargeback | Asaas |
| Plano contratado | EngageFit |
| Estado comercial derivado | EngageFit, sincronizado com Asaas |
| Acesso da academia | EngageFit |
| Limites e consumo de mensagens | EngageFit |
| Dados de cartão | Asaas; nunca persistidos no EngageFit |

## Fluxo

1. O administrador cadastra os dados de cobrança da academia.
2. O EngageFit cria ou reutiliza o cliente Asaas usando `box_id` como
   `externalReference`.
3. O administrador escolhe plano, vencimento e forma de pagamento.
4. O EngageFit cria a assinatura Asaas e salva o identificador externo.
5. O Asaas gera cobranças e envia eventos para o webhook.
6. O webhook autentica `asaas-access-token`, persiste o evento de forma
   idempotente e aplica a projeção financeira.
7. `PAYMENT_CONFIRMED` ou `PAYMENT_RECEIVED` libera o acesso.
8. `PAYMENT_OVERDUE` inicia a tolerância.
9. A conciliação suspende assinaturas cuja tolerância venceu e corrige eventos
   eventualmente perdidos consultando a API do Asaas.
10. Estorno, chargeback ou cancelamento bloqueiam o acesso conforme a política.

## Separação de estados

O ciclo administrativo de `boxes.status` não é alterado automaticamente pelo
billing. A projeção `boxes.billing_access_blocked` é independente.

O acesso do owner exige simultaneamente:

- academia administrativamente `active`;
- billing não bloqueado.

Uma confirmação de pagamento nunca reativa uma suspensão administrativa.

## Segurança

- chave da API Asaas somente em variável de ambiente;
- webhook protegido por token forte e comparação constante;
- eventos duplicados são aceitos sem reaplicar efeitos;
- payload bruto fica restrito ao banco e sujeito à retenção;
- erros públicos não incluem resposta bruta do Asaas;
- nenhuma informação de cartão entra na API ou no banco do EngageFit;
- chamadas externas possuem timeout;
- auditoria administrativa registra alterações sem segredos.

## Disponibilidade

O webhook confirma recebimento após persistir e processar o evento. Eventos
falhos permanecem registrados para nova tentativa. A conciliação periódica é o
segundo caminho para corrigir divergências.

## Extração futura

O acesso ao Asaas ocorre por `services.BillingGateway` e a persistência por
`repositories.BillingRepository`. Essa fronteira permite trocar o provedor ou
extrair um serviço independente sem contaminar casos de uso de alunos,
campanhas e mensageria.
