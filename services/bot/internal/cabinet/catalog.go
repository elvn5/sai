package cabinet

import (
	"html"
	"sort"

	"github.com/elvn5/sai/services/bot/internal/screens"
)

func numeric(a any) float64 {
	switch n := a.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}
func filterTours(items []record, query record) []record {
	out := []record{}
	for _, item := range items {
		days := numeric(item["durationDays"])
		duration := str(query["duration"])
		if query["free"] == true && item["cancellationPolicy"] != "free" {
			continue
		}
		if duration == "1" && days != 1 || duration == "2-3" && (days < 2 || days > 3) || duration == "4+" && days < 4 {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch str(query["sort"]) {
		case "price_asc":
			if a["price"] == nil {
				return false
			}
			if b["price"] == nil {
				return true
			}
			return numeric(a["price"]) < numeric(b["price"])
		case "price_desc":
			return numeric(a["price"]) > numeric(b["price"])
		case "rating":
			return numeric(a["ratingAverage"]) > numeric(b["ratingAverage"])
		default:
			return str(a["startDate"]) < str(b["startDate"])
		}
	})
	return out
}
func (v *view) destinationAlert(destination string, back action) (screens.Screen, error) {
	data, err := v.request("GET", "/tour-alerts/mine", nil)
	if err != nil {
		return screens.Screen{}, err
	}
	on := false
	if values, ok := object(data)["destinations"].([]any); ok {
		for _, value := range values {
			if value == destination {
				on = true
			}
		}
	}
	label, method := "Включить уведомления", "PUT"
	if on {
		label, method = "Выключить уведомления", "DELETE"
	}
	v.button(label, func(v *view) (screens.Screen, error) {
		_, err := v.request(method, "/tour-alerts", record{"destination": destination})
		if err != nil {
			return screens.Screen{}, err
		}
		return back(v)
	})
	v.button("Назад", back)
	return v.screen(v.l.T("Уведомления по направлению") + ": " + html.EscapeString(v.option(destination))), nil
}
