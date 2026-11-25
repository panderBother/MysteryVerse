package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

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
	tokens    tokenGenerator
	passwords *security.PasswordHasher
	limiter   ratelimit.Limiter
	cfg       config.AuthConfig
	now       func() time.Time
	dummyHash string
	hmacKey   []byte
}

type tokenGenerator interface {
	New() (string, error)
}

type cryptoTokenGenerator struct{}

func (cryptoTokenGenerator) New() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func NewService(
	repo storage.Repository,
	cfg config.AuthConfig,
	limiter ratelimit.Limiter,
) (*Service, error) {
	hasher := security.NewPasswordHasher(security.Argon2Params{})
	dummyHash, err := hasher.Hash("invalid-password-timing-placeholder")
	if err != nil {
		return nil, fmt.Errorf("prepare password verifier: %w", err)
	}

	return &Service{
		repo:      repo,
		ids:       id.ULIDGenerator{},
		tokens:    cryptoTokenGenerator{},
		passwords: hasher,
		limiter:   limiter,
		cfg:       cfg,
		now:       time.Now,
		dummyHash: dummyHash,
		hmacKey:   []byte(cfg.IdentityHMACKey),
	}, nil
}

func (s *Service) Register(
	ctx context.Context,
	input RegisterInput,
	meta RequestMetadata,
) (SessionResult, error) {
	normalized, err := normalizeRegisterInput(input)
	if err != nil {
		return SessionResult{}, err
	}
	if meta.IdempotencyKey == "" {
		return SessionResult{}, apperror.New(
			"VALIDATION_ERROR",
			"Idempotency-Key is required.",
			false,
			nil,
		)
	}

	allowed, err := s.allowRateLimit(
		ctx,
		"register:"+meta.IPAddress+":"+normalized.Identifier,
		s.cfg.RegisterRateLimit,
		s.cfg.RegisterRateWindow,
	)
	if err != nil {
		return SessionResult{}, err
	}
	if !allowed {
		return SessionResult{}, rateLimitError(time.Minute)
	}

	passwordHash, err := s.passwords.Hash(normalized.Password)
	if err != nil {
		return SessionResult{}, apperror.New(
			"INTERNAL_ERROR",
			"Password processing failed.",
			true,
			nil,
		)
	}

	requestHash := hashJSON(normalized)
	actorScope := "ip:" + hashText(meta.IPAddress)
	var result SessionResult

	err = s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, exists, err := s.getOrCreateIdempotency(
			ctx,
			tx,
			"identity.register",
			actorScope,
			meta.IdempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		if exists && record.ResourceID.Valid {
			user, err := tx.GetUser(ctx, record.ResourceID.String)
			if err != nil {
				return mapIdentityStorageError(err)
			}
			profile, err := tx.GetUserProfile(ctx, user.ID)
			if err != nil {
				return mapIdentityStorageError(err)
			}
			session, err := s.issueSession(ctx, tx, user.ID, meta, false)
			if err != nil {
				return err
			}
			result = SessionResult{
				User:             userView(user),
				Profile:          profileView(profile),
				Privacy:          privacyFromJSON(profile.PrivacyJson),
				Session:          session,
				IdempotentReplay: true,
			}
			return nil
		}

		userID := s.ids.New()
		user := sqlc.CreateUserParams{
			ID:             userID,
			PublicNo:       "MV" + userID[len(userID)-10:],
			Status:         "active",
			RiskLevel:      0,
			TermsVersion:   normalized.TermsVersion,
			PrivacyVersion: normalized.PrivacyVersion,
		}
		if err := tx.CreateUser(ctx, user); err != nil {
			return mapIdentityStorageError(err)
		}

		identityID := s.ids.New()
		if err := tx.CreateAuthIdentity(ctx, sqlc.CreateAuthIdentityParams{
			ID:              identityID,
			UserID:          userID,
			Provider:        normalized.Provider,
			ProviderSubject: normalized.Identifier,
			IdentifierHash:  s.identifierHash(normalized.Identifier),
			IdentifierHint:  identifierHint(normalized.Identifier),
			PasswordHash:    sql.NullString{String: passwordHash, Valid: true},
			VerifiedAt:      sql.NullTime{},
			IsPrimary:       true,
			Status:          "active",
		}); err != nil {
			return mapIdentityStorageError(err)
		}

		profileID := s.ids.New()
		privacyJSON, _ := json.Marshal(defaultPrivacy)
		if err := tx.CreateUserProfile(ctx, sqlc.CreateUserProfileParams{
			ID:               profileID,
			UserID:           userID,
			Nickname:         normalized.Nickname,
			AvatarUri:        sql.NullString{},
			Bio:              sql.NullString{},
			Locale:           "zh-CN",
			Timezone:         "Asia/Shanghai",
			PreferencesJson:  json.RawMessage(`{"schema_version":"1.0.0"}`),
			PrivacyJson:      privacyJSON,
			ModerationStatus: "approved",
		}); err != nil {
			return mapIdentityStorageError(err)
		}

		statsID := s.ids.New()
		if err := tx.CreateUserStatistics(ctx, sqlc.CreateUserStatisticsParams{
			ID:           statsID,
			UserID:       userID,
			StatsJson:    json.RawMessage(`{"schema_version":"1.0.0"}`),
			CalculatedAt: s.now().UTC(),
		}); err != nil {
			return mapIdentityStorageError(err)
		}

		session, err := s.issueSession(ctx, tx, userID, meta, false)
		if err != nil {
			return err
		}
		if err := tx.UpdateUserLastLogin(ctx, sqlc.UpdateUserLastLoginParams{
			LastLoginAt: sql.NullTime{Time: s.now().UTC(), Valid: true},
			ID:          userID,
		}); err != nil {
			return mapIdentityStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 201, Valid: true},
			ResourceType:   sql.NullString{String: "user", Valid: true},
			ResourceID:     sql.NullString{String: userID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapIdentityStorageError(err)
		}

		result = SessionResult{
			User:    userViewFromCreate(user),
			Profile: profileViewFromValues(profileID, userID, normalized.Nickname, privacyJSON),
			Privacy: defaultPrivacy,
			Session: session,
		}
		return nil
	})
	if err != nil {
		return SessionResult{}, err
	}

	return result, nil
}

