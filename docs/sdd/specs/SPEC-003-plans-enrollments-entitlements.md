# SPEC-003 — Planos, matrículas e direitos de acesso

- Status: `draft`
- Versão: `0.2`
- Data: 2026-08-18
- Depende de: `ARCH-001`, `SPEC-001/B-D`, `SPEC-002`
- Bloqueia: contratos, cobrança do aluno, agenda, reservas e acesso

## 1. Resultado esperado

O owner cria ofertas comerciais simples, com adicionais opcionais, e matricula
uma pessoa sem precisar configurar dezenas de opções. O sistema sabe quais
modalidades, serviços, unidades, reservas e presenças aquela matrícula permite,
preservando exatamente os termos aceitos mesmo quando o plano comercial mudar
no futuro.

## 2. Problema

**Existente:** `Student.source=box_member` identifica mensalistas, mas não há
plano comercial, contrato, vigência nem regra de acesso. O campo
`membership_started_at` sustenta retenção, não uma matrícula completa.

**Existente:** `billing_plans` representa a cobrança do EngageFit para a
academia e não pode ser reutilizado para planos vendidos aos alunos.

**Benchmark:** Next Fit modela modalidade, duração, preço e diferentes formas
de acesso, incluindo livre, sessões por semana e pacote de aulas. Alterações de
algumas configurações podem impactar contratos vigentes.

**Inferência EngageFit:** plano é um template versionado; matrícula aponta para
uma versão imutável. Alterar uma oferta cria nova versão e não muda direitos já
vendidos sem uma operação explícita de migração.

## 3. Escopo

### Incluído

- catálogo de planos da organização;
- versões imutáveis após primeiro uso;
- preço de referência e recorrência;
- modalidades e unidades elegíveis;
- serviços elegíveis e modo de atendimento quando aplicável;
- acesso livre, sessões por período e pacote de aulas;
- adicionais versionados somados ao plano base;
- matrícula de uma pessoa em uma unidade;
- vigência, pausa e encerramento operacional;
- cálculo central de direito para reservar e fazer check-in;
- resumo no perfil e no app do aluno;
- compatibilidade com mensalistas atuais.

### Fora do escopo

- geração e assinatura do documento contratual;
- cobrança, parcelamento, Pix, boleto ou cartão;
- desconto, cupom, comissão e campanha comercial;
- agenda e consumo concreto de reserva;
- multa e cálculo financeiro de cancelamento;
- venda online self-service;
- planos corporativos negociados com Wellhub/TotalPass.

## 4. Evidências e benchmark

- **Benchmark — Next Fit:** modalidade é diferente de contrato/plano; ofertas
  podem permitir acesso padrão, sessões por semana ou pacote de aulas.
- **Benchmark — Next Fit:** duração, preço, frequência e forma de pagamento são
  configurados na oferta, e o app pode disponibilizar planos para compra.
- **Benchmark — Tecnofit:** contratos representam mensalidades, pacotes e
  recorrência, determinando tempo de frequência e fidelidade.
- **Inferência EngageFit:** a criação padrão mostra somente nome, preço,
  recorrência e modalidades; limites avançados aparecem quando o owner escolhe
  `Limitar frequência`.
- **Inferência EngageFit:** termos usados por matrículas existentes não sofrem
  alteração retroativa silenciosa.
- **Entrada Alados — `REQ-001`:** corrida pode ser plano base com adicionais de
  musculação, Recovery ou outras ofertas; Pilates exige combinações de
  frequência e atendimento em grupo/individual.

Referências:

- https://ajuda.nextfit.com.br/support/solutions/articles/69000813031-como-criar-um-contrato-geral-
- https://ajuda.nextfit.com.br/support/solutions/articles/69000851957-quais-s%C3%A3o-os-impactos-na-altera%C3%A7%C3%A3o-de-contratos-
- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000699531-como-efetuar-vendas-no-cadastro-do-aluno-contrato-servico-e-produto-

## 5. Vocabulário

