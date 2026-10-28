// Command screenrec records the FrameCheck demo walkthrough from a live app:
// a visible cursor moves on eased, slightly curved paths, text is typed key
// by key, and the page is captured with the DevTools screencast. It writes
// JPEG frames, an ffmpeg concat list (frame durations from capture timestamps,
// with the model wait sped up), and markers.json with the time of each scene
// in the final video, for the voice-over script.
//
//	go run ./cmd/screenrec -out /tmp/rec -design ../showcase/kopiku-design.png -build ../showcase/kopiku-build.png
//
// The app must be running (web on :5181, API on :8089). The run it starts is a
// real, billable Muse Spark request when the API uses the meta provider.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	cssW, cssH = 1536, 864 // rendered at 1.25× into 1920×1080
	scale      = 1.25
	waitSpeed  = 2.5 // how much faster the model wait plays in the video
)

type frame struct {
	t    float64
	path string
}

type marker struct {
	Name string  `json:"name"`
	At   float64 `json:"at"` // seconds in the final video
	raw  float64
}

type rec struct {
	ctx     context.Context
	rng     *rand.Rand
	x, y    float64
	mu      sync.Mutex
	frames  []frame
	out     string
	markers []marker
	down    bool // left button held, so moves report it
}

func main() {
	out := flag.String("out", "", "output directory")
	design := flag.String("design", "", "design image to upload")
	build := flag.String("build", "", "build image to upload")
	app := flag.String("app", "http://127.0.0.1:5181/#/", "app URL")
	probe := flag.Bool("probe", false, "capture one second of the landing page and print the frame size")
	winExtra := flag.Int("win-extra", 143, "extra window height (CSS px) for browser chrome, so the viewport is a full 1080 px")
	flag.Parse()
	if *out == "" || *design == "" || *build == "" {
		flag.Usage()
		os.Exit(2)
	}
	for _, p := range []*string{design, build, out} {
		abs, err := filepath.Abs(*p)
		if err != nil {
			log.Fatal(err)
		}
		*p = abs
	}
	if err := os.MkdirAll(filepath.Join(*out, "frames"), 0o755); err != nil {
		log.Fatal(err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// A real 1.25 device scale, so screencast frames are a sharp 1920×1080.
		chromedp.WindowSize(cssW, cssH+*winExtra),
		chromedp.Flag("force-device-scale-factor", fmt.Sprintf("%.2f", scale)),
		chromedp.Flag("hide-scrollbars", true),
	)
	actx, cancelA := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelA()
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	ctx, cancelT := context.WithTimeout(ctx, 10*time.Minute)
	defer cancelT()

	r := &rec{ctx: ctx, rng: rand.New(rand.NewSource(42)), x: cssW * 0.82, y: cssH * 0.9, out: *out}

	chromedp.ListenTarget(ctx, func(ev any) {
		if e, ok := ev.(*page.EventScreencastFrame); ok {
			go func() {
				data, err := base64.StdEncoding.DecodeString(e.Data)
				if err == nil {
					r.mu.Lock()
					p := filepath.Join(*out, "frames", fmt.Sprintf("f_%06d.jpg", len(r.frames)))
					r.frames = append(r.frames, frame{t: float64(e.Metadata.Timestamp.Time().UnixNano()) / 1e9, path: p})
					r.mu.Unlock()
					_ = os.WriteFile(p, data, 0o644)
				}
				_ = chromedp.Run(ctx, page.ScreencastFrameAck(e.SessionID))
			}()
		}
	})

	must(chromedp.Run(ctx,
		chromedp.EmulateViewport(cssW, cssH, chromedp.EmulateScale(scale)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(overlayJS).Do(ctx)
			return err
		}),
		page.SetInterceptFileChooserDialog(true),
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(filepath.Join(*out, "downloads")),
		chromedp.Navigate(*app),
		chromedp.Poll(`document.fonts.status === 'loaded' && !!document.querySelector('h1')`, nil, chromedp.WithPollingTimeout(30*time.Second)),
		chromedp.Sleep(800*time.Millisecond),
	))
	r.js(fmt.Sprintf(`__fc.cursor(%f, %f)`, r.x, r.y))
	must(chromedp.Run(ctx, page.StartScreencast().WithFormat(page.ScreencastFormatJpeg).WithQuality(90).
		WithMaxWidth(1920).WithMaxHeight(1080).WithEveryNthFrame(1)))
	time.Sleep(600 * time.Millisecond)

	if *probe {
		time.Sleep(time.Second)
	} else {
		r.script(*design, *build)
	}

	time.Sleep(1500 * time.Millisecond)
	must(chromedp.Run(ctx, page.StopScreencast()))
	time.Sleep(800 * time.Millisecond)
	r.finish()
}

