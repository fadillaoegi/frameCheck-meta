# Voice-over FrameCheck demo

Untuk video **`video/framecheck-demo-full.mp4`** (2:12): intro 0:00–0:07, rekaman layar 0:07–2:05, closing 2:05–2:12.

Waktu di bawah adalah **waktu di video penuh**. Setiap kalimat sudah diuji muat di jendelanya dengan pembacaan ±165 kata/menit (tempo bicara santai). Mulai bicara di "Mulai"; boleh selesai lebih awal, jangan melewati "Selesai".

**Panduan timing:** `video/build/framecheck-demo-guide-tts.mp4` adalah video yang sama dengan suara robot (TTS) di posisi yang tepat. Tonton sambil membaca naskah untuk merasakan ritmenya, lalu rekam suaramu sendiri. **Jangan submit versi TTS.**

| # | Mulai | Selesai | Di layar | Voice-over (EN) |
| --- | --- | --- | --- | --- |
| — | 0:00 | 0:07 | Intro 3D: lempeng desain, bingkai build, titik merah, logo | *(musik saja)* |
| 1 | 0:07.8 | 0:12.6 | Halaman depan, kursor di atas tumpukan 3D | Every front-end developer knows this question: does the build really match the design? |
| 2 | 0:12.8 | 0:16.0 | Scroll ke "Five steps" | FrameCheck answers it in five steps. |
| 3 | 0:15.3 | 0:19.7 | Tiga kartu kategori | Three kinds of defect: missing, clipped, misplaced. |
| 4 | 0:19.9 | 0:25.3 | Mengetik "Kopiku checkout", Create project | Let's try it on a checkout page I designed for a coffee subscription. |
| 5 | 0:25.4 | 0:29.3 | Memilih gambar desain dan build | I drop in the design export, and a screenshot of the build. |
| 6 | 0:29.5 | 0:39.6 | Viewport 1440, klik "Use 1", tinggi 900, state "checkout", Upload | Both must be captured the same way. FrameCheck reads the image size and suggests the pixel ratio, so the two line up pixel for pixel. |
| 7 | 0:39.8 | 0:52.1 | Slot Design/Build terisi, mengetik catatan, Create comparison | I pair them, and add a note about what matters on this page. Notes give the model context, but they can't change its rules or its output format. |
| 8 | 0:52.3 | 0:56.4 | Klik "Run analysis" | Now, one request to Meta's Muse Spark, with both images. |
| 9 | 0:56.5 | 1:02.7 | Scanline menunggu (dipercepat 2,5×, ada label) | It answers in strict, structured JSON: a category, a summary, and a box on each image. |
| 10 | 1:02.8 | 1:06.0 | Badge **MUSE SPARK**, jumlah token | Three findings, with the model and token cost. |
| 11 | 1:06.1 | 1:14.6 | Temuan 1: promo code hilang, zoom, kotak putus-putus, Accept | The promo code row is missing. Both images zoom to the same spot, with a dashed box where it should be. Accept. |
| 12 | 1:14.7 | 1:22.1 | Temuan 2: tombol Pay lebih sempit, Accept | The pay button is narrower than designed, boxed on both images, so the difference is easy to check. |
| 13 | 1:22.2 | 1:29.8 | Temuan 3: judul produk terpotong, Accept | And the first product title is cut off. Nothing counts until a person decides, so each one gets a human verdict. |
| 14 | 1:29.9 | 1:40.3 | Tab Swipe, pembatas digeser | Swipe across the two images to see each change at the same position: the promo row vanishes, the button shrinks, the title is cut. |
| 15 | 1:40.4 | 1:57.4 | Layers 3D: diputar, Overlay ↔ Exploded, garis bukti | Or pull them apart in 3D. The build floats above the design. Flatten it into an overlay, or turn the stack and follow a finding from the design straight up to the build. Every box is anchored to the original pixels, so it lines up at any zoom. |
| 16 | 1:57.5 | 2:05.3 | Download Markdown, pratinjau file laporan | Finally, export the reviewed findings as Markdown for the pull request, with the model, the tokens, and every human decision. |
| 17 | 2:08.2 | 2:12.0 | Closing: logo FrameCheck | FrameCheck. Visual QA, with evidence. |

## Cara merekam dan menggabungkan

1. Rekam per baris (lebih mudah diulang) atau sekali jalan sambil menonton `framecheck-demo-guide-tts.mp4` tanpa suara. Mikrofon dekat, ruangan senyap, bicara santai dan jelas.
2. Di editor video (CapCut, DaVinci Resolve, iMovie, Premiere), taruh `framecheck-demo-full.mp4` di timeline, lalu tempatkan klip suaramu di waktu "Mulai".
3. Musik latar di rekaman layar sengaja pelan (puncak −20 dB). Jika suaramu masih tertutup, turunkan audio video 3–6 dB selama kamu bicara.
4. Export 1080p 30fps, H.264, AAC.

## Fakta di balik naskah (semuanya benar untuk rekaman ini)

- Model: `muse-spark-1.3`, satu request berisi dua gambar, output `json_schema` ketat.
- Hasil di video: 3 temuan (missing element, layout mismatch, clipped text), semuanya cocok dengan cacat yang sengaja dibuat (`demo/showcase/truth.json`), semuanya di-*accept* di video.
- Desain "Kopiku" dan kedua gambarnya dibuat sendiri (`demo/site/kopiku.html`); bukan file Figma pihak lain.
- Bagian menunggu model dipercepat 2,5× dan diberi label di layar.
