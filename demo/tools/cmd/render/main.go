// Command render screenshots one page variant at an exact viewport and prints
// the bounding boxes of its data-fc components, for showcase images that hold
// several defects at once (cmd/capture allows one defect per case).
//
//	go run ./cmd/render -page ../site/kopiku.html -variant showcase -out ../showcase/build.png
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	runtime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	pagePath := flag.String("page", "", "HTML page")
	variant := flag.String("variant", "reference", "?variant= value")
	out := flag.String("out", "", "output PNG")
	w := flag.Int64("w", 1440, "viewport width (CSS px)")
	h := flag.Int64("h", 900, "viewport height (CSS px)")
	flag.Parse()
	abs, err := filepath.Abs(*pagePath)
	if err != nil || *out == "" {
		log.Fatal("need -page and -out")
	}
	u := url.URL{Scheme: "file", Path: abs, RawQuery: "variant=" + url.QueryEscape(*variant)}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var buf []byte
	var rects map[string][4]float64
	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(*w, *h),
		chromedp.Navigate(u.String()),
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`Object.fromEntries([...document.querySelectorAll('[data-fc]')].map(e => {
			const r = e.getBoundingClientRect(); return [e.dataset.fc, [Math.round(r.x), Math.round(r.y), Math.round(r.width), Math.round(r.height)]]; }))`, &rects),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, buf, 0o644); err != nil {
		log.Fatal(err)
	}
	j, _ := json.Marshal(rects)
	fmt.Println(string(j))
}
