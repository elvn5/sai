# nginx в проде

nginx настраивается на сервере вручную, его конфига в репозитории нет. Сайт (`apps/site`, контейнер `site`) слушает `CLIENT_PORT`, админка (`apps/panel`, контейнер `panel`) — `ADMIN_PORT`, API (`gateway`) — `API_PORT`. Порты ниже (`3002`, `3005`) — подставьте свои.

Контейнеры публикуют эти порты только на `127.0.0.1` (переменная `BIND_ADDRESS`), так что снаружи к ним не попасть в обход nginx и HTTPS. Поэтому в `proxy_pass` и `upstream` должен стоять `127.0.0.1:<порт>`, а не внешний IP сервера. С `localhost` тоже работает, но nginx сначала пробует `::1` и пишет в лог ошибку на каждое такое соединение. Для отката можно задать в GitHub переменную `BIND_ADDRESS=0.0.0.0` и задеплоить заново.

## 1. Переход с Nuxt (один раз, релиз с фазой 7)

До фазы 7 на `CLIENT_PORT` работал Nuxt, а Angular-сайт — рядом на `SITE_PORT`, и nginx отдавал ему перенесённые разделы. После релиза контейнер `site` слушает `CLIENT_PORT`, а `SITE_PORT` больше никто не слушает. Админке ничего менять не нужно: `panel` занимает тот же `ADMIN_PORT`, что и Nuxt-админка.

1. **До релиза**, если в nginx уже есть upstream на `SITE_PORT`, добавьте ему запасной сервер на `CLIENT_PORT` — тогда разделы не упадут в 502, пока вы не обновили конфиг:

   ```nginx
   upstream ermekten_site { server 127.0.0.1:3005; server 127.0.0.1:3002 backup; keepalive 16; }
   ```

   `sudo nginx -t && sudo systemctl reload nginx`. Пока `SITE_PORT` отвечает, запасной сервер не используется.
2. Выпустить релиз (тег `vX.Y.Z`). Деплой удалит контейнеры `tourism` и `admin` и поднимет `site` и `panel`.
3. Заменить фрагменты из раздела 2 и удалить переменную `SITE_PORT` в GitHub → Settings → Environments → `production`.

## 2. Конфиг

```nginx
# В блоке http {}: кеш HTML. Публичные страницы рендерятся анонимно и
# одинаковы для всех; сайт сам задаёт срок (Cache-Control: s-maxage=300,
# stale-while-revalidate=600). Личные страницы (вход, профиль, кабинет)
# приходят с no-store и не кешируются, как и ответы без Cache-Control.
proxy_cache_path /var/cache/nginx/ermekten-site levels=1:2 keys_zone=ermekten_site:20m
                 max_size=1g inactive=1h use_temp_path=off;

upstream ermekten_site { server 127.0.0.1:3002; keepalive 16; }
```

```nginx
# В server {} для ermekten.com — вместо прежних location.
location / {
    proxy_pass http://ermekten_site;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_cache ermekten_site;
    proxy_cache_key $scheme$host$request_uri;
    proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
    proxy_cache_background_update on;
    proxy_cache_lock on;
    add_header X-Cache-Status $upstream_cache_status always;
}
```

Заголовки проксирования оставьте как сейчас: `Host` обязательно `$host` (сайт отвечает только для разрешённых хостов), плюс `X-Forwarded-For`/`X-Forwarded-Proto`. Ответы `503` (сбой API) сайт отдаёт без кеша.

Применить: `sudo nginx -t && sudo systemctl reload nginx`.

## 3. Проверка

- `curl -sI https://ermekten.com/blog` — `X-Cache-Status: MISS`, при повторе `HIT`; в HTML нет `/_nuxt/`;
- `curl -sI https://ermekten.com/profile` — `X-Robots-Tag: noindex`, `Cache-Control: no-store`;
- `curl -s https://ermekten.com/sitemap.xml | head` — список всех страниц (`urlset`: разделы, туры, блог, организаторы, маршруты); `robots.txt` отвечает сайт;
- вход по паролю и через Telegram, возврат из банка на `/payments/return?order=…` (адрес возврата — `SITE_URL` сервиса платежей), кабинет партнёра, Telegram Mini App;
- на странице маршрута грузится карта (`/media/leaflet/leaflet.css` — 200) и прогноз погоды;
- админка открывается и входит.

После переключения 1–2 недели следить за Search Console: покрытие, ошибки, Core Web Vitals.

## 4. Откат

Nuxt-образы больше не собираются, но прошлые остались в реестре. Откат: GitHub → Actions → Deploy Production → Run workflow, в «Use workflow from» и в поле `tag` — прошлый тег (его версия workflow поднимет Nuxt), затем вернуть прежние `location` в nginx.
