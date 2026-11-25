package room

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/security"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type Service struct {
	repo      storage.Repository
	ids       id.Generator
	passwords *security.PasswordHasher
	limiter   ratelimit.Limiter
	cfg       config.AuthConfig
	now       func() time.Time
	hmacKey   []byte
}

func NewService(
	repo storage.Repository,
	cfg config.AuthConfig,
	limiter ratelimit.Limiter,
) *Service {
	return &Service{
		repo:      repo,
		ids:       id.ULIDGenerator{},
		passwords: security.NewPasswordHasher(security.Argon2Params{}),
		limiter:   limiter,
		cfg:       cfg,
		now:       time.Now,
		hmacKey:   []byte(cfg.IdentityHMACKey),
	}
}

func (s *Service) Create(
	ctx context.Context,
	actorUserID string,
	input CreateInput,
	idempotencyKey string,
) (CreateResult, error) {
	if idempotencyKey == "" {
		return CreateResult{}, validationError("Idempotency-Key is required")
	}
	normalized, err := s.validateCreateInput(ctx, actorUserID, input)
	if err != nil {
		return CreateResult{}, err
	}

	var passwordHash string
	if normalized.Visibility == "password" {
		passwordHash, err = s.passwords.Hash(normalized.Password)
		if err != nil {
			return CreateResult{}, internalError()
		}
	}

	requestHash := hashJSON(normalized)
	actorScope := "user:" + actorUserID
	var result CreateResult

	err = s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(ctx, tx, "room.create", actorScope, idempotencyKey, requestHash)
		if err != nil {
			return err
		}
		if replay && record.ResourceID.Valid {
			roomRecord, err := tx.GetGameRoom(ctx, record.ResourceID.String)
			if err != nil {
				return mapRoomStorageError(err)
			}
			memberRecord, err := tx.GetActiveRoomMemberByUser(
				ctx,
				sqlc.GetActiveRoomMemberByUserParams{
					RoomID: roomRecord.ID,
					UserID: sql.NullString{String: actorUserID, Valid: true},
				},
			)
			if err != nil {
				return mapRoomStorageError(err)
			}
			member, err := s.memberView(ctx, tx, memberRecord)
			if err != nil {
				return err
			}
			result = CreateResult{
				Room:             roomView(roomRecord),
				Member:           member,
				InviteHint:       InviteHint{CodeEnabled: true},
				RoomEventSeq:     roomRecord.CurrentEventSeq,
				IdempotentReplay: true,
			}
			return nil
		}

		roomID := s.ids.New()
		memberID := s.ids.New()
		now := s.now().UTC()
		expiresAt := now.Add(s.cfg.RoomWaitingTTL)
		modeJSON, _ := json.Marshal(normalized.ModeConfig)

		roomParams := sqlc.CreateGameRoomParams{
			ID:                      roomID,
			RoomCodeHash:            s.roomCodeHash(roomID),
			HostUserID:              actorUserID,
			ScriptVersionID:         normalized.ScriptVersionID,
			GameType:                normalized.GameType,
			RuleTemplateID:          nullableStringParam(normalized.RuleTemplateID),
			Title:                   normalized.Title,
			Visibility:              normalized.Visibility,
			PasswordHash:            sql.NullString{String: passwordHash, Valid: passwordHash != ""},
			CapacityMin:             uint8(normalized.Capacity.Min),
			CapacityMax:             uint8(normalized.Capacity.Max),
			ModeConfigJson:          modeJSON,
			ModeConfigSchemaVersion: "1.0.0",
			PrivacyPolicyVersion:    "2026-11",
			Status:                  "waiting",
			ExpiresAt:               sql.NullTime{Time: expiresAt, Valid: true},
		}
		if err := tx.CreateGameRoom(ctx, roomParams); err != nil {
			return mapRoomStorageError(err)
		}
		if err := tx.CreateRoomMember(ctx, sqlc.CreateRoomMemberParams{
			ID:              memberID,
			RoomID:          roomID,
			UserID:          sql.NullString{String: actorUserID, Valid: true},
			ParticipantKind: "human",
			SeatNo:          sql.NullInt16{Int16: 1, Valid: true},
			RoomRole:        "host",
			MemberStatus:    "joined",
			JoinedAt:        now,
			ConnectionState: sql.NullString{String: "online", Valid: true},
		}); err != nil {
			return mapRoomStorageError(err)
		}

		seq := uint64(1)
		if err := s.appendEvent(ctx, tx, roomID, seq, "room.created", actorUserID, map[string]any{
			"schema_version": "1.0.0",
			"host_user_id":   actorUserID,
			"member_id":      memberID,
		}); err != nil {
			return err
		}
		if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
			Status:          "waiting",
			CurrentEventSeq: seq,
			ID:              roomID,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		var invitationToken string
		if normalized.Visibility == "private" || normalized.Visibility == "invite_only" {
			invitationToken, err = s.createInvitation(
				ctx,
				tx,
				roomID,
				actorUserID,
				10,
				time.Hour,
				nil,
			)
			if err != nil {
				return err
			}
		}

		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 201, Valid: true},
			ResourceType:   sql.NullString{String: "room", Valid: true},
			ResourceID:     sql.NullString{String: roomID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		result = CreateResult{
			Room: roomViewFromParams(roomParams, seq, 1),
			Member: MemberView{
				ID:           memberID,
				UserID:       actorUserID,
				SeatNo:       int16Pointer(1),
				RoomRole:     "host",
				MemberStatus: "joined",
				JoinedAt:     now,
			},
			InviteHint:      InviteHint{CodeEnabled: true},
			RoomEventSeq:    seq,
			InvitationToken: invitationToken,
		}
		return nil
	})
	if err != nil {
		return CreateResult{}, err
	}
	if result.Member.Nickname == "" {
		user, err := s.repo.GetUser(ctx, result.Member.UserID)
		if err != nil {
			return CreateResult{}, mapRoomStorageError(err)
		}
		profile, err := s.repo.GetUserProfile(ctx, result.Member.UserID)
		if err != nil {
			return CreateResult{}, mapRoomStorageError(err)
		}
		result.Member = mergeMemberIdentity(result.Member, user, profile)
	}
	return result, nil
}

