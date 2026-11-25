package room

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func (s *Service) SendMessage(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input SendMessageInput,
	idempotencyKey string,
) (SendMessageResult, error) {
	if idempotencyKey == "" {
		return SendMessageResult{}, validationError("Idempotency-Key is required")
	}
	if err := id.Validate(input.ClientMessageID); err != nil {
		return SendMessageResult{}, validationError("client_message_id must be a ULID")
	}
	text := strings.TrimSpace(input.Text)
	if text == "" || utf8.RuneCountInString(text) > s.cfg.ChatMaxLength {
		return SendMessageResult{}, apperror.New(
			"MESSAGE_TOO_LONG",
			"Message text is empty or exceeds the allowed length.",
			false,
			map[string]int{"max_length": s.cfg.ChatMaxLength},
		)
	}

	decision, err := s.limiter.Allow(
		ctx,
		"room-chat:"+roomID+":"+actorUserID,
		s.cfg.ChatRateLimit,
		s.cfg.ChatRateWindow,
	)
	if err != nil {
		return SendMessageResult{}, apperror.New(
			"DEPENDENCY_ERROR",
			"Chat rate limiting is temporarily unavailable.",
			true,
			nil,
		)
	}
	if !decision.Allowed {
		return SendMessageResult{}, apperror.New(
			"RATE_LIMITED",
			"Too many room chat messages.",
			true,
			map[string]int64{"retry_after_seconds": decision.RetryAfterSeconds},
		)
	}
	now := s.now().UTC()
	muteCount, err := s.repo.CountActiveMutes(ctx, sqlc.CountActiveMutesParams{
		UserID:     actorUserID,
		StartsAt:   now,
		ExpiresAt:  sql.NullTime{Time: now, Valid: true},
		ScopeRefID: sql.NullString{String: roomID, Valid: true},
	})
	if err != nil {
		return SendMessageResult{}, mapRoomStorageError(err)
	}
	if muteCount > 0 {
		return SendMessageResult{}, apperror.New(
			"MUTED",
			"The user cannot send room chat messages.",
			false,
			nil,
		)
	}

	idempotencyHash := hashToken(idempotencyKey)
	contentHash := hashBytes([]byte(text))
	var result SendMessageResult

	err = s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		roomRecord, err := tx.GetGameRoomForUpdate(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		if isClosedStatus(roomRecord.Status) {
			return roomClosedError()
		}
		if roomRecord.Status != "waiting" {
			return roomNotWaitingError()
		}
		if _, err := tx.GetActiveRoomMemberByUser(ctx, sqlc.GetActiveRoomMemberByUserParams{
			RoomID: roomID,
			UserID: sql.NullString{String: actorUserID, Valid: true},
		}); err != nil {
			if isStorageCode(err, database.CodeNotFound) {
				return apperror.New(
					"NOT_ROOM_MEMBER",
					"The user is not an active room member.",
					false,
					nil,
				)
			}
			return mapRoomStorageError(err)
		}

		existing, replay, err := s.findExistingMessage(
			ctx,
			tx,
			roomID,
			actorUserID,
			input.ClientMessageID,
			idempotencyHash,
			contentHash,
		)
		if err != nil {
			return err
		}
		if replay {
			view, err := s.messageView(ctx, tx, existing)
			if err != nil {
				return err
			}
			result = SendMessageResult{
				Message: view,
				EventID: existing.EventID,
				ChatSeq: existing.ChatSeq,
				Replay:  true,
			}
			return nil
		}

		latest, err := tx.ListRoomMessagesBefore(ctx, sqlc.ListRoomMessagesBeforeParams{
			RoomID:  roomID,
			ChatSeq: ^uint64(0),
			Limit:   1,
		})
		if err != nil {
			return mapRoomStorageError(err)
		}
		chatSeq := uint64(1)
		if len(latest) > 0 {
			chatSeq = latest[0].ChatSeq + 1
		}

		messageID := s.ids.New()
		eventID := s.ids.New()
		sentAt := s.now().UTC()
		if err := tx.CreateRoomMessage(ctx, sqlc.CreateRoomMessageParams{
			ID:                 messageID,
			EventID:            eventID,
			RoomID:             roomID,
			ChatSeq:            chatSeq,
			SenderUserID:       actorUserID,
			ClientMessageID:    input.ClientMessageID,
			IdempotencyKeyHash: idempotencyHash,
			ContentText:        text,
			ContentHash:        contentHash,
			ModerationStatus:   "allowed",
			SchemaVersion:      1,
			SentAt:             sentAt,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		view, err := s.messageView(ctx, tx, sqlc.RoomMessage{
			ID:               messageID,
			RoomID:           roomID,
			ChatSeq:          chatSeq,
			SenderUserID:     actorUserID,
			ClientMessageID:  input.ClientMessageID,
			MessageType:      "text",
			ContentText:      text,
			ContentHash:      contentHash,
			ModerationStatus: "allowed",
			SentAt:           sentAt,
			EventID:          eventID,
			SchemaVersion:    1,
		})
		if err != nil {
			return err
		}
		result = SendMessageResult{
			Message: view,
			EventID: eventID,
			ChatSeq: chatSeq,
		}
		return nil
	})
	if err != nil {
		return SendMessageResult{}, err
	}
	return result, nil
}

func (s *Service) Messages(
	ctx context.Context,
	actorUserID string,
	roomID string,
	cursor *uint64,
	limit int,
) (MessagePage, error) {
	if err := s.requireMember(ctx, actorUserID, roomID); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	before := ^uint64(0)
	if cursor != nil {
		if *cursor == 0 {
			return MessagePage{}, apperror.New("INVALID_CURSOR", "The cursor is invalid.", false, nil)
		}
		before = *cursor
	}
	rows, err := s.repo.ListRoomMessagesBefore(ctx, sqlc.ListRoomMessagesBeforeParams{
		RoomID:  roomID,
		ChatSeq: before,
		Limit:   int32(limit + 1),
	})
	if err != nil {
		return MessagePage{}, mapRoomStorageError(err)
	}

	var nextCursor *uint64
	if len(rows) > limit {
		rows = rows[:limit]
		value := rows[len(rows)-1].ChatSeq
		nextCursor = &value
	}
	items := make([]MessageView, 0, len(rows))
	for index := len(rows) - 1; index >= 0; index-- {
		view, err := s.messageView(ctx, s.repo, rows[index])
		if err != nil {
			return MessagePage{}, err
		}
		items = append(items, view)
	}
	return MessagePage{Items: items, NextCursor: nextCursor}, nil
}

func (s *Service) findExistingMessage(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
	senderID string,
	clientMessageID string,
	idempotencyHash []byte,
	contentHash []byte,
) (sqlc.RoomMessage, bool, error) {
	byClient, err := repo.GetRoomMessageByClientMessageID(
		ctx,
		sqlc.GetRoomMessageByClientMessageIDParams{
			RoomID:          roomID,
			SenderUserID:    senderID,
			ClientMessageID: clientMessageID,
		},
	)
	if err == nil {
		if !equalBytes(byClient.ContentHash, contentHash) {
			return sqlc.RoomMessage{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"client_message_id was reused with different text.",
				false,
				nil,
			)
		}
		return byClient, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.RoomMessage{}, false, mapRoomStorageError(err)
	}

	byKey, err := repo.GetRoomMessageByIdempotencyKey(
		ctx,
		sqlc.GetRoomMessageByIdempotencyKeyParams{
			RoomID:             roomID,
			SenderUserID:       senderID,
			IdempotencyKeyHash: idempotencyHash,
		},
	)
	if err == nil {
		if !equalBytes(byKey.ContentHash, contentHash) ||
			byKey.ClientMessageID != clientMessageID {
			return sqlc.RoomMessage{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"Idempotency-Key was reused with a different request.",
				false,
				nil,
			)
		}
		return byKey, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.RoomMessage{}, false, mapRoomStorageError(err)
	}
	return sqlc.RoomMessage{}, false, nil
}

func (s *Service) messageView(
	ctx context.Context,
	repo storage.Repository,
	message sqlc.RoomMessage,
) (MessageView, error) {
	profile, err := repo.GetUserProfile(ctx, message.SenderUserID)
	if err != nil {
		return MessageView{}, mapRoomStorageError(err)
	}
	return MessageView{
		MessageID: message.ID,
		EventID:   message.EventID,
		Sender: SenderView{
			UserID:      message.SenderUserID,
			DisplayName: profile.Nickname,
			AvatarURL:   nullableString(profile.AvatarUri),
		},
		Text:             message.ContentText,
		ChatSeq:          message.ChatSeq,
		SentAt:           message.SentAt,
		ModerationStatus: message.ModerationStatus,
	}, nil
}

func (s *Service) RoomEvents(
	ctx context.Context,
	actorUserID string,
	roomID string,
	afterSeq uint64,
	limit int,
) ([]RoomEventView, uint64, error) {
	if err := s.requireMember(ctx, actorUserID, roomID); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.repo.ListRoomEventsAfter(ctx, sqlc.ListRoomEventsAfterParams{
		RoomID: roomID,
		Seq:    afterSeq,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, 0, mapRoomStorageError(err)
	}
	events := make([]RoomEventView, 0, len(rows))
	current := afterSeq
	for _, row := range rows {
		events = append(events, RoomEventView{
			EventID:       row.ID,
			EventType:     row.EventType,
			Stream:        "room",
			StreamID:      row.RoomID,
			Seq:           row.Seq,
			SchemaVersion: 1,
			ActorType:     nullableString(row.ActorType),
			ActorID:       nullableString(row.ActorID),
			Timestamp:     row.OccurredAt,
			Payload:       row.PayloadJson,
		})
		current = row.Seq
	}
	return events, current, nil
}

func equalBytes(left []byte, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	result := byte(0)
	for index := range left {
		result |= left[index] ^ right[index]
	}
	return result == 0
}
