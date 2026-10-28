# Naskah video demo FrameCheck

Target durasi **±2:30** (rules tidak menyebut batas; cek form submission). Narasi dalam bahasa Inggris untuk juri internasional; petunjuk produksi dalam bahasa Indonesia.

Juri menilai **Demo 25%**: "working, impressive demo, holds up live, and the quality of the demo video". Video ini harus memperlihatkan **Muse Spark sungguhan** bekerja, bukan fixture.

## Persiapan sebelum merekam

- [ ] **Bersihkan database lokal** dari project tes (terutama project tes aplikasi kantor). Daftar project terlihat di halaman depan. Screenshot aplikasi kantor **tidak boleh** muncul di video (bukan milikmu, ada data pribadi).
- [ ] Server memakai provider `meta` dengan izin request yang cukup untuk rekaman + cadangan (mis. 3).
- [ ] Siapkan pasangan demo dari halaman **Lembar**: `demo/fixtures/reference.png` + `actual-missing-signin.png` (atau pasangan lain yang hasil Muse Spark-nya sudah dicek bagus).
- [ ] Browser: jendela 1920×1080, zoom 100%, sembunyikan bookmark bar dan ekstensi, mode jangan ganggu aktif.
- [ ] Rekam layar 1080p 30fps (QuickTime / OBS). Rekam narasi terpisah dengan mikrofon jika bisa, lalu gabungkan.
- [ ] Analisis Muse Spark butuh ±13 detik: di edit, percepat bagian menunggu (mis. 4×) dan beri label kecil "sped up".

## Alur

| Waktu | Gambar | Narasi (EN) |
| --- | --- | --- |
| 0:00–0:07 | `submission/video/framecheck-intro.mp4` | *(musik intro saja)* |
| 0:07–0:22 | Figma/desain di kiri, browser di kanan, mata berpindah-pindah (bisa pakai dua gambar Lembar berdampingan) | "Every front-end developer knows this: flipping between the design and the browser, hunting for what's different. A button that never rendered. A headline cut off. A card that slipped thirty pixels. Pixel-diff tools flag everything and explain nothing." |
| 0:22–0:35 | Halaman depan FrameCheck, gerakkan mouse di atas hero 3D | "FrameCheck asks Meta's Muse Spark the question a reviewer would ask — and makes every answer checkable." |
| 0:35–0:55 | Buat project, upload desain + build. Tunjukkan saran pixel ratio otomatis, lalu Pair dan klik salah satu contoh catatan | "Upload the design and a screenshot of the build, captured at the same size. FrameCheck reads the images and checks they can be compared. Add a note about what matters." |
| 0:55–1:15 | Klik **Run analysis**. Scanline berjalan (percepat). Temuan muncul, badge **MUSE SPARK** dan jumlah token terlihat | "One request to Muse Spark with both images. The answer comes back as strict structured output: a category, a plain-language summary, how to verify it, and a box on each image." |
| 1:15–1:40 | Klik temuan: zoom design vs build, kotak putus-putus "should be here". Tekan **Accept**. Untuk temuan lain (jika ada yang keliru) tekan **Reject** dan isi alasan | "Click a finding and both images zoom to the same spot. A missing element is drawn where it should have been. Nothing counts until a person decides: accept the real defects, reject false alarms with a reason." |
| 1:40–1:58 | Tab **Swipe**, geser pembatas. Lalu **Layers 3D**: putar tumpukan, geser slider Overlay ↔ Exploded, tunjuk garis bukti | "Swipe across the two images, or pull them apart in 3D. The post links a finding on the design straight up to the build." |
| 1:58–2:10 | **Download Markdown**, buka file-nya sekilas | "Export the reviewed findings as Markdown for your pull request, or JSON for your tools." |
| 2:10–2:23 | `README.md` bagian Evaluation / tabel hasil (atau slide sederhana) | "We test on a self-made page with twelve labelled defects and a separate holdout page. A pixel diff finds two of twelve. *[isi angka Muse Spark setelah evaluasi]*" |
| 2:23–2:30 | `submission/video/framecheck-outro.mp4` | *(musik closing)* |

## Kalimat cadangan

- Jika ada temuan Muse Spark yang keliru: "When the model is wrong, the reviewer rejects it with a reason, and the reason goes into the export." Ini justru memperlihatkan nilai review manusia; jangan dipotong.
- Penutup alternatif: "FrameCheck: visual QA with evidence. Built on Meta Muse Spark."

## Jangan

- Jangan menampilkan API key, file `.env`, halaman Billing, atau email.
- Jangan menyebut hasil fixture/mock sebagai hasil AI. Jika memakai klip fixture, label `FIXTURE` harus terlihat.
- Jangan menampilkan aplikasi kantor atau data pribadi apa pun.
