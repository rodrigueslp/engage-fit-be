package featureflags

import (
	"context"
	"errors"
	"strings"
)

type Definition struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Group       string   `json:"group"`
	Description string   `json:"description"`
	Environment string   `json:"environment"`
	Editable    bool     `json:"editable"`
	Requires    []string `json:"requires"`
}

type Item struct {
	Definition
	Configured bool  `json:"configured"`
	Enabled    bool  `json:"enabled"`
	Override   *bool `json:"override"`
}

// Editable flags are request-gated and have no startup-only service or worker.
var definitions = []Definition{
	{"whatsapp", "WhatsApp", "Comunicação", "Acesso a mensagens e configuração do canal. O envio real tem controle separado.", "FEATURE_WHATSAPP_ENABLED", true, nil},
	{"email", "E-mail", "Comunicação", "Acesso a campanhas de e-mail. O envio real tem controle separado.", "FEATURE_EMAIL_ENABLED", true, nil},
	{"automation", "Automação", "Comunicação", "Acesso a automações. A execução periódica depende do worker configurado no servidor.", "FEATURE_AUTOMATION_ENABLED", true, nil},
	{"workouts", "Treino do dia", "Comunicação", "Criação e envio de treinos, inclusive no aplicativo do aluno.", "FEATURE_WORKOUTS_ENABLED", true, nil},
	{"llm", "IA para treinos", "Comunicação", "Geração por IA; requer chave do provedor configurada no servidor.", "FEATURE_LLM_ENABLED", true, []string{"workouts"}},
	{"billing", "Cobrança da plataforma", "Financeiro", "Gateway e webhook configurados na inicialização.", "FEATURE_BILLING_ENABLED", false, nil},
	{"member_billing", "Cobrança de alunos", "Financeiro", "Cobranças vinculadas a matrículas.", "FEATURE_MEMBER_BILLING_ENABLED", false, []string{"enrollments"}},
	{"catalog", "Catálogo", "Gestão", "Modalidades, espaços e estrutura.", "FEATURE_CATALOG_ENABLED", true, nil},
	{"service_catalog", "Serviços", "Gestão", "Serviços ofertados no catálogo.", "FEATURE_SERVICE_CATALOG_ENABLED", true, []string{"catalog"}},
	{"people", "Pessoas", "Gestão", "Identidade e dados protegidos; depende de chaves no servidor.", "FEATURE_PEOPLE_FOUNDATION_ENABLED", true, nil},
	{"memberships", "Planos", "Gestão", "Catálogo de planos.", "FEATURE_MEMBERSHIPS_ENABLED", true, []string{"catalog"}},
	{"enrollments", "Matrículas", "Gestão", "Vínculo da pessoa com o plano.", "FEATURE_ENROLLMENTS_ENABLED", true, []string{"memberships", "people"}},
	{"contracts", "Contratos", "Gestão", "Documentos e aceite; inicialização de serviços necessária.", "FEATURE_CONTRACTS_ENABLED", false, []string{"enrollments"}},
	{"contract_acceptance_simulator", "Simulador de aceite", "Gestão", "Somente ambientes de teste.", "CONTRACT_ACCEPTANCE_SIMULATOR_ENABLED", false, []string{"contracts"}},
	{"contract_lifecycle", "Ciclo contratual", "Gestão", "Assinatura e ações legais.", "FEATURE_CONTRACT_LIFECYCLE_ENABLED", false, []string{"contracts"}},
	{"health_forms", "Formulários de saúde", "Gestão", "Dados sensíveis e formulários.", "FEATURE_HEALTH_FORMS_ENABLED", false, []string{"enrollments", "people"}},
	{"scheduling", "Profissionais", "Agenda", "Cadastro de profissionais da agenda.", "FEATURE_SCHEDULING_ENABLED", false, nil},
	{"scheduling_grid", "Grade de aulas", "Agenda", "Grade recorrente e materializador inicializado no servidor.", "FEATURE_SCHEDULING_GRID_ENABLED", false, []string{"scheduling", "catalog"}},
	{"reservations", "Reservas", "Agenda", "Reservas e lista de espera.", "FEATURE_RESERVATIONS_ENABLED", true, []string{"scheduling_grid", "enrollments", "people"}},
	{"attendance", "Presença", "Agenda", "Admissão e presença nas aulas.", "FEATURE_ATTENDANCE_ENABLED", true, []string{"reservations"}},
	{"announcements", "Comunicados", "Engajamento", "Comunicados no aplicativo.", "FEATURE_ANNOUNCEMENTS_ENABLED", true, nil},
	{"commerce", "Quiosque", "Financeiro", "Venda avulsa na unidade.", "FEATURE_COMMERCE_ENABLED", true, []string{"people"}},
	{"whatsapp_real_send", "Envio real por WhatsApp", "Operação", "Permissão para chamar o provedor real de WhatsApp.", "WHATSAPP_ALLOW_REAL_SEND", false, []string{"whatsapp"}},
	{"email_real_send", "Envio real por e-mail", "Operação", "Permissão para enviar e-mails reais.", "EMAIL_ALLOW_REAL_SEND", false, []string{"email"}},
	{"automation_worker", "Worker de automação", "Operação", "Execução periódica das automações.", "AUTOMATION_WORKER_ENABLED", false, []string{"automation"}},
	{"scheduling_materializer", "Materializador da agenda", "Operação", "Manutenção automática das sessões futuras.", "SCHEDULING_MATERIALIZER_ENABLED", false, []string{"scheduling_grid"}},
}

