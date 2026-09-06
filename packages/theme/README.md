# @app/theme

Единая фирменная цветовая схема Acme для всех фронтендов
(Tailwind CSS v4 + Nuxt UI v3).

## Цвета логотипа

| Токен   | Роль в Nuxt UI | Базовый цвет | Значение    |
| ------- | -------------- | ------------ | ----------- |
| `brand` | `primary`      | 500          | `#C33A22`   |
| `steel` | `neutral`      | 800          | `#434141`   |

## Подключение

В `assets/css/main.css` приложения:

```css
@import "tailwindcss";
@import "@nuxt/ui";
@import "@app/theme/theme.css";
```

В `app.config.ts`:

```ts
export default defineAppConfig({
  ui: { colors: { primary: 'brand', neutral: 'steel' } },
});
```

Доступны утилиты `bg-brand-500`, `text-steel-700`, `border-steel-200` и т.д.
Файл также содержит стили анимации переключения темы через View Transitions API.
