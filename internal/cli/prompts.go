package cli

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

// PromptForTicker prompts the user to enter a stock ticker symbol
func PromptForTicker() (string, error) {
	var ticker string
	prompt := &survey.Input{
		Message: "Enter the stock ticker symbol (e.g., AAPL, MSFT, GOOGL):",
		Help:    "Please enter a valid stock ticker symbol for analysis",
	}
	
	err := survey.AskOne(prompt, &ticker, survey.WithValidator(func(val interface{}) error {
		str := val.(string)
		str = strings.TrimSpace(strings.ToUpper(str))
		if len(str) == 0 {
			return fmt.Errorf("ticker symbol cannot be empty")
		}
		if len(str) > 10 {
			return fmt.Errorf("ticker symbol too long (max 10 characters)")
		}
		// Basic format validation
		matched, _ := regexp.MatchString(`^[A-Z0-9.-]+$`, str)
		if !matched {
			return fmt.Errorf("invalid ticker format (use letters, numbers, dots, and hyphens only)")
		}
		return nil
	}))
	
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(strings.ToUpper(ticker)), nil
}

// PromptForAnalysisDate prompts the user to enter an analysis date
func PromptForAnalysisDate() (time.Time, error) {
	var dateStr string
	prompt := &survey.Input{
		Message: "Enter the analysis date (YYYY-MM-DD) or press Enter for today:",
		Help:    "Format: YYYY-MM-DD (e.g., 2024-01-15). Leave empty for today's date.",
		Default: time.Now().Format("2006-01-02"),
	}
	
	err := survey.AskOne(prompt, &dateStr, survey.WithValidator(func(val interface{}) error {
		str := strings.TrimSpace(val.(string))
		if str == "" {
			return nil // Allow empty for today's date
		}
		
		// Parse the date
		parsedDate, err := time.Parse("2006-01-02", str)
		if err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}
		
		// Don't allow future dates beyond tomorrow
		tomorrow := time.Now().AddDate(0, 0, 1)
		if parsedDate.After(tomorrow) {
			return fmt.Errorf("analysis date cannot be more than 1 day in the future")
		}
		
		// Don't allow dates too far in the past (e.g., more than 5 years)
		fiveYearsAgo := time.Now().AddDate(-5, 0, 0)
		if parsedDate.Before(fiveYearsAgo) {
			return fmt.Errorf("analysis date cannot be more than 5 years in the past")
		}
		
		return nil
	}))
	
	if err != nil {
		return time.Time{}, err
	}
	
	if strings.TrimSpace(dateStr) == "" {
		return time.Now(), nil
	}
	
	return time.Parse("2006-01-02", strings.TrimSpace(dateStr))
}

// PromptForAnalysts prompts the user to select analyst team members
func PromptForAnalysts() ([]AnalystType, error) {
	var selectedAnalysts []string
	
	options := []string{
		MarketAnalyst.GetDisplayName(),
		SocialAnalyst.GetDisplayName(),
		NewsAnalyst.GetDisplayName(),
		FundamentalsAnalyst.GetDisplayName(),
	}
	
	prompt := &survey.MultiSelect{
		Message: "Select analyst team members:",
		Options: options,
		Help:    "Choose one or more analysts to include in your analysis team. Use space to select, enter to confirm.",
		Default: options, // Default to all analysts
	}
	
	err := survey.AskOne(prompt, &selectedAnalysts, survey.WithValidator(func(val interface{}) error {
		// For MultiSelect, we need to check the length of the selection
		selected, ok := val.([]survey.OptionAnswer)
		if !ok {
			return fmt.Errorf("invalid selection type")
		}
		if len(selected) == 0 {
			return fmt.Errorf("you must select at least one analyst")
		}
		return nil
	}))
	
	if err != nil {
		return nil, err
	}
	
	// Convert display names back to AnalystType
	var result []AnalystType
	for _, selected := range selectedAnalysts {
		switch selected {
		case MarketAnalyst.GetDisplayName():
			result = append(result, MarketAnalyst)
		case SocialAnalyst.GetDisplayName():
			result = append(result, SocialAnalyst)
		case NewsAnalyst.GetDisplayName():
			result = append(result, NewsAnalyst)
		case FundamentalsAnalyst.GetDisplayName():
			result = append(result, FundamentalsAnalyst)
		}
	}
	
	return result, nil
}

