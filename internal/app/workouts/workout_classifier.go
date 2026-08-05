package workouts

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"boxengage/backend/internal/domain"
)

const workoutClassificationVersion = "rules-v1"

var (
	workoutDurationPattern = regexp.MustCompile(`(?i)\b(?:AMRAP|EMOM)\s+(\d{1,3})\s*['’]`)
	leadingMovementPattern = regexp.MustCompile(`^\s*\d+(?:[.,]\d+)?\s+(.+?)\s*$`)
	fazerMovementPattern   = regexp.MustCompile(`(?i)\bfazer\s+(?:\d+(?:[.,]\d+)?\s+)?(.+?)\s*$`)
)

func ClassifyWorkoutText(rawText string) domain.WorkoutClassification {
	rawText = normalizeWorkoutText(rawText)
	classification := domain.WorkoutClassification{
		Version:          workoutClassificationVersion,
		GeneratedBy:      "rules",
		SuggestedTitle:   "Treino do dia",
		Sections:         []domain.WorkoutSection{},
		Formats:          []string{},
		MovementMentions: []string{},
	}
	if rawText == "" {
		return classification
	}

	classification.Sections = splitWorkoutSections(rawText)
	classification.Formats = detectWorkoutFormats(rawText)
	classification.DurationSeconds = detectWorkoutDuration(rawText)
	classification.MovementMentions = detectMovementMentions(classification.Sections)
	classification.SuggestedTitle = suggestedWorkoutTitle(classification)
	return classification
}

func normalizeWorkoutText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	empty := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(cleaned) > 0 && !empty {
				cleaned = append(cleaned, "")
			}
			empty = true
			continue
		}
		cleaned = append(cleaned, line)
		empty = false
	}
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return strings.Join(cleaned, "\n")
}

func splitWorkoutSections(rawText string) []domain.WorkoutSection {
	sections := make([]domain.WorkoutSection, 0, 4)
	current := domain.WorkoutSection{Type: domain.WorkoutSectionOther, Title: "Treino"}
	flush := func() {
		current.Content = strings.TrimSpace(current.Content)
		if current.Content != "" || current.Type != domain.WorkoutSectionOther || len(sections) == 0 {
			sections = append(sections, current)
		}
	}

	for _, line := range strings.Split(rawText, "\n") {
		if sectionType, ok := workoutSectionHeading(line); ok {
			if strings.TrimSpace(current.Content) != "" || current.Type != domain.WorkoutSectionOther {
				flush()
			}
			current = domain.WorkoutSection{Type: sectionType, Title: line}
			continue
		}
		if current.Content != "" {
			current.Content += "\n"
		}
		current.Content += line
	}
	flush()

	if len(sections) > 1 && sections[0].Type == domain.WorkoutSectionOther && sections[0].Content == "" {
		sections = sections[1:]
	}
	return sections
}

func workoutSectionHeading(line string) (domain.WorkoutSectionType, bool) {
	if len([]rune(strings.TrimSpace(line))) > 80 {
		return "", false
	}
	folded := foldWorkoutText(line)
	switch {
	case headingStartsWith(folded, "WARM UP"), headingStartsWith(folded, "WARMUP"), headingStartsWith(folded, "AQUECIMENTO"):
		return domain.WorkoutSectionWarmup, true
	case headingStartsWith(folded, "SKILL"), headingStartsWith(folded, "TECNICA"):
		return domain.WorkoutSectionSkill, true
	case headingStartsWith(folded, "STRENGTH"), headingStartsWith(folded, "FORCA"):
		return domain.WorkoutSectionStrength, true
	case headingStartsWith(folded, "WORKOUT OF THE DAY"), headingStartsWith(folded, "WOD"), headingStartsWith(folded, "METCON"), headingStartsWith(folded, "CONDITIONING"):
		return domain.WorkoutSectionWOD, true
	case headingStartsWith(folded, "ACCESSORY"), headingStartsWith(folded, "ACESSORIO"):
		return domain.WorkoutSectionAccessory, true
	case headingStartsWith(folded, "COOLDOWN"), headingStartsWith(folded, "COOL DOWN"), headingStartsWith(folded, "VOLTA A CALMA"):
		return domain.WorkoutSectionCooldown, true
	default:
		return "", false
	}
}

