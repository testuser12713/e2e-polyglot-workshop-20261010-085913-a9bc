# Kfz-Werkstatt-Kundenportal

Ein Kundenportal für eine Kfz-Werkstatt. Eine Go-API verwaltet Kunden, Fahrzeuge
und Werkstattaufträge auf einem echten PostgreSQL und stößt beim Fertigstellen
eines Auftrags eine Rechnung über eine Valkey-Warteschlange an. Ein Python-Worker
holt diese Aufträge ab, berechnet die Rechnung (Arbeitszeit × Stundensatz plus
Teile, 19 % MwSt, alle Beträge in ganzen Cent), speichert sie in PostgreSQL und
legt eine Benachrichtigung im Postausgang ab; ein echter Mailversand findet nicht
statt. Eine Vite/React/TypeScript-Web-App bietet einen öffentlichen Kundenbereich
(Terminanfrage, Statusabfrage, Rechnungsansicht) und einen angemeldeten
Werkstattbereich (Auftragsliste, Positionen, Status, Dashboard).

Dieses Repository enthält den API-Dienst unter `backend/`. Der Worker
(`worker/`) und die Web-App (`frontend/`) werden von ihren eigenen Tickets
aufgebaut.

## Tech Stack

- **Werkstatt-API**: Go (aktuelle Version), `net/http` aus der Standardbibliothek,
  `pgx/v5` für PostgreSQL, `golang-jwt/jwt/v5` für Bearer-Token,
  `golang.org/x/crypto/bcrypt` für Passwort-Hashes, `valkey-go` für Valkey
- **Datenbank**: PostgreSQL 18
- **Warteschlange**: Valkey 9.1
- **Konfiguration**: Umgebungsvariablen gemäß dem Startvertrag in `RUN.json`

## Installation

Voraussetzungen: Go (aktuelle Version) und Docker (für die beiden Datenspeicher).

```bash
# Datenbank und Warteschlange starten (PostgreSQL auf 5432, Valkey auf 6379)
docker compose up -d

# Go-Abhängigkeiten laden
cd backend
go mod download
```

## Konfiguration

Die API liest ihre Konfiguration ausschließlich aus Umgebungsvariablen. Fehlt ein
benötigter Wert, nennt die Startmeldung den Variablennamen und der Prozess endet
mit einem Exit-Code ungleich Null; für Geheimnisse gibt es nie einen
Literallwert im Repository.

| Variable | Pflicht | Bedeutung |
| --- | --- | --- |
| `DATABASE_URL` | ja | PostgreSQL-Verbindung, z. B. `postgresql://app:app@localhost:5432/app` |
| `VALKEY_URL` | ja | Valkey-Verbindung, z. B. `redis://localhost:6379/0` |
| `AUTH_TOKEN_SECRET` | ja (Geheimnis) | Schlüssel zum Signieren der Bearer-Token |
| `WORKSHOP_EMPLOYEE_EMAIL` | ja | E-Mail des ersten Mitarbeiters (wird beim Start angelegt) |
| `WORKSHOP_EMPLOYEE_PASSWORD` | ja (Geheimnis) | Passwort des ersten Mitarbeiters (wird nur gehasht gespeichert) |
| `HOURLY_RATE_CENTS` | nein | Stundensatz in ganzen Cent (Standard `0`) |
| `AUTH_TOKEN_TTL_MINUTES` | nein | Gültigkeitsdauer des Tokens in Minuten (Standard `480`) |
| `FRONTEND_ORIGIN` | nein | Erlaubter Frontend-Ursprung für CORS (Standard `http://localhost:5173`) |
| `PORT` | nein | HTTP-Port (Standard `8000`) |

`RUN.json` erklärt `AUTH_TOKEN_SECRET` und `WORKSHOP_EMPLOYEE_PASSWORD` als
`generate`-Werte: Sie existieren bewusst nicht im Repository. Für einen lokalen
Start erzeugt man sie selbst, zum Beispiel:

```bash
export AUTH_TOKEN_SECRET="$(openssl rand -hex 32)"
export WORKSHOP_EMPLOYEE_PASSWORD="$(openssl rand -hex 16)"
```

## Starten (Entwicklung)

```bash
cd backend
export DATABASE_URL="postgresql://app:app@localhost:5432/app"
export VALKEY_URL="redis://localhost:6379/0"
export AUTH_TOKEN_SECRET="$(openssl rand -hex 32)"
export WORKSHOP_EMPLOYEE_EMAIL="chef@werkstatt.example"
export WORKSHOP_EMPLOYEE_PASSWORD="$(openssl rand -hex 16)"
go run ./cmd/api
```

