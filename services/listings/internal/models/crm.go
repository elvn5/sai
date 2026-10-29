package models

import (
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

// CRMClient is one of a partner's clients: an account that signed up for
// their tours (Account) or someone they added by hand. Name, phone and
// Telegram are what the partner typed in.
type CRMClient struct {
	ID        string              `json:"id"`
	UserID    *string             `json:"userId"`
	Account   *usersclient.Author `json:"account"`
	Name      *string             `json:"name"`
	Phone     *string             `json:"phone"`
	Telegram  *string             `json:"telegram"`
	Note      *string             `json:"note"`
	Tags      []string            `json:"tags"`
	CreatedAt jsontime.Time       `json:"createdAt"`
	Stats     CRMClientStats      `json:"stats"`
	History   []CRMClientTour     `json:"history,omitempty"`
	Chat      *string             `json:"chatId,omitempty"`
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
}

// CRMPayment is money the partner received from a participant.
type CRMPayment struct {
	ID        string        `json:"id"`
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
