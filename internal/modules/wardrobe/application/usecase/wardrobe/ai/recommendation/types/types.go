// Package types defines shared data structures, interfaces, and constants for the outfit recommendation workflow.
package types

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const (
	// CandidateSourceRetrieval indicates the candidate was fetched through hybrid search.
	CandidateSourceRetrieval = repositories.HybridCandidateSourceHybrid
	// CandidateSourceFallback indicates the candidate was fetched via fallback search.
	CandidateSourceFallback = repositories.HybridCandidateSourceFallback
	// CandidateSourceStrictFallback indicates strict fallback matching.
	CandidateSourceStrictFallback = "strict-fallback"
	// CandidateSourceRelaxedFallback indicates relaxed fallback matching.
	CandidateSourceRelaxedFallback = "relaxed-fallback"
	// CandidateSourceGeneralFallback indicates general fallback matching.
	CandidateSourceGeneralFallback = "general-fallback"

	// RetrievalTermSourceDictionary indicates a query term extracted from localized dictionaries.
	RetrievalTermSourceDictionary = "dictionary"
	// RetrievalTermSourceRaw indicates a query term extracted from raw text.
	RetrievalTermSourceRaw = "raw"
	// RetrievalTermSourceTaxonomy indicates a query term expanded using taxonomy.
	RetrievalTermSourceTaxonomy = "taxonomy"
)

// CandidateForPrompt represents a candidate wardrobe item prepared for the LLM prompt payload.
type CandidateForPrompt struct {
	Item *entities.WardrobeItem
	Tags []string
}

// RerankStats holds statistics about candidate scoring and diversification.
type RerankStats struct {
	MinScore         float64
	MaxScore         float64
	AvgScore         float64
	DiversifiedCount int
}

// CandidateForRanking represents a candidate wardrobe item processed in the scoring/ranking pipeline.
type CandidateForRanking struct {
	Item            *entities.WardrobeItem
	Source          string
	VectorScore     float64
	LexicalScore    float64
	RetrievalScore  float64
	RetrievalRank   int
	RetrievalSource string
}

// RankedCandidate represents a scored and ranked wardrobe item.
type RankedCandidate struct {
	Item          *entities.WardrobeItem
	Score         float64
	Tags          []string
	Source        string
	RetrievalRank int
}

// FallbackCandidateCounts counts fallback candidates processed across matching tiers.
type FallbackCandidateCounts struct {
	Strict  int
	Relaxed int
	General int
}

// Total returns the total number of fallback candidates.
func (c FallbackCandidateCounts) Total() int {
	return c.Strict + c.Relaxed + c.General
}

// RecommendationRetrievalQuery encapsulates the parameters used for candidate retrieval.
type RecommendationRetrievalQuery struct {
	SemanticQuery  string
	LexicalTerms   []RetrievalTerm
	ExcludedTerms  []RetrievalTerm
	HardFilters    repositories.RecommendationHardFilters
	RewriterSource string
}

// RetrievalTerm represents an individual term used for lexical search matching.
type RetrievalTerm struct {
	Value        string
	Source       string
	TargetFields []string
	SourceReason string
}

// KeywordMatch represents a localized dictionary match in the NLP parsing phase.
type KeywordMatch struct {
	Category string
	Value    string
	Keyword  string
	Start    int
	End      int
	Source   string
}

// RecommendationQueryRewriter defines the contract for query rewriting in hybrid RAG.
type RecommendationQueryRewriter interface {
	Rewrite(ctx context.Context, intent dto.ParsedIntent) (RecommendationRetrievalQuery, error)
}

// LlmOutfitResponse represents the structured JSON schema returned by the LLM.
type LlmOutfitResponse struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Items       []struct {
		Role           string   `json:"role"`
		PrimaryID      string   `json:"primary_id"`
		AlternativeIDs []string `json:"alternative_ids"`
	} `json:"items"`
}