func (s *Service) Login(
	ctx context.Context,
	input LoginInput,
	meta RequestMetadata,
) (SessionResult, error) {
	normalized, err := normalizeLoginInput(input)
	if err != nil {
		return SessionResult{}, err
	}

	allowed, err := s.allowRateLimit(
		ctx,
		"login:"+meta.IPAddress+":"+normalized.Identifier,
		s.cfg.LoginRateLimit,
		s.cfg.LoginRateWindow,
	)
	if err != nil {
		return SessionResult{}, err
	}
	if !allowed {
		return SessionResult{}, rateLimitError(time.Minute)
	}

	authIdentity, err := s.repo.GetAuthIdentityBySubject(
		ctx,
		sqlc.GetAuthIdentityBySubjectParams{
			Provider:        normalized.Provider,
			ProviderSubject: normalized.Identifier,
		},
	)
	if err != nil {
		_, _ = s.passwords.Verify(s.dummyHash, normalized.Password)
		if isStorageCode(err, database.CodeNotFound) {
			return SessionResult{}, authInvalidError()
		}
		return SessionResult{}, mapIdentityStorageError(err)
	}
	if !authIdentity.PasswordHash.Valid {
		_, _ = s.passwords.Verify(s.dummyHash, normalized.Password)
		return SessionResult{}, authInvalidError()
	}

	valid, err := s.passwords.Verify(authIdentity.PasswordHash.String, normalized.Password)
	if err != nil {
		return SessionResult{}, apperror.New(
			"INTERNAL_ERROR",
			"Password verification failed.",
			true,
			nil,
		)
	}
	if !valid {
		return SessionResult{}, authInvalidError()
	}

	user, err := s.repo.GetUser(ctx, authIdentity.UserID)
	if err != nil {
		return SessionResult{}, mapIdentityStorageError(err)
	}
	if err := ensureActiveAccount(user.Status); err != nil {
		return SessionResult{}, err
	}

	profile, err := s.repo.GetUserProfile(ctx, user.ID)
	if err != nil {
		return SessionResult{}, mapIdentityStorageError(err)
	}

	var session SessionCredentials
	if err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		issued, err := s.issueSession(
			ctx,
			tx,
			user.ID,
			meta,
			normalized.RememberDevice,
		)
		if err != nil {
			return err
		}
		if err := tx.UpdateUserLastLogin(ctx, sqlc.UpdateUserLastLoginParams{
			LastLoginAt: sql.NullTime{Time: s.now().UTC(), Valid: true},
			ID:          user.ID,
		}); err != nil {
			return mapIdentityStorageError(err)
		}
		session = issued
		return nil
	}); err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		User:    userView(user),
		Profile: profileView(profile),
		Privacy: privacyFromJSON(profile.PrivacyJson),
		Session: session,
	}, nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	rawSessionToken string,
) (AuthenticatedSession, error) {
	if rawSessionToken == "" {
		return AuthenticatedSession{}, authRequiredError()
	}

	session, err := s.repo.GetSessionUserByToken(ctx, hashTextBytes(rawSessionToken))
	if err != nil {
		if isStorageCode(err, database.CodeNotFound) {
			return AuthenticatedSession{}, authRequiredError()
		}
		return AuthenticatedSession{}, mapIdentityStorageError(err)
	}
	if session.RevokedAt.Valid {
		return AuthenticatedSession{}, apperror.New(
			"SESSION_REVOKED",
			"The session has been revoked.",
			false,
			nil,
		)
	}
	if !s.now().UTC().Before(session.SessionExpiresAt) {
		return AuthenticatedSession{}, authRequiredError()
	}
	if err := ensureActiveAccount(session.UserStatus); err != nil {
		return AuthenticatedSession{}, err
	}

	return AuthenticatedSession{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		CSRFHash:  session.CsrfTokenHash,
		ExpiresAt: session.SessionExpiresAt,
		User: UserView{
			ID:       session.UserID,
			PublicNo: session.PublicNo,
			Status:   session.UserStatus,
		},
		Profile: ProfileView{
			Nickname:  session.Nickname,
			AvatarURL: nullableString(session.AvatarUri),
		},
	}, nil
}

