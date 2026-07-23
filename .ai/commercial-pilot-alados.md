# Proposta comercial do piloto — CrossFit Alados

Status: rascunho para validação interna
Data: 2026-07-23

## 1. Resumo da oferta

O CrossFit Alados participará do piloto comercial assistido do EngageFit por
90 dias.

Valor:

- R$ 297 por mês;
- uma única mensalidade;
- sem taxa de implantação;
- até 300 mensagens de WhatsApp por ciclo mensal incluídas;
- sem cobrança automática de excedentes.

O piloto começa somente depois da homologação do primeiro envio real pelo
WhatsApp. Testes técnicos anteriores à homologação não iniciam a cobrança nem
consomem a franquia comercial.

## 2. O que está incluído

- uso do EngageFit por uma unidade do CrossFit Alados;
- um acesso de proprietário;
- importação de check-ins de Wellhub e TotalPass por CSV/XLSX;
- dashboard de frequência e acompanhamento;
- campanhas com metas por plataforma;
- identificação de alunos próximos da meta e alunos em risco;
- controle de brindes e entregas;
- relatórios e exportações disponíveis no produto;
- automações que estiverem homologadas e habilitadas para o piloto;
- conexão dedicada do WhatsApp do Alados;
- até 300 mensagens de WhatsApp por ciclo mensal;
- configuração inicial assistida;
- suporte direto durante o piloto;
- reuniões de acompanhamento nos marcos de 30, 60 e 90 dias.

E-mail e Treino do dia não compõem a promessa comercial inicial enquanto
permanecerem ocultos ou não homologados no produto.

## 3. Regra da franquia de WhatsApp

A mensalidade já inclui a utilização do WhatsApp dentro da franquia. O cliente
não recebe uma segunda cobrança de mensageria.

Para fins da franquia:

- conta cada mensagem de saída aceita sincronamente pelo provedor;
- mensagem bloqueada antes de chegar ao provedor não consome franquia;
- falha síncrona, sem aceitação pelo provedor, libera a reserva;
- mensagem aceita e posteriormente marcada como não entregue continua contando,
  enquanto não existir conciliação confiável do custo final;
- a franquia é renovada a cada ciclo mensal;
- mensagens não utilizadas não acumulam para o mês seguinte;
- não existe cobrança automática por mensagem excedente;
- ao atingir o limite, novos disparos são bloqueados até a renovação do ciclo ou
  uma mudança de plano previamente aceita pelo cliente.

Essa definição acompanha o comportamento conservador já implementado na
governança de mensageria do EngageFit.

## 4. Limites operacionais do piloto

Política inicial recomendada para o Alados:

| Controle | Limite |
|---|---:|
| Mensagens por dia | 100 |
| Mensagens por mês | 300 |
| Destinatários por disparo | 100 |
| Alerta de consumo | 80% |
| Custo estimado por mensagem | USD 0,10 |
| Orçamento estimado por dia | USD 10 |
| Orçamento estimado por mês | USD 30 |
| Timezone | America/Sao_Paulo |

O custo estimado de USD 0,10 é uma reserva conservadora de segurança, não o
valor informado ou cobrado do cliente.

Durante o piloto, qualquer aumento de limite deve ser precedido de revisão do
consumo real da subconta Twilio. A alteração não deve gerar cobrança avulsa
surpresa. Se a franquia se mostrar insuficiente de forma recorrente, a solução é
migrar o cliente para outra mensalidade no ciclo seguinte.

## 5. O que não está incluído

- gestão financeira dos alunos da academia;
- cobrança das mensalidades dos alunos;
- catraca ou controle de acesso;
- prescrição de treinos;
- integração automática com APIs de Wellhub ou TotalPass;
- garantia de entrega de mensagens por WhatsApp;
- campanhas fora das regras, políticas ou aprovações da Meta/Twilio;
- customizações exclusivas não previstas no escopo do piloto;
- mais de uma unidade ou mais de um owner.

## 6. Responsabilidades

### EngageFit

- manter o ambiente e os dados do tenant isolados;
- operar os limites e bloqueios de mensageria;
- acompanhar o consumo da subconta Twilio;
- corrigir falhas do produto dentro do escopo do piloto;
- preservar credenciais e dados conforme os controles técnicos existentes;
- avisar o Alados quando a franquia atingir 80%;
- não executar disparos reais sem a configuração e autorização apropriadas.

### CrossFit Alados

