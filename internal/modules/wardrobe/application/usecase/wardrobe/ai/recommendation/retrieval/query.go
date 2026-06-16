// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
)

// LocalRecommendationQueryRewriter writes retrieval queries locally using rule-based parsing.
type LocalRecommendationQueryRewriter struct{}

// NewLocalRecommendationQueryRewriter builds a LocalRecommendationQueryRewriter.
func NewLocalRecommendationQueryRewriter() *LocalRecommendationQueryRewriter {
	return &LocalRecommendationQueryRewriter{}
}

// Rewrite constructs a baseline retrieval query locally from the parsed NLP intent.
func (LocalRecommendationQueryRewriter) Rewrite(_ context.Context, intent dto.ParsedIntent) (types.RecommendationRetrievalQuery, error) {
	terms := BuildSourceAwareRetrievalTerms(intent)
	excludedTerms := BuildSourceAwareExcludedTerms(intent)

	return types.RecommendationRetrievalQuery{
		SemanticQuery:  intent.SemanticQuery,
		LexicalTerms:   NormalizeRetrievalTerms(terms),
		ExcludedTerms:  NormalizeRetrievalTerms(excludedTerms),
		RewriterSource: "local",
		HardFilters: repositories.RecommendationHardFilters{
			Seasonality: BuildSeasonalityHardFilters(intent.PositiveConstraints),
		},
	}, nil
}

// LLMRecommendationQueryRewriter delegates query rewriting to an LLM service, falling back to local parsing.
type LLMRecommendationQueryRewriter struct {
	aiService ai.IAIService
	cfg       *config.Config
	local     LocalRecommendationQueryRewriter
}

// NewLLMRecommendationQueryRewriter builds an LLMRecommendationQueryRewriter.
func NewLLMRecommendationQueryRewriter(aiService ai.IAIService, cfg *config.Config) *LLMRecommendationQueryRewriter {
	return &LLMRecommendationQueryRewriter{
		aiService: aiService,
		cfg:       cfg,
		local:     LocalRecommendationQueryRewriter{},
	}
}

type llmRecommendationRetrievalQuery struct {
	SemanticQuery string   `json:"semantic_query"`
	LexicalTerms  []string `json:"lexical_terms"`
	ExcludedTerms []string `json:"excluded_terms"`
	HardFilters   struct {
		Seasonality   []string `json:"seasonality"`
		CategorySlugs []string `json:"category_slugs"`
	} `json:"hard_filters"`
}

// Rewrite rewrites the recommendation intent into a refined query via the AI/LLM service.
func (r LLMRecommendationQueryRewriter) Rewrite(ctx context.Context, intent dto.ParsedIntent) (types.RecommendationRetrievalQuery, error) {
	if r.aiService == nil {
		return types.RecommendationRetrievalQuery{}, errors.New("llm rewriter ai service is nil")
	}
	localQuery, err := r.local.Rewrite(ctx, intent)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}

	systemPrompt, userPrompt, err := buildLLMRecommendationRewriterPrompts(intent, localQuery)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	response, err := r.aiService.GenerateChatText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	return validateLLMRecommendationRetrievalQuery(response, r.cfg)
}