- **Plano:** identidade comercial reutilizável, por exemplo `CrossFit 3x`.
- **Versão do plano:** conjunto imutável de termos oferecidos em um período.
- **Matrícula (`Enrollment`):** adesão de uma pessoa a uma versão do plano.
- **Direito (`Entitlement`):** regra que autoriza modalidade, reserva ou
  presença.
- **Contrato:** documento jurídico associado à matrícula em spec posterior.
- **Cobrança:** obrigação financeira associada à matrícula em spec posterior.

Na UI brasileira, `plano` e `matrícula` são os termos primários. O backend não
usa `billing_plan` para este contexto.

## 6. Regras e invariantes

1. Plano pertence a uma organização.
2. Versão publicada é imutável depois de referenciada por matrícula.
3. Alterar preço, duração, recorrência ou direitos cria nova versão.
4. Matrícula pertence a uma pessoa e possui unidade principal.
5. Uma matrícula aponta para exatamente uma versão do plano.
6. Direitos podem abranger uma ou mais unidades e modalidades permitidas.
7. Acesso livre não exige reserva; acesso agendado será consumido pela agenda.
8. Sessões por período e pacote são regras diferentes.
9. Pausa preserva histórico e deslocamento de vigência só ocorre por comando
   explícito definido na spec de contratos.
10. Situação financeira não é armazenada no status da matrícula.
11. Decisão de acesso recebe matrícula, direitos, unidade, modalidade e tempo;
    futuras restrições financeiras entram como razão externa explícita.
12. Wellhub/TotalPass não são tratados como planos locais fictícios; suas
    elegibilidades entram por identidades e integrações próprias.
13. Direito referencia modalidade ou serviço por ID estável, nunca texto livre.
14. Adicional disponível pertence a uma versão de plano e possui preço e
    direitos imutáveis depois da primeira matrícula.
15. A matrícula registra explicitamente os adicionais escolhidos; alterar a
    oferta futura não muda essa composição.
16. Preço comercial contratado é a soma determinística do plano base com os
    adicionais selecionados, antes de descontos que pertençam a spec posterior.
17. Atendimento `group` ou `individual` é uma restrição de direito quando
    comercialmente relevante; a sessão concreta pertence à agenda.
18. Renovação recorrente é termo comercial da versão; execução e meio de
    cobrança pertencem a `memberbilling`.

## 7. Tipos de direito iniciais

### `open_access`

Permite acesso à modalidade ou serviço durante a vigência sem exigir reserva.
Pode limitar quantidade de acessos por dia.

### `sessions_per_period`

Permite quantidade de sessões por dia, semana ou mês. A contagem final será
integrada à reserva/presença sem duplicar consumo.

### `class_pack`

Concede saldo total de aulas durante a vigência. Reagendamento, expiração e
estorno de crédito serão fechados com a spec de reservas.

Configuração de horários específicos fica fora da primeira versão. Caso o
Alados demonstre necessidade, será adicionada como política versionada, não
como dezenas de campos obrigatórios no plano básico.

## 8. Fluxos do usuário

### Criar plano simples no celular

1. Owner toca em `Novo plano`.
2. Informa nome, preço, recorrência/duração e seleciona modalidades.
3. Seleciona serviços e adicionais somente quando necessários.
4. O sistema sugere `Acesso livre` ou o padrão mais comum da modalidade.
5. Owner pode abrir `Limites de acesso` para sessões por semana/pacote e modo
   individual/grupo.
6. Uma tela de revisão mostra em linguagem natural:
   `CrossFit, 3 vezes por semana, R$ 189/mês, renovação mensal`.
7. Owner publica o plano.

### Matricular pessoa

1. No perfil, owner toca em `Vender plano`.
2. Escolhe entre poucos planos ativos e vê preço/direitos essenciais.
3. Define início; fim é calculado quando aplicável.
4. Confirma uma matrícula `pending_contract` ou `active`, conforme as features
   de contrato/pagamento disponíveis.
