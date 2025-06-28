package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Model pricing structure
type modelPricing struct {
	InputCostPerToken       float64 `json:"input_cost_per_token"`
	OutputCostPerToken      float64 `json:"output_cost_per_token"`
	CacheCreateCostPerToken float64 `json:"cache_creation_input_token_cost"`
	CacheReadCostPerToken   float64 `json:"cache_read_input_token_cost"`
}

// Global variable to store fetched prices
var modelPrices map[string]modelPricing

// Fetch latest pricing from LiteLLM
func fetchModelPricing() error {
	const pricingURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

	resp, err := http.Get(pricingURL)
	if err != nil {
		return fmt.Errorf("failed to fetch pricing: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch pricing: status %d", resp.StatusCode)
	}

	var allPricing map[string]modelPricing
	if err := json.NewDecoder(resp.Body).Decode(&allPricing); err != nil {
		return fmt.Errorf("failed to decode pricing data: %w", err)
	}

	// Filter and store only Claude models
	modelPrices = make(map[string]modelPricing)
	for model, pricing := range allPricing {
		if strings.Contains(model, "claude") {
			modelPrices[model] = pricing
		}
	}

	if len(modelPrices) == 0 {
		return fmt.Errorf("no Claude pricing data found")
	}

	return nil
}

// Get pricing for a model by matching model name
func getModelPricing(modelName string) modelPricing {
	if modelPrices == nil {
		return modelPricing{} // Return zero values if prices not loaded
	}

	// Try exact match first
	if price, ok := modelPrices[modelName]; ok {
		return price
	}

	// Try various matching strategies
	for key, price := range modelPrices {
		// Check if model name contains the key
		if strings.Contains(modelName, key) {
			return price
		}
		// Check if key contains model name parts
		if strings.Contains(key, "opus") && strings.Contains(modelName, "opus") {
			return price
		}
		if strings.Contains(key, "sonnet") && strings.Contains(modelName, "sonnet") {
			return price
		}
		if strings.Contains(key, "haiku") && strings.Contains(modelName, "haiku") {
			return price
		}
	}

	// Return zero values if not found
	return modelPricing{}
}

// Calculate cost based on token usage and model
func calculateCost(usage map[string]interface{}, modelName string) float64 {
	inputTokens, _ := getTokenCount(usage, "input_tokens")
	outputTokens, _ := getTokenCount(usage, "output_tokens")
	cacheCreateTokens, _ := getTokenCount(usage, "cache_creation_input_tokens")
	cacheReadTokens, _ := getTokenCount(usage, "cache_read_input_tokens")

	pricing := getModelPricing(modelName)

	inputCost := float64(inputTokens) * pricing.InputCostPerToken
	outputCost := float64(outputTokens) * pricing.OutputCostPerToken
	cacheCreateCost := float64(cacheCreateTokens) * pricing.CacheCreateCostPerToken
	cacheReadCost := float64(cacheReadTokens) * pricing.CacheReadCostPerToken

	return inputCost + outputCost + cacheCreateCost + cacheReadCost
}

// Helper function to get token count from usage data
func getTokenCount(usage map[string]interface{}, key string) (int, bool) {
	if val, ok := usage[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v), true
		case int:
			return v, true
		}
	}
	return 0, false
}
