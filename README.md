# Realtime Trading Backend (Golang)

Mini backend trading system yang dibangun dengan Golang menggunakan **Clean Architecture**. Sistem ini mendukung REST API untuk manajemen order, WebSocket untuk data pasar realtime, dan menggunakan SQLite untuk persistence serta Redis untuk caching data pasar dan message broker.

---

## 1. Source Code
### a. Git Repository
- Repository: `https://github.com/sautmanurung1/mini-exchange`

### b. Struktur Project
Proyek ini mengikuti standar **Clean Architecture**:
```text
.
├── cmd/
│   └── server/          # Entry point aplikasi (main.go)
├── internal/
│   ├── delivery/        # Layer Transport (HTTP & WebSocket)
│   ├── domain/          # Layer Entitas & Interface (Abstraksi)
│   ├── repository/      # Layer Akses Data (Database & Redis)
│   └── usecase/         # Layer Logika Bisnis (Matching Engine, Auth)
└── README.md            # Dokumentasi Utama
```

---

## 2. README Requirements

### a. Cara Menjalankan Project
1. **Prerequisites**: Pastikan Golang (v1.20+) dan Redis Server (localhost:6379) sudah terinstall.
2. **Setup Env**: `cp .env.example .env`
3. **Install**: `go mod tidy`
4. **Run**: `go run cmd/server/main.go`

### b. Design Arsitektur (Singkat)
Sistem menggunakan **Clean Architecture** untuk memisahkan logika bisnis dari detail teknis (database/web). Dependensi hanya mengarah ke dalam (Domain). State transaksional disimpan di **SQLite**, sementara state pasar realtime disimpan dan didistribusikan via **Redis**.

### c. Flow System
1. Client mengirim order via **REST API** (dengan JWT).
2. `OrderUseCase` memvalidasi dan menyimpan order ke **Database**.
3. `MatchingEngine` memproses order tersebut menggunakan **Sharded Mutex** dan **Database Transaction**.
4. Jika terjadi Match, record `Trade` dibuat dan `Order` diupdate.
5. Event (Order/Trade/Ticker) di-publish ke **Redis Pub/Sub**.
6. **WebSocket Hub** menerima event dari Redis dan meneruskannya ke client yang relevan.

### d. Assumption yang Digunakan
- Sistem menangani beban menengah (~1000 order/menit).
- Client WebSocket (~500) melakukan subscribe ke 1-5 simbol saham.
- Redis dianggap selalu tersedia untuk mendukung fitur message broker.

### e. Penjelasan Potensi Race Condition
Race condition terjadi jika dua order untuk saham yang sama mencoba me-match order lawan yang sama secara simultan. Kami mengatasinya dengan:
1. **Sharded Mutex**: Lock per `StockCode` di level aplikasi.
2. **Database Transaction**: Menjamin atomisitas perubahan data di level disk.

### f. Strategi Broadcast Non-Blocking
Menggunakan goroutine per koneksi dengan **Buffered Channel**. Saat Hub mengirim pesan, digunakan idiom `select` dengan `default` case:
```go
select {
case client.send <- payload:
default: // Jika buffer penuh, skip pesan untuk client tersebut (Non-blocking)
}
```

### g. Tiga Bottleneck Utama dan Cara Mengatasinya
1. **SQLite Write Lock**: SQLite membatasi penulisan konkuren. *Solusi*: Migrasi ke PostgreSQL.
2. **Pub/Sub Parsing**: Parsing JSON di satu goroutine Hub bisa lambat. *Solusi*: Gunakan worker pool untuk parsing.
3. **Memory/FD Limit**: Batas koneksi TCP pada satu server. *Solusi*: Horizontal scaling dengan Load Balancer.

---

## 3. API Documentation

### a. List Endpoint & Payload

#### 1. Auth: Register
- **Endpoint**: `POST /api/auth/register`
- **Request Body**:
```json
{
  "username": "user123",
  "password": "password123"
}
```
- **Response**: `201 Created`

