# SPEC-004/A — Fundação de contratos e emissão vinculada à matrícula

- Status: `ready`
- Versão: `1.0`
- Data: 2026-08-24
- Depende de: `ARCH-001`, `SPEC-002/A`, `SPEC-003/A-B-C`
- Bloqueia: aceite eletrônico produtivo (`SPEC-004/B`) e o contrato interno
  consumido por `SPEC-005`
- Ownership principal: módulo novo `contracts`
- Contratos colaboradores: `memberships` e `people`, somente pelas portas
  públicas desta spec
- Repositórios envolvidos em uma implementação futura: `engage-fit-be` e
  `engage-fit-fe`

## Estratégia e autoridade desta spec

Esta é a primeira slice da iniciativa `SPEC-004`. Ela fecha somente a
fundação de documentos contratuais, a emissão vinculada a uma matrícula já
existente e um aceite exercitável por port com simulador local.

Produto, arquitetura e privacidade revisaram esta spec e aprovaram as quatro
decisões da seção 20. O documento está em `ready` e é o único contrato
executável da slice A. Essa autorização continua limitada à fundação local
descrita aqui: não autoriza integração produtiva, dados reais, deploy,
habilitação de feature flag ou qualquer escopo da `SPEC-004/B-D`. A
spec-guia `SPEC-003` e os documentos de direção não podem ampliar seu escopo
silenciosamente.

## 1. Resultado esperado

O owner configura um modelo contratual versionado, gera uma prévia com os
termos comerciais exatos de uma matrícula, cria uma instância imutável e a
emite para aceite. Em desenvolvimento e homologação local, um simulador permite
percorrer o aceite sem provedor, mensagem ou e-mail real.

O owner acompanha no celular uma linha de estado simples:

```text
Rascunho -> Aguardando aceite -> Aceito
```

O documento preserva plano, versão, adicionais, valores e vigência existentes
na matrícula no momento da geração. Ele não altera a matrícula e não afirma
pagamento, acesso, reserva, consumo, cancelamento ou situação de saúde.

## 2. Problema e evidências

- **Existente:** `SPEC-003/C` registra uma matrícula operacional para a
  `Person`, vinculada à versão publicada exata do plano.
- **Existente:** `SPEC-003/B` preserva em `enrollment_addons` o nome, preço e
  resumo dos adicionais selecionados, e a leitura de memberships retorna
  preço base, adicionais e total.
- **Existente:** matrícula `active` não significa contrato assinado,
  pagamento, adimplência ou acesso autorizado.
- **Direção arquitetural:** `ARCH-001` atribui documentos, versões e aceite ao
  módulo `contracts`; respostas de PAR-Q pertencem a `healthforms` e cobrança
  do aluno pertence a `memberbilling`.
- **Entrada Alados:** o contrato deve ser gerado depois do cadastro ou da
  adesão comercial e futuramente poderá usar um provedor de assinatura.
- **Inferência EngageFit:** a primeira slice precisa ser útil e testável sem
  escolher um fornecedor. Um port estreito e um simulador exclusivamente local
  permitem validar modelo, hashes, estados e UX antes da dependência externa.
- **Inferência EngageFit:** a instância emitida é uma fotografia; alterar
  template, plano ou catálogo depois não reescreve o documento.

## 3. Escopo

### Incluído

- catálogo de templates contratuais por organização e unidade de origem;
- rascunho, publicação imutável e nova versão de template;
- corpo textual em Markdown restrito, declaração de aceite e placeholders
  permitidos;
- preview determinístico com os termos da matrícula;
- instância contratual vinculada a `enrollment_id`, `person_id`, unidade e
  versão exata do template;
- snapshot comercial versionado do plano base e adicionais da matrícula;
- estados de instância, eventos duráveis, auditoria, revisão otimista e
  idempotência;
- emissão e preparação de aceite por port;
- adapter `local_simulator`, sem rede, proibido em produção;
- sessão local de aceite com token opaco de uso único e armazenamento somente
  do hash;
- consulta de estado e histórico pelo owner;
- integração ao privacy export e política explícita para dados do simulador;
- feature flag desligada por padrão e rollout aditivo;
- UX mobile-first de templates, preview, emissão e acompanhamento.

### Fora do escopo

- PAR-Q, anamnese, laudo, lesão, condição de saúde ou qualquer dado pessoal
  sensível;
- cobrança do aluno, parcela, recorrência executada, Pix, boleto, cartão,
  estorno, inadimplência ou conciliação;
- adapter de assinatura eletrônica real, Gov.br, certificado, biometria,
  selfie, OTP ou validação documental;
- afirmação de validade jurídica do aceite produzido pelo simulador;
- envio real por WhatsApp, e-mail, push ou outro canal;
- pausa, trancamento, aditivo, renovação, distrato, cancelamento jurídico,
  multa ou deslocamento de vigência;
- agenda, profissional, aula, vaga, reserva, lista de espera ou no-show;
- saldo, consumo, catraca, check-in ou decisão de acesso;
- alterar, encerrar ou reconciliar a matrícula;
- PDF, impressão, certificado visual ou armazenamento de arquivo em object
  storage;
- contrato de Wellhub/TotalPass ou alteração do vínculo com parceiro;
- multiunidade, seletor de unidade ou autorização além de `users.box_id`.

## 4. Vocabulário

- **Template de contrato:** identidade reutilizável do documento na
  organização, por exemplo `Termo de matrícula padrão`.
