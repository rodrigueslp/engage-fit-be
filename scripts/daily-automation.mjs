import { existsSync, readFileSync } from 'node:fs';
import { basename } from 'node:path';

const apiBaseUrl = process.env.API_BASE_URL ?? 'http://localhost:8080';
const ownerEmail = process.env.DEMO_OWNER_EMAIL ?? 'owner@example.com';
const ownerPassword = process.env.DEMO_OWNER_PASSWORD ?? 'change-me';
const checkinsFile = process.env.DAILY_CHECKINS_FILE ?? '';
const checkinsSource = process.env.DAILY_CHECKINS_SOURCE ?? 'totalpass';
const sendMessages = process.env.DAILY_SEND_MESSAGES === 'true';
const resendMessages = process.env.DAILY_RESEND_MESSAGES === 'true';
const automationTimezone = process.env.DAILY_AUTOMATION_TIMEZONE ?? 'America/Sao_Paulo';
const automationIdempotencyKey = process.env.DAILY_AUTOMATION_IDEMPOTENCY_KEY ?? `daily:${dateInTimezone(new Date(), automationTimezone)}`;

async function main() {
  await assertApiIsRunning();
  const token = await login();
  let run;

  const audit = {
    imported: false,
    recalculated: 0,
    skippedMessageCampaigns: 0,
    sentMessageCampaigns: 0,
    failedMessageCampaigns: 0,
  };

  try {
    run = await createAutomationRun(token);
    if (run.idempotent_replay) {
      console.log(`Automacao ${automationIdempotencyKey} ja foi iniciada anteriormente; nenhuma etapa foi repetida.`);
      return;
    }

    if (checkinsFile) {
      await importDailyCheckins(token);
      audit.imported = true;
    }

    const activeCampaigns = (await authedJson(token, '/api/v1/campaigns')).filter((campaign) => campaign.active);
    for (const campaign of activeCampaigns) {
      await authedFetch(token, `/api/v1/campaigns/${campaign.id}/recalculate-progress`, { method: 'POST' });
      audit.recalculated += 1;
    }

    if (sendMessages) {
      const activeCampaignIDs = new Set(activeCampaigns.map((campaign) => campaign.id));
      const messageCampaigns = await authedJson(token, '/api/v1/message-campaigns');
      for (const messageCampaign of messageCampaigns) {
        if (!activeCampaignIDs.has(messageCampaign.campaign_id) || (messageCampaign.sent_at && !resendMessages)) {
          audit.skippedMessageCampaigns += 1;
          continue;
        }

        const response = await authedFetch(token, `/api/v1/message-campaigns/${messageCampaign.id}/send`, { method: 'POST' });
        const result = await response.json();
        audit.sentMessageCampaigns += result.sent ?? 0;
        audit.failedMessageCampaigns += result.failed ?? 0;
      }
    }

    if (run?.id) {
      await finishAutomationRun(token, run.id, 'success', audit);
    }
    printAudit(audit);
  } catch (error) {
    if (run?.id) {
      await finishAutomationRun(token, run.id, 'failed', audit, error.message);
    }
    throw error;
  }
}

async function createAutomationRun(token) {
  return authedJson(token, '/api/v1/automation/runs', {
    method: 'POST',
    headers: { 'Idempotency-Key': automationIdempotencyKey },
    body: JSON.stringify({
      source: checkinsFile ? checkinsSource : '',
      filename: checkinsFile ? basename(checkinsFile) : '',
    }),
  });
}

function dateInTimezone(date, timezone) {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(date);
  const value = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${value.year}-${value.month}-${value.day}`;
}

async function finishAutomationRun(token, runId, status, audit, errorMessage = '') {
  await authedFetch(token, `/api/v1/automation/runs/${runId}`, {
    method: 'PATCH',
    body: JSON.stringify({
      status,
      imported: audit.imported,
      recalculated_campaigns: audit.recalculated,
      skipped_message_campaigns: audit.skippedMessageCampaigns,
      sent_messages: audit.sentMessageCampaigns,
      failed_messages: audit.failedMessageCampaigns,
      error_message: errorMessage,
    }),
  });
}

function printAudit(audit) {
  console.log('Automacao diaria concluida.');
  console.log(`Importacao: ${audit.imported ? `${checkinsSource} (${basename(checkinsFile)})` : 'nenhum arquivo configurado'}`);
  console.log(`Campanhas recalculadas: ${audit.recalculated}`);
  console.log(`Mensagens enviadas: ${audit.sentMessageCampaigns}`);
  console.log(`Falhas de mensagem: ${audit.failedMessageCampaigns}`);
  console.log(`Campanhas de mensagem ignoradas: ${audit.skippedMessageCampaigns}`);
}

async function assertApiIsRunning() {
  try {
    const response = await fetch(`${apiBaseUrl}/health`);
    await assertResponse(response, 'verificar healthcheck da API');
  } catch (error) {
    throw new Error(`API indisponivel em ${apiBaseUrl}. Suba o backend antes de rodar a automacao diaria.`);
  }
}

async function login() {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ownerEmail, password: ownerPassword }),
  });
  await assertResponse(response, 'login da automacao diaria');
  const body = await response.json();
  return body.access_token;
}

async function importDailyCheckins(token) {
  if (!existsSync(checkinsFile)) {
    throw new Error(`Arquivo DAILY_CHECKINS_FILE nao encontrado: ${checkinsFile}`);
  }

  const formData = new FormData();
  formData.append('source', checkinsSource);
  formData.append('file', new Blob([readFileSync(checkinsFile)]), basename(checkinsFile));

  const response = await fetch(`${apiBaseUrl}/api/v1/imports`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: formData,
  });
  await assertResponse(response, 'importar check-ins diarios');
}

async function authedJson(token, path, options = {}) {
  const response = await authedFetch(token, path, options);
  return response.json();
}

async function authedFetch(token, path, options = {}) {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...(options.headers ?? {}),
    },
  });
  await assertResponse(response, path);
  return response;
}

async function assertResponse(response, action) {
  if (response.ok) return;
  const body = await response.text();
  throw new Error(`Falha ao ${action}: HTTP ${response.status} ${body}`);
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
