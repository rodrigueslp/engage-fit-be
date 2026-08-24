package domain

import (
	"strings"
	"testing"
)

func TestValidateTemplateTextLimits(t *testing.T) {
	tests := []struct {
		name      string
		tmplName  string
		title     string
		body      string
		statement string
		wantErr   bool
	}{
		{
			name:      "valid template",
			tmplName:  "Termo Padrão",
			title:     "Contrato de Prestação de Serviços",
			body:      "# Contrato\nEste é o corpo do contrato para {{member_name}}.",
			statement: "Li e concordo com todos os termos deste contrato.",
			wantErr:   false,
		},
		{
			name:      "empty name",
			tmplName:  "",
			title:     "Título",
			body:      "Corpo",
			statement: "Declaração com mais de dez chars",
			wantErr:   true,
		},
		{
			name:      "statement too short",
			tmplName:  "Nome",
			title:     "Título",
			body:      "Corpo",
			statement: "Curto",
			wantErr:   true,
		},
		{
			name:      "title too long",
			tmplName:  "Nome",
			title:     strings.Repeat("a", 161),
			body:      "Corpo",
			statement: "Declaração com mais de dez chars",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateTextLimits(tt.tmplName, tt.title, tt.body, tt.statement)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplateTextLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMarkdownContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "safe markdown",
			content: "# Título\n\nTexto com **negrito** e _itálico_.\n\n- Item 1\n- Item 2",
			wantErr: false,
		},
		{
			name:    "disallow script tag",
			content: "Texto <script>alert(1)</script>",
			wantErr: true,
		},
		{
			name:    "disallow iframe",
			content: "Texto <iframe src=\"https://evil.test\"></iframe>",
			wantErr: true,
		},
		{
			name:    "disallow generic html tag",
			content: "Texto <div class=\"hidden\">teste</div>",
			wantErr: true,
		},
		{
			name:    "disallow markdown images",
			content: "Texto com imagem ![logo](https://example.test/img.png)",
			wantErr: true,
		},
		{
			name:    "disallow javascript links",
			content: "Clique [aqui](javascript:alert(1)) para continuar",
			wantErr: true,
		},
		{
			name:    "disallow data links",
			content: "Clique [aqui](data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMarkdownContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMarkdownContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		texts   []string
		wantErr bool
	}{
		{
			name:    "allowed placeholders",
			texts:   []string{"Olá {{member_name}}, seu plano é {{plan_name}} no valor de {{total_price}}."},
			wantErr: false,
		},
		{
			name:    "unknown placeholder",
			texts:   []string{"Olá {{member_cpf}}, seu plano é {{plan_name}}."},
			wantErr: true,
		},
		{
			name:    "malformed placeholder unclosed",
			texts:   []string{"Olá {{member_name, seu plano é {{plan_name}}."},
			wantErr: true,
		},
		{
			name:    "nested braces",
			texts:   []string{"Olá {{{member_name}}}, seu plano é {{plan_name}}."},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlaceholders(tt.texts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePlaceholders() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderTemplateText(t *testing.T) {
	tmpl := "Olá {{member_name}}, sua unidade é {{unit_name}} na organização {{organization_name}}. Plano {{plan_name}} ({{plan_version}}), ciclo {{billing_cycle}}. Base {{base_price}}, Adicionais {{addons_price}} ({{addons_summary}}), Total {{total_price}}. Vigência: {{starts_on}} até {{valid_until}}. Acesso: {{access_summary}}."

	ctx := ContractRenderContext{
		OrganizationName: "CrossFit Alados Org",
		UnitName:         "Alados Box 1",
		MemberName:       "Carlos Silva",
		PlanName:         "CrossFit Pro",
		PlanVersion:      "v1",
		BillingCycle:     "Mensal",
		BasePrice:        "R$ 250,00",
		AddonsSummary:    "Recovery 2x (R$ 50,00)",
		AddonsPrice:      "R$ 50,00",
		TotalPrice:       "R$ 300,00",
		StartsOn:         "24/08/2026",
		ValidUntil:       "Sem término definido",
		AccessSummary:    "CrossFit · acesso livre",
	}

	rendered := RenderTemplateText(tmpl, ctx)
	expected := "Olá Carlos Silva, sua unidade é Alados Box 1 na organização CrossFit Alados Org. Plano CrossFit Pro (v1), ciclo Mensal. Base R$ 250,00, Adicionais R$ 50,00 (Recovery 2x (R$ 50,00)), Total R$ 300,00. Vigência: 24/08/2026 até Sem término definido. Acesso: CrossFit · acesso livre."

	if rendered != expected {
		t.Errorf("RenderTemplateText() got %q, want %q", rendered, expected)
	}
}

func TestFormatBRL(t *testing.T) {
	cases := []struct {
		cents int
		want  string
	}{
		{0, "R$ 0,00"},
		{50, "R$ 0,50"},
		{100, "R$ 1,00"},
		{18900, "R$ 189,00"},
		{1250000, "R$ 12.500,00"},
		{-5000, "-R$ 50,00"},
	}

	for _, tc := range cases {
		got := FormatBRL(tc.cents)
		if got != tc.want {
			t.Errorf("FormatBRL(%d) = %q, want %q", tc.cents, got, tc.want)
		}
	}
}

func TestCanonicalHashingDeterminism(t *testing.T) {
	hash1, err := CalculateTemplateVersionContentSHA256("Título", "Corpo", "Declaração de aceite válida")
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := CalculateTemplateVersionContentSHA256("Título", "Corpo", "Declaração de aceite válida")
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 || len(hash1) != 64 {
		t.Errorf("hashes should be deterministic and 64 chars: %s vs %s", hash1, hash2)
	}

	docHash1, err := CalculateDocumentSHA256("Título", "Corpo", "Declaração de aceite válida", hash1)
	if err != nil {
		t.Fatal(err)
	}
	docHash2, err := CalculateDocumentSHA256("Título", "Corpo", "Declaração de aceite válida", hash1)
	if err != nil {
		t.Fatal(err)
	}
	if docHash1 != docHash2 || len(docHash1) != 64 {
		t.Errorf("doc hashes should be deterministic: %s vs %s", docHash1, docHash2)
	}
}