5. O perfil passa a mostrar plano, vigência e próxima pendência.

### Alterar oferta

1. Owner edita um plano já vendido.
2. A interface informa que matrículas atuais manterão os termos.
3. Ao salvar, cria nova versão para vendas futuras.
4. Migração de matrículas existentes será uma ação separada, com preview e
   confirmação, fora da primeira implementação.

## 9. Experiência

- Lista de planos mostra nome, valor, resumo de acesso e matrículas ativas.
- Planos arquivados ficam fora da seleção padrão, mas preservam histórico.
- Formulário usa linguagem operacional, não enums técnicos.
- Resumo textual é produzido por regras determinísticas.
- Campos avançados ficam recolhidos.
- No perfil do aluno, a ação principal depende do estado: `Concluir contrato`,
  `Resolver pagamento`, `Pausar` ou `Renovar`.
- Não mostrar configurações de agenda antes de uma modalidade exigir reserva.

## 10. Modelo de dados proposto

### `membership_plans`

- `id`, `organization_id`, `code`, `status`;
- versão atual publicada;
- timestamps.

### `membership_plan_versions`

- `id`, `plan_id`, `version`;
- `name`, `description` opcional;
- `price_cents`, `currency=BRL`;
- `billing_cycle`: `none`, `monthly`, `quarterly`, `semiannual`, `annual`;
- duração opcional e regra de renovação;
- `status`: `draft`, `published`, `retired`;
- `published_at`, timestamps;
- unicidade `(plan_id, version)`.

Preço é termo comercial de referência. Cobrança real pertence a
`memberbilling`.

### `membership_plan_entitlements`

- `plan_version_id` e exatamente um alvo entre `modality_id` ou `service_id`;
- unidades permitidas ou regra `home_unit_only`;
- `access_type`;
- quantidade, período e limite diário quando aplicáveis;
- `delivery_mode` opcional: `group` ou `individual`;
- `booking_required`;
- configuração versionada e validada por constraint.

### `membership_plan_version_addons`

- `id`, `plan_version_id`, nome e descrição curta;
- `price_delta_cents` não negativo;
- direitos próprios em tabela associativa equivalente aos direitos do plano;
- posição de exibição e timestamps.

Como pertence à versão publicada, o adicional não é editado retroativamente.
Uma nova composição exige nova versão do plano.

### `enrollments`

- `id`, `organization_id`, `person_id`, `home_unit_id`;
- `plan_version_id`;
- `status`: `pending`, `active`, `paused`, `ended`, `cancelled`;
- `starts_on`, `ends_on`, timestamps;
- motivo e autor de transições sensíveis.

Eventos de status ficam em `enrollment_events` para auditoria e reconstrução da
linha do tempo.

### `enrollment_addons`

- `enrollment_id`, `plan_version_addon_id`;
- preço incremental e resumo de termos preservados para auditoria;
- unicidade por matrícula/adicional.

## 11. Serviço de decisão de acesso

Contrato interno inicial:

```text
EvaluateAccess(person, unit, modality, occurred_at, purpose)
  -> allowed | denied
  -> reason_code
  -> enrollment_id
  -> entitlement_id
  -> remaining_usage quando calculável
```

`purpose` começa com `checkin` e `booking`. O serviço não chama gateway
financeiro. Ele recebe bloqueios externos já projetados ou consulta port
específico para evitar acoplamento com Asaas.

Razões estáveis incluem:

- `no_active_enrollment`;
- `outside_validity`;
- `modality_not_included`;
- `unit_not_included`;
- `usage_limit_reached`;
- `enrollment_paused`;
- `external_access_blocked`.

## 12. Contratos HTTP propostos

