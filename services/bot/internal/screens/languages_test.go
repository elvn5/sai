package screens_test

import (
	"testing"
	"unicode"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/drafts"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// cyrillic finds the first Cyrillic letter in s.
func cyrillic(s string) (rune, bool) {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return r, true
		}
	}
	return 0, false
}

// Every screen drawn in English from Latin data has no Russian left in it:
// a text the code forgot to translate shows up here.
func TestScreensInEnglish(t *testing.T) {
	en := i18n.EN
	now := at(10, 20, 12, 0)
	returns := at(10, 24, 19, 0)
	joined := at(10, 18, 9, 0)
	cost, seats, days := 500, 12, 3
	text, place, phone := "Pace: easy", "Karakol", "+996 555 000 000"
	ann, bob := str("Ann"), str("bob")
	arrived := at(10, 24, 5, 50)
	going := api.Outing{
		ID: "o1", Title: "Uchitel Peak", StartsAt: at(10, 24, 6, 0), ReturnsAt: &returns, Status: "open",
		MeetingPoint: "Vefa mall", Transport: "bus", Seats: 10, Taken: 2, CostPerPerson: &cost, Level: "advanced",
		Description: &text, Contact: &text, IsJoined: true, Trail: &api.TrailRef{ID: "t1", Slug: "ala-archa", Name: "Ala-Archa"},
		Organizer:    api.Person{ID: "p0", FirstName: str("Aibek")},
		Participants: []api.Person{{ID: "p1", FirstName: ann}, {ID: "p2", Username: bob}, {ID: "p3"}},
		Roster: []api.RosterEntry{
			{Person: api.Person{ID: "p1", FirstName: ann}, JoinedAt: joined, ArrivedAt: &arrived},
			{Person: api.Person{ID: "p3"}, JoinedAt: joined},
		},
		CanRate: true, MyRating: &api.RatingMine{Rating: 4, Text: &text},
	}
	organized := going
	organized.IsJoined, organized.IsOrganizer, organized.StartsAt = false, true, at(10, 20, 18, 0)
	past := going
	past.StartsAt = at(10, 10, 6, 0)
	cancelled := going
	cancelled.Status = "cancelled"
	tour := api.Tour{
		ID: "t1", Slug: "karakol", Title: "Karakol trek", Excerpt: &text, Price: &cost, DurationDays: &days, Location: &place,
		StartDate: "2026-11-02", EndDate: "2026-11-04", RatingAverage: 4.8, ReviewCount: 12, Content: "<p>Lakes.</p>",
		Program: []api.ProgramDay{{Title: "Arrival", Text: "Camp"}, {Text: "Pass"}}, Capacity: &seats,
		Phone: &phone, Whatsapp: &phone, Telegram: &phone,
	}
	date := "2026-11-02"
	orders := []api.Order{
		{Kind: "tour", ItemID: "t1", ItemDate: &date, Quantity: 2, Amount: 1000, ContactName: "Ann", ContactPhone: phone, Status: "paid"},
		{Kind: "tour", ItemID: "t1", Quantity: 1, Amount: 500, ContactName: "Bob", ContactPhone: phone, Status: "refunded"},
	}
	published := at(10, 18, 9, 0)
	post := api.BlogPost{ID: "b1", Title: "Autumn in the mountains", Excerpt: &text, PublishedAt: &published, Content: "<p>Go.</p>"}
	sum := &api.OutingsSummary{Outings: 5, DistanceKm: 42, ElevationGain: 3100}
	links := screens.Links{Bot: "ermekten_bot"}

	all := map[string]screens.Screen{
		"my outings":       screens.MyOutings(en, []api.Outing{going, organized, cancelled}, sum, screens.FromMine, now),
		"my past outings":  screens.MyOutings(en, nil, nil, screens.FromPast, now),
		"no outings":       screens.MyOutings(en, nil, nil, screens.FromMine, now),
		"find":             screens.FindOutings(en, []api.Outing{going, organized}, 0),
		"find nothing":     screens.FindOutings(en, nil, 0),
		"card":             screens.OutingCard(en, going, screens.FromFind, links, now),
		"organizer's card": screens.OutingCard(en, organized, screens.FromPast, links, now),
		"past card":        screens.OutingCard(en, past, screens.FromMine, links, now),
		"cancelled card":   screens.OutingCard(en, cancelled, screens.FromMine, links, now),
		"cancel":           screens.CancelConfirm(en, organized, screens.FromMine),
		"leave":            screens.LeaveConfirm(en, going, screens.FromMine),
		"tours":            screens.Tours(en, screens.ToursView{Tab: screens.TabUpcoming, Tours: []api.Tour{tour, tour}}, links),
		"no favorites":     screens.Tours(en, screens.ToursView{Tab: screens.TabFavorites, AlertOn: true}, links),
		"no tours":         screens.Tours(en, screens.ToursView{Tab: screens.TabUpcoming}, links),
		"tour":             screens.TourMore(en, tour, screens.TabUpcoming, 0, 0, true, links),
		"blog":             screens.Blog(en, []api.BlogPost{post, post}, 0, true, links),
		"empty blog":       screens.Blog(en, nil, 0, false, links),
		"article":          screens.Article(en, post, 0, 0, links),
		"notifications":    screens.Notifications(en, screens.NotifySettings{ToursDestinations: 2}),
		"notifications on": screens.Notifications(en, screens.NotifySettings{ToursAll: true, Blog: true}),
		"room":             screens.OrganizerHome(en, []api.Outing{organized}, []api.Tour{tour}, now),
		"empty room":       screens.OrganizerHome(en, nil, nil, now),
		"roster":           screens.Roster(en, organized, nil, links, at(10, 20, 17, 0)),
		"roster later":     screens.Roster(en, organized, nil, links, at(10, 10, 17, 0)),
		"write":            screens.BroadcastPrompt(en, organized),
		"preview":          screens.BroadcastPreview(en, organized, "See you"),
		"message":          screens.BroadcastMessage(en, organized, "See you"),
		"sent":             screens.BroadcastSent(en, organized, 1, 3),
		"organizer's tour": screens.OrganizerTour(en, tour, 0, &api.StatCounts{Views: 3, Contacts: 1}, orders, links),
		"program":          screens.OrganizerProgram(en, tour, 0, 0),
		"no program":       screens.OrganizerProgram(en, api.Tour{Title: "Empty"}, 0, 0),
		"bookings":         screens.OrganizerBookings(en, tour, 0, orders),
		"no bookings":      screens.OrganizerBookings(en, tour, 0, nil),
		"rate":             screens.RateOuting(en, past),
		"thanks":           screens.RateThanks(en, past, now),
		"review":           screens.ReviewPrompt(en, past),
		"trail state":      screens.ConditionState(en, past),
		"traction":         screens.ConditionTraction(en, past, 1),
		"report sent":      screens.ConditionThanks(en, past, 1, 2),
		"draft gone":       screens.DraftCancelled(en),
		"welcome":          {Text: screens.Welcome(en, "Ann", &going)},
	}
	fresh := drafts.New("ann", now)
	fresh.Taken, fresh.Search = 2, "nothing"
	for _, step := range drafts.Steps {
		d := fresh
		d.Step, d.Date, d.Time = step, "2026-10-24", "06:00"
		all["new outing at "+string(step)] = screens.DraftScreen(en, d, now)
	}
	edited := drafts.FromOuting(organized, "ann")
	all["edited outing"] = screens.DraftScreen(en, edited, now)
	edited.Step = drafts.StepFields
	all["fields"] = screens.DraftScreen(en, edited, now)
	edited.Step, edited.Reviewing = drafts.StepCost, true
	all["changing a field"] = screens.DraftScreen(en, edited, now)

	for name, s := range all {
		if r, ok := cyrillic(s.Text); ok {
			t.Errorf("%s: %q in %q", name, r, s.Text)
		}
		for _, row := range layout(s) {
			if r, ok := cyrillic(row); ok {
				t.Errorf("%s: %q in the buttons %q", name, r, row)
			}
		}
	}
}
