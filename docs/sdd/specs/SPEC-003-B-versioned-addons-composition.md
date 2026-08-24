# SPEC-003/B — Adicionais versionados e composição comercial

- Status: `validated locally`
- Versão: `1.0`
- Data: 2026-08-21
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-002/A`, `SPEC-003/A`, `SPEC-003/C`
- Bloqueia: contratos (`SPEC-004`), cobrança do aluno (`SPEC-005`), agenda (`SPEC-006`), acesso (`SPEC-003/D` e `SPEC-008`)
- Ownership principal: módulo `memberships`
- Repositórios envolvidos: `engage-fit-be` e `engage-fit-fe`

---

## Estratégia de Fatiamento e Execução

Para manter as branches pequenas, revisáveis e integráveis com segurança, a iniciativa `SPEC-003/B` é dividida em duas sub-fatias verticais sequenciais:

```text
SPEC-003/B1: Catálogo de adicionais na versão do plano
  ├── Backend: schema, domínio, imutabilidade, clonagem, DTOs e endpoints de planos
  └── Frontend: gerenciamento de adicionais no cadastro/edição de rascunho de planos
       │
       ▼
SPEC-003/B2: Seleção de adicionais e composição na matrícula
  ├── Backend: schema de matrícula, validação de versão, cálculo do total e snapshot imutável
  └── Frontend: seleção de adicionais no painel de matrícula com recálculo em tempo real e aviso aos parceiros
