package domain

import (
	"fmt"
	"sort"
	"strings"
)

type OfficialWhatsappTemplate struct {
	Type        MessageTemplateType
	Label       string
	Description string
	Body        string
	Audience    MessageAudience
}

var officialWhatsappTemplates = map[MessageTemplateType]OfficialWhatsappTemplate{
	MessageTemplateAlmostThere: {
		Type:        MessageTemplateAlmostThere,
		Label:       "Falta pouco para alcançar a meta",
		Description: "Enviada quando o aluno está próximo de atingir a meta.",
		Audience:    MessageAudienceAlmostThere,
		Body: "Olá, {{student_name}}! Aqui é o EngageFit em nome do {{box_name}}.\n\n" +
			"Falta pouco para você bater sua meta: você já está com {{current_checkins}} check-ins e a meta é {{target_checkins}}.\n\n" +
			"Continue firme. Você está quase lá!",
	},
	MessageTemplateGoalReached: {
		Type:        MessageTemplateGoalReached,
		Label:       "Você atingiu a meta",
		Description: "Enviada quando o aluno bate a meta da campanha.",
		Audience:    MessageAudienceAchieved,
		Body: "Parabéns, {{student_name}}! Aqui é o EngageFit em nome do {{box_name}}.\n\n" +
			"Você atingiu sua meta com {{current_checkins}} check-ins e está participando da recompensa: {{reward_name}}.\n\n" +
			"Mandou muito bem!",
	},
	MessageTemplateWeMissYou: {
		Type:        MessageTemplateWeMissYou,
		Label:       "Sentimos sua falta",
		Description: "Enviada para alunos com baixa frequência ou sem check-ins recentes.",
		Audience:    MessageAudienceInactive,
		Body: "Olá, {{student_name}}! Aqui é o EngageFit em nome do {{box_name}}.\n\n" +
			"Sentimos sua falta nos treinos. Ainda dá tempo de voltar à rotina e buscar sua meta de {{target_checkins}} check-ins.\n\n" +
			"Bora voltar aos treinos?",
	},
}

func OfficialWhatsappTemplates() []OfficialWhatsappTemplate {
	templates := make([]OfficialWhatsappTemplate, 0, len(officialWhatsappTemplates))
	for _, template := range officialWhatsappTemplates {
		templates = append(templates, template)
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Type < templates[j].Type
	})
	return templates
}

func OfficialWhatsappTemplateByType(templateType MessageTemplateType) (OfficialWhatsappTemplate, bool) {
	template, ok := officialWhatsappTemplates[templateType]
	return template, ok
}

func ValidateOfficialWhatsappTemplateType(templateType MessageTemplateType) error {
	if _, ok := OfficialWhatsappTemplateByType(templateType); !ok {
		return fmt.Errorf("unsupported WhatsApp template type %q", templateType)
	}
	return nil
}

func RenderOfficialWhatsappTemplate(templateType MessageTemplateType, values map[string]string) (string, error) {
	template, ok := OfficialWhatsappTemplateByType(templateType)
	if !ok {
		return "", fmt.Errorf("unsupported WhatsApp template type %q", templateType)
	}
	content := template.Body
	for key, value := range values {
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
	}
	return content, nil
}