- **Versão do template:** título, corpo, declaração de aceite e conjunto de
  placeholders que se tornam imutáveis depois da publicação.
- **Instância de contrato:** documento renderizado para uma matrícula e uma
  pessoa específicas.
- **Snapshot comercial:** objeto canônico com versão do plano, composição e
  vigência observadas na matrícula ao gerar a instância.
- **Emissão:** transição que fecha o rascunho e o coloca aguardando aceite.
- **Sessão de aceite:** credencial curta e de uso único que abre a experiência
  do adapter configurado.
- **Aceite:** evidência verificada pelo port e vinculada ao hash exato do
  documento; nesta slice, somente evidência de simulador local.
- **Assinatura eletrônica:** capacidade produtiva com identidade e evidências
  de provedor, deliberadamente posterior.

Na UI usar `Modelo de contrato`, `Prévia`, `Emitir`, `Aguardando aceite` e
`Aceito`. Não chamar o resultado local de `assinado digitalmente`.

## 5. Fluxos do usuário

### 5.1 Criar e publicar um modelo

1. Owner abre `Contratos` e toca em `Novo modelo`.
2. Informa nome interno, título do documento, corpo e declaração de aceite.
3. A interface oferece somente os placeholders permitidos, com inserção por
   toque; não exige que o owner conheça sintaxe técnica.
4. Uma prévia usa dados sintéticos identificados como exemplo.
5. Owner salva o rascunho e publica após uma revisão explícita.
6. A versão publicada fica imutável. Alterações posteriores exigem `Nova
   versão`.

Não haverá template jurídico semeado pela migration. O conteúdo inicial é
responsabilidade do operador e precisa de revisão adequada antes de uso real.

### 5.2 Gerar prévia para uma matrícula

1. No painel de matrícula do aluno, owner abre a seção `Contrato`.
2. Seleciona um modelo ativo com versão publicada.
3. O backend resolve a matrícula pela porta de `memberships` e a pessoa pela
   porta de `people`.
4. A prévia mostra título, corpo renderizado, plano, versão, adicionais,
   valores, início e validade quando houver.
5. A tela informa: `Esta etapa não gera cobrança, não altera a matrícula e não
   libera acesso.`
6. O backend retorna o hash da prévia. Nada é persistido por essa consulta.

### 5.3 Criar rascunho e emitir

1. Owner confirma a prévia e cria a instância em `draft` usando o hash
   revisado.
2. O sistema persiste documento e snapshots imutáveis.
3. Owner ainda pode descartar somente esse rascunho não emitido.
4. Ao tocar em `Emitir para aceite`, a aplicação revalida que matrícula,
   pessoa, template e hash continuam elegíveis.
5. A instância passa para `pending_acceptance` e recebe `issued_at`.
6. Em ambiente com simulador habilitado, owner pode gerar um link local de
   homologação. O sistema não envia o link.

### 5.4 Aceitar no simulador local

1. O port prepara uma sessão com token opaco e validade curta.
2. A URL local abre o mesmo documento e sua declaração de aceite.
3. A tela identifica claramente `Simulador local — sem validade de assinatura
   eletrônica`.
4. A pessoa de teste confirma a declaração e o hash exibido.
5. O adapter devolve evidência normalizada; aplicação locka a sessão e a
   instância, grava o aceite e o evento e muda o status para `accepted` na
   mesma transação.
6. Retry idêntico retorna o mesmo aceite. Token usado não aceita novo comando.

### 5.5 Consultar estado

- rascunho: ação primária `Emitir para aceite`;
- aguardando: ação primária local `Gerar link de homologação` quando não houver
  sessão válida; caso contrário, `Copiar link` só aparece na resposta inicial
  que contém o segredo;
- aceito: mostra data, método `Simulador local` e hash abreviado, sem botão para
  alterar;
- descartado: fica somente no histórico, sem reativação.

## 6. Experiência mobile-first do owner

### Navegação e hierarquia

- criar `#/contracts`, no grupo `Gestão`, visível somente para `OWNER` e
  capability `contracts`;
- a página administra modelos; a operação de uma instância começa no painel
  de matrícula já existente em `Alunos`;
- não adicionar a ação a cada aluno na carga inicial nem gerar N+1: contratos
  são carregados somente quando o painel de matrícula é aberto;
- em `360x800`, usar cards, formulário em uma coluna e painel com rolagem
  interna, sem tabela comprimida ou overflow horizontal;
- uma ação primária por estado; arquivar, descartar rascunho e gerar nova
  versão ficam como ações secundárias.

### Editor e preview

- campos iniciais: `Nome do modelo`, `Título do documento` e `Conteúdo`;
- `Declaração de aceite` fica visível antes da publicação porque é parte do
  documento versionado;
- placeholders aparecem como chips com rótulos de produto;
- a UI não oferece HTML, JavaScript, imagens remotas, anexos ou upload;
- preview mantém texto legível, quebra de linha e resumo comercial em cards;
- valores usam BRL e datas usam o formato local, mas o hash é calculado a
  partir da representação canônica do backend.

### Estados obrigatórios

- loading contextual;
- vazio sem template publicado, com link para criar/publicar;
- matrícula inelegível;
- prévia alterada antes da confirmação;
- erro recuperável com `request_id`;
- conflito de revisão em outra aba;
- rascunho, aguardando, sessão expirada, aceito e descartado;
- foco inicial, retorno de foco, `Escape`, labels e alvo touch mínimo de 44px.

