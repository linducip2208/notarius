package app

// Katalog seed checklist + template konsep (pure, tanpa DB).
// Diekstrak dari handler dmsChecklistSeed (dms.go) & templateSeed (ext.go)
// agar bisa diuji murni tanpa database.

type checklistSeed struct {
	code, name string
	items      []string
}

// standardChecklistSeeds mengembalikan seluruh paket standar checklist.
func standardChecklistSeeds() []checklistSeed {
	return []checklistSeed{
		{"PT-DIRIAN", "Standar Pendirian PT", []string{
			"KTP + NPWP semua pendiri (foto jelas, NIK 16 digit)",
			"KTP + NPWP Direksi & Komisaris",
			"3 opsi nama PT (urutan prioritas)",
			"Modal dasar / ditempatkan / disetor + bukti setoran",
			"Komposisi saham (nama — lembar — nominal)",
			"Domisili + alamat lengkap PT (+ bukti sewa/kepemilikan)",
			"KBLI usaha + email & HP aktif PT",
			"Data BO: nama, % saham, PEP/non-PEP",
		}},
		{"CV-DIRIAN", "Standar Pendirian CV", []string{
			"KTP + NPWP semua sekutu (aktif & pasif)",
			"Nama CV + domisili usaha",
			"Porsi modal tiap sekutu + siapa sekutu aktif",
			"Alamat lengkap + email & HP aktif",
		}},
		{"FIRMA", "Standar Pendirian Firma", []string{
			"KTP + NPWP semua sekutu firma",
			"Nama firma + domisili usaha",
			"Perjanjian penyertaan modal + pembagian laba",
			"Alamat lengkap + email & HP aktif",
		}},
		{"YAYASAN-DIRIAN", "Standar Pendirian Yayasan", []string{
			"KTP + NPWP pendiri, pengurus & pengawas",
			"Nama yayasan + maksud & tujuan (AD)",
			"Kekayaan awal dipisahkan + bukti setoran",
			"Susunan organ (pembina/pengurus/pengawas)",
			"Domisili + alamat lengkap sekretariat",
		}},
		{"KOPERASI", "Standar Akta Koperasi", []string{
			"KTP minimal 9 pendiri (+ NPWP bila ada)",
			"Nama koperasi + AD/ART konsep",
			"Susunan pengurus & pengawas",
			"Rencana usaha + domisili + modal awal",
		}},
		{"PPAT-AJB", "Standar AJB", []string{
			"Sertifikat asli + cek bersih BPN (tidak sengketa/blokir)",
			"KTP + KK + NPWP penjual & pembeli + pasangan (persetujuan suami/istri)",
			"PBB lunas 10 tahun + NOP sesuai sertifikat",
			"PPh penjual lunas + BPHTB pembeli lunas (validasi)",
			"Surat belum sengketa + kuasa bila dikuasakan",
		}},
		{"PPAT-HIBAH", "Standar Hibah", []string{
			"Sertifikat asli + cek bersih BPN",
			"KTP pemberi & penerima hibah (+ pasangan pemberi)",
			"PBB lunas + BPHTB hibah",
			"Hubungan keluarga / alasan hibah tercatat",
		}},
		{"PPAT-APHB", "Standar APHB", []string{
			"Sertifikat waris + SKW (surat keterangan waris)",
			"KTP semua ahli waris + kesepakatan pembagian",
			"PBB lunas + BPHTB waris/bagi",
		}},
		{"PPAT-APHT", "Standar APHT/SKMHT", []string{
			"Sertifikat + SKMHT bila dikuasakan",
			"KTP pemberi & penerima HT + data kredit bank",
			"Peringkat HT + nilai tanggungan disepakati",
		}},
		{"PPAT-TUKAR", "Standar Tukar Menukar", []string{
			"Dua sertifikat asli + cek bersih BPN keduanya",
			"KTP + KK + NPWP kedua belah pihak (+ pasangan)",
			"PBB lunas 10 tahun kedua objek + nilai tukar disepakati",
		}},
		{"PPAT-SKMHT", "Standar SKMHT", []string{
			"Sertifikat + KTP pemberi & penerima kuasa",
			"Data kredit bank + batas waktu SKMHT dicatat",
		}},
		{"PPAT-INBRENG", "Standar Inbreng", []string{
			"Sertifikat + KTP pemilik tanah",
			"Akta pendirian/perubahan PT + RUPS persetujuan inbreng",
			"Nilai inbreng + penyesuaian modal disetor",
		}},
		{"PT-UBAH", "Standar Perubahan PT", []string{
			"Akta terakhir + SK AHU terakhir",
			"Data perubahan (AD/komposisi/pengurus/modal)",
			"Persetujuan pemegang saham / keputusan RUPS",
		}},
		{"PT-BUBAR", "Standar Pembubaran PT", []string{
			"Akta terakhir + SK AHU terakhir",
			"Keputusan RUPS pembubaran + penunjukan likuidator",
			"KTP likuidator + neraca penutup",
		}},
		{"CV-UBAH", "Standar Perubahan CV", []string{
			"Akta terakhir + SKT terakhir",
			"Data perubahan + kesepakatan sekutu",
		}},
		{"JUALBELI-NONPPAT", "Standar Jual-Beli non-PPAT", []string{
			"KTP para pihak + objek & harga disepakati",
			"Bukti kepemilikan objek (bukan sertifikat tanah)",
			"Cara & jadwal pembayaran",
		}},
		{"KUASA", "Standar Surat Kuasa", []string{
			"KTP pemberi & penerima kuasa",
			"Ruang lingkup kuasa + batas waktu",
		}},
		{"PRENUP", "Standar Prenup/Postnup", []string{
			"KTP calon/pasangan + KK",
			"Daftar harta bawaan + pemisahan disepakati",
		}},
		{"WASIAT", "Standar Akta Wasiat", []string{
			"KTP pewasiat + daftar ahli waris",
			"Objek wasiat + 2 saksi (dicatat saat pengesahan)",
		}},
		{"AHU-BANTU", "Standar Asistensi AHU", []string{
			"Data perseroan lengkap + akta terakhir",
			"Akun AHU + draft pengajuan diperiksa notaris",
		}},
		{"OSS-BANTU", "Standar Asistensi OSS", []string{
			"Data perusahaan + KBLI + NIB eksisting bila ada",
			"Akun OSS + alamat email aktif perusahaan",
		}},
		{"BUNDEL-PT-OSS", "Paket Pendirian PT + NIB + NPWP", []string{
			"KTP + NPWP semua pendiri, direksi & komisaris",
			"3 opsi nama PT + komposisi saham (nama — lembar — nominal)",
			"Modal dasar / ditempatkan / disetor + bukti setoran",
			"Domisili + KBLI usaha + email & HP aktif PT",
			"Data BO: nama, % saham, PEP/non-PEP",
			"Akun AHU/OSS + NPWP perusahaan (diurus paralel pasca-akta)",
		}},
		{"BUNDEL-AJB-BALIKNAMA", "Paket AJB + Balik Nama", []string{
			"Sertifikat asli + cek bersih BPN (tidak sengketa/blokir)",
			"KTP + KK + NPWP penjual & pembeli (+ persetujuan pasangan)",
			"PBB lunas 10 tahun + NOP sesuai sertifikat",
			"PPh penjual lunas + BPHTB pembeli lunas (validasi)",
			"AJB ditandatangani + resi lapor balik nama BPN",
		}},
		{"BUNDEL-CV-NIB", "Paket CV + NIB", []string{
			"KTP + NPWP semua sekutu (aktif & pasif)",
			"Nama CV + porsi modal tiap sekutu + siapa sekutu aktif",
			"Domisili usaha + KBLI + alamat lengkap",
			"Email & HP aktif CV + akun OSS untuk penerbitan NIB",
		}},
		{"HUTANG", "Standar Hutang Piutang", []string{
			"KTP kreditur & debitur (+ pasangan bila menikah)",
			"Pokok hutang + tenor + bunga bila ada disepakati",
			"Jadwal angsuran + cara pembayaran dicatat",
			"Jaminan bila ada (bukti kepemilikan + nilai disepakati)",
			"Kuasa bila dikuasakan",
		}},
		{"PK-UMUM", "Standar Perjanjian Umum", []string{
			"KTP para pihak (+ NPWP bila ada)",
			"Maksud & tujuan perjanjian disepakati",
			"Hak & kewajiban + jangka waktu",
			"Tanda tangan + penyelesaian sengketa disepakati",
		}},
		{"SEWA", "Standar Sewa Menyewa", []string{
			"KTP pemilik & penyewa",
			"Objek sewa + alamat lengkap",
			"Jangka waktu sewa (mulai — selesai)",
			"Harga sewa + cara pembayaran disepakati",
		}},
	}
}

