package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// --- A tour's expenses and staff ---

func (s *Store) TourExpenses(ctx context.Context, tourID string) ([]models.CRMExpense, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, category, amount, note, created_at FROM tour_expenses
		WHERE tour_id = $1 ORDER BY created_at`, tourID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMExpense, error) {
		var e models.CRMExpense
		var created time.Time
		err := r.Scan(&e.ID, &e.Category, &e.Amount, &e.Note, &created)
		e.CreatedAt = jsontime.New(created)
		return e, err
	})
	if out == nil {
		out = []models.CRMExpense{}
	}
	return out, err
}

func (s *Store) AddExpense(ctx context.Context, tourID, category string, amount int, note *string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO tour_expenses (tour_id, category, amount, note) VALUES ($1, $2, $3, $4)`,
		tourID, category, amount, note)
	return err
}

// DeleteExpense removes an expense of one of the partner's tours.
func (s *Store) DeleteExpense(ctx context.Context, partnerID, id string) (string, error) {
	var tourID string
	err := s.db.QueryRow(ctx, `DELETE FROM tour_expenses e USING tours t
		WHERE e.id = $2 AND t.id = e.tour_id AND t.user_id = $1 RETURNING t.id::text`, partnerID, id).Scan(&tourID)
	return tourID, notFound(err)
}

func (s *Store) TourStaff(ctx context.Context, tourID string) ([]models.CRMStaff, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, role, name, phone, note FROM tour_staff
		WHERE tour_id = $1 ORDER BY created_at`, tourID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMStaff, error) {
		var m models.CRMStaff
		return m, r.Scan(&m.ID, &m.Role, &m.Name, &m.Phone, &m.Note)
	})
	if out == nil {
		out = []models.CRMStaff{}
	}
	return out, err
}

func (s *Store) AddStaff(ctx context.Context, tourID string, m models.CRMStaff) error {
	_, err := s.db.Exec(ctx, `INSERT INTO tour_staff (tour_id, role, name, phone, note) VALUES ($1, $2, $3, $4, $5)`,
		tourID, m.Role, m.Name, m.Phone, m.Note)
	return err
}

// DeleteStaff takes someone off one of the partner's tours.
func (s *Store) DeleteStaff(ctx context.Context, partnerID, id string) (string, error) {
	var tourID string
	err := s.db.QueryRow(ctx, `DELETE FROM tour_staff m USING tours t
		WHERE m.id = $2 AND t.id = m.tour_id AND t.user_id = $1 RETURNING t.id::text`, partnerID, id).Scan(&tourID)
	return tourID, notFound(err)
}

// --- Leads ---

const leadColumns = `l.id::text, c.id::text, c.user_id::text, c.name, c.source,
	t.id::text, t.title, to_char(t.start_date, 'YYYY-MM-DD'), l.stage, l.lost_reason, l.note, l.created_at, l.updated_at`

func scanLead(r pgx.Row) (models.CRMLead, error) {
	var l models.CRMLead
	var tourID, title, start *string
	var created, updated time.Time
	err := r.Scan(&l.ID, &l.Client.ID, &l.Client.UserID, &l.Client.Name, &l.Source,
		&tourID, &title, &start, &l.Stage, &l.LostReason, &l.Note, &created, &updated)
	if tourID != nil {
		l.Tour = &models.CRMTourRef{ID: *tourID, Title: *title, StartDate: *start}
	}
	l.CreatedAt, l.UpdatedAt = jsontime.New(created), jsontime.New(updated)
	return l, err
}

// Leads lists the partner's leads (of a client, when clientID is set),
// latest change first.
func (s *Store) Leads(ctx context.Context, partnerID, clientID string) ([]models.CRMLead, error) {
	rows, err := s.db.Query(ctx, `SELECT `+leadColumns+` FROM crm_leads l
		JOIN crm_clients c ON c.id = l.client_id LEFT JOIN tours t ON t.id = l.tour_id
		WHERE l.partner_id = $1 AND ($2 = '' OR l.client_id::text = $2)
		ORDER BY l.updated_at DESC LIMIT 500`, partnerID, clientID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMLead, error) { return scanLead(r) })
	if out == nil {
		out = []models.CRMLead{}
	}
	return out, err
}

func (s *Store) Lead(ctx context.Context, partnerID, id string) (models.CRMLead, error) {
	l, err := scanLead(s.db.QueryRow(ctx, `SELECT `+leadColumns+` FROM crm_leads l
		JOIN crm_clients c ON c.id = l.client_id LEFT JOIN tours t ON t.id = l.tour_id
		WHERE l.partner_id = $1 AND l.id = $2`, partnerID, id))
	return l, notFound(err)
}

// LeadInput is what the partner sets on a lead; nil fields stay as they
// are (on create: stage "new").
type LeadInput struct {
	ClientID   *string
	TourID     *string
	SetTour    bool
	Stage      *string
	LostReason *string
	SetLost    bool
	Note       *string
	SetNote    bool
}

// ownedRefs checks that the client and the tour (when given) are the
// partner's.
func (s *Store) ownedRefs(ctx context.Context, partnerID string, clientID, tourID *string) error {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT ($2::uuid IS NULL OR EXISTS (SELECT 1 FROM crm_clients WHERE id = $2 AND partner_id = $1))
		AND ($3::uuid IS NULL OR EXISTS (SELECT 1 FROM tours WHERE id = $3 AND user_id = $1))`,
		partnerID, clientID, tourID).Scan(&ok)
	if err == nil && !ok {
		return ErrNotFound
	}
	return err
}

