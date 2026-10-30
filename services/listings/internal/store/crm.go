package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// ErrNotManual: the client or participant is an account (signed up
// themselves), which the partner can't delete — only cancel.
var ErrNotManual = errors.New("not added by hand")

// A participant's client: the one they were added as, or — for an account
// that signed up — the partner's client with that account.
const participantClientJoin = `JOIN tours t ON t.id = p.tour_id
	JOIN crm_clients c ON c.id = p.client_id OR (p.user_id IS NOT NULL AND c.user_id = p.user_id AND c.partner_id = t.user_id)`

// What a participant owes and has paid.
const dueSQL = `coalesce(p.amount_due, t.price, 0)`

// signedSQL is a payment's amount with refunds negative.
const signedSQL = `CASE WHEN pay.kind = 'refund' THEN -pay.amount ELSE pay.amount END`
const paidSQL = `(SELECT coalesce(sum(` + signedSQL + `), 0) FROM tour_payments pay WHERE pay.participant_id = p.id)`

const clientColumns = `c.id::text, c.user_id::text, c.name, c.phone, c.telegram, c.note, c.tags, c.source, c.created_at,
	coalesce(a.tours, 0), a.last_tour, coalesce(a.due, 0), coalesce(a.paid, 0)`

// clientStats sums a client's places with the partner (c.partner_id).
const clientStats = `LEFT JOIN LATERAL (
	SELECT count(*)::int AS tours, to_char(max(t.start_date), 'YYYY-MM-DD') AS last_tour,
		sum(` + dueSQL + `)::int AS due, sum(` + paidSQL + `)::int AS paid
	FROM tour_participants p JOIN tours t ON t.id = p.tour_id
	WHERE t.user_id = c.partner_id AND p.` + activeSQL + `
		AND (p.client_id = c.id OR (c.user_id IS NOT NULL AND p.user_id = c.user_id))
) a ON true`

func scanClient(r pgx.Row) (models.CRMClient, error) {
	var c models.CRMClient
	var created time.Time
	err := r.Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Telegram, &c.Note, &c.Tags, &c.Source, &created,
		&c.Stats.Tours, &c.Stats.LastTour, &c.Stats.Due, &c.Stats.Paid)
	c.CreatedAt = jsontime.New(created)
	c.Tags = nonNil(c.Tags)
	return c, err
}

// CRMClients lists the partner's clients, newest first.
func (s *Store) CRMClients(ctx context.Context, partnerID string) ([]models.CRMClient, error) {
	rows, err := s.db.Query(ctx, `SELECT `+clientColumns+` FROM crm_clients c `+clientStats+`
		WHERE c.partner_id = $1 ORDER BY c.created_at DESC`, partnerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMClient, error) { return scanClient(r) })
}

// CRMClient is one of the partner's clients with their places on the
// partner's tours, latest first.
func (s *Store) CRMClient(ctx context.Context, partnerID, id string) (models.CRMClient, error) {
	c, err := scanClient(s.db.QueryRow(ctx, `SELECT `+clientColumns+` FROM crm_clients c `+clientStats+`
		WHERE c.partner_id = $1 AND c.id = $2`, partnerID, id))
	if err != nil {
		return c, notFound(err)
	}
	rows, err := s.db.Query(ctx, `SELECT p.id::text, t.id::text, t.title, t.slug, t.locale,
		to_char(t.start_date, 'YYYY-MM-DD'), p.status, `+dueSQL+`, `+paidSQL+`::int, p.arrived_at IS NOT NULL
		FROM tour_participants p `+participantClientJoin+`
		WHERE c.id = $1 AND t.user_id = $2 ORDER BY t.start_date DESC`, id, partnerID)
	if err != nil {
		return c, err
	}
	c.History, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMClientTour, error) {
		var h models.CRMClientTour
		err := r.Scan(&h.ParticipantID, &h.TourID, &h.Title, &h.Slug, &h.Locale, &h.StartDate, &h.Status, &h.Due, &h.Paid, &h.Arrived)
		return h, err
	})
	if c.History == nil {
		c.History = []models.CRMClientTour{}
	}
	return c, err
}

// ClientInput is what the partner types in about a client.
type ClientInput struct {
	Name, Phone, Telegram, Note, Source *string
	Tags                                []string
}

