package repository

import (
	"context"
	"database/sql"
	"shm/internal/model"
)

type Chats struct {
	db *sql.DB
}

func NewChatsRepo(db *sql.DB) *Chats {
	return &Chats{db}
}

func (s *Chats) AddChat(ctx context.Context, chat model.Chat) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chats (id, is_subscribed) 
		VALUES (?, TRUE) 
		ON CONFLICT (id) DO UPDATE SET is_subscribed = TRUE`,
		chat.Id,
	)
	return err
}

func (s *Chats) UpdateChat(ctx context.Context, chatId int64, isSub bool) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE chats SET is_subscribed = ? WHERE id = ?",
		isSub, chatId,
	)
	return err
}

func (s *Chats) GetAllSubscribedOnSiteChats(
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
