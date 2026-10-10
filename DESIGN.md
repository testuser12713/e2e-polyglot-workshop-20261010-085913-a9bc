# Design — Project Identity

> This document is project-long-lived. Tokens are not changed without
> the Architect's approval. Developers MUST use these tokens
> instead of improvising their own colors/spacings.

## Style Direction

Calm, professional light UI: near-white surfaces and a muted petrol-blue accent for trust, with a warm amber and clear green/red semantics for the order status model — precise like a workshop job sheet, friendly enough for a public customer area.

## Colors

- `--color-bg`: **#F5F7FA**
- `--color-surface`: **#FFFFFF**
- `--color-surface_alt`: **#EDF1F6**
- `--color-fg`: **#16212E**
- `--color-muted`: **#5A6B7C**
- `--color-border`: **#DCE2EA**
- `--color-accent`: **#0F5C99**
- `--color-accent_hover`: **#0C4A7C**
- `--color-accent_active`: **#093A61**
- `--color-accent_soft`: **#E3EEF7**
- `--color-on_accent`: **#FFFFFF**
- `--color-focus_ring`: **#2E86D6**
- `--color-success`: **#0E8A5F**
- `--color-warning`: **#A86A06**
- `--color-danger`: **#B3261E**
- `--color-status_requested_bg`: **#E8ECF1**
- `--color-status_requested_fg`: **#45525F**
- `--color-status_confirmed_bg`: **#E3EEF7**
- `--color-status_confirmed_fg`: **#0F5C99**
- `--color-status_in_progress_bg`: **#FBF0DC**
- `--color-status_in_progress_fg`: **#A86A06**
- `--color-status_done_bg`: **#E2F3EC**
- `--color-status_done_fg`: **#0E8A5F**
- `--color-status_picked_up_bg`: **#E4E8ED**
- `--color-status_picked_up_fg`: **#3B4754**

## Typography

- `font_family`: system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif
- `font_family_mono`: ui-monospace, 'SFMono-Regular', 'Cascadia Mono', Menlo, Consolas, 'Liberation Mono', monospace
- `heading_weight`: 600
- `body_weight`: 400
- `size_h1`: 28px / 1.25 / 600
- `size_h2`: 22px / 1.3 / 600
- `size_h3`: 18px / 1.35 / 600
- `size_body`: 15px / 1.5 / 400
- `size_label`: 13px / 1.4 / 500
- `size_caption`: 12px / 1.4 / 400
- `numeric_variant`: font-variant-numeric: tabular-nums on every money, quantity, mileage, duration and order-number value so columns align
- `font_loading`: system stack only — no webfont, no CDN; if a brand webfont is ever wanted it must be self-hosted from the app origin (AC-34)

## Spacing Scale

- `--space-0`: 4px
- `--space-1`: 8px
- `--space-2`: 12px
- `--space-3`: 16px
- `--space-4`: 24px
- `--space-5`: 32px
- `--space-6`: 48px

## Border-Radii

- `--radius-sm`: 6px
- `--radius-md`: 10px
- `--radius-lg`: 16px
- `--radius-pill`: 999px

## Components

### Button

