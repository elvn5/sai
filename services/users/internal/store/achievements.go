package store

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/users/internal/models"
)

// ErrInUse: the achievement has been awarded, so it can only be archived.
var ErrInUse = errors.New("achievement in use")

// awardCodeAlphabet leaves out 0/O and 1/I/L, which are easy to misread.
const awardCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// AwardCodeLength is the length of a user's award code.
const AwardCodeLength = 8

const achievementColumns = `a.id::text, a.title, a.description, a.icon, a.rarity, a.is_active,
	a.sort_order, a.created_at, a.updated_at`

func scanAchievementInto(a *models.Achievement, dest ...any) []any {
	return append([]any{&a.ID, &a.Title, &a.Description, &a.Icon, &a.Rarity, &a.IsActive, &a.SortOrder}, dest...)
}

func scanAchievement(row pgx.Row) (models.Achievement, error) {
	var a models.Achievement
	var created, updated time.Time
	err := row.Scan(scanAchievementInto(&a, &created, &updated)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	finishAchievement(&a, created, updated)
	return a, err
}

func finishAchievement(a *models.Achievement, created, updated time.Time) {
	a.CreatedAt, a.UpdatedAt = jsontime.New(created), jsontime.New(updated)
	a.Points = models.AchievementRarities[a.Rarity]
	if a.Title == nil {
		a.Title = map[string]string{}
	}
	if a.Description == nil {
		a.Description = map[string]string{}
	}
}

func (s *Store) collectAchievements(ctx context.Context, query string, args ...any) ([]models.Achievement, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Achievement, error) { return scanAchievement(r) })
}

const achievementOrder = ` ORDER BY a.sort_order, a.created_at`

// Achievements lists the catalog; activeOnly hides archived entries.
func (s *Store) Achievements(ctx context.Context, activeOnly bool) ([]models.Achievement, error) {
	return s.collectAchievements(ctx, `SELECT `+achievementColumns+` FROM achievements a
		WHERE a.is_active OR NOT $1`+achievementOrder, activeOnly)
}

func (s *Store) Achievement(ctx context.Context, id string) (models.Achievement, error) {
	return scanAchievement(s.db.QueryRow(ctx, `SELECT `+achievementColumns+` FROM achievements a
		WHERE a.id::text = $1`, id))
}

func (s *Store) InsertAchievement(ctx context.Context, a models.Achievement) (models.Achievement, error) {
	return scanAchievement(s.db.QueryRow(ctx, `INSERT INTO achievements AS a
		(title, description, icon, rarity, is_active, sort_order) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+achievementColumns, a.Title, a.Description, a.Icon, a.Rarity, a.IsActive, a.SortOrder))
}

func (s *Store) UpdateAchievement(ctx context.Context, a models.Achievement) (models.Achievement, error) {
	return scanAchievement(s.db.QueryRow(ctx, `UPDATE achievements AS a SET title = $2, description = $3,
		icon = $4, rarity = $5, is_active = $6, sort_order = $7, updated_at = now()
		WHERE a.id::text = $1 RETURNING `+achievementColumns,
		a.ID, a.Title, a.Description, a.Icon, a.Rarity, a.IsActive, a.SortOrder))
}

// DeleteAchievement removes a catalog entry nobody has earned yet.
func (s *Store) DeleteAchievement(ctx context.Context, id string) error {
	var awarded bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_achievements
		WHERE achievement_id::text = $1)`, id).Scan(&awarded); err != nil {
		return err
	}
	if awarded {
		return ErrInUse
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM achievements WHERE id::text = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AwardCode returns the user's award code, generating it on first use.
func (s *Store) AwardCode(ctx context.Context, userID string) (string, error) {
	var code *string
	if err := s.db.QueryRow(ctx, `SELECT award_code FROM users WHERE id::text = $1`, userID).Scan(&code); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if code != nil {
		return *code, nil
	}
	for range 5 {
		candidate := newAwardCode()
		// The WHERE keeps a concurrent first request's code.
		_, err := s.db.Exec(ctx, `UPDATE users SET award_code = $2 WHERE id::text = $1 AND award_code IS NULL`,
			userID, candidate)
		if errors.Is(duplicate(err), ErrDuplicate) {
			continue
		}
		if err != nil {
			return "", err
		}
		return s.AwardCode(ctx, userID)
	}
	return "", errors.New("could not generate a unique award code")
}

func newAwardCode() string {
	b := make([]byte, AwardCodeLength)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = awardCodeAlphabet[int(b[i])%len(awardCodeAlphabet)]
	}
	return string(b)
}

// ByAwardCode finds the user who owns code.
func (s *Store) ByAwardCode(ctx context.Context, code string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE award_code = $1`, code))
}

// EarnedAchievementIDs lists the achievements userID already has.
func (s *Store) EarnedAchievementIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT achievement_id::text FROM user_achievements
		WHERE user_id::text = $1`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

const awardSelect = `SELECT ua.id::text, ua.note, ua.awarded_at, ua.seen_at IS NULL,
	u.id::text, u.first_name, u.username, u.avatar_url,
	g.id::text, g.first_name, g.username, g.avatar_url,
	` + achievementColumns + `
	FROM user_achievements ua
	JOIN users u ON u.id = ua.user_id
	JOIN achievements a ON a.id = ua.achievement_id
	LEFT JOIN users g ON g.id = ua.awarded_by`

func scanAward(row pgx.Row) (models.Award, error) {
	var w models.Award
	var awardedAt, created, updated time.Time
	var giverID *string
	var giver models.Person
	dest := []any{&w.ID, &w.Note, &awardedAt, &w.IsNew,
		&w.User.ID, &w.User.FirstName, &w.User.Username, &w.User.AvatarURL,
		&giverID, &giver.FirstName, &giver.Username, &giver.AvatarURL}
	dest = append(dest, scanAchievementInto(&w.Achievement, &created, &updated)...)
	err := row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return w, ErrNotFound
	}
	w.AwardedAt = jsontime.New(awardedAt)
	if giverID != nil {
		giver.ID = *giverID
		w.AwardedBy = &giver
	}
	finishAchievement(&w.Achievement, created, updated)
	return w, err
}

func (s *Store) collectAwards(ctx context.Context, query string, args ...any) ([]models.Award, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Award, error) { return scanAward(r) })
}

