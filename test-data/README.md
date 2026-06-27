# Dados de teste

## WhatsApp - falta pouco e meta atingida

1. Rode `make demo-reset-seed`.
2. O seed cria uma campanha TotalPass com meta `10` e cinco alunos usando o telefone do Luiz (`5511963834712`):
   - Luiz: `9/10`, entra na audiencia `Falta pouco`.
   - Deborah: `8/10`, entra na audiencia `Falta pouco` se ainda houver pelo menos 2 dias restantes na campanha.
   - Bruno Teste: `7/10`, fica fora de `Falta pouco` por estar abaixo de 80%.
   - Carla Teste: `10/10`, entra na audiencia `Meta atingida`.
   - Marina Risco: `3/10`, entra na audiencia `Aluno em risco` por estar ha mais de 7 dias sem check-in.
3. O seed tambem cria tres campanhas de mensagem na tela `WhatsApp`:
   - `Disparo teste - falta pouco`, audiencia `almost_there`.
   - `Disparo teste - meta atingida`, audiencia `achieved`.
   - `Disparo teste - aluno em risco`, audiencia `inactive`.
4. Ao enviar `Disparo teste - aluno em risco`, o aluno abordado fica com status `Em observacao` e nao recebe nova mensagem de risco por 14 dias.
5. Importe `test-data/totalpass-checkins-hit-goal.csv` ou `test-data/totalpass-checkins-23-06-2026.xlsx` pela tela de importacoes, selecionando a fonte `TotalPass`.
6. Apos a importacao, Luiz, Deborah e Bruno ficam com `10/10` check-ins e entram na audiencia `Meta atingida`.
7. Configure o Twilio em `Configuracoes` e envie a campanha desejada na tela `WhatsApp`.

Para envio real local apenas nesses dois telefones, use no `.env`:

```env
WHATSAPP_ALLOW_REAL_SEND=true
WHATSAPP_DEV_ALLOWED_RECIPIENT_PHONES=5511963834712
```
