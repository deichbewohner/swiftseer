package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func (c *Client) GetResults(
	ctx context.Context,
	reportID int,
	opts *models.ResultsOptions,
) ([]byte, error) {
	url := c.buildURL(fmt.Sprintf("/reports/%d/results/fc", reportID))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	if opts != nil {
		if opts.IncludeKBestModels > 0 {
			q.Add("include_k_best_models", fmt.Sprintf("%d", opts.IncludeKBestModels))
		}
		if opts.IncludeBacktesting {
			q.Add("include_backtesting", "true")
		} else {
			q.Add("include_backtesting", "false")
		}
		if opts.IncludeDiscardedModels {
			q.Add("include_discarded_models", "true")
		} else {
			q.Add("include_discarded_models", "false")
		}
	} else {
		q.Add("include_k_best_models", "1")
		q.Add("include_backtesting", "false")
		q.Add("include_discarded_models", "false")
	}
	req.URL.RawQuery = q.Encode()

	resultsCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	body, err := c.doRequestRaw(resultsCtx, req)
	if err != nil {
		return nil, fmt.Errorf("results request failed: %w", err)
	}

	return body, nil
}