// CreateClient adds a client by hand (someone without an account here).
func (s *Store) CreateClient(ctx context.Context, partnerID string, in ClientInput) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO crm_clients (partner_id, name, phone, telegram, note, tags, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		partnerID, in.Name, in.Phone, in.Telegram, in.Note, nonNil(in.Tags), in.Source).Scan(&id)
	return id, err
}

// UpdateClient replaces what the partner typed in about a client.
func (s *Store) UpdateClient(ctx context.Context, partnerID, id string, in ClientInput) error {
	tag, err := s.db.Exec(ctx, `UPDATE crm_clients SET name = $3, phone = $4, telegram = $5, note = $6, tags = $7,
		source = $8, updated_at = now() WHERE partner_id = $1 AND id = $2`,
		partnerID, id, in.Name, in.Phone, in.Telegram, in.Note, nonNil(in.Tags), in.Source)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteClient removes a client the partner added by hand, with their
// places and payments.
func (s *Store) DeleteClient(ctx context.Context, partnerID, id string) error {
	var account bool
	err := s.db.QueryRow(ctx, `SELECT user_id IS NOT NULL FROM crm_clients WHERE partner_id = $1 AND id = $2`,
		partnerID, id).Scan(&account)
	if err != nil {
		return notFound(err)
	}
	if account {
		return ErrNotManual
	}
	_, err = s.db.Exec(ctx, `DELETE FROM crm_clients WHERE partner_id = $1 AND id = $2`, partnerID, id)
	return err
}

// IsClient reports whether the account is one of the partner's clients.
func (s *Store) IsClient(ctx context.Context, partnerID, userID string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM crm_clients WHERE partner_id = $1 AND user_id = $2)`,
		partnerID, userID).Scan(&ok)
	return ok, err
}

// Group lists everyone on a tour (places first, then the waitlist and the
// cancelled, each in sign-up order) with their payments.
func (s *Store) Group(ctx context.Context, tourID string) ([]models.CRMParticipant, error) {
	rows, err := s.db.Query(ctx, `SELECT p.id::text, c.id::text, p.user_id::text, c.name, c.phone, c.telegram,
		p.status, p.joined_at, p.arrived_at, p.amount_due, `+dueSQL+`, `+paidSQL+`::int,
		CASE WHEN `+profileFormWins+` THEN pp.form ELSE p.form END,
		CASE WHEN `+profileFormWins+` THEN pp.updated_at ELSE p.form_at END, p.waiver_at, p.waiver_version
		FROM tour_participants p `+participantClientJoin+`
		LEFT JOIN participant_profiles pp ON pp.user_id = p.user_id
		WHERE p.tour_id = $1
		ORDER BY CASE p.status WHEN 'waitlist' THEN 1 WHEN 'cancelled' THEN 2 ELSE 0 END, p.joined_at`, tourID)
	if err != nil {
		return nil, err
	}
	group, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMParticipant, error) {
		var p models.CRMParticipant
		var joined time.Time
		var arrived, formAt, waiverAt *time.Time
		err := r.Scan(&p.ID, &p.ClientID, &p.UserID, &p.Name, &p.Phone, &p.Telegram, &p.Status, &joined, &arrived,
			&p.AmountDue, &p.Due, &p.Paid, &p.Form, &formAt, &waiverAt, &p.WaiverVersion)
		p.JoinedAt, p.ArrivedAt = jsontime.New(joined), jsontime.Ptr(arrived)
		p.FormAt, p.WaiverAt = jsontime.Ptr(formAt), jsontime.Ptr(waiverAt)
		p.Payments = []models.CRMPayment{}
		return p, err
	})
	if err != nil || len(group) == 0 {
		return group, err
	}
	rows, err = s.db.Query(ctx, `SELECT pay.participant_id::text, pay.id::text, pay.kind, pay.amount, pay.method,
		to_char(pay.paid_on, 'YYYY-MM-DD'), pay.note, pay.created_at
		FROM tour_payments pay JOIN tour_participants p ON p.id = pay.participant_id
		WHERE p.tour_id = $1 ORDER BY pay.paid_on, pay.created_at`, tourID)
	if err != nil {
		return nil, err
	}
	index := map[string]int{}
	for i, p := range group {
		index[p.ID] = i
	}
	_, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (struct{}, error) {
		var owner string
		var pay models.CRMPayment
		var created time.Time
		if err := r.Scan(&owner, &pay.ID, &pay.Kind, &pay.Amount, &pay.Method, &pay.PaidOn, &pay.Note, &created); err != nil {
			return struct{}{}, err
		}
		pay.CreatedAt = jsontime.New(created)
		if i, ok := index[owner]; ok {
			group[i].Payments = append(group[i].Payments, pay)
		}
		return struct{}{}, nil
	})
	return group, err
}

// ParticipantTour is the tour a participant (by row id) is on, and
// whether they're an account.
func (s *Store) ParticipantTour(ctx context.Context, participantID string) (tourID string, userID *string, err error) {
	err = s.db.QueryRow(ctx, `SELECT tour_id::text, user_id::text FROM tour_participants WHERE id = $1`,
		participantID).Scan(&tourID, &userID)
	return tourID, userID, notFound(err)
}

// AddParticipant puts the partner's client on a tour with status (a place
// or the waitlist); a client cancelled earlier is put back. ErrFull when
// a place is asked for and none is left.
func (s *Store) AddParticipant(ctx context.Context, tourID, partnerID, clientID, status string) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID *string
	if err := tx.QueryRow(ctx, `SELECT user_id::text FROM crm_clients WHERE partner_id = $1 AND id = $2`,
		partnerID, clientID).Scan(&userID); err != nil {
		return "", notFound(err)
	}
	if status != StatusWaitlist {
		if err := checkPlace(ctx, tx, tourID); err != nil {
			return "", err
		}
	}
	var id, previous string
	column, person := "client_id", clientID
	if userID != nil {
		column, person = "user_id", *userID
	}
	err = tx.QueryRow(ctx, `SELECT id::text, status FROM tour_participants WHERE tour_id = $1 AND `+column+` = $2`,
		tourID, person).Scan(&id, &previous)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `INSERT INTO tour_participants (tour_id, `+column+`, status) VALUES ($1, $2, $3)
			RETURNING id::text`, tourID, person, status).Scan(&id)
	case err == nil && previous != StatusCancelled:
		return "", ErrDuplicate
	case err == nil:
		_, err = tx.Exec(ctx, `UPDATE tour_participants SET status = $2, joined_at = now(), arrived_at = NULL
			WHERE id = $1`, id, status)
	}
	if err != nil {
		return "", duplicate(err)
	}
	return id, tx.Commit(ctx)
}

// checkPlace locks the tour and fails with ErrFull when it has no free
// place.
func checkPlace(ctx context.Context, tx pgx.Tx, tourID string) error {
	var capacity *int
	var taken int
	err := tx.QueryRow(ctx, `SELECT capacity, (SELECT count(*) FROM tour_participants p
		WHERE p.tour_id = t.id AND p.`+activeSQL+`)::int FROM tours t WHERE id = $1 FOR UPDATE`, tourID).Scan(&capacity, &taken)
	if err != nil {
		return notFound(err)
	}
	if capacity != nil && taken >= *capacity {
		return ErrFull
	}
	return nil
}

// ParticipantChange is what the partner changes about a participant; nil
// fields stay as they are.
type ParticipantChange struct {
	Status *string
	// AmountDue: set to Due (nil: back to the tour's price).
	SetAmountDue bool
	AmountDue    *int
	Arrived      *bool
}

// UpdateParticipant applies a change. Taking a place needs a free one
// (ErrFull); giving one up passes it to the first on the waitlist, who is
// returned.
func (s *Store) UpdateParticipant(ctx context.Context, participantID string, ch ParticipantChange, now time.Time) (*Promoted, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tourID, status string
	if err := tx.QueryRow(ctx, `SELECT tour_id::text, status FROM tour_participants WHERE id = $1`,
		participantID).Scan(&tourID, &status); err != nil {
		return nil, notFound(err)
	}
	active := func(s string) bool { return s == StatusNew || s == StatusConfirmed }
	var promoted *Promoted
	if ch.Status != nil && *ch.Status != status {
		if active(*ch.Status) && !active(status) {
			if err := checkPlace(ctx, tx, tourID); err != nil {
				return nil, err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE tour_participants SET status = $2,
			arrived_at = CASE WHEN $2 IN ('new', 'confirmed') THEN arrived_at END WHERE id = $1`,
			participantID, *ch.Status); err != nil {
			return nil, err
		}
		if active(status) && !active(*ch.Status) {
			if promoted, err = promote(ctx, tx, tourID); err != nil {
				return nil, err
			}
		}
	}
	if ch.SetAmountDue {
		if _, err := tx.Exec(ctx, `UPDATE tour_participants SET amount_due = $2 WHERE id = $1`,
			participantID, ch.AmountDue); err != nil {
			return nil, err
		}
	}
	if ch.Arrived != nil {
		var at *time.Time
		if *ch.Arrived {
			at = &now
		}
		if _, err := tx.Exec(ctx, `UPDATE tour_participants
			SET arrived_at = CASE WHEN $2::timestamptz IS NULL THEN NULL ELSE coalesce(arrived_at, $2) END
			WHERE id = $1`, participantID, at); err != nil {
			return nil, err
		}
	}
	return promoted, tx.Commit(ctx)
}