func (s *Store) CreateLead(ctx context.Context, partnerID string, in LeadInput) (string, error) {
	if err := s.ownedRefs(ctx, partnerID, in.ClientID, in.TourID); err != nil {
		return "", err
	}
	stage := "new"
	if in.Stage != nil {
		stage = *in.Stage
	}
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO crm_leads (partner_id, client_id, tour_id, stage, lost_reason, note)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		partnerID, in.ClientID, in.TourID, stage, in.LostReason, in.Note).Scan(&id)
	return id, err
}

func (s *Store) UpdateLead(ctx context.Context, partnerID, id string, in LeadInput) error {
	if err := s.ownedRefs(ctx, partnerID, nil, in.TourID); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `UPDATE crm_leads SET
		stage = coalesce($3, stage),
		tour_id = CASE WHEN $4 THEN $5::uuid ELSE tour_id END,
		lost_reason = CASE WHEN $6 THEN $7 WHEN coalesce($3, stage) <> 'lost' THEN NULL ELSE lost_reason END,
		note = CASE WHEN $8 THEN $9 ELSE note END,
		updated_at = now()
		WHERE partner_id = $1 AND id = $2`,
		partnerID, id, in.Stage, in.SetTour, in.TourID, in.SetLost, in.LostReason, in.SetNote, in.Note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteLead(ctx context.Context, partnerID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM crm_leads WHERE partner_id = $1 AND id = $2`, partnerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// OpenInquiry makes a traveler who writes to a partner their client (from
// Ermekten) with a new lead, unless one is open already.
func (s *Store) OpenInquiry(ctx context.Context, partnerID, userID string) error {
	_, err := s.db.Exec(ctx, `WITH client AS (
			INSERT INTO crm_clients (partner_id, user_id, source) VALUES ($1, $2, 'ermekten')
			ON CONFLICT (partner_id, user_id) DO UPDATE SET partner_id = EXCLUDED.partner_id
			RETURNING id)
		INSERT INTO crm_leads (partner_id, client_id)
		SELECT $1, id FROM client c WHERE NOT EXISTS (
			SELECT 1 FROM crm_leads l WHERE l.client_id = c.id AND l.stage NOT IN ('won', 'lost'))`,
		partnerID, userID)
	return err
}

// winLeads marks a client's open leads won when they sign up for one of
// the partner's tours (in the sign-up's transaction).
func winLeads(ctx context.Context, tx pgx.Tx, partnerID, userID, tourID string) error {
	_, err := tx.Exec(ctx, `UPDATE crm_leads l SET stage = 'won', tour_id = coalesce(l.tour_id, $3::uuid), updated_at = now()
		FROM crm_clients c WHERE c.id = l.client_id AND c.partner_id = $1 AND c.user_id = $2
		AND l.stage NOT IN ('won', 'lost')`, partnerID, userID, tourID)
	return err
}

// --- Tasks ---

const taskColumns = `k.id::text, k.title, to_char(k.due_on, 'YYYY-MM-DD'), k.done_at,
	c.id::text, c.user_id::text, c.name, t.id::text, t.title, to_char(t.start_date, 'YYYY-MM-DD')`

// Tasks lists the partner's tasks: open ones by due day (done ones too
// with all), of a client when clientID is set.
func (s *Store) Tasks(ctx context.Context, partnerID, clientID string, all bool) ([]models.CRMTask, error) {
	rows, err := s.db.Query(ctx, `SELECT `+taskColumns+` FROM crm_tasks k
		LEFT JOIN crm_clients c ON c.id = k.client_id LEFT JOIN tours t ON t.id = k.tour_id
		WHERE k.partner_id = $1 AND ($2 = '' OR k.client_id::text = $2) AND ($3 OR k.done_at IS NULL)
		ORDER BY k.done_at IS NOT NULL, k.due_on, k.created_at LIMIT 500`, partnerID, clientID, all)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMTask, error) {
		var k models.CRMTask
		var done *time.Time
		var clientID, userID, name, tourID, title, start *string
		err := r.Scan(&k.ID, &k.Title, &k.DueOn, &done, &clientID, &userID, &name, &tourID, &title, &start)
		k.Done, k.DoneAt = done != nil, jsontime.Ptr(done)
		if clientID != nil {
			k.Client = &models.CRMClientRef{ID: *clientID, UserID: userID, Name: name}
		}
		if tourID != nil {
			k.Tour = &models.CRMTourRef{ID: *tourID, Title: *title, StartDate: *start}
		}
		return k, err
	})
	if out == nil {
		out = []models.CRMTask{}
	}
	return out, err
}

// TaskInput is a task as the partner writes it; on update nil fields stay.
type TaskInput struct {
	Title    *string
	DueOn    *string
	ClientID *string
	TourID   *string
	Done     *bool
}

func (s *Store) CreateTask(ctx context.Context, partnerID string, in TaskInput) error {
	if err := s.ownedRefs(ctx, partnerID, in.ClientID, in.TourID); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `INSERT INTO crm_tasks (partner_id, title, due_on, client_id, tour_id)
		VALUES ($1, $2, $3::date, $4, $5)`, partnerID, in.Title, in.DueOn, in.ClientID, in.TourID)
	return err
}

func (s *Store) UpdateTask(ctx context.Context, partnerID, id string, in TaskInput, now time.Time) error {
	tag, err := s.db.Exec(ctx, `UPDATE crm_tasks SET title = coalesce($3, title), due_on = coalesce($4::date, due_on),
		done_at = CASE WHEN $5::boolean IS NULL THEN done_at WHEN $5 THEN coalesce(done_at, $6) ELSE NULL END
		WHERE partner_id = $1 AND id = $2`, partnerID, id, in.Title, in.DueOn, in.Done, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTask(ctx context.Context, partnerID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM crm_tasks WHERE partner_id = $1 AND id = $2`, partnerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Templates and broadcasts ---

func (s *Store) Templates(ctx context.Context, partnerID string) ([]models.CRMTemplate, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, title, body FROM crm_templates WHERE partner_id = $1
		ORDER BY lower(title) LIMIT 200`, partnerID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMTemplate, error) {
		var t models.CRMTemplate
		return t, r.Scan(&t.ID, &t.Title, &t.Body)
	})
	if out == nil {
		out = []models.CRMTemplate{}
	}
	return out, err
}

func (s *Store) SaveTemplate(ctx context.Context, partnerID, id, title, body string) error {
	if id == "" {
		_, err := s.db.Exec(ctx, `INSERT INTO crm_templates (partner_id, title, body) VALUES ($1, $2, $3)`, partnerID, title, body)
		return err
	}
	tag, err := s.db.Exec(ctx, `UPDATE crm_templates SET title = $3, body = $4 WHERE partner_id = $1 AND id = $2`,
		partnerID, id, title, body)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTemplate(ctx context.Context, partnerID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM crm_templates WHERE partner_id = $1 AND id = $2`, partnerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Audience is who a broadcast goes to: a tour's group (those with a place
// or on the waitlist), clients with a tag, or all the partner's clients.
type Audience struct {
	TourID string
	Tag    string
}

// Recipients lists the accounts in an audience and how many clients in it
// have none (added by hand: they can't get a chat message).
func (s *Store) Recipients(ctx context.Context, partnerID string, a Audience) ([]string, int, error) {
	var rows pgx.Rows
	var err error
	switch {
	case a.TourID != "":
		rows, err = s.db.Query(ctx, `SELECT p.user_id::text FROM tour_participants p JOIN tours t ON t.id = p.tour_id
			WHERE t.id = $2 AND t.user_id = $1 AND p.status <> 'cancelled' ORDER BY p.joined_at`, partnerID, a.TourID)
	default:
		rows, err = s.db.Query(ctx, `SELECT user_id::text FROM crm_clients
			WHERE partner_id = $1 AND ($2 = '' OR $2 = ANY (tags)) ORDER BY created_at`, partnerID, a.Tag)
	}
	if err != nil {
		return nil, 0, err
	}
	all, err := pgx.CollectRows(rows, pgx.RowTo[*string])
	var ids []string
	for _, id := range all {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	return ids, len(all) - len(ids), err
}

func (s *Store) SaveBroadcast(ctx context.Context, partnerID, audience, body string, sent, skipped int) error {
	_, err := s.db.Exec(ctx, `INSERT INTO crm_broadcasts (partner_id, audience, body, sent, skipped) VALUES ($1, $2, $3, $4, $5)`,
		partnerID, audience, body, sent, skipped)
	return err
}

func (s *Store) Broadcasts(ctx context.Context, partnerID string) ([]models.CRMBroadcast, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, audience, body, sent, skipped, created_at FROM crm_broadcasts
		WHERE partner_id = $1 ORDER BY created_at DESC LIMIT 50`, partnerID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CRMBroadcast, error) {
		var b models.CRMBroadcast
		var created time.Time
		err := r.Scan(&b.ID, &b.Audience, &b.Body, &b.Sent, &b.Skipped, &created)
		b.CreatedAt = jsontime.New(created)
		return b, err
	})
	if out == nil {
		out = []models.CRMBroadcast{}
	}
	return out, err
}

// --- Reports ---

// CRMReport sums up the partner's departures starting from–to (inclusive)
// and the leads opened then.
func (s *Store) CRMReport(ctx context.Context, partnerID, from, to string) (models.CRMReport, error) {
	r := models.CRMReport{From: from, To: to}
	rows, err := s.db.Query(ctx, `SELECT t.id::text, t.title, to_char(t.start_date, 'YYYY-MM-DD'), t.capacity,
		count(p.id) FILTER (WHERE p.`+activeSQL+`)::int,
		count(p.id) FILTER (WHERE p.`+activeSQL+` AND p.arrived_at IS NOT NULL)::int,
		count(p.id) FILTER (WHERE p.status = 'cancelled')::int,
		coalesce((SELECT sum(`+signedSQL+`) FROM tour_payments pay JOIN tour_participants q ON q.id = pay.participant_id
			WHERE q.tour_id = t.id), 0)::int,
		coalesce((SELECT sum(amount) FROM tour_expenses e WHERE e.tour_id = t.id), 0)::int
		FROM tours t LEFT JOIN tour_participants p ON p.tour_id = t.id
		WHERE t.user_id = $1 AND t.start_date BETWEEN $2::date AND $3::date
		GROUP BY t.id ORDER BY t.start_date DESC, t.title LIMIT 500`, partnerID, from, to)
	if err != nil {
		return r, err
	}
	var cancelled []int
	r.ByTour, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.CRMReportTour, error) {
		var t models.CRMReportTour
		var c int
		err := row.Scan(&t.ID, &t.Title, &t.StartDate, &t.Capacity, &t.Taken, &t.Arrived, &c, &t.Revenue, &t.Expenses)
		cancelled = append(cancelled, c)
		return t, err
	})
	if err != nil {
		return r, err
	}
	if r.ByTour == nil {
		r.ByTour = []models.CRMReportTour{}
	}
	for i, t := range r.ByTour {
		r.Tours++
		r.Participants += t.Taken
		r.Arrived += t.Arrived
		r.Cancelled += cancelled[i]
		r.Revenue += t.Revenue
		r.Expenses += t.Expenses
		if t.Capacity != nil {
			r.Places += *t.Capacity
			r.Taken += t.Taken
		}
		// No-shows only count where the organizer marked arrivals.
		if t.Arrived > 0 {
			r.NoShows += t.Taken - t.Arrived
		}
	}
	err = s.db.QueryRow(ctx, `WITH period AS (
			SELECT DISTINCT c.id FROM tour_participants p `+participantClientJoin+`
			WHERE t.user_id = $1 AND p.`+activeSQL+` AND t.start_date BETWEEN $2::date AND $3::date)
		SELECT count(*)::int, count(*) FILTER (WHERE EXISTS (
			SELECT 1 FROM tour_participants p `+participantClientJoin+`
			WHERE c.id = period.id AND t.user_id = $1 AND p.`+activeSQL+` AND t.start_date < $2::date))::int
		FROM period`, partnerID, from, to).Scan(&r.Clients, &r.RepeatClients)
	if err != nil {
		return r, err
	}
	rows, err = s.db.Query(ctx, `SELECT c.source, count(*)::int, count(*) FILTER (WHERE l.stage = 'won')::int,
		count(*) FILTER (WHERE l.stage = 'lost')::int
		FROM crm_leads l JOIN crm_clients c ON c.id = l.client_id
		WHERE l.partner_id = $1 AND l.created_at::date BETWEEN $2::date AND $3::date
		GROUP BY c.source ORDER BY count(*) DESC`, partnerID, from, to)
	if err != nil {
		return r, err
	}
	r.Leads, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.CRMLeadSource, error) {
		var l models.CRMLeadSource
		return l, row.Scan(&l.Source, &l.Total, &l.Won, &l.Lost)
	})
	if r.Leads == nil {
		r.Leads = []models.CRMLeadSource{}
	}
	return r, err
}
