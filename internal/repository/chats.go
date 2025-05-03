package repository

import (
	"context"
	"shm/internal/model"
)

type ChatsProvider interface {
	AddChat(ctx context.Context, chat model.Chat) error
	UpdateChat(ctx context.Context, chat model.Chat) error
	GetAllSubscribedOnSiteChats(ctx context.Context, url string) ([]model.Chat, error)
}
