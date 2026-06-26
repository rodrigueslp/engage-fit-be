package parsers

import (
	"context"
	"strings"
	"testing"

	"boxengage/backend/internal/domain"
)

func TestCheckinParserParsesCSVWithSemicolon(t *testing.T) {
	input := strings.NewReader("nome;email;telefone;data;hora\nAna Silva;ana@example.com;11999999999;22/06/2026;07:30\n")

	result, err := NewCheckinParser().Parse(context.Background(), input, domain.SourceWellhub, "wellhub.csv")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].StudentName != "Ana Silva" {
		t.Fatalf("expected student name Ana Silva, got %q", result[0].StudentName)
	}
	if result[0].StudentEmail != "ana@example.com" {
		t.Fatalf("expected email ana@example.com, got %q", result[0].StudentEmail)
	}
	if result[0].CheckinDate.Format("2006-01-02") != "2026-06-22" {
		t.Fatalf("expected date 2026-06-22, got %s", result[0].CheckinDate.Format("2006-01-02"))
	}
	if result[0].CheckinTime == nil || result[0].CheckinTime.Format("15:04") != "07:30" {
		t.Fatal("expected checkin time 07:30")
	}
}

func TestCheckinParserSkipsRowsWithoutNameOrDate(t *testing.T) {
	input := strings.NewReader("name,email,date\n,missing@example.com,2026-06-22\nJohn,john@example.com,\n")

	result, err := NewCheckinParser().Parse(context.Background(), input, domain.SourceTotalPass, "totalpass.csv")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(result))
	}
}

func TestCheckinParserParsesTotalPassTokenExport(t *testing.T) {
	input := strings.NewReader("Criado em,22 de Junho de 2026 15:42\nDescrição,Planilha listando todos os tokens\nID,Código,Nome,Plano da academia,Colaborador,Repasse,Validado em\n535602390,T00MYU,crossfit alados,Crossfit,João Paulo da Rocha,\" 14,01\",22/06/2026 12:23:45\n")

	result, err := NewCheckinParser().Parse(context.Background(), input, domain.SourceTotalPass, "totalpass.csv")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].StudentName != "João Paulo da Rocha" {
		t.Fatalf("expected TotalPass collaborator name, got %q", result[0].StudentName)
	}
	if result[0].StudentExternalID != "535602390" {
		t.Fatalf("expected external id from ID column, got %q", result[0].StudentExternalID)
	}
	if result[0].CheckinDate.Format("2006-01-02") != "2026-06-22" {
		t.Fatalf("expected date 2026-06-22, got %s", result[0].CheckinDate.Format("2006-01-02"))
	}
	if result[0].CheckinTime == nil || result[0].CheckinTime.Format("15:04:05") != "12:23:45" {
		t.Fatal("expected TotalPass validation time 12:23:45")
	}
}

func TestCheckinParserParsesWellhubExport(t *testing.T) {
	input := strings.NewReader("Veja todos os check-ins que você teve de 2026-06-21 a 2026-06-21.\n \n{Data} Data do check-in\n{Hora}Horário do check-in\nData,Hora,ID da unidade,Unidade,Visitante,ID do Wellhub,Produto,Período de check-in,Tipo de check-in,Pagamento\n2026-06-21,16:54,84de7bab,CrossFit Alados,Marília Damacena da Silva,3403206168288,Esportes,Horários padrão,Visita,17.25\n")

	result, err := NewCheckinParser().Parse(context.Background(), input, domain.SourceWellhub, "wellhub.csv")
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0].StudentName != "Marília Damacena da Silva" {
		t.Fatalf("expected Wellhub visitor name, got %q", result[0].StudentName)
	}
	if result[0].StudentExternalID != "3403206168288" {
		t.Fatalf("expected external id from ID do Wellhub, got %q", result[0].StudentExternalID)
	}
	if result[0].CheckinDate.Format("2006-01-02") != "2026-06-21" {
		t.Fatalf("expected date 2026-06-21, got %s", result[0].CheckinDate.Format("2006-01-02"))
	}
	if result[0].CheckinTime == nil || result[0].CheckinTime.Format("15:04") != "16:54" {
		t.Fatal("expected Wellhub time 16:54")
	}
}