O frontend nunca chama o simulador em produção e não mantém token de aceite em
`localStorage`, log, analytics ou URL de navegação interna além da página de
simulação explicitamente aberta.

## 7. Regras e invariantes

1. Template pertence a uma organização e registra a unidade de origem.
2. Enquanto não houver contexto multiunidade, somente a unidade de origem
   consulta e opera template e instância.
3. Nome normalizado do template é único na organização, inclusive arquivado.
4. Um template possui no máximo um rascunho e uma versão publicada atual.
5. Versão publicada ou retirada é imutável; nova edição clona para novo
   rascunho com próximo número sem lacuna.
6. Publicação exige título, corpo, declaração de aceite e placeholders válidos.
7. A versão publicada recebe hash SHA-256 da representação canônica do seu
   conteúdo e esquema.
8. Nesta slice existe apenas `contract_kind=membership_terms`.
9. Toda instância pertence a exatamente uma organização, unidade, pessoa,
   matrícula e versão publicada de template.
10. O repositório de `contracts` não consulta tabelas de `memberships`,
    `people`, `students` ou `billing`; resolução ocorre por portas públicas.
11. A matrícula precisa estar efetivamente `scheduled` ou `active` para criar
    rascunho e emitir. `expired` ou `ended` retorna erro sem mutação.
12. Instância persiste `person_id` e `enrollment_id`; nunca persiste
    `student_id` como identidade contratual.
13. Snapshot comercial é obtido no servidor. O cliente não envia preço,
    adicional, nome do plano, validade ou resumo editável.
14. O snapshot inclui zero ou mais adicionais da matrícula e preserva a ordem
    determinística por posição e ID.
15. `total_price_cents = base_price_cents + addons_price_cents` precisa
    coincidir com o snapshot de memberships; divergência aborta a operação.
16. Documento renderizado, snapshots e hashes são imutáveis desde a criação do
    rascunho. Correção exige descartar o rascunho antes de emitir e criar outro.
17. Existe no máximo uma instância não descartada de
    `membership_terms` por matrícula.
18. Estados persistidos são `draft`, `pending_acceptance`, `accepted` e
    `discarded`.
19. Transições permitidas são somente `draft -> pending_acceptance`,
    `draft -> discarded` e `pending_acceptance -> accepted`.
20. Instância emitida ou aceita nunca é editada, descartada, excluída ou
    reemitida por comando de produto. A única remoção da slice é o purge
    operacional de artefato sintético vencido do simulador, conforme a seção
    14.
21. Antes do aceite, a aplicação revalida que a matrícula continua
    `scheduled|active`; caso contrário, mantém a instância pendente e bloqueia
    o aceite com erro estável, sem encerrar ou cancelar nada.
22. Aceite referencia o hash exato persistido; hash divergente nunca é
    normalizado ou aceito silenciosamente.
23. Existe no máximo um aceite por instância e uma sessão ativa por instância.
24. Criar template, publicar, criar versão, criar instância, emitir, gerar
    sessão e aceitar são idempotentes no escopo da organização.
25. Mudanças de rascunho e de estado exigem `expected_revision`.
26. Evento, auditoria, resposta idempotente e mutação local pertencem à mesma
    transação.
27. O simulador não chama rede, não autentica juridicamente o signatário e é
    recusado em `APP_ENV=production`.
28. Nenhuma operação cria cobrança, comunicação, reserva, consumo, check-in ou
    decisão de acesso.

## 8. Conteúdo, placeholders e renderização

### Formato

- `document_format` inicial é `text/markdown`;
- corpo entre 1 e 50.000 caracteres; título entre 1 e 160; declaração de
  aceite entre 10 e 1.000;
- Markdown é renderizado com allowlist sem HTML bruto, script, iframe, imagem
  remota ou URL executável;
- o backend preserva o Markdown canônico e o frontend usa renderer seguro;
- o hash considera versão do renderer, título, corpo renderizado, declaração
  de aceite e snapshots canônicos.

### Placeholders permitidos em `schema_version=1`

| Token | Origem |
|---|---|
| `{{organization_name}}` | contexto autenticado de organização |
| `{{unit_name}}` | unidade autenticada |
| `{{member_name}}` | port público de `people` |
| `{{plan_name}}` | snapshot de `memberships` |
| `{{plan_version}}` | snapshot de `memberships` |
| `{{billing_cycle}}` | snapshot de `memberships` |
| `{{base_price}}` | preço base formatado pelo backend |
| `{{addons_summary}}` | lista determinística ou `Sem adicionais` |
| `{{addons_price}}` | soma dos adicionais formatada |
| `{{total_price}}` | total formatado |
| `{{starts_on}}` | início da matrícula |
| `{{valid_until}}` | data ou `Sem término definido` |
| `{{access_summary}}` | resumo comercial já produzido por memberships |

Token desconhecido, não fechado ou ausente no contexto bloqueia publicação ou
preview com erro de campo. Não há loops, condicionais, includes, chamadas a IA
ou variáveis livres.

CPF, endereço, e-mail e telefone não fazem parte do schema 1. Qualificação
jurídica completa depende da revisão da `SPEC-004/B`; o simulador não deve ser
usado como contrato produtivo.

## 9. Modelo de domínio e dados proposto

O número de migration não é reservado por esta spec. O integrador escolherá a
sequência no merge; a implementação futura parte do schema integrado até
`059` e não modifica migrations publicadas.

