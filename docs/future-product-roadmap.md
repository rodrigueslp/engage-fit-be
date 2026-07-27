# EngageFit — visão futura de produto e retenção

Documento de visão e exploração de produto. Não representa funcionalidades já
implementadas nem compromisso de prazo. Serve para orientar conversas com
clientes, descoberta de necessidades e priorização depois da validação inicial
com os primeiros boxes.

Atualizado em: 2026-07-25

## 1. Direção estratégica

O EngageFit não precisa se tornar outro sistema completo de gestão de academia.
Seu território mais forte é ser a camada de inteligência e operação de retenção
que usa os dados dos sistemas já existentes no box.

Posicionamento sugerido:

> O EngageFit ajuda o box a perceber cedo quem está se afastando, agir no
> momento certo e medir quais ações realmente fazem os alunos voltarem.

O produto atual já fecha o primeiro ciclo:

```text
check-ins -> identificação de comportamento -> campanha/meta -> comunicação -> incentivo
```

A evolução desejada é fechar um ciclo de retenção mensurável:

```text
detectar risco -> recomendar ação -> executar contato -> acompanhar retorno -> provar o resultado
```

O resultado que deve ser vendido não é quantidade de mensagens ou dashboards,
mas frequência, hábito, relacionamento e receita recorrente protegida.

## 2. Oportunidades de maior valor

### 2.1 Radar de retenção

Evoluir o risco atual, hoje baseado principalmente em dias desde o último
check-in, para uma visão relativa ao comportamento individual do aluno:

- frequência habitual e regularidade semanal;
- queda em relação ao próprio histórico;
- tempo de matrícula;
- resposta a mensagens e campanhas;
- retorno depois de contatos anteriores;
- proximidade de renovação, quando houver integração financeira.

O primeiro modelo deve ser explicável e baseado em regras: saudável, atenção,
em risco, crítico e recuperado. Posteriormente, modelos preditivos podem
estimar probabilidade de afastamento, sempre explicando os sinais que levaram à
classificação.

Exemplo de explicação desejável: “Risco alto porque a frequência caiu 60% nas
últimas três semanas e o aluno está há nove dias sem treinar.”

### 2.2 Jornadas do ciclo de vida

Criar jornadas contínuas, além de campanhas isoladas:

- primeiros 7, 14 e 30 dias;
- incentivo à segunda presença depois da aula inicial;
- reconhecimento de marcos e constância;
- queda de frequência;
- renovação e falha de pagamento;
- cancelamento solicitado;
- reativação de ex-alunos.

Uma jornada pode combinar alerta, mensagem, tarefa humana, registro do motivo e
verificação de retorno. O aluno não deve receber a mesma abordagem em todos os
casos: lesão, viagem, dificuldade financeira, horário e insatisfação exigem
tratamentos diferentes.

### 2.3 Central diária de ações para a equipe

Transformar indicadores em uma fila operacional de “quem precisa de atenção
hoje e por quê”, com atribuição a coach ou funcionário:

- aluno novo sem segunda presença;
- queda acentuada de frequência;
- resposta que exige intervenção humana;
- aluno que voltou e merece reconhecimento;
- renovação próxima;
- caso crítico sem contato realizado.

Cada item pode permitir ligar, conversar presencialmente, enviar mensagem,
registrar motivo, adiar ou concluir acompanhamento. Isso aproxima o produto da
rotina real do box.

### 2.4 Medição de retorno e ROI

O sistema deve evoluir de “mensagens enviadas” para “resultado da intervenção”:

- quantos alunos receberam uma ação;
- quantos voltaram em 3, 7 ou 14 dias;
- qual mensagem, horário ou jornada teve melhor retorno;
- quantos alunos em risco foram recuperados;
- custo por aluno recuperado;
- receita mensal potencialmente protegida;
- desempenho por unidade, equipe ou coach.

Quando houver volume suficiente, testes A/B e grupos de controle podem ajudar a
separar correlação de efeito real. A comunicação comercial deve ser cuidadosa:
uma volta depois da mensagem é evidência de resultado, mas não prova sozinha
que a mensagem foi a causa.

### 2.5 Feedback e motivos de afastamento

Perguntas curtas podem revelar se o problema é horário, lesão, viagem,
financeiro, atendimento, dificuldade técnica, sensação de não pertencimento ou
falta de evolução percebida.

As respostas podem alterar a jornada, pausar cobranças inadequadas e abrir uma
tarefa para a equipe. IA pode classificar e resumir respostas livres, mas casos
de saúde, cancelamento ou reclamação devem chegar a uma pessoa.

### 2.6 Desafios, consistência e comunidade

As campanhas, metas e brindes atuais podem evoluir para temporadas e desafios
de constância:

- metas personalizadas;
- sequências de presença;
- desafios por equipes ou amigos;
- badges e marcos;
- desafios de retorno;
- eventos internos;
- preparação para provas de Hyrox;
- competições entre unidades do mesmo proprietário.

É preferível premiar consistência e evolução individual, e não apenas volume
absoluto de treinos, para não favorecer somente os alunos já mais ativos.

### 2.7 WhatsApp em duas vias

O canal pode evoluir de envio proativo para conversa acompanhada:

