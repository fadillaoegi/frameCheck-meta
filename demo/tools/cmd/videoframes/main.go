// Command videoframes renders a deterministic HTML animation to PNG frames.
//
// The page must define window.render(t) and set window.__ready and
// window.__duration (seconds). The page's directory is served over a local
// HTTP server so ES modules load.
//
//	go run ./cmd/videoframes -page ../../submission/video/src/intro.html -out /tmp/frames
//	go run ./cmd/videoframes -page ... -out /tmp/stills -at 0,2,3.5,6   # stills only
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	pagePath := flag.String("page", "", "HTML file to render")
	out := flag.String("out", "", "output directory for PNG frames")
	fps := flag.Int("fps", 30, "frames per second")
	width := flag.Int64("w", 1920, "viewport width")
	height := flag.Int64("h", 1080, "viewport height")
	at := flag.String("at", "", "comma-separated times in seconds: render only these stills")
	query := flag.String("query", "", "query string passed to the page (without '?')")
	flag.Parse()
	if *pagePath == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	abs, err := filepath.Abs(*pagePath)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	go http.Serve(ln, http.FileServer(http.Dir(filepath.Dir(abs))))
	url := fmt.Sprintf("http://%s/%s", ln.Addr(), filepath.Base(abs))
	if *query != "" {
		url += "?" + *query
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	var duration float64
	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(*width, *height),
		chromedp.Navigate(url),
		chromedp.Poll(`window.__ready === true`, nil, chromedp.WithPollingTimeout(30*time.Second)),
		chromedp.Evaluate(`window.__duration`, &duration),
	)
	if err != nil {
		log.Fatalf("load %s: %v", url, err)
	}

	var times []float64
	if *at != "" {
		for _, s := range strings.Split(*at, ",") {
			v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				log.Fatalf("bad -at value %q", s)
			}
			times = append(times, v)
		}
	} else {
		n := int(duration * float64(*fps))
		for i := 0; i < n; i++ {
			times = append(times, float64(i)/float64(*fps))
		}
	}

	start := time.Now()
	for i, t := range times {
		var buf []byte
		err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`window.render(%f)`, t), nil),
			// Wait two animation frames so the compositor has applied the new transforms.
			chromedp.Evaluate(`new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => r(true))))`, nil,
				func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
				return err
			}),
		)
		if err != nil {
			log.Fatalf("frame %d (t=%.3f): %v", i, t, err)
		}
		name := fmt.Sprintf("frame_%05d.png", i)
		if *at != "" {
			name = fmt.Sprintf("still_%06.2f.png", t)
		}
		if err := os.WriteFile(filepath.Join(*out, name), buf, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("rendered %d frames (%.1fs of animation) in %s\n", len(times), duration, time.Since(start).Round(time.Second))
}