func (s *Service) ValidateCSRF(session AuthenticatedSession, rawToken string) error {
	if rawToken == "" {
		return apperror.New("CSRF_INVALID", "CSRF token is required.", false, nil)
	}
	provided := hashTextBytes(rawToken)
	if len(provided) != len(session.CSRFHash) ||
		subtle.ConstantTimeCompare(provided, session.CSRFHash) != 1 {
		return apperror.New("CSRF_INVALID", "CSRF token is invalid.", false, nil)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, session AuthenticatedSession) (MeView, error) {
	user, err := s.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return MeView{}, mapIdentityStorageError(err)
	}
	profile, err := s.repo.GetUserProfile(ctx, session.UserID)
	if err != nil {
		return MeView{}, mapIdentityStorageError(err)
	}
	stats, err := s.repo.GetUserStatistics(ctx, session.UserID)
	if err != nil {
		return MeView{}, mapIdentityStorageError(err)
	}

	return MeView{
		User:       userView(user),
		Profile:    profileView(profile),
		Privacy:    privacyFromJSON(profile.PrivacyJson),
		Statistics: statisticsView(stats),
		Capabilities: []string{
			"create_room",
			"join_room",
			"chat_waiting_room",
		},
	}, nil
}

func (s *Service) UpdateMe(
	ctx context.Context,
	session AuthenticatedSession,
	input UpdateMeInput,
) (ProfileView, PrivacyView, error) {
	if input.RowVersion == 0 {
		return ProfileView{}, PrivacyView{}, apperror.New(
			"VALIDATION_ERROR",
			"row_version is required.",
			false,
			nil,
		)
	}

	profile, err := s.repo.GetUserProfile(ctx, session.UserID)
	if err != nil {
		return ProfileView{}, PrivacyView{}, mapIdentityStorageError(err)
	}
	privacy := privacyFromJSON(profile.PrivacyJson)

	nickname := profile.Nickname
	avatar := profile.AvatarUri
	bio := profile.Bio
	moderationStatus := profile.ModerationStatus

	if patch := input.ProfilePatch; patch != nil {
		if patch.Nickname != nil {
			value := strings.TrimSpace(*patch.Nickname)
			if err := validateNickname(value); err != nil {
				return ProfileView{}, PrivacyView{}, err
			}
			nickname = value
			moderationStatus = "pending"
		}
		if patch.AvatarURL != nil {
			value := strings.TrimSpace(*patch.AvatarURL)
			if utf8.RuneCountInString(value) > 512 {
				return ProfileView{}, PrivacyView{}, validationError("avatar_url is too long")
			}
			avatar = sql.NullString{String: value, Valid: value != ""}
			moderationStatus = "pending"
		}
		if patch.Bio != nil {
			value := strings.TrimSpace(*patch.Bio)
			if utf8.RuneCountInString(value) > 500 {
				return ProfileView{}, PrivacyView{}, validationError("bio is too long")
			}
			bio = sql.NullString{String: value, Valid: value != ""}
		}
	}

	if patch := input.PrivacyPatch; patch != nil {
		applyPrivacyPatch(&privacy, patch)
	}
	if err := validatePrivacy(privacy); err != nil {
		return ProfileView{}, PrivacyView{}, err
	}
	privacyJSON, _ := json.Marshal(privacy)

	err = s.repo.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		Nickname:         nickname,
		AvatarUri:        avatar,
		Bio:              bio,
		Locale:           profile.Locale,
		Timezone:         profile.Timezone,
		PreferencesJson:  profile.PreferencesJson,
		PrivacyJson:      privacyJSON,
		ModerationStatus: moderationStatus,
		UserID:           session.UserID,
		RowVersion:       input.RowVersion,
	})
	if errors.Is(err, storage.ErrNoRowsAffected) {
		return ProfileView{}, PrivacyView{}, apperror.New(
			"VERSION_CONFLICT",
			"The profile was modified by another request.",
			true,
			nil,
		)
	}
	if err != nil {
		return ProfileView{}, PrivacyView{}, mapIdentityStorageError(err)
	}

	updated, err := s.repo.GetUserProfile(ctx, session.UserID)
	if err != nil {
		return ProfileView{}, PrivacyView{}, mapIdentityStorageError(err)
	}
	return profileView(updated), privacy, nil
}

