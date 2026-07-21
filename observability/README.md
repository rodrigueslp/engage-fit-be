# Observabilidade local

Esta stack recebe os três sinais OpenTelemetry da API e os apresenta no Grafana:

- métricas: OpenTelemetry Collector -> Prometheus;
- logs estruturados: OpenTelemetry Collector -> Loki;
- traces distribuídos: OpenTelemetry Collector -> Tempo.

## Executar

```bash
make observability-up
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
DATABASE_URL='postgres://boxengage:boxengage@localhost:5432/boxengage?sslmode=disable' \
go run ./cmd/api
```

Abra `http://localhost:3000` (usuário e senha `admin`) e acesse o painel
`EngageFit / EngageFit - Visão geral`. As portas locais são Grafana `3000`,
Prometheus `9090`, Loki `3100`, Tempo `3200` e OTLP HTTP/gRPC `4318`/`4317`.

Prometheus carrega regras locais de alerta para ausencia do heartbeat da API,
taxa de 5xx, readiness, automacoes falhas/stale, importacoes e gateways. Consulte
`http://localhost:9090/alerts`. Essas regras validam a deteccao local; o destino
de notificacao sera configurado depois no Grafana Cloud/Railway.

O painel tambem apresenta importacoes, automacoes e gateways. As dimensoes sao
somente enums de baixa cardinalidade; IDs de tenant/aluno, telefone, e-mail,
URL concreta e `request_id` nao sao labels de metrica.

O endpoint direto `GET /metrics` é habilitado por padrão somente fora de
produção. Em produção, habilite-o explicitamente e configure
`PROMETHEUS_BEARER_TOKEN` com pelo menos 32 caracteres.

## Railway e Grafana Cloud

No Railway, use as métricas nativas para CPU, RAM, disco e rede. Para sinais da
aplicação, configure `OTEL_ENABLED=true`, o endpoint OTLP HTTP do Grafana Cloud
em `OTEL_EXPORTER_OTLP_ENDPOINT` e a autenticação fornecida pelo Grafana Cloud
em `OTEL_EXPORTER_OTLP_HEADERS`. Não é necessário alterar o código nem expor
`/metrics` publicamente.

O backend não registra corpos, senhas, tokens nem parâmetros SQL. Os logs HTTP
incluem `request_id`, `trace_id` e `span_id`, permitindo navegar entre um erro e
seu trace no Grafana.
