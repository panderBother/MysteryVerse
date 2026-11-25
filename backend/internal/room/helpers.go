package room

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func (s *Service) validateCreateInput(
	ctx context.Context,
	actorUserID string,
	input CreateInput,
) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.GameType = strings.TrimSpace(input.GameType)
	input.Visibility = strings.TrimSpace(input.Visibility)
	input.Password = strings.TrimSpace(input.Password)
	if input.RuleTemplateID != nil {
		value := strings.TrimSpace(*input.RuleTemplateID)
		if value == "" {
			input.RuleTemplateID = nil
		} else {
			input.RuleTemplateID = &value
		}
	}

	if err := id.Validate(input.ScriptVersionID); err != nil {
		return CreateInput{}, apperror.New(
			"CONTENT_UNAVAILABLE",
			"The requested script version is unavailable.",
			false,
			nil,
		)
	}
	if input.RuleTemplateID != nil && id.Validate(*input.RuleTemplateID) != nil {
		return CreateInput{}, validationError("rule_template_id must be a ULID")
	}
	if input.Title == "" || utf8.RuneCountInString(input.Title) > 120 {
		return CreateInput{}, validationError("title must contain between 1 and 120 characters")
	}
	if !oneOf(input.GameType, "script_murder", "werewolf", "rpg") {
		return CreateInput{}, validationError("game_type is invalid")
	}
	if !oneOf(input.Visibility, "private", "public", "password", "invite_only") {
		return CreateInput{}, validationError("visibility is invalid")
	}
	if input.Visibility == "password" {
		if length := utf8.RuneCountInString(input.Password); length < 4 || length > 64 {
			return CreateInput{}, validationError("password must contain between 4 and 64 characters")
		}
	}
	if input.Capacity.Min <= 0 || input.Capacity.Max <= 0 || input.Capacity.Min > input.Capacity.Max {
		return CreateInput{}, validationError("capacity is invalid")
	}
	if !input.PrivacyConfirmed {
		return CreateInput{}, apperror.New(
			"VALIDATION_ERROR",
			"privacy_confirmed must be true.",
			false,
			nil,
		)
	}
	if input.ModeConfig == nil {
		input.ModeConfig = map[string]any{}
	}
	input.ModeConfig["schema_version"] = "1.0.0"

	version, err := s.repo.GetScriptVersionForRoom(ctx, input.ScriptVersionID)
	if err != nil {
		if isStorageCode(err, database.CodeNotFound) {
			return CreateInput{}, apperror.New(
				"CONTENT_UNAVAILABLE",
				"The requested script version is unavailable.",
				false,
				nil,
			)
		}
		return CreateInput{}, mapRoomStorageError(err)
	}
	if version.GameType != input.GameType ||
		!playableVersion(version, actorUserID) {
		return CreateInput{}, apperror.New(
			"CONTENT_UNAVAILABLE",
			"The requested script version cannot be played.",
			false,
			nil,
		)
	}
	if input.Capacity.Min < int(version.MinPlayers) ||
		input.Capacity.Max > int(version.MaxPlayers) {
		return CreateInput{}, apperror.New(
			"PLAYER_COUNT_MISMATCH",
			"The room capacity is outside the script player range.",
			false,
			nil,
		)
	}
	return input, nil
}

func playableVersion(version sqlc.GetScriptVersionForRoomRow, actorUserID string) bool {
	if version.OwnerUserID == actorUserID &&
		oneOf(version.VersionStatus, "draft", "validating", "review") {
		return true
	}
	return oneOf(version.VersionStatus, "published_private", "published_public", "unlisted") &&
		oneOf(version.ScriptStatus, "published", "unlisted")
}

func (s *Service) projectionFor(
	ctx context.Context,
	repo storage.Repository,
	roomRecord sqlc.GameRoom,
	actorUserID string,
) (Projection, error) {
	members, err := s.listMembers(ctx, repo, roomRecord.ID)
	if err != nil {
		return Projection{}, err
	}
	readyCount := int64(0)
	for _, member := range members {
		if member.MemberStatus == "ready" {
			readyCount++
		}
	}
	activeCount := int64(len(members))
	canStart := activeCount >= int64(roomRecord.CapacityMin) && activeCount == readyCount
	capabilities := []string{"leave_room", "send_waiting_message"}
	if roomRecord.Status == "waiting" {
		for _, member := range members {
			if member.UserID == actorUserID {
				if member.MemberStatus == "ready" {
					capabilities = append(capabilities, "unready")
				} else {
					capabilities = append(capabilities, "ready")
				}
				break
			}
		}
	}
	if roomRecord.HostUserID == actorUserID && roomRecord.Status == "waiting" {
		capabilities = append(capabilities, "close_room", "transfer_host", "kick")
	}
	return Projection{
		Room:               roomView(roomRecord),
		Members:            members,
		ViewerCapabilities: capabilities,
		RoomEventSeq:       roomRecord.CurrentEventSeq,
		CanStart:           canStart,
	}, nil
}

