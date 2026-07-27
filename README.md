# Crypto Spot Signal

Monorepo paper-signal untuk memantau pasar SPOT Gate secara real time. Sistem membaca public trades, ticker, candlestick, dan incremental order book. Backend menghitung volume, aggressive buy/sell delta, liquidity, multi-timeframe trend, spoofing heuristic, signal score, dan hasil signal.

## Batas sistem

- Hanya membaca market data publik.
- Tidak memiliki endpoint pembuatan order.
- Tidak membutuhkan API key Gate untuk konfigurasi default.
- Signal adalah paper signal, bukan jaminan keuntungan.
- Spoof score bersifat heuristik. Order book SPOT tetap dapat dimanipulasi.
- DeepSeek atau Grok hanya meninjau candidate yang sudah dibuat rule engine.

## Isi proyek

```text
backend/       Go market ingestor, local order book, feature engine,
               signal engine, PostgreSQL, Redis, REST, dan WebSocket
ai-service/    FastAPI adapter deterministic, DeepSeek, dan Grok
web/           Next.js dashboard, chart, order book, scanner,
               performance summary, dan browser notification
docs/          Arsitektur, scoring, API, dan deployment
scripts/       Pemeriksaan struktur dan safety boundary
```

## Fitur yang sudah tersedia

- Gate SPOT WebSocket reconnect dan heartbeat.
- REST candle backfill.
- REST order book snapshot dan WebSocket sequence validation.
- Semua interval candle Gate yang dikonfigurasi.
- Trade volume dan aggressive buy/sell delta satu menit.
- Relative volume.
- EMA 9 dan EMA 20 per timeframe.
- Multi-timeframe trend alignment.
- Spread, bid depth, ask depth, dan book imbalance.
- Removed-liquidity spoof heuristic.
- Data-quality gate.
- BUY setup dan BUY confirmed paper signal.
- Signal cooldown.
- Deterministic fallback saat AI mati atau gagal.
- DeepSeek JSON mode adapter.
- Grok structured-output adapter.
- TimescaleDB tables untuk candle, order book metric, feature, signal, dan outcome.
- Return evaluation 5 menit, 15 menit, 1 jam, dan 4 jam.
- MFE, MAE, target hit, dan invalidation hit.
- Dashboard real time melalui WebSocket.
- Candlestick chart dan top order book levels.
- Browser notification berdasarkan minimum score.

## Menjalankan dengan Docker

1. Buat `.env` dari contoh.

```bash
cp .env.example .env
```

Pada Windows, `start.bat` akan membuat `.env` secara otomatis jika belum ada.

2. Jalankan semua service.

```bash
docker compose up --build
```

3. Buka layanan.

```text
Dashboard       http://localhost:3000
Backend health  http://localhost:8080/health
AI health       http://localhost:8090/health
```

## Konfigurasi pasar

Default memakai data Gate secara langsung:

```env
MARKET_MODE=gate
```

Untuk menguji dashboard tanpa koneksi exchange, ubah menjadi:

```env
MARKET_MODE=mock
```

Default pair dan timeframe:

```env
GATE_PAIRS=BTC_USDT,ETH_USDT,SOL_USDT,XRP_USDT,DOGE_USDT
GATE_TIMEFRAMES=10s,1m,5m,15m,30m,1h,4h,8h,1d,7d
GATE_ORDERBOOK_INTERVAL=100ms
```

Mulai dari pair likuid. Jangan langsung memasukkan seluruh pair. Setiap pair menambah trade stream, ticker, order book, candle subscription, memory, database writes, dan reconnect workload.

## Mode AI

Default menggunakan deterministic review:

```env
AI_ENABLED=false
AI_PROVIDER=none
```

DeepSeek:

```env
AI_ENABLED=true
AI_PROVIDER=deepseek
AI_API_KEY=isi_api_key
AI_MODEL=isi_model_yang_tersedia_di_akun
```

Grok:

```env
AI_ENABLED=true
AI_PROVIDER=grok
AI_API_KEY=isi_api_key
AI_MODEL=isi_model_yang_tersedia_di_akun
```

Nama model disimpan dalam environment variable karena ketersediaan model dapat berbeda menurut akun dan berubah dari waktu ke waktu.

## Endpoint

```text
GET /health
GET /api/v1/scanner
GET /api/v1/signals?limit=100
GET /api/v1/signals/{id}
GET /api/v1/pairs/{symbol}
GET /api/v1/performance/summary
GET /api/v1/config
GET /ws
```

Detail payload terdapat di `docs/API.md`.

## Notifikasi

Dashboard meminta izin browser dan menampilkan notifikasi saat signal baru melewati minimum score yang dipilih. Jalankan situs melalui `localhost` atau HTTPS. Implementasi dalam paket ini ditujukan untuk browser yang sedang membuka atau menjalankan situs. Push saat browser benar-benar tertutup membutuhkan VAPID subscription backend yang belum dimasukkan ke MVP ini.

## Pemeriksaan lokal

```bash
python scripts/smoke_check.py
cd backend && go test ./internal/market ./internal/features
cd ../ai-service && python -m unittest discover -s tests
```

## Tahap sebelum penggunaan modal nyata

1. Kumpulkan data dalam beberapa kondisi pasar.
2. Evaluasi outcome per pair dan timeframe.
3. Masukkan fee dan slippage ke label hasil.
4. Lakukan walk-forward validation.
5. Jalankan paper trading dalam periode yang cukup.
6. Kalibrasi threshold per liquidity tier.
7. Pisahkan service eksekusi jika suatu hari dikembangkan.
8. Jangan memberikan izin withdrawal kepada API key exchange.

## Referensi teknis

Implementasi Gate mengikuti dokumentasi API v4 untuk public SPOT WebSocket, candlestick, book ticker, incremental order book, dan REST order book snapshot. Adapter AI memakai JSON response untuk DeepSeek dan schema-based structured output untuk Grok.