- fornecer arquivos válidos de Wellhub e TotalPass;
- revisar campanhas, audiências e mensagens antes do envio;
- garantir que possui base legal e preferências de contato adequadas para falar
  com seus alunos;
- comunicar pedidos de não contato;
- não utilizar a plataforma para spam ou mensagens incompatíveis com as
  políticas do WhatsApp;
- participar das revisões do piloto e informar problemas e resultados.

As responsabilidades de controlador e operador, bases legais, contrato,
política de privacidade e textos públicos ainda devem passar por validação
jurídica antes do uso comercial definitivo.

## 7. Política de pagamento

- vencimento mensal em data acordada;
- pagamento por cartão recorrente, Pix ou boleto;
- cobrança gerada em uma plataforma externa, inicialmente Asaas;
- lembrete no vencimento e após o atraso;
- tolerância de 7 dias corridos;
- após 7 dias sem pagamento, o acesso e as automações podem ser suspensos;
- o pagamento confirmado permite a reativação;
- atraso ou cancelamento nunca arquiva nem apaga automaticamente os dados.

Durante o piloto, a conciliação da cobrança e eventual suspensão podem ser
operadas manualmente pelo administrador do EngageFit.

## 8. Métricas de sucesso

O piloto será avaliado pelos seguintes indicadores:

- frequência e regularidade das importações;
- quantidade de campanhas operadas;
- quantidade de alunos identificados como próximos da meta;
- quantidade de alunos identificados em risco;
- mensagens autorizadas, bloqueadas, aceitas e com falha;
- consumo mensal e custo real de WhatsApp;
- tempo operacional economizado em relação às planilhas e disparos manuais;
- alunos que voltaram a registrar check-in após uma ação de engajamento;
- brindes pendentes e entregues;
- percepção do owner sobre utilidade, facilidade e confiança no produto.

O EngageFit ainda não possui receita financeira por aluno. Portanto, o piloto
não deve prometer cálculo automático de receita recuperada. A relação entre
mensagem e retorno ao check-in poderá ser apurada manualmente durante os 90
dias e transformada em relatório de conversão posteriormente.

## 9. Revisões do piloto

### Após 30 dias

- conferir estabilidade operacional;
- revisar a qualidade das importações;
- comparar franquia estimada e consumo real;
- registrar dificuldades de uso;
- ajustar rotinas sem ampliar automaticamente o escopo.

### Após 60 dias

- medir retorno de alunos após mensagens;
- revisar campanhas e segmentações mais úteis;
- calcular custo real médio por mensagem;
- estimar suporte necessário por academia.

### Após 90 dias

- consolidar o caso de sucesso;
- decidir continuidade;
- definir mensalidade e franquia comercial definitiva;
- decidir se número dedicado será parte de um plano superior;
- validar se há dados suficientes para criar mais de um plano.

## 10. Conversão depois do piloto

Referência inicial, ainda sujeita aos dados dos 90 dias:

- Plano EngageFit: R$ 397 por mês por unidade;
- até 500 mensagens de WhatsApp por ciclo mensal;
- uma única cobrança;
- bloqueio ao atingir a franquia;
- sem excedente automático;
- condições adicionais para várias unidades, maior volume ou suporte especial.

O Alados poderá receber uma condição de cliente fundador na conversão. Essa
condição deve ser registrada por prazo determinado e não deve impedir futuras
correções quando os custos de Twilio, Meta, infraestrutura ou suporte mudarem.

## 11. Pontos que precisam estar prontos antes do início cobrado

1. Template necessário aprovado na Twilio.
2. Primeiro envio real entregue e auditado.
3. Limites do piloto gravados na política da academia.
4. Automação agendada mantida desligada até homologação explícita.
5. Destinatários e campanhas revisados.
6. Forma de pagamento e vencimento aceitos pelo cliente.
7. Termo de piloto assinado.
8. Rotina mínima de backup e recuperação definida.
9. Responsabilidades de privacidade e comunicação descritas no termo.

## 12. Texto curto para apresentar ao cliente

> O piloto assistido do EngageFit terá duração de 90 dias e mensalidade única de
> R$ 297. O valor inclui a plataforma, implantação assistida, suporte e até 300
> mensagens de WhatsApp por mês. Não haverá cobrança separada ou excedente
> automático: ao atingir a franquia, novos envios ficam pausados até o próximo
> ciclo ou até uma mudança de plano previamente combinada. Durante o piloto,
> acompanharemos frequência, campanhas, retorno de alunos e consumo para definir
> juntos a configuração comercial definitiva.