### `contract_templates`

- `id`, `organization_id`, `home_unit_id`;
- `code`, `name`, `name_key`, `contract_kind=membership_terms`;
- `status=active|archived`;
- timestamps;
- unique `(organization_id, name_key)` e `(id, organization_id)`;
- FK composta de unidade e organização.

`code` é opaco e estável para integrações futuras; não é editável depois da
primeira publicação.

### `contract_template_versions`

- `id`, `organization_id`, `template_id`, `version_number`, `revision`;
- `status=draft|published|retired`;
- `title`, `body_markdown`, `acceptance_statement`;
- `placeholder_schema_version=1`, `renderer_version=markdown-v1`;
- `content_sha256`, `published_at`, timestamps;
- unique `(template_id, version_number)`;
- índices únicos parciais para um rascunho e uma versão publicada;
- trigger que impede mutação/remoção de versões `published|retired`, exceto a
  transição controlada `published -> retired`.

### `contract_instances`

- IDs: `id`, `organization_id`, `unit_id`, `person_id`, `enrollment_id`,
  `template_version_id`;
- `contract_kind=membership_terms`;
- `status=draft|pending_acceptance|accepted|discarded` e `revision`;
- `snapshot_schema_version=1`;
- `commercial_snapshot JSONB` e `commercial_snapshot_sha256`;
- `party_snapshot JSONB` contendo somente organização, unidade, `person_id` e
  nome exibido;
- `document_title`, `document_body_markdown`, `document_sha256`;
- `acceptance_statement`, `acceptance_statement_version`;
- `created_by_user_id`, `issued_by_user_id`, `discarded_by_user_id`;
- `issued_at`, `accepted_at`, `discarded_at`, timestamps;
- `retention_class=local_simulator|production_contract` e
  `retention_due_at` nullable.

O `commercial_snapshot` canônico contém:

```json
{
  "schema_version": 1,
  "enrollment_id": "uuid",
  "enrollment_revision": 1,
  "plan_id": "uuid",
  "plan_version_id": "uuid",
  "plan_version_number": 1,
  "plan_name": "CrossFit Pro",
  "billing_cycle": "monthly",
  "currency": "BRL",
  "base_price_cents": 18900,
  "addons_price_cents": 5000,
  "total_price_cents": 23900,
  "starts_on": "2026-08-24",
  "valid_until": null,
  "base_summary": "CrossFit · acesso livre",
  "addons": [
    {
      "plan_version_addon_id": "uuid",
      "name": "Recovery 2x",
      "price_delta_cents": 5000,
      "summary": "Recovery · 2 sessões por semana"
    }
  ]
}
```

Adicionar uma chave candidata aditiva em `enrollments` que permita FK composta
de `(enrollment_id, organization_id, person_id, home_unit_id)`. A FK de
`contract_instances` usa os quatro campos e impede misturar matrícula, pessoa
ou unidade dentro da mesma organização. A migration não altera dados ou
semântica de memberships.

Um índice único parcial limita a uma instância por matrícula/tipo quando
`status <> 'discarded'`. Trigger impede qualquer alteração de documento,
snapshots, IDs ou hashes; somente as transições e timestamps listados na seção
7 podem mudar. Não existe delete no repository ou na API de produto. O
repository operacional de retenção pode remover somente a árvore inteira de
uma instância `retention_class=local_simulator` vencida, em transação e depois
das validações da seção 14.

### `contract_acceptance_sessions`

- `id`, `organization_id`, `contract_instance_id`;
- `adapter=local_simulator`, `adapter_reference` opaca;
- `token_hash` SHA-256 do token aleatório de alta entropia; token bruto nunca
  é persistido;
- `status=active|used|expired|superseded`;
- `expires_at`, `used_at`, timestamps;
- no máximo uma sessão `active` por instância;
- criar sessão nova invalida a anterior ainda ativa na mesma transação.

Sessão padrão expira em 30 minutos. A URL/token aparece somente na resposta de
criação; logs, eventos, auditoria e command response persistida não a contêm.
Se a resposta se perder, o owner cria uma nova sessão com nova chave, e a
anterior é invalidada.

### `contract_acceptances`

- `id`, `organization_id`, `contract_instance_id`,
  `acceptance_session_id`;
- `adapter`, `adapter_reference`, `evidence_schema_version=1`;
- `person_id`, `document_sha256`, `acceptance_statement_version`;
- `evidence JSONB` minimizado e `accepted_at`;
- unique por instância e por sessão.

No simulador, evidence contém somente adapter, referência opaca, hashes,
versão da declaração e horário. Não contém IP, user-agent, headers, token,
payload HTTP, CPF ou contato. Essa evidência não é apresentada como assinatura
legal.

### `contract_command_requests`

- escopo `(organization_id, command, idempotency_key)`;
- comandos: `template_create|template_publish|template_new_version|instance_create|instance_issue|instance_discard|acceptance_session_create|accept`;
- fingerprint SHA-256 do payload canônico, IDs resultantes e resposta segura;
- chave de 8–128 caracteres em `[A-Za-z0-9._:-]`;
- mesma chave/fingerprint retorna resultado; uso diferente retorna
  `409 idempotency_key_reused`.

Token e URL de sessão não entram na resposta persistida. Replay de criação de
sessão retorna o mesmo ID sem revelar novamente a credencial.

### `contract_events`