func buildLLMRecommendationRewriterPrompts(intent dto.ParsedIntent, localQuery types.RecommendationRetrievalQuery) (string, string, error) {
	systemPrompt := strings.Join([]string{
		"You rewrite outfit recommendation intent into a compact retrieval query.",
		"Return ONLY one JSON object. Do not wrap it in markdown.",
		"Allowed keys: semantic_query, lexical_terms, excluded_terms, hard_filters.",
		"hard_filters may contain only seasonality and category_slugs arrays.",
		"Do not output SQL, PostgreSQL tsquery syntax, operators, or executable query text.",
		"Use only concise taxonomy-aligned terms likely to appear in wardrobe metadata.",
	}, "\n")

	payload := map[string]any{
		"intent": intent,
		"local_baseline": map[string]any{
			"semantic_query": localQuery.SemanticQuery,
			"lexical_terms":  ExtractTermStrings(localQuery.LexicalTerms),
			"excluded_terms": ExtractTermStrings(localQuery.ExcludedTerms),
			"hard_filters":   localQuery.HardFilters,
		},
		"allowed": map[string]any{
			"seasonality":    []string{"spring", "summer", "autumn", "winter"},
			"category_slugs": RecommendationAllowedCategorySlugs(),
		},
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, string(bytes), nil
}

func validateLLMRecommendationRetrievalQuery(raw string, cfg *config.Config) (types.RecommendationRetrievalQuery, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.RecommendationRetrievalQuery{}, errors.New("empty llm rewriter output")
	}
	var parsed llmRecommendationRetrievalQuery
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return types.RecommendationRetrievalQuery{}, fmt.Errorf("invalid llm rewriter json: %w", err)
	}
	if containsUnsafeQuerySyntax(parsed.SemanticQuery) {
		return types.RecommendationRetrievalQuery{}, errors.New("semantic query contains unsafe query syntax")
	}
	semanticQuery := strings.TrimSpace(parsed.SemanticQuery)
	maxSemanticLength := 512
	maxLexicalTerms := 24
	maxExcludedTerms := 24
	if cfg != nil {
		if cfg.RAG.RecommendationRewriterMaxSemanticLength > 0 {
			maxSemanticLength = cfg.RAG.RecommendationRewriterMaxSemanticLength
		}
		if cfg.RAG.RecommendationRewriterMaxLexicalTerms > 0 {
			maxLexicalTerms = cfg.RAG.RecommendationRewriterMaxLexicalTerms
		}
		if cfg.RAG.RecommendationRewriterMaxExcludedTerms > 0 {
			maxExcludedTerms = cfg.RAG.RecommendationRewriterMaxExcludedTerms
		}
	}

	if len(semanticQuery) > maxSemanticLength {
		return types.RecommendationRetrievalQuery{}, fmt.Errorf("semantic query exceeds max length %d", maxSemanticLength)
	}

	lexicalTerms, err := validateLLMTerms(parsed.LexicalTerms, maxLexicalTerms, "lexical_terms")
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	excludedTerms, err := validateLLMTerms(parsed.ExcludedTerms, maxExcludedTerms, "excluded_terms")
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	seasonality, err := validateLLMSeasonalityFilters(parsed.HardFilters.Seasonality)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	categorySlugs, err := validateLLMCategoryFilters(parsed.HardFilters.CategorySlugs)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}

	lexicalTermsSlice := make([]types.RetrievalTerm, len(lexicalTerms))
	for i, t := range lexicalTerms {
		lexicalTermsSlice[i] = types.RetrievalTerm{Value: t, Source: types.RetrievalTermSourceRaw}
	}
	excludedTermsSlice := make([]types.RetrievalTerm, len(excludedTerms))
	for i, t := range excludedTerms {
		excludedTermsSlice[i] = types.RetrievalTerm{Value: t, Source: types.RetrievalTermSourceRaw}
	}

	return types.RecommendationRetrievalQuery{
		SemanticQuery:  semanticQuery,
		LexicalTerms:   lexicalTermsSlice,
		ExcludedTerms:  excludedTermsSlice,
		RewriterSource: "llm",
		HardFilters: repositories.RecommendationHardFilters{
			Seasonality:   seasonality,
			CategorySlugs: categorySlugs,
		},
	}, nil
}

func validateLLMTerms(terms []string, max int, field string) ([]string, error) {
	if len(terms) > max {
		return nil, fmt.Errorf("%s exceeds max count %d", field, max)
	}
	if slices.ContainsFunc(terms, containsUnsafeQuerySyntax) {
		return nil, fmt.Errorf("%s contains unsafe query syntax", field)
	}
	return NormalizeTermSet(terms), nil
}