func (s *Service) Join(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input JoinInput,
	idempotencyKey string,
) (JoinResult, error) {
	if idempotencyKey == "" {
		return JoinResult{}, validationError("Idempotency-Key is required")
	}
	if err := id.Validate(roomID); err != nil {
		return JoinResult{}, roomNotFoundError()
	}

	var result JoinResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		roomRecord, err := tx.GetGameRoomForUpdate(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		now := s.now().UTC()
		if isExpired(roomRecord, now) {
			if err := s.expireLockedRoom(ctx, tx, roomRecord); err != nil {
				return err
			}
			return roomClosedError()
		}
		if isClosedStatus(roomRecord.Status) {
			return roomClosedError()
		}
		if roomRecord.Status != "waiting" {
			return roomNotWaitingError()
		}

		existing, err := tx.GetActiveRoomMemberByUser(
			ctx,
			sqlc.GetActiveRoomMemberByUserParams{
				RoomID: roomID,
				UserID: sql.NullString{String: actorUserID, Valid: true},
			},
		)
		if err == nil {
			member, err := s.memberView(ctx, tx, existing)
			if err != nil {
				return err
			}
			result = JoinResult{
				Member:           member,
				RoomProjection:   roomView(roomRecord),
				RoomEventSeq:     roomRecord.CurrentEventSeq,
				IdempotentReplay: true,
			}
			return nil
		}
		if !isStorageCode(err, database.CodeNotFound) {
			return mapRoomStorageError(err)
		}

		var invitation *sqlc.InvitationToken
		if roomRecord.Visibility == "private" || roomRecord.Visibility == "invite_only" {
			if strings.TrimSpace(input.InvitationToken) == "" {
				return invitationInvalidError()
			}
			invitationRecord, err := tx.GetInvitationForUpdate(ctx, hashToken(input.InvitationToken))
			if err != nil {
				return invitationInvalidError()
			}
			if invitationRecord.RoomID != roomID ||
				invitationRecord.RevokedAt.Valid ||
				!now.Before(invitationRecord.ExpiresAt) ||
				invitationRecord.UseCount >= invitationRecord.MaxUses ||
				!invitationAllows(invitationRecord.InviteeRuleJson, actorUserID) {
				return invitationInvalidError()
			}
			invitation = &invitationRecord
		}
		if roomRecord.Visibility == "password" {
			if !roomRecord.PasswordHash.Valid {
				return internalError()
			}
			valid, err := s.passwords.Verify(roomRecord.PasswordHash.String, input.Password)
			if err != nil {
				return internalError()
			}
			if !valid {
				return apperror.New(
					"PASSWORD_INVALID",
					"The room password is invalid.",
					false,
					nil,
				)
			}
		}

		members, err := tx.ListActiveRoomMembers(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		if int64(len(members)) >= int64(roomRecord.CapacityMax) {
			return roomFullError()
		}
		seat, err := chooseSeat(int16(roomRecord.CapacityMax), members, input.PreferredSeatNo)
		if err != nil {
			return err
		}

		memberID := s.ids.New()
		if err := tx.CreateRoomMember(ctx, sqlc.CreateRoomMemberParams{
			ID:              memberID,
			RoomID:          roomID,
			UserID:          sql.NullString{String: actorUserID, Valid: true},
			ParticipantKind: "human",
			SeatNo:          sql.NullInt16{Int16: seat, Valid: true},
			RoomRole:        "player",
			MemberStatus:    "joined",
			JoinedAt:        now,
			ConnectionState: sql.NullString{String: "online", Valid: true},
		}); err != nil {
			return mapRoomStorageError(err)
		}
		if invitation != nil {
			if err := tx.IncrementInvitationUse(ctx, invitation.ID); err != nil {
				return mapRoomStorageError(err)
			}
		}

		seq := roomRecord.CurrentEventSeq + 1
		if err := s.appendEvent(ctx, tx, roomID, seq, "room.member_joined", actorUserID, map[string]any{
			"schema_version": "1.0.0",
			"member_id":      memberID,
			"seat_no":        seat,
		}); err != nil {
			return err
		}
		if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
			Status:          roomRecord.Status,
			CurrentEventSeq: seq,
			StartedAt:       roomRecord.StartedAt,
			ClosedAt:        roomRecord.ClosedAt,
			ID:              roomID,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		member, err := s.memberView(ctx, tx, sqlc.RoomMember{
			ID:              memberID,
			RoomID:          roomID,
			UserID:          sql.NullString{String: actorUserID, Valid: true},
			ParticipantKind: "human",
			SeatNo:          sql.NullInt16{Int16: seat, Valid: true},
			RoomRole:        "player",
			MemberStatus:    "joined",
			JoinedAt:        now,
			ConnectionState: sql.NullString{String: "online", Valid: true},
			RowVersion:      1,
			ActiveSeatNo:    sql.NullInt16{Int16: seat, Valid: true},
		})
		if err != nil {
			return err
		}
		roomRecord.CurrentEventSeq = seq
		result = JoinResult{
			Member:         member,
			RoomProjection: roomView(roomRecord),
			RoomEventSeq:   seq,
		}
		return nil
	})
	if err != nil {
		return JoinResult{}, err
	}
	return result, nil
}