func headingStartsWith(value, heading string) bool {
	if value == heading {
		return true
	}
	if !strings.HasPrefix(value, heading) || len(value) == len(heading) {
		return false
	}
	next := value[len(heading)]
	return next == ' ' || next == '-' || next == ':'
}

func detectWorkoutFormats(rawText string) []string {
	folded := foldWorkoutText(rawText)
	formats := make([]string, 0, 4)
	add := func(value string) {
		for _, current := range formats {
			if current == value {
				return
			}
		}
		formats = append(formats, value)
	}
	if strings.Contains(folded, "AMRAP") {
		add("amrap")
	}
	if strings.Contains(folded, "EMOM") {
		add("emom")
	}
	if strings.Contains(folded, "FOR TIME") {
		add("for_time")
	}
	if strings.Contains(folded, "TABATA") {
		add("tabata")
	}
	if strings.Contains(folded, "A CADA ") || strings.Contains(folded, "EVERY ") {
		add("interval")
	}
	if strings.Contains(folded, "MAIOR CARGA") || strings.Contains(folded, "MAX LOAD") || strings.Contains(folded, "1RM") {
		add("max_effort")
	}
	return formats
}

func detectWorkoutDuration(rawText string) int {
	match := workoutDurationPattern.FindStringSubmatch(rawText)
	if len(match) != 2 {
		return 0
	}
	minutes, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return minutes * 60
}

func detectMovementMentions(sections []domain.WorkoutSection) []string {
	movements := make([]string, 0, 12)
	seen := make(map[string]struct{})
	add := func(value string) {
		value = strings.Trim(strings.TrimSpace(value), ".,;:-")
		folded := foldWorkoutText(value)
		if value == "" || strings.HasPrefix(folded, "ROUNDS") || strings.HasPrefix(folded, "MINUTO") || strings.HasPrefix(folded, "ROUND") {
			return
		}
		if _, exists := seen[folded]; exists {
			return
		}
		seen[folded] = struct{}{}
		movements = append(movements, value)
	}

	for _, section := range sections {
		if section.Type == domain.WorkoutSectionSkill {
			parts := strings.FieldsFunc(section.Title, func(r rune) bool { return r == '-' || r == ':' })
			if len(parts) > 1 {
				add(parts[len(parts)-1])
			}
		}
		for _, line := range strings.Split(section.Content, "\n") {
			if match := leadingMovementPattern.FindStringSubmatch(line); len(match) == 2 {
				add(match[1])
				continue
			}
			if match := fazerMovementPattern.FindStringSubmatch(line); len(match) == 2 {
				add(match[1])
			}
		}
	}
	return movements
}

func suggestedWorkoutTitle(classification domain.WorkoutClassification) string {
	parts := make([]string, 0, 2)
	for _, section := range classification.Sections {
		if section.Type != domain.WorkoutSectionSkill && section.Type != domain.WorkoutSectionStrength {
			continue
		}
		titleParts := strings.FieldsFunc(section.Title, func(r rune) bool { return r == '-' || r == ':' })
		if len(titleParts) > 1 && strings.TrimSpace(titleParts[len(titleParts)-1]) != "" {
			parts = append(parts, strings.TrimSpace(titleParts[len(titleParts)-1]))
			break
		}
	}
	if len(classification.Formats) > 0 {
		format := strings.ToUpper(strings.ReplaceAll(classification.Formats[0], "_", " "))
		if classification.DurationSeconds > 0 {
			format += " " + strconv.Itoa(classification.DurationSeconds/60) + "'"
		}
		parts = append(parts, format)
	}
	if len(parts) == 0 {
		return "Treino do dia"
	}
	return strings.Join(parts, " + ")
}

func foldWorkoutText(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		switch r {
		case 'Á', 'À', 'Ã', 'Â', 'Ä':
			return 'A'
		case 'É', 'È', 'Ê', 'Ë':
			return 'E'
		case 'Í', 'Ì', 'Î', 'Ï':
			return 'I'
		case 'Ó', 'Ò', 'Õ', 'Ô', 'Ö':
			return 'O'
		case 'Ú', 'Ù', 'Û', 'Ü':
			return 'U'
		case 'Ç':
			return 'C'
		default:
			if unicode.IsSpace(r) {
				return ' '
			}
			return r
		}
	}, value)
	return strings.Join(strings.Fields(value), " ")
}