// AwardsOf lists what userID has earned, newest first.
func (s *Store) AwardsOf(ctx context.Context, userID string) ([]models.Award, error) {
	return s.collectAwards(ctx, awardSelect+` WHERE ua.user_id::text = $1 ORDER BY ua.awarded_at DESC`, userID)
}

// AwardsGivenBy lists what partnerID has handed out, newest first.
func (s *Store) AwardsGivenBy(ctx context.Context, partnerID string, limit int) ([]models.Award, error) {
	return s.collectAwards(ctx, awardSelect+` WHERE ua.awarded_by::text = $1
		ORDER BY ua.awarded_at DESC LIMIT $2`, partnerID, limit)
}

// RecentAwards lists the latest awards across everyone.
func (s *Store) RecentAwards(ctx context.Context, limit int) ([]models.Award, error) {
	return s.collectAwards(ctx, awardSelect+` ORDER BY ua.awarded_at DESC LIMIT $1`, limit)
}

// Award gives userID the achievement. ErrDuplicate: they already have it.
func (s *Store) Award(ctx context.Context, userID, achievementID, partnerID string, note *string) (models.Award, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO user_achievements (user_id, achievement_id, awarded_by, note)
		VALUES ($1, $2, $3, $4) RETURNING id::text`, userID, achievementID, partnerID, note).Scan(&id)
	if err != nil {
		return models.Award{}, duplicate(err)
	}
	return scanAward(s.db.QueryRow(ctx, awardSelect+` WHERE ua.id::text = $1`, id))
}

// MarkAwardsSeen clears the "new" flag on everything userID has earned.
func (s *Store) MarkAwardsSeen(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `UPDATE user_achievements SET seen_at = now()
		WHERE user_id::text = $1 AND seen_at IS NULL`, userID)
	return err
}

func (s *Store) RevokeAward(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM user_achievements WHERE id::text = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// HallOfFame ranks users by achievement points (ties: who got there
// first), with each one's achievements, rarest first.
func (s *Store) HallOfFame(ctx context.Context, limit int) ([]models.HallEntry, error) {
	rows, err := s.db.Query(ctx, `WITH scores AS (
		SELECT ua.user_id,
		       SUM(CASE a.rarity WHEN 'legendary' THEN 100 WHEN 'epic' THEN 50 WHEN 'rare' THEN 25 ELSE 10 END) AS points,
		       COUNT(*) AS cnt, MAX(ua.awarded_at) AS last_at
		FROM user_achievements ua JOIN achievements a ON a.id = ua.achievement_id
		GROUP BY ua.user_id
		ORDER BY points DESC, last_at ASC
		LIMIT $1)
	SELECT u.id::text, u.first_name, u.username, u.avatar_url, s.points, s.cnt
	FROM scores s JOIN users u ON u.id = s.user_id
	ORDER BY s.points DESC, s.last_at ASC`, limit)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.HallEntry, error) {
		var e models.HallEntry
		err := r.Scan(&e.User.ID, &e.User.FirstName, &e.User.Username, &e.User.AvatarURL, &e.Points, &e.Count)
		return e, err
	})
	if err != nil || len(entries) == 0 {
		return []models.HallEntry{}, err
	}

	ids := make([]string, len(entries))
	byUser := make(map[string]*models.HallEntry, len(entries))
	for i := range entries {
		entries[i].Rank = i + 1
		entries[i].Achievements = []models.Achievement{}
		ids[i] = entries[i].User.ID
		byUser[entries[i].User.ID] = &entries[i]
	}
	rows, err = s.db.Query(ctx, `SELECT ua.user_id::text, `+achievementColumns+`
		FROM user_achievements ua JOIN achievements a ON a.id = ua.achievement_id
		WHERE ua.user_id::text = ANY($1)
		ORDER BY CASE a.rarity WHEN 'legendary' THEN 0 WHEN 'epic' THEN 1 WHEN 'rare' THEN 2 ELSE 3 END, ua.awarded_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var a models.Achievement
		var created, updated time.Time
		if err := rows.Scan(append([]any{&userID}, scanAchievementInto(&a, &created, &updated)...)...); err != nil {
			return nil, err
		}
		finishAchievement(&a, created, updated)
		byUser[userID].Achievements = append(byUser[userID].Achievements, a)
	}
	return entries, rows.Err()
}
