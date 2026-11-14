package main

import (
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/setup"
)

func provideChatService(repo service.ChatRepository, cfg *config.Config, models *service.ChatModelService, credentials *service.ChatCredentialService, auxiliary *service.ChatAuxiliaryService) (*service.ChatService, error) {
	directory := strings.TrimSpace(cfg.Chat.StorageDir)
	if directory == "" {
		directory = filepath.Join(setup.GetDataDir(), "chat")
	}
	storage, err := service.NewChatStorage(directory)
	if err != nil {
		return nil, err
	}
	chat, err := service.NewChatService(repo, storage, cfg, models, credentials, auxiliary)
	if err != nil {
		_ = storage.Close()
		return nil, err
	}
	return chat, nil
}