Variants: primary (bg=accent #0F5C99, fg=on_accent), secondary (bg=surface, fg=accent, 1px border=#C9D6E4), ghost (transparent, fg=accent), destructive (bg=danger #B3261E, fg=white, only for 'Position löschen'). Padding 12px/20px (16px/24px on lg), radius=md 10px, body_weight 500, size_body. States: default as above; hover = primary bg accent_hover #0C4A7C / secondary bg accent_soft #E3EEF7; active = accent_active #093A61 plus translateY(1px); focus-visible = 2px outline focus_ring #2E86D6 with 2px offset, never removed; disabled = opacity 0.5, cursor not-allowed, no hover change; loading = label replaced by inline spinner, aria-busy=true, disabled. min-height 44px and min-width 44px on every size (mobile tap target). Full-width (100%) inside forms below 640px, auto width on desktop. Labels in German, e.g. 'Termin anfragen', 'Auftrag bestätigen', 'Status setzen', 'Anmelden', 'Abmelden', 'Speichern', 'Abbrechen', 'Löschen'.

### TextField

Label above field (size_label, fg=fg), required marker '*'. Field: height 44px, padding 10px/12px, bg=surface, 1px border=#C9D6E4, radius=sm 6px, size_body. States: default neutral — NO error text while untouched (AC-25); focus = border accent + 2px focus_ring halo; filled = as default; error (only after blur-with-input or submit) = border danger #B3261E + helper text below in danger, size_caption, prefixed '!'; disabled = bg surface_alt, fg=muted. Helper text slot below (size_caption, fg=muted) for formats like 'z. B. B-AB 1234'. Types used: text, email, tel, number, date, password, textarea (min-height 96px). Password field gets a show/hide toggle button (44px hit area, no value logged).

### Select

Native <select> styled like TextField (height 44px, radius sm, chevron icon right, 12px inset). Used for status filters and the position type ('Arbeitsposition' / 'Teilposition'). First option always an all-values entry, e.g. 'Alle Status'. Focus/disabled/error states identical to TextField. Full width on mobile.

### SearchField

Input with leading magnifier icon and a clear (x) button; placeholder in German 'Kennzeichen suchen'. Searches license plate as case-insensitive substring; matching fragment may be visually emphasized in the result row. Debounce 250ms, min-height 44px, full width on mobile above the list. Empty result shows EmptyState, never a blank area.

### StatusBadge

Pill (radius=pill, padding 4px/10px, size_caption, weight 500) with a 6px dot plus the German status word: 'angefragt' on status_requested_bg/fg, 'bestätigt' on status_confirmed_bg/fg, 'in Arbeit' on status_in_progress_bg/fg, 'fertig' on status_done_bg/fg, 'abgeholt' on status_picked_up_bg/fg. Text is always the word, colour is never the only signal (readable for colour-vision deficiencies). Same component in customer area, workshop list and dashboard counters.

### Card / Panel

bg=surface, 1px border border #DCE2EA, radius=lg 16px, padding 24px (16px below 640px), subtle shadow 0 1px 2px rgba(22,33,46,0.06). Optional header row: size_h3 title left, action (e.g. Button secondary) right, wraps to stacked below 640px. Used for order detail, invoice, dashboard tiles, login.

### OrderListRow

Desktop ≥640px: table row — Auftragsnummer (mono, copy button), Kennzeichen (uppercase), Kunde, Wunschtermin (DD.MM.YYYY), StatusBadge, chevron; row height 56px, hover bg surface_alt, entire row is one click target to open the detail. Below 640px the same data becomes a stacked card: line 1 = Auftragsnummer + StatusBadge, line 2 = Kennzeichen + Fahrzeug, line 3 = Wunschtermin; no horizontal scroll (AC-26). Numbers right-aligned with tabular-nums.

### PositionRow

One line per order position. Arbeitsposition: Beschreibung, 'Arbeitszeit' with value formatted as duration, Stundensatz snapshot if present. Teilposition: Bezeichnung, Menge, Einzelpreis, Zeilensumme. Columns on desktop: Beschreibung (flex) | Menge/Zeit (right) | Einzelpreis (right) | Summe (right) | actions (Bearbeiten/Löschen ghost buttons, 44px). Below 640px label/value pairs stacked, actions right-aligned. Deleted row leaves immediately and the order totals below re-render.

### TotalsBlock

Right-aligned block under the positions: 'Zwischensumme', '19 % MwSt.', 'Gesamtbetrag' (weight 600, size_h3). All amounts via MoneyText. Values update in place after any position create/edit/delete. Separated from positions by a 1px border and 16px padding.

### MoneyText

Renders an integer cent value in German format with tabular-nums: '1.234,56 €' (dot = thousands, comma = decimals, always 2 decimals, trailing space before €). Negative values get a leading '−'. Used for every amount in the product (positions, Zwischensumme, MwSt., Gesamtbetrag, Monatsumsatz) — never float, never a different format anywhere.

### AppBar

Height 64px (56px mobile), bg=surface, bottom border, sticky top. Customer area: logo/wordmark left, tabs 'Termin', 'Status', 'Rechnung' right (desktop) / full-width tab bar below the bar (mobile). Workshop area: wordmark left, links 'Dashboard', 'Aufträge' plus the workshop e-mail and 'Abmelden' right; on mobile a 44px menu button opens a full-screen sheet with the same links. Active route gets accent colour plus a 2px underline (never colour alone).

### Tabs

Customer area top-level navigation between the three public flows. Desktop: horizontal text tabs, 44px tall, accent underline on active. Mobile ≤640px: full-width segmented control, equal thirds, 44px tall, tap area = whole third. Keyboard: arrow keys move, Home/End jump.

### InlineAlert / Toast

Feedback channel for every workshop action (AC-18, AC-20, AC-24). Success: bg status_done_bg, left border 3px success, icon check, message e.g. 'Auftrag bestätigt.' Error/409: bg #FBEAE8, left border 3px danger, message e.g. 'Dieser Übergang ist nicht erlaubt.' plus the human-readable API message. Renders inline above the affected card and, for list actions, as a toast bottom-center on mobile / top-right on desktop, auto-dismiss after 6s with a manual close. aria-live=polite for success, assertive for errors.

### ConfirmDialog

Modal for the four status transitions ('Auftrag bestätigen?', 'In Arbeit setzen?', 'Auftrag fertig melden?', 'Als abgeholt markieren?'). Overlay rgba(22,33,46,0.45), panel surface radius lg, max-width 480px, padding 24px, title size_h3, body size_body, footer with Button secondary 'Abbrechen' + primary confirm. On mobile the panel is bottom-sheet-like (full width, radius lg top corners, safe-area padding). Focus trapped, Escape cancels, focus returns to the trigger.

### EmptyState

Centered block, padding 48px/24px: 32px line icon in muted, size_h3 heading, one sentence body in muted, optional action Button. Mandatory for every list/query: 'Noch keine Aufträge', 'Keine Treffer für diesen Filter', 'Für diesen Auftrag liegt noch keine Rechnung vor' (AC-13, AC-17), 'Status nicht gefunden — bitte Auftragsnummer und Kennzeichen prüfen' (neutral, no hint about existence, AC-12).

### LoadingSkeleton

List rows and cards render 3 grey placeholder lines (bg surface_alt, radius sm, 200ms shimmer) while a fetch is running; buttons submitting show their own spinner. No layout jump between skeleton and loaded state.

### DashboardStatCard

Card with size_caption label, big MoneyText-or-number in size_h1/600 with tabular-nums, and a one-line muted caption. Three tiles: 'Offene Aufträge' (angefragt, bestätigt, in Arbeit), 'Heute fertiggestellt' (count by the timestamp of the change to 'fertig'), 'Umsatz laufender Monat' (MoneyText of the current calendar month). Desktop: 3 columns in one row; mobile: stacked full width. Numbers re-render visibly when an order becomes 'fertig' (AC-21).

### LegalFooter

Present on EVERY route of both areas (AC-33): top border, bg=surface_alt, padding 24px, size_caption fg=muted. Contains links 'Impressum' and 'Datenschutz' plus the workshop name. Links are plain internal routes, 44px tap height on mobile; external requests are not made before consent (AC-34).

### LoginForm

Card, max-width 400px centered: TextField 'E-Mail' (type=email), TextField 'Passwort' (type=password with show/hide), primary Button 'Anmelden' full width. Errors only after submit (AC-25); on 401 a single neutral message 'E-Mail oder Passwort ist falsch.' — identical for unknown e-mail and wrong password (AC-29); on 429 'Zu viele Versuche. Bitte kurz warten.' Never echoes credentials; no credential or token appears in any UI log line (AC-35).

### InvoiceView

Card with header 'Rechnung' + Auftragsnummer, meta row (Kennzeichen, Rechnungsdatum DD.MM.YYYY), the position list reusing PositionRow in read-only mode, then TotalsBlock. Before status 'fertig' the route shows EmptyState wording 'Die Rechnung ist ab dem Status fertig verfügbar.' instead of any partial data (AC-13).

### StatusTimeline

Vertical list of history entries: StatusBadge at left, datetime right-aligned in muted mono-ish tabular style, newest last (chronological). 2px connector line in border colour; past entries full opacity, append-only so nothing is ever removed after further transitions (AC-06). Stacks below the status headline on mobile.

## Layout Principles

- Container max-width 1120px, centered; page padding 16px below 640px, 24px up to 1024px, 32px above. Forms are single-column and capped at 560px, invoices/detail cards at 720px, the workshop order list may use the full 1120px.
- Breakpoints: base 390px (phone, the reference width from the Design Directives), sm 640px (list tables switch to stacked cards, buttons go full width), md 768px, lg 1024px (dashboard tiles go 3-across, tabs move into the AppBar), xl 1280px (max container). No horizontal scrolling at 390px in either area (AC-26).
- Grid: single column on mobile; 12-column grid from lg (forms 5/7, dashboard 4/4/4, order detail 7/5 for positions vs. summary). Spacing between sections always 24px, between related fields 12px, inside a card 24px (16px on mobile).
- ONE format for every value the product shows — dates: 'DD.MM.YYYY' (e.g. 14.03.2025). Times: 24-hour 'HH:MM' (stored UTC, displayed in Europe/Berlin). Date+time: 'DD.MM.YYYY, HH:MM Uhr'. Durations (Arbeitszeit): decimal-comma hours with one decimal, e.g. '2,5 h'. Quantities: whole numbers with unit, e.g. '2 Stk.'. Mileage: '128.450 km'. Money: whole-cent integers rendered as '1.234,56 €' with dot thousands separator, comma decimals, always two decimals, via the MoneyText component. Order numbers: uppercase mono, e.g. 'AUF-2025-0187'. License plates: normalized uppercase 'B-AB 1234'. Every screen and every ticket uses exactly these formats.
- Every interactive element is a real 44x44px minimum hit area (buttons, tab thirds, row chevrons, icon toggles) and shows focus-visible as a 2px focus_ring outline with 2px offset; focus is never suppressed.
- Every fetch has three explicit states: LoadingSkeleton, EmptyState or the real content, and an error state that shows only the stable API error message — never SQL text, stack traces or personal data (AC-32, AC-35).
- Status is always communicated by word + shape + colour together (StatusBadge text, StatusTimeline), never by colour alone; the five statuses keep their fixed colours across customer area, list and dashboard.
- LegalFooter with 'Impressum' and 'Datenschutz' appears on every route of both areas, and no third-party font, script or CDN is loaded before consent — all assets come from the app origin (AC-33, AC-34).
- Forms start neutral: no error styling until a field has been typed into and left or the form has been submitted; required fields are marked, helper text names the expected format (AC-25).
- Destructive or irreversible workshop actions (status transitions, deleting a position) go through ConfirmDialog and report their result via InlineAlert/Toast, so no click ever stays without visible effect (AC-24).
