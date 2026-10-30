package models

import (
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

// CRMClient is one of a partner's clients: an account that signed up for
// their tours (Account) or someone they added by hand. Name, phone and
// Telegram are what the partner typed in.
type CRMClient struct {
	ID       string              `json:"id"`
	UserID   *string             `json:"userId"`
	Account  *usersclient.Author `json:"account"`
	Name     *string             `json:"name"`
	Phone    *string             `json:"phone"`
	Telegram *string             `json:"telegram"`
	Note     *string             `json:"note"`
	Tags     []string            `json:"tags"`
	// Where they came from (ermekten, instagram, whatsapp, ...); nil if
	// not known.
	Source    *string        `json:"source"`
	CreatedAt jsontime.Time  `json:"createdAt"`
	Stats     CRMClientStats `json:"stats"`
	// On the card: their leads and open tasks.
	Leads   []CRMLead       `json:"leads,omitempty"`
	Tasks   []CRMTask       `json:"tasks,omitempty"`
	History []CRMClientTour `json:"history,omitempty"`
	Chat    *string         `json:"chatId,omitempty"`
}

// CRMClientStats sums up a client's places on the partner's tours
// (cancelled and waitlisted ones aside).
type CRMClientStats struct {
	Tours    int     `json:"tours"`
	LastTour *string `json:"lastTour"`
	Due      int     `json:"due"`
	Paid     int     `json:"paid"`
}

// CRMClientTour is one of a client's places, on the client's card.
type CRMClientTour struct {
	ParticipantID string `json:"participantId"`
	TourID        string `json:"tourId"`
	Title         string `json:"title"`
	Slug          string `json:"slug"`
	Locale        string `json:"locale"`
	StartDate     string `json:"startDate"`
	Status        string `json:"status"`
	Due           int    `json:"due"`
	Paid          int    `json:"paid"`
	Arrived       bool   `json:"arrived"`
}

// CRMParticipant is one row of a tour's group in the CRM.
type CRMParticipant struct {
	ID        string              `json:"id"`
	ClientID  string              `json:"clientId"`
	UserID    *string             `json:"userId"`
	Account   *usersclient.Author `json:"account"`
	Name      *string             `json:"name"`
	Phone     *string             `json:"phone"`
	Telegram  *string             `json:"telegram"`
	Status    string              `json:"status"`
	JoinedAt  jsontime.Time       `json:"joinedAt"`
	ArrivedAt *jsontime.Time      `json:"arrivedAt"`
	// What they owe: AmountDue when the partner set it, else the price.
	AmountDue *int         `json:"amountDue"`
	Due       int          `json:"due"`
	Paid      int          `json:"paid"`
	Payments  []CRMPayment `json:"payments"`
	// Their form and when it was filled, and when they accepted the rules
	// (which version).
	Form          *ParticipantForm `json:"form"`
	FormAt        *jsontime.Time   `json:"formAt"`
	WaiverAt      *jsontime.Time   `json:"waiverAt"`
	WaiverVersion *string          `json:"waiverVersion"`
}

// CRMGroup is a tour's group: everyone on it with the totals of those who
// have a place.
type CRMGroup struct {
	Tour         Tour             `json:"tour"`
	Participants []CRMParticipant `json:"participants"`
	Totals       CRMTotals        `json:"totals"`
	Expenses     []CRMExpense     `json:"expenses"`
	Staff        []CRMStaff       `json:"staff"`
}

type CRMTotals struct {
	Active   int `json:"active"`
	Waitlist int `json:"waitlist"`
	Arrived  int `json:"arrived"`
	Due      int `json:"due"`
	Paid     int `json:"paid"`
	// Of those with a place: how many filled the form and accepted the
	// rules.
	Forms   int `json:"forms"`
	Waivers int `json:"waivers"`
	// What the tour cost, and what's left of what was paid.
	Expenses int `json:"expenses"`
	Margin   int `json:"margin"`
}

// CRMPayment is money the partner received from a participant.
type CRMPayment struct {
	ID string `json:"id"`
	// "payment", or "refund" (counts against what they paid).
	Kind      string        `json:"kind"`
	Amount    int           `json:"amount"`
	Method    string        `json:"method"`
	PaidOn    string        `json:"paidOn"`
	Note      *string       `json:"note"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// CRMSummary is the partner's dashboard: money and the next tours.
type CRMSummary struct {
	Clients       int `json:"clients"`
	PaidThisMonth int `json:"paidThisMonth"`
	// Owed by those with a place on tours that haven't ended a month ago.
	Outstanding int               `json:"outstanding"`
	Upcoming    []CRMUpcomingTour `json:"upcoming"`
}

type CRMUpcomingTour struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Locale    string `json:"locale"`
	StartDate string `json:"startDate"`
	Capacity  *int   `json:"capacity"`
	Taken     int    `json:"taken"`
	Waitlist  int    `json:"waitlist"`
	Due       int    `json:"due"`
	Paid      int    `json:"paid"`
}

// ChatMessage is one message in a chat between a partner and a traveler.
type ChatMessage struct {
	ID        string        `json:"id"`
	SenderID  string        `json:"senderId"`
	Body      string        `json:"body"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// ChatSummary is one of the visitor's chats in their list: "partner" when
// they run tours and write to a client, "client" when they write to an
// organizer.
type ChatSummary struct {
	ID          string             `json:"id"`
	PartnerID   string             `json:"partnerId"`
	UserID      string             `json:"userId"`
	Role        string             `json:"role"`
	Peer        usersclient.Author `json:"peer"`
	LastMessage ChatMessage        `json:"lastMessage"`
	Unread      int                `json:"unread"`
}

// ChatThread is a chat with its latest messages (or those before a time),
// and when the other side last read it.
type ChatThread struct {
	ChatSummary
	PeerReadAt *jsontime.Time `json:"peerReadAt"`
	Messages   []ChatMessage  `json:"messages"`
}

// CRMExpense is what a departure cost the partner.
type CRMExpense struct {
	ID        string        `json:"id"`
	Category  string        `json:"category"`
	Amount    int           `json:"amount"`
	Note      *string       `json:"note"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// CRMStaff is someone who works on a departure: a guide, a driver...
type CRMStaff struct {
	ID    string  `json:"id"`
	Role  string  `json:"role"`
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
	Note  *string `json:"note"`
}

// CRMClientRef names a client on a lead or a task.
type CRMClientRef struct {
	ID      string              `json:"id"`
	UserID  *string             `json:"userId"`
	Name    *string             `json:"name"`
	Account *usersclient.Author `json:"account"`
}

// CRMTourRef names a tour on a lead or a task.
type CRMTourRef struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
}

// CRMLead is an inquiry on its way to a booking: new, working, offer,
// then won or lost (with a reason).
type CRMLead struct {
	ID         string        `json:"id"`
	Client     CRMClientRef  `json:"client"`
	Source     *string       `json:"source"`
	Tour       *CRMTourRef   `json:"tour"`
	Stage      string        `json:"stage"`
	LostReason *string       `json:"lostReason"`
	Note       *string       `json:"note"`
	CreatedAt  jsontime.Time `json:"createdAt"`
	UpdatedAt  jsontime.Time `json:"updatedAt"`
}

// CRMTask is a to-do of the partner's, due on a day.
type CRMTask struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	DueOn  string         `json:"dueOn"`
	Done   bool           `json:"done"`
	Client *CRMClientRef  `json:"client"`
	Tour   *CRMTourRef    `json:"tour"`
	DoneAt *jsontime.Time `json:"doneAt"`
}

// CRMTemplate is a text the partner sends again and again.
type CRMTemplate struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// CRMBroadcast is a message sent to many clients at once: how many got it
// in their chat and how many were skipped (no account).
type CRMBroadcast struct {
	ID        string        `json:"id"`
	Audience  string        `json:"audience"`
	Body      string        `json:"body"`
	Sent      int           `json:"sent"`
	Skipped   int           `json:"skipped"`
	CreatedAt jsontime.Time `json:"createdAt"`
}

// CRMReport sums up the partner's departures that started in a period
// (from–to, YYYY-MM-DD), their money and their leads.
type CRMReport struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Tours int    `json:"tours"`
	// Places on tours with a capacity, and how many of them were taken.
	Places int `json:"places"`
	Taken  int `json:"taken"`
	// Participants with a place; those marked as arrived and not (on
	// tours where the organizer marked anyone); cancellations.
	Participants int `json:"participants"`
	Arrived      int `json:"arrived"`
	NoShows      int `json:"noShows"`
	Cancelled    int `json:"cancelled"`
	// Clients with a place in the period, and those of them who had gone
	// on one of the partner's tours before.
	Clients       int `json:"clients"`
	RepeatClients int `json:"repeatClients"`
	Revenue       int `json:"revenue"`
	Expenses      int `json:"expenses"`
	// Leads opened in the period, by where the client came from.
	Leads []CRMLeadSource `json:"leads"`
	// The period's tours, latest first.
	ByTour []CRMReportTour `json:"byTour"`
}

type CRMLeadSource struct {
	Source *string `json:"source"`
	Total  int     `json:"total"`
	Won    int     `json:"won"`
	Lost   int     `json:"lost"`
}

type CRMReportTour struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	Capacity  *int   `json:"capacity"`
	Taken     int    `json:"taken"`
	Arrived   int    `json:"arrived"`
	Revenue   int    `json:"revenue"`
	Expenses  int    `json:"expenses"`
}

// AdminTour is a tour in the panel's list: where it stands (Phase) and
// its group's counts.
type AdminTour struct {
	Tour Tour `json:"tour"`
	// "upcoming", "current" (the group met and the tour isn't over) or
	// "past".
	Phase  string          `json:"phase"`
	Counts AdminTourCounts `json:"counts"`
}

// AdminTourCounts are a tour's participants by status; of those with a
// place: how many came, filled the form and accepted the rules, what they
// owe and paid.
type AdminTourCounts struct {
	Active    int `json:"active"`
	Waitlist  int `json:"waitlist"`
	Cancelled int `json:"cancelled"`
	Arrived   int `json:"arrived"`
	Forms     int `json:"forms"`
	Waivers   int `json:"waivers"`
	Due       int `json:"due"`
	Paid      int `json:"paid"`
}

// AdminGroup is one tour in the panel: its group as the organizer sees
// it, with the organizer and where the tour stands.
type AdminGroup struct {
	CRMGroup
	Phase     string              `json:"phase"`
	Organizer *usersclient.Author `json:"organizer"`
}