// DeleteParticipant takes a client the partner added off a tour (with the
// payments); a freed place goes to the waitlist.
func (s *Store) DeleteParticipant(ctx context.Context, participantID string) (*Promoted, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tourID, status string
	var account bool
	if err := tx.QueryRow(ctx, `SELECT tour_id::text, status, user_id IS NOT NULL FROM tour_participants WHERE id = $1`,
		participantID).Scan(&tourID, &status, &account); err != nil {
		return nil, notFound(err)
	}
	if account {
		return nil, ErrNotManual
	}
	if _, err := tx.Exec(ctx, `DELETE FROM tour_participants WHERE id = $1`, participantID); err != nil {
		return nil, err
	}
	var promoted *Promoted
	if status == StatusNew || status == StatusConfirmed {
		if promoted, err = promote(ctx, tx, tourID); err != nil {
			return nil, err
		}
	}
	return promoted, tx.Commit(ctx)
}

// PaymentInput is a payment the partner records.
type PaymentInput struct {
	// "payment" or "refund".
	Kind   string
	Amount int
	Method string
	PaidOn string
	Note   *string
}

func (s *Store) AddPayment(ctx context.Context, participantID string, in PaymentInput) (models.CRMPayment, error) {
	var p models.CRMPayment
	var created time.Time
	err := s.db.QueryRow(ctx, `INSERT INTO tour_payments (participant_id, kind, amount, method, paid_on, note)
		VALUES ($1, $2, $3, $4, $5::date, $6)
		RETURNING id::text, kind, amount, method, to_char(paid_on, 'YYYY-MM-DD'), note, created_at`,
		participantID, in.Kind, in.Amount, in.Method, in.PaidOn, in.Note).Scan(&p.ID, &p.Kind, &p.Amount, &p.Method, &p.PaidOn, &p.Note, &created)
	p.CreatedAt = jsontime.New(created)
	return p, err
}