Beim Start öffnet die API den PostgreSQL-Pool, wendet das Schema
(`migrations/0001_init.sql`) idempotent an und legt den ersten Mitarbeiter an,
falls er noch nicht existiert. `GET /api/health` antwortet mit
`{"status":"ok"}`, solange die Datenbank erreichbar ist.

## Build für Produktion

```bash
cd backend
go build -o workshop-api ./cmd/api
DATABASE_URL=... VALKEY_URL=... AUTH_TOKEN_SECRET=... \
  WORKSHOP_EMPLOYEE_EMAIL=... WORKSHOP_EMPLOYEE_PASSWORD=... \
  ./workshop-api
```

## API-Verwendung

Alle Pfade liegen unter `/api`. Geschützte Werkstatt-Routen erwarten den Header
`Authorization: Bearer <JWT HS256>`; die Token-Claims sind `sub` (Mitarbeiter-ID),
`email` und `exp`. Geldbeträge sind immer ganze Cent.

### Endpunkte

| Methode | Pfad | Zweck |
| --- | --- | --- |
| `GET` | `/api/health` | Health-Check, Antwort `{"status":"ok"}` |
| `POST` | `/api/public/appointments` | Terminanfrage mit Kunde, Fahrzeug und Auftrag |
| `GET` | `/api/public/orders/{order_number}?plate=` | Öffentliche Statusabfrage |
| `GET` | `/api/public/orders/{order_number}/invoice?plate=` | Öffentliche Rechnungsansicht |
| `POST` | `/api/workshop/login` | Anmeldung, liefert Token |
| `GET` | `/api/workshop/orders?status=&plate=` | Auftragsliste mit Filter und Suche |
| `GET` | `/api/workshop/orders/{order_number}` | Auftragsdetail |
| `POST` | `/api/workshop/orders/{order_number}/items` | Position anlegen |
| `PATCH` | `/api/workshop/orders/{order_number}/items/{item_id}` | Position ändern |
| `DELETE` | `/api/workshop/orders/{order_number}/items/{item_id}` | Position löschen |
| `POST` | `/api/workshop/orders/{order_number}/status` | Status setzen |
| `GET` | `/api/workshop/dashboard` | Dashboard-Kennzahlen |

Der Status eines Auftrags durchläuft
`angefragt → bestaetigt → in_arbeit → fertig → abgeholt`.

### Beispiele

Terminanfrage:

```http
POST /api/public/appointments
Content-Type: application/json

{
  "customer": { "name": "Anna Beispiel", "email": "anna@example.test", "phone": "0151 2345678" },
  "vehicle":  { "plate": "KA-AB 123", "brand": "VW", "model": "Golf", "mileage": 120000 },
  "order":    { "preferred_date": "2026-11-03", "description": "Bremsen quietschen" }
}
```

Antwort `201`:

```json
{ "order_number": "A-2026-0001", "status": "angefragt" }
```

Anmeldung:

```http
POST /api/workshop/login
Content-Type: application/json

{ "email": "chef@werkstatt.example", "password": "..." }
```

Antwort `200`:

```json
{ "token": "<JWT>", "expires_at": "2026-11-03T12:00:00Z" }
```

### Fehlerformat

Jede Fehlerantwort hat denselben Körper mit stabilem Fehlercode und lesbarer
Meldung:

```json
{ "error": { "code": "not_found", "message": "resource not found" } }
```

Mögliche Codes: `validation_failed` (400), `unauthorized` (401), `not_found`
(404), `conflict` (409), `rate_limited` (429), `internal_error` (500) und
`not_implemented` (501, nur für noch nicht implementierte Routen). Eine
500-Antwort enthält nur den stabilen Fehlercode und eine generische Meldung —
weder SQL-Text noch Stacktrace, Dateipfad oder Datenbankfehlermeldung.

## Tests

```bash
cd backend
go test ./...
```

Die API-Tests laufen mit `net/http/httptest` gegen ein echtes PostgreSQL aus
`DATABASE_URL`.

## Funktionsumfang dieses Diensts (Skelett)

- Konfiguration aus Umgebungsvariablen, Validierung beim Start mit Nennung
  fehlender Variablennamen
- PostgreSQL-Pool, idempotentes Schema (`migrations/0001_init.sql`) und Anlage
  des ersten Mitarbeiters beim Start
- `GET /api/health` mit echter Datenbankprüfung
- Vollständig verdrahteter Router mit allen Vertrags-Routen; noch nicht
  implementierte Routen antworten `501 not_implemented`
- Einheitlicher Fehlerkörper, CORS nur für den konfigurierten Frontend-Ursprung,
  generische 500-Antworten
- Seams für die Feature-Tickets: Token-Ausgabe und -Prüfung, Valkey-Publisher
  und die Store-Methodensignaturen
