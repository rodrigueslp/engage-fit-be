import { existsSync, readFileSync } from 'node:fs';
import { basename } from 'node:path';

const apiBaseUrl = process.env.API_BASE_URL ?? 'http://localhost:8080';
const ownerEmail = process.env.DEMO_OWNER_EMAIL ?? 'owner@example.com';
const ownerPassword = process.env.DEMO_OWNER_PASSWORD ?? 'change-me';
const totalPassFilePath = process.env.DEMO_TOTALPASS_FILE ?? '';
const luizPhone = '5511963834712';
const demoStudents = [
  { name: 'Luiz', email: 'luiz@example.com', phone: luizPhone, checkins: 9, scenario: 'falta 1 check-in' },
  { name: 'Deborah', email: 'deborah@example.com', phone: luizPhone, checkins: 8, scenario: 'faltam 2 check-ins' },
  { name: 'Bruno Teste', email: 'bruno.teste@example.com', phone: luizPhone, checkins: 7, scenario: 'abaixo do corte de falta pouco' },
  { name: 'Carla Teste', email: 'carla.teste@example.com', phone: luizPhone, checkins: 10, scenario: 'meta ja atingida' },
  { name: 'Marina Risco', email: 'marina.risco@example.com', phone: luizPhone, checkins: 3, startOffsetDays: 12, scenario: 'sem check-in ha mais de 7 dias' },
];

const almostThereTemplateContent =
  'Ola, {{name}}! Falta pouco para voce bater a meta do mes no {{box_name}}: voce ja fez {{current_checkins}} de {{target_checkins}} check-ins e faltam {{remaining_checkins}} para garantir {{reward_name}}.';

const achievedTemplateContent =
  'Ola, {{name}}! Parabens, voce atingiu a meta do mes no {{box_name}} com {{current_checkins}} check-ins e garantiu {{reward_name}}.';

const inactiveTemplateContent =
  'Ola, {{name}}! Sentimos sua falta no {{box_name}}. Que tal voltar aos treinos esta semana e seguir firme na sua rotina?';

async function main() {
  await assertApiIsRunning();
  await createOwnerIfNeeded();
  const token = await login();

  const campaign = await createCampaign(token);
  await createGoal(token, campaign.id, 'totalpass', 10);
  await createReward(token, campaign.id);

  await importCheckins(token, 'totalpass', totalpassCsv(), totalPassFilePath);
  await recalculateCampaign(token, campaign.id);

  const almostThereTemplate = await createTemplate(token, 'Falta pouco demo', almostThereTemplateContent);
  const achievedTemplate = await createTemplate(token, 'Meta atingida demo', achievedTemplateContent);
  const inactiveTemplate = await createTemplate(token, 'Aluno em risco demo', inactiveTemplateContent);
  await createMessageCampaign(token, campaign.id, almostThereTemplate.id, 'Disparo teste - falta pouco', 'almost_there');
  await createMessageCampaign(token, campaign.id, achievedTemplate.id, 'Disparo teste - meta atingida', 'achieved');
  await createMessageCampaign(token, campaign.id, inactiveTemplate.id, 'Disparo teste - aluno em risco', 'inactive');

  console.log('');
  console.log('Demo pronto para teste de WhatsApp.');
  console.log(`Login: ${ownerEmail}`);
  console.log(`Senha: ${ownerPassword}`);
  console.log(`Campanha: ${campaign.name}`);
  console.log('Alunos seed TotalPass:');
  for (const student of demoStudents) {
    console.log(`- ${student.name}: ${student.checkins}/10 (${student.scenario}) -> ${student.phone}`);
  }
  console.log('Importe test-data/totalpass-checkins-hit-goal.csv para levar Luiz, Deborah e Bruno a 10/10.');
  console.log('Depois configure Twilio em Configuracoes e envie a campanha na tela WhatsApp.');
}

async function assertApiIsRunning() {
  try {
    const response = await fetch(`${apiBaseUrl}/health`);
    await assertResponse(response, 'verificar healthcheck da API');
  } catch (error) {
    throw new Error(`API indisponivel em ${apiBaseUrl}. Suba o backend antes de rodar o seed demo.`);
  }
}

async function createOwnerIfNeeded() {
  const response = await fetch(`${apiBaseUrl}/api/v1/setup/owner`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      box_name: 'CrossFit Alados',
      owner_name: 'Owner Demo',
      owner_email: ownerEmail,
      password: ownerPassword,
    }),
  });

  if (response.ok || response.status === 400 || response.status === 409 || response.status === 500) {
    return;
  }
  await assertResponse(response, 'criar owner demo');
}