// DeletePayment removes a payment on one of the partner's tours.
func (s *Store) DeletePayment(ctx context.Context, partnerID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM tour_payments pay USING tour_participants p, tours t
		WHERE pay.id = $2 AND p.id = pay.participant_id AND t.id = p.tour_id AND t.user_id = $1`, partnerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CRMSummary is the partner's money and next tours as of today (Bishkek).
func (s *Store) CRMSummary(ctx context.Context, partnerID string, today time.Time) (models.CRMSummary, error) {
	var out models.CRMSummary
	day := today.Format(time.DateOnly)
	err := s.db.QueryRow(ctx, `SELECT
		(SELECT count(*)::int FROM crm_clients WHERE partner_id = $1),
		(SELECT coalesce(sum(`+signedSQL+`), 0)::int FROM tour_payments pay
			JOIN tour_participants p ON p.id = pay.participant_id JOIN tours t ON t.id = p.tour_id
			WHERE t.user_id = $1 AND pay.paid_on >= date_trunc('month', $2::date)),
		(SELECT coalesce(sum(greatest(`+dueSQL+` - `+paidSQL+`, 0)), 0)::int
			FROM tour_participants p JOIN tours t ON t.id = p.tour_id
			WHERE t.user_id = $1 AND p.`+activeSQL+` AND t.end_date >= $2::date - 30)`,
		partnerID, day).Scan(&out.Clients, &out.PaidThisMonth, &out.Outstanding)
	if err != nil {
		return out, err
	}
	rows, err := s.db.Query(ctx, `SELECT t.id::text, t.title, t.slug, t.locale, to_char(t.start_date, 'YYYY-MM-DD'), t.capacity,
		count(p.id) FILTER (WHERE p.`+activeSQL+`)::int, count(p.id) FILTER (WHERE p.status = 'waitlist')::int,
		coalesce(sum(`+dueSQL+`) FILTER (WHERE p.`+activeSQL+`), 0)::int,
		coalesce(sum(`+paidSQL+`) FILTER (WHERE p.`+activeSQL+`), 0)::int
		FROM tours t LEFT JOIN tour_participants p ON p.tour_id = t.id
		WHERE t.user_id = $1 AND t.end_date >= $2::date
		GROUP BY t.id ORDER BY t.start_date, t.title LIMIT 10`, partnerID, day)
	if err != nil {
		return out, err
	}
	out.Upcoming, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMUpcomingTour, error) {
		var u models.CRMUpcomingTour
		err := r.Scan(&u.ID, &u.Title, &u.Slug, &u.Locale, &u.StartDate, &u.Capacity, &u.Taken, &u.Waitlist, &u.Due, &u.Paid)
		return u, err
	})
	if out.Upcoming == nil {
		out.Upcoming = []models.CRMUpcomingTour{}
	}
	return out, err
}

// SaveForm stores userID's form on a tour (nil keeps the one there) and,
// with acceptWaiver, that they accepted the rules (version) now.
func (s *Store) SaveForm(ctx context.Context, tourID, userID string, form *models.ParticipantForm, acceptWaiver bool, now time.Time) error {
	var waiver *string
	if acceptWaiver {
		v := models.WaiverVersion
		waiver = &v
	}
	tag, err := s.db.Exec(ctx, `UPDATE tour_participants SET
		form = coalesce($3, form), form_at = CASE WHEN $3::jsonb IS NULL THEN form_at ELSE $5 END,
		waiver_version = coalesce($4, waiver_version), waiver_at = CASE WHEN $4::text IS NULL THEN waiver_at ELSE $5 END
		WHERE tour_id = $1 AND user_id = $2 AND status <> 'cancelled'`, tourID, userID, form, waiver, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetParticipantForm stores a participant's form as the partner filled it
// (a client added by hand, or what an account told them).
func (s *Store) SetParticipantForm(ctx context.Context, participantID string, form *models.ParticipantForm, now time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE tour_participants SET form = $2::jsonb, form_at = CASE WHEN $2::jsonb IS NULL THEN NULL ELSE $3::timestamptz END
		WHERE id = $1`, participantID, form, now)
	return err
}

