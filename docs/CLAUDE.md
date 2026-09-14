# CLAUDE.md — Real-Time Collaborative Markdown Editor

> Bu dosya Claude Code'un her oturumda okuduğu kalıcı proje bağlamıdır.
> Kısa, kesin, kararları içerir. Uzun anlatım `docs/` altındadır.

## Proje Nedir

Birden fazla kullanıcının aynı Markdown dokümanını aynı anda, veri kaybı olmadan
düzenlediği full-stack gerçek zamanlı editör. Çakışma çözümü **CRDT (Yjs)** ile,
senkronizasyon **custom WebSocket server** ile, kalıcılık **op log + snapshot
checkpoint (event-sourcing-lite)** ile yapılır.

Çözülen çekirdek problem: naif "last-write-wins" veri kaybına yol açar. CRDT ile
her replika bağımsız düzenlenip deterministik biçimde merge edilir; hiçbir edit kaybolmaz.

## Kesin Mimari Kararlar (değiştirme, gerekçe docs/ADR'de)

| Karar | Seçim | Neden |
|-------|-------|-------|
| CRDT tipi | Yjs `Y.Text` | Markdown düz metin; ProseMirror node ağacı gereksiz |
| Editör | CodeMirror 6 + `y-codemirror.next` | Yjs binding olgun, Markdown lang desteği hazır |
| Sync protokolü | `y-protocols` (sync + awareness) | Yeniden icat etme; battle-tested wire format |
| WS server | Custom `ws` server (Next dışında ayrı process) | Next App Router native WS desteklemez; auth+persistence bizde |
| Persistence | Postgres: op log + periyodik snapshot compaction | Replay + audit + performans dengesi |
| İlk yükleme | Server Component + Server Action (doküman create/list) | SSR ile ilk state, sonra client WS devralır |
| ORM | Prisma | Migration + tip güvenliği |
| Auth | JWT (httpOnly cookie) → WS handshake'te doğrulanır | REST ve WS aynı kimlik |

## Repo Yapısı (monorepo, pnpm workspaces)

```
apps/
  web/              # Next.js App Router (UI, Server Actions, REST auth)
  ws-server/        # Bağımsız Node.js ws server (Yjs sync + persistence)
packages/
  db/               # Prisma schema + client (her iki app import eder)
  protocol/         # Paylaşılan WS mesaj tipleri, auth token doğrulama
  shared/           # Ortak tipler, zod şemaları
```

Ana kural: **web ve ws-server ayrı process'lerdir**, Postgres üzerinden değil,
paylaşılan `packages/db` üzerinden aynı şemayı kullanır. WS server Next'in içinde
DEĞİLDİR — bu bilinçli bir mimari karardır (docs/adr/0002).

## Teknoloji Sürümleri (sabit)

- Node.js 20 LTS, pnpm 9
- Next.js 15 (App Router, React 19)
- yjs 13, y-protocols, y-codemirror.next, codemirror 6
- ws 8
- PostgreSQL 16, Prisma 6
- TypeScript 5 (strict), zod, jose (JWT)
- Test: vitest (unit), playwright (multi-client E2E)

## Kritik Değişmezler (INVARIANTS — asla ihlal etme)

1. Sunucu Yjs state'ini **binary update** olarak saklar/aktarır — asla düz metin diff değil.
2. Her WS bağlantısı, ilk mesajdan ÖNCE auth doğrulamasından geçer. Doğrulanmamış
   socket hiçbir doküman odasına yazamaz/okuyamaz.
3. Op log **append-only**'dir. Snapshot alınınca eski update'ler silinebilir ama
   snapshot'tan önceki hiçbir update snapshot yazılmadan silinmez (sıra: snapshot commit → prune).
4. Bir dokümanın sunucudaki otoriter Y.Doc'u tek bir process'te tek instance'tır
   (şimdilik tek-node; ölçekleme docs/adr/0005'te ele alınır, uygulanmaz).
5. Version vector / state vector Yjs tarafından yönetilir; elle version integer
   TUTMA — Yjs'in state vector'ünü kullan (docs/concepts version-vector'ü açıklar).

## Geliştirme Sırası (fazlar — sırayla ilerle)

Detay: `docs/roadmap.md`. Her faz bitince ilgili faz için testler yeşil olmalı.

1. Faz 0 — Monorepo iskeleti, Prisma şeması, migration, health check
2. Faz 1 — Auth (register/login, JWT cookie, Server Actions)
3. Faz 2 — Doküman CRUD (Server Actions + Server Component listeleme/SSR yükleme)
4. Faz 3 — WS server: y-protocols sync, auth handshake, in-memory Y.Doc odaları
5. Faz 4 — Persistence: update append, load-on-open, snapshot compaction
6. Faz 5 — Frontend: CodeMirror + Yjs binding, awareness (cursor/presence)
7. Faz 6 — Op log replay endpoint + audit görünümü (event sourcing gösterimi)
8. Faz 7 — Testler (multi-client Playwright), sağlamlaştırma, README + mimari diyagram

## Çalışma Kuralları (Claude Code için)

- Her faz başında ilgili `docs/` dosyasını oku, sonra kod yaz.
- Kod yazmadan önce ilgili paketin mevcut dosyalarını `view` et; varsayma.
- Migration gerektiren şema değişikliğinde `prisma migrate dev` çalıştır, SQL'i kontrol et.
- Yeni bağımlılık eklemeden önce gerekçesini bir cümleyle söyle.
- Test yazmadan bir fazı "bitti" sayma.
- INVARIANTS listesini ihlal eden bir çözüm önerirsen DUR ve önce bunu belirt.
- Türkçe açıkla, kod ve commit mesajları İngilizce.

## Komutlar

```bash
pnpm install
pnpm db:migrate          # prisma migrate dev
pnpm db:studio           # prisma studio
pnpm dev                 # web + ws-server paralel (turbo/concurrently)
pnpm --filter web dev
pnpm --filter ws-server dev
pnpm test                # vitest
pnpm test:e2e            # playwright multi-client
pnpm typecheck
```
