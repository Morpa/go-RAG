package app

import (
	"context"

	"github.com/Morpa/go-rag/chat"
	"github.com/Morpa/go-rag/config"
	"github.com/Morpa/go-rag/llm"
)

func Run(ctx context.Context, cfg config.Config) error {
	client := llm.New(cfg)
	return chat.RunREPL(ctx, client, chat.Options{
		SystemPromptFile: cfg.SystemPromptFile,
	})
}