func (s *Service) Projection(
	ctx context.Context,
	actorUserID string,
	roomID string,
) (Projection, error) {
	roomRecord, err := s.repo.GetGameRoom(ctx, roomID)
	if err != nil {
		return Projection{}, mapRoomStorageError(err)
	}
	if isExpired(roomRecord, s.now().UTC()) {
		if err := s.ExpireRoom(ctx, roomID); err != nil {
			return Projection{}, err
		}
		return Projection{}, roomClosedError()
	}
	if roomRecord.Status == "closed" || roomRecord.Status == "cancelled" || roomRecord.Status == "aborted" {
		return Projection{}, roomClosedError()
	}
	if err := s.requireMember(ctx, actorUserID, roomID); err != nil {
		return Projection{}, err
	}
	return s.projectionFor(ctx, s.repo, roomRecord, actorUserID)
}

func (s *Service) Leave(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input LeaveInput,
	idempotencyKey string,
) (LeaveResult, error) {
	if idempotencyKey == "" {
		return LeaveResult{}, validationError("Idempotency-Key is required")
	}

	var result LeaveResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		roomRecord, err := tx.GetGameRoomForUpdate(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		switch roomRecord.Status {
		case "waiting":
		case "closed", "cancelled", "aborted":
			return roomClosedError()
		default:
			return apperror.New("ROOM_IN_GAME", "The room is already in game.", false, nil)
		}
		if input.ExpectedRoomEventSeq != roomRecord.CurrentEventSeq {
			return versionConflictError()
		}

		member, err := tx.GetActiveRoomMemberByUser(
			ctx,
			sqlc.GetActiveRoomMemberByUserParams{
				RoomID: roomID,
				UserID: sql.NullString{String: actorUserID, Valid: true},
			},
		)
		if err != nil {
			if isStorageCode(err, database.CodeNotFound) {
				latest, latestErr := tx.GetLatestRoomMemberByUser(
					ctx,
					sqlc.GetLatestRoomMemberByUserParams{
						RoomID: roomID,
						UserID: sql.NullString{String: actorUserID, Valid: true},
					},
				)
				if latestErr == nil && latest.LeftAt.Valid {
					result = LeaveResult{
						MemberStatus: "left",
						RoomStatus:   roomRecord.Status,
					}
					return nil
				}
				return notRoomMemberError()
			}
			return mapRoomStorageError(err)
		}
		now := s.now().UTC()
		if err := tx.UpdateRoomMemberLeave(ctx, sqlc.UpdateRoomMemberLeaveParams{
			LeftAt: sql.NullTime{Time: now, Valid: true},
			ID:     member.ID,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		seq := roomRecord.CurrentEventSeq + 1
		if err := s.appendEvent(ctx, tx, roomID, seq, "room.member_left", actorUserID, map[string]any{
			"schema_version": "1.0.0",
			"member_id":      member.ID,
			"reason":         input.Reason,
		}); err != nil {
			return err
		}

		roomStatus := roomRecord.Status
		transferRequired := roomRecord.HostUserID == actorUserID
		if transferRequired {
			members, err := tx.ListActiveRoomMembers(ctx, roomID)
			if err != nil {
				return mapRoomStorageError(err)
			}
			if len(members) == 0 {
				seq++
				if err := s.appendEvent(ctx, tx, roomID, seq, "room.closed", actorUserID, map[string]any{
					"schema_version": "1.0.0",
					"reason":         "host_left_no_members",
				}); err != nil {
					return err
				}
				roomStatus = "cancelled"
				if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
					Status:          roomStatus,
					CurrentEventSeq: seq,
					StartedAt:       roomRecord.StartedAt,
					ClosedAt:        sql.NullTime{Time: now, Valid: true},
					ID:              roomID,
				}); err != nil {
					return mapRoomStorageError(err)
				}
			} else {
				target := members[0]
				if err := tx.UpdateRoomMemberRole(ctx, sqlc.UpdateRoomMemberRoleParams{
					RoomRole: "host",
					ID:       target.ID,
				}); err != nil {
					return mapRoomStorageError(err)
				}
				if err := tx.UpdateGameRoomHost(ctx, sqlc.UpdateGameRoomHostParams{
					HostUserID:   target.UserID.String,
					ID:           roomID,
					HostUserID_2: actorUserID,
				}); err != nil {
					return mapRoomStorageError(err)
				}
				seq++
				if err := s.appendEvent(ctx, tx, roomID, seq, "room.host_transferred", actorUserID, map[string]any{
					"schema_version":   "1.0.0",
					"new_host_user_id": target.UserID.String,
				}); err != nil {
					return err
				}
				if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
					Status:          roomStatus,
					CurrentEventSeq: seq,
					StartedAt:       roomRecord.StartedAt,
					ClosedAt:        roomRecord.ClosedAt,
					ID:              roomID,
				}); err != nil {
					return mapRoomStorageError(err)
				}
			}
		} else {
			if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
				Status:          roomStatus,
				CurrentEventSeq: seq,
				StartedAt:       roomRecord.StartedAt,
				ClosedAt:        roomRecord.ClosedAt,
				ID:              roomID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
		}

		result = LeaveResult{
			MemberStatus:         "left",
			RoomStatus:           roomStatus,
			HostTransferRequired: transferRequired,
		}
		return nil
	})
	if err != nil {
		return LeaveResult{}, err
	}
	return result, nil
}