// script is the walkthrough. Scene names become voice-over cues.
func (r *rec) script(design, build string) {
	// 1. Landing page.
	r.mark("home")
	r.pause(900)
	r.hoverEl(`__fc.q('div[style*="perspective"]')`, 1100)
	r.wiggle(2200)
	r.pause(500)

	r.mark("how-it-works")
	r.scrollToEl(`__fc.text('h2', 'Five steps')`, -90, 1400)
	r.hoverEl(`__fc.q('ol li', 2)`, 700)
	r.pause(1300)
	r.mark("categories")
	r.scrollToEl(`__fc.text('h2', 'Three kinds of defect')`, -90, 1200)
	r.hoverEl(`__fc.text('h3', 'Missing element')`, 600)
	r.hoverEl(`__fc.text('h3', 'Clipped text')`, 500)
	r.hoverEl(`__fc.text('h3', 'Layout mismatch')`, 500)
	r.pause(900)

	// 2. New project.
	r.mark("new-project")
	r.scrollToEl(`__fc.q('input[placeholder^="e.g. Checkout"]')`, -260, 1100)
	r.clickEl(`__fc.q('input[placeholder^="e.g. Checkout"]')`)
	r.typeText("Kopiku checkout")
	r.pause(400)
	r.clickEl(`__fc.text('button', 'Create project')`)
	r.waitFor(`!!__fc.text('h2', 'Upload a design and a build')`)
	r.pause(900)

	// 3. Upload and capture settings.
	r.mark("upload")
	r.hoverEl(`__fc.text('label', 'Design (reference)')`, 800)
	r.clickVisual()
	r.upload(0, design)
	r.pause(700)
	r.hoverEl(`__fc.text('label', 'Build (screenshot)')`, 700)
	r.clickVisual()
	r.upload(1, build)
	r.pause(900)

	r.mark("capture")
	r.fillField("Viewport width", "1440")
	r.pause(500)
	r.clickEl(`__fc.exact('button', 'Use 1')`)
	r.pause(500)
	r.fillField("Viewport height", "900")
	r.fillField("Page state", "checkout")
	r.pause(500)
	r.clickEl(`__fc.exact('button', 'Upload')`)
	r.waitFor(`document.querySelectorAll('select option').length >= 2 && !__fc.text('button', 'Uploading')`)
	r.pause(900)

	// 4. Pair and notes.
	r.mark("pair")
	r.hoverEl(`__fc.text('span', 'Design (reference)')`, 700)
	r.hoverEl(`__fc.text('span', 'Build (screenshot)')`, 600)
	r.clickEl(`__fc.q('textarea')`)
	r.typeText("Checkout page. The promo code row and the pay button must match the design.")
	r.pause(500)
	r.clickEl(`__fc.text('button', 'Create comparison')`)
	r.waitFor(`!!__fc.text('button', 'Run analysis')`)
	r.pause(1100)

	// 5. Analyse with Muse Spark.
	r.mark("analyse")
	r.hoverEl(`__fc.text('p', 'Next: run the analysis')`, 900)
	r.pause(600)
	r.clickEl(`__fc.text('button', 'Run analysis')`)
	r.pause(1200)
	r.mark("wait-start")
	r.js(`__fc.badge('⏩ Waiting for Muse Spark — sped up ' + ` + fmt.Sprintf("%q", fmt.Sprintf("%.1f×", waitSpeed)) + `)`)
	r.wiggleUntil(`document.querySelectorAll('article').length > 0 || !!__fc.text('p', 'analysis failed')`, 120*time.Second)
	r.js(`__fc.badge(null)`)
	r.mark("wait-end")
	r.pause(900)
	if r.bool(`!!__fc.text('p', 'analysis failed')`) {
		log.Fatal("the Muse Spark run failed; see the page")
	}

	// 6. Results and review.
	r.mark("results")
	r.hoverEl(`__fc.q('span[title$="model output"]')`, 900)
	r.pause(900)
	if r.bool(`!!__fc.text('span', 'tokens')`) { // mock runs have no token counts
		r.hoverEl(`__fc.text('span', 'tokens')`, 600)
		r.pause(900)
	}
	n := r.int(`document.querySelectorAll('article').length`)
	for i := 0; i < n; i++ {
		r.mark(fmt.Sprintf("finding-%d", i+1))
		// Keep the images in view: only scroll far enough to show the card's top.
		r.scrollToEl(fmt.Sprintf(`__fc.q('article', %d)`, i), -360, 900)
		r.clickEl(fmt.Sprintf(`__fc.q('article', %d).querySelector('p')`, i))
		r.pause(1400)
		r.hoverEl(fmt.Sprintf(`__fc.q('article', %d).querySelector('figure')`, i), 700)
		r.pause(1300)
		r.clickEl(fmt.Sprintf(`__fc.text('button', 'Accept', __fc.q('article', %d))`, i))
		r.pause(1100)
	}

	// 7. Swipe.
	r.mark("swipe")
	r.scrollToEl(`__fc.q('[role=tablist]')`, -84, 900)
	r.clickEl(`__fc.exact('button', 'Swipe')`)
	r.pause(700)
	r.dragRange(`__fc.q('input[aria-label^="Divider"]')`, []float64{0.5, 0.2, 0.85, 0.45})
	r.pause(600)

	// 8. Layers 3D.
	r.mark("layers")
	r.clickEl(`__fc.exact('button', 'Layers 3D')`)
	r.pause(900)
	r.scrollToEl(`__fc.q('.cursor-grab')`, -150, 900)
	r.pause(600)
	r.dragBy(`__fc.q('.cursor-grab')`, -170, 40, 1600)
	r.pause(500)
	r.dragRange(`__fc.q('input.accent-neon')`, []float64{1, 0.05, 1})
	r.pause(500)
	r.scrollToEl(`__fc.q('[role=tablist]')`, -84, 700)
	r.clickEl(`__fc.q('article', 0).querySelector('p')`)
	r.pause(1800)

	// 9. Export.
	r.mark("export")
	r.scrollTo(0, 900)
	r.clickEl(`__fc.text('button', 'Download Markdown')`)
	r.pause(900)
	r.js(`__fc.preview()`)
	r.pause(600)
	r.hoverEl(`__fc.q('#fc-preview pre')`, 900)
	r.smoothScrollIn(`#fc-preview pre`, 260, 2200)
	r.pause(1500)
	r.mark("end")
	r.js(`__fc.preview(false)`)
	r.pause(900)
}

