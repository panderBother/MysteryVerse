package identity

import (
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type UserView struct {
	ID       string `json:"id"`
	PublicNo string `json:"public_no"`
	Status   string `json:"status"`
}

type ProfileView struct {
	Nickname         string  `json:"nickname"`
	AvatarURL        *string `json:"avatar_url"`
	Bio              *string `json:"bio"`
	Locale           string  `json:"locale"`
	Timezone         string  `json:"timezone"`
	ModerationStatus string  `json:"moderation_status"`
	RowVersion       uint32  `json:"row_version"`
}

type PrivacyView struct {
	HistoryVisibility   string `json:"history_visibility"`
	OnlineVisibility    string `json:"online_visibility"`
	ProfileVisibility   string `json:"profile_visibility"`
	FriendRequestPolicy string `json:"friend_request_policy"`
}

type StatisticsView struct {
	GamesStarted      uint32 `json:"games_started"`
	GamesCompleted    uint32 `json:"games_completed"`
	GamesWon          uint32 `json:"games_won"`
	ScriptMurderGames uint32 `json:"script_murder_games"`
	WerewolfGames     uint32 `json:"werewolf_games"`
	RPGGames          uint32 `json:"rpg_games"`
	Experience        uint64 `json:"experience"`
	Level             uint32 `json:"level"`
}

type SessionCredentials struct {
	SessionID    string
	SessionToken string
	CSRFToken    string
	ExpiresAt    time.Time
}

type SessionResult struct {
	User             UserView
	Profile          ProfileView
	Privacy          PrivacyView
	Session          SessionCredentials
	IdempotentReplay bool
}

type MeView struct {
	User         UserView       `json:"user"`
	Profile      ProfileView    `json:"profile"`
	Privacy      PrivacyView    `json:"privacy"`
	Statistics   StatisticsView `json:"statistics"`
	Capabilities []string       `json:"capabilities"`
}

type AuthenticatedSession struct {
	SessionID string
	UserID    string
	CSRFHash  []byte
	ExpiresAt time.Time
	User      UserView
	Profile   ProfileView
}

type RegisterInput struct {
	Provider       string `json:"provider"`
	Identifier     string `json:"identifier"`
	Password       string `json:"password"`
	Nickname       string `json:"nickname"`
	TermsVersion   string `json:"terms_version"`
	PrivacyVersion string `json:"privacy_version"`
}

type LoginInput struct {
	Provider       string `json:"provider"`
	Identifier     string `json:"identifier"`
	Password       string `json:"password"`
	RememberDevice bool   `json:"remember_device"`
}

type ProfilePatch struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
}

type PrivacyPatch struct {
	HistoryVisibility   *string `json:"history_visibility"`
	OnlineVisibility    *string `json:"online_visibility"`
	ProfileVisibility   *string `json:"profile_visibility"`
	FriendRequestPolicy *string `json:"friend_request_policy"`
}

type UpdateMeInput struct {
	ProfilePatch *ProfilePatch `json:"profile_patch"`
	PrivacyPatch *PrivacyPatch `json:"privacy_patch"`
	RowVersion   uint32        `json:"row_version"`
}

type RequestMetadata struct {
	IPAddress      string
	UserAgent      string
	DeviceLabel    string
	IdempotencyKey string
}

var defaultPrivacy = PrivacyView{
	HistoryVisibility:   "private",
	OnlineVisibility:    "friends",
	ProfileVisibility:   "public",
	FriendRequestPolicy: "friends_of_friends",
}

func userView(user sqlc.User) UserView {
	return UserView{
		ID:       user.ID,
		PublicNo: user.PublicNo,
		Status:   user.Status,
	}
}