// PromptForResearchDepth prompts the user to select research depth
func PromptForResearchDepth() (ResearchDepth, error) {
	var selected string
	
	options := []string{
		fmt.Sprintf("Shallow (%d round) - Quick analysis", ShallowResearch.GetResearchRounds()),
		fmt.Sprintf("Medium (%d rounds) - Balanced analysis", MediumResearch.GetResearchRounds()),
		fmt.Sprintf("Deep (%d rounds) - Comprehensive analysis", DeepResearch.GetResearchRounds()),
	}
	
	prompt := &survey.Select{
		Message: "Select research depth:",
		Options: options,
		Help:    "Choose the depth of analysis. More rounds provide more comprehensive results but take longer.",
		Default: options[1], // Default to medium
	}
	
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return "", err
	}
	
	// Map selection back to ResearchDepth
	switch {
	case strings.HasPrefix(selected, "Shallow"):
		return ShallowResearch, nil
	case strings.HasPrefix(selected, "Medium"):
		return MediumResearch, nil
	case strings.HasPrefix(selected, "Deep"):
		return DeepResearch, nil
	default:
		return MediumResearch, nil
	}
}

// PromptForLLMProvider prompts the user to select an LLM provider
func PromptForLLMProvider() (LLMProvider, error) {
	var selected string
	
	options := []string{
		string(OpenAIProvider) + " - OpenAI GPT models",
		string(AnthropicProvider) + " - Anthropic Claude models",
		string(GoogleProvider) + " - Google Gemini models",
		string(OpenRouterProvider) + " - OpenRouter (multiple providers)",
		string(OllamaProvider) + " - Ollama (local models)",
	}
	
	prompt := &survey.Select{
		Message: "Select LLM provider:",
		Options: options,
		Help:    "Choose your preferred language model provider. Make sure you have appropriate API keys configured.",
		Default: options[0], // Default to OpenAI
	}
	
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return "", err
	}
	
	// Extract provider from selection
	provider := strings.Split(selected, " -")[0]
	return LLMProvider(provider), nil
}

// PromptForModels prompts the user to select quick and deep thinking models
func PromptForModels(provider LLMProvider) (string, string, error) {
	quickModels, deepModels := provider.GetProviderModels()
	
	if len(quickModels) == 0 || len(deepModels) == 0 {
		return "", "", fmt.Errorf("no models available for provider %s", provider)
	}
	
	// Prompt for quick thinking model
	var quickModel string
	quickPrompt := &survey.Select{
		Message: "Select quick-thinking model (for rapid analysis):",
		Options: quickModels,
		Help:    "This model is used for quick analysis and tool calls.",
		Default: quickModels[0],
	}
	
	err := survey.AskOne(quickPrompt, &quickModel)
	if err != nil {
		return "", "", err
	}
	
	// Prompt for deep thinking model
	var deepModel string
	deepPrompt := &survey.Select{
		Message: "Select deep-thinking model (for complex reasoning):",
		Options: deepModels,
		Help:    "This model is used for complex reasoning and final decisions.",
		Default: deepModels[0],
	}
	
	err = survey.AskOne(deepPrompt, &deepModel)
	if err != nil {
		return "", "", err
	}
	
	return quickModel, deepModel, nil
}

// PromptForConfirmation prompts the user to confirm their selections
func PromptForConfirmation(selections UserSelections) (bool, error) {
	// Format analysts list
	analystNames := make([]string, len(selections.Analysts))
	for i, analyst := range selections.Analysts {
		analystNames[i] = analyst.GetDisplayName()
	}
	
	summary := fmt.Sprintf(`
Analysis Configuration Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 Ticker Symbol:     %s
📅 Analysis Date:     %s
👥 Analyst Team:      %s
🔍 Research Depth:    %s (%d rounds)
🤖 LLM Provider:      %s
⚡ Quick Model:       %s
🧠 Deep Model:        %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
		selections.Ticker,
		selections.AnalysisDate.Format("2006-01-02"),
		strings.Join(analystNames, ", "),
		selections.ResearchDepth,
		selections.ResearchDepth.GetResearchRounds(),
		selections.LLMProvider,
		selections.QuickModel,
		selections.DeepModel,
	)
	
	fmt.Println(summary)
	
	var confirmed bool
	prompt := &survey.Confirm{
		Message: "Proceed with this analysis configuration?",
		Default: true,
	}
	
	err := survey.AskOne(prompt, &confirmed)
	return confirmed, err
}

// PromptForRestartOrExit prompts user when analysis completes
func PromptForRestartOrExit() (bool, error) {
	var choice string
	prompt := &survey.Select{
		Message: "Analysis completed! What would you like to do next?",
		Options: []string{
			"Start a new analysis",
			"Exit CortexGo",
		},
		Default: "Exit CortexGo",
	}
	
	err := survey.AskOne(prompt, &choice)
	if err != nil {
		return false, err
	}
	
	return choice == "Start a new analysis", nil
}