// profileFormWins: the participant's profile form (pp) is shown instead of
// the one on the tour (p) — there is none there, or the profile's is newer.
const profileFormWins = `pp.form IS NOT NULL AND (p.form IS NULL OR p.form_at IS NULL OR pp.updated_at > p.form_at)`

// MyForm is userID's form: the tour's or their profile's, whichever is
// newer; and when they accepted the rules.
func (s *Store) MyForm(ctx context.Context, tourID, userID string) (*models.ParticipantForm, *time.Time, error) {
	var form *models.ParticipantForm
	var waiverAt *time.Time
	err := s.db.QueryRow(ctx, `SELECT CASE WHEN `+profileFormWins+` THEN pp.form ELSE p.form END, p.waiver_at FROM tour_participants p
		LEFT JOIN participant_profiles pp ON pp.user_id = p.user_id
		WHERE p.tour_id = $1 AND p.user_id = $2`, tourID, userID).Scan(&form, &waiverAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return form, waiverAt, err
}

// ProfileForm is the form userID keeps in their profile and when they
// last saved it (nil, nil if none).
func (s *Store) ProfileForm(ctx context.Context, userID string) (*models.ParticipantForm, *time.Time, error) {
	var form models.ParticipantForm
	var updated time.Time
	err := s.db.QueryRow(ctx, `SELECT form, updated_at FROM participant_profiles WHERE user_id = $1`, userID).Scan(&form, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &form, &updated, nil
}

// SaveProfileForm keeps userID's form in their profile; nil removes it.
func (s *Store) SaveProfileForm(ctx context.Context, userID string, form *models.ParticipantForm, now time.Time) error {
	if form == nil {
		_, err := s.db.Exec(ctx, `DELETE FROM participant_profiles WHERE user_id = $1`, userID)
		return err
	}
	_, err := s.db.Exec(ctx, `INSERT INTO participant_profiles (user_id, form, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET form = EXCLUDED.form, updated_at = EXCLUDED.updated_at`, userID, form, now)
	return err
}
