package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// orNil: an empty filter is none.
func orNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- The driver's card ---

func (s *Store) DriverProfile(ctx context.Context, driverID string) (models.DriverProfile, error) {
	var p models.DriverProfile
	err := s.db.QueryRow(ctx, `SELECT user_id::text, name, phone, region, about, published
		FROM driver_profiles WHERE user_id = $1`, driverID).
		Scan(&p.UserID, &p.Name, &p.Phone, &p.Region, &p.About, &p.Published)
	return p, notFound(err)
}

func (s *Store) SaveDriverProfile(ctx context.Context, p models.DriverProfile) error {
	_, err := s.db.Exec(ctx, `INSERT INTO driver_profiles (user_id, name, phone, region, about, published)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET name = EXCLUDED.name, phone = EXCLUDED.phone, region = EXCLUDED.region,
			about = EXCLUDED.about, published = EXCLUDED.published, updated_at = now()`,
		p.UserID, p.Name, p.Phone, p.Region, p.About, p.Published)
	return err
}

// --- Vehicles ---

const vehicleColumns = `v.id::text, v.kind, v.model, v.plate, v.seats, v.year,
	to_char(v.insurance_until, 'YYYY-MM-DD'), to_char(v.inspection_until, 'YYYY-MM-DD'), v.note, v.active`

func scanVehicle(r pgx.Row) (models.DriverVehicle, error) {
	var v models.DriverVehicle
	err := r.Scan(&v.ID, &v.Kind, &v.Model, &v.Plate, &v.Seats, &v.Year, &v.InsuranceUntil, &v.InspectionUntil, &v.Note, &v.Active)
	return v, err
}

// Vehicles lists a driver's vehicles (the active ones only, when
// activeOnly), the active ones first.
func (s *Store) Vehicles(ctx context.Context, driverID string, activeOnly bool) ([]models.DriverVehicle, error) {
	rows, err := s.db.Query(ctx, `SELECT `+vehicleColumns+` FROM driver_vehicles v
		WHERE v.driver_id = $1 AND (v.active OR NOT $2) ORDER BY v.active DESC, v.created_at`, driverID, activeOnly)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.DriverVehicle, error) { return scanVehicle(r) })
	if out == nil {
		out = []models.DriverVehicle{}
	}
	return out, err
}

func (s *Store) Vehicle(ctx context.Context, driverID, id string) (models.DriverVehicle, error) {
	v, err := scanVehicle(s.db.QueryRow(ctx, `SELECT `+vehicleColumns+` FROM driver_vehicles v
		WHERE v.driver_id = $1 AND v.id = $2`, driverID, id))
	return v, notFound(err)
}

func (s *Store) AddVehicle(ctx context.Context, driverID string, v models.DriverVehicle) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO driver_vehicles (driver_id, kind, model, plate, seats, year,
		insurance_until, inspection_until, note, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8::date, $9, $10) RETURNING id::text`,
		driverID, v.Kind, v.Model, v.Plate, v.Seats, v.Year, v.InsuranceUntil, v.InspectionUntil, v.Note, v.Active).Scan(&id)
	return id, err
}

func (s *Store) UpdateVehicle(ctx context.Context, driverID string, v models.DriverVehicle) error {
	tag, err := s.db.Exec(ctx, `UPDATE driver_vehicles SET kind = $3, model = $4, plate = $5, seats = $6, year = $7,
		insurance_until = $8::date, inspection_until = $9::date, note = $10, active = $11
		WHERE driver_id = $1 AND id = $2`,
		driverID, v.ID, v.Kind, v.Model, v.Plate, v.Seats, v.Year, v.InsuranceUntil, v.InspectionUntil, v.Note, v.Active)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteVehicle removes a vehicle; its trips and expenses stay, without it.
func (s *Store) DeleteVehicle(ctx context.Context, driverID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM driver_vehicles WHERE driver_id = $1 AND id = $2`, driverID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// --- Trips ---

const tripPaidSQL = `coalesce((SELECT sum(pay.amount) FROM driver_trip_payments pay WHERE pay.trip_id = tr.id), 0)::int`