- organização, unidade, pessoa, matrícula, instância, ator e timestamp;
- ator `owner|acceptance_subject|system`, com `actor_user_id` somente para
  owner;
- ações:
  - `contract.draft_created`;
  - `contract.draft_discarded`;
  - `contract.issued`;
  - `contract.acceptance_session_created`;
  - `contract.acceptance_session_superseded`;
  - `contract.accepted`;
- `before_data`/`after_data` contêm status, revisões, IDs e hashes, nunca corpo
  do documento, nome ou token.

### `contract_template_audit_events`

- organização, unidade, template, versão, owner e timestamp;
- ações `template.draft_created|draft_updated|published|new_version_created|archived|reactivated`;
- snapshots de auditoria preservam hashes, versão e campos estruturais, sem
  copiar o corpo para logs.

## 10. Contratos internos e fronteiras

### Porta pública de memberships

`contracts` não acessa models ou tabelas de memberships:

```text
ResolveContractEnrollmentSnapshot(
  organization_id,
  unit_id,
  enrollment_id,
  as_of
) ->
  person_id
  enrollment_revision
  effective_status
  plan/version identity
  billing_cycle, currency
  base/addons/total prices
  starts_on, valid_until
  base summary and ordered addon snapshots
```

A porta retorna `scheduled|active|expired|ended`; contracts aceita somente os
dois primeiros nos pontos definidos. Ela não expõe repository, GORM model,
CPF, aluno ou lógica de acesso.

### Porta pública de people

```text
ResolveContractParty(organization_id, person_id)
  -> display_name, profile_status, eligible
```

O schema 1 recebe somente nome de exibição. Pessoa arquivada ou anonimizada é
inelegível e outro tenant parece inexistente.

### Port de aceite

```text
ContractAcceptancePort.Prepare(instance_id, document_sha256, expires_at)
  -> adapter_reference, launch_token, launch_path

ContractAcceptancePort.Verify(session, submission)
  -> verified, accepted_at, normalized_evidence
```

O port não recebe CPF, telefone, e-mail, preço editável ou payload HTTP bruto.
`local_simulator` é o único adapter da slice e não abre rede. Adapter real
exigirá uma spec aceita com claim idempotente antes do efeito externo,
timeouts, reconciliação, webhook autenticado e retenção de evidência.

## 11. Contratos HTTP propostos

Todas as rotas privadas ficam sob `/api/v1`, exigem `OWNER`, tenant da sessão,
CSRF nos comandos e capability `contracts`.

### Templates

- `GET /contract-templates?include_archived=false`
- `GET /contract-templates/:template_id`
- `POST /contract-templates` — `Idempotency-Key`, cria template e versão 1
  draft;
- `PATCH /contract-templates/:template_id/draft` — payload completo e
  `expected_revision`;
- `POST /contract-templates/:template_id/publish` — `Idempotency-Key` e
  `draft_revision`;
- `POST /contract-templates/:template_id/versions` — `Idempotency-Key`, clona
  versão publicada;
- `POST /contract-templates/:template_id/archive`;
- `POST /contract-templates/:template_id/reactivate`.

### Instância vinculada à matrícula

- `GET /units/:unit_id/enrollments/:enrollment_id/contracts`
- `POST /units/:unit_id/enrollments/:enrollment_id/contracts/preview`
- `POST /units/:unit_id/enrollments/:enrollment_id/contracts` — cria draft com
  `Idempotency-Key`;
- `GET /units/:unit_id/contracts/:contract_id`
- `POST /units/:unit_id/contracts/:contract_id/issue`
- `POST /units/:unit_id/contracts/:contract_id/discard`
- `POST /units/:unit_id/contracts/:contract_id/acceptance-sessions` — somente
  quando simulador local estiver habilitado.

Preview recebe somente `template_id` e `template_version_id`. Criação recebe os
mesmos IDs, `expected_preview_sha256` e nenhuma informação comercial editável.
Issue/discard recebem `expected_revision`.

Exemplo resumido de preview:

```json
{
  "template_version_id": "uuid",
  "enrollment_id": "uuid",
  "document_title": "Termo de matrícula — CrossFit Pro",
  "document_body_markdown": "...",
  "acceptance_statement": "Li e aceito os termos apresentados.",
  "commercial_snapshot": {
    "base_price_cents": 18900,
    "addons_price_cents": 5000,
    "total_price_cents": 23900,
    "currency": "BRL",
    "addons": [
      {
        "plan_version_addon_id": "uuid",
        "name": "Recovery 2x",
        "price_delta_cents": 5000,
        "summary": "Recovery · 2 sessões por semana"
      }
    ]
  },
  "preview_sha256": "64-hex",
  "warnings": [
    "A prévia não gera cobrança e não altera a matrícula."
  ]
}
```

### Simulador local

- `GET /api/v1/public/contract-acceptance/:token`
- `POST /api/v1/public/contract-acceptance/:token/accept` — exige
  `Idempotency-Key` e body com `document_sha256`,
  `acceptance_statement_version` e `accepted=true`.

As rotas não são registradas em produção. GET usa `Cache-Control: no-store`,
não expõe outros contratos e retorna somente o documento da sessão. POST tem
rate limit, lock de linha e resposta genérica para token inválido, expirado,
usado ou de outro escopo.

## 12. Erros estáveis