func (s *Service) listMembers(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
) ([]MemberView, error) {
	rows, err := repo.ListActiveRoomMembers(ctx, roomID)
	if err != nil {
		return nil, mapRoomStorageError(err)
	}
	members := make([]MemberView, 0, len(rows))
	for _, row := range rows {
		members = append(members, MemberView{
			ID:           row.ID,
			UserID:       row.UserID.String,
			PublicNo:     row.PublicNo,
			Nickname:     row.Nickname,
			AvatarURL:    nullableString(row.AvatarUri),
			SeatNo:       nullableInt16(row.SeatNo),
			RoomRole:     row.RoomRole,
			MemberStatus: row.MemberStatus,
			ReadyAt:      nullableTime(row.ReadyAt),
			JoinedAt:     row.JoinedAt,
		})
	}
	return members, nil
}

func (s *Service) memberView(
	ctx context.Context,
	repo storage.Repository,
	member sqlc.RoomMember,
) (MemberView, error) {
	user, err := repo.GetUser(ctx, member.UserID.String)
	if err != nil {
		return MemberView{}, mapRoomStorageError(err)
	}
	profile, err := repo.GetUserProfile(ctx, member.UserID.String)
	if err != nil {
		return MemberView{}, mapRoomStorageError(err)
	}
	return mergeMemberIdentity(MemberView{
		ID:           member.ID,
		UserID:       member.UserID.String,
		SeatNo:       nullableInt16(member.SeatNo),
		RoomRole:     member.RoomRole,
		MemberStatus: member.MemberStatus,
		ReadyAt:      nullableTime(member.ReadyAt),
		JoinedAt:     member.JoinedAt,
	}, user, profile), nil
}

func mergeMemberProfile(member MemberView, profile sqlc.UserProfile) MemberView {
	member.Nickname = profile.Nickname
	member.AvatarURL = nullableString(profile.AvatarUri)
	return member
}

func mergeMemberIdentity(
	member MemberView,
	user sqlc.User,
	profile sqlc.UserProfile,
) MemberView {
	member = mergeMemberProfile(member, profile)
	member.PublicNo = user.PublicNo
	return member
}

func (s *Service) requireMember(
	ctx context.Context,
	actorUserID string,
	roomID string,
) error {
	_, err := s.repo.GetActiveRoomMemberByUser(
		ctx,
		sqlc.GetActiveRoomMemberByUserParams{
			RoomID: roomID,
			UserID: sql.NullString{String: actorUserID, Valid: true},
		},
	)
	if err != nil {
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
	return nil
}

func (s *Service) appendEvent(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
	seq uint64,
	eventType string,
	actorUserID string,
	payload map[string]any,
) error {
	raw, _ := json.Marshal(payload)
	if err := repo.CreateRoomEvent(ctx, sqlc.CreateRoomEventParams{
		ID:              s.ids.New(),
		RoomID:          roomID,
		Seq:             seq,
		EventType:       eventType,
		ActorType:       sql.NullString{String: "user", Valid: actorUserID != ""},
		ActorID:         sql.NullString{String: actorUserID, Valid: actorUserID != ""},
		VisibilityScope: "room",
		PayloadJson:     raw,
		PayloadHash:     hashBytes(raw),
		OccurredAt:      s.now().UTC(),
	}); err != nil {
		return mapRoomStorageError(err)
	}
	return nil
}

func (s *Service) expireLockedRoom(
	ctx context.Context,
	repo storage.Repository,
	roomRecord sqlc.GameRoom,
) error {
	seq := roomRecord.CurrentEventSeq + 1
	if err := s.appendEvent(ctx, repo, roomRecord.ID, seq, "room.closed", "", map[string]any{
		"schema_version": "1.0.0",
		"reason":         "waiting_room_expired",
	}); err != nil {
		return err
	}
	if err := repo.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
		Status:          "cancelled",
		CurrentEventSeq: seq,
		StartedAt:       roomRecord.StartedAt,
		ClosedAt:        sql.NullTime{Time: s.now().UTC(), Valid: true},
		ID:              roomRecord.ID,
	}); err != nil {
		return mapRoomStorageError(err)
	}
	return nil
}

