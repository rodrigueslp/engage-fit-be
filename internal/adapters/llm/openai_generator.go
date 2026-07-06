package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"boxengage/backend/internal/ports/services"
)

type OpenAIGenerator struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIGenerator(apiKey, model string, timeoutSeconds int) OpenAIGenerator {
	if model == "" {
		model = "gpt-4.1-mini"
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	return OpenAIGenerator{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
	}
}

func (g OpenAIGenerator) GenerateWorkoutMessage(ctx context.Context, input services.WorkoutMessageGenerationInput) (*services.WorkoutMessageGenerationOutput, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return nil, errors.New("OPENAI_API_KEY is not configured")
	}

	systemPrompt := strings.Join([]string{
		"Voce gera mensagens curtas de WhatsApp para alunos de uma academia/box.",
		"Escreva em portugues do Brasil, tom pratico, proximo e seguro.",
		"A mensagem deve fazer o aluno se sentir visto: comece usando exatamente o placeholder {{first_name}} em uma saudacao natural.",
		"Nao apenas resuma o treino; acrescente um diferencial util para o aluno, como musculaturas envolvidas, foco tecnico, intencao do estimulo ou uma dica concreta de execucao.",
		"Nao prescreva dieta individual, nao de orientacao medica e nao prometa resultado fisico.",
		"Use dicas tecnicas gerais, recomende falar com o coach em caso de dor, duvida de carga ou necessidade de adaptacao.",
		"Responda apenas com a mensagem final, sem titulo, markdown ou explicacoes.",
	}, " ")

	workoutText := strings.TrimSpace(input.RawText)
	if workoutText == "" {
		workoutText = strings.TrimSpace(strings.Join([]string{input.Title, input.Goal, input.Movements, input.CoachNotes}, "\n"))
	}

	userPrompt := fmt.Sprintf(
		"Academia: %s\nData: %s\nTexto bruto do treino enviado pelo coach:\n%s\nPublico: %s\nTarefa: interprete o treino inteiro e gere uma mensagem de WhatsApp que va alem do texto original. Inclua: 1) uma saudacao com {{first_name}}, 2) o foco do treino em linguagem simples, 3) uma ou duas musculaturas/capacidades trabalhadas quando fizer sentido, 4) uma dica pratica para executar melhor ou com mais seguranca.\nLimite: ate 750 caracteres.",
		input.BoxName,
		input.Date,
		workoutText,
		input.Audience,
	)

	payload := map[string]any{
		"model": g.model,
		"input": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_output_tokens": 280,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai request failed: status=%d body=%s", resp.StatusCode, string(responseBody))
	}

	text := responseText(responseBody)
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("openai response did not include generated text")
	}

	return &services.WorkoutMessageGenerationOutput{Provider: "openai", Model: g.model, Body: strings.TrimSpace(text)}, nil
}

func responseText(body []byte) string {
	var parsed struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	if parsed.OutputText != "" {
		return parsed.OutputText
	}
	for _, output := range parsed.Output {
		for _, content := range output.Content {
			if content.Text != "" {
				return content.Text
			}
		}
	}
	return ""
}
