# FEATURE INVENTORY 4.0 — FASE 1A (status jujur per fitur)

Kolom: Nama | Source | Impl | DB | UI | Integrasi | Test | Kelemahan (≤2) | Rekomendasi (≤1) | Prio.
Impl: IMPLEMENTED_AND_VERIFIED=IV, IMPLEMENTED_BUT_BROKEN=IB, PARTIALLY_IMPLEMENTED=PART, UI_ONLY=UIO, NOT_CONNECTED=NC, NOT_IMPLEMENTED=NI, NEEDS_LEGAL_REVIEW=NLR. DB: RW=handler baca/tulis DB terverifikasi, R=read-only, -=tidak ada. UI: OK/parsial/-. Integrasi: -=internal saja, MOCK=simulasi lokal, NONE=hilang.

## clients

| Klien CRUD+wizard | `app/clients_crm.go`, `clients/` | IV | RW | OK | - | minim | slug unik blm cek; duplikat NIK lolos | tambah unique index + uji | P1 |
| CRM prospek/follow-up | `app/clients_crm.go`, `016_client_crm.sql` | PART | RW | OK | - | ada | konversi prospek→klien manual; tanpa riwayat | wizard konversi satu klik | P2 |
| Identitas KTP/KK tersimpan | `clients/identities.go` | PART | RW | parsial | - | ada | relasi identitas↔OCR belum otomatis | isi otomatis dari hasil OCR | P1 |

## requests + matter_parties

| Service request lifecycle | `requests/`, `app.go` (~status move) | IV | RW | OK | - | ada | transisi di 2 tempat (pkg+handler) | satukan guard transisi | P0 |
| Para pihak matter | `026_matter_workflow.sql`, `corp/` | PART | RW | parsial | - | - | peran pihak tak divalidasi; tanpa TTD pihak | validasi peran + log persetujuan | P1 |
| Janji/appointment+SLA | `requests/detail.go`, `remind/sla.go` | IV | RW | OK | - | ada | proteksi race slot ganda | unique index slot | P2 |

## deeds + templates

| Minuta draf→approve→nomor | `deeds/deeds.go`, `app/ext.go` | IV | RW | OK | - | - | nomor unik per-kantor saja;Fields rentan XSS | tambah sanitasi konten | P0 |
| Template dokumen + seed | `doctpl/`, `docengine/`, `seed_catalog.go` | PART | RW | OK | - | ada | draf 1-klik tanpa review paksa | wajibkan approve notaris | P1 |
| Legalisasi/warmeking register | `deeds/deeds.go:RegisterLegalization` | PART | RW | parsial | - | - | alur maju manual; **kewenangan perlu advokat cek** | NLR: verifikasi UUJN sblm rilis | P0 |

## DMS enterprise

| File versi/workflow/share | `dms/dms.go`, `app/dms*.go` | IV | RW | OK | - | - | `ClaimOCR` in-memory race multi-instans | pindah klaim ke `SELECT FOR UPDATE` | P0 |
| Arsip fisik/hold/disposisi | `dms/dms.go` (lokasi, hold) | IV | RW | OK | - | - | riwayat pindah hanya map longgar | tipiskan skema history | P2 |
| Custom field+approval chain | `dms/enterprise.go` | PART | RW | parsial | - | - | validasi tipe field lemah | tambah validasi tipe | P1 |
| TTE tanda tangan elektronik | label "siap TTE" di `app.go` | NI | - | UIO | NONE | - | klaim menyesatkan; tanpa BSrE | NLR: hapus klaim/hubungkan PSrE | P0 |

## finance + quotation

| Invoice+pembayaran+QR | `finance/finance.go`, `app/verify.go` | IV | RW | OK | - | ada | void tanpa jejak alasan wajib | wajibkan alasan + audit | P1 |
| Quotation+expire+konversi | `finance/quotation.go`, `app/quotations.go` | IV | RW | OK | - | ada | nomor manual rawan duplikat | generator nomor atomik | P1 |

## portal klien

| Tracking+upload+invoice portal | `app/verify.go` (portal*), `006_portal.sql` | PART | RW | OK | - | - | tokenisasinya HMAC baik; konfirmasi bayar manual | tambah unggah bukti bayar | P1 |

## PPAT register

| Wizard AJB PPAT | `app/ppat_wizard.go`, `021_ppat.sql` | PART | RW | OK | MOCK | - | hanya simpan notes teks; **tanpa cek kewenangan PPAT** | NLR: batasi peran PPAT + validasi sertipikat | P0 |

## wasiat

| Register wasiat+serah terima | `deeds/wills.go`, `app/will_wizard.go` | PART | R/W parsial | parsial | - | - | daftar wasiat pusat (Kemenkumham) tidak terhubung; **rezim hukum sensitif** | NLR: integrasi/penafian daftar pusat | P0 |

## CMS + growth

| CMS publik+media+SEO | `cms/`, `app/cms*.go`, `015/018` | IV | RW | OK | - | ada | sanitasi ganda (safehtml+cms) inkonsisten | satukan policy sanitasi | P2 |
| Konsultasi/booking/kalkulator/testimoni | `app/cms_growth_pub.go`, `019` | PART | RW | OK | NONE | - | notifikasi WA hanya link; tanpa kirim nyata | hubungkan gateway WA | P2 |

## users / RBAC 9 peran

| Auth+MFA+lockout | `auth/`, `users/mfa.go`, `mfa/` | IV | RW | OK | - | ada | recovery code blm dicek | tambah recovery + uji | P0 |
| RBAC 9 peran | `rbac/rbac.go`, `010_role_permissions.sql` | PART | R | parsial | - | - | matriks kode vs tabel DB bisa drift; auditor read-only blm diuji | uji matriks vs DB | P0 |

## OCR enterprise

| Parse KTP/KK/NPWP + tesseract | `ocr/` (ktp_parse, kk_npwp, tesseract) | PART | R | parsial | NONE | ada | butuh biner tesseract eksternal; kualitas tanpa skor ambang | fallback antrean manual saat biner absen | P1 |

## remind / workers

| Reminder SLA/digest/ulang tahun | `remind/`, `app/verify.go:remindersRun` | PART | R/W parsial | parsial | NONE | ada | `LogSender` no-op default (tidak kirim); picu manual | jadwalkan cron + sender nyata | P1 |
| Worker pool | `workers/workers.go` | PART | - | - | - | - | OCR queue tidak pakai pool ini (jalur terpisah) | satukan eksekusi OCR ke pool | P2 |

## verify / QR

| Verifikasi HMAC+QR publik | `verify/`, `app/verify.go` | IV | R | OK | - | ada | rate-limit ada; kadaluarsa kode tak ada | tambah masa berlaku kode | P1 |

## Ringkasan NLR (wajib sebelum klaim produksi): legalisasi/warmeking, TTE/BSrE, PPAT, wasiat/daftar pusat, klaim integrasi AHU/OSS (`gov/` = MOCK lokal).
