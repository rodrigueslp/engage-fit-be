# SPEC-005 — Financeiro do aluno

- Estado: `validated locally`
- Última atualização: 2026-09-16
- Dependências: `SPEC-002`, `SPEC-003/A-B-C`, contrato interno de `SPEC-004/A`
- Módulo proprietário: `memberbilling`

## 1. Resultado

O owner consegue iniciar e acompanhar o financeiro de uma matrícula usando o
preço e o ciclo já congelados nela. O sistema gera cobrança única ou recorrente,
oferece checkout simulado por Pix, boleto ou cartão, registra pagamentos manuais
por Pix, dinheiro ou maquininha, concilia liquidações do provedor e registra
estornos em um ledger imutável.

Cadastro, matrícula, contrato, pagamento e acesso continuam estados
independentes. A implementação não consulta nem reutiliza entidades do módulo
`billing`, que cobra o EngageFit das academias.

## 2. Decisões executadas

- um contrato financeiro pertence a uma matrícula e preserva pessoa, unidade,
  ciclo, preço, moeda e meio de pagamento;
- existe no máximo um contrato financeiro por matrícula;
- a primeira cobrança nasce com o contrato; contratos recorrentes mantêm
  `next_due_on` e materializam parcelas subsequentes explicitamente;
- vencimento não é persistido como mudança destrutiva: a leitura projeta
  `overdue` sobre uma cobrança pendente cuja data passou;
- pagamentos e estornos são registros append-only em `member_payments`; totais
  projetados na cobrança têm constraints contra pagamento acima do total e
  estorno acima do valor pago;
- baixa manual exige valor, forma, instante, responsável autenticado e motivo;
- evento do provedor é único por organização e a reconciliação também exige
  `Idempotency-Key`;
- advisory locks serializam a mesma chave, a materialização do contrato e as
  mutações de uma cobrança; duas baixas concorrentes não ultrapassam o saldo;
- dados de checkout não contêm cartão ou credencial. O simulador devolve somente
  instruções inequivocamente não pagáveis;
- rotas exigem owner, unidade da sessão, tenancy composto e capability
  `member_billing`, desligada por padrão;
- exportação do titular inclui contratos financeiros, cobranças e ledger.

## 3. Fluxos

### 3.1 Iniciar financeiro

1. owner abre `Financeiro` dentro da matrícula;
2. escolhe Pix, boleto ou cartão e o primeiro vencimento;
3. backend resolve a matrícula pelo serviço público de `memberships`, sem ler
   suas tabelas diretamente;
4. contrato e primeira cobrança são persistidos na mesma transação;
5. o adapter local gera referência e instruções simuladas;
6. repetição da chave retorna o mesmo resultado; payload diferente conflita.

### 3.2 Recorrência

O owner materializa a próxima cobrança indicada por `next_due_on`. O intervalo
vem do ciclo mensal, trimestral, semestral ou anual da matrícula; fim de mês é
limitado ao último dia válido, sem deriva para o mês seguinte. Pagamento único
não possui próxima data.

### 3.3 Liquidação e estorno

- baixa manual: `pix`, `cash` ou `card_machine`, inclusive parcial;
- conciliação: evento externo estável liquida total ou parcialmente sem duplicar;
- estorno: cria lançamento inverso e nunca excede o líquido pago;
- cobrança cancelada ou valor acima do saldo recusa a transição.

## 4. Persistência

Migration `068_create_member_billing.sql`:

- `member_billing_contracts` — snapshot financeiro por matrícula;
- `member_charges` — parcelas, vencimento, projeção e referência do provedor;
- `member_payments` — ledger de baixas, liquidações e estornos;
- `member_billing_command_requests` — idempotência por organização e comando.

Todas as referências de organização/unidade/pessoa/matrícula usam FKs simples
ou compostas. Não há PII duplicada nas novas tabelas.

## 5. HTTP

| Método e rota | Resultado |
|---|---|
| `GET /api/v1/units/:unit_id/enrollments/:enrollment_id/member-billing` | resumo, contrato, cobranças e pagamentos |
| `POST .../enrollments/:enrollment_id/member-billing` | cria contrato e primeira cobrança |
| `POST .../member-billing/contracts/:contract_id/materialize` | gera próxima parcela recorrente |
| `POST .../member-billing/charges/:charge_id/manual-payments` | baixa manual auditável |
| `POST .../member-billing/charges/:charge_id/reconcile` | concilia evento do provedor |
| `POST .../member-billing/charges/:charge_id/refunds` | registra estorno |

Todas as mutações exigem `Idempotency-Key` entre 8 e 128 caracteres seguros.
Escopo adversário retorna `404` sem revelar recursos de outra unidade.

## 6. UX

O painel mobile mostra situação geral, total aberto, pago e estornado, parcelas,
vencimentos e o ledger append-only de cada parcela. Baixa manual, conciliação
simulada e estorno aceitam valor parcial e mostram o limite disponível antes da
confirmação. A configuração inicial explicita que o gateway é um simulador. As
operações ficam agrupadas dentro da parcela e a interface não pede nem armazena
dados de cartão.

Em desenvolvimento local, após uma mutação o painel oferece uma ação explícita
de homologação que reenvia a mesma chave e o mesmo payload. A própria tela
compara contrato, totais, parcelas e IDs do ledger e só confirma a idempotência
quando nada foi duplicado. Esse controle técnico não integra o build produtivo.

## 7. Critérios validados localmente

- [x] migration completa em PostgreSQL vazio e segundo run sem aplicação;
- [x] separação estrutural entre `memberbilling` e billing SaaS;
- [x] geração única e recorrente com preço/ciclo da matrícula;
- [x] Pix, boleto e cartão exercitados no adapter local sem rede;
- [x] baixa manual, conciliação, parcialidade e estorno protegidos por regras;
- [x] idempotência e capability específicas;
- [x] tenancy composto e RBAC owner;
- [x] export de privacidade sem nova PII;
- [x] painel validado em `360x800` por Playwright e revisão visual, incluindo
  valores parciais, ledger, conciliação, estorno e reenvio idempotente;
- [x] suíte Go, typecheck e build do frontend.

## 8. Fora do escopo e rollout posterior

- credenciais, conta conectada, split ou repasse real da academia;
- webhook público assinado de um gateway escolhido;
- captura de cartão no EngageFit, PCI, antifraude e chargeback;
- nota fiscal, contabilidade, comissão, caixa/estoque do quiosque;
- bloqueio automático de acesso por inadimplência;
- pausa, multa, cancelamento jurídico ou renovação contratual da `SPEC-004/D`;
- exposição no app do aluno, pertencente à `SPEC-010`.

Até a escolha e homologação do gateway real, `FEATURE_MEMBER_BILLING_ENABLED`
permanece desligada em produção. O simulador é a única implementação instalada e
não deve ser tratado como comprovante de pagamento externo.