const tripColumns = `tr.id::text, tr.status, tr.starts_at, tr.ends_at, tr.route_from, tr.route_to, tr.passengers,
	tr.price, tr.note, tr.decline_reason, tr.created_at, ` + tripPaidSQL + `,
	coalesce((SELECT sum(e.amount) FROM driver_expenses e WHERE e.trip_id = tr.id), 0)::int,
	v.id::text, v.kind, v.model, v.plate, v.seats, v.year,
	to_char(v.insurance_until, 'YYYY-MM-DD'), to_char(v.inspection_until, 'YYYY-MM-DD'), v.note, v.active,
	c.id::text, c.user_id::text, c.name, c.phone,
	t.id::text, t.title, to_char(t.start_date, 'YYYY-MM-DD'), to_char(t.end_date, 'YYYY-MM-DD'), t.meeting_point, t.meeting_time,
	(SELECT count(*) FROM tour_participants q WHERE q.tour_id = t.id AND q.` + activeSQL + `)::int`

const tripJoins = `FROM driver_trips tr LEFT JOIN driver_vehicles v ON v.id = tr.vehicle_id
	LEFT JOIN crm_clients c ON c.id = tr.client_id LEFT JOIN tours t ON t.id = tr.tour_id`

func scanTrip(r pgx.Row) (models.DriverTrip, error) {
	var tr models.DriverTrip
	var starts, created time.Time
	var ends *time.Time
	var vID, vKind, vModel *string
	var vSeats *int
	var vActive *bool
	var v models.DriverVehicle
	var cID *string
	var cl models.DriverTripClient
	var tID, tTitle, tStart, tEnd *string
	var tour models.DriverTripTour
	var participants *int
	err := r.Scan(&tr.ID, &tr.Status, &starts, &ends, &tr.RouteFrom, &tr.RouteTo, &tr.Passengers,
		&tr.Price, &tr.Note, &tr.DeclineReason, &created, &tr.Paid, &tr.Expenses,
		&vID, &vKind, &vModel, &v.Plate, &vSeats, &v.Year, &v.InsuranceUntil, &v.InspectionUntil, &v.Note, &vActive,
		&cID, &cl.UserID, &cl.Name, &cl.Phone,
		&tID, &tTitle, &tStart, &tEnd, &tour.MeetingPoint, &tour.MeetingTime, &participants)
	if err != nil {
		return tr, err
	}
	tr.StartsAt, tr.CreatedAt = jsontime.New(starts), jsontime.New(created)
	tr.EndsAt = jsontime.Ptr(ends)
	if vID != nil {
		v.ID, v.Kind, v.Model, v.Seats, v.Active = *vID, *vKind, *vModel, *vSeats, *vActive
		tr.Vehicle = &v
	}
	if cID != nil {
		cl.ID = *cID
		tr.Client = &cl
	}
	if tID != nil {
		tour.ID, tour.Title, tour.StartDate, tour.EndDate = *tID, *tTitle, *tStart, *tEnd
		if participants != nil {
			tour.Participants = *participants
		}
		tr.Tour = &tour
	}
	if tr.Price != nil && (tr.Status == "accepted" || tr.Status == "done") {
		tr.Due = max(0, *tr.Price-tr.Paid)
	}
	return tr, nil
}

func collectTrips(rows pgx.Rows, err error) ([]models.DriverTrip, error) {
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.DriverTrip, error) { return scanTrip(r) })
	if out == nil {
		out = []models.DriverTrip{}
	}
	return out, err
}

// TripFilter narrows the driver's trips; empty fields don't filter.
type TripFilter struct {
	Status   string
	ClientID string
	From, To string // YYYY-MM-DD, in Bishkek
}

// Trips lists a driver's trips, soonest first (the latest first when
// looking at past ones only).
func (s *Store) Trips(ctx context.Context, driverID string, f TripFilter) ([]models.DriverTrip, error) {
	return collectTrips(s.db.Query(ctx, `SELECT `+tripColumns+` `+tripJoins+`
		WHERE tr.driver_id = $1 AND ($2 = '' OR tr.status = $2) AND ($3 = '' OR tr.client_id::text = $3)
		AND ($4::date IS NULL OR (tr.starts_at AT TIME ZONE 'Asia/Bishkek')::date >= $4::date)
		AND ($5::date IS NULL OR (tr.starts_at AT TIME ZONE 'Asia/Bishkek')::date <= $5::date)
		ORDER BY tr.starts_at LIMIT 500`, driverID, f.Status, f.ClientID, orNil(f.From), orNil(f.To)))
}

