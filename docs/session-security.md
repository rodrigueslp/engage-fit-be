# Seguranca de sessao e navegador

## Decisao

O navegador autentica com um cookie de sessao `HttpOnly`. O JWT continua sendo o formato assinado da sessao, mas deixa de ser persistido ou lido pelo JavaScript. O header Bearer permanece aceito para scripts operacionais e testes que nao executam em um navegador.

Requisicoes autenticadas por cookie que alteram estado exigem um token CSRF enviado em cookie legivel pelo navegador e no header `X-CSRF-Token`. A comparacao deve ser constante. Requisicoes Bearer nao usam CSRF porque a credencial nao e anexada automaticamente pelo navegador.

CORS usa allowlist explicita. Credenciais so sao permitidas para uma origem listada; curingas nao sao aceitos com cookies. Em desenvolvimento, o proxy Vite mantem frontend e API na mesma origem aparente.

## Cookies

- Sessao: `HttpOnly`, `Path=/`, `SameSite=Lax` por padrao e `Secure` obrigatorio em production.
- CSRF: nao `HttpOnly`, mesmo `Path`, `SameSite` e `Secure` da sessao.
- Logout, troca de senha e sessao invalida removem ambos no navegador.
- O backend continua validando `auth_version` no PostgreSQL em toda requisicao.

`SameSite=None` somente deve ser usado quando frontend e API forem realmente cross-site; nesse modo `Secure` e obrigatorio. A preferencia para o Railway e manter ambos sob o mesmo site, ainda que em subdominios diferentes.

## Modelo de ameaca

O cookie `HttpOnly` reduz o impacto de XSS porque o token nao pode ser lido diretamente nem permanece em `localStorage`. Ele nao elimina a necessidade de CSP e de evitar injecao de HTML/script. O token CSRF impede que outro site force operacoes usando cookies anexados automaticamente. A allowlist CORS impede leitura e chamadas com credenciais a partir de origens nao autorizadas.

Bearer tokens devem ser tratados como segredos de curta duracao por scripts. Nunca devem aparecer em logs, URLs ou arquivos versionados.

## Divisao com a infraestrutura

A aplicacao define CSP e headers seguros que independem de TLS. HSTS somente deve ser emitido em production sobre HTTPS e depois da confirmacao de que o dominio e todos os subdominios relevantes funcionam exclusivamente com TLS. Railway/proxy deve encaminhar `X-Forwarded-Proto` apenas atraves dos proxies configurados como confiaveis.

## Evidencia exigida

- login cria cookies com atributos esperados;
- `/auth/me` funciona sem `localStorage`;
- operacao mutavel por cookie sem CSRF retorna `403`;
- CSRF valido permite a operacao;
- Bearer continua funcionando para os scripts;
- logout e troca de senha revogam a sessao;
- origem fora da allowlist nao recebe CORS;
- CSP e headers de seguranca sao verificados por teste automatizado.
