# Переключение разделов на Angular в проде (nginx)

Как отдать перенесённые разделы новому сайту (`apps/site`), пока остальное обслуживает Nuxt. Стратегия — в разделе 5 `docs/angular-migration-plan.md`. nginx настраивается на сервере вручную; в репозитории его конфига нет.

## 1. Запустить Angular-сайт рядом с Nuxt

1. В GitHub → Settings → Environments → `production` → Variables добавить `SITE_PORT` — свободный порт на сервере (например, `3005`).
2. Выпустить релиз (тег `vX.Y.Z`). Деплой соберёт образ `ermekten-site` и, раз задан `SITE_PORT`, поднимет контейнер `site` (compose-профиль `site`) рядом с Nuxt. Пока nginx не настроен, трафика на нём нет.
3. Проверить на сервере: `curl -H 'Host: ermekten.com' http://127.0.0.1:$SITE_PORT/healthz` → `{"status":"ok"}`.

## 2. Сравнить обход до переключения

С сервера (или любой машины, откуда доступен порт), после выкладки на staging и ручной проверки:

```bash
node scripts/compare-crawl.mjs --old https://ermekten.com --new http://127.0.0.1:3005 --host ermekten.com --sections /blog,/guides,/gallery,/hall-of-fame,/hotels,/local-guides,/tours,/trails,/outings,/
```

Скрипт берёт адреса из sitemap нового сайта и для каждой страницы сравнивает статус, title, description, canonical, robots, h1, hreflang и JSON-LD (`/` в списке — только главная). Код выхода 0 — можно переключать. Добавления на новой стороне (JSON-LD, hreflang) — это ожидаемые улучшения. Страниц отдельных выходов в sitemap нет (они `noindex`, как и в Nuxt), поэтому одну-две стоит открыть и сравнить вручную.

Также перед переключением: Lighthouse по разделам и ручная проверка в Telegram Mini App.

## 3. Конфиг nginx

Фрагменты для существующего `server { server_name ermekten.com; … }`. Порты `3002` (Nuxt, `CLIENT_PORT`) и `3005` (`SITE_PORT`) — подставьте свои. Заголовки проксирования оставьте как сейчас (`Host` обязательно `$host`: Angular отвечает только для разрешённых хостов).

```nginx
# В блоке http {}: кеш HTML. Страницы рендерятся анонимно, поэтому одинаковы
# для всех; сайт сам задаёт срок (Cache-Control: s-maxage=300,
# stale-while-revalidate=600), nginx его соблюдает.
proxy_cache_path /var/cache/nginx/ermekten-site levels=1:2 keys_zone=ermekten_site:20m
                 max_size=1g inactive=1h use_temp_path=off;

upstream ermekten_nuxt { server 127.0.0.1:3002; keepalive 16; }
upstream ermekten_site { server 127.0.0.1:3005; keepalive 16; }
```

```nginx
# В server {} для ermekten.com, перед location /.

# Файлы сборки Angular (имена с хешем: буквы, цифры, «_» и «-») и его
# служебные адреса.
location ~ ^/(main|chunk|styles|polyfills)-[A-Za-z0-9_-]+\.(js|css)$ { proxy_pass http://ermekten_site; }
location ^~ /media/ { proxy_pass http://ermekten_site; }
location = /app-config.json { proxy_pass http://ermekten_site; }

# sitemap: у нового сайта полнее (маршруты, выходы, hreflang).
location = /sitemap.xml { proxy_pass http://ermekten_site; }
location ^~ /sitemaps/ { proxy_pass http://ermekten_site; }

# Перенесённые разделы: волна 1 (контент), волна 2 (каталоги) и волна 3
# (маршруты, выходы). Каждая следующая волна дописывает свои разделы сюда и
# в MIGRATED_SECTIONS (apps/site/src/core/config/sections.ts,
# apps/tourism/plugins/migrated-sections.client.ts). Переключать можно и
# по одной волне: тогда в списке оставьте только её разделы.
location ~ ^/(ky/|en/|de/)?(blog|guides|gallery|hall-of-fame|hotels|local-guides|tours|trails|outings)(/|$) {
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

# Волна 3: главная — только сам адрес «/» и «/ky», «/en», «/de»
# (адрес со слэшем на конце Angular перенаправит 301).
location ~ ^/((ky|en|de)/?)?$ {
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

# Волна 4: вход, профиль, оплата, кабинет партнёра, выдача достижений.
# Эти страницы для вошедшего и рендерятся в браузере: без кеша.
location ~ ^/(ky/|en/|de/)?(login|profile|payments|partner|achievements)(/|$) {
    proxy_pass http://ermekten_site;
}

# Всё остальное — по-прежнему Nuxt.
location / { proxy_pass http://ermekten_nuxt; }
```

**После волны 4** у Nuxt не остаётся страниц: все пути сайта отдаёт Angular. Когда волна 4 проверена, `location /` можно направить на `ermekten_site` — тогда и неизвестные адреса (404), `robots.txt` и `logo.svg` отвечает Angular, а Nuxt не получает трафика. Блоки разделов при этом можно оставить как есть (ради кеша) или удалить. Nuxt и его контейнер убираются в фазе 7.

Применить: `sudo nginx -t && sudo systemctl reload nginx`.

Проверить:
- `curl -sI https://ermekten.com/blog` — ответ от Angular (в HTML нет `/_nuxt/`), `X-Cache-Status: MISS`, а при повторе `HIT`;
- `curl -s https://ermekten.com/sitemap.xml | head` — индекс `sitemapindex`;
- `/`, `/trails`, `/outings` — Angular, если включена волна 3; `/login`, `/profile`, `/partner` — Angular, если включена волна 4 (до этого — Nuxt);
- после волны 4: вход по паролю и через Telegram, возврат из банка на `/payments/return?order=…` (адрес возврата — `SITE_URL` сервиса платежей), кабинет партнёра;
- на странице маршрута грузится карта (`/media/leaflet/leaflet.css` отвечает 200 от Angular) и прогноз погоды;
- вход сохраняется при переходах между частями (cookie общие), страница `/blog/new` открывается у партнёра.

## 4. Откат

Удалить `location` раздела (и при необходимости sitemap) и выполнить `sudo nginx -t && sudo systemctl reload nginx` — трафик вернётся к Nuxt. Контейнер `site` можно не трогать; чтобы убрать его совсем, удалите переменную `SITE_PORT` и задеплойте.

## 5. После переключения

1–2 недели следить за Search Console: покрытие, ошибки, Core Web Vitals.