async function login() {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ownerEmail, password: ownerPassword }),
  });
  await assertResponse(response, 'login demo');
  const body = await response.json();
  return body.access_token;
}

async function createCampaign(token) {
  const month = currentMonthRange();
  const response = await authedFetch(token, '/api/v1/campaigns', {
    method: 'POST',
    body: JSON.stringify({
      name: `Brinde do mes ${month.label}`,
      description: 'Campanha demo para validar meta TotalPass e disparo de WhatsApp.',
      start_date: month.start,
      end_date: month.end,
    }),
  });
  await assertResponse(response, 'criar campanha demo');
  return response.json();
}

async function createGoal(token, campaignId, source, targetCheckins) {
  const response = await authedFetch(token, `/api/v1/campaigns/${campaignId}/goals`, {
    method: 'POST',
    body: JSON.stringify({ source, target_checkins: targetCheckins }),
  });
  await assertResponse(response, `criar meta ${source}`);
}

async function createReward(token, campaignId) {
  const response = await authedFetch(token, `/api/v1/campaigns/${campaignId}/rewards`, {
    method: 'POST',
    body: JSON.stringify({
      name: 'Camiseta exclusiva Alados',
      description: 'Brinde demo para alunos que atingirem a meta do mes.',
      quantity: 100,
    }),
  });
  await assertResponse(response, 'criar brinde demo');
}

async function importCheckins(token, source, csv, filePath) {
  const formData = new FormData();
  formData.append('source', source);
  if (filePath && existsSync(filePath)) {
    formData.append('file', new Blob([readFileSync(filePath)]), basename(filePath));
  } else {
    formData.append('file', new Blob([csv], { type: 'text/csv' }), `${source}-demo.csv`);
  }

  const response = await fetch(`${apiBaseUrl}/api/v1/imports`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: formData,
  });
  await assertResponse(response, `importar check-ins ${source}`);
}

async function recalculateCampaign(token, campaignId) {
  const response = await authedFetch(token, `/api/v1/campaigns/${campaignId}/recalculate-progress`, {
    method: 'POST',
  });
  await assertResponse(response, 'recalcular campanha demo');
}

async function createTemplate(token, name, content) {
  const response = await authedFetch(token, '/api/v1/message-templates', {
    method: 'POST',
    body: JSON.stringify({
      name,
      content,
    }),
  });
  await assertResponse(response, 'criar template demo');
  return response.json();
}

async function createMessageCampaign(token, campaignId, templateId, name, audience) {
  const response = await authedFetch(token, '/api/v1/message-campaigns', {
    method: 'POST',
    body: JSON.stringify({
      name,
      campaign_id: campaignId,
      audience,
      template_id: templateId,
    }),
  });
  await assertResponse(response, 'criar campanha de mensagem demo');
  return response.json();
}

function totalpassCsv() {
  return buildCsv(demoStudents.flatMap((student) => studentRows(student.name, student.email, student.phone, student.checkins, student.startOffsetDays)));
}

function studentRows(name, email, phone, amount, startOffsetDays = 1) {
  const dates = recentDates(amount, startOffsetDays);
  return dates.map((date, index) => ({ nome: name, email, telefone: phone, data: date, hora: `${String(7 + (index % 3)).padStart(2, '0')}:30` }));
}

function buildCsv(rows) {
  const lines = ['nome,email,telefone,data,hora'];
  for (const row of rows) {
    lines.push([row.nome, row.email, row.telefone, row.data, row.hora].join(','));
  }
  return `${lines.join('\n')}\n`;
}

function recentDates(amount, startOffsetDays = 1) {
  const result = [];
  const today = new Date();
  for (let i = startOffsetDays; i < startOffsetDays + amount; i += 1) {
    const date = new Date(today);
    date.setDate(today.getDate() - i);
    result.push(formatDate(date));
  }
  return result;
}

function currentMonthRange() {
  const today = new Date();
  const start = new Date(today.getFullYear(), today.getMonth(), 1);
  const end = new Date(today.getFullYear(), today.getMonth() + 1, 0);
  return {
    start: formatDate(start),
    end: formatDate(end),
    label: `${String(today.getMonth() + 1).padStart(2, '0')}/${today.getFullYear()}`,
  };
}

function formatDate(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

async function authedFetch(token, path, options = {}) {
  return fetch(`${apiBaseUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...(options.headers ?? {}),
    },
  });
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