func (s *Service) Logout(
	ctx context.Context,
	session AuthenticatedSession,
	allDevices bool,
) (int64, error) {
	now := sql.NullTime{Time: s.now().UTC(), Valid: true}
	reason := sql.NullString{String: "logout", Valid: true}

	if allDevices {
		count, err := s.repo.RevokeAllUserSessions(ctx, sqlc.RevokeAllUserSessionsParams{
			RevokedAt:    now,
			RevokeReason: reason,
			UserID:       session.UserID,
		})
		if err != nil {
			return 0, mapIdentityStorageError(err)
		}
		return count, nil
	}

	err := s.repo.RevokeSession(ctx, sqlc.RevokeSessionParams{
		RevokedAt:    now,
		RevokeReason: reason,
		ID:           session.SessionID,
	})
	if errors.Is(err, storage.ErrNoRowsAffected) {
		return 0, nil
	}
	if err != nil {
		return 0, mapIdentityStorageError(err)
	}
	return 1, nil
}

func (s *Service) issueSession(
	ctx context.Context,
	repo storage.Repository,
	userID string,
	meta RequestMetadata,
	rememberDevice bool,
) (SessionCredentials, error) {
	rawSessionToken, err := s.tokens.New()
	if err != nil {
		return SessionCredentials{}, apperror.New(
			"INTERNAL_ERROR",
			"Session creation failed.",
			true,
			nil,
		)
	}
	rawCSRFToken, err := s.tokens.New()
	if err != nil {
		return SessionCredentials{}, apperror.New(
			"INTERNAL_ERROR",
			"CSRF token creation failed.",
			true,
			nil,
		)
	}

	now := s.now().UTC()
	ttl := s.cfg.SessionTTL
	if rememberDevice {
		ttl = s.cfg.RememberSessionTTL
	}
	expiresAt := now.Add(ttl)
	sessionID := s.ids.New()

	if err := repo.CreateUserSession(ctx, sqlc.CreateUserSessionParams{
		ID:               sessionID,
		UserID:           userID,
		SessionTokenHash: hashTextBytes(rawSessionToken),
		CsrfTokenHash:    hashTextBytes(rawCSRFToken),
		DeviceLabel:      sql.NullString{String: meta.DeviceLabel, Valid: meta.DeviceLabel != ""},
		IpHash: sql.NullString{
			String: string(hashTextBytes(meta.IPAddress)),
			Valid:  meta.IPAddress != "",
		},
		UserAgentHash: sql.NullString{
			String: string(hashTextBytes(meta.UserAgent)),
			Valid:  meta.UserAgent != "",
		},
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}); err != nil {
		return SessionCredentials{}, mapIdentityStorageError(err)
	}

	return SessionCredentials{
		SessionID:    sessionID,
		SessionToken: rawSessionToken,
		CSRFToken:    rawCSRFToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) getOrCreateIdempotency(
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
				"The previous request with this idempotency key is still in progress.",
				true,
				nil,
			)
		}
		return record, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.IdempotencyRecord{}, false, mapIdentityStorageError(err)
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
		return sqlc.IdempotencyRecord{}, false, mapIdentityStorageError(err)
	}
	return sqlc.IdempotencyRecord{ID: recordID}, false, nil
}

