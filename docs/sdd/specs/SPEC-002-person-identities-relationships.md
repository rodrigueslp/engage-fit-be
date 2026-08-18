# SPEC-002 — Pessoa, identidades e vínculos

- Status: `reviewed`
- Versão: `0.3`
- Data: 2026-08-18
- Depende de: `ARCH-001`, `SPEC-001/A-B`
- Bloqueia: planos/matrículas, contratos, cobrança do aluno e histórico unificado

## Estratégia de entrega

Esta é a spec-guia da iniciativa. Ela foi dividida para não misturar cadastro,
migração de leitura e consolidação de identidades na mesma entrega:

- [`SPEC-002/A`](SPEC-002-A-person-foundation-manual-registration.md) —
  fundação, backfill um-para-um e cadastro manual compatível; estado
  `validated locally`;
- `SPEC-002/B` — perfil unificado, edição e migração gradual de leituras;
- `SPEC-002/C` — identidades externas, revisão e consolidação segura.

Somente a slice A está autorizada para implementação. Os contratos amplos
abaixo orientam as próximas slices, mas permanecem sujeitos a detalhamento.

## 1. Resultado esperado

O owner cadastra uma pessoa em poucos segundos usando nome, WhatsApp e CPF e
pode complementar e-mail e endereço sem bloquear a entrada inicial. O EngageFit
mantém uma visão humana única mesmo quando essa pessoa frequenta mais de uma
unidade ou também usa Wellhub/TotalPass, sem realizar associações silenciosas e
inseguras.

## 2. Problema

**Existente:** `Student` pertence a um box e possui uma única `source`. A mesma
pessoa em Wellhub e TotalPass é intencionalmente representada por dois
registros. `AthleteAccount` é separado e ligado aos students por convite.

**Existente:** imports corporativos podem não conter telefone, CPF ou
identificador estável. O sistema já evita merge entre origens e exige revisão
em casos ambíguos.

**Benchmark:** sistemas concorrentes concentram contratos, financeiro, acesso
e treino no perfil do aluno, mas frequentemente apresentam cadastro amplo e
estados derivados de várias configurações.

**Inferência:** `Person` será a identidade humana; origem, participação na
unidade, conta de acesso e matrícula serão vínculos separados.

## 3. Escopo

### Incluído

- pessoa em escopo de organização;
- cadastro local compacto com nome, WhatsApp e CPF;
- e-mail e endereço opcionais por abertura progressiva;
- participação da pessoa em uma ou mais unidades;
- associação com registros atuais de `Student`;
- múltiplas identidades externas;
- ligação futura/compatível com `AthleteAccount`;
- busca por nome, telefone e CPF;
- detecção e revisão de possíveis duplicidades;
- histórico resumido e ações principais no perfil;
- migração segura dos dados atuais.

### Fora do escopo

- CRM completo de leads;
- responsáveis, dependentes e menores de idade;
- prontuário ou ficha médica;
- endereço obrigatório no primeiro cadastro;
- venda de plano e contrato;
- merge automático entre pessoas de organizações distintas;
- biometria e reconhecimento facial.

## 4. Evidências e benchmark

- **Benchmark — Next Fit:** o cadastro antecede venda de plano, treino e
  avaliação; o status do cliente é correlacionado ao contrato.
- **Benchmark — Tecnofit:** o fluxo público observado pode começar por prospect
  e depois converter em cliente, adicionando uma etapa que não é necessária
  para a operação inicial do EngageFit.
- **Inferência EngageFit:** cadastro manual começa como pessoa da unidade sem
  exigir um funil comercial. Lead/CRM poderá ser uma capacidade posterior.
- **Inferência EngageFit:** situação cadastral, situação financeira, matrícula
  e permissão de acesso são estados separados; a UI pode resumi-los sem
  colapsá-los em um único enum ambíguo.

Referências:

- https://ajuda.nextfit.com.br/support/solutions/articles/69000554077-como-cadastrar-um-aluno-cliente-geral-
- https://ajuda.tecnofit.com.br/pt-BR/support/solutions/articles/67000330812-como-cadastrar-um-prospect-e-converter-em-aluno-

## 5. Regras e invariantes

1. `Person` representa um ser humano, não um contrato nem uma origem de
   check-in.
2. CPF normalizado e confirmado é único dentro da organização.
3. CPF nunca é usado como senha e não aparece integralmente em listas/logs.
4. Telefone é normalizado para E.164; possuir telefone não representa opt-in.
5. Mesmo telefone pode gerar alerta, mas não merge automático.
6. Nome nunca é chave global de identidade.
7. Identidade externa é única no escopo definido pelo provedor.
8. Uma pessoa pode participar de várias unidades da mesma organização.
9. Uma pessoa pode possuir várias identidades externas e matrículas.
10. Registros importados sem dados suficientes podem existir incompletos e ser
    enriquecidos posteriormente.