#### 2. Auth: Login
- **Endpoint**: `POST /api/auth/login`
- **Request Body**:
```json
{
  "username": "user123",
  "password": "password123"
}
```
- **Response**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### 3. Order: Create Order (Protected)
- **Endpoint**: `POST /api/orders`
- **Header**: `Authorization: Bearer <JWT_TOKEN>`
- **Request Body**:
```json
{
  "stock_code": "AAPL",
  "side": "BUY",
  "price": 150.5,
  "quantity": 100
}
```
- **Response**:
```json
{
  "id": "uuid-order-123",
  "user_id": "user-uuid",
  "stock_code": "AAPL",
  "side": "BUY",
  "price": 150.5,
  "quantity": 100,
  "remaining": 100,
  "status": "OPEN",
  "created_at": "2023-10-27T10:00:00Z"
}
```

#### 4. Order: List Orders
- **Endpoint**: `GET /api/orders?stock=AAPL&status=OPEN`
- **Query Params**: `stock` (optional), `status` (optional: OPEN, FILLED, PARTIAL_FILLED, CANCELLED)
- **Response**: Array of Order objects.

#### 5. Market: Trade History
- **Endpoint**: `GET /api/trades?stock=AAPL`
- **Response**:
```json
[
  {
    "id": "trade-uuid",
    "stock_code": "AAPL",
    "price": 150.5,
    "quantity": 10,
    "buyer_id": "buyer-uuid",
    "seller_id": "seller-uuid",
    "created_at": "2023-10-27T10:05:00Z"
  }
]
```

#### 6. Market: Snapshot
- **Endpoint**: `GET /api/market/snapshot?stock=AAPL`
- **Response**:
```json
{
  "ticker": {
    "stock_code": "AAPL",
    "last_price": 150.5,
    "change": 0.5,
    "volume": 5000
  },
  "order_book": {
    "stock_code": "AAPL",
    "bids": [{"price": 150.0, "quantity": 100}],
    "offers": [{"price": 151.0, "quantity": 50}]
  },
  "trades": [...]
}
```

---

## 4. WebSocket Documentation

### a. Cara Connect
Buka koneksi ke: `ws://localhost:8080/ws`

### b. Cara Subscribe & Unsubscribe
Kirim pesan JSON setelah terkoneksi:
- **Subscribe**: `{"action": "subscribe", "symbol": "AAPL"}`
- **Unsubscribe**: `{"action": "unsubscribe", "symbol": "AAPL"}`

### c. Format Message (Server to Client)
Setiap update akan dikirim melalui channel spesifik:

#### 1. Channel: `market.ticker`
Dikirim saat ada perubahan harga atau volume.
```json
{
  "channel": "market.ticker",
  "data": {
    "stock_code": "AAPL",
    "last_price": 150.75,
    "change": 0.25,
    "volume": 5100
  }
}
```

#### 2. Channel: `market.trade`
Dikirim setiap terjadi transaksi (match).
```json
{
  "channel": "market.trade",
  "data": {
    "id": "trade-uuid",
    "stock_code": "AAPL",
    "price": 150.75,
    "quantity": 10,
    "created_at": "2023-10-27T10:10:00Z"
  }
}
```

#### 3. Channel: `market.orderbook`
Update status bid/offer terbaru.
```json
{
  "channel": "market.orderbook",
  "data": {
    "stock_code": "AAPL",
    "bids": [{"price": 150.70, "quantity": 200}],
    "offers": [{"price": 150.80, "quantity": 150}]
  }
}
```

#### 4. Channel: `order.update`
Update status order milik user (jika user tersebut terhubung).
```json
{
  "channel": "order.update",
  "data": {
    "id": "order-uuid",
    "status": "FILLED",
    "remaining": 0
  }
}
```

### d. Rekomendasi Tools dan Cara Testing
1. **Postman**: Pilih "WebSocket Request", masukkan `ws://localhost:8080/ws`.
2. **wscat**: 
   ```bash
   # Install: npm install -g wscat
   wscat -c ws://localhost:8080/ws
   > {"action": "subscribe", "symbol": "AAPL"}
   ```