func (s *Store) Trip(ctx context.Context, driverID, id string) (models.DriverTrip, error) {
	tr, err := scanTrip(s.db.QueryRow(ctx, `SELECT `+tripColumns+` `+tripJoins+`
		WHERE tr.driver_id = $1 AND tr.id = $2`, driverID, id))
	return tr, notFound(err)
}

// TripInput is what the driver sets on a trip.
type TripInput struct {
	VehicleID  *string
	ClientID   *string
	TourID     *string
	Status     string
	StartsAt   time.Time
	EndsAt     *time.Time
	RouteFrom  string
	RouteTo    string
	Passengers *int
	Price      *int
	Note       *string
	Decline    *string
}

// AddTrip writes down a trip; the vehicle and the client must be the
// driver's own (else ErrNotFound).
func (s *Store) AddTrip(ctx context.Context, driverID string, in TripInput) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO driver_trips (driver_id, vehicle_id, client_id, tour_id, status, starts_at,
		ends_at, route_from, route_to, passengers, price, note)
		SELECT $1, $2::uuid, $3::uuid, $4::uuid, $5, $6::timestamptz, $7::timestamptz, $8, $9, $10::int, $11::int, $12
		WHERE ($2::uuid IS NULL OR EXISTS (SELECT 1 FROM driver_vehicles WHERE id = $2::uuid AND driver_id = $1))
		AND ($3::uuid IS NULL OR EXISTS (SELECT 1 FROM crm_clients WHERE id = $3::uuid AND partner_id = $1))
		RETURNING id::text`,
		driverID, in.VehicleID, in.ClientID, in.TourID, in.Status, in.StartsAt, in.EndsAt, in.RouteFrom, in.RouteTo,
		in.Passengers, in.Price, in.Note).Scan(&id)
	return id, notFound(err)
}

// UpdateTrip replaces what the driver set on one of their trips (the tour
// it's for stays).
func (s *Store) UpdateTrip(ctx context.Context, driverID, id string, in TripInput) error {
	tag, err := s.db.Exec(ctx, `UPDATE driver_trips SET vehicle_id = $3::uuid, client_id = $4::uuid, status = $5,
		starts_at = $6, ends_at = $7, route_from = $8, route_to = $9, passengers = $10, price = $11, note = $12,
		decline_reason = $13, updated_at = now()
		WHERE driver_id = $1 AND id = $2
		AND ($3::uuid IS NULL OR EXISTS (SELECT 1 FROM driver_vehicles WHERE id = $3::uuid AND driver_id = $1))
		AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM crm_clients WHERE id = $4::uuid AND partner_id = $1))`,
		driverID, id, in.VehicleID, in.ClientID, in.Status, in.StartsAt, in.EndsAt, in.RouteFrom, in.RouteTo,
		in.Passengers, in.Price, in.Note, in.Decline)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteTrip removes a trip the driver wrote down themself (organizers'
// requests are declined or cancelled instead).
func (s *Store) DeleteTrip(ctx context.Context, driverID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM driver_trips WHERE driver_id = $1 AND id = $2 AND tour_id IS NULL`, driverID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// --- Payments and expenses ---

