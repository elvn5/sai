package cabinet

import (
	"html"
	"net/url"
	"strings"

	"github.com/elvn5/sai/services/bot/internal/screens"
)

var waiverPoints = []string{
	"Правила участия и риски",
	"Я здоров(а), у меня нет противопоказаний к нагрузкам и высоте, а о болезнях и лекарствах я написал(а) в анкете.",
	"Я следую указаниям гида и не ухожу от группы без его разрешения; гид может снять меня с маршрута ради безопасности.",
	"Я возьму снаряжение и одежду по списку организатора; без нужного снаряжения меня могут не допустить к выходу.",
	"Маршрут, время и программа могут измениться из-за погоды и состояния тропы — решение принимает гид.",
	"Я понимаю, что походы в горах связаны с рисками — травмы, резкая смена погоды, удалённость от медпомощи, — и участвую добровольно.",
	"Советуем страховку, которая покрывает активный отдых в горах; без неё расходы на эвакуацию и лечение несу я.",
}

func (v *view) waiver(id string, back action) (screens.Screen, error) {
	lines := []string{}
	for _, point := range waiverPoints {
		lines = append(lines, html.EscapeString(v.l.T(point)))
	}
	v.button("Я прочитал(а) правила, понимаю риски и согласен(на) с ними", func(v *view) (screens.Screen, error) {
		_, err := v.request("PUT", "/tours/"+url.PathEscape(id)+"/form", record{"acceptWaiver": true})
		if err != nil {
			return screens.Screen{}, err
		}
		return back(v)
	})
	v.button("Назад", back)
	return v.screen(strings.Join(lines, "\n\n")), nil
}
