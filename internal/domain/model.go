package domain

// Provider identifies a provider returned by the selected client.
type Provider struct {
	ID string `json:"id"`
}

// Model describes a model returned by the selected provider.
type Model struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ReleaseDate string   `json:"release_date,omitempty"`
	Variants    []string `json:"variants,omitempty"`
}

// ModelCatalog contains every provider and model discovered for one client.
type ModelCatalog struct {
	Providers        []Provider         `json:"providers"`
	ModelsByProvider map[string][]Model `json:"modelsByProvider"`
}

// ModelSelection holds the model override selected for one Karl agent.
type ModelSelection struct {
	Client  string `json:"client,omitempty"`
	Agent   string `json:"agent"`
	Model   string `json:"model"`
	Variant string `json:"variant,omitempty"`
}

// ModelConfigurationResult describes saved model overrides and their projection sync.
type ModelConfigurationResult struct {
	Changes []ModelSelection `json:"changes"`
	Sync    Result           `json:"sync"`
}
