package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

var (
	ErrNoRowsAffected    = errors.New("storage operation affected no rows")
	ErrNestedTransaction = errors.New("nested storage transactions are not allowed")
)

type Repository interface {
	WithTransaction(ctx context.Context, fn func(context.Context, Repository) error) error

	CreateUser(ctx context.Context, params sqlc.CreateUserParams) error
	GetUser(ctx context.Context, id string) (sqlc.User, error)
	GetAuthIdentityBySubject(
		ctx context.Context,
		params sqlc.GetAuthIdentityBySubjectParams,
	) (sqlc.AuthIdentity, error)
	CreateAuthIdentity(ctx context.Context, params sqlc.CreateAuthIdentityParams) error
	CreateUserProfile(ctx context.Context, params sqlc.CreateUserProfileParams) error
	GetUserProfile(ctx context.Context, userID string) (sqlc.UserProfile, error)
	UpdateUserProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) error
	CreateUserStatistics(ctx context.Context, params sqlc.CreateUserStatisticsParams) error
	GetUserStatistics(ctx context.Context, userID string) (sqlc.UserStatistic, error)
	CreateUserSession(ctx context.Context, params sqlc.CreateUserSessionParams) error
	GetSessionUserByToken(
		ctx context.Context,
		sessionTokenHash []byte,
	) (sqlc.GetSessionUserByTokenRow, error)
	RevokeSession(ctx context.Context, params sqlc.RevokeSessionParams) error
	RevokeAllUserSessions(ctx context.Context, params sqlc.RevokeAllUserSessionsParams) (int64, error)
	UpdateSessionLastSeen(ctx context.Context, params sqlc.UpdateSessionLastSeenParams) error
	UpdateUserLastLogin(ctx context.Context, params sqlc.UpdateUserLastLoginParams) error

	CreateScript(ctx context.Context, params sqlc.CreateScriptParams) error
	GetScript(ctx context.Context, id string) (sqlc.Script, error)
	GetScriptAny(ctx context.Context, id string) (sqlc.Script, error)
	GetScriptForUpdate(ctx context.Context, id string) (sqlc.Script, error)
	ListPublicScripts(
		ctx context.Context,
		params sqlc.ListPublicScriptsParams,
	) ([]sqlc.ListPublicScriptsRow, error)
	GetPublicScriptDetail(ctx context.Context, id string) (sqlc.GetPublicScriptDetailRow, error)
	UpdateScriptMetadata(ctx context.Context, params sqlc.UpdateScriptMetadataParams) error
	SoftDeleteScript(ctx context.Context, params sqlc.SoftDeleteScriptParams) error
	SetScriptPublishedVersion(
		ctx context.Context,
		params sqlc.SetScriptPublishedVersionParams,
	) error
	SetScriptActiveDraftVersion(
		ctx context.Context,
		params sqlc.SetScriptActiveDraftVersionParams,
	) error
	SetScriptUnlisted(ctx context.Context, params sqlc.SetScriptUnlistedParams) error
	GetScriptVersion(ctx context.Context, id string) (sqlc.ScriptVersion, error)
	GetScriptVersionForUpdate(ctx context.Context, id string) (sqlc.ScriptVersion, error)
	ListScriptVersions(ctx context.Context, params sqlc.ListScriptVersionsParams) ([]sqlc.ScriptVersion, error)
	GetNextScriptVersionNo(ctx context.Context, scriptID string) (int32, error)
	CreateScriptVersion(ctx context.Context, params sqlc.CreateScriptVersionParams) error
	UpdateScriptDraftValidation(ctx context.Context, params sqlc.UpdateScriptDraftValidationParams) error
	BumpScriptDraftRowVersion(ctx context.Context, params sqlc.BumpScriptDraftRowVersionParams) error
	PublishScriptVersion(ctx context.Context, params sqlc.PublishScriptVersionParams) error
	MarkScriptVersionUnlisted(ctx context.Context, id string) error
	GetRuleTemplateByID(ctx context.Context, id string) (sqlc.RuleTemplate, error)
	GetActiveRuleTemplateForScript(
		ctx context.Context,
		params sqlc.GetActiveRuleTemplateForScriptParams,
	) (sqlc.RuleTemplate, error)
	CreateScriptPublication(ctx context.Context, params sqlc.CreateScriptPublicationParams) error
	GetScriptPublicationForUpdate(ctx context.Context, id string) (sqlc.ScriptPublication, error)
	GetScriptPublicationByChannel(
		ctx context.Context,
		params sqlc.GetScriptPublicationByChannelParams,
	) (sqlc.ScriptPublication, error)
	SupersedeActiveScriptPublications(
		ctx context.Context,
		params sqlc.SupersedeActiveScriptPublicationsParams,
	) error
	ReactivateScriptPublication(
		ctx context.Context,
		params sqlc.ReactivateScriptPublicationParams,
	) error
	UnpublishScriptPublication(
		ctx context.Context,
		params sqlc.UnpublishScriptPublicationParams,
	) error
	ListScriptChapters(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptChapter, error)
	GetScriptChapter(ctx context.Context, params sqlc.GetScriptChapterParams) (sqlc.ScriptChapter, error)
	CreateScriptChapter(ctx context.Context, params sqlc.CreateScriptChapterParams) error
	UpdateScriptChapter(ctx context.Context, params sqlc.UpdateScriptChapterParams) error
	DeleteScriptChapter(ctx context.Context, params sqlc.DeleteScriptChapterParams) error
	ListScriptStages(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptPhase, error)
	GetScriptStage(ctx context.Context, params sqlc.GetScriptStageParams) (sqlc.ScriptPhase, error)
	CreateScriptStage(ctx context.Context, params sqlc.CreateScriptStageParams) error
	UpdateScriptStage(ctx context.Context, params sqlc.UpdateScriptStageParams) error
	DeleteScriptStage(ctx context.Context, params sqlc.DeleteScriptStageParams) error
	ListScriptCharacters(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptCharacter, error)
	GetScriptCharacter(
		ctx context.Context,
		params sqlc.GetScriptCharacterParams,
	) (sqlc.ScriptCharacter, error)
	CreateScriptCharacter(ctx context.Context, params sqlc.CreateScriptCharacterParams) error
	UpdateScriptCharacter(ctx context.Context, params sqlc.UpdateScriptCharacterParams) error
	DeleteScriptCharacter(ctx context.Context, params sqlc.DeleteScriptCharacterParams) error
	ListScriptClues(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptClue, error)
	GetScriptClue(ctx context.Context, params sqlc.GetScriptClueParams) (sqlc.ScriptClue, error)
	CreateScriptClue(ctx context.Context, params sqlc.CreateScriptClueParams) error
	UpdateScriptClue(ctx context.Context, params sqlc.UpdateScriptClueParams) error
	DeleteScriptClue(ctx context.Context, params sqlc.DeleteScriptClueParams) error
	ListScriptEndings(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptEnding, error)
	GetScriptEnding(ctx context.Context, params sqlc.GetScriptEndingParams) (sqlc.ScriptEnding, error)
	CreateScriptEnding(ctx context.Context, params sqlc.CreateScriptEndingParams) error
	UpdateScriptEnding(ctx context.Context, params sqlc.UpdateScriptEndingParams) error
	DeleteScriptEnding(ctx context.Context, params sqlc.DeleteScriptEndingParams) error
	ListScriptAssets(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptAsset, error)
	GetScriptAsset(ctx context.Context, params sqlc.GetScriptAssetParams) (sqlc.ScriptAsset, error)
	CreateScriptAsset(ctx context.Context, params sqlc.CreateScriptAssetParams) error
	UpdateScriptAsset(ctx context.Context, params sqlc.UpdateScriptAssetParams) error
	DeleteScriptAsset(ctx context.Context, params sqlc.DeleteScriptAssetParams) error
	ListScriptSearchPoints(
		ctx context.Context,
		scriptVersionID string,
	) ([]sqlc.ScriptSearchPoint, error)
	GetScriptSearchPoint(
		ctx context.Context,
		params sqlc.GetScriptSearchPointParams,
	) (sqlc.ScriptSearchPoint, error)
	CreateScriptSearchPoint(ctx context.Context, params sqlc.CreateScriptSearchPointParams) error
	UpdateScriptSearchPoint(ctx context.Context, params sqlc.UpdateScriptSearchPointParams) error
	DeleteScriptSearchPoint(ctx context.Context, params sqlc.DeleteScriptSearchPointParams) error
	ListScriptSearchPointClues(
		ctx context.Context,
		scriptVersionID string,
	) ([]sqlc.ScriptSearchPointClue, error)
	UpsertScriptSearchPointClue(ctx context.Context, params sqlc.UpsertScriptSearchPointClueParams) error
	DeleteScriptSearchPointClue(
		ctx context.Context,
		params sqlc.DeleteScriptSearchPointClueParams,
	) error
	GetScriptVersionForRoom(ctx context.Context, id string) (sqlc.GetScriptVersionForRoomRow, error)
	ListScriptPhasesForGame(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptPhase, error)
	ListScriptCharactersForGame(
		ctx context.Context,
		scriptVersionID string,
	) ([]sqlc.ScriptCharacter, error)
	ListScriptCluesForGame(ctx context.Context, scriptVersionID string) ([]sqlc.ScriptClue, error)
	ListScriptSearchPointsForGame(
		ctx context.Context,
		scriptVersionID string,
	) ([]sqlc.ScriptSearchPoint, error)
	ListSearchPointCluesForGame(
		ctx context.Context,
		scriptVersionID string,
	) ([]sqlc.ListSearchPointCluesForGameRow, error)
	GetRuleTemplateForGame(ctx context.Context, id string) (sqlc.RuleTemplate, error)

	CreateGameRoom(ctx context.Context, params sqlc.CreateGameRoomParams) error
	GetGameRoom(ctx context.Context, id string) (sqlc.GameRoom, error)
	GetGameRoomByCode(ctx context.Context, codeHash []byte) (sqlc.GameRoom, error)
	GetGameRoomForUpdate(ctx context.Context, id string) (sqlc.GameRoom, error)
	UpdateGameRoomMutation(ctx context.Context, params sqlc.UpdateGameRoomMutationParams) error
	UpdateGameRoomHost(ctx context.Context, params sqlc.UpdateGameRoomHostParams) error
	CreateRoomEvent(ctx context.Context, params sqlc.CreateRoomEventParams) error
	ListRoomEventsAfter(
		ctx context.Context,
		params sqlc.ListRoomEventsAfterParams,
	) ([]sqlc.RoomEvent, error)
	CreateRoomMember(ctx context.Context, params sqlc.CreateRoomMemberParams) error
	ListRoomMembers(ctx context.Context, roomID string) ([]sqlc.RoomMember, error)
	CountActiveRoomSeats(ctx context.Context, roomID string) (int64, error)
	GetActiveRoomMemberByUser(
		ctx context.Context,
		params sqlc.GetActiveRoomMemberByUserParams,
	) (sqlc.RoomMember, error)
	GetLatestRoomMemberByUser(
		ctx context.Context,
		params sqlc.GetLatestRoomMemberByUserParams,
	) (sqlc.RoomMember, error)
	GetRoomMemberForUpdate(ctx context.Context, id string) (sqlc.RoomMember, error)
	ListActiveRoomMembers(
		ctx context.Context,
		roomID string,
	) ([]sqlc.ListActiveRoomMembersRow, error)
	UpdateRoomMemberReady(ctx context.Context, params sqlc.UpdateRoomMemberReadyParams) error
	UpdateRoomMemberLeave(ctx context.Context, params sqlc.UpdateRoomMemberLeaveParams) error
	UpdateRoomMemberRole(ctx context.Context, params sqlc.UpdateRoomMemberRoleParams) error
	CountActiveRoomMembers(ctx context.Context, roomID string) (int64, error)
	CountReadyRoomMembers(ctx context.Context, roomID string) (int64, error)
	CreateInvitationToken(ctx context.Context, params sqlc.CreateInvitationTokenParams) error
	GetInvitationForUpdate(
		ctx context.Context,
		tokenHash []byte,
	) (sqlc.InvitationToken, error)
	IncrementInvitationUse(ctx context.Context, id string) error
	ListPublicRooms(
		ctx context.Context,
		params sqlc.ListPublicRoomsParams,
	) ([]sqlc.ListPublicRoomsRow, error)
	CloseExpiredRooms(ctx context.Context, params sqlc.CloseExpiredRoomsParams) (int64, error)
	ListExpiredWaitingRoomIDs(
		ctx context.Context,
		params sqlc.ListExpiredWaitingRoomIDsParams,
	) ([]string, error)
	GetRoomMessageByClientMessageID(
		ctx context.Context,
		params sqlc.GetRoomMessageByClientMessageIDParams,
	) (sqlc.RoomMessage, error)
	GetRoomMessageByIdempotencyKey(
		ctx context.Context,
		params sqlc.GetRoomMessageByIdempotencyKeyParams,
	) (sqlc.RoomMessage, error)
	CreateRoomMessage(ctx context.Context, params sqlc.CreateRoomMessageParams) error
	ListRoomMessagesBefore(
		ctx context.Context,
		params sqlc.ListRoomMessagesBeforeParams,
	) ([]sqlc.RoomMessage, error)
	CountActiveMutes(ctx context.Context, params sqlc.CountActiveMutesParams) (int64, error)
	GetIdempotencyRecord(
		ctx context.Context,
		params sqlc.GetIdempotencyRecordParams,
	) (sqlc.IdempotencyRecord, error)
	CreateIdempotencyRecord(ctx context.Context, params sqlc.CreateIdempotencyRecordParams) error
	CompleteIdempotencyRecord(ctx context.Context, params sqlc.CompleteIdempotencyRecordParams) error

	CreateGameSession(ctx context.Context, params sqlc.CreateGameSessionParams) error
	GetGameSession(ctx context.Context, id string) (sqlc.GameSession, error)
	GetGameSessionForUpdate(ctx context.Context, id string) (sqlc.GameSession, error)
	GetGameSessionByRoom(ctx context.Context, roomID string) (sqlc.GameSession, error)
	UpdateGameSessionVersion(ctx context.Context, params sqlc.UpdateGameSessionVersionParams) error
	UpdateGameSessionCommand(ctx context.Context, params sqlc.UpdateGameSessionCommandParams) error
	CreateGamePlayer(ctx context.Context, params sqlc.CreateGamePlayerParams) error
	ListGamePlayers(ctx context.Context, gameSessionID string) ([]sqlc.GamePlayer, error)
	GetGamePlayerByUser(
		ctx context.Context,
		params sqlc.GetGamePlayerByUserParams,
	) (sqlc.GamePlayer, error)
	GetGamePlayerByID(ctx context.Context, id string) (sqlc.GamePlayer, error)
	UpdateGamePlayerRuntimeStatus(
		ctx context.Context,
		params sqlc.UpdateGamePlayerRuntimeStatusParams,
	) error
	CreateGameEvent(ctx context.Context, params sqlc.CreateGameEventParams) error
	ListGameEventsAfter(ctx context.Context, params sqlc.ListGameEventsAfterParams) ([]sqlc.GameEvent, error)
	CreateGameState(ctx context.Context, params sqlc.CreateGameStateParams) error
	GetGameState(ctx context.Context, gameSessionID string) (sqlc.GameState, error)
	GetGameStateForUpdate(ctx context.Context, gameSessionID string) (sqlc.GameState, error)
	UpdateGameState(ctx context.Context, params sqlc.UpdateGameStateParams) error
	CreateGameRole(ctx context.Context, params sqlc.CreateGameRoleParams) error
	GetGameRoleByPlayer(
		ctx context.Context,
		params sqlc.GetGameRoleByPlayerParams,
	) (sqlc.GameRole, error)
	ListGameRoles(ctx context.Context, gameSessionID string) ([]sqlc.GameRole, error)
	RevealGameRoles(ctx context.Context, params sqlc.RevealGameRolesParams) (int64, error)
	CreateGameIntent(ctx context.Context, params sqlc.CreateGameIntentParams) error
	GetGameIntentByIdempotency(
		ctx context.Context,
		params sqlc.GetGameIntentByIdempotencyParams,
	) (sqlc.GameIntent, error)
	CompleteGameIntent(ctx context.Context, params sqlc.CompleteGameIntentParams) error
	UpsertGameVote(ctx context.Context, params sqlc.UpsertGameVoteParams) (int64, error)
	ListGameVotesForRound(
		ctx context.Context,
		params sqlc.ListGameVotesForRoundParams,
	) ([]sqlc.GameVote, error)
	CreateGameSearchClaim(ctx context.Context, params sqlc.CreateGameSearchClaimParams) error
	CountGameSearchClaimsForPoint(
		ctx context.Context,
		params sqlc.CountGameSearchClaimsForPointParams,
	) (int64, error)
	CountGameSearchClaimsForPlayerPhase(
		ctx context.Context,
		params sqlc.CountGameSearchClaimsForPlayerPhaseParams,
	) (int64, error)
	CreateGameResult(ctx context.Context, params sqlc.CreateGameResultParams) error
	GetGameResultBySession(ctx context.Context, gameSessionID string) (sqlc.GameResult, error)
	CreateGameTimer(ctx context.Context, params sqlc.CreateGameTimerParams) error
	GetLatestGameTimerForUpdate(
		ctx context.Context,
		gameSessionID string,
	) (sqlc.GameTimer, error)
	UpdateGameTimer(ctx context.Context, params sqlc.UpdateGameTimerParams) error
	ListExpiredRunningGameIDs(
		ctx context.Context,
		params sqlc.ListExpiredRunningGameIDsParams,
	) ([]string, error)
	CreateSecretPayload(ctx context.Context, params sqlc.CreateSecretPayloadParams) error
	GetSecretPayload(ctx context.Context, id string) (sqlc.SecretPayload, error)

	CreateNPCMemory(ctx context.Context, params sqlc.CreateNPCMemoryParams) error
	ListNPCMemories(
		ctx context.Context,
		gameSessionID string,
		gameNPCID string,
		now time.Time,
		limit int32,
	) ([]sqlc.NpcMemory, error)

	CreateReport(ctx context.Context, params sqlc.CreateReportParams) error
	GetReport(ctx context.Context, id string) (sqlc.Report, error)
	CreateAuditLog(ctx context.Context, params sqlc.CreateAuditLogParams) error
	ListAuditLogsByObject(
		ctx context.Context,
		objectType string,
		objectID string,
		limit int32,
	) ([]sqlc.AuditLog, error)
}