func (s *Service) allowRateLimit(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, error) {
	decision, err := s.limiter.Allow(ctx, key, limit, window)
	if err != nil {
		return false, apperror.New(
			"DEPENDENCY_ERROR",
			"Rate limiting is temporarily unavailable.",
			true,
			nil,
		)
	}
	return decision.Allowed, nil
}

func (s *Service) identifierHash(identifier string) []byte {
	mac := hmac.New(sha256.New, s.hmacKey)
	_, _ = mac.Write([]byte("identity:" + identifier))
	return mac.Sum(nil)
}

func normalizeRegisterInput(input RegisterInput) (RegisterInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Identifier = strings.ToLower(strings.TrimSpace(input.Identifier))
	input.Nickname = strings.TrimSpace(input.Nickname)
	input.TermsVersion = strings.TrimSpace(input.TermsVersion)
	input.PrivacyVersion = strings.TrimSpace(input.PrivacyVersion)

	if input.Provider != "email" {
		return RegisterInput{}, validationError("provider must be email")
	}
	if err := validateEmail(input.Identifier); err != nil {
		return RegisterInput{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return RegisterInput{}, err
	}
	if err := validateNickname(input.Nickname); err != nil {
		return RegisterInput{}, err
	}
	if input.TermsVersion == "" || input.PrivacyVersion == "" {
		return RegisterInput{}, apperror.New(
			"TERMS_NOT_ACCEPTED",
			"Terms and privacy policy must be accepted.",
			false,
			nil,
		)
	}
	return input, nil
}

func normalizeLoginInput(input LoginInput) (LoginInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Identifier = strings.ToLower(strings.TrimSpace(input.Identifier))
	if input.Provider != "email" || input.Identifier == "" || input.Password == "" {
		return LoginInput{}, authInvalidError()
	}
	return input, nil
}

func validateEmail(value string) error {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return validationError("identifier must be a valid email address")
	}
	return nil
}

func validatePassword(value string) error {
	length := utf8.RuneCountInString(value)
	if length < 8 || length > 128 {
		return validationError("password must contain between 8 and 128 characters")
	}
	return nil
}

func validateNickname(value string) error {
	if value == "" || utf8.RuneCountInString(value) > 64 {
		return validationError("nickname must contain between 1 and 64 characters")
	}
	return nil
}

func validatePrivacy(value PrivacyView) error {
	if !oneOf(value.HistoryVisibility, "private", "friends", "public") {
		return validationError("history_visibility is invalid")
	}
	if !oneOf(value.OnlineVisibility, "private", "friends", "public") {
		return validationError("online_visibility is invalid")
	}
	if !oneOf(value.ProfileVisibility, "private", "friends", "public") {
		return validationError("profile_visibility is invalid")
	}
	if !oneOf(value.FriendRequestPolicy, "nobody", "friends_of_friends", "everyone") {
		return validationError("friend_request_policy is invalid")
	}
	return nil
}