type Store interface {
	List(context.Context) (map[string]bool, error)
	Set(context.Context, string, bool, string, string, string) error
}

type Service struct {
	base         map[string]bool
	store        Store
	availability map[string]bool
}

func NewService(base map[string]bool, store Store) *Service {
	return &Service{base: base, store: store}
}

func (s *Service) WithAvailability(availability map[string]bool) *Service {
	s.availability = availability
	return s
}

func (s *Service) editable(d Definition) bool {
	if !d.Editable {
		return false
	}
	available, known := s.availability[d.Key]
	return !known || available
}

func (s *Service) Effective(ctx context.Context) (map[string]bool, error) {
	values := make(map[string]bool, len(s.base))
	for key, enabled := range s.base {
		values[key] = enabled
	}
	if s.store != nil {
		overrides, err := s.store.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, d := range definitions {
			if d.Editable {
				if enabled, ok := overrides[d.Key]; ok && (!enabled || s.editable(d)) {
					values[d.Key] = enabled
				}
			}
		}
	}
	for range definitions {
		for _, d := range definitions {
			for _, dependency := range d.Requires {
				if !values[dependency] {
					values[d.Key] = false
				}
			}
		}
	}
	return values, nil
}

func (s *Service) List(ctx context.Context) ([]Item, error) {
	current, err := s.Effective(ctx)
	if err != nil {
		return nil, err
	}
	overrides := map[string]bool{}
	if s.store != nil {
		overrides, err = s.store.List(ctx)
		if err != nil {
			return nil, err
		}
	}
	items := make([]Item, 0, len(definitions))
	for _, d := range definitions {
		item := Item{Definition: d, Configured: s.base[d.Key], Enabled: current[d.Key]}
		item.Editable = s.editable(d)
		if value, ok := overrides[d.Key]; ok && d.Editable {
			item.Override = &value
		}
		items = append(items, item)
	}
	return items, nil
}

var ErrNotEditable = errors.New("esta funcionalidade exige alteração da configuração do servidor")
var ErrReasonRequired = errors.New("informe o motivo da alteração")
var ErrDependencyDisabled = errors.New("ative primeiro as funcionalidades necessárias")

func (s *Service) Set(ctx context.Context, key string, enabled bool, reason, adminID, ip string) error {
	if s.store == nil {
		return ErrNotEditable
	}
	editable := false
	for _, d := range definitions {
		if d.Key == key {
			editable = s.editable(d)
			break
		}
	}
	if !editable {
		return ErrNotEditable
	}
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	if enabled {
		current, err := s.Effective(ctx)
		if err != nil {
			return err
		}
		for _, d := range definitions {
			if d.Key == key {
				for _, dependency := range d.Requires {
					if !current[dependency] {
						return ErrDependencyDisabled
					}
				}
				break
			}
		}
	}
	return s.store.Set(ctx, key, enabled, strings.TrimSpace(reason), adminID, ip)
}
