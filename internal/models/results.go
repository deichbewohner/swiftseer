package models

type ResultsOptions struct {
    IncludeKBestModels     int  `json:"include_k_best_models,omitempty"`
    IncludeBacktesting     bool `json:"include_backtesting,omitempty"`
    IncludeDiscardedModels bool `json:"include_discarded_models,omitempty"`
}