func (s *Store) TripPayments(ctx context.Context, tripID string) ([]models.DriverPayment, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, amount, method, to_char(paid_on, 'YYYY-MM-DD'), note, created_at
		FROM driver_trip_payments WHERE trip_id = $1 ORDER BY paid_on, created_at`, tripID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.DriverPayment, error) {
		var p models.DriverPayment
		var created time.Time
		err := r.Scan(&p.ID, &p.Amount, &p.Method, &p.PaidOn, &p.Note, &created)
		p.CreatedAt = jsontime.New(created)
		return p, err
	})
	if out == nil {
		out = []models.DriverPayment{}
	}
	return out, err
}

func (s *Store) AddTripPayment(ctx context.Context, driverID, tripID string, p models.DriverPayment) error {
	tag, err := s.db.Exec(ctx, `INSERT INTO driver_trip_payments (trip_id, amount, method, paid_on, note)
		SELECT id, $3::int, $4, $5::date, $6 FROM driver_trips WHERE driver_id = $1 AND id = $2`,
		driverID, tripID, p.Amount, p.Method, p.PaidOn, p.Note)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteTripPayment removes a payment of one of the driver's trips and
// answers with the trip's id.
func (s *Store) DeleteTripPayment(ctx context.Context, driverID, id string) (string, error) {
	var tripID string
	err := s.db.QueryRow(ctx, `DELETE FROM driver_trip_payments p USING driver_trips tr
		WHERE p.id = $2 AND tr.id = p.trip_id AND tr.driver_id = $1 RETURNING tr.id::text`, driverID, id).Scan(&tripID)
	return tripID, notFound(err)
}

// ExpenseFilter narrows the driver's expenses; empty fields don't filter.
type ExpenseFilter struct {
	From, To string
	TripID   string
}

func (s *Store) DriverExpenses(ctx context.Context, driverID string, f ExpenseFilter) ([]models.DriverExpense, error) {
	rows, err := s.db.Query(ctx, `SELECT e.id::text, e.category, e.amount, to_char(e.spent_on, 'YYYY-MM-DD'), e.note,
		e.vehicle_id::text, v.model, e.trip_id::text
		FROM driver_expenses e LEFT JOIN driver_vehicles v ON v.id = e.vehicle_id
		WHERE e.driver_id = $1 AND ($2::date IS NULL OR e.spent_on >= $2::date) AND ($3::date IS NULL OR e.spent_on <= $3::date)
		AND ($4 = '' OR e.trip_id::text = $4)
		ORDER BY e.spent_on DESC, e.created_at DESC LIMIT 1000`, driverID, orNil(f.From), orNil(f.To), f.TripID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.DriverExpense, error) {
		var e models.DriverExpense
		return e, r.Scan(&e.ID, &e.Category, &e.Amount, &e.SpentOn, &e.Note, &e.VehicleID, &e.Vehicle, &e.TripID)
	})
	if out == nil {
		out = []models.DriverExpense{}
	}
	return out, err
}

// AddDriverExpense writes down an expense; its vehicle and trip must be
// the driver's own (else ErrNotFound).
func (s *Store) AddDriverExpense(ctx context.Context, driverID string, e models.DriverExpense) error {
	tag, err := s.db.Exec(ctx, `INSERT INTO driver_expenses (driver_id, vehicle_id, trip_id, category, amount, spent_on, note)
		SELECT $1, $2::uuid, $3::uuid, $4, $5::int, $6::date, $7
		WHERE ($2::uuid IS NULL OR EXISTS (SELECT 1 FROM driver_vehicles WHERE id = $2::uuid AND driver_id = $1))
		AND ($3::uuid IS NULL OR EXISTS (SELECT 1 FROM driver_trips WHERE id = $3::uuid AND driver_id = $1))`,
		driverID, e.VehicleID, e.TripID, e.Category, e.Amount, e.SpentOn, e.Note)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) DeleteDriverExpense(ctx context.Context, driverID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM driver_expenses WHERE driver_id = $1 AND id = $2`, driverID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// --- Dashboard and report ---

// Documents that run out within this many days show on the dashboard.
const documentWarnDays = 30

func (s *Store) DriverSummary(ctx context.Context, driverID string, now time.Time) (models.DriverSummary, error) {
	var out models.DriverSummary
	month := now.Format("2006-01") + "-01"
	err := s.db.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM driver_trips WHERE driver_id = $1 AND status = 'requested')::int,
		coalesce((SELECT sum(p.amount) FROM driver_trip_payments p JOIN driver_trips tr ON tr.id = p.trip_id
			WHERE tr.driver_id = $1 AND p.paid_on >= $2::date), 0)::int,
		coalesce((SELECT sum(amount) FROM driver_expenses WHERE driver_id = $1 AND spent_on >= $2::date), 0)::int,
		coalesce((SELECT sum(greatest(tr.price - `+tripPaidSQL+`, 0)) FROM driver_trips tr
			WHERE tr.driver_id = $1 AND tr.status IN ('accepted', 'done') AND tr.price IS NOT NULL), 0)::int,
		(SELECT count(*) FROM driver_vehicles WHERE driver_id = $1 AND active)::int,
		EXISTS (SELECT 1 FROM driver_profiles WHERE user_id = $1),
		coalesce((SELECT published FROM driver_profiles WHERE user_id = $1), false)`, driverID, month).
		Scan(&out.Requests, &out.MonthIncome, &out.MonthExpenses, &out.Due, &out.Vehicles, &out.HasProfile, &out.Published)
	if err != nil {
		return out, err
	}
	if out.Upcoming, err = collectTrips(s.db.Query(ctx, `SELECT `+tripColumns+` `+tripJoins+`
		WHERE tr.driver_id = $1 AND tr.status IN ('requested', 'accepted') AND coalesce(tr.ends_at, tr.starts_at) >= $2
		ORDER BY tr.starts_at LIMIT 5`, driverID, now)); err != nil {
		return out, err
	}
	rows, err := s.db.Query(ctx, `SELECT id::text, model, kind, to_char(until, 'YYYY-MM-DD') FROM (
			SELECT id, model, 'insurance' AS kind, insurance_until AS until FROM driver_vehicles WHERE driver_id = $1 AND active
			UNION ALL
			SELECT id, model, 'inspection', inspection_until FROM driver_vehicles WHERE driver_id = $1 AND active) d
		WHERE until IS NOT NULL AND until <= $2::date + $3::int ORDER BY until`, driverID, now.Format(time.DateOnly), documentWarnDays)
	if err != nil {
		return out, err
	}
	out.Documents, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.DriverDocument, error) {
		var d models.DriverDocument
		return d, r.Scan(&d.VehicleID, &d.Vehicle, &d.Kind, &d.Until)
	})
	if out.Documents == nil {
		out.Documents = []models.DriverDocument{}
	}
	return out, err
}

// DriverReport sums up the trips that started and the money that moved
// between from and to (YYYY-MM-DD).
func (s *Store) DriverReport(ctx context.Context, driverID, from, to string) (models.DriverReport, error) {
	r := models.DriverReport{From: from, To: to, ByCategory: map[string]int{}}
	const tripDay = `(tr.starts_at AT TIME ZONE 'Asia/Bishkek')::date`
	err := s.db.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE status = 'done')::int,
		count(*) FILTER (WHERE status = 'cancelled')::int,
		coalesce(sum(passengers) FILTER (WHERE status = 'done'), 0)::int
		FROM driver_trips tr WHERE driver_id = $1 AND `+tripDay+` BETWEEN $2::date AND $3::date`, driverID, from, to).
		Scan(&r.Trips, &r.Cancelled, &r.Passengers)
	if err != nil {
		return r, err
	}
	rows, err := s.db.Query(ctx, `WITH months AS (
			SELECT to_char(m, 'YYYY-MM') AS month FROM generate_series(date_trunc('month', $2::date), $3::date, '1 month') m),
		trips AS (SELECT to_char(`+tripDay+`, 'YYYY-MM') AS month, count(*)::int AS n FROM driver_trips tr
			WHERE driver_id = $1 AND status = 'done' AND `+tripDay+` BETWEEN $2::date AND $3::date GROUP BY 1),
		income AS (SELECT to_char(p.paid_on, 'YYYY-MM') AS month, sum(p.amount)::int AS n FROM driver_trip_payments p
			JOIN driver_trips tr ON tr.id = p.trip_id WHERE tr.driver_id = $1 AND p.paid_on BETWEEN $2::date AND $3::date GROUP BY 1),
		spent AS (SELECT to_char(spent_on, 'YYYY-MM') AS month, sum(amount)::int AS n FROM driver_expenses
			WHERE driver_id = $1 AND spent_on BETWEEN $2::date AND $3::date GROUP BY 1)
		SELECT m.month, coalesce(t.n, 0), coalesce(i.n, 0), coalesce(s.n, 0) FROM months m
		LEFT JOIN trips t USING (month) LEFT JOIN income i USING (month) LEFT JOIN spent s USING (month)
		ORDER BY m.month`, driverID, from, to)
	if err != nil {
		return r, err
	}
	r.ByMonth, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DriverReportMonth, error) {
		var m models.DriverReportMonth
		return m, row.Scan(&m.Month, &m.Trips, &m.Income, &m.Expenses)
	})
	if err != nil {
		return r, err
	}
	for _, m := range r.ByMonth {
		r.Income += m.Income
		r.Expenses += m.Expenses
	}
	r.Profit = r.Income - r.Expenses
	rows, err = s.db.Query(ctx, `SELECT category, sum(amount)::int FROM driver_expenses
		WHERE driver_id = $1 AND spent_on BETWEEN $2::date AND $3::date GROUP BY category`, driverID, from, to)
	if err != nil {
		return r, err
	}
	var category string
	var amount int
	if _, err := pgx.ForEachRow(rows, []any{&category, &amount}, func() error {
		r.ByCategory[category] = amount
		return nil
	}); err != nil {
		return r, err
	}
	// Per vehicle; trips and expenses without one are a row of their own.
	rows, err = s.db.Query(ctx, `WITH trips AS (
			SELECT tr.vehicle_id, count(*) FILTER (WHERE tr.status = 'done')::int AS n,
				coalesce(sum((SELECT sum(p.amount) FROM driver_trip_payments p WHERE p.trip_id = tr.id
					AND p.paid_on BETWEEN $2::date AND $3::date)), 0)::int AS income
			FROM driver_trips tr WHERE tr.driver_id = $1 AND (`+tripDay+` BETWEEN $2::date AND $3::date
				OR EXISTS (SELECT 1 FROM driver_trip_payments p WHERE p.trip_id = tr.id AND p.paid_on BETWEEN $2::date AND $3::date))
			GROUP BY tr.vehicle_id),
		spent AS (SELECT vehicle_id, sum(amount)::int AS n FROM driver_expenses
			WHERE driver_id = $1 AND spent_on BETWEEN $2::date AND $3::date GROUP BY vehicle_id)
		SELECT v.id::text, v.model, coalesce(t.n, 0), coalesce(t.income, 0), coalesce(s.n, 0)
		FROM trips t FULL JOIN spent s ON coalesce(s.vehicle_id, '00000000-0000-0000-0000-000000000000') = coalesce(t.vehicle_id, '00000000-0000-0000-0000-000000000000')
		LEFT JOIN driver_vehicles v ON v.id = coalesce(t.vehicle_id, s.vehicle_id)
		ORDER BY coalesce(t.income, 0) DESC`, driverID, from, to)
	if err != nil {
		return r, err
	}
	r.ByVehicle, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DriverReportCar, error) {
		var c models.DriverReportCar
		return c, row.Scan(&c.ID, &c.Model, &c.Trips, &c.Income, &c.Expenses)
	})
	if r.ByVehicle == nil {
		r.ByVehicle = []models.DriverReportCar{}
	}
	return r, err
}