func (s *Service) Ready(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input ReadyInput,
	idempotencyKey string,
) (ReadyResult, error) {
	if idempotencyKey == "" {
		return ReadyResult{}, validationError("Idempotency-Key is required")
	}

	var result ReadyResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
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
		if input.ExpectedRoomEventSeq != roomRecord.CurrentEventSeq {
			return versionConflictError()
		}
		member, err := tx.GetActiveRoomMemberByUser(
			ctx,
			sqlc.GetActiveRoomMemberByUserParams{
				RoomID: roomID,
				UserID: sql.NullString{String: actorUserID, Valid: true},
			},
		)
		if err != nil {
			if isStorageCode(err, database.CodeNotFound) {
				return notRoomMemberError()
			}
			return mapRoomStorageError(err)
		}

		status := "joined"
		readyAt := sql.NullTime{}
		if input.Ready {
			status = "ready"
			readyAt = sql.NullTime{Time: s.now().UTC(), Valid: true}
		}
		if member.MemberStatus != status {
			if err := tx.UpdateRoomMemberReady(ctx, sqlc.UpdateRoomMemberReadyParams{
				MemberStatus: status,
				ReadyAt:      readyAt,
				ID:           member.ID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			seq := roomRecord.CurrentEventSeq + 1
			eventType := "room.member_unready"
			if input.Ready {
				eventType = "room.member_ready"
			}
			if err := s.appendEvent(ctx, tx, roomID, seq, eventType, actorUserID, map[string]any{
				"schema_version": "1.0.0",
				"member_id":      member.ID,
			}); err != nil {
				return err
			}
			if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
				Status:          roomRecord.Status,
				CurrentEventSeq: seq,
				StartedAt:       roomRecord.StartedAt,
				ClosedAt:        roomRecord.ClosedAt,
				ID:              roomID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			roomRecord.CurrentEventSeq = seq
			member.MemberStatus = status
			member.ReadyAt = readyAt
			member.RowVersion++
		}

		readyCount, err := tx.CountReadyRoomMembers(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		activeCount, err := tx.CountActiveRoomMembers(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		view, err := s.memberView(ctx, tx, member)
		if err != nil {
			return err
		}
		result = ReadyResult{
			Member:       view,
			ReadyCount:   readyCount,
			CanStart:     activeCount >= int64(roomRecord.CapacityMin) && readyCount == activeCount,
			RoomEventSeq: roomRecord.CurrentEventSeq,
		}
		return nil
	})
	if err != nil {
		return ReadyResult{}, err
	}
	return result, nil
}

func (s *Service) HostAction(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input HostActionInput,
	idempotencyKey string,
) (HostActionResult, error) {
	if idempotencyKey == "" {
		return HostActionResult{}, validationError("Idempotency-Key is required")
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action != "close" && action != "transfer_host" && action != "kick" {
		return HostActionResult{}, validationError("unsupported host action")
	}

	var result HostActionResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		roomRecord, err := tx.GetGameRoomForUpdate(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		if roomRecord.HostUserID != actorUserID {
			return hostRequiredError()
		}
		if action == "close" &&
			(roomRecord.Status == "cancelled" || roomRecord.Status == "closed") {
			result = HostActionResult{
				AcceptedAction: action,
				RoomStatus:     roomRecord.Status,
				RoomEventSeq:   roomRecord.CurrentEventSeq,
			}
			return nil
		}
		if roomRecord.Status != "waiting" {
			return roomNotWaitingError()
		}
		if input.ExpectedRoomEventSeq != roomRecord.CurrentEventSeq {
			return versionConflictError()
		}

		now := s.now().UTC()
		seq := roomRecord.CurrentEventSeq + 1
		roomStatus := roomRecord.Status
		closedAt := roomRecord.ClosedAt

		switch action {
		case "close":
			roomStatus = "cancelled"
			closedAt = sql.NullTime{Time: now, Valid: true}
			if err := s.appendEvent(ctx, tx, roomID, seq, "room.closed", actorUserID, map[string]any{
				"schema_version": "1.0.0",
				"reason":         input.Reason,
			}); err != nil {
				return err
			}
		case "transfer_host":
			target, err := tx.GetRoomMemberForUpdate(ctx, input.TargetMemberID)
			if err != nil || target.RoomID != roomID || target.LeftAt.Valid {
				return targetNotMemberError()
			}
			if target.UserID.String == actorUserID {
				return validationError("target_member_id must identify another member")
			}
			current, err := tx.GetActiveRoomMemberByUser(ctx, sqlc.GetActiveRoomMemberByUserParams{
				RoomID: roomID,
				UserID: sql.NullString{String: actorUserID, Valid: true},
			})
			if err != nil {
				return mapRoomStorageError(err)
			}
			if err := tx.UpdateRoomMemberRole(ctx, sqlc.UpdateRoomMemberRoleParams{
				RoomRole: "host",
				ID:       target.ID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			if err := tx.UpdateRoomMemberRole(ctx, sqlc.UpdateRoomMemberRoleParams{
				RoomRole: "player",
				ID:       current.ID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			if err := tx.UpdateGameRoomHost(ctx, sqlc.UpdateGameRoomHostParams{
				HostUserID:   target.UserID.String,
				ID:           roomID,
				HostUserID_2: actorUserID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			if err := s.appendEvent(ctx, tx, roomID, seq, "room.host_transferred", actorUserID, map[string]any{
				"schema_version":   "1.0.0",
				"new_host_user_id": target.UserID.String,
			}); err != nil {
				return err
			}
		case "kick":
			target, err := tx.GetRoomMemberForUpdate(ctx, input.TargetMemberID)
			if err != nil || target.RoomID != roomID || target.LeftAt.Valid {
				return targetNotMemberError()
			}
			if target.UserID.String == actorUserID {
				return validationError("host cannot kick themselves")
			}
			if err := tx.UpdateRoomMemberLeave(ctx, sqlc.UpdateRoomMemberLeaveParams{
				LeftAt: sql.NullTime{Time: now, Valid: true},
				ID:     target.ID,
			}); err != nil {
				return mapRoomStorageError(err)
			}
			if err := s.appendEvent(ctx, tx, roomID, seq, "room.member_kicked", actorUserID, map[string]any{
				"schema_version": "1.0.0",
				"member_id":      target.ID,
				"reason":         input.Reason,
			}); err != nil {
				return err
			}
		}

		if err := tx.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
			Status:          roomStatus,
			CurrentEventSeq: seq,
			StartedAt:       roomRecord.StartedAt,
			ClosedAt:        closedAt,
			ID:              roomID,
		}); err != nil {
			return mapRoomStorageError(err)
		}

		members, err := s.listMembers(ctx, tx, roomID)
		if err != nil {
			return err
		}
		result = HostActionResult{
			AcceptedAction:       action,
			RoomStatus:           roomStatus,
			Members:              members,
			RoomEventSeq:         seq,
			RequiresConfirmation: false,
		}
		return nil
	})
	if err != nil {
		return HostActionResult{}, err
	}
	return result, nil
}

func (s *Service) ExpireRoom(ctx context.Context, roomID string) error {
	return s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		roomRecord, err := tx.GetGameRoomForUpdate(ctx, roomID)
		if err != nil {
			return mapRoomStorageError(err)
		}
		if roomRecord.Status != "waiting" || !isExpired(roomRecord, s.now().UTC()) {
			return nil
		}
		return s.expireLockedRoom(ctx, tx, roomRecord)
	})
}

func (s *Service) ExpireWaitingRooms(ctx context.Context) (int, error) {
	ids, err := s.repo.ListExpiredWaitingRoomIDs(ctx, sqlc.ListExpiredWaitingRoomIDsParams{
		ExpiresAt: sql.NullTime{Time: s.now().UTC(), Valid: true},
		Limit:     200,
	})
	if err != nil {
		return 0, mapRoomStorageError(err)
	}
	closed := 0
	for _, roomID := range ids {
		if err := s.ExpireRoom(ctx, roomID); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

func (s *Service) ListPublic(
	ctx context.Context,
	gameType string,
	limit int32,
) ([]PublicRoomCard, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.repo.ListPublicRooms(ctx, sqlc.ListPublicRoomsParams{
		ExpiresAt: sql.NullTime{Time: s.now().UTC(), Valid: true},
		Limit:     limit,
	})
	if err != nil {
		return nil, mapRoomStorageError(err)
	}
	cards := make([]PublicRoomCard, 0, len(rows))
	for _, row := range rows {
		if gameType != "" && row.GameType != gameType {
			continue
		}
		cards = append(cards, PublicRoomCard{
			RoomID:         row.ID,
			GameType:       row.GameType,
			ScriptTitle:    row.ScriptTitle,
			HostPublicName: row.HostNickname,
			Players:        row.ActiveMembers,
			Capacity:       int(row.CapacityMax),
			Visibility:     row.Visibility,
			HasPassword:    row.PasswordHash.Valid,
			Status:         row.Status,
		})
	}
	return cards, nil
}
