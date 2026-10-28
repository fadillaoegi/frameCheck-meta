// Command htmlrender renders a local HTML file to a PNG (exact pixel size) or a
// PDF (A4/A3, backgrounds printed), for posters, thumbnails and documents.
//
//	go run ./cmd/htmlrender -page poster.html -out poster.png -w 1240 -h 1754 -scale 2
//	go run ./cmd/htmlrender -page brief.html -out brief.pdf -pdf A4
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	pagePath := flag.String("page", "", "HTML file")
	out := flag.String("out", "", "output .png or .pdf")
	w := flag.Int64("w", 1280, "viewport width (CSS px), PNG only")
	h := flag.Int64("h", 720, "viewport height (CSS px), PNG only")
	scale := flag.Float64("scale", 1, "device scale factor, PNG only")
	pdf := flag.String("pdf", "", "paper size for PDF output: A4 or A3")
	flag.Parse()
	abs, err := filepath.Abs(*pagePath)
	if err != nil || *out == "" {
		log.Fatal("need -page and -out")
	}
	u := url.URL{Scheme: "file", Path: abs}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	fontsReady := chromedp.Evaluate(`document.fonts.ready.then(() => new Promise(r => setTimeout(() => r(true), 400)))`, nil,
		func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) })

	var buf []byte
	if *pdf != "" {
		// Inches.
		pw, ph := 8.27, 11.69
		if *pdf == "A3" {
			pw, ph = 11.69, 16.54
		}
		err = chromedp.Run(ctx, chromedp.Navigate(u.String()), fontsReady, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().WithPrintBackground(true).WithPaperWidth(pw).WithPaperHeight(ph).
				WithMarginTop(0).WithMarginBottom(0).WithMarginLeft(0).WithMarginRight(0).WithPreferCSSPageSize(true).Do(ctx)
			return err
		}))
	} else {
		err = chromedp.Run(ctx, chromedp.EmulateViewport(*w, *h, chromedp.EmulateScale(*scale)), chromedp.Navigate(u.String()), fontsReady,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
				return err
			}))
	}
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, buf, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s (%d KB)\n", *out, len(buf)/1024)
}
