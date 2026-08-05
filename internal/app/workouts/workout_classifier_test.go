package workouts

import (
	"reflect"
	"testing"

	"boxengage/backend/internal/domain"
)

func TestClassifyWorkoutTextFromCoachMessage(t *testing.T) {
	raw := `WARM UP

Passagem Técnica com PVC para Snatch

3 Rounds
3 Snatch Grip Deadlift
3 Power Snatch
3 Snatch Balance
3 Overhead Squat

SKILL - Snatch

Achar a maior carga do Dia para 1 Snatch

WORKOUT OF THE DAY

AMRAP 18'

90 Double Unders
30 Dumbbell Snatches
6 Wall Walks

A cada 3 minutos, iniciando no minuto 0', fazer 15 Dumbbell Squats`

	got := ClassifyWorkoutText(raw)
	if got.Version != workoutClassificationVersion || got.GeneratedBy != "rules" {
		t.Fatalf("unexpected classifier metadata: %#v", got)
	}
	if got.SuggestedTitle != "Snatch + AMRAP 18'" {
		t.Fatalf("unexpected title: %q", got.SuggestedTitle)
	}
	if got.DurationSeconds != 18*60 {
		t.Fatalf("unexpected duration: %d", got.DurationSeconds)
	}
	if !reflect.DeepEqual(got.Formats, []string{"amrap", "interval", "max_effort"}) {
		t.Fatalf("unexpected formats: %#v", got.Formats)
	}
	wantTypes := []domain.WorkoutSectionType{domain.WorkoutSectionWarmup, domain.WorkoutSectionSkill, domain.WorkoutSectionWOD}
	if len(got.Sections) != len(wantTypes) {
		t.Fatalf("unexpected sections: %#v", got.Sections)
	}
	for index, want := range wantTypes {
		if got.Sections[index].Type != want {
			t.Fatalf("section %d: got %q, want %q", index, got.Sections[index].Type, want)
		}
	}
	for _, movement := range []string{"Snatch", "Power Snatch", "Double Unders", "Dumbbell Snatches", "Wall Walks", "Dumbbell Squats"} {
		if !containsString(got.MovementMentions, movement) {
			t.Errorf("movement %q not found in %#v", movement, got.MovementMentions)
		}
	}
}

func TestClassifyWorkoutTextKeepsUnknownTextAsOther(t *testing.T) {
	got := ClassifyWorkoutText("Corrida leve\nMobilidade de quadril")
	if len(got.Sections) != 1 || got.Sections[0].Type != domain.WorkoutSectionOther {
		t.Fatalf("unexpected sections: %#v", got.Sections)
	}
	if got.Sections[0].Content != "Corrida leve\nMobilidade de quadril" {
		t.Fatalf("unexpected content: %q", got.Sections[0].Content)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
