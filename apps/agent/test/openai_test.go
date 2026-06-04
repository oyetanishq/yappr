package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/oyetanishq/yappr/apps/agent/internal/service/reviewer"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"go.uber.org/zap"
)

func TestOpenAICallLLM(t *testing.T) {
	// Setup a dummy config pointing to our mock server
	cfg, err := config.Load("../.env.test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logger := zap.NewNop()
	openAIReviewer := reviewer.NewOpenAIReviewer(cfg, logger)

	// Call the LLM (which will hit our mock server)
	ctx := context.Background()
	systemPrompt := "You are a helpful assistant."
	userMessage := "Hello world"

	response, err := openAIReviewer.CallLLM(ctx, systemPrompt, userMessage, cfg.OpenAI.BaseModel)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	fmt.Println(response)
}
