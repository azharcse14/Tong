<div align="center">

# ☕ Chai Shop API

**Cha-er dokaner menu, ekdom Go-r style-e.**
Ekta halka, fast REST API — menu CRUD, image upload, Bangla/English response, ar built-in Swagger docs.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-pgx%20v5-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)
![k6](https://img.shields.io/badge/load%20test-k6-7D64FF?logo=k6&logoColor=white)

</div>

---

## ✨ Ki ace

| | Feature | Details |
|---|---|---|
| 🍵 | **Menu CRUD** | Item add, dekha, update, delete |
| 📄 | **Pagination** | `?page=1&limit=10` (max `100`) |
| 🖼️ | **Image upload** | `jpg` / `jpeg` / `png`, max 5MB |
| 🌐 | **Bilingual response** | `Accept-Language: bn` dile Bangla message, otherwise English |
| ✅ | **Validation** | Name 2–100 char, price `> 0` |
| 📚 | **Swagger UI** | `/docs` e ready |
| 🛡️ | **Middleware** | Request Logger + panic Recoverer |
| 🌱 | **Seed command** | 1000 ta fake item dhukiye de |
| 🔥 | **Load test** | k6 script ready |

Shudhu stdlib `net/http` (Go 1.22+ routing) — kono heavy framework nai.

---

## 🧱 Project structure

```
chai-shop/
├── main.go                  # server entrypoint + routes
├── cmd/seed/main.go         # 1000 item seed command
├── internal/
│   ├── db/                  # pgx connection pool
│   ├── handler/             # menu, upload, docs, middleware
│   └── i18n/                # en.json, bn.json, Localizer
├── migrations/              # database schema (golang-migrate)
├── swagger.yaml             # OpenAPI 3 spec
├── load_test.js             # k6 load test
├── Dockerfile               # multi-stage build
├── docker-compose.yml       # app + postgres
└── uploads/                 # uploaded images
```

---

## 🚀 Kemne chalabe

### Option 1 — Docker (shobcheye shohoj) 🐳

```bash
docker compose up --build
```

API ready: **http://localhost:8080**
Postgres host-e: `localhost:5433` (user `postgres`, password `password`, db `chaidukan`)

### Option 2 — Local-e chalao 💻

**Lagbe:** Go 1.27+, PostgreSQL, [golang-migrate](https://github.com/golang-migrate/migrate)

```bash
# 1. Database banao
createdb chaidukan

# 2. .env file banao
cat > .env <<'EOF'
DB_URL=postgresql://postgres:password@localhost:5432/chaidukan
PORT=8080
EOF

# 3. Table create koro
make migrate-up

# 4. Run
go run .
```

Terminal-e dekhbe: `দোকান খুলেছে! Port 8080 এ চলছে...` ☕

### Environment variables

| Variable | Kaj | Example |
|---|---|---|
| `DB_URL` | Postgres connection string | `postgresql://postgres:password@localhost:5432/chaidukan` |
| `PORT` | Server port | `8080` |

---

## 📡 API

Base URL: `http://localhost:8080` · Interactive docs: **[/docs](http://localhost:8080/docs)**

| Method | Endpoint | Kaj |
|---|---|---|
| `GET` | `/menu` | Menu list (paginated) |
| `POST` | `/menu` | Notun item |
| `PUT` | `/menu/{id}` | Item update |
| `DELETE` | `/menu/{id}` | Item delete |
| `POST` | `/menu/{id}/image` | Image upload (`multipart/form-data`, field: `image`) |
| `GET` | `/uploads/{file}` | Upload kora image |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/swagger.yaml` | OpenAPI spec |

### Quick try

```bash
# TOKEN ta POST /auth/login theke pabe

# Item add
curl -X POST localhost:8080/menu \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name_bn":"মিল্ক টি","name_en":"Milk Tea","price":30}'

# List (Bangla message soho)
curl "localhost:8080/menu?page=1&limit=5" -H "Accept-Language: bn"

# Update
curl -X PUT localhost:8080/menu/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name_bn":"লেবু চা","name_en":"Lemon Tea","price":25}'

# Image upload
curl -X POST localhost:8080/menu/1/image -H "Authorization: Bearer $TOKEN" -F "image=@./chai.jpg"

# Delete
curl -X DELETE localhost:8080/menu/1 -H "Authorization: Bearer $TOKEN"
```

### Response format

Shob response-er shape ek-i:

```json
{
  "success": true,
  "message": "সফল হয়েছে",
  "data": [{ "id": 1, "name": "মিল্ক টি", "price": 30 }],
  "meta": { "total": 1, "page": 1, "limit": 10, "total_pages": 1 }
}
```

> `meta` shudhu list endpoint-e ase. Error hole `success: false` ar `data: null`.

### Status codes

| Code | Kokhon |
|---|---|
| `200` / `201` | Success / Created |
| `400` | Invalid payload, invalid id, name/price validation fail, `limit > 100`, bad image type (content-o check hoy) |
| `404` | `PUT` / image upload-e item khuje pawa jayni |
| `500` | Database error |

---

## 📚 Swagger docs

Server chalu thakle browser-e kholo:

| URL | Ki pabe |
|---|---|
| **http://localhost:8080/docs** | Swagger UI — endpoint dekho, **Try it out** diye sorasori request pathao |
| http://localhost:8080/swagger.yaml | Raw OpenAPI 3 spec (Postman / Insomnia-te import kora jay) |

Spec-er source file: [`swagger.yaml`](swagger.yaml). Notun endpoint add korle ei file-e o update koro, tahole `/docs` e auto dekhabe. Docker image-e `swagger.yaml` already copy kora hoy.

---

## 🌐 Language

`Accept-Language` header-er prothom 2 akkhor dekhe language thik hoy:

- `bn` → বাংলা
- `en` (ba onno kichu) → English

Notun message add korte `internal/i18n/en.json` ar `bn.json`-e same key dao.

---

## 🌱 Seed data

1000 ta random chai/snacks item dhukate:

```bash
go run ./cmd/seed
```

---

## 🔥 Load test

[k6](https://k6.io) lagbe. Server chalu rekhe:

```bash
k6 run load_test.js
```

Default: `10000` VU, `10s`, `GET /menu`. `load_test.js`-e `vus` / `duration` bodlate paro.

---

## 🗃️ Purano database ache?

`make migrate-up` chalao. Khub purano table-e `image_url` na thakle:

```sql
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS image_url TEXT;
```

---

## 🛠️ Tech stack

- **Go** `net/http` (method + path-pattern routing)
- **pgx v5** — Postgres driver + connection pool
- **go-playground/validator** — struct validation
- **godotenv** — `.env` loading
- **Docker / Compose** — Go multi-stage build + Postgres
- **k6** — load testing

---

## 🛠️ Migration

```bash
make migrate-up          # migration apply করো
make migrate-down        # migration undo করো
make migrate-create name=add_phone_to_users  # নতুন migration বানাও
```

<div align="center">

Made with ☕ and Go

</div>