func applyPrivacyPatch(privacy *PrivacyView, patch *PrivacyPatch) {
	if patch.HistoryVisibility != nil {
		privacy.HistoryVisibility = *patch.HistoryVisibility
	}
	if patch.OnlineVisibility != nil {
		privacy.OnlineVisibility = *patch.OnlineVisibility
	}
	if patch.ProfileVisibility != nil {
		privacy.ProfileVisibility = *patch.ProfileVisibility
	}
	if patch.FriendRequestPolicy != nil {
		privacy.FriendRequestPolicy = *patch.FriendRequestPolicy
	}
}

func privacyFromJSON(raw json.RawMessage) PrivacyView {
	if len(raw) == 0 {
		return defaultPrivacy
	}
	privacy := defaultPrivacy
	_ = json.Unmarshal(raw, &privacy)
	return privacy
}

func profileView(profile sqlc.UserProfile) ProfileView {
	return ProfileView{
		Nickname:         profile.Nickname,
		AvatarURL:        nullableString(profile.AvatarUri),
		Bio:              nullableString(profile.Bio),
		Locale:           profile.Locale,
		Timezone:         profile.Timezone,
		ModerationStatus: profile.ModerationStatus,
		RowVersion:       profile.RowVersion,
	}
}

func profileViewFromValues(
	id string,
	userID string,
	nickname string,
	privacyJSON []byte,
) ProfileView {
	_ = id
	_ = userID
	_ = privacyJSON
	return ProfileView{
		Nickname:         nickname,
		Locale:           "zh-CN",
		Timezone:         "Asia/Shanghai",
		ModerationStatus: "approved",
		RowVersion:       1,
	}
}

func userViewFromCreate(params sqlc.CreateUserParams) UserView {
	return UserView{
		ID:       params.ID,
		PublicNo: params.PublicNo,
		Status:   params.Status,
	}
}

func statisticsView(value sqlc.UserStatistic) StatisticsView {
	return StatisticsView{
		GamesStarted:      value.GamesStarted,
		GamesCompleted:    value.GamesCompleted,
		GamesWon:          value.GamesWon,
		ScriptMurderGames: value.ScriptMurderGames,
		WerewolfGames:     value.WerewolfGames,
		RPGGames:          value.RpgGames,
		Experience:        value.Experience,
		Level:             value.Level,
	}
}

func identifierHint(identifier string) string {
	parts := strings.SplitN(identifier, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	first := ""
	if parts[0] != "" {
		first = parts[0][:1]
	}
	return first + "***@" + parts[1]
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func hashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashTextBytes(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func hashJSON(value any) []byte {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return sum[:]
}

func ensureActiveAccount(status string) error {
	switch status {
	case "active":
		return nil
	case "suspended", "disabled", "deleted":
		return apperror.New(
			"ACCOUNT_RESTRICTED",
			"The account cannot access this operation.",
			false,
			nil,
		)
	default:
		return authInvalidError()
	}
}

func mapIdentityStorageError(err error) error {
	var storageErr *database.Error
	if errors.As(err, &storageErr) {
		switch storageErr.Code {
		case database.CodeConflict:
			return apperror.New(
				"IDENTIFIER_ALREADY_EXISTS",
				"The account identifier is already registered.",
				false,
				nil,
			)
		case database.CodeDependencyUnavailable, database.CodeDeadlineExceeded:
			return apperror.New(
				"DEPENDENCY_ERROR",
				"The identity service is temporarily unavailable.",
				true,
				nil,
			)
		}
	}
	return apperror.New("INTERNAL_ERROR", "The identity operation failed.", true, nil)
}

func isStorageCode(err error, code string) bool {
	var storageErr *database.Error
	return errors.As(err, &storageErr) && storageErr.Code == code
}

func validationError(message string) error {
	return apperror.New("VALIDATION_ERROR", message, false, nil)
}

func authInvalidError() error {
	return apperror.New(
		"AUTH_INVALID",
		"Authentication credentials are invalid.",
		false,
		nil,
	)
}

func authRequiredError() error {
	return apperror.New("AUTH_REQUIRED", "Authentication is required.", false, nil)
}

func rateLimitError(retryAfter time.Duration) error {
	return apperror.New(
		"RATE_LIMITED",
		"Too many requests. Try again later.",
		true,
		map[string]int64{"retry_after_seconds": max(int64(retryAfter/time.Second), 1)},
	)
}

func oneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}