func (s *Service) idempotency(
	ctx context.Context,
	repo storage.Repository,
	scope string,
	actorScope string,
	key string,
	requestHash []byte,
) (sqlc.IdempotencyRecord, bool, error) {
	record, err := repo.GetIdempotencyRecord(ctx, sqlc.GetIdempotencyRecordParams{
		Scope:          scope,
		ActorScope:     actorScope,
		IdempotencyKey: key,
	})
	if err == nil {
		if !hmac.Equal(record.RequestHash, requestHash) {
			return sqlc.IdempotencyRecord{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The idempotency key was reused with a different request.",
				false,
				nil,
			)
		}
		if record.Status != "completed" {
			return sqlc.IdempotencyRecord{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The previous request is still in progress.",
				true,
				nil,
			)
		}
		return record, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.IdempotencyRecord{}, false, mapRoomStorageError(err)
	}

	recordID := s.ids.New()
	if err := repo.CreateIdempotencyRecord(ctx, sqlc.CreateIdempotencyRecordParams{
		ID:             recordID,
		Scope:          scope,
		ActorScope:     actorScope,
		IdempotencyKey: key,
		RequestHash:    requestHash,
		ExpiresAt:      s.now().UTC().Add(24 * time.Hour),
	}); err != nil {
		return sqlc.IdempotencyRecord{}, false, mapRoomStorageError(err)
	}
	return sqlc.IdempotencyRecord{ID: recordID}, false, nil
}

func chooseSeat(
	maxSeats int16,
	members []sqlc.ListActiveRoomMembersRow,
	preferred *int16,
) (int16, error) {
	occupied := make(map[int16]bool, len(members))
	for _, member := range members {
		if member.ActiveSeatNo.Valid {
			occupied[member.ActiveSeatNo.Int16] = true
		}
	}
	if preferred != nil {
		if *preferred < 1 || *preferred > maxSeats {
			return 0, validationError("preferred_seat_no is outside the room capacity")
		}
		if occupied[*preferred] {
			return 0, apperror.New("ROOM_FULL", "The preferred seat is unavailable.", false, nil)
		}
		return *preferred, nil
	}
	for seat := int16(1); seat <= maxSeats; seat++ {
		if !occupied[seat] {
			return seat, nil
		}
	}
	return 0, roomFullError()
}

func (s *Service) createInvitation(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
	createdBy string,
	maxUses uint32,
	ttl time.Duration,
	inviteeUserIDs []string,
) (string, error) {
	rawToken, err := randomToken()
	if err != nil {
		return "", internalError()
	}
	var rule *json.RawMessage
	if len(inviteeUserIDs) > 0 {
		raw, _ := json.Marshal(map[string]any{"user_ids": inviteeUserIDs})
		value := json.RawMessage(raw)
		rule = &value
	}
	if err := repo.CreateInvitationToken(ctx, sqlc.CreateInvitationTokenParams{
		ID:              s.ids.New(),
		RoomID:          roomID,
		CreatedBy:       createdBy,
		TokenHash:       hashToken(rawToken),
		MaxUses:         maxUses,
		ExpiresAt:       s.now().UTC().Add(ttl),
		InviteeRuleJson: rule,
	}); err != nil {
		return "", mapRoomStorageError(err)
	}
	return rawToken, nil
}

func invitationAllows(rule *json.RawMessage, userID string) bool {
	if rule == nil {
		return true
	}
	var parsed struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.Unmarshal(*rule, &parsed); err != nil {
		return false
	}
	if len(parsed.UserIDs) == 0 {
		return false
	}
	for _, allowed := range parsed.UserIDs {
		if allowed == userID {
			return true
		}
	}
	return false
}

