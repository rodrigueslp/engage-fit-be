const baseURL = process.env.API_BASE_URL || 'http://localhost:18080';
const suffix = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
const password = 'Smoke-test-password-2026!';

async function request(path, options = {}, expectedStatus = 200) {
  const response = await fetch(`${baseURL}${path}`, {
    ...options,
    headers: {
      ...(options.body && !(options.body instanceof FormData) ? { 'content-type': 'application/json' } : {}),
      ...options.headers,
    },
  });
  const text = await response.text();
  if (response.status !== expectedStatus) {
    throw new Error(`${options.method || 'GET'} ${path}: expected ${expectedStatus}, got ${response.status}: ${text}`);
  }
  return text ? JSON.parse(text) : undefined;
}

function authenticated(token, method = 'GET', body) {
  return {
    method,
    headers: { authorization: `Bearer ${token}` },
    ...(body === undefined ? {} : { body: body instanceof FormData ? body : JSON.stringify(body) }),
  };
}

async function createTenant(label) {
  const email = `smoke-${label}-${suffix}@example.test`;
  await request('/api/v1/setup/owner', {
    method: 'POST',
    body: JSON.stringify({
      box_name: `Smoke ${label}`,
      owner_name: `Owner ${label}`,
      owner_email: email,
      password,
    }),
  }, 201);
  const login = await request('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  if (!login.access_token) throw new Error(`login for tenant ${label} returned no token`);
  return { email, token: login.access_token };
}

async function browserSessionSmoke() {
  const email = `smoke-browser-${suffix}@example.test`;
  await request('/api/v1/setup/owner', {
    method: 'POST',
    body: JSON.stringify({ box_name: 'Smoke Browser', owner_name: 'Browser Owner', owner_email: email, password }),
  }, 201);
  const loginResponse = await fetch(`${baseURL}/api/v1/auth/login`, {
    method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ email, password }),
  });
  if (loginResponse.status !== 200) throw new Error(`browser login failed: ${loginResponse.status}`);
  const setCookies = loginResponse.headers.getSetCookie?.() || [];
  const sessionPart = setCookies.find((value) => value.startsWith('engagefit_session='));
  const csrfPart = setCookies.find((value) => value.startsWith('engagefit_session_csrf='));
  if (!sessionPart?.includes('HttpOnly') || !sessionPart.includes('SameSite=Lax')) throw new Error('secure session cookie attributes are missing');
  if (!csrfPart || csrfPart.includes('HttpOnly')) throw new Error('csrf cookie is missing or incorrectly HttpOnly');
  const sessionCookie = sessionPart.split(';', 1)[0];
  const csrfCookie = csrfPart.split(';', 1)[0];
  const csrfToken = csrfCookie.slice(csrfCookie.indexOf('=') + 1);
  const cookie = `${sessionCookie}; ${csrfCookie}`;
  await request('/api/v1/auth/me', { headers: { cookie } });
  await request('/api/v1/auth/logout', { method: 'POST', headers: { cookie } }, 403);
  await request('/api/v1/auth/logout', { method: 'POST', headers: { cookie, 'x-csrf-token': csrfToken } }, 204);
  await request('/api/v1/auth/me', { headers: { cookie } }, 401);
}

await request('/health/live');
await request('/health/ready');

const tenantA = await createTenant('A');
const tenantB = await createTenant('B');
const meA = await request('/api/v1/auth/me', authenticated(tenantA.token));
const meB = await request('/api/v1/auth/me', authenticated(tenantB.token));
if (meA.box_id === meB.box_id) throw new Error('smoke tenants unexpectedly share a box');

async function importSingleStudent() {
  const form = new FormData();
  form.set('source', 'totalpass');
  form.set('file', new Blob(['nome,email,telefone,data,hora\nPrivacy Smoke,privacy-smoke@example.test,+5511999999999,2026-07-20,08:30\n'], { type: 'text/csv' }), 'smoke.csv');
  return request('/api/v1/imports', authenticated(tenantB.token, 'POST', form), 201);
}

const firstImport = await importSingleStudent();
if (firstImport.students !== 1 || firstImport.checkins !== 1) throw new Error(`unexpected first import: ${JSON.stringify(firstImport)}`);

