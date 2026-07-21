# Runbook operacional da aplicacao

## Diagnostico por request ID

Toda resposta HTTP de erro possui `code`, `message` e `request_id`; o mesmo ID e retornado em `X-Request-ID` e aparece nos logs estruturados. A interface acrescenta `suporte: <request_id>` a mensagem apresentada ao usuario.

1. Copie o ID informado pela interface.
2. Procure nos logs pelo campo exato `request_id`.
3. Use `trace_id` do mesmo log para abrir o trace quando OpenTelemetry estiver habilitado.
4. Consulte somente os campos normalizados: rota, status, operacao e resultado.
5. Nao solicite senha, token, credencial, corpo de mensagem ou planilha completa para diagnostico inicial.

## API/readiness

- `/health/live` falha: processo/container indisponivel; confira reinicios e `shutdown_*`.
- `/health/ready` retorna `503`: PostgreSQL nao respondeu no timeout; confira disponibilidade e pool antes de reiniciar.
- `internal_error`: correlacione pelo request ID; a resposta nunca deve conter SQL ou erro bruto.
- `rate_limit_exceeded`: respeite `Retry-After`; reiniciar a API nao e procedimento operacional.

## Login e sessao

- `invalid_credentials`: confirme e-mail normalizado; nao revele se o usuario existe.
- `session_invalid`: token expirou ou foi revogado por logout/troca de senha/reset administrativo; autentique novamente.
- `csrf_invalid`: confira envio de cookies com credenciais e header `X-CSRF-Token`; nao desabilite CSRF.
- Falha apenas no navegador: confira `CORS_ALLOWED_ORIGINS`, `AUTH_COOKIE_SECURE`, `SameSite` e HTTPS.

## Importacoes

- `upload_too_large`/`request_too_large`: reduza o arquivo; nao aumente limites sem revisar memoria e volume esperado.
- `invalid_request`: confira fonte e formato Wellhub/TotalPass.
- Falha interna: use request ID, historico da importacao e metricas `engagefit_imports_*`; nao registre linhas/PII nos logs.
- Reimportacao com zero check-ins pode ser deduplicacao ou supressao de identidade anonimizada, nao necessariamente falha.

## Automacoes

- Run `failed`: confira modo, contagens e erro normalizado no historico.
- Run `running` alem de `AUTOMATION_STALE_RUN_MINUTES`: revise efeitos externos antes de executar com nova chave; nao repita automaticamente.
- `engagefit_automation_stale_runs_total`: exige revisao operacional.
- Ausencia de execucao: confira flag, agenda, timezone, dias e catch-up window.

## Gateways

- `email_provider_failed`, `llm_generation_failed` e `whatsapp_provider_failed` sao mensagens deliberadamente normalizadas.
- Use trace e metricas `engagefit_gateway_*`; credenciais e resposta bruta nao devem ir para UI/log.
- Development bloqueia efeitos externos sem permissao explicita.
- WhatsApp mede somente aceite/falha sincrona nesta fase; entrega final via StatusCallback esta fora do escopo atual.

## Capacidades

`capability_disabled` e `404` significam que a feature esta desligada. Confira `/api/v1/capabilities` e as variaveis `FEATURE_*`; nao contorne o middleware chamando casos de uso diretamente.

## Privacidade e segredos

- Credencial indecifravel: restaure a chave antiga no keyring e siga a rotacao documentada; nao edite ciphertext.
- Pedido de titular: use exportacao/anonimizacao na tela Alunos e registre motivo.
- Retencao: execute dry-run primeiro e revise contagens antes de `--apply`.
- Suspeita de incidente: preserve evidencias, restrinja acesso e siga `docs/privacy-runbook.md`.
