package models

import (
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

// DriverProfile is a driver's card in the organizers' catalog.
type DriverProfile struct {
	UserID    string  `json:"userId"`
	Name      string  `json:"name"`
	Phone     *string `json:"phone"`
	Region    *string `json:"region"`
	About     *string `json:"about"`
	Published bool    `json:"published"`
}

// DriverVehicle is one of a driver's vehicles. Its documents run out on
// InsuranceUntil and InspectionUntil (YYYY-MM-DD).
type DriverVehicle struct {
	ID              string  `json:"id"`
	Kind            string  `json:"kind"`
	Model           string  `json:"model"`
	Plate           *string `json:"plate"`
	Seats           int     `json:"seats"`
	Year            *int    `json:"year"`
	InsuranceUntil  *string `json:"insuranceUntil"`
	InspectionUntil *string `json:"inspectionUntil"`
	Note            *string `json:"note"`
	Active          bool    `json:"active"`
}

// DriverTripTour is the tour an organizer asked transport for, as the
// driver sees it: where and when to pick up the group and how big it is
// (no participants' names).
type DriverTripTour struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	StartDate    string  `json:"startDate"`
	EndDate      string  `json:"endDate"`
	MeetingPoint *string `json:"meetingPoint"`
	MeetingTime  *string `json:"meetingTime"`
	Participants int     `json:"participants"`
}

// DriverTripClient is who ordered a trip: one of the driver's clients.
type DriverTripClient struct {
	ID      string              `json:"id"`
	UserID  *string             `json:"userId"`
	Account *usersclient.Author `json:"account"`
	Name    *string             `json:"name"`
	Phone   *string             `json:"phone"`
}

// DriverTrip is a trip in the driver's CRM. Due is the price less what was
// paid (0 without a price).
type DriverTrip struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	StartsAt      jsontime.Time     `json:"startsAt"`
	EndsAt        *jsontime.Time    `json:"endsAt"`
	RouteFrom     string            `json:"routeFrom"`
	RouteTo       string            `json:"routeTo"`
	Passengers    *int              `json:"passengers"`
	Price         *int              `json:"price"`
	Paid          int               `json:"paid"`
	Due           int               `json:"due"`
	Note          *string           `json:"note"`
	DeclineReason *string           `json:"declineReason"`
	Vehicle       *DriverVehicle    `json:"vehicle"`
	Client        *DriverTripClient `json:"client"`
	Tour          *DriverTripTour   `json:"tour"`
	Expenses      int               `json:"expenses"`
	CreatedAt     jsontime.Time     `json:"createdAt"`
	// On the trip's own page.
	Payments    []DriverPayment `json:"payments,omitempty"`
	ExpenseList []DriverExpense `json:"expenseList,omitempty"`
	ChatID      *string         `json:"chatId,omitempty"`
}

type DriverPayment struct {
	ID        string        `json:"id"`
	Amount    int           `json:"amount"`
	Method    string        `json:"method"`
	PaidOn    string        `json:"paidOn"`
	Note      *string       `json:"note"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

type DriverExpense struct {
	ID        string  `json:"id"`
	Category  string  `json:"category"`
	Amount    int     `json:"amount"`
	SpentOn   string  `json:"spentOn"`
	Note      *string `json:"note"`
	VehicleID *string `json:"vehicleId"`
	Vehicle   *string `json:"vehicle"`
	TripID    *string `json:"tripId"`
}

// DriverDocument is a vehicle document that runs out soon (or already did).
type DriverDocument struct {
	VehicleID string `json:"vehicleId"`
	Vehicle   string `json:"vehicle"`
	// "insurance" or "inspection".
	Kind  string `json:"kind"`
	Until string `json:"until"`
}

// DriverSummary is the driver's dashboard.
type DriverSummary struct {
	Requests      int              `json:"requests"`
	Upcoming      []DriverTrip     `json:"upcoming"`
	MonthIncome   int              `json:"monthIncome"`
	MonthExpenses int              `json:"monthExpenses"`
	Due           int              `json:"due"`
	Documents     []DriverDocument `json:"documents"`
	HasProfile    bool             `json:"hasProfile"`
	Published     bool             `json:"published"`
	Vehicles      int              `json:"vehicles"`
}

// DriverReportMonth is one month of a driver's report.
type DriverReportMonth struct {
	Month    string `json:"month"` // YYYY-MM
	Trips    int    `json:"trips"`
	Income   int    `json:"income"`
	Expenses int    `json:"expenses"`
}

// DriverReport sums up a period: trips done, money in and out.
type DriverReport struct {
	From       string              `json:"from"`
	To         string              `json:"to"`
	Trips      int                 `json:"trips"`
	Cancelled  int                 `json:"cancelled"`
	Passengers int                 `json:"passengers"`
	Income     int                 `json:"income"`
	Expenses   int                 `json:"expenses"`
	Profit     int                 `json:"profit"`
	ByMonth    []DriverReportMonth `json:"byMonth"`
	ByCategory map[string]int      `json:"byCategory"`
	ByVehicle  []DriverReportCar   `json:"byVehicle"`
}

// DriverReportCar is what one vehicle earned and cost in the period.
type DriverReportCar struct {
	ID       *string `json:"id"`
	Model    *string `json:"model"`
	Trips    int     `json:"trips"`
	Income   int     `json:"income"`
	Expenses int     `json:"expenses"`
}

// CatalogDriver is a driver in the organizers' catalog. Busy: they have a
// trip on the day asked about.
type CatalogDriver struct {
	DriverProfile
	Account  *usersclient.Author `json:"account"`
	Vehicles []DriverVehicle     `json:"vehicles"`
	Trips    int                 `json:"trips"`
	Busy     bool                `json:"busy"`
}

// TransportRequest is a trip an organizer asked a driver for, on the
// tour's group page.
type TransportRequest struct {
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	StartsAt      jsontime.Time  `json:"startsAt"`
	RouteFrom     string         `json:"routeFrom"`
	RouteTo       string         `json:"routeTo"`
	Passengers    *int           `json:"passengers"`
	Price         *int           `json:"price"`
	Note          *string        `json:"note"`
	DeclineReason *string        `json:"declineReason"`
	Driver        DriverProfile  `json:"driver"`
	Vehicle       *DriverVehicle `json:"vehicle"`
}
