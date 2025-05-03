package sqlite

import (
	"context"
	"database/sql"
	"shm/internal/model"
)

type ChatsRepo struct {
	db *sql.DB
}

func NewChatsRepo(db *sql.DB) *ChatsRepo {
	return &ChatsRepo{db}
}

func (s *ChatsRepo) AddChat(ctx context.Context, chat model.Chat) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chats (id, is_subscribed)
		VALUES (?, ?)
		ON CONFLICT (id) DO UPDATE SET is_subscribed = TRUE`,
		chat.Id, chat.IsSubscribed,
	)
	return err
}

func (s *ChatsRepo) UpdateChat(ctx context.Context, chat model.Chat) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE chats SET id = ?, is_subscribed = ? WHERE id = ?",
		chat.Id, chat.IsSubscribed, chat.Id,
	)
	return err
}

func (s *ChatsRepo) GetAllSubscribedOnSiteChats(
	ctx context.Context,
	url string,
) ([]model.Chat, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT c.id
		FROM chats as c
		JOIN chat_to_site as cs
		ON c.id = cs.chat_id
		JOIN sites as s
		ON cs.site_id = s.id
		WHERE c.is_subscribed = TRUE AND s.url = ?`,
		url,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []model.Chat
	for rows.Next() {
		var chat model.Chat

		err = rows.Scan(&chat.Id)
		if err != nil {
			return nil, err
		}

		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}