11. Merge exige ação humana auditável, comparação visível e possibilidade de
    impedir operação se houver dependências incompatíveis.
12. Anonimização e exportação percorrem todos os vínculos da pessoa.

## 6. Fluxos do usuário

### Cadastro manual padrão

1. Owner toca em `Novo aluno`.
2. Formulário mostra três campos: nome, WhatsApp e CPF.
3. Telefone e CPF usam máscara visual, mas a API recebe valores normalizados.
4. O sistema verifica CPF exato e possíveis correspondências de telefone.
5. Sem conflito, cria pessoa, participação na unidade e perfil local
   `box_member` na mesma transação.
6. A tela abre o perfil com ações `Vender plano`, `Convidar para o app` e
   `Registrar check-in`; ações indisponíveis explicam o requisito faltante.
7. `Mais informações` permite completar e-mail e endereço agora ou depois; uma
   spec consumidora poderá exigir esses dados antes de contrato, cobrança ou
   emissão específica.

O cadastro manual de mensalista exige os três campos. Fluxos de importação
Wellhub/TotalPass podem criar pessoa incompleta porque a fonte não os fornece.

### CPF já existente na organização

1. O sistema não cria duplicata.
2. Mostra nome mascarado/contexto permitido e pergunta se o owner quer vincular
   a pessoa à unidade atual.
3. O vínculo só é criado se o usuário tiver permissão para ambas as operações.

### Possível duplicidade por telefone

1. O sistema mostra aviso não bloqueante com dados mínimos mascarados.
2. Owner escolhe `Usar pessoa existente` ou `Criar mesmo assim` com motivo.
3. Nome semelhante sem CPF/telefone não provoca merge automático.

### Perfil mobile

O topo mostra nome, situação operacional resumida e três ações contextuais. As
seções seguintes são `Matrícula`, `Financeiro`, `Agenda e presença`, `Contato`
e `Histórico`; somente módulos já disponíveis aparecem.

## 7. Experiência e conteúdo

- Busca e botão `Novo aluno` ficam visíveis no topo.
- Busca aceita nome, telefone ou CPF; resultados mascaram dados sensíveis.
- Listagem mobile usa cards com nome, vínculo/plano e próxima ação, não tabela.
- Campos avançados ficam em `Mais informações` e nenhum é exigido sem uma spec
  consumidora.
- Mensagens usam `pessoa` apenas internamente; na UI operacional pode aparecer
  `aluno` ou `cliente` conforme o contexto.
- O perfil não exibe um único status genérico. Exemplo: `Matrícula ativa`,
  `Pagamento atrasado` e `Acesso bloqueado` são sinais independentes.

## 8. Modelo de dados proposto

### `persons`

- `id`, `organization_id`;
- `name`, `preferred_name` opcional;
- `phone_e164`, `email` opcional;
- `cpf_encrypted`, `cpf_lookup_hash`;
- `profile_status`: `active`, `archived`, `anonymized`;
- timestamps.

CPF deve usar cifra autenticada já adotada pelo projeto. A busca/unique usa
HMAC com chave exclusiva ou mecanismo equivalente resistente a enumeração; não
usar SHA-256 simples devido ao espaço reduzido de CPFs possíveis.

### `person_addresses`

- `id`, `person_id`, `type` inicialmente `home`;
- CEP, logradouro, número, complemento, bairro, cidade, UF e país;
- timestamps e origem/autor da última alteração;
- no máximo um endereço residencial ativo por pessoa na primeira versão.

Endereço não participa de deduplicação e não aparece em listas. A interface
coleta somente os campos necessários e permite cadastro inicial sem endereço.

### `person_unit_relationships`

- `id`, `person_id`, `unit_id`;
- `relationship_status`: `active`, `inactive`, `blocked`;
- `started_at`, `ended_at`, motivo e timestamps;
- unicidade por pessoa/unidade.

Esse status descreve participação operacional, não pagamento ou contrato.

### `person_external_identities`

- `id`, `person_id`, `unit_id` quando o provedor exigir;
- `provider`: inicialmente `wellhub`, `totalpass`;
- `external_id`, metadados mínimos, verificação e timestamps;
- unicidade no escopo do provedor.

### Compatibilidade com `students`

- adicionar `students.person_id` inicialmente opcional;
- cada `student` existente recebe uma pessoa própria no primeiro backfill;
- `athlete_student_links` confirmados podem consolidar deterministicamente
  perfis ligados à mesma conta dentro da organização;