func roomView(roomRecord sqlc.GameRoom) RoomView {
	return RoomView{
		ID:              roomRecord.ID,
		GameType:        roomRecord.GameType,
		ScriptVersionID: roomRecord.ScriptVersionID,
		Title:           roomRecord.Title,
		HostUserID:      roomRecord.HostUserID,
		Visibility:      roomRecord.Visibility,
		HasPassword:     roomRecord.PasswordHash.Valid,
		Capacity: Capacity{
			Min: int(roomRecord.CapacityMin),
			Max: int(roomRecord.CapacityMax),
		},
		Status:          roomRecord.Status,
		CurrentEventSeq: roomRecord.CurrentEventSeq,
		ExpiresAt:       nullableTime(roomRecord.ExpiresAt),
		RowVersion:      roomRecord.RowVersion,
	}
}

func roomViewFromParams(
	params sqlc.CreateGameRoomParams,
	eventSeq uint64,
	rowVersion uint32,
) RoomView {
	return RoomView{
		ID:              params.ID,
		GameType:        params.GameType,
		ScriptVersionID: params.ScriptVersionID,
		Title:           params.Title,
		HostUserID:      params.HostUserID,
		Visibility:      params.Visibility,
		HasPassword:     params.PasswordHash.Valid,
		Capacity: Capacity{
			Min: int(params.CapacityMin),
			Max: int(params.CapacityMax),
		},
		Status:          params.Status,
		CurrentEventSeq: eventSeq,
		ExpiresAt:       nullableTime(params.ExpiresAt),
		RowVersion:      rowVersion,
	}
}

func isExpired(roomRecord sqlc.GameRoom, now time.Time) bool {
	return roomRecord.Status == "waiting" &&
		roomRecord.ExpiresAt.Valid &&
		!now.Before(roomRecord.ExpiresAt.Time)
}

func isClosedStatus(status string) bool {
	return status == "closed" || status == "cancelled" || status == "aborted"
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableInt16(value sql.NullInt16) *int16 {
	if !value.Valid {
		return nil
	}
	result := value.Int16
	return &result
}

func int16Pointer(value int16) *int16 {
	return &value
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func nullableStringParam(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func hashJSON(value any) []byte {
	raw, _ := json.Marshal(value)
	return hashBytes(raw)
}

func hashBytes(value []byte) []byte {
	sum := sha256.Sum256(value)
	return sum[:]
}

func hashToken(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func randomToken() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (s *Service) roomCodeHash(roomID string) []byte {
	mac := hmac.New(sha256.New, s.hmacKey)
	_, _ = mac.Write([]byte("room-code:" + roomID))
	return mac.Sum(nil)
}

func mapRoomStorageError(err error) error {
	var storageErr *database.Error
	if errors.As(err, &storageErr) {
		switch storageErr.Code {
		case database.CodeNotFound:
			return roomNotFoundError()
		case database.CodeConflict:
			return roomNotWaitingError()
		case database.CodeDependencyUnavailable, database.CodeDeadlineExceeded:
			return apperror.New(
				"DEPENDENCY_ERROR",
				"The room service is temporarily unavailable.",
				true,
				nil,
			)
		}
	}
	return internalError()
}

func isStorageCode(err error, code string) bool {
	var storageErr *database.Error
	return errors.As(err, &storageErr) && storageErr.Code == code
}

func oneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func validationError(message string) error {
	return apperror.New("VALIDATION_ERROR", message, false, nil)
}

func internalError() error {
	return apperror.New("INTERNAL_ERROR", "The room operation failed.", true, nil)
}

func roomNotFoundError() error {
	return apperror.New("ROOM_NOT_FOUND", "The room does not exist or is not visible.", false, nil)
}

func roomFullError() error {
	return apperror.New("ROOM_FULL", "The room has no available seats.", false, nil)
}

func roomNotWaitingError() error {
	return apperror.New("ROOM_NOT_WAITING", "The room is not waiting for players.", false, nil)
}

func roomClosedError() error {
	return apperror.New("ROOM_CLOSED", "The room is closed or expired.", false, nil)
}

func hostRequiredError() error {
	return apperror.New("HOST_REQUIRED", "The current user is not the room host.", false, nil)
}

func notRoomMemberError() error {
	return apperror.New("NOT_ROOM_MEMBER", "The user is not an active room member.", false, nil)
}

func versionConflictError() error {
	return apperror.New(
		"STATE_VERSION_CONFLICT",
		"The room state changed. Refresh and retry.",
		true,
		nil,
	)
}

func targetNotMemberError() error {
	return apperror.New("TARGET_NOT_MEMBER", "The target is not an active room member.", false, nil)
}

func invitationInvalidError() error {
	return apperror.New("INVITATION_INVALID", "The invitation is invalid or unavailable.", false, nil)
}
