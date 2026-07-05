package orchestrator

import (
	"context"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"

	"github.com/google/uuid"
)

func mapOptions(opt aiexecutor_dto.OrchestratorOptions, requestID uuid.UUID) app_ai.TextGenerationOptions {
	return app_ai.TextGenerationOptions{
		MaxOutputTokens:  opt.MaxOutputTokens,
		Temperature:      opt.Temperature,
		ResponseMIMEType: opt.ResponseMIMEType,
		ResponseSchema:   opt.ResponseSchema,
		UserID:           opt.UserID,
		Operation:        opt.Operation,
		RequestID:        requestID,
	}
}

// ----- IAIOrchestrator -----

func (uc *OrchestratorUseCase) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	plan, err := uc.resolveExecutionPlan(ctx, uuid.Nil, aiexecutor_dto.AIOperationVision, aiexecutor_dto.AITokenEstimationMeta{})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}
	result, err := uc.aiService.AnalyzeImage(ctx, imageUrl, prompt)
	if err != nil {
		_ = uc.costUC.MarkUnknown(ctx, decision.RequestID, err.Error())
		return "", err
	}
	_ = uc.costUC.Confirm(ctx, decision.RequestID, aiexecutor_dto.AIUsage{
		Provider: plan.Provider,
		Model:    plan.Model,
	})
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	plan, err := uc.resolveExecutionPlan(ctx, uuid.Nil, aiexecutor_dto.AIOperationEmbedding, aiexecutor_dto.AITokenEstimationMeta{})
	if err != nil {
		return nil, err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return nil, err
	}
	result, err := uc.aiService.GenerateEmbeddings(ctx, chunks)
	if err != nil {
		_ = uc.costUC.MarkUnknown(ctx, decision.RequestID, err.Error())
		return nil, err
	}
	_ = uc.costUC.Confirm(ctx, decision.RequestID, aiexecutor_dto.AIUsage{
		Provider: plan.Provider,
		Model:    plan.Model,
	})
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (string, error) {
	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, options.Operation, aiexecutor_dto.AITokenEstimationMeta{})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}
	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		return "", err
	}
	result, err := uc.aiService.GenerateChatText(ctx, systemPrompt, userPrompt, mapOptions(options, plan.RequestID))
	if err != nil {
		_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
		return "", err
	}
	usage := aiexecutor_dto.AIUsage{
		Provider: plan.Provider,
		Model:    plan.Model,
	}
	if err = uc.costUC.Confirm(ctx, decision.RequestID, usage); err != nil {
		return "", err
	}
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (string, error) {
	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, options.Operation, aiexecutor_dto.AITokenEstimationMeta{})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}
	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		return "", err
	}
	result, err := uc.aiService.GenerateRecommendationText(ctx, systemPrompt, userPrompt, mapOptions(options, plan.RequestID))
	if err != nil {
		_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
		return "", err
	}
	usage := aiexecutor_dto.AIUsage{
		Provider: plan.Provider,
		Model:    plan.Model,
	}
	if err = uc.costUC.Confirm(ctx, decision.RequestID, usage); err != nil {
		return "", err
	}
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, options.Operation, aiexecutor_dto.AITokenEstimationMeta{})
	if err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}

	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}

	aiTextChan, aiErrChan := uc.aiService.GenerateChatTextStream(ctx, systemPrompt, userPrompt, mapOptions(options, plan.RequestID))

	go func() {
		defer close(textChan)
		defer close(errChan)

		for chunk := range aiTextChan {
			textChan <- chunk
		}

		if err, ok := <-aiErrChan; ok {
			_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
			errChan <- err
			return
		}
		_ = uc.costUC.Confirm(ctx, decision.RequestID, aiexecutor_dto.AIUsage{
			Provider: plan.Provider,
			Model:    plan.Model,
		})
	}()

	return textChan, errChan
}