| HTTP | Código | Situação |
|---:|---|---|
| 400 | `invalid_contract_input` | JSON, UUID, template ou campo inválido |
| 400 | `invalid_idempotency_key` | header ausente ou fora do padrão |
| 404 | `contract_template_not_found` | template ausente ou outro tenant |
| 404 | `contract_instance_not_found` | instância ausente ou outro escopo |
| 404 | `contract_enrollment_not_found` | matrícula/pessoa fora do escopo |
| 409 | `contract_template_name_conflict` | nome normalizado já usado |
| 409 | `contract_template_revision_conflict` | rascunho mudou em outra aba |
| 409 | `contract_template_draft_exists` | já existe rascunho |
| 409 | `contract_instance_exists` | matrícula já possui instância não descartada |
| 409 | `contract_preview_changed` | hash revisado não corresponde ao servidor |
| 409 | `contract_revision_conflict` | estado/revisão mudou |
| 409 | `contract_already_issued` | comando inválido sobre emitido/aceito |
| 409 | `contract_acceptance_document_changed` | hash/declaração divergente |
| 409 | `idempotency_key_reused` | chave reutilizada com outro fingerprint |
| 422 | `contract_template_invalid` | corpo ou placeholders inválidos |
| 422 | `contract_enrollment_unavailable` | matrícula não está scheduled/active |
| 422 | `contract_party_ineligible` | pessoa inativa/anonimizada |
| 422 | `contract_acceptance_unavailable` | adapter/simulador desligado |
| 404 | `contract_acceptance_session_invalid` | token inválido, expirado ou usado |

Respostas seguem o envelope atual com `request_id`. SQL, constraint, corpo do
documento, nome, token e evidência bruta nunca aparecem no erro ou log.

## 13. Autorização, tenancy e segurança

- somente `OWNER` vê conteúdo, valores, templates e instâncias;
- `COACH` e `PLATFORM_ADMIN` não acessam a página nem as rotas;
- `organization_id`, `unit_id`, `person_id` e `enrollment_id` não vêm do body
  como autoridade;
- `unit_id` da rota precisa ser `users.box_id`; UUID diferente retorna `404`;
- toda query filtra organização e unidade antes do ID;
- FKs compostas reforçam matrícula, pessoa, unidade e template do mesmo
  tenant;
- simulador usa pelo menos 32 bytes aleatórios, token em URL somente no
  ambiente local e comparação constante do hash;
- tokens não entram em access log, telemetry, analytics, evento ou
  persistência; a configuração local deve suprimir a URL concreta dos logs;
- páginas de preview/aceite usam `no-store`, sem service worker cache e sem
  recursos remotos;
- documento Markdown é sanitizado no backend e novamente renderizado com
  allowlist no frontend;
- nenhum comando aceita HTML, JavaScript, URL de callback ou template remoto.

## 14. Privacidade, exportação e retenção

### Minimização e classificação

- PII nova: `member_name` no snapshot de parte e no documento renderizado;
- termos comerciais, hashes, IDs e datas não devem ser tratados como livres
  de proteção apenas porque não contêm contato;
- CPF, telefone, e-mail, endereço, dados de saúde e payload de provedor não
  entram na slice;
- logs e métricas usam somente ação, resultado, status, versão e duração; IDs
  podem aparecer em logs controlados, nunca como labels de alta cardinalidade.

### Exportação

O privacy export compõe uma porta pública de leitura de `contracts`; o
repositório de privacy não consulta as tabelas novas. Para a pessoa exportada,
incluir:

- metadados da instância e estados;
- template/versão referenciados;
- snapshot comercial completo;
- documento renderizado;
- eventos da instância;
- aceite e evidência normalizada quando existirem;
- sessões apenas como status e datas, nunca token/hash interno.

Gerar o export cria o audit event de privacidade já existente. Templates sem
instância da pessoa não entram.

### Anonimização e retenção

- uma instância aceita é documento imutável com PII e não pode ser reescrita
  silenciosamente pela anonimização genérica;
- rascunhos e emissões não aceitas também contêm PII e precisam de política de
  descarte/invalidação antes de produção;
- até revisão jurídica definir base, prazo e resposta ao pedido de eliminação,
  `retention_class=production_contract` não pode ser criado e a capability
  permanece desligada em produção;
- instâncias do `local_simulator` só podem usar dados sintéticos, recebem
  `retention_due_at=created_at+30 dias` e são removidas pelo comando de
  retenção junto de sessões, aceites e eventos relacionados;
- tokens são apagados/inutilizados no uso, e sessões expiradas são elegíveis à
  limpeza em até 24 horas;
- o comando futuro `engagefit-contract-retention` é dry-run por padrão, exige
  `--apply`, trabalha em transação/lotes, registra somente contagens e nunca
  imprime documento ou nome;
- legal hold, retenção de contratos produtivos e eventual redação/eliminação
  são decisões da `SPEC-004/B` e bloqueiam a ativação produtiva, não os testes
  locais da fundação.

## 15. Observabilidade e operação

Métricas sem labels de IDs ou conteúdo:

- `engagefit.contracts.commands{command,result}`;
- `engagefit.contracts.command.duration{command}`;
- `engagefit.contracts.instances{status}` apenas como contador de transições;
- `engagefit.contracts.conflicts{reason}`;
- `engagefit.contracts.acceptance_sessions{result}`;
- `engagefit.contracts.acceptances{adapter,result}`.

Logs distinguem preview, criação, replay, emissão, sessão, aceite, conflito e
rollback. Falha de evento, auditoria, command request ou evidência aborta a
transação local. Não há alerta externo, worker de polling ou reconciliação de
provedor nesta slice.

