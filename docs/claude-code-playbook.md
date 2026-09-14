# Claude Code Playbook — Nasıl Çalışacaksın

Bu paketi Claude Code'a verdikten sonra izleyeceğin operasyonel akış. Amaç:
Claude Code'un uzun oturumlarda dağılmasını önlemek, faz faz ilerlemek, ve her
fazı test edilmiş halde kapatmak.

## 0. Kurulum

1. Bu klasörün TAMAMINI proje kök dizinine kopyala (CLAUDE.md kökte olmalı).
2. `.claude/agents/` altındaki dört subagent otomatik yüklenir.
3. Claude Code'u proje kökünde başlat. İlk mesajda:
   > "CLAUDE.md ve docs/ altını oku. Sonra Faz 0'a başla. Kod yazmadan önce
   > planını göster."

## 1. Faz Döngüsü (her faz için tekrarla)

Her fazda şu ritmi uygula:

```
1. "Faz N'e başlayalım. docs/roadmap.md Faz N'i ve ilgili docs dosyalarını oku,
    planı çıkar." → Claude planı gösterir, sen onaylarsın.
2. Claude uygun subagent'ı çağırarak (veya sen açıkça isteyerek) kodu yazar.
3. "Bu fazın kabul kriterlerine göre testleri yaz ve çalıştır." (test-e2e-engineer)
4. Testler yeşil → commit. "Faz N tamam, commit at, Faz N+1'e geçmeden özetle."
5. Bir sonraki faza geç.
```

Kural: **bir fazı bitmeden diğerine atlama.** Claude atlamaya çalışırsa durdur.

## 2. Subagent'ları Ne Zaman Çağır

Claude çoğu zaman kendi seçer; ama sen açıkça yönlendirebilirsin:

| İş | Subagent |
|----|----------|
| Prisma şema, migration, op log, snapshot, replay | persistence-engineer |
| Yjs sync, y-protocols, awareness, WS mesaj mantığı | crdt-sync-engineer |
| Next sayfaları, Server Actions, auth, SSR bootstrap | nextjs-app-engineer |
| Vitest / Playwright, multi-client senaryolar | test-e2e-engineer |

Açık çağırma örneği:
> "crdt-sync-engineer ile Faz 3 sync handshake'ini yaz."

## 3. Bağlam Yönetimi (uzun oturum hijyeni)

- Uzun oturumda Claude eski kararları unutabilir. CLAUDE.md INVARIANTS bunu
  önler ama yine de: yeni oturuma başlarken "CLAUDE.md'yi tekrar oku" de.
- Büyük refactor öncesi `git commit` at; Claude'un geri alınabilir noktalar bırakması iyi.
- Claude bir INVARIANT'ı ihlal eden kod yazarsa: "Bu INVARIANT #X'i ihlal ediyor,
  düzelt" — hangi numara olduğunu söylemen düzeltmeyi hızlandırır.

## 4. Sık Yapılan Hatalara Karşı Uyarılar (Claude'a hatırlat)

- WebSocket'i Next Server Component içinde açmaya çalışmak → HATA. Ayrı ws-server + client component.
- Metni düz string diff olarak senkronlamak → INVARIANT #1 ihlali. Binary update kullan.
- Elle `version: int` kolonu ekleyip artırmak → INVARIANT #5 ihlali. Yjs state vector kullan.
- Snapshot'tan önce op log prune etmek → INVARIANT #3 ihlali. Sıra: commit → prune.
- Awareness verisini DB'ye yazmak → efemer, persist etme.
- Auth handshake'i atlayıp sonra eklemek → INVARIANT #2. Handshake ilk mesajdan önce.

## 5. Faz Bitiş Kontrol Listesi (her faz sonu sor)

- [ ] Kabul kriterleri karşılandı mı? (roadmap.md)
- [ ] İlgili testler yazıldı ve yeşil mi?
- [ ] Hiçbir INVARIANT ihlal edilmedi mi?
- [ ] `pnpm typecheck` temiz mi?
- [ ] Commit atıldı mı? (İngilizce mesaj)

## 6. Mülakat Hazırlığı (proje bitince)

docs/concepts/crdt-vs-ot.md ve version-vector.md'yi kendi cümlelerinle
anlatabilene kadar çalış. "Zorluk noktası" tam olarak bunlar:
- Last-write-wins neden veri kaybettirir.
- OT vs CRDT farkı ve neden CRDT seçtin.
- Version/state vector ne işe yarar, tek integer version'dan farkı.
- "Optimistic concurrency" burada neden reject değil reconcile.