// ---------- actions ----------

func (r *rec) mark(name string) {
	r.markers = append(r.markers, marker{Name: name, raw: float64(time.Now().UnixNano()) / 1e9})
	log.Printf("scene %s", name)
}

func (r *rec) pause(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

func (r *rec) js(expr string) {
	must(chromedp.Run(r.ctx, chromedp.Evaluate(expr, nil, await)))
}

func (r *rec) bool(expr string) bool {
	var b bool
	must(chromedp.Run(r.ctx, chromedp.Evaluate(expr, &b)))
	return b
}

func (r *rec) int(expr string) int {
	var n int
	must(chromedp.Run(r.ctx, chromedp.Evaluate(expr, &n)))
	return n
}

func await(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }

// center returns the viewport centre of an element expression.
func (r *rec) center(el string) (float64, float64) {
	var pt []float64
	must(chromedp.Run(r.ctx, chromedp.Evaluate(fmt.Sprintf(`(() => { const e = %s; if (!e) throw new Error('element not found: %s');
		const b = e.getBoundingClientRect(); return [b.x + b.width / 2, b.y + b.height / 2]; })()`, el, strings.ReplaceAll(el, "'", "")), &pt)))
	return pt[0], pt[1]
}

// moveTo glides the cursor on a gently curved, eased path at 60 Hz.
func (r *rec) moveTo(x, y float64, ms int) {
	x0, y0 := r.x, r.y
	dist := math.Hypot(x-x0, y-y0)
	if ms <= 0 {
		ms = int(math.Max(380, math.Min(1100, 260+dist*0.9)))
	}
	// Control point off the straight line, like a wrist arc.
	bend := (r.rng.Float64() - 0.5) * 0.35 * dist
	mx, my := (x0+x)/2, (y0+y)/2
	nx, ny := 0.0, 0.0
	if dist > 0 {
		nx, ny = -(y-y0)/dist, (x-x0)/dist
	}
	cx, cy := mx+nx*bend, my+ny*bend
	steps := ms * 60 / 1000
	start := time.Now()
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		e := t * t * t * (t*(t*6-15) + 10) // smootherstep
		px := (1-e)*(1-e)*x0 + 2*(1-e)*e*cx + e*e*x
		py := (1-e)*(1-e)*y0 + 2*(1-e)*e*cy + e*e*y
		r.mouse(input.MouseMoved, px, py)
		if d := time.Until(start.Add(time.Duration(i) * time.Second / 60)); d > 0 {
			time.Sleep(d)
		}
	}
	r.x, r.y = x, y
}