const campaignB = await request('/api/v1/campaigns', authenticated(tenantB.token, 'POST', {
  name: 'Campaign B',
  description: 'Tenant isolation smoke test',
  start_date: '2026-01-01',
  end_date: '2026-12-31',
}), 201);
const goalB = await request(`/api/v1/campaigns/${campaignB.id}/goals`, authenticated(tenantB.token, 'POST', {
  source: 'totalpass',
  target_checkins: 10,
}));
const rewardB = await request(`/api/v1/campaigns/${campaignB.id}/rewards`, authenticated(tenantB.token, 'POST', {
  name: 'Reward B',
  description: 'Private to tenant B',
  quantity: 5,
}), 201);

for (const [path, options] of [
  [`/api/v1/campaigns/${campaignB.id}`, authenticated(tenantA.token)],
  [`/api/v1/campaigns/${campaignB.id}/goals`, authenticated(tenantA.token)],
  [`/api/v1/campaigns/${campaignB.id}/progress`, authenticated(tenantA.token)],
  [`/api/v1/campaigns/${campaignB.id}/rewards`, authenticated(tenantA.token)],
  [`/api/v1/campaigns/${campaignB.id}/goals`, authenticated(tenantA.token, 'POST', { source: 'gympass', target_checkins: 1 })],
  [`/api/v1/campaigns/${campaignB.id}/rewards`, authenticated(tenantA.token, 'POST', { name: 'Intrusion', description: '', quantity: 1 })],
  [`/api/v1/rewards/${rewardB.id}`, authenticated(tenantA.token, 'PUT', { name: 'Changed', description: '', quantity: 99 })],
  [`/api/v1/rewards/${rewardB.id}`, authenticated(tenantA.token, 'DELETE')],
]) {
  await request(path, options, 404);
}

const goalsB = await request(`/api/v1/campaigns/${campaignB.id}/goals`, authenticated(tenantB.token));
const rewardsB = await request(`/api/v1/campaigns/${campaignB.id}/rewards`, authenticated(tenantB.token));
if (goalsB.length !== 1 || goalsB[0].id !== goalB.id) throw new Error('tenant B goal was changed by cross-tenant requests');
if (rewardsB.length !== 1 || rewardsB[0].name !== 'Reward B') throw new Error('tenant B reward was changed by cross-tenant requests');

await request(`/api/v1/campaigns/${campaignB.id}/recalculate-progress`, authenticated(tenantB.token, 'POST'), 204);
const summary = await request('/api/v1/dashboard/summary', authenticated(tenantB.token));
if (summary.total_students !== 1 || summary.total_checkins !== 1) throw new Error(`dashboard summary has an invalid contract: ${JSON.stringify(summary)}`);

const studentsB = await request('/api/v1/students', authenticated(tenantB.token));
if (studentsB.length !== 1) throw new Error('imported student was not listed');
const studentB = studentsB[0];
await request(`/api/v1/students/${studentB.id}/privacy-export`, authenticated(tenantA.token), 404);
await request(`/api/v1/students/${studentB.id}/contact-preference`, authenticated(tenantA.token, 'PATCH', { status: 'opted_out', source: 'cross_tenant' }), 404);
await request(`/api/v1/students/${studentB.id}/anonymize`, authenticated(tenantA.token, 'POST', { confirmed: true, reason: 'cross tenant attempt' }), 404);
await request(`/api/v1/students/${studentB.id}/contact-preference`, authenticated(tenantB.token, 'PATCH', { status: 'opted_out', source: 'smoke_test' }), 204);
const privacyExport = await request(`/api/v1/students/${studentB.id}/privacy-export`, authenticated(tenantB.token));
if (privacyExport.student.email !== 'privacy-smoke@example.test' || privacyExport.checkins.length !== 1) throw new Error('privacy export is incomplete');
await request(`/api/v1/students/${studentB.id}/anonymize`, authenticated(tenantB.token, 'POST', { confirmed: true, reason: 'smoke erasure request' }), 204);
const replayImport = await importSingleStudent();
if ((replayImport.students ?? 0) !== 0 || (replayImport.checkins ?? 0) !== 0) throw new Error(`anonymized identity was reimported: ${JSON.stringify(replayImport)}`);
const anonymizedStudents = await request('/api/v1/students', authenticated(tenantB.token));
if (anonymizedStudents.length !== 1 || anonymizedStudents[0].email !== '' || !anonymizedStudents[0].anonymized_at) throw new Error('student was not anonymized');

await request('/api/v1/auth/logout', authenticated(tenantA.token, 'POST'), 204);
await request('/api/v1/auth/me', authenticated(tenantA.token), 401);
await browserSessionSmoke();

console.log('API smoke test passed: health, setup, bearer and cookie auth, CSRF, import, tenant isolation, campaign, reward, dashboard, privacy, and logout.');