type templateSeed struct{ name, cat, body string }

// ---- workflow engine seeds (migrasi 027) ----

type workflowStepSeed struct {
	key, label, requiredRole string
	requireChecklist         bool
	requireEvidence          string
	skippable                bool
	slaDays                  int // target hari penyelesaian langkah (<=0 = tanpa target)
}

type workflowSeed struct {
	code, name string
	steps      []workflowStepSeed
}

// standardWorkflowSeeds mengembalikan 3 template standar per layanan:
// PT-DIRIAN (9 langkah), PPAT-AJB (7), KUASA (5). Murni tanpa DB.
func standardWorkflowSeeds() []workflowSeed {
	return []workflowSeed{
		{"PT-DIRIAN", "Alur Pendirian PT", []workflowStepSeed{
			{"Intake", "Penerimaan", "staff", false, "", false, 1},
			{"Pihak", "Data pihak", "staff", false, "", false, 2},
			{"Data", "Data perusahaan", "staff", false, "", false, 2},
			{"Syarat", "Kelengkapan syarat", "staff", true, "", false, 3},
			{"Draf", "Penyusunan draf", "staff", false, "", false, 3},
			{"Review", "Reviu notaris", "notary", false, "", false, 2},
			{"Sahkan", "Pengesahan", "notary", false, "", false, 2},
			{"Register", "Registrasi AHU", "staff", false, "", true, 3},
			{"Serah", "Penyerahan", "staff", false, "", false, 1},
		}},
		{"PPAT-AJB", "Alur AJB", []workflowStepSeed{
			{"Intake", "Penerimaan", "staff", false, "", false, 1},
			{"Pihak", "Para pihak", "staff", false, "", false, 2},
			{"Objek", "Verifikasi objek", "staff", true, "", false, 2},
			{"Resi", "Cek bersih & resi", "staff", true, "", false, 3},
			{"Akta", "Pembuatan akta", "notary", false, "", false, 3},
			{"Lapor", "Pelaporan BPN", "staff", false, "BPN_RECEIPT", false, 2},
			{"Serah", "Penyerahan", "staff", false, "", false, 1},
		}},
		{"KUASA", "Alur Surat Kuasa", []workflowStepSeed{
			{"Intake", "Penerimaan", "staff", false, "", false, 1},
			{"Pihak", "Para pihak", "staff", false, "", false, 1},
			{"Draf", "Penyusunan draf", "staff", false, "", false, 2},
			{"Sahkan", "Pengesahan", "notary", false, "", false, 1},
			{"Serah", "Penyerahan", "staff", false, "", false, 1},
		}},
	}
}

// standardTemplateSeeds mengembalikan seluruh template KONSEP (DRAFT).
// Template konsep tidak bisa dipakai membuat draf — notaris wajib review
// & approve dulu. Fallback aman: tetap DRAFT.
func standardTemplateSeeds() []templateSeed {
	return []templateSeed{
		{"Standar Pendirian PT", "PT", tplPT},
		{"Standar Pendirian CV", "CV", tplCV},
		{"Standar AJB", "AJB", tplAJB},
		{"Standar Hibah", "Hibah", tplHibah},
		{"Standar Surat Kuasa", "Kuasa", tplKuasa},
		{"Standar Sewa Menyewa", "Sewa", tplSewa},
		{"Standar Prenup", "Prenup", tplPrenup},
		{"Standar Perjanjian Kerjasama", "Perjanjian", tplPK},
		{"Standar Wasiat", "Will", tplWasiat},
	}
}