func (r *rec) mouse(t input.MouseType, x, y float64) {
	ev := input.DispatchMouseEvent(t, x, y)
	if t == input.MousePressed || t == input.MouseReleased {
		ev = ev.WithButton(input.Left).WithClickCount(1)
	} else if t == input.MouseMoved && r.down {
		ev = ev.WithButton(input.Left).WithButtons(1)
	}
	must(chromedp.Run(r.ctx, ev))
}

func (r *rec) hoverEl(el string, ms int) {
	x, y := r.center(el)
	r.moveTo(x+r.jitter(6), y+r.jitter(4), ms)
}

func (r *rec) clickEl(el string) {
	r.ensureVisible(el)
	r.hoverEl(el, 0)
	r.pause(120 + r.rng.Intn(140))
	r.mouse(input.MousePressed, r.x, r.y)
	r.pause(70 + r.rng.Intn(50))
	r.mouse(input.MouseReleased, r.x, r.y)
	r.pause(250)
}

// clickVisual shows a click without sending it (used where a real click would
// open the native file chooser).
func (r *rec) clickVisual() {
	r.pause(150)
	r.js(fmt.Sprintf(`__fc.ripple(%f, %f)`, r.x, r.y))
	r.pause(200)
}

// upload sets the file on the index-th file input, as the chooser would.
func (r *rec) upload(index int, file string) {
	r.js(fmt.Sprintf(`document.querySelectorAll('input[type=file]')[%d].setAttribute('data-rec-upload', '%d')`, index, index))
	must(chromedp.Run(r.ctx, chromedp.SetUploadFiles(fmt.Sprintf(`input[data-rec-upload="%d"]`, index), []string{file}, chromedp.ByQuery)))
}

func (r *rec) typeText(s string) {
	for _, ch := range s {
		must(chromedp.Run(r.ctx, input.InsertText(string(ch))))
		d := 45 + r.rng.Intn(70)
		if ch == ' ' {
			d += 40
		}
		if ch == '.' || ch == ',' {
			d += 160
		}
		r.pause(d)
	}
}

// fillField selects a labelled input's text and types over it.
func (r *rec) fillField(label, value string) {
	el := fmt.Sprintf(`__fc.text('label', %q).querySelector('input')`, label)
	r.clickEl(el)
	r.js(el + `.select()`)
	r.pause(150)
	r.typeText(value)
	r.pause(250)
}

func (r *rec) ensureVisible(el string) {
	var inView bool
	must(chromedp.Run(r.ctx, chromedp.Evaluate(fmt.Sprintf(`(() => { const b = (%s).getBoundingClientRect(); return b.top > 60 && b.bottom < innerHeight - 20; })()`, el), &inView)))
	if !inView {
		r.scrollToEl(el, -220, 900)
	}
}

func (r *rec) scrollToEl(el string, offset float64, ms int) {
	var y float64
	must(chromedp.Run(r.ctx, chromedp.Evaluate(fmt.Sprintf(`(%s).getBoundingClientRect().top + scrollY + %f`, el, offset), &y)))
	r.scrollTo(y, ms)
}

func (r *rec) scrollTo(y float64, ms int) {
	r.js(fmt.Sprintf(`__fc.scroll(%f, %d)`, math.Max(0, y), ms))
}

func (r *rec) smoothScrollIn(sel string, by float64, ms int) {
	r.js(fmt.Sprintf(`__fc.scrollIn(%q, %f, %d)`, sel, by, ms))
}

func (r *rec) wiggle(ms int) {
	end := time.Now().Add(time.Duration(ms) * time.Millisecond)
	bx, by := r.x, r.y
	for time.Now().Before(end) {
		r.moveTo(bx+r.jitter(140), by+r.jitter(90), 700+r.rng.Intn(400))
	}
}