O suporte usa `request_id`, `contract_id` e estado. Nunca solicita token,
documento completo, CPF ou screenshot com dados pessoais.

## 16. Capability, configuração e rollout

Adicionar no futuro:

```text
FEATURE_CONTRACTS_ENABLED=false
CONTRACT_ACCEPTANCE_ADAPTER=disabled|local_simulator
CONTRACT_ACCEPTANCE_SIMULATOR_ENABLED=false
CONTRACT_ACCEPTANCE_SESSION_TTL_MINUTES=30
```

Regras:

- `contracts=true` exige `enrollments=true`; as dependências já obrigatórias
  de memberships e people continuam válidas;
- `local_simulator` exige `contracts=true`, ambiente não produtivo e flag do
  simulador;
- production falha no startup se adapter ou simulador local estiver
  habilitado;
- `/capabilities` expõe `contracts` e
  `contract_acceptance_simulator` como booleans seguros;
- campos ausentes são `false` no frontend;
- nenhum default habilita contratos ou simulador.

Rollout compatível:

1. migration aditiva sobre `059`, sem backfill ou seed;
2. reconciliar que todas as tabelas novas começam vazias;
3. publicar backend com flags desligadas;
4. publicar frontend em qualquer ordem, tolerando capabilities ausentes;
5. habilitar contratos e simulador somente em banco/ambiente temporário com
   dados sintéticos;
6. homologar template, preview, emissão, expiração, aceite e export;
7. desligar o simulador e limpar os artefatos ao terminar;
8. não habilitar produção nesta slice.

Rollback operacional desliga as flags e volta os binários se necessário. O
schema aditivo permanece. A migration não altera instâncias de matrícula nem
fluxos existentes.

## 17. Compatibilidade e limites arquiteturais

- backend continua monólito modular e frontend continua uma Vite SPA;
- `contracts` depende de portas públicas, não de repositories ou DTOs HTTP de
  `memberships`/`people`;
- memberships e people não passam a depender de contracts;
- `billing` SaaS e o futuro `memberbilling` não são importados;
- nenhuma query de contrato participa da listagem atual de alunos;
- ausência da capability mantém frontend e backend atuais inalterados;
- matrícula sem contrato continua válida operacionalmente e não muda check-in;
- matrícula antiga não recebe contrato inventado por backfill;
- versão de plano, adicional ou template publicada depois não altera instância
  existente;
- `docs/system-design.md` só será atualizado depois de implementação e
  validação; esta preparação documental não descreve comportamento atual.

## 18. Critérios de aceite da implementação futura

### Produto e UX

- owner cria, edita e publica modelo em `360x800` sem overflow;
- versão publicada e corpo não podem ser alterados no PostgreSQL;
- preview usa exatamente plano, versão, adicionais, total e vigência da
  matrícula;
- criação com hash de preview divergente não persiste instância;
- rascunho pode ser descartado, mas emitido/aceito não pode;
- emissão mostra claramente ausência de cobrança, acesso e envio real;
- simulador local percorre documento -> declaração -> aceite;
- owner vê estados e histórico sem carregar contratos em N+1;
- loading, vazio, erro, conflito, sessão expirada e retry são recuperáveis;
- coach/platform admin/capability falsa não veem a superfície.

### Dados, concorrência e segurança

- migration em banco vazio e upgrade desde `059` criam zero contratos;
- segunda execução do migrator aplica zero migrations;
- duas organizações/unidades não leem, inferem ou associam dados entre si;
- FK composta rejeita mistura de pessoa, matrícula e unidade;
- duas criações concorrentes deixam uma instância não descartada;
- duas emissões ou dois aceites concorrentes produzem uma transição/evidência;
- replay igual retorna o resultado; chave reutilizada não executa novamente;
- rollback de evento/auditoria/idempotência desfaz a mutação;
- triggers rejeitam mutação de versão publicada e snapshot/documento;
- token bruto existe somente na resposta inicial, nunca no banco/log;
- token inválido/expirado/usado tem resposta indistinguível;
- Markdown adversário não executa script, HTML ou recurso remoto.

### Fronteiras e efeitos ausentes

- teste arquitetural comprova ausência de queries de contracts nas tabelas de
  memberships, people, students, billing, check-ins e agenda;
- privacy export inclui documento/snapshot/aceite pelo port público;
- retenção do simulador remove dados sintéticos vencidos em dry-run/apply;
- não é criado PAR-Q, cobrança, mensagem, e-mail, reserva, consumo, check-in
  ou decisão de acesso;
- nenhum adapter faz chamada de rede;
- produção e banco compartilhado permanecem inalterados.

## 19. Testes mínimos obrigatórios

### Domínio e aplicação

- normalização de template, validação de conteúdo e matriz de placeholders;
- renderer/hash canônico, incluindo ordem de adicionais, moeda, datas e
  `valid_until` nulo;
- máquina de estados e revisões válidas/inválidas;
- preview alterado, matrícula/pessoa inelegível e template substituído;
- idempotência igual/diferente em todos os comandos críticos;
- clock/timezone determinísticos para sessão e aceite;
- port de memberships e people com adversários de tenant;
- port local de aceite, expiração, rotação de sessão e prova minimizada;
- falha injetada em evento/auditoria/command request com rollback total.

### PostgreSQL, migration e HTTP