- registrar respostas como “estou viajando”, “estou lesionado” ou “quero voltar”;
- pausar mensagens inadequadas;
- abrir tarefa para a equipe;
- responder dúvidas simples;
- encaminhar cancelamentos e casos sensíveis;
- medir a conversão da conversa em retorno.

### 2.8 Indicações

Alunos engajados podem receber convite para indicar amigos no momento adequado:
após uma meta, feedback positivo, retorno consistente ou aniversário de
matrícula. Isso conecta retenção e aquisição sem interromper alunos em risco.

## 3. Onde a IA agrega valor

IA deve economizar tempo ou melhorar decisões, e não apenas produzir textos.

Possibilidades:

- resumo diário dos alunos que merecem atenção;
- explicação dos fatores de risco;
- recomendação da próxima melhor ação;
- geração assistida de campanhas, públicos e textos;
- classificação de motivos de ausência;
- resumo de respostas e conversas;
- detecção de padrões por turma, horário ou unidade;
- identificação de campanhas que geram mensagens, mas pouco retorno;
- previsão de afastamento, com modelo explicável e revisão humana.

Exemplo de resumo futuro:

> Hoje há 14 alunos que merecem atenção: cinco alunos novos sem segunda
> presença, seis com queda superior a 50% e três que responderam mencionando
> incompatibilidade de horário.

IA não deve decidir sozinha sobre lesões, saúde, reclamações graves,
cancelamentos ou mensagens potencialmente sensíveis.

## 4. Integrações que aumentam o valor

Quanto mais rápido o evento chegar ao EngageFit, mais cedo a equipe pode agir.
Vale estudar integrações com:

- catracas e controle de acesso;
- sistemas de gestão de boxes;
- agenda e reserva de aulas;
- pagamentos e matrículas;
- plataformas de treino;
- eventos e competições;
- formulários de feedback.

O objetivo não é substituir esses sistemas, mas criar uma visão única de
engajamento, frequência, relacionamento e risco.

## 5. Roadmap conceitual

### Fundamentos de maior retorno

1. Entrada automática ou mais frequente de check-ins.
2. Radar de retenção baseado em mudança de comportamento.
3. Jornadas de primeiros 30 dias e queda de frequência.
4. Central diária de ações para a equipe.
5. Medição de retorno após cada intervenção.

### Expansão de engajamento

6. Feedback e motivos de ausência.
7. Renovação e reativação.
8. Desafios de consistência e comunidade.
9. Indicações.
10. WhatsApp em duas vias.

### IA depois de existir histórico suficiente

11. Score preditivo e explicável.
12. Próxima melhor ação.
13. Copiloto de campanhas.
14. Classificação e resumo de respostas.
15. Descoberta de padrões operacionais.

Não é necessário implementar essa lista agora. Os primeiros três boxes devem
ser usados para aprender quais sinais antecedem afastamento, quais ações a
equipe realmente executa e quais resultados podem ser medidos.

## 6. Métricas para aprender no piloto

Além de funcionamento e usabilidade, observar:

- frequência média por aluno;
- alunos que entram em risco;
- alunos que voltam sem intervenção;
- alunos que voltam depois de mensagem ou contato humano;
- tempo entre alerta e retorno;
- resposta por tipo de mensagem;
- falsos positivos do alerta;
- motivos de ausência;
- esforço semanal da equipe;
- cancelamentos, quando o box puder fornecer o dado;
- comportamento de alunos novos em 7, 14 e 30 dias;
- diferenças entre CrossFit e Hyrox.

Perguntas úteis para os donos e coaches:

- Quem vocês gostariam que o sistema tivesse avisado antes?
- Qual informação faltou para abordar esse aluno?
- Qual alerta foi ignorado e por quê?
- Que ação foi tomada fora da plataforma?
- Como vocês sabem hoje que um aluno está prestes a sair?

## 7. Frases para conversas comerciais

Estas frases descrevem direção, sem prometer data ou funcionalidade pronta:

> Hoje já usamos a frequência para criar campanhas e contatos direcionados.
> Nossa evolução é detectar a queda de engajamento antes que ela vire
> cancelamento.

> Estamos estudando jornadas específicas para os primeiros 30 dias do aluno,
> quando o hábito ainda está sendo formado.

> Queremos que o sistema não apenas mostre indicadores, mas diga para a equipe
> quem precisa de atenção hoje e acompanhe se essa pessoa voltou.

> No futuro, a plataforma deve mostrar não só quantas mensagens foram enviadas,
> mas quantos alunos retomaram os treinos depois de cada ação.

> A IA poderá ajudar a identificar padrões de afastamento e recomendar a melhor
> abordagem, sempre com a decisão final da equipe.

> A ideia não é substituir o sistema de gestão do box. É usar os dados que ele
> já possui para aumentar frequência, relacionamento e retenção.

## 8. Limites estratégicos

Evitar dispersão prematura em ficha de treino completa, ERP financeiro, folha de
pagamento, estoque amplo, agenda genérica, rede social própria ou aplicativo de
aluno sem uma função clara de retenção. Esses mercados desviam do diferencial e
possuem concorrentes maduros.

O território estratégico do EngageFit é:

> perceber que o aluno está se afastando, ajudar o box a agir e comprovar se a
> ação funcionou.
