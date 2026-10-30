// Package models defines the payments service's entities and their JSON.
package models

import "github.com/elvn5/sai/services/pkg/jsontime"

// Order statuses.
const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusFailed   = "failed"
	StatusExpired  = "expired"
	StatusRefunded = "refunded"
)

var Statuses = []string{StatusPending, StatusPaid, StatusFailed, StatusExpired, StatusRefunded}

// Person is the public part of a profile (from the users service).
type Person struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
}

type Order struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind"`
	ItemID        string         `json:"itemId"`
	ItemTitle     string         `json:"itemTitle"`
	ItemSlug      *string        `json:"itemSlug"`
	ItemLocale    *string        `json:"itemLocale"`
	ItemDate      *string        `json:"itemDate"`
	ItemImageURL  *string        `json:"itemImageUrl"`
	Quantity      int            `json:"quantity"`
	UnitPrice     int            `json:"unitPrice"`
	Amount        int            `json:"amount"`
	Fee           int            `json:"fee"`
	PayeeAmount   int            `json:"payeeAmount"`
	UserID        string         `json:"userId"`
	PayeeID       string         `json:"payeeId"`
	ContactName   string         `json:"contactName"`
	ContactPhone  string         `json:"contactPhone"`
	Comment       *string        `json:"comment"`
	Status        string         `json:"status"`
	Provider      string         `json:"-"`
	PaymentURL    *string        `json:"paymentUrl"`
	ProviderError *string        `json:"providerError,omitempty"`
	ExpiresAt     jsontime.Time  `json:"expiresAt"`
	PaidAt        *jsontime.Time `json:"paidAt"`
	RefundNote    *string        `json:"refundNote"`
	PayoutID      *string        `json:"payoutId"`
	CreatedAt     jsontime.Time  `json:"createdAt"`
	// Resolved for partner and admin views.
	Payer *Person `json:"payer,omitempty"`
	Payee *Person `json:"payee,omitempty"`
}

// Balance is what's owed to one partner: paid orders not yet paid out.
type Balance struct {
	Payee  Person `json:"payee"`
	Amount int    `json:"amount"`
	Orders int    `json:"orders"`
}

type Payout struct {
	ID        string        `json:"id"`
	Payee     Person        `json:"payee"`
	Amount    int           `json:"amount"`
	Orders    int           `json:"orders"`
	Note      *string       `json:"note"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// Finances is the partner cabinet's money summary for one partner.
type Finances struct {
	// Paid bookings not yet paid out, and how many.
	Owed       int `json:"owed"`
	OwedOrders int `json:"owedOrders"`
	// Everything paid out so far.
	PaidOut int `json:"paidOut"`
	// Paid bookings since the start of this month (Bishkek time).
	Month   MonthSummary `json:"month"`
	Payouts []Payout     `json:"payouts"`
}

type MonthSummary struct {
	Orders int `json:"orders"`
	Guests int `json:"guests"`
	// What customers paid, the platform's commission and the partner's share.
	Revenue int `json:"revenue"`
	Fee     int `json:"fee"`
	Earned  int `json:"earned"`
}

// Gateway statuses of a transaction (GoPay's own).
var TransactionStatuses = []string{"CREATED", "PENDING", "COMMITTED", "FAILED", "EXPIRED", "CANCELLED"}

// Transaction is the gateway's record of an order's payment (GoPay), for
// the admin panel's online payments.
type Transaction struct {
	OrderID         string         `json:"orderId"`
	Provider        string         `json:"provider"`
	PaymentID       *string        `json:"paymentId"`
	ProviderOrderID string         `json:"providerOrderId"`
	Status          string         `json:"status"`
	Amount          string         `json:"amount"`
	Testing         bool           `json:"testing"`
	CheckoutURL     *string        `json:"checkoutUrl"`
	GatewayCreated  *jsontime.Time `json:"gatewayCreatedAt"`
	ExpiresAt       *jsontime.Time `json:"expiresAt"`
	CommittedAt     *jsontime.Time `json:"committedAt"`
	BankOpDate      *string        `json:"bankOpDate"`
	LastError       *string        `json:"lastError"`
	CheckedAt       jsontime.Time  `json:"checkedAt"`
	CreatedAt       jsontime.Time  `json:"createdAt"`
	UpdatedAt       jsontime.Time  `json:"updatedAt"`
	Order           *Order         `json:"order,omitempty"`
	// Webhook deliveries about it, on the transaction's own card.
	Events []ProviderEvent `json:"events,omitempty"`
}

// ProviderEvent is a webhook delivery from the gateway.
type ProviderEvent struct {
	ID          int64          `json:"id"`
	Event       string         `json:"event"`
	PaymentID   *string        `json:"paymentId"`
	Testing     bool           `json:"testing"`
	Attempts    int            `json:"attempts"`
	ReceivedAt  jsontime.Time  `json:"receivedAt"`
	ProcessedAt *jsontime.Time `json:"processedAt"`
	Error       *string        `json:"error"`
}

// TransactionFilter narrows the admin's list; dates are Bishkek days.
type TransactionFilter struct {
	Status *string
	From   *string
	To     *string
	Limit  int
}