- migrations completas em PostgreSQL temporário vazio;
- segundo migrator com zero aplicações;
- upgrade real desde `059` com pessoas, planos, adicionais e matrículas,
  comprovando zero backfill;
- constraints, FKs compostas, índices parciais e triggers de imutabilidade;
- corridas reais de criação, emissão, sessão e aceite;
- owner permitido; coach/platform admin bloqueados;
- capabilities, startup inválido em production, CSRF e rate limit público;
- erros seguros, `no-store`, token/PII ausentes em logs e responses adversárias;
- privacy export, anonymization guard e retenção do simulador.

### Frontend

- capabilities ausentes/falsas escondem menu, seção e rota;
- editor mínimo, chips de placeholders, preview sintético e publicação;
- painel da matrícula com preview, hash alterado, draft, emissão e histórico;
- simulador local identificado como não produtivo;
- token não vai para storage/analytics;
- Playwright em `360x800`, foco, Escape, teclado e ausência de overflow;
- regressão completa, TypeScript, build Vite e `git diff --check`.

Baseline adicional: `go test ./...`, `go vet ./...` e race detector em Linux
com PostgreSQL real.

## 20. Decisões da revisão para `ready`

Produto, arquitetura e privacidade aprovaram em 2026-08-24:

1. `{{member_name}}` é suficiente no `placeholder_schema_version=1` para a
   homologação local. CPF, endereço, e-mail e telefone ficam fora da slice e
   não podem ser obtidos por atalho no módulo `people`.
2. Título aceita de 1 a 160 caracteres, corpo Markdown de 1 a 50.000 e
   declaração de aceite de 10 a 1.000. HTML bruto, scripts, iframes, imagens
   remotas e URLs executáveis são proibidos; renderer, sanitização e hash
   continuam versionados e determinísticos.
3. O owner pode criar o conteúdo contratual. O EngageFit não fornece,
   semeia, valida ou declara qualquer template juridicamente adequado. Essa
   limitação precisa permanecer explícita na experiência e na documentação.
4. Cada matrícula possui no máximo uma instância não descartada de
   `membership_terms` nesta fundação, garantida no PostgreSQL e na aplicação.

Essas decisões fecham a fundação local sem ampliar PII, efeitos externos ou
semântica jurídica. Não foi identificada divergência com `ARCH-001`,
`SPEC-002/A` ou `SPEC-003/A-B-C`.

A revisão das fronteiras confirmou:

- `contracts` é dono de templates, versões, instâncias, sessões, aceites e
  eventos, sem consulta direta às tabelas colaboradoras;
- `memberships` expõe por port o snapshot exato e imutável da matrícula, sem
  permitir que contrato altere vigência, estado ou composição;
- `people` expõe por port somente nome de exibição e elegibilidade no schema
  1, sem CPF ou dados de contato;
- `privacy` consome leitura pública de `contracts`; retenção produtiva e
  resposta a eliminação continuam bloqueadas para a `SPEC-004/B`, enquanto o
  simulador usa dados sintéticos com purge curto;
- `healthforms` continua dono exclusivo de PAR-Q e dados de saúde; esta slice
  não cria tabela, DTO, permissão ou retenção compartilhada com contratos.

### Continuam abertas e bloqueiam produção/`SPEC-004/B`

5. Campos de qualificação legal das partes e sua fonte de verdade.
6. Provedor/método de assinatura, autenticação do signatário, evidências,
   webhook, reconciliação e entrega de cópia.
7. Base, prazo e marco inicial de retenção de contrato produtivo, legal hold e
   resposta a anonimização/eliminação.
8. Se aceite continua permitido quando a matrícula termina depois da emissão;
   A escolhe bloquear de forma conservadora até revisão jurídica.
9. Canal futuro de entrega: PWA, link copiado, provedor, e-mail ou WhatsApp.
10. Termos de aditivo, pausa, trancamento, renovação, distrato e cancelamento.

## 21. Fatiamento posterior da SPEC-004

```text
SPEC-004/A — Fundação de contratos e emissão vinculada à matrícula
  template/versionamento + snapshot + estados + port/simulador local

SPEC-004/B — Aceite eletrônico produtivo e experiência do signatário
  qualificação das partes + provedor real + webhook/reconciliação + cópia

SPEC-004/C — PAR-Q e formulários sensíveis
  módulo healthforms, autorização específica, consentimento e retenção própria

SPEC-004/D — Ciclo jurídico posterior à emissão
  aditivos, renovação, pausa/trancamento e cancelamento, coordenado com 005
```

B depende das portas de A. A slice C pode ser detalhada em paralelo sobre
`people` e `memberships`, pois pertence ao módulo independente `healthforms`;
eventual jornada conjunta não autoriza compartilhar tabelas ou retenção. D
depende das decisões jurídicas de B e dos contratos financeiros de
`SPEC-005`.

## 22. Definition of Done da preparação documental

- spec revisada e promovida a `ready`, sem afirmar implementação;
- quatro decisões da fundação local registradas sem divergência
  arquitetural;
- catálogo, README e rastreabilidade de `REQ-001` atualizados;
- fronteiras com `memberships`, `people`, `healthforms`, `memberbilling`,
  agenda e acesso explícitas;
- modelo, contratos, estados, erros, privacidade, rollout, critérios e testes
  descritos;
- prompt de implementação separado e aplicável à versão `ready`;
- nenhuma alteração em `docs/system-design.md`, código, migration, endpoint,
  banco, flag ou ambiente externo.
