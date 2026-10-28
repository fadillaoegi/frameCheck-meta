// Command overlays renders HTML snippets to transparent 1920×1080 PNGs, for
// burning captions and labels into a video when ffmpeg has no text filters.
//
//	go run ./cmd/overlays -spec overlays.json
//
// The spec is a JSON array of {"out": "path.png", "html": "<!doctype html>..."}.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type item struct {
	Out  string `json:"out"`
	HTML string `json:"html"`
}

func main() {
	spec := flag.String("spec", "", "JSON spec file")
	flag.Parse()
	raw, err := os.ReadFile(*spec)
	if err != nil {
		log.Fatal(err)
	}
	var items []item
	if err := json.Unmarshal(raw, &items); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080),
		emulation.SetDefaultBackgroundColorOverride().WithColor(&cdp.RGBA{R: 0, G: 0, B: 0, A: 0}),
	); err != nil {
		log.Fatal(err)
	}
	for _, it := range items {
		var buf []byte
		url := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(it.HTML))
		err := chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
				return err
			}),
		)
		if err != nil {
			log.Fatalf("%s: %v", it.Out, err)
		}
		if err := os.WriteFile(it.Out, buf, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("rendered %d overlays\n", len(items))
}