func validateLLMSeasonalityFilters(values []string) ([]string, error) {
	allowed := map[string]bool{"spring": true, "summer": true, "autumn": true, "winter": true}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !allowed[value] {
			return nil, fmt.Errorf("invalid seasonality hard filter %q", value)
		}
	}
	return NormalizeTermSet(values), nil
}

func validateLLMCategoryFilters(values []string) ([]string, error) {
	allowed := map[string]bool{}
	for _, slug := range RecommendationAllowedCategorySlugs() {
		allowed[slug] = true
	}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !allowed[value] {
			return nil, fmt.Errorf("invalid category hard filter %q", value)
		}
	}
	return NormalizeTermSet(values), nil
}

func containsUnsafeQuerySyntax(value string) bool {
	normalized := strings.ToLower(value)
	unsafeFragments := []string{"@@", "plainto_tsquery", "websearch_to_tsquery", "to_tsquery", "select ", " where ", " from ", "drop ", "delete ", "insert ", "update ", ";", "--"}
	for _, fragment := range unsafeFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}



// BuildRecommendationSemanticQuery format NLP intent fields into a pipe-delimited query string for embedding.
func BuildRecommendationSemanticQuery(
	intent dto.ParsedIntent,
	originalDetails string,
	hasExplicitOptions bool,
) string {
	var parts []string
	if len(intent.Occasion) > 0 {
		parts = append(parts, "occasion: "+strings.Join(intent.Occasion, ", "))
	}
	if len(intent.StyleTarget) > 0 {
		parts = append(parts, "style: "+strings.Join(intent.StyleTarget, ", "))
	}
	if len(intent.ColorTone) > 0 {
		parts = append(parts, "color tone: "+strings.Join(intent.ColorTone, ", "))
	}
	if len(intent.PositiveConstraints) > 0 {
		parts = append(parts, "constraints: "+strings.Join(intent.PositiveConstraints, ", "))
	}
	if len(intent.NegativeConstraints) > 0 {
		parts = append(parts, "avoid: "+strings.Join(intent.NegativeConstraints, ", "))
	}

	details := strings.TrimSpace(originalDetails)
	if details != "" {
		if hasExplicitOptions {
			parts = append(parts, "details context: "+details)
		} else {
			parts = append(parts, "details: "+details)
		}
	}

	return strings.Join(parts, " | ")
}

// ExpandRecommendationLexicalTerms expands a parsed intent into a list of normalized strings.
func ExpandRecommendationLexicalTerms(intent dto.ParsedIntent) []string {
	terms := ExpandRecommendationLexicalRetrievalTerms(intent)
	values := make([]string, 0, len(terms))
	for _, term := range terms {
		values = append(values, term.Value)
	}
	return NormalizeTermSet(values)
}

// ExpandRecommendationLexicalRetrievalTerms returns structured expanded RetrievalTerms for intent matching.
func ExpandRecommendationLexicalRetrievalTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupOccasion, intent.Occasion)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupStyle, intent.StyleTarget)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupWeather, intent.PositiveConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupSeason, intent.PositiveConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupColorTone, intent.ColorTone)...)
	return terms
}

