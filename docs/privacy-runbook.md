# Privacidade e LGPD - runbook da aplicacao

## Responsabilidades propostas para validacao juridica

- A academia decide finalidade, base legal, alunos e mensagens: papel esperado de controladora.
- EngageFit processa os dados conforme configuracao da academia: papel esperado de operador.
- Esta classificacao, os prazos e os textos de consentimento precisam ser confirmados em contrato e por assessoria juridica antes do go-live.

## Preferencia de contato

Na tela `Alunos`, o owner registra `Autorizado`, `Nao contatar` ou `Nao informado` e a origem do registro. `opted_out` e alunos anonimizados sao excluidos das audiencias de WhatsApp, e-mail e Treino do dia, inclusive de rascunhos criados antes do opt-out.

## Solicitacao do titular

1. Confirme a identidade do solicitante fora do EngageFit.
2. Exporte o JSON em `Alunos > Exportar`; a operacao gera auditoria.
3. Para correcao, ajuste o sistema de origem e reimporte. Preferencia de contato pode ser alterada diretamente.
4. Para exclusao, exporte primeiro se necessario e use `Anonimizar`, informando motivo e confirmando a acao irreversivel.
5. A anonimizacao remove nome, e-mail e telefone do aluno e dos historicos de destinatarios, preserva metricas/check-ins anonimos e cria uma supressao hash para impedir que a mesma identidade seja recriada por importacao.

## Retencao

- destinatarios de WhatsApp/e-mail/Treino: 365 dias;
- logs de geracao LLM: 90 dias;
- execucoes de automacao: 180 dias;
- importacoes e check-ins: 730 dias;
- auditoria de privacidade: 1825 dias;
- supressoes de reimportacao: sem expiracao automatica, para respeitar a exclusao.

Sempre execute primeiro `make privacy-retention-dry-run`. Depois de revisar as contagens, execute `make privacy-retention-apply`. A exclusao ocorre em uma transacao e deve ser agendada externamente apenas apos homologacao.

## Incidente

Preserve evidencias sem copiar PII para tickets, restrinja acessos, identifique academias e categorias afetadas, registre horario/escopo/medidas e acione os responsaveis juridicos. Tokens, credenciais e chaves potencialmente expostos devem ser rotacionados. A decisao de notificacao a titulares/ANPD deve seguir avaliacao juridica e os prazos aplicaveis.