// wiggleUntil drifts the cursor idly while waiting for a condition.
func (r *rec) wiggleUntil(cond string, limit time.Duration) {
	end := time.Now().Add(limit)
	bx, by := r.x, r.y
	for !r.bool(cond) {
		if time.Now().After(end) {
			log.Fatal("timed out waiting for: " + cond)
		}
		r.moveTo(bx+r.jitter(60), by+r.jitter(40), 900+r.rng.Intn(500))
		r.pause(300)
	}
}

func (r *rec) waitFor(cond string) {
	must(chromedp.Run(r.ctx, chromedp.Poll(cond, nil, chromedp.WithPollingTimeout(30*time.Second))))
}

// dragRange drags across a range input to the given fractions of its width.
func (r *rec) dragRange(el string, stops []float64) {
	var b []float64
	must(chromedp.Run(r.ctx, chromedp.Evaluate(fmt.Sprintf(`(() => { const r = (%s).getBoundingClientRect(); return [r.x, r.y, r.width, r.height]; })()`, el), &b)))
	at := func(f float64) (float64, float64) { return b[0] + 10 + f*(b[2]-20), b[1] + b[3]/2 }
	x, y := at(stops[0])
	r.moveTo(x, y, 0)
	r.pause(200)
	r.down = true
	r.mouse(input.MousePressed, r.x, r.y)
	for _, f := range stops[1:] {
		x, y := at(f)
		r.moveTo(x, y+r.jitter(3), 900+r.rng.Intn(500))
		r.pause(250)
	}
	r.mouse(input.MouseReleased, r.x, r.y)
	r.down = false
}

func (r *rec) dragBy(el string, dx, dy float64, ms int) {
	r.hoverEl(el, 0)
	r.pause(200)
	r.down = true
	r.mouse(input.MousePressed, r.x, r.y)
	r.moveTo(r.x+dx, r.y+dy, ms)
	r.moveTo(r.x-dx*0.4, r.y-dy*0.3, ms/2)
	r.mouse(input.MouseReleased, r.x, r.y)
	r.down = false
}

func (r *rec) jitter(n float64) float64 { return (r.rng.Float64()*2 - 1) * n }

// ---------- output ----------

func (r *rec) finish() {
	r.mu.Lock()
	defer r.mu.Unlock()
	sort.Slice(r.frames, func(i, j int) bool { return r.frames[i].t < r.frames[j].t })
	if len(r.frames) < 2 {
		log.Fatal("no frames captured")
	}
	var ws, we float64
	for _, m := range r.markers {
		switch m.Name {
		case "wait-start":
			ws = m.raw
		case "wait-end":
			we = m.raw
		}
	}
	t0 := r.frames[0].t
	// outTime maps capture time to video time, compressing the model wait.
	outTime := func(t float64) float64 {
		switch {
		case ws == 0 || t <= ws:
			return t - t0
		case t <= we:
			return ws - t0 + (t-ws)/waitSpeed
		default:
			return ws - t0 + (we-ws)/waitSpeed + (t - we)
		}
	}
	var list strings.Builder
	for i, f := range r.frames {
		next := f.t + 1.0/30
		if i+1 < len(r.frames) {
			next = r.frames[i+1].t
		}
		d := outTime(next) - outTime(f.t)
		if d <= 0 {
			continue
		}
		fmt.Fprintf(&list, "file '%s'\nduration %.4f\n", f.path, d)
	}
	last := r.frames[len(r.frames)-1].path
	fmt.Fprintf(&list, "file '%s'\n", last) // concat demuxer needs the last file repeated
	must(os.WriteFile(filepath.Join(r.out, "frames.txt"), []byte(list.String()), 0o644))

	for i := range r.markers {
		r.markers[i].At = math.Round(outTime(r.markers[i].raw)*10) / 10
	}
	j, _ := json.MarshalIndent(map[string]any{
		"duration": math.Round(outTime(r.frames[len(r.frames)-1].t)*10) / 10,
		"frames":   len(r.frames),
		"markers":  r.markers,
	}, "", "  ")
	must(os.WriteFile(filepath.Join(r.out, "markers.json"), j, 0o644))
	fmt.Printf("captured %d frames; video %.1fs\n", len(r.frames), outTime(r.frames[len(r.frames)-1].t))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