// --- The organizers' side ---

// Drivers is the organizers' catalog: published drivers with their active
// vehicles (seats at least minSeats), busy when they have a trip on day
// (YYYY-MM-DD, optional).
func (s *Store) Drivers(ctx context.Context, day string, minSeats int) ([]models.CatalogDriver, error) {
	rows, err := s.db.Query(ctx, `SELECT d.user_id::text, d.name, d.phone, d.region, d.about, d.published,
		(SELECT count(*) FROM driver_trips tr WHERE tr.driver_id = d.user_id AND tr.status = 'done')::int,
		$1::date IS NOT NULL AND EXISTS (SELECT 1 FROM driver_trips tr WHERE tr.driver_id = d.user_id AND tr.status = 'accepted'
			AND $1::date BETWEEN (tr.starts_at AT TIME ZONE 'Asia/Bishkek')::date
			AND (coalesce(tr.ends_at, tr.starts_at) AT TIME ZONE 'Asia/Bishkek')::date)
		FROM driver_profiles d
		WHERE d.published AND EXISTS (SELECT 1 FROM driver_vehicles v WHERE v.driver_id = d.user_id AND v.active AND v.seats >= $2)
		ORDER BY d.name LIMIT 200`, orNil(day), minSeats)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.CatalogDriver, error) {
		var d models.CatalogDriver
		return d, r.Scan(&d.UserID, &d.Name, &d.Phone, &d.Region, &d.About, &d.Published, &d.Trips, &d.Busy)
	})
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Vehicles, err = s.Vehicles(ctx, out[i].UserID, true); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []models.CatalogDriver{}
	}
	return out, nil
}

