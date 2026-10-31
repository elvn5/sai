package screens_test

import (
	"testing"
	"unicode"

	"github.com/elvn5/sai/services/bot/internal/api"
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
	joined := at(10, 18, 9, 0)
	cost, seats, days := 500, 12, 3
	text, place, phone, meet, point := "Pace: easy", "Karakol", "+996 555 000 000", "06:00", "Vefa mall"
	ann, bob := str("Ann"), str("bob")
	arrived := at(10, 24, 5, 50)
	tour := api.Tour{
		ID: "t1", Slug: "karakol", Title: "Karakol trek", Excerpt: &text, Price: &cost, DurationDays: &days, Location: &place,
		StartDate: "2026-11-02", EndDate: "2026-11-04", RatingAverage: 4.8, ReviewCount: 12, Content: "<p>Lakes.</p>",
		Program: []api.ProgramDay{{Title: "Arrival", Text: "Camp"}, {Text: "Pass"}}, Capacity: &seats,
		Phone: &phone, Whatsapp: &phone, Telegram: &phone, MeetingTime: &meet, MeetingPoint: &point, Taken: 2,
		Organizer: &api.Person{ID: "p0", FirstName: str("Aibek")},
	}
	going := tour
	going.StartDate, going.EndDate, going.IsJoined = "2026-10-24", "2026-10-24", true
	going.CanRate, going.MyRating = true, &api.RatingMine{Rating: 4, Text: &text}
	organized := tour
	organized.StartDate, organized.EndDate, organized.IsOrganizer = "2026-10-21", "2026-10-21", true
	organized.Roster = []api.RosterEntry{
		{Person: api.Person{ID: "p1", FirstName: ann}, JoinedAt: joined, ArrivedAt: &arrived},
		{Person: api.Person{ID: "p2", Username: bob}, JoinedAt: joined},
		{Person: api.Person{ID: "p3"}, JoinedAt: joined},
	}
	past := going
	past.StartDate, past.EndDate = "2026-10-10", "2026-10-10"
	unlimited := tour
	unlimited.Capacity = nil
	date := "2026-11-02"
	orders := []api.Order{
		{Kind: "tour", ItemID: "t1", ItemDate: &date, Quantity: 2, Amount: 1000, ContactName: "Ann", ContactPhone: phone, Status: "paid"},
		{Kind: "tour", ItemID: "t1", Quantity: 1, Amount: 500, ContactName: "Bob", ContactPhone: phone, Status: "refunded"},
	}
	published := at(10, 18, 9, 0)
	post := api.BlogPost{ID: "b1", Title: "Autumn in the mountains", Excerpt: &text, PublishedAt: &published, Content: "<p>Go.</p>"}
	links := screens.Links{Bot: "ermekten_bot"}
	me := api.Me{ID: "p1", FirstName: ann, Username: bob, Phone: &phone, PartnerStatus: api.PartnerApproved,
		PartnerKind: api.KindOrganizer, HasPassword: true}
	form := api.ParticipantForm{EmergencyName: "Mom", EmergencyPhone: phone, Experience: "mountaineering", Health: &text, Notes: &text}
	mine := api.MyAchievements{Code: "AB12CD34", Points: 51, Awards: []api.Award{
		{Achievement: api.Achievement{ID: "a1", Title: map[string]string{"ru": "Вершина", "en": "First summit"}, Points: 50}},
		{Achievement: api.Achievement{ID: "a2", Title: map[string]string{"en": "Early bird"}, Points: 1}},
	}}
	leaders := []api.HallEntry{{Rank: 1, User: api.Person{ID: "p1", FirstName: ann}, Points: 51}}

	all := map[string]screens.Screen{
		"my tours":         screens.MyTours(en, []api.Tour{going, organized, unlimited, past}, screens.FromMine, now),
		"my past tours":    screens.MyTours(en, nil, screens.FromPast, now),
		"no tours yet":     screens.MyTours(en, nil, screens.FromMine, now),
		"card":             screens.TourCard(en, going, screens.FromTours, links, now),
		"stranger's card":  screens.TourCard(en, unlimited, screens.FromMine, links, now),
		"organizer's card": screens.TourCard(en, organized, screens.FromPast, links, now),
		"past card":        screens.TourCard(en, past, screens.FromMine, links, now),
		"leave":            screens.LeaveConfirm(en, going, screens.FromMine),
		"tours":            screens.Tours(en, screens.ToursView{Tab: screens.TabUpcoming, Tours: []api.Tour{tour, tour}}, links),
		"no favorites":     screens.Tours(en, screens.ToursView{Tab: screens.TabFavorites, AlertOn: true}, links),
		"no tours":         screens.Tours(en, screens.ToursView{Tab: screens.TabUpcoming}, links),
		"tour":             screens.TourMore(en, tour, screens.TabUpcoming, 0, 0, true, links, now),
		"joined tour":      screens.TourMore(en, going, screens.TabUpcoming, 0, 0, false, links, now),
		"blog":             screens.Blog(en, []api.BlogPost{post, post}, 0, true, links),
		"empty blog":       screens.Blog(en, nil, 0, false, links),
		"article":          screens.Article(en, post, 0, 0, links),
		"notifications":    screens.Notifications(en, screens.NotifySettings{ToursDestinations: 2}),
		"notifications on": screens.Notifications(en, screens.NotifySettings{ToursAll: true, Blog: true}),
		"room":             screens.OrganizerHome(en, []api.Tour{organized, tour}, now),
		"empty room":       screens.OrganizerHome(en, nil, now),
		"roster":           screens.Roster(en, organized, nil, links, at(10, 20, 17, 0)),
		"roster later":     screens.Roster(en, organized, nil, links, at(10, 10, 17, 0)),
		"empty roster":     screens.Roster(en, tour, nil, links, now),
		"write":            screens.BroadcastPrompt(en, organized),
		"preview":          screens.BroadcastPreview(en, organized, "See you"),
		"message":          screens.BroadcastMessage(en, organized, "See you"),
		"sent":             screens.BroadcastSent(en, organized, 1, 3),
		"organizer's tour": screens.OrganizerTour(en, tour, 0, &api.StatCounts{Views: 3, Contacts: 1}, orders, links),
		"program":          screens.OrganizerProgram(en, tour, 0, 0),
		"no program":       screens.OrganizerProgram(en, api.Tour{Title: "Empty"}, 0, 0),
		"bookings":         screens.OrganizerBookings(en, tour, 0, orders),
		"no bookings":      screens.OrganizerBookings(en, tour, 0, nil),
		"rate":             screens.RateTour(en, past),
		"thanks":           screens.RateThanks(en, past),
		"review":           screens.ReviewPrompt(en, past),
		"welcome":          {Text: screens.Welcome(en, "Ann", &going)},
		"profile": screens.Profile(en, screens.ProfileView{Me: me, Joined: []api.Tour{going, past}, JoinedKnown: true,
			Awards: &mine, FormKnown: true, Form: &form, Organizing: true}, now),
		"empty profile":    screens.Profile(en, screens.ProfileView{Me: api.Me{PartnerStatus: api.PartnerPending, PartnerKind: api.KindDriver}, FormKnown: true}, now),
		"form":             screens.FormScreen(en, &form, ""),
		"no form":          screens.FormScreen(en, nil, ""),
		"form name":        screens.FormStep(en, screens.StepName, "Mom", ""),
		"form phone":       screens.FormStep(en, screens.StepPhone, "", ""),
		"form level":       screens.FormStep(en, screens.StepExperience, "hiking", ""),
		"form health":      screens.FormStep(en, screens.StepHealth, "", ""),
		"form notes":       screens.FormStep(en, screens.StepNotes, "Vegan", ""),
		"password":         screens.PasswordScreen(en, me, ""),
		"first password":   screens.PasswordScreen(en, api.Me{Email: &place}, ""),
		"no login":         screens.PasswordScreen(en, api.Me{}, ""),
		"password current": screens.PasswordStep(en, screens.StepCurrent, ""),
		"password new":     screens.PasswordStep(en, screens.StepNew, ""),
		"password again":   screens.PasswordStep(en, screens.StepConfirm, ""),
		"awards":           screens.Awards(en, mine, leaders, "p1"),
		"no awards":        screens.Awards(en, api.MyAchievements{Code: "X"}, nil, "p1"),
		"partner":          screens.Partner(en, api.Me{PartnerStatus: api.PartnerRejected}, true, ""),
		"organizer":        screens.Partner(en, me, true, ""),
		"driver":           screens.Partner(en, api.Me{PartnerStatus: api.PartnerApproved, PartnerKind: api.KindDriver}, true, ""),
		"applied":          screens.Partner(en, api.Me{PartnerStatus: api.PartnerPending}, true, ""),
	}

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