```

- **`SPEC-003/B1`**: Define e persiste adicionais como parte da versão do plano. Cada adicional possui nome, preço incremental não negativo (`price_delta_cents >= 0`), descrição e direitos próprios sobre modalidades ou serviços ofertáveis. Ao publicar o plano, seus adicionais tornam-se imutáveis. Ao criar uma nova versão, os adicionais são clonados para o novo rascunho.
- **`SPEC-003/B2`**: Permite selecionar adicionais opcionais no momento da matrícula do aluno. Valida que os adicionais pertencem estritamente à versão publicada informada, grava um snapshot imutável em `enrollment_addons`, calcula deterministicamente o preço total (`total = preço_base + soma(adicionais)`), exibe o detalhamento comercial no painel de matrícula e preserva com clareza o aviso de "Plano próprio da academia" para alunos parceiros (Wellhub e TotalPass).

---

## 1. Resultado Esperado

1. **Catálogo flexível**: O owner pode criar planos com ofertas complementares (por exemplo: plano base `CrossFit Livre R$ 189/mês` com adicional opcional `Recovery 2x/sem + R$ 50/mês` ou `Acesso Spinning Incluso + R$ 0/mês`).
2. **Matrícula transparente**: Ao matricular um aluno, o owner visualiza os adicionais disponíveis para o plano selecionado, marca os adicionais desejados via checkboxes, e confere a decomposição clara: *Preço base + Adicionais = Preço Total*.
3. **Imutabilidade e auditoria**: A versão exata dos adicionais e os termos contratados ficam congelados na matrícula. Alterações futuras no catálogo de planos não mudam retroativamente o valor ou os direitos de matrículas já criadas.
4. **Isolamento de efeitos**: A composição de adicionais é estritamente comercial nesta etapa; não gera cobrança automática (Asaas), documento jurídico, reserva ou consumo de check-in.
5. **Clareza para parceiros**: Alunos originados de Wellhub e TotalPass mantêm o aviso explícito de que a matrícula e seus adicionais são da academia e não alteram o vínculo com o parceiro externo.

---

## 2. Problema e Evidências

- **Existente:** `SPEC-003/A` entregou o catálogo de planos com direitos base imutáveis. `SPEC-003/C` entregou a matrícula operacional vinculando a pessoa à versão exata do plano.
- **Limitação atual:** A matrícula atual suporta apenas o preço e os direitos base do plano. Não é possível representar ofertas combinadas comuns em boxes (ex.: plano com serviço de Recovery, musculação complementar ou modalidades complementares) sem criar dezenas de planos combinados duplicados.
- **Benchmark — Next Fit e Tecnofit:** Sistemas legados tratam adicionais ou como itens de contrato ou como serviços recorrentes associados à venda. Frequentemente permitem edição retroativa ou desvinculada de versão, gerando inconsistências contábeis e operacionais.
- **Diretriz EngageFit:** O EngageFit trata o adicional como parte da fotografia versionada do plano. O adicional não é uma entidade flutuante global nem sofre alteração retroativa após a venda. O preço total é a soma determinística do plano base com os adicionais selecionados.
- **Entrada Alados (`REQ-PLAN-03`):** Box Alados demanda planos de corrida ou CrossFit com adicionais opcionais de Recovery e modalidades complementares, mantendo controle simples no celular.

---

## 3. Escopo

### 3.1 Incluído

#### Em `SPEC-003/B1` (Catálogo de adicionais):
- Estrutura de adicionais pertencentes estritamente a `membership_plan_versions`;
- Cada adicional possui: `name`, `description`, `price_delta_cents` (inteiro não negativo, `>= 0`), `position` e lista de direitos;
- Direitos do adicional com alvo único em modalidade (`modality_id`) ou serviço ofertável (`service_offering_id`) ativo na organização e habilitado na unidade;
- Tipos de acesso nos direitos do adicional: `open_access`, `sessions_per_period` e `class_pack`, com limite diário, modo de atendimento (`group`/`individual`) e flag `booking_required`;
- Validação no domínio e no banco impedindo adicionais sem direitos ou com valores negativos;
- Suporte a `price_delta_cents = 0` (adicional gratuito/incluso como serviço opcional);
- Edição de adicionais dentro do rascunho da versão do plano (`POST /membership-plans`, `PATCH /membership-plans/:id/draft`);
- Imutabilidade no PostgreSQL: triggers impedem `INSERT`, `UPDATE` ou `DELETE` em adicionais e direitos de versões `published` ou `retired`;
- Clonagem automática de todos os adicionais e seus direitos ao criar nova versão (`POST /membership-plans/:id/versions`);
- Resumo em linguagem natural gerado deterministicamente para cada adicional;
- Interface mobile-first na página de Planos para gerenciar adicionais dentro do formulário de rascunho de plano.

#### Em `SPEC-003/B2` (Composição na matrícula):
- Campo opcional `addon_ids: []UUID` no comando de criação de matrícula (`POST /units/:unit_id/students/:student_id/enrollments`);
- Validação estrita de que todos os `addon_ids` pertencem à versão informada (`plan_version_id`), rejeitando com erro seguro qualquer ID desconhecido ou de outra versão/tenant;
- Gravação atômica dos adicionais contratados em `enrollment_addons`, com snapshot de nome, preço incremental e resumo dos direitos;
- Cálculo do valor total: `base_price_cents`, `addons_price_cents` e `total_price_cents`;
- Resposta de detalhamento e listagem de matrícula enriquecida com os adicionais contratados e valores decompostos;
- Auditoria transacional em `enrollment_events` registrando os adicionais escolhidos no snapshot do evento `enrollment.created`;
- Exportação de privacidade (LGPD) incluindo os adicionais contratados sem duplicar dados pessoais;
- Interface mobile-first no `EnrollmentPanel`: seleção de adicionais via checkboxes com visualização de valor e direitos, recálculo dinâmico da soma na revisão, e manutenção obrigatória do aviso *"Plano próprio da academia"* para alunos Wellhub/TotalPass.

### 3.2 Fora do Escopo

- Desconto comercial, cupom, bolsa ou valor incremental negativo;
- Alteração, remoção ou acréscimo de adicionais no meio de uma matrícula vigente (alterações exigem encerramento e nova matrícula);
- Contrato impresso ou digital, assinatura e termos jurídicos (`SPEC-004`);
- Cobrança real, emissão de boletos, Pix, cartão, recorrência Asaas ou conciliação financeira (`SPEC-005`);
- Grade de horários, vagas de turmas e controle de capacidade (`SPEC-006`);
- Reservas, agendamento de sessões de Recovery ou lista de espera (`SPEC-007`);
- Validação e bloqueio de catraca / check-in com base no saldo de adicionais (`SPEC-003/D` e `SPEC-008`);
- Catálogo de produtos de quiosque ou vendas avulsas de balcão (`SPEC-013`);
- Matrícula self-service no aplicativo do aluno (`SPEC-010`).

---

## 4. Vocabulário

- **Adicional da Versão (`Plan Version Addon`):** Oferta opcional vinculada a uma versão específica de plano, com preço incremental e direitos próprios.
- **Preço Incremental (`price_delta_cents`):** Valor adicionado ao preço base do plano em centavos inteiros de BRL. Deve ser maior ou igual a zero (`>= 0`).
- **Composição Comercial:** Conjunto formado pelo plano base e zero ou mais adicionais selecionados no momento da matrícula.
- **Preço Total (`total_price_cents`):** Soma exata de `base_price_cents + sum(selected_addons.price_delta_cents)`.
- **Adicional Contratado (`Enrollment Addon`):** Snapshot imutável do adicional gravado no momento da matrícula, preservando o valor e os termos vigentes no ato da adesão.

---

## 5. Regras e Invariantes

1. **Pertencimento estrito à versão:** Todo adicional pertence a exatamente uma `membership_plan_versions`. Não existem adicionais soltos ou globais compartilhados entre planos independentes.
2. **Preço não negativo:** `price_delta_cents >= 0`. Preço zero (`0`) é permitido para representar adicionais inclusos ou cortesias de catálogo; valores negativos são estritamente proibidos e rejeitados no schema e no domínio.
3. **Direitos obrigatórios:** Todo adicional deve conter ao menos um direito sobre modalidade ou serviço ofertável.
4. **Alvos ativos:** Cada direito de adicional referencia exatamente uma modalidade ativa ou serviço ofertável ativo e habilitado na unidade da organização.
5. **Imutabilidade pós-publicação:** Quando uma versão de plano é publicada (`status = 'published'`), todos os seus adicionais e respectivos direitos tornam-se imutáveis no banco de dados via trigger.
6. **Clonagem em nova versão:** Criar uma nova versão de plano clona todos os adicionais e direitos da versão anterior para a nova versão em rascunho com novos UUIDs.
7. **Validação de versão na matrícula:** A criação de matrícula só aceita adicionais pertencentes à `plan_version_id` informada. Se um `addon_id` for inválido, inexistente, de outro plano ou de outra versão, a requisição é rejeitada com `422 membership_plan_addon_not_found`.
8. **Unicidade na seleção:** O cliente não pode enviar o mesmo `addon_id` duplicado no payload da matrícula; duplicidades retornam `422 membership_plan_addon_duplicate`.
9. **Snapshot imutável na matrícula:** Uma vez criada a matrícula, os adicionais contratados são gravados em `enrollment_addons` com `addon_name`, `price_delta_cents` e `entitlement_summary`. Alterações no plano nunca afetam matrículas já existentes.
10. **Adesão atômica:** A matrícula e seus adicionais são persistidos na mesma transação de banco de dados. Se a validação de um adicional falhar, toda a operação sofre rollback.
11. **Fórmula de cálculo:** `total_price_cents = base_price_cents + addons_price_cents`. Esse cálculo é determinístico, sem arredondamentos ou casas decimais flutuantes.
12. **Origens externas:** Alunos com origem `wellhub` ou `totalpass` podem receber matrícula com adicionais; o aviso "Plano próprio da academia" deve permanecer visível durante todo o processo.

---

## 6. Modelo de Dados e Esquema PostgreSQL

### 6.1 Tabelas de `SPEC-003/B1` (Catálogo)

```sql
-- Adicionais pertencentes à versão do plano
CREATE TABLE membership_plan_version_addons (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    plan_version_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    price_delta_cents INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_plan_version_addons_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT membership_plan_version_addons_version_org_fk
        FOREIGN KEY (plan_version_id, organization_id)
        REFERENCES membership_plan_versions(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_plan_version_addons_version_name_key
        UNIQUE (plan_version_id, name),
    CONSTRAINT membership_plan_version_addons_name_check
        CHECK (char_length(name) BETWEEN 1 AND 100),
    CONSTRAINT membership_plan_version_addons_description_check
        CHECK (char_length(description) <= 500),
    CONSTRAINT membership_plan_version_addons_price_delta_check
        CHECK (price_delta_cents >= 0),
    CONSTRAINT membership_plan_version_addons_position_check
        CHECK (position >= 0)
);

CREATE INDEX idx_plan_version_addons_version_pos
    ON membership_plan_version_addons(plan_version_id, position);

-- Direitos específicos do adicional
CREATE TABLE membership_plan_version_addon_entitlements (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    addon_id UUID NOT NULL,
    modality_id UUID,
    service_offering_id UUID,
    access_type VARCHAR(32) NOT NULL,
    quantity INTEGER,
    period_unit VARCHAR(8),
    daily_limit INTEGER,
    delivery_mode VARCHAR(16),
    booking_required BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT membership_addon_entitlements_addon_org_fk
        FOREIGN KEY (addon_id, organization_id)
        REFERENCES membership_plan_version_addons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_modality_org_fk
        FOREIGN KEY (modality_id, organization_id)
        REFERENCES modalities(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_service_org_fk
        FOREIGN KEY (service_offering_id, organization_id)
        REFERENCES service_offerings(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT membership_addon_entitlements_target_check CHECK (
        (modality_id IS NOT NULL)::integer + (service_offering_id IS NOT NULL)::integer = 1
    ),
    CONSTRAINT membership_addon_entitlements_access_type_check
        CHECK (access_type IN ('open_access', 'sessions_per_period', 'class_pack')),
    CONSTRAINT membership_addon_entitlements_configuration_check CHECK (
        (access_type = 'open_access' AND quantity IS NULL AND period_unit IS NULL) OR
        (access_type = 'sessions_per_period' AND quantity BETWEEN 1 AND 10000 AND period_unit IN ('day', 'week', 'month')) OR
        (access_type = 'class_pack' AND quantity BETWEEN 1 AND 10000 AND period_unit IS NULL)
    ),
    CONSTRAINT membership_addon_entitlements_daily_limit_check
        CHECK (daily_limit IS NULL OR daily_limit BETWEEN 1 AND 10000),
    CONSTRAINT membership_addon_entitlements_delivery_mode_check
        CHECK (delivery_mode IS NULL OR delivery_mode IN ('group', 'individual')),
    CONSTRAINT membership_addon_entitlements_position_check
        CHECK (position >= 0),
    CONSTRAINT membership_addon_entitlements_target_unique
        UNIQUE NULLS NOT DISTINCT (addon_id, modality_id, service_offering_id)
);

CREATE INDEX idx_plan_addon_entitlements_addon_pos
    ON membership_plan_version_addon_entitlements(addon_id, position);

-- Triggers de Imutabilidade para Adicionais
CREATE OR REPLACE FUNCTION prevent_published_plan_addon_mutation()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(16);
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        SELECT status INTO v_status
        FROM membership_plan_versions
        WHERE id = OLD.plan_version_id;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        SELECT status INTO v_status
        FROM membership_plan_versions
        WHERE id = NEW.plan_version_id;
    END IF;

    IF v_status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan version addons are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_membership_plan_version_addons
BEFORE INSERT OR UPDATE OR DELETE ON membership_plan_version_addons
FOR EACH ROW EXECUTE FUNCTION prevent_published_plan_addon_mutation();

CREATE OR REPLACE FUNCTION prevent_published_plan_addon_entitlement_mutation()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(16);
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        SELECT v.status INTO v_status
        FROM membership_plan_version_addons a
        JOIN membership_plan_versions v ON v.id = a.plan_version_id
        WHERE a.id = OLD.addon_id;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        SELECT v.status INTO v_status
        FROM membership_plan_version_addons a
        JOIN membership_plan_versions v ON v.id = a.plan_version_id
        WHERE a.id = NEW.addon_id;
    END IF;

    IF v_status IN ('published', 'retired') THEN
        RAISE EXCEPTION 'published membership plan addon entitlements are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_membership_plan_addon_entitlements
BEFORE INSERT OR UPDATE OR DELETE ON membership_plan_version_addon_entitlements
FOR EACH ROW EXECUTE FUNCTION prevent_published_plan_addon_entitlement_mutation();
```

### 6.2 Tabelas de `SPEC-003/B2` (Matrícula)

```sql
-- Snapshot imutável de adicionais contratados na matrícula
CREATE TABLE enrollment_addons (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    plan_version_addon_id UUID NOT NULL,
    addon_name VARCHAR(100) NOT NULL,
    price_delta_cents INTEGER NOT NULL,
    entitlement_summary VARCHAR(240) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT enrollment_addons_id_org_key UNIQUE (id, organization_id),
    CONSTRAINT enrollment_addons_enrollment_org_fk
        FOREIGN KEY (enrollment_id, organization_id)
        REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_addons_addon_org_fk
        FOREIGN KEY (plan_version_addon_id, organization_id)
        REFERENCES membership_plan_version_addons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT enrollment_addons_enrollment_addon_key
        UNIQUE (enrollment_id, plan_version_addon_id),
    CONSTRAINT enrollment_addons_name_check
        CHECK (char_length(addon_name) BETWEEN 1 AND 100),
    CONSTRAINT enrollment_addons_price_delta_check
        CHECK (price_delta_cents >= 0),
    CONSTRAINT enrollment_addons_summary_check
        CHECK (char_length(entitlement_summary) <= 240)
);

CREATE INDEX idx_enrollment_addons_enrollment
    ON enrollment_addons(enrollment_id);

-- Imutabilidade do snapshot de adicionais da matrícula
CREATE OR REPLACE FUNCTION prevent_enrollment_addon_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        RAISE EXCEPTION 'enrollment addons are immutable once created' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_enrollment_addons
BEFORE UPDATE OR DELETE ON enrollment_addons
FOR EACH ROW EXECUTE FUNCTION prevent_enrollment_addon_mutation();
```

---

## 7. Experiência e Fluxos do Usuário

### 7.1 Gestão de Adicionais no Plano (`SPEC-003/B1` — `#/plans`)

1. **Abertura progressiva**: No formulário de rascunho do plano (`PlanFormModal`), após a seção de Direitos Base, há o bloco *"Adicionais opcionais (opcional)"*.
2. **Criação de Adicional**:
   - Botão `+ Adicionar opcional`.
   - Campos:
     - `Nome do adicional` (ex.: *"Recovery 2x"* ou *"Musculação complementar"*);
     - `Valor adicional (R$)` (ex.: `50,00` ou `0,00` para incluso);
     - `Descrição curta` (opcional);
     - `Direitos concedidos`: seleção de modalidade ou serviço ativo com tipo de acesso (`Acesso livre`, `Sessões por período` ou `Pacote`).
3. **Visualização em Cards**:
   - Cada adicional cadastrado no rascunho aparece em um card compacto com nome, valor (ex.: `+ R$ 50,00/mês`), resumo do direito e botões de editar e excluir.
4. **Publicação**: Ao tocar em `Publicar plano`, a tela de revisão apresenta os Direitos Base e a lista de Adicionais Opcionais configurados.

### 7.2 Seleção na Matrícula (`SPEC-003/B2` — `Alunos -> Matrícula`)

1. **Descoberta de Adicionais**:
   - No `EnrollmentPanel`, ao selecionar um plano que contenha adicionais cadastrados em sua versão publicada, surge a seção *"Adicionais disponíveis"*.
   - Se o plano não possuir adicionais, a seção não é exibida.
2. **Seleção Interativa**:
   - Cada adicional é listado com um checkbox acessível, nome, badge de preço (ex.: `+ R$ 50,00` ou `Incluso`) e descrição dos direitos.
3. **Revisão Dinâmica em Tempo Real**:
   - Ao marcar ou desmarcar checkboxes, o bloco de revisão de valores atualiza instantaneamente:
     ```text
     Plano base:       R$ 189,00/mês
     Adicionais (1):   + R$ 50,00/mês
     --------------------------------
     Total do plano:   R$ 239,00/mês
     ```
4. **Aviso aos Parceiros**:
   - Se o aluno tiver origem `wellhub` ou `totalpass`, o banner informativo *"Plano próprio da academia"* permanece visível, deixando claro que os adicionais também compõem a matrícula própria e não alteram o convênio externo.
5. **Confirmação e Histórico**:
   - Ao confirmar a matrícula, o painel exibe o card da matrícula ativa com o valor total, o detalhamento dos adicionais contratados e seus respectivos direitos.
   - O histórico de matrículas passadas/encerradas preserva os adicionais que foram contratados naquela ocasião.

---

## 8. Contratos HTTP e DTOs

### 8.1 Endpoints de Planos (`SPEC-003/B1`)

#### `GET /api/v1/membership-plans/:id`
Retorna a versão publicada e o rascunho (se houver), enriquecidos com a lista de adicionais:

```json
{
  "id": "c1f7a08b-2d1e-4c7b-8f3a-1e9d5a6b7c8d",
  "name": "CrossFit Pro",
  "status": "active",
  "published_version": {
    "id": "e4a8b12c-3d4e-5f6a-7b8c-9d0e1f2a3b4c",
    "version_number": 1,
    "name": "CrossFit Pro",
    "price_cents": 18900,
    "billing_cycle": "monthly",
    "entitlements": [
      {
        "id": "f5b9c23d-4e5f-6a7b-8c9d-0e1f2a3b4c5d",
        "modality_id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
        "modality_name": "CrossFit",
        "access_type": "open_access",
        "summary": "CrossFit · acesso livre"
      }
    ],
    "addons": [
      {
        "id": "b7c8d9e0-1f2a-3b4c-5d6e-7f8a9b0c1d2e",
        "name": "Recovery 2x",
        "description": "Sessões semanais de recuperação",
        "price_delta_cents": 5000,
        "position": 0,
        "summary": "Recovery · 2 sessões por semana",
        "entitlements": [
          {
            "id": "d9e0f1a2-3b4c-5d6e-7f8a-9b0c1d2e3f4a",
            "service_offering_id": "f1e2d3c4-b5a6-7f8e-9d0c-1b2a3f4e5d6c",
            "service_offering_name": "Recovery",
            "access_type": "sessions_per_period",
            "quantity": 2,
            "period_unit": "week",
            "daily_limit": 1,
            "delivery_mode": "individual",
            "booking_required": true,
            "position": 0
          }
        ]
      }
    ]
  }
}
```

#### `POST /api/v1/membership-plans` e `PATCH /api/v1/membership-plans/:id/draft`
Aceitam a lista de adicionais no payload do rascunho:

```json
{
  "name": "CrossFit Pro",
  "price_cents": 18900,
  "billing_cycle": "monthly",
  "entitlements": [...],
  "addons": [
    {
      "name": "Recovery 2x",
      "description": "Sessões semanais de recuperação",
      "price_delta_cents": 5000,
      "position": 0,
      "entitlements": [
        {
          "service_offering_id": "f1e2d3c4-b5a6-7f8e-9d0c-1b2a3f4e5d6c",
          "modality_id": null,
          "access_type": "sessions_per_period",
          "quantity": 2,
          "period_unit": "week",
          "daily_limit": 1,
          "delivery_mode": "individual",
          "booking_required": true,
          "position": 0
        }
      ]
    }
  ]
}
```

### 8.2 Endpoints de Matrícula (`SPEC-003/B2`)

#### `POST /api/v1/units/:unit_id/students/:student_id/enrollments`
- **Header:** `Idempotency-Key: <chave-unica>`
- **Payload:**
```json
{
  "plan_id": "c1f7a08b-2d1e-4c7b-8f3a-1e9d5a6b7c8d",
  "plan_version_id": "e4a8b12c-3d4e-5f6a-7b8c-9d0e1f2a3b4c",
  "starts_on": "2026-08-21",
  "addon_ids": [
    "b7c8d9e0-1f2a-3b4c-5d6e-7f8a9b0c1d2e"
  ]
}
```

#### `GET /api/v1/units/:unit_id/students/:student_id/enrollments`
Retorna a matrícula ativa com composição detalhada:

```json
{
  "open_enrollment": {
    "id": "a3b4c5d6-e7f8-9a0b-1c2d-3e4f5a6b7c8d",
    "person_id": "7b8c9d0e-1f2a-3b4c-5d6e-7f8a9b0c1d2e",
    "plan_id": "c1f7a08b-2d1e-4c7b-8f3a-1e9d5a6b7c8d",
    "plan_version_id": "e4a8b12c-3d4e-5f6a-7b8c-9d0e1f2a3b4c",
    "plan_name": "CrossFit Pro",
    "version_number": 1,
    "billing_cycle": "monthly",
    "base_price_cents": 18900,
    "addons_price_cents": 5000,
    "total_price_cents": 23900,
    "currency": "BRL",
    "starts_on": "2026-08-21",
    "valid_until": null,
    "effective_status": "active",
    "lifecycle_status": "active",
    "base_summary": "CrossFit · acesso livre",
    "addons": [
      {
        "id": "11223344-5566-7788-99aa-bbccddeeff00",
        "plan_version_addon_id": "b7c8d9e0-1f2a-3b4c-5d6e-7f8a9b0c1d2e",
        "name": "Recovery 2x",
        "price_delta_cents": 5000,
        "summary": "Recovery · 2 sessões por semana"
      }
    ]
  },
  "history": [...]
}
```

---

## 9. Erros Estáveis

| Código | HTTP | Situação |
|---|---:|---|
| `membership_plan_addon_invalid` | 422 | Nome vazio, descrição excessiva, `price_delta_cents < 0` ou adicional sem direitos |
| `membership_plan_addon_target_invalid` | 422 | Modalidade ou serviço ofertável do adicional inexistente, inativo ou de outra organização |
| `membership_plan_addon_duplicate` | 422 | Nome de adicional duplicado no rascunho ou mesmo adicional selecionado mais de uma vez na matrícula |
| `membership_plan_addon_not_found` | 422 | `addon_id` informado na matrícula não pertence à versão de plano informada ou não existe |

---

## 10. Autorização, Privacidade e Auditoria

- **Autorização:** Apenas usuários com papel `OWNER` na unidade autenticada podem gerenciar adicionais de planos e criar matrículas com adicionais.
- **Tenancy:** Toda consulta e mutação valida `organization_id` e `home_unit_id` da sessão; chaves compostas no banco e filtros explícitos impedem vazamento entre organizações/unidades.
- **Auditoria:**
  - `membership_plan_audit_events`: Registra as alterações de adicionais nas ações de rascunho, publicação e nova versão do plano.
  - `enrollment_events`: Registra o snapshot completo dos adicionais contratados e do valor total no payload do evento `enrollment.created`.
- **Privacidade (LGPD):** Os registros de `enrollment_addons` são contemplados no serviço de exportação de dados da pessoa (`Privacy Export`), listando os planos e adicionais históricos sem expor dados de terceiros.

---

## 11. Rollout e Migrações

1. **Rollout da Slice B1:**
   - Migration aditiva criando `membership_plan_version_addons` e `membership_plan_version_addon_entitlements` com seus respectivos triggers.
   - Nenhuma dependência sobre dados legados; planos existentes continuam válidos com array `addons: []`.
   - Feature flag existente `FEATURE_MEMBERSHIP_PLANS_ENABLED` controla a interface de planos.
2. **Rollout da Slice B2:**
   - Migration aditiva criando `enrollment_addons` e seus triggers.
   - Matrículas preexistentes continuam válidas com `addons: []`, `addons_price_cents = 0` e `total_price_cents = base_price_cents`.
   - Feature flag existente `FEATURE_ENROLLMENTS_ENABLED` controla a criação e visualização de matrículas.
3. **Ambientes:** As migrations rodam no banco local de desenvolvimento/testes; produção permanece intocada até autorização formal.

---

## 12. Critérios de Aceite

### Critérios de `SPEC-003/B1` (Catálogo de Adicionais):
- [ ] Owner cria plano em rascunho contendo um ou mais adicionais com nome, preço incremental (`>= 0`) e direitos específicos de modalidade ou serviço;
- [ ] Preço incremental zero (`price_delta_cents = 0`) é aceito com sucesso para adicionais inclusos;
- [ ] Preço incremental negativo é rejeitado no backend com erro 422 e validado no frontend;
- [ ] Adicional sem direitos ou com modalidade/serviço inativo é rejeitado no salvamento/publicação;
- [ ] Ao publicar o plano, triggers do PostgreSQL impedem qualquer alteração ou exclusão direta em `membership_plan_version_addons` e seus direitos;
- [ ] Ao criar nova versão do plano (`POST /versions`), todos os adicionais e seus direitos são clonados para o novo rascunho;
- [ ] A interface mobile de planos (`360x800`) permite gerenciar adicionais sem overflow horizontal.

### Critérios de `SPEC-003/B2` (Composição na Matrícula):
- [ ] Owner abre aluno e vê os adicionais disponíveis ao selecionar um plano com adicionais;
- [ ] Marcar adicionais recalcula o valor total em tempo real na tela de revisão: `Total = Preço base + soma(Adicionais)`;
- [ ] Matricular com adicionais grava o registro em `enrollments` e os itens correspondentes em `enrollment_addons` na mesma transação;
- [ ] Informar um `addon_id` que não pertença à versão selecionada retorna erro 422 seguro;
- [ ] A resposta da API e o painel de matrícula retornam `base_price_cents`, `addons_price_cents`, `total_price_cents` e a lista de adicionais contratados;
- [ ] Para alunos com origem `wellhub` ou `totalpass`, o aviso *"Plano próprio da academia"* permanece visível na seleção e na revisão; para `box_member`, o aviso não é exibido;
- [ ] Matrículas criadas anteriormente continuam funcionando normalmente com `addons: []`;
- [ ] A criação da matrícula é idempotente e imutável após commit.

---

## 13. Testes Mínimos Obrigatórios

### Backend Go:
1. **Domínio:** Cálculo do valor total, validação de preços incrementais (`>= 0`), resumo textual de adicionais e rejeição de alvos duplicados ou vazios.
2. **PostgreSQL & Migrations:**
   - Execução das migrations em banco limpo e verificação de idempotência;
   - Triggers de imutabilidade: garantir que `UPDATE` e `DELETE` em adicionais de versões publicadas/retiradas falhem com SQLSTATE `55000`;
   - Triggers de imutabilidade em `enrollment_addons`: garantir que snapshots de matrícula não sejam editáveis;
   - Foreign keys compostas garantindo isolamento estrito de organização/unidade.
3. **Serviços e Repositórios:**
   - Criação, edição de rascunho, publicação e clonagem de planos com adicionais;
   - Matrícula com 0, 1 e múltiplos adicionais;
   - Rejeição atômica quando um dos adicionais pertence a outro plano ou outra versão;
   - Idempotência com replay de `Idempotency-Key` retornando exatamente o mesmo payload;
   - Exportação de privacidade (LGPD) contendo histórico de adicionais.
4. **Suíte completa:** `go test ./...`, `go vet ./...` e `go test -race -count=1 ./...` em ambiente PostgreSQL real.

### Frontend React & Playwright:
1. **Gerenciamento de Planos:** Cadastro de plano com adicionais, validação de campos obrigatórios, máscara monetária BRL e publicação.
2. **Painel de Matrícula:**
   - Exibição de adicionais disponíveis ao selecionar plano;
   - Interação de checkboxes com recálculo em tempo real do valor total;
   - Preservação obrigatória do aviso *"Plano próprio da academia"* para alunos parceiros;
   - Submissão da matrícula e renderização correta do detalhamento no card ativo e no histórico.
3. **Responsividade:** Verificação visual e funcional em viewport mobile `360x800` sem quebras de layout ou overflow horizontal.
4. **Qualidade:** `tsc -b`, `vite build` e Playwright completo.

---

## 14. Decisões que Exigem Validação do Produto / Alados

As seguintes decisões de negócio foram delimitadas para manter a spec executável, mas devem ser acompanhadas e validadas na rotina com o Alados:

1. **Sem desconto progressivo em combos:** Não há regra de *"leve 2 adicionais e ganhe 10% de desconto"*. O valor total é a soma pura e simples dos preços incrementais. Caso o Alados exija preços promocionais em pacotes de adicionais, a recomendação é cadastrar uma versão de plano específica ou um adicional combinado (ex.: *"Combo Recovery + Musculação"*).
2. **Adicionais com preço zero:** Adicionais podem ter `price_delta_cents = 0` para indicar serviços opcionais inclusos no plano (ex.: avaliação física opcional gratuita).
3. **Rigidez durante a vigência:** O aluno não pode cancelar ou adicionar um serviço adicional avulso mantendo a matrícula base sem encerrá-la. Se o aluno desejar mudar os adicionais, a academia encerra a matrícula atual e cria uma nova matrícula.
4. **Sem integração de agendamento:** O adicional concede o direito comercial e define `booking_required` ou modo de atendimento, mas o agendamento concreto da sessão de Recovery será tratado exclusivamente na `SPEC-006` (Agenda) e `SPEC-007` (Reservas).

---

## 15. Não Objetivos e Proibições para o Agente Executor

- **NÃO** implementar geração de contratos, documentos jurídicos ou termos em PDF (`SPEC-004`);
- **NÃO** implementar integração de cobrança com gateway Asaas, emissão de boletos ou cobrança no cartão (`SPEC-005`);
- **NÃO** alterar regras de consumo de check-in, bloqueio de catraca ou presença de alunos (`SPEC-003/D` e `SPEC-008`);
- **NÃO** criar regras de desconto, cupons ou valores negativos;
- **NÃO** modificar retroativamente matrículas existentes;
- **NÃO** realizar `git push`, deploy em produção ou alterações no banco de produção;
- **NÃO** remover ou modificar os arquivos preservados `.ai/acordo-fundadores-engagefit.md` e `docs/proposta-engagefit-guilherme.html`.