- nenhuma consolidação cross-source é feita apenas por nome;
- `students` continua alimentando check-ins, campanhas e retenção até specs
  posteriores migrarem as leituras.

### Compatibilidade com `athlete_accounts`

Adicionar vínculo opcional e único entre conta e pessoa. Login e sessões
continuam independentes de usuários administrativos. A conta não é criada
automaticamente no cadastro operacional.

## 9. Contratos HTTP propostos

- `GET /api/v1/units/:unit_id/people`
- `POST /api/v1/units/:unit_id/people`
- `GET /api/v1/units/:unit_id/people/:id`
- `PATCH /api/v1/units/:unit_id/people/:id`
- `POST /api/v1/units/:unit_id/people/:id/link`
- `GET /api/v1/units/:unit_id/people/duplicate-candidates`
- `POST /api/v1/people/:id/merge` — somente após regra específica aprovada

Criação aceita `Idempotency-Key`. Respostas de lista nunca retornam CPF
integral nem ciphertext. A busca por CPF é `POST` ou query protegida que não
registra o valor em URL/log.

Erros estáveis incluem:

- `person_cpf_invalid`;
- `person_cpf_conflict`;
- `person_possible_duplicate`;
- `person_unit_already_linked`;
- `person_merge_not_allowed`.

## 10. Autorização, privacidade e auditoria

- owner cria e altera; permissões futuras podem liberar recepção;
- coach consulta somente campos operacionais necessários e não vê CPF;
- enquanto `SPEC-001/C` não existir, toda rota com `unit_id` exige igualdade
  com `users.box_id`; a allowlist multiunidade substituirá essa regra depois;
- platform admin não consulta PII por padrão;
- visualização excepcional de CPF completo exige permissão, motivo e auditoria;
- endereço segue a mesma minimização: coach não o recebe sem necessidade
  operacional específica;
- contato por WhatsApp continua exigindo consentimento explícito;
- criação, alteração de identificador, vínculo e merge geram auditoria;
- exportação reúne pessoa, unidades, perfis de origem, conta, check-ins e dados
  já cobertos pelo fluxo LGPD, incluindo endereços;
- anonimização remove/transforma PII e mantém métricas compatíveis;
- logs usam `person_id` somente quando necessário para suporte controlado e
  nunca telefone, CPF ou nome como labels de métrica.

## 11. Rollout e compatibilidade

1. Criar tabelas novas e coluna opcional em `students`.
2. Backfill um-para-um para evitar merges indevidos.
3. Reconciliar contagens por organização, unidade e origem.
4. Passar novas criações manuais pelo serviço de pessoa, mantendo a resposta
   antiga para consumidores atuais.
5. Migrar telas e consultas gradualmente.
6. Tornar `person_id` obrigatório somente quando todos os caminhos de criação
   estiverem convertidos.

Feature flag sugerida: `FEATURE_PEOPLE_FOUNDATION_ENABLED`.

## 12. Critérios de aceite

- owner cadastra mensalista com nome, WhatsApp e CPF em uma tela mobile;
- owner pode complementar e-mail/endereço sem preencher novamente os dados
  essenciais;
- retry do cadastro não cria duplicata;
- mesmo CPF em outra unidade da organização oferece vínculo, não nova pessoa;
- mesmo CPF em outra organização não vaza existência nem dados;
- telefone repetido alerta e exige decisão, mas não faz merge automático;
- aluno importado sem CPF continua funcional;
- pessoa pode reunir perfis Wellhub e TotalPass somente por confirmação segura;
- campanhas, retenção, check-ins e app do aluno atuais não regridem;
- CPF não aparece em logs, URLs, listas ou exportações não autorizadas;
- endereço não aparece em listas e é removido/anonimizado com a pessoa;
- exportação e anonimização incluem as novas tabelas.

## 13. Testes mínimos

- normalização e validação de CPF/telefone;
- persistência, autorização, exportação e anonimização de endereço;
- cifra e lookup seguro de CPF;
- concorrência de dois cadastros com mesmo CPF;
- isolamento entre organizações e unidades;
- backfill com alunos homônimos e múltiplas origens;
- vínculo confirmado por conta do atleta;
- privacy export, anonymization e suppression;
- API HTTP com idempotência e códigos de conflito;
- Playwright mobile de cadastro, duplicidade e perfil.

## 14. Decisões abertas

- política para menor/dependente sem CPF próprio;
- permissão exata da recepção e visualização parcial de documentos;
- gatilho que torna e-mail/endereço necessário para contrato, pagamento ou
  emissão;
- momento em que o perfil de origem `students` deixa de ser exposto nas APIs;
- UX final de merge manual e reversão operacional.
