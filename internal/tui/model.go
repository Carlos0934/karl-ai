package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/carlos0934/karl-ai/internal/domain"
)

// SavedModelSource provides the current model override for an agent.
type SavedModelSource interface {
	ModelForAgent(ctx context.Context, agent string) (domain.AgentConfig, error)
}

// Selection captures a developer's choice of model for a specific agent.
type Selection struct {
	Client   string `json:"client"`
	Agent    string `json:"agent"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Variant  string `json:"variant,omitempty"`
}

type interactiveScreen uint8

const (
	screenLoadingClients interactiveScreen = iota
	screenClient
	screenLoadingCatalog
	screenAgent
	screenLoadingSaved
	screenProvider
	screenModel
	screenVariant
	screenManualModel
	screenManualVariant
	screenReview
	screenDiscard
)

type configurationSession struct {
	step               configurationStep
	clients            []domain.ClientInfo
	client             string
	agent              string
	provider           string
	providerDraft      string
	model              string
	modelDraft         string
	variant            string
	variantDraft       string
	variantModel       string
	manualModel        string
	manualVariant      string
	manualVariantModel string
	catalog            domain.ModelCatalog
	catalogErr         error
	models             []domain.Model
	variants           []string
	catalogs           map[string]domain.ModelCatalog
	catalogErrors      map[string]error
	catalogLoaded      map[string]bool
	originals          map[string]domain.AgentConfig
	pending            map[string]Selection
	effective          domain.AgentConfig
}

func newConfigurationSession() configurationSession {
	return configurationSession{
		catalogs:      make(map[string]domain.ModelCatalog),
		catalogErrors: make(map[string]error),
		catalogLoaded: make(map[string]bool),
		originals:     make(map[string]domain.AgentConfig),
		pending:       make(map[string]Selection),
	}
}

func (s *configurationSession) resetDraftsForAgent(agent string) {
	s.providerDraft = ""
	s.modelDraft = ""
	s.variantDraft = ""
	s.variantModel = ""
	s.manualModel = ""
	s.manualVariant = ""
	s.manualVariantModel = ""
}

type interactiveConfigureModel struct {
	ctx              context.Context
	cancelContext    context.CancelFunc
	root             string
	discovery        domain.Discovery
	savedSource      SavedModelSource
	session          configurationSession
	screen           interactiveScreen
	form             *huh.Form
	field            huh.Field
	backEnabled      bool
	value            string
	confirmed        bool
	manualErr        error
	manualBack       interactiveScreen
	variants         map[string][]string
	result           []Selection
	err              error
	window           tea.WindowSizeMsg
	hasWindow        bool
	formWindowScreen interactiveScreen
	requestSeq       uint64
	activeRequest    uint64
	requestCancel    context.CancelFunc
	formGeneration   uint64
	completedView    string
}

func newInteractiveConfigureModel(ctx context.Context, root string, discovery domain.Discovery, saved SavedModelSource) *interactiveConfigureModel {
	sessionContext, cancel := context.WithCancel(ctx)
	return &interactiveConfigureModel{
		ctx:           sessionContext,
		cancelContext: cancel,
		root:          root,
		discovery:     discovery,
		savedSource:   saved,
		session:       newConfigurationSession(),
		screen:        screenLoadingClients,
	}
}