- `GET/POST /api/v1/membership-plans`
- `GET /api/v1/membership-plans/:id`
- `PATCH /api/v1/membership-plans/:id/draft`
- `POST /api/v1/membership-plans/:id/publish`
- `POST /api/v1/membership-plans/:id/retire`
- `GET /api/v1/units/:unit_id/enrollments`
- `POST /api/v1/units/:unit_id/enrollments`
- `GET /api/v1/units/:unit_id/enrollments/:id`
- `POST /api/v1/units/:unit_id/enrollments/:id/pause`
- `POST /api/v1/units/:unit_id/enrollments/:id/resume`
- `POST /api/v1/units/:unit_id/enrollments/:id/end`
- `POST /api/v1/access-decisions/evaluate` — preferencialmente interno; expor
  somente se catracas/integradores exigirem.

Publicação e matrícula exigem `Idempotency-Key`. Mudança de versão usa controle
otimista para impedir sobrescrita por duas abas/agentes.

A criação da matrícula aceita somente IDs de adicionais pertencentes à versão
selecionada. A resposta retorna preço base, adicionais e total como valores
separados para revisão e auditoria.

## 13. Autorização, privacidade e auditoria

- owner gerencia catálogo e matrículas;
- permissões futuras podem permitir venda pela recepção sem editar plano;
- coach pode consultar direito operacional necessário para aula, não preço ou
  situação financeira;
- publicação, matrícula, pausa e encerramento geram auditoria;
- histórico de termos não pode ser apagado por arquivamento;
- respostas respeitam escopo de organização/unidade e minimização de PII.

## 14. Rollout e compatibilidade

- tabelas usam prefixo/nome inequívoco para não colidir com `billing_plans`;
- mensalistas `box_member` existentes não recebem preço ou termos inventados;
- pode ser criado plano administrativo `Legado — sem plano atribuído`, sem
  valor e sem alterar acesso, apenas se necessário para compatibilidade;
- até a atribuição real, check-in atual continua permitido pela regra legada;
- telemetria mede quantos mensalistas permanecem sem matrícula;
- ativação por unidade usa `FEATURE_MEMBERSHIPS_ENABLED`.

O rollout não pode bloquear acesso de aluno existente por ausência de dados que
o sistema antigo nunca coletou.

## 15. Critérios de aceite

- owner cria e publica plano simples no celular;
- resumo do plano corresponde deterministicamente aos direitos;
- preço e direitos da matrícula correspondem ao plano base mais adicionais;
- pacote de Spinning, Pilates semanal e plano base de corrida com adicional
  podem ser representados sem criar entidades concorrentes;
- primeira matrícula usa versão 1 e continua inalterada após publicação da
  versão 2;
- pessoa com matrícula ativa acessa modalidade/unidade permitida;
- limites negam com razão estável e auditável;
- matrícula pausada não perde histórico;
- coach não acessa preço/financeiro sem permissão;
- plano de outra organização não pode ser consultado ou matriculado;
- mensalistas atuais continuam fazendo check-in durante o rollout;
- `billing_plans` e cobrança SaaS não sofrem alteração funcional.

## 16. Testes mínimos

- validação de combinações de entitlement;
- composição e isolamento de adicionais, incluindo alvo modalidade/serviço;
- imutabilidade e concorrência de versões;
- idempotência de publicação e matrícula;
- isolamento entre organizações e unidades;
- transições válidas/inválidas de matrícula;
- avaliação de acesso por vigência, modalidade, unidade e limite;
- compatibilidade com `box_member` legado;
- API HTTP e PostgreSQL real;
- Playwright mobile de criação de plano e matrícula.

## 17. Decisões abertas

- estado inicial da matrícula antes de contrato/pagamento estarem disponíveis;
- se preço zero representa cortesia ou exige tipo explícito;
- política de matrícula simultânea em mais de um plano;
- se adicionais poderão ser compartilhados entre planos ou sempre pertencerão
  a uma versão específica;
- combinações permitidas, desconto e remoção de adicional durante vigência;
- regra comercial e unidade de cobrança definitiva de Recovery;
- se Pilates individual será serviço separado ou direito com
  `delivery_mode=individual`;
- tratamento definitivo de trancamento e deslocamento de vigência;
- qual contador é fonte de verdade para sessões reservadas versus presenças.