// RequestTransport asks a published driver for a trip for one of the
// organizer's tours. The organizer becomes the driver's client.
func (s *Store) RequestTransport(ctx context.Context, organizerID, tourID, driverID string, in TripInput) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var published bool
	if err := tx.QueryRow(ctx, `SELECT published FROM driver_profiles WHERE user_id = $1`, driverID).Scan(&published); err != nil {
		return "", notFound(err)
	}
	if !published {
		return "", ErrNotFound
	}
	var clientID string
	if err := tx.QueryRow(ctx, `INSERT INTO crm_clients (partner_id, user_id, source) VALUES ($1, $2, 'ermekten')
		ON CONFLICT (partner_id, user_id) DO UPDATE SET partner_id = EXCLUDED.partner_id
		RETURNING id::text`, driverID, organizerID).Scan(&clientID); err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO driver_trips (driver_id, vehicle_id, client_id, tour_id, status, starts_at,
		route_from, route_to, passengers, price, note)
		SELECT $1, $2::uuid, $3::uuid, $4::uuid, 'requested', $5::timestamptz, $6, $7, $8::int, $9::int, $10
		WHERE $2::uuid IS NULL OR EXISTS (SELECT 1 FROM driver_vehicles WHERE id = $2::uuid AND driver_id = $1 AND active)
		RETURNING id::text`,
		driverID, in.VehicleID, clientID, tourID, in.StartsAt, in.RouteFrom, in.RouteTo, in.Passengers, in.Price, in.Note).Scan(&id)
	if err != nil {
		return "", notFound(err)
	}
	return id, tx.Commit(ctx)
}

const transportColumns = `tr.id::text, tr.status, tr.starts_at, tr.route_from, tr.route_to, tr.passengers, tr.price,
	tr.note, tr.decline_reason, d.user_id::text, d.name, d.phone, d.region, d.about, d.published,
	v.id::text, v.kind, v.model, v.plate, v.seats, v.year,
	to_char(v.insurance_until, 'YYYY-MM-DD'), to_char(v.inspection_until, 'YYYY-MM-DD'), v.note, v.active`

// TransportRequests lists the trips asked for a tour, the latest first.
func (s *Store) TransportRequests(ctx context.Context, tourID string) ([]models.TransportRequest, error) {
	rows, err := s.db.Query(ctx, `SELECT `+transportColumns+` FROM driver_trips tr
		JOIN driver_profiles d ON d.user_id = tr.driver_id LEFT JOIN driver_vehicles v ON v.id = tr.vehicle_id
		WHERE tr.tour_id = $1 ORDER BY tr.created_at DESC`, tourID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.TransportRequest, error) {
		var t models.TransportRequest
		var starts time.Time
		var vID, vKind, vModel *string
		var vSeats *int
		var vActive *bool
		var v models.DriverVehicle
		err := r.Scan(&t.ID, &t.Status, &starts, &t.RouteFrom, &t.RouteTo, &t.Passengers, &t.Price, &t.Note, &t.DeclineReason,
			&t.Driver.UserID, &t.Driver.Name, &t.Driver.Phone, &t.Driver.Region, &t.Driver.About, &t.Driver.Published,
			&vID, &vKind, &vModel, &v.Plate, &vSeats, &v.Year, &v.InsuranceUntil, &v.InspectionUntil, &v.Note, &vActive)
		t.StartsAt = jsontime.New(starts)
		if vID != nil {
			v.ID, v.Kind, v.Model, v.Seats, v.Active = *vID, *vKind, *vModel, *vSeats, *vActive
			t.Vehicle = &v
		}
		return t, err
	})
	if out == nil {
		out = []models.TransportRequest{}
	}
	return out, err
}

// CancelTransport: the organizer calls off a trip asked for one of their
// tours. Answers with the tour's id and the driver's.
func (s *Store) CancelTransport(ctx context.Context, organizerID, id string) (tourID, driverID string, err error) {
	err = s.db.QueryRow(ctx, `UPDATE driver_trips tr SET status = 'cancelled', updated_at = now() FROM tours t
		WHERE tr.id = $2 AND t.id = tr.tour_id AND t.user_id = $1 AND tr.status IN ('requested', 'accepted')
		RETURNING t.id::text, tr.driver_id::text`, organizerID, id).Scan(&tourID, &driverID)
	return tourID, driverID, notFound(err)
}

// IsPublishedDriver: id is a driver organizers can find (and write to).
func (s *Store) IsPublishedDriver(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM driver_profiles WHERE user_id = $1 AND published)`, id).Scan(&ok)
	return ok, err
}

// AddAccountClient makes an account one of the partner's clients (an
// organizer who writes to a driver).
func (s *Store) AddAccountClient(ctx context.Context, partnerID, userID string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO crm_clients (partner_id, user_id, source) VALUES ($1, $2, 'ermekten')
		ON CONFLICT (partner_id, user_id) DO NOTHING`, partnerID, userID)
	return err
}
