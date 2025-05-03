package service

import (
	"context"
	"shm/internal/config"
	"shm/internal/model"
	"shm/internal/repository"
)

type ChatsService struct {
	chats  repository.ChatsProvider
	config config.CommonConfig
}

func NewChatsService(chats repository.ChatsProvider, config config.CommonConfig) *ChatsService {
	return &ChatsService{
		chats:  chats,
		config: config,
	}
}

func (c *ChatsService) SubscribeChat(ctx context.Context, chatId int64) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.DbQueryTimeoutSec)
	defer cancel()
	// TODO if chat is already exists?
	return c.chats.AddChat(ctx, model.Chat{Id: chatId, IsSubscribed: true})
}

func (c *ChatsService) UnsubscribeChat(ctx context.Context, chatId int64) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.DbQueryTimeoutSec)
	defer cancel()
	// TODO if chat is not exists?
	return c.chats.UpdateChat(ctx, model.Chat{Id: chatId, IsSubscribed: false})
}

func (c *ChatsService) GetAllSubscribedOnSiteChats(
	ctx context.Context,
	url string,
) ([]model.Chat, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.DbQueryTimeoutSec)
	defer cancel()

	return c.chats.GetAllSubscribedOnSiteChats(ctx, url)
}
