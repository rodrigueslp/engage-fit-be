# SPEC-013 — Quiosque e vendas avulsas

- Estado: `validated locally`
- Última atualização: 2026-09-17
- Dependências: `SPEC-002`, port financeiro neutro derivado da `SPEC-005`
- Módulo proprietário: `commerce`

## 1. Resultado

O owner mantém um catálogo simples por unidade e registra uma venda de balcão
paga por Pix, dinheiro ou maquininha. A venda congela nome, SKU, quantidade e
preço dos produtos, soma o total no servidor e guarda um pagamento manual
auditável. Uma pessoa ativa da unidade pode ser vinculada sem duplicar PII; a
venda também pode permanecer anônima.

## 2. Decisões executadas

- produto pertence à unidade, possui preço positivo e pode ser desativado sem
  apagar o histórico;
- itens repetidos são consolidados e o preço enviado pelo cliente é ignorado;
- venda, itens e pagamento único são persistidos atomicamente, e constraints
  diferidas exigem reconciliação exata entre os três totais;
- o comando de venda exige `Idempotency-Key`, usa advisory lock e rejeita a
  reutilização da chave com outro payload;
- somente owner autorizado na unidade acessa as rotas; escopo adversário não
  revela produto, pessoa ou venda;
- `CommercePaymentRecorder` separa o módulo de `memberbilling`; a implementação
  instalada registra apenas meios manuais e não chama rede;
- vendas ligadas a uma pessoa entram na exportação de privacidade;
- `FEATURE_COMMERCE_ENABLED` fica desligada por padrão e depende da fundação de
  pessoas.

## 3. Persistência e HTTP

Migration `071_create_commerce.sql`: `commerce_products`, `commerce_sales`,
`commerce_sale_items`, `commerce_sale_payments` e
`commerce_command_requests`, todas com escopo composto de organização/unidade.

| Método e rota | Resultado |
|---|---|
| `GET /api/v1/units/:unit_id/commerce/products` | catálogo da unidade |
| `POST /api/v1/units/:unit_id/commerce/products` | cria produto |
| `PATCH /api/v1/units/:unit_id/commerce/products/:id` | edita preço, SKU e situação |
| `GET /api/v1/units/:unit_id/commerce/sales` | vendas recentes |
| `POST /api/v1/units/:unit_id/commerce/sales` | registra venda e pagamento manual |

## 4. UX

A tela mobile `Quiosque` reúne cadastro, catálogo, quantidades da venda atual,
vínculo opcional por busca de pessoa, meio de pagamento, total e histórico.
Ela explicita que não movimenta estoque nem emite documento fiscal.

## 5. Critérios validados localmente

- [x] cálculo no servidor e snapshot de preço exercitados por teste unitário;
- [x] idempotência e chave reutilizada com payload divergente cobertas;
- [x] migration `001..071`, segundo run, isolamento e totais reconciliados em PostgreSQL;
- [x] capability, owner-only, tenancy composto e exportação de privacidade;
- [x] TypeScript, build e Playwright mobile focado;
- [x] suítes backend e frontend completas.

## 6. Fora do escopo

Estoque e inventário, fornecedores, compras, comissão, caixa com abertura e
fechamento, desconto, parcelamento, estorno, emissão fiscal, gateway, entrega e
e-commerce. Uma evolução dessas áreas exige novo recorte e regras próprias.