// BuildSourceAwareRetrievalTerms maps explicit intent properties and raw text terms into a retrieval term slice.
func BuildSourceAwareRetrievalTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0, len(intent.LexicalTerms)+len(intent.Occasion)+len(intent.StyleTarget)+len(intent.ColorTone)+len(intent.PositiveConstraints))
	for _, value := range intent.Occasion {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.StyleTarget {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.ColorTone {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.PositiveConstraints {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	for _, value := range intent.LexicalTerms {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceRaw})
	}
	terms = append(terms, ExpandRecommendationLexicalRetrievalTerms(intent)...)
	return terms
}

// BuildSourceAwareExcludedTerms compiles terms that should be excluded from candidate results based on constraints.
func BuildSourceAwareExcludedTerms(intent dto.ParsedIntent) []types.RetrievalTerm {
	terms := make([]types.RetrievalTerm, 0, len(intent.ExcludedStyles)+len(intent.ExcludedColorTones)+len(intent.ExcludedWeather))

	// Excluded Styles
	for _, value := range intent.ExcludedStyles {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupStyle, intent.ExcludedStyles)...)

	// Excluded Color Tones
	for _, value := range intent.ExcludedColorTones {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupColorTone, intent.ExcludedColorTones)...)

	// Excluded Weather
	for _, value := range intent.ExcludedWeather {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupWeather, intent.ExcludedWeather)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupSeason, intent.ExcludedWeather)...)

	// Negative Constraints (Avoided Terms)
	for _, value := range ExtractAvoidTerms(intent.NegativeConstraints) {
		terms = append(terms, types.RetrievalTerm{Value: value, Source: types.RetrievalTermSourceDictionary})
	}

	// Expand exclusions via taxonomyGroupExcluded
	var allExclusions []string
	allExclusions = append(allExclusions, intent.ExcludedStyles...)
	allExclusions = append(allExclusions, intent.ExcludedColorTones...)
	allExclusions = append(allExclusions, intent.ExcludedWeather...)
	allExclusions = append(allExclusions, ExtractAvoidTerms(intent.NegativeConstraints)...)
	terms = append(terms, ExpandTaxonomyTerms(taxonomyGroupExcluded, allExclusions)...)

	return terms
}

// NormalizeRetrievalTerms filters out stop words, normalizes casing, and deduplicates retrieval terms by priority.
func NormalizeRetrievalTerms(terms []types.RetrievalTerm) []types.RetrievalTerm {
	byValue := map[string]types.RetrievalTerm{}
	for _, term := range terms {
		value := strings.ToLower(strings.TrimSpace(term.Value))
		if value == "" || parser.LexicalStopwords[value] {
			continue
		}
		term.Value = value
		existing, exists := byValue[value]
		if !exists || retrievalTermSourcePriority(term.Source) < retrievalTermSourcePriority(existing.Source) {
			byValue[value] = term
		}
	}

	normalized := make([]types.RetrievalTerm, 0, len(byValue))
	for _, term := range byValue {
		normalized = append(normalized, term)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Value < normalized[j].Value
	})
	return normalized
}

// ExtractTermStrings converts a slice of RetrievalTerms into a slice of raw strings.
func ExtractTermStrings(terms []types.RetrievalTerm) []string {
	strs := make([]string, len(terms))
	for i, term := range terms {
		strs[i] = term.Value
	}
	return strs
}

func retrievalTermSourcePriority(source string) int {
	switch source {
	case types.RetrievalTermSourceDictionary:
		return 0
	case types.RetrievalTermSourceRaw:
		return 1
	case types.RetrievalTermSourceTaxonomy:
		return 2
	default:
		return 3
	}
}

// ExtractAvoidTerms parses negative constraints (e.g. "avoid-style:xxx") to get the raw term.
func ExtractAvoidTerms(negativeConstraints []string) []string {
	var terms []string
	for _, constraint := range negativeConstraints {
		constraint = strings.TrimSpace(constraint)
		if after, ok := strings.CutPrefix(constraint, "avoid-term:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-style:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-color-tone:"); ok {
			terms = append(terms, after)
			continue
		}
		if after, ok := strings.CutPrefix(constraint, "avoid-weather:"); ok {
			terms = append(terms, after)
		}
	}
	return terms
}

// NormalizeTermSet deduplicates a slice of strings, sorts them, and removes stop words.
func NormalizeTermSet(terms []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || parser.LexicalStopwords[term] {
			continue
		}
		if !seen[term] {
			seen[term] = true
			normalized = append(normalized, term)
		}
	}
	sort.Strings(normalized)
	return normalized
}

// AppendWithoutDuplicate appends a string to a slice only if it is not already present (case-insensitive).
func AppendWithoutDuplicate(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	values = append(values, value)
	sort.Strings(values)
	return values
}