type MySQLRepository struct {
	db            *database.DB
	queries       *sqlc.Queries
	transactional bool
}

func NewMySQLRepository(db *database.DB) *MySQLRepository {
	return &MySQLRepository{
		db:      db,
		queries: sqlc.New(db.SQL()),
	}
}

func (r *MySQLRepository) WithTransaction(
	ctx context.Context,
	fn func(context.Context, Repository) error,
) error {
	if r.transactional {
		return ErrNestedTransaction
	}
	if fn == nil {
		return errors.New("transaction callback is required")
	}

	return r.db.WithTransaction(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		transactional := &MySQLRepository{
			db:            r.db,
			queries:       sqlc.New(tx),
			transactional: true,
		}
		return fn(ctx, transactional)
	})
}

func (r *MySQLRepository) CreateUser(ctx context.Context, params sqlc.CreateUserParams) error {
	count, err := r.queries.CreateUser(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetUser(ctx context.Context, id string) (sqlc.User, error) {
	user, err := r.queries.GetUser(ctx, id)
	return user, database.MapError(err)
}

func (r *MySQLRepository) GetAuthIdentityBySubject(
	ctx context.Context,
	params sqlc.GetAuthIdentityBySubjectParams,
) (sqlc.AuthIdentity, error) {
	identity, err := r.queries.GetAuthIdentityBySubject(ctx, params)
	return identity, database.MapError(err)
}

func (r *MySQLRepository) CreateAuthIdentity(
	ctx context.Context,
	params sqlc.CreateAuthIdentityParams,
) error {
	count, err := r.queries.CreateAuthIdentity(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateUserProfile(
	ctx context.Context,
	params sqlc.CreateUserProfileParams,
) error {
	count, err := r.queries.CreateUserProfile(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetUserProfile(
	ctx context.Context,
	userID string,
) (sqlc.UserProfile, error) {
	profile, err := r.queries.GetUserProfile(ctx, userID)
	return profile, database.MapError(err)
}

func (r *MySQLRepository) UpdateUserProfile(
	ctx context.Context,
	params sqlc.UpdateUserProfileParams,
) error {
	count, err := r.queries.UpdateUserProfile(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateUserStatistics(
	ctx context.Context,
	params sqlc.CreateUserStatisticsParams,
) error {
	count, err := r.queries.CreateUserStatistics(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetUserStatistics(
	ctx context.Context,
	userID string,
) (sqlc.UserStatistic, error) {
	statistics, err := r.queries.GetUserStatistics(ctx, userID)
	return statistics, database.MapError(err)
}

func (r *MySQLRepository) CreateUserSession(
	ctx context.Context,
	params sqlc.CreateUserSessionParams,
) error {
	count, err := r.queries.CreateUserSession(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetSessionUserByToken(
	ctx context.Context,
	sessionTokenHash []byte,
) (sqlc.GetSessionUserByTokenRow, error) {
	session, err := r.queries.GetSessionUserByToken(ctx, sessionTokenHash)
	return session, database.MapError(err)
}

func (r *MySQLRepository) RevokeSession(
	ctx context.Context,
	params sqlc.RevokeSessionParams,
) error {
	count, err := r.queries.RevokeSession(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) RevokeAllUserSessions(
	ctx context.Context,
	params sqlc.RevokeAllUserSessionsParams,
) (int64, error) {
	count, err := r.queries.RevokeAllUserSessions(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) UpdateSessionLastSeen(
	ctx context.Context,
	params sqlc.UpdateSessionLastSeenParams,
) error {
	return database.MapError(r.queries.UpdateSessionLastSeen(ctx, params))
}

func (r *MySQLRepository) UpdateUserLastLogin(
	ctx context.Context,
	params sqlc.UpdateUserLastLoginParams,
) error {
	return database.MapError(r.queries.UpdateUserLastLogin(ctx, params))
}

func (r *MySQLRepository) CreateScript(ctx context.Context, params sqlc.CreateScriptParams) error {
	count, err := r.queries.CreateScript(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetScript(ctx context.Context, id string) (sqlc.Script, error) {
	script, err := r.queries.GetScript(ctx, id)
	return script, database.MapError(err)
}

func (r *MySQLRepository) GetScriptAny(ctx context.Context, id string) (sqlc.Script, error) {
	script, err := r.queries.GetScriptAny(ctx, id)
	return script, database.MapError(err)
}

func (r *MySQLRepository) GetScriptForUpdate(
	ctx context.Context,
	id string,
) (sqlc.Script, error) {
	script, err := r.queries.GetScriptForUpdate(ctx, id)
	return script, database.MapError(err)
}

func (r *MySQLRepository) ListPublicScripts(
	ctx context.Context,
	params sqlc.ListPublicScriptsParams,
) ([]sqlc.ListPublicScriptsRow, error) {
	rows, err := r.queries.ListPublicScripts(ctx, params)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetPublicScriptDetail(
	ctx context.Context,
	id string,
) (sqlc.GetPublicScriptDetailRow, error) {
	row, err := r.queries.GetPublicScriptDetail(ctx, id)
	return row, database.MapError(err)
}

func (r *MySQLRepository) UpdateScriptMetadata(
	ctx context.Context,
	params sqlc.UpdateScriptMetadataParams,
) error {
	count, err := r.queries.UpdateScriptMetadata(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) SoftDeleteScript(
	ctx context.Context,
	params sqlc.SoftDeleteScriptParams,
) error {
	count, err := r.queries.SoftDeleteScript(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) SetScriptPublishedVersion(
	ctx context.Context,
	params sqlc.SetScriptPublishedVersionParams,
) error {
	count, err := r.queries.SetScriptPublishedVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) SetScriptActiveDraftVersion(
	ctx context.Context,
	params sqlc.SetScriptActiveDraftVersionParams,
) error {
	count, err := r.queries.SetScriptActiveDraftVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) SetScriptUnlisted(
	ctx context.Context,
	params sqlc.SetScriptUnlistedParams,
) error {
	count, err := r.queries.SetScriptUnlisted(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetScriptVersion(
	ctx context.Context,
	id string,
) (sqlc.ScriptVersion, error) {
	version, err := r.queries.GetScriptVersion(ctx, id)
	return version, database.MapError(err)
}

func (r *MySQLRepository) GetScriptVersionForUpdate(
	ctx context.Context,
	id string,
) (sqlc.ScriptVersion, error) {
	version, err := r.queries.GetScriptVersionForUpdate(ctx, id)
	return version, database.MapError(err)
}

func (r *MySQLRepository) ListScriptVersions(
	ctx context.Context,
	params sqlc.ListScriptVersionsParams,
) ([]sqlc.ScriptVersion, error) {
	versions, err := r.queries.ListScriptVersions(ctx, params)
	return versions, database.MapError(err)
}

func (r *MySQLRepository) GetNextScriptVersionNo(
	ctx context.Context,
	scriptID string,
) (int32, error) {
	value, err := r.queries.GetNextScriptVersionNo(ctx, scriptID)
	return value, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptVersion(
	ctx context.Context,
	params sqlc.CreateScriptVersionParams,
) error {
	count, err := r.queries.CreateScriptVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptDraftValidation(
	ctx context.Context,
	params sqlc.UpdateScriptDraftValidationParams,
) error {
	count, err := r.queries.UpdateScriptDraftValidation(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) BumpScriptDraftRowVersion(
	ctx context.Context,
	params sqlc.BumpScriptDraftRowVersionParams,
) error {
	count, err := r.queries.BumpScriptDraftRowVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) PublishScriptVersion(
	ctx context.Context,
	params sqlc.PublishScriptVersionParams,
) error {
	count, err := r.queries.PublishScriptVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) MarkScriptVersionUnlisted(ctx context.Context, id string) error {
	count, err := r.queries.MarkScriptVersionUnlisted(ctx, id)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetRuleTemplateByID(
	ctx context.Context,
	id string,
) (sqlc.RuleTemplate, error) {
	template, err := r.queries.GetRuleTemplateByID(ctx, id)
	return template, database.MapError(err)
}

func (r *MySQLRepository) GetActiveRuleTemplateForScript(
	ctx context.Context,
	params sqlc.GetActiveRuleTemplateForScriptParams,
) (sqlc.RuleTemplate, error) {
	template, err := r.queries.GetActiveRuleTemplateForScript(ctx, params)
	return template, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptPublication(
	ctx context.Context,
	params sqlc.CreateScriptPublicationParams,
) error {
	count, err := r.queries.CreateScriptPublication(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetScriptPublicationForUpdate(
	ctx context.Context,
	id string,
) (sqlc.ScriptPublication, error) {
	publication, err := r.queries.GetScriptPublicationForUpdate(ctx, id)
	return publication, database.MapError(err)
}

func (r *MySQLRepository) GetScriptPublicationByChannel(
	ctx context.Context,
	params sqlc.GetScriptPublicationByChannelParams,
) (sqlc.ScriptPublication, error) {
	publication, err := r.queries.GetScriptPublicationByChannel(ctx, params)
	return publication, database.MapError(err)
}

func (r *MySQLRepository) SupersedeActiveScriptPublications(
	ctx context.Context,
	params sqlc.SupersedeActiveScriptPublicationsParams,
) error {
	_, err := r.queries.SupersedeActiveScriptPublications(ctx, params)
	return database.MapError(err)
}

func (r *MySQLRepository) ReactivateScriptPublication(
	ctx context.Context,
	params sqlc.ReactivateScriptPublicationParams,
) error {
	count, err := r.queries.ReactivateScriptPublication(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UnpublishScriptPublication(
	ctx context.Context,
	params sqlc.UnpublishScriptPublicationParams,
) error {
	count, err := r.queries.UnpublishScriptPublication(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptChapters(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptChapter, error) {
	rows, err := r.queries.ListScriptChapters(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptChapter(
	ctx context.Context,
	params sqlc.GetScriptChapterParams,
) (sqlc.ScriptChapter, error) {
	row, err := r.queries.GetScriptChapter(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptChapter(
	ctx context.Context,
	params sqlc.CreateScriptChapterParams,
) error {
	count, err := r.queries.CreateScriptChapter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptChapter(
	ctx context.Context,
	params sqlc.UpdateScriptChapterParams,
) error {
	count, err := r.queries.UpdateScriptChapter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptChapter(
	ctx context.Context,
	params sqlc.DeleteScriptChapterParams,
) error {
	count, err := r.queries.DeleteScriptChapter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptStages(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptPhase, error) {
	rows, err := r.queries.ListScriptStages(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptStage(
	ctx context.Context,
	params sqlc.GetScriptStageParams,
) (sqlc.ScriptPhase, error) {
	row, err := r.queries.GetScriptStage(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptStage(
	ctx context.Context,
	params sqlc.CreateScriptStageParams,
) error {
	count, err := r.queries.CreateScriptStage(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptStage(
	ctx context.Context,
	params sqlc.UpdateScriptStageParams,
) error {
	count, err := r.queries.UpdateScriptStage(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptStage(
	ctx context.Context,
	params sqlc.DeleteScriptStageParams,
) error {
	count, err := r.queries.DeleteScriptStage(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptCharacters(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptCharacter, error) {
	rows, err := r.queries.ListScriptCharacters(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptCharacter(
	ctx context.Context,
	params sqlc.GetScriptCharacterParams,
) (sqlc.ScriptCharacter, error) {
	row, err := r.queries.GetScriptCharacter(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptCharacter(
	ctx context.Context,
	params sqlc.CreateScriptCharacterParams,
) error {
	count, err := r.queries.CreateScriptCharacter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptCharacter(
	ctx context.Context,
	params sqlc.UpdateScriptCharacterParams,
) error {
	count, err := r.queries.UpdateScriptCharacter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptCharacter(
	ctx context.Context,
	params sqlc.DeleteScriptCharacterParams,
) error {
	count, err := r.queries.DeleteScriptCharacter(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptClues(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptClue, error) {
	rows, err := r.queries.ListScriptClues(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptClue(
	ctx context.Context,
	params sqlc.GetScriptClueParams,
) (sqlc.ScriptClue, error) {
	row, err := r.queries.GetScriptClue(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptClue(
	ctx context.Context,
	params sqlc.CreateScriptClueParams,
) error {
	count, err := r.queries.CreateScriptClue(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptClue(
	ctx context.Context,
	params sqlc.UpdateScriptClueParams,
) error {
	count, err := r.queries.UpdateScriptClue(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptClue(
	ctx context.Context,
	params sqlc.DeleteScriptClueParams,
) error {
	count, err := r.queries.DeleteScriptClue(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptEndings(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptEnding, error) {
	rows, err := r.queries.ListScriptEndings(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptEnding(
	ctx context.Context,
	params sqlc.GetScriptEndingParams,
) (sqlc.ScriptEnding, error) {
	row, err := r.queries.GetScriptEnding(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptEnding(
	ctx context.Context,
	params sqlc.CreateScriptEndingParams,
) error {
	count, err := r.queries.CreateScriptEnding(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptEnding(
	ctx context.Context,
	params sqlc.UpdateScriptEndingParams,
) error {
	count, err := r.queries.UpdateScriptEnding(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptEnding(
	ctx context.Context,
	params sqlc.DeleteScriptEndingParams,
) error {
	count, err := r.queries.DeleteScriptEnding(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptAssets(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptAsset, error) {
	rows, err := r.queries.ListScriptAssets(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptAsset(
	ctx context.Context,
	params sqlc.GetScriptAssetParams,
) (sqlc.ScriptAsset, error) {
	row, err := r.queries.GetScriptAsset(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptAsset(
	ctx context.Context,
	params sqlc.CreateScriptAssetParams,
) error {
	count, err := r.queries.CreateScriptAsset(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptAsset(
	ctx context.Context,
	params sqlc.UpdateScriptAssetParams,
) error {
	count, err := r.queries.UpdateScriptAsset(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptAsset(
	ctx context.Context,
	params sqlc.DeleteScriptAssetParams,
) error {
	count, err := r.queries.DeleteScriptAsset(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptSearchPoints(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptSearchPoint, error) {
	rows, err := r.queries.ListScriptSearchPoints(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetScriptSearchPoint(
	ctx context.Context,
	params sqlc.GetScriptSearchPointParams,
) (sqlc.ScriptSearchPoint, error) {
	row, err := r.queries.GetScriptSearchPoint(ctx, params)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateScriptSearchPoint(
	ctx context.Context,
	params sqlc.CreateScriptSearchPointParams,
) error {
	count, err := r.queries.CreateScriptSearchPoint(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateScriptSearchPoint(
	ctx context.Context,
	params sqlc.UpdateScriptSearchPointParams,
) error {
	count, err := r.queries.UpdateScriptSearchPoint(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) DeleteScriptSearchPoint(
	ctx context.Context,
	params sqlc.DeleteScriptSearchPointParams,
) error {
	count, err := r.queries.DeleteScriptSearchPoint(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListScriptSearchPointClues(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptSearchPointClue, error) {
	rows, err := r.queries.ListScriptSearchPointClues(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) UpsertScriptSearchPointClue(
	ctx context.Context,
	params sqlc.UpsertScriptSearchPointClueParams,
) error {
	return database.MapError(r.queries.UpsertScriptSearchPointClue(ctx, params))
}

func (r *MySQLRepository) DeleteScriptSearchPointClue(
	ctx context.Context,
	params sqlc.DeleteScriptSearchPointClueParams,
) error {
	count, err := r.queries.DeleteScriptSearchPointClue(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetScriptVersionForRoom(
	ctx context.Context,
	id string,
) (sqlc.GetScriptVersionForRoomRow, error) {
	version, err := r.queries.GetScriptVersionForRoom(ctx, id)
	return version, database.MapError(err)
}

func (r *MySQLRepository) ListScriptPhasesForGame(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptPhase, error) {
	rows, err := r.queries.ListScriptPhasesForGame(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) ListScriptCharactersForGame(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptCharacter, error) {
	rows, err := r.queries.ListScriptCharactersForGame(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) ListScriptCluesForGame(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptClue, error) {
	rows, err := r.queries.ListScriptCluesForGame(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) ListScriptSearchPointsForGame(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ScriptSearchPoint, error) {
	rows, err := r.queries.ListScriptSearchPointsForGame(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) ListSearchPointCluesForGame(
	ctx context.Context,
	scriptVersionID string,
) ([]sqlc.ListSearchPointCluesForGameRow, error) {
	rows, err := r.queries.ListSearchPointCluesForGame(ctx, scriptVersionID)
	return rows, database.MapError(err)
}

func (r *MySQLRepository) GetRuleTemplateForGame(
	ctx context.Context,
	id string,
) (sqlc.RuleTemplate, error) {
	row, err := r.queries.GetRuleTemplateForGame(ctx, id)
	return row, database.MapError(err)
}

func (r *MySQLRepository) CreateGameRoom(ctx context.Context, params sqlc.CreateGameRoomParams) error {
	count, err := r.queries.CreateGameRoom(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameRoom(ctx context.Context, id string) (sqlc.GameRoom, error) {
	room, err := r.queries.GetGameRoom(ctx, id)
	return room, database.MapError(err)
}

func (r *MySQLRepository) GetGameRoomByCode(
	ctx context.Context,
	codeHash []byte,
) (sqlc.GameRoom, error) {
	room, err := r.queries.GetGameRoomByCode(ctx, codeHash)
	return room, database.MapError(err)
}

func (r *MySQLRepository) GetGameRoomForUpdate(
	ctx context.Context,
	id string,
) (sqlc.GameRoom, error) {
	room, err := r.queries.GetGameRoomForUpdate(ctx, id)
	return room, database.MapError(err)
}

func (r *MySQLRepository) UpdateGameRoomMutation(
	ctx context.Context,
	params sqlc.UpdateGameRoomMutationParams,
) error {
	count, err := r.queries.UpdateGameRoomMutation(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateGameRoomHost(
	ctx context.Context,
	params sqlc.UpdateGameRoomHostParams,
) error {
	count, err := r.queries.UpdateGameRoomHost(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateRoomEvent(
	ctx context.Context,
	params sqlc.CreateRoomEventParams,
) error {
	count, err := r.queries.CreateRoomEvent(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListRoomEventsAfter(
	ctx context.Context,
	params sqlc.ListRoomEventsAfterParams,
) ([]sqlc.RoomEvent, error) {
	events, err := r.queries.ListRoomEventsAfter(ctx, params)
	return events, database.MapError(err)
}

func (r *MySQLRepository) CreateRoomMember(
	ctx context.Context,
	params sqlc.CreateRoomMemberParams,
) error {
	count, err := r.queries.CreateRoomMember(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListRoomMembers(
	ctx context.Context,
	roomID string,
) ([]sqlc.RoomMember, error) {
	members, err := r.queries.ListRoomMembers(ctx, roomID)
	return members, database.MapError(err)
}

func (r *MySQLRepository) CountActiveRoomSeats(
	ctx context.Context,
	roomID string,
) (int64, error) {
	count, err := r.queries.CountActiveRoomSeats(ctx, roomID)
	return count, database.MapError(err)
}

func (r *MySQLRepository) GetActiveRoomMemberByUser(
	ctx context.Context,
	params sqlc.GetActiveRoomMemberByUserParams,
) (sqlc.RoomMember, error) {
	member, err := r.queries.GetActiveRoomMemberByUser(ctx, params)
	return member, database.MapError(err)
}

func (r *MySQLRepository) GetLatestRoomMemberByUser(
	ctx context.Context,
	params sqlc.GetLatestRoomMemberByUserParams,
) (sqlc.RoomMember, error) {
	member, err := r.queries.GetLatestRoomMemberByUser(ctx, params)
	return member, database.MapError(err)
}

func (r *MySQLRepository) GetRoomMemberForUpdate(
	ctx context.Context,
	id string,
) (sqlc.RoomMember, error) {
	member, err := r.queries.GetRoomMemberForUpdate(ctx, id)
	return member, database.MapError(err)
}

func (r *MySQLRepository) ListActiveRoomMembers(
	ctx context.Context,
	roomID string,
) ([]sqlc.ListActiveRoomMembersRow, error) {
	members, err := r.queries.ListActiveRoomMembers(ctx, roomID)
	return members, database.MapError(err)
}

func (r *MySQLRepository) UpdateRoomMemberReady(
	ctx context.Context,
	params sqlc.UpdateRoomMemberReadyParams,
) error {
	count, err := r.queries.UpdateRoomMemberReady(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateRoomMemberLeave(
	ctx context.Context,
	params sqlc.UpdateRoomMemberLeaveParams,
) error {
	count, err := r.queries.UpdateRoomMemberLeave(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateRoomMemberRole(
	ctx context.Context,
	params sqlc.UpdateRoomMemberRoleParams,
) error {
	count, err := r.queries.UpdateRoomMemberRole(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CountActiveRoomMembers(
	ctx context.Context,
	roomID string,
) (int64, error) {
	count, err := r.queries.CountActiveRoomMembers(ctx, roomID)
	return count, database.MapError(err)
}

func (r *MySQLRepository) CountReadyRoomMembers(
	ctx context.Context,
	roomID string,
) (int64, error) {
	count, err := r.queries.CountReadyRoomMembers(ctx, roomID)
	return count, database.MapError(err)
}

func (r *MySQLRepository) CreateInvitationToken(
	ctx context.Context,
	params sqlc.CreateInvitationTokenParams,
) error {
	count, err := r.queries.CreateInvitationToken(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetInvitationForUpdate(
	ctx context.Context,
	tokenHash []byte,
) (sqlc.InvitationToken, error) {
	invitation, err := r.queries.GetInvitationForUpdate(ctx, tokenHash)
	return invitation, database.MapError(err)
}

func (r *MySQLRepository) IncrementInvitationUse(
	ctx context.Context,
	id string,
) error {
	count, err := r.queries.IncrementInvitationUse(ctx, id)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListPublicRooms(
	ctx context.Context,
	params sqlc.ListPublicRoomsParams,
) ([]sqlc.ListPublicRoomsRow, error) {
	rooms, err := r.queries.ListPublicRooms(ctx, params)
	return rooms, database.MapError(err)
}

func (r *MySQLRepository) CloseExpiredRooms(
	ctx context.Context,
	params sqlc.CloseExpiredRoomsParams,
) (int64, error) {
	count, err := r.queries.CloseExpiredRooms(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) ListExpiredWaitingRoomIDs(
	ctx context.Context,
	params sqlc.ListExpiredWaitingRoomIDsParams,
) ([]string, error) {
	ids, err := r.queries.ListExpiredWaitingRoomIDs(ctx, params)
	return ids, database.MapError(err)
}

func (r *MySQLRepository) GetRoomMessageByClientMessageID(
	ctx context.Context,
	params sqlc.GetRoomMessageByClientMessageIDParams,
) (sqlc.RoomMessage, error) {
	message, err := r.queries.GetRoomMessageByClientMessageID(ctx, params)
	return message, database.MapError(err)
}

func (r *MySQLRepository) GetRoomMessageByIdempotencyKey(
	ctx context.Context,
	params sqlc.GetRoomMessageByIdempotencyKeyParams,
) (sqlc.RoomMessage, error) {
	message, err := r.queries.GetRoomMessageByIdempotencyKey(ctx, params)
	return message, database.MapError(err)
}

func (r *MySQLRepository) CreateRoomMessage(
	ctx context.Context,
	params sqlc.CreateRoomMessageParams,
) error {
	count, err := r.queries.CreateRoomMessage(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListRoomMessagesBefore(
	ctx context.Context,
	params sqlc.ListRoomMessagesBeforeParams,
) ([]sqlc.RoomMessage, error) {
	messages, err := r.queries.ListRoomMessagesBefore(ctx, params)
	return messages, database.MapError(err)
}

func (r *MySQLRepository) CountActiveMutes(
	ctx context.Context,
	params sqlc.CountActiveMutesParams,
) (int64, error) {
	count, err := r.queries.CountActiveMutes(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) GetIdempotencyRecord(
	ctx context.Context,
	params sqlc.GetIdempotencyRecordParams,
) (sqlc.IdempotencyRecord, error) {
	record, err := r.queries.GetIdempotencyRecord(ctx, params)
	return record, database.MapError(err)
}

func (r *MySQLRepository) CreateIdempotencyRecord(
	ctx context.Context,
	params sqlc.CreateIdempotencyRecordParams,
) error {
	count, err := r.queries.CreateIdempotencyRecord(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CompleteIdempotencyRecord(
	ctx context.Context,
	params sqlc.CompleteIdempotencyRecordParams,
) error {
	count, err := r.queries.CompleteIdempotencyRecord(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateGameSession(
	ctx context.Context,
	params sqlc.CreateGameSessionParams,
) error {
	count, err := r.queries.CreateGameSession(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameSession(
	ctx context.Context,
	id string,
) (sqlc.GameSession, error) {
	session, err := r.queries.GetGameSession(ctx, id)
	return session, database.MapError(err)
}

func (r *MySQLRepository) GetGameSessionForUpdate(
	ctx context.Context,
	id string,
) (sqlc.GameSession, error) {
	session, err := r.queries.GetGameSessionForUpdate(ctx, id)
	return session, database.MapError(err)
}

func (r *MySQLRepository) GetGameSessionByRoom(
	ctx context.Context,
	roomID string,
) (sqlc.GameSession, error) {
	session, err := r.queries.GetGameSessionByRoom(ctx, roomID)
	return session, database.MapError(err)
}

func (r *MySQLRepository) UpdateGameSessionVersion(
	ctx context.Context,
	params sqlc.UpdateGameSessionVersionParams,
) error {
	count, err := r.queries.UpdateGameSessionVersion(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpdateGameSessionCommand(
	ctx context.Context,
	params sqlc.UpdateGameSessionCommandParams,
) error {
	count, err := r.queries.UpdateGameSessionCommand(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateGamePlayer(
	ctx context.Context,
	params sqlc.CreateGamePlayerParams,
) error {
	count, err := r.queries.CreateGamePlayer(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListGamePlayers(
	ctx context.Context,
	gameSessionID string,
) ([]sqlc.GamePlayer, error) {
	players, err := r.queries.ListGamePlayers(ctx, gameSessionID)
	return players, database.MapError(err)
}

func (r *MySQLRepository) GetGamePlayerByUser(
	ctx context.Context,
	params sqlc.GetGamePlayerByUserParams,
) (sqlc.GamePlayer, error) {
	player, err := r.queries.GetGamePlayerByUser(ctx, params)
	return player, database.MapError(err)
}

func (r *MySQLRepository) GetGamePlayerByID(
	ctx context.Context,
	id string,
) (sqlc.GamePlayer, error) {
	player, err := r.queries.GetGamePlayerByID(ctx, id)
	return player, database.MapError(err)
}

func (r *MySQLRepository) UpdateGamePlayerRuntimeStatus(
	ctx context.Context,
	params sqlc.UpdateGamePlayerRuntimeStatusParams,
) error {
	count, err := r.queries.UpdateGamePlayerRuntimeStatus(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateGameEvent(
	ctx context.Context,
	params sqlc.CreateGameEventParams,
) error {
	count, err := r.queries.CreateGameEvent(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListGameEventsAfter(
	ctx context.Context,
	params sqlc.ListGameEventsAfterParams,
) ([]sqlc.GameEvent, error) {
	events, err := r.queries.ListGameEventsAfter(ctx, params)
	return events, database.MapError(err)
}

func (r *MySQLRepository) CreateGameState(
	ctx context.Context,
	params sqlc.CreateGameStateParams,
) error {
	count, err := r.queries.CreateGameState(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameState(
	ctx context.Context,
	gameSessionID string,
) (sqlc.GameState, error) {
	state, err := r.queries.GetGameState(ctx, gameSessionID)
	return state, database.MapError(err)
}

func (r *MySQLRepository) GetGameStateForUpdate(
	ctx context.Context,
	gameSessionID string,
) (sqlc.GameState, error) {
	state, err := r.queries.GetGameStateForUpdate(ctx, gameSessionID)
	return state, database.MapError(err)
}

func (r *MySQLRepository) UpdateGameState(
	ctx context.Context,
	params sqlc.UpdateGameStateParams,
) error {
	count, err := r.queries.UpdateGameState(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CreateGameRole(
	ctx context.Context,
	params sqlc.CreateGameRoleParams,
) error {
	count, err := r.queries.CreateGameRole(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameRoleByPlayer(
	ctx context.Context,
	params sqlc.GetGameRoleByPlayerParams,
) (sqlc.GameRole, error) {
	role, err := r.queries.GetGameRoleByPlayer(ctx, params)
	return role, database.MapError(err)
}

func (r *MySQLRepository) ListGameRoles(
	ctx context.Context,
	gameSessionID string,
) ([]sqlc.GameRole, error) {
	roles, err := r.queries.ListGameRoles(ctx, gameSessionID)
	return roles, database.MapError(err)
}

func (r *MySQLRepository) RevealGameRoles(
	ctx context.Context,
	params sqlc.RevealGameRolesParams,
) (int64, error) {
	count, err := r.queries.RevealGameRoles(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) CreateGameIntent(
	ctx context.Context,
	params sqlc.CreateGameIntentParams,
) error {
	count, err := r.queries.CreateGameIntent(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameIntentByIdempotency(
	ctx context.Context,
	params sqlc.GetGameIntentByIdempotencyParams,
) (sqlc.GameIntent, error) {
	intent, err := r.queries.GetGameIntentByIdempotency(ctx, params)
	return intent, database.MapError(err)
}

func (r *MySQLRepository) CompleteGameIntent(
	ctx context.Context,
	params sqlc.CompleteGameIntentParams,
) error {
	count, err := r.queries.CompleteGameIntent(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) UpsertGameVote(
	ctx context.Context,
	params sqlc.UpsertGameVoteParams,
) (int64, error) {
	count, err := r.queries.UpsertGameVote(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) ListGameVotesForRound(
	ctx context.Context,
	params sqlc.ListGameVotesForRoundParams,
) ([]sqlc.GameVote, error) {
	votes, err := r.queries.ListGameVotesForRound(ctx, params)
	return votes, database.MapError(err)
}

func (r *MySQLRepository) CreateGameSearchClaim(
	ctx context.Context,
	params sqlc.CreateGameSearchClaimParams,
) error {
	count, err := r.queries.CreateGameSearchClaim(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) CountGameSearchClaimsForPoint(
	ctx context.Context,
	params sqlc.CountGameSearchClaimsForPointParams,
) (int64, error) {
	count, err := r.queries.CountGameSearchClaimsForPoint(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) CountGameSearchClaimsForPlayerPhase(
	ctx context.Context,
	params sqlc.CountGameSearchClaimsForPlayerPhaseParams,
) (int64, error) {
	count, err := r.queries.CountGameSearchClaimsForPlayerPhase(ctx, params)
	return count, database.MapError(err)
}

func (r *MySQLRepository) CreateGameResult(
	ctx context.Context,
	params sqlc.CreateGameResultParams,
) error {
	count, err := r.queries.CreateGameResult(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetGameResultBySession(
	ctx context.Context,
	gameSessionID string,
) (sqlc.GameResult, error) {
	result, err := r.queries.GetGameResultBySession(ctx, gameSessionID)
	return result, database.MapError(err)
}

func (r *MySQLRepository) CreateGameTimer(
	ctx context.Context,
	params sqlc.CreateGameTimerParams,
) error {
	count, err := r.queries.CreateGameTimer(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetLatestGameTimerForUpdate(
	ctx context.Context,
	gameSessionID string,
) (sqlc.GameTimer, error) {
	timer, err := r.queries.GetLatestGameTimerForUpdate(ctx, gameSessionID)
	return timer, database.MapError(err)
}

func (r *MySQLRepository) UpdateGameTimer(
	ctx context.Context,
	params sqlc.UpdateGameTimerParams,
) error {
	count, err := r.queries.UpdateGameTimer(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListExpiredRunningGameIDs(
	ctx context.Context,
	params sqlc.ListExpiredRunningGameIDsParams,
) ([]string, error) {
	ids, err := r.queries.ListExpiredRunningGameIDs(ctx, params)
	return ids, database.MapError(err)
}

func (r *MySQLRepository) CreateSecretPayload(
	ctx context.Context,
	params sqlc.CreateSecretPayloadParams,
) error {
	count, err := r.queries.CreateSecretPayload(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetSecretPayload(
	ctx context.Context,
	id string,
) (sqlc.SecretPayload, error) {
	payload, err := r.queries.GetSecretPayload(ctx, id)
	return payload, database.MapError(err)
}

func (r *MySQLRepository) CreateNPCMemory(
	ctx context.Context,
	params sqlc.CreateNPCMemoryParams,
) error {
	count, err := r.queries.CreateNPCMemory(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListNPCMemories(
	ctx context.Context,
	gameSessionID string,
	gameNPCID string,
	now time.Time,
	limit int32,
) ([]sqlc.NpcMemory, error) {
	memories, err := r.queries.ListNPCMemories(ctx, sqlc.ListNPCMemoriesParams{
		GameSessionID: gameSessionID,
		GameNpcID:     gameNPCID,
		ExpiresAt:     sql.NullTime{Time: now.UTC(), Valid: true},
		Limit:         limit,
	})
	return memories, database.MapError(err)
}

func (r *MySQLRepository) CreateReport(
	ctx context.Context,
	params sqlc.CreateReportParams,
) error {
	count, err := r.queries.CreateReport(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) GetReport(
	ctx context.Context,
	id string,
) (sqlc.Report, error) {
	report, err := r.queries.GetReport(ctx, id)
	return report, database.MapError(err)
}

func (r *MySQLRepository) CreateAuditLog(
	ctx context.Context,
	params sqlc.CreateAuditLogParams,
) error {
	count, err := r.queries.CreateAuditLog(ctx, params)
	return requireOne(count, err)
}

func (r *MySQLRepository) ListAuditLogsByObject(
	ctx context.Context,
	objectType string,
	objectID string,
	limit int32,
) ([]sqlc.AuditLog, error) {
	logs, err := r.queries.ListAuditLogsByObject(ctx, sqlc.ListAuditLogsByObjectParams{
		ObjectType: objectType,
		ObjectID:   objectID,
		Limit:      limit,
	})
	return logs, database.MapError(err)
}

func requireOne(count int64, err error) error {
	if err != nil {
		return database.MapError(err)
	}
	if count != 1 {
		return ErrNoRowsAffected
	}
	return nil
}
