package script

import (
	"context"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type LoadMode string

const (
	LoadModeStart   LoadMode = "start"
	LoadModeRuntime LoadMode = "runtime"
	LoadModeEditor  LoadMode = "editor"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) LoadPlayable(
	ctx context.Context,
	repo storage.Repository,
	scriptVersionID string,
	mode LoadMode,
) (PlayableVersion, error) {
	versionRow, err := repo.GetScriptVersion(ctx, scriptVersionID)
	if err != nil {
		return PlayableVersion{}, contentUnavailableError("The script version does not exist.")
	}
	var scriptRow sqlc.Script
	if mode == LoadModeRuntime {
		scriptRow, err = repo.GetScriptAny(ctx, versionRow.ScriptID)
	} else {
		scriptRow, err = repo.GetScript(ctx, versionRow.ScriptID)
	}
	if err != nil {
		return PlayableVersion{}, contentUnavailableError("The script does not exist.")
	}
	if scriptRow.GameType != "script_murder" {
		return PlayableVersion{}, contentUnavailableError("Only script-murder content is currently playable.")
	}
	if err := validateLoadableVersion(scriptRow, versionRow, mode); err != nil {
		return PlayableVersion{}, err
	}

	content, err := loadContent(ctx, repo, scriptRow, versionRow)
	if err != nil {
		return PlayableVersion{}, err
	}
	result := PlayableVersion{
		Script:  content.Script,
		Version: content.Version,
		Content: content,
	}
	if versionRow.RuleTemplateID.Valid {
		var template sqlc.RuleTemplate
		if mode == LoadModeStart {
			template, err = repo.GetActiveRuleTemplateForScript(
				ctx,
				sqlc.GetActiveRuleTemplateForScriptParams{
					ID:       versionRow.RuleTemplateID.String,
					GameType: scriptRow.GameType,
				},
			)
		} else {
			template, err = repo.GetRuleTemplateByID(ctx, versionRow.RuleTemplateID.String)
		}
		if err != nil {
			return PlayableVersion{}, contentUnavailableError(
				"The script rule template is unavailable.",
			)
		}
		mapped := ruleTemplateFromRow(template)
		result.RuleTemplate = &mapped
	}
	return result, nil
}

func (l *Loader) LoadEditor(
	ctx context.Context,
	repo storage.Repository,
	scriptVersionID string,
) (Content, error) {
	loaded, err := l.LoadPlayable(ctx, repo, scriptVersionID, LoadModeEditor)
	if err != nil {
		return Content{}, err
	}
	return loaded.Content, nil
}

func validateLoadableVersion(scriptRow sqlc.Script, versionRow sqlc.ScriptVersion, mode LoadMode) error {
	switch mode {
	case LoadModeStart:
		if oneOf(versionRow.Status, "published_private", "published_public") &&
			scriptRow.Status == "published" &&
			versionIsValidated(versionRow) {
			return nil
		}
		return contentUnavailableError("Only a validated published script version can be played.")
	case LoadModeRuntime:
		if oneOf(
			versionRow.Status,
			"published_private",
			"published_public",
			"unlisted",
			"archived",
		) && versionIsValidated(versionRow) {
			return nil
		}
		return contentUnavailableError("The bound game content version is unavailable.")
	case LoadModeEditor:
		return nil
	default:
		return contentUnavailableError("The content load mode is invalid.")
	}
}

func versionIsValidated(version sqlc.ScriptVersion) bool {
	return version.FrozenAt.Valid &&
		oneOf(version.ValidationStatus, "passed", "warning") &&
		version.BlockingIssueCount == 0
}

func loadContent(
	ctx context.Context,
	repo storage.Repository,
	scriptRow sqlc.Script,
	versionRow sqlc.ScriptVersion,
) (Content, error) {
	chapterRows, err := repo.ListScriptChapters(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load chapters", err)
	}
	stageRows, err := repo.ListScriptStages(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load stages", err)
	}
	characterRows, err := repo.ListScriptCharacters(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load characters", err)
	}
	clueRows, err := repo.ListScriptClues(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load clues", err)
	}
	endingRows, err := repo.ListScriptEndings(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load endings", err)
	}
	assetRows, err := repo.ListScriptAssets(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load assets", err)
	}
	searchPointRows, err := repo.ListScriptSearchPoints(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load search points", err)
	}
	pointClueRows, err := repo.ListScriptSearchPointClues(ctx, versionRow.ID)
	if err != nil {
		return Content{}, contentLoadError("load search point clues", err)
	}

	content := Content{
		Script:       scriptFromRow(scriptRow),
		Version:      versionFromRow(versionRow),
		Chapters:     make([]Chapter, 0, len(chapterRows)),
		Stages:       make([]Stage, 0, len(stageRows)),
		Characters:   make([]Character, 0, len(characterRows)),
		Clues:        make([]Clue, 0, len(clueRows)),
		Endings:      make([]Ending, 0, len(endingRows)),
		Assets:       make([]Asset, 0, len(assetRows)),
		SearchPoints: make([]SearchPoint, 0, len(searchPointRows)),
		PointClues:   make([]SearchPointClue, 0, len(pointClueRows)),
	}
	for _, row := range chapterRows {
		content.Chapters = append(content.Chapters, chapterFromRow(row))
	}
	for _, row := range stageRows {
		content.Stages = append(content.Stages, stageFromRow(row))
	}
	for _, row := range characterRows {
		content.Characters = append(content.Characters, characterFromRow(row))
	}
	for _, row := range clueRows {
		content.Clues = append(content.Clues, clueFromRow(row))
	}
	for _, row := range endingRows {
		content.Endings = append(content.Endings, endingFromRow(row))
	}
	for _, row := range assetRows {
		content.Assets = append(content.Assets, assetFromRow(row))
	}
	for _, row := range searchPointRows {
		content.SearchPoints = append(content.SearchPoints, searchPointFromRow(row))
	}
	for _, row := range pointClueRows {
		content.PointClues = append(content.PointClues, searchPointClueFromRow(row))
	}
	return content, nil
}

func contentUnavailableError(message string) error {
	return apperror.New("CONTENT_UNAVAILABLE", message, false, nil)
}

func contentLoadError(stage string, err error) error {
	if err == nil {
		return nil
	}
	return apperror.New(
		"DEPENDENCY_ERROR",
		"The script content could not be loaded.",
		true,
		map[string]string{"stage": stage},
	)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
