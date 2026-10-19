package popup

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	_ "golang.org/x/image/webp"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/api"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// UI constants (in device-independent pixels) for a 1920x1080 reference design.
const (
	logoMaxWidth   = 160
	closeBtnSize   = 40
	closeBtnMargin = 24
	logoMargin     = 24
	panelMargin    = 32
	panelPadding   = 24
	panelRadius    = 16
	linkIconSize   = 14
	fadeInDuration = 0.5 // seconds
)

// announcementGame renders the announcement popup.
type announcementGame struct {
	ann       models.AgentAnnouncement
	img       *ebiten.Image
	imgMu     sync.RWMutex
	logo      *ebiten.Image
	apiClient *api.Client

	screenWidth  int
	screenHeight int

	closeRect rect
	linkRect  rect
	linkURL   string
	alpha     float64 // fade-in progress, 0..1

	titleFace       font.Face
	descriptionFace font.Face
	linkFace        font.Face
}

type rect struct {
	x, y, w, h int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (g *announcementGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	if g.alpha < 1.0 {
		g.alpha += 1.0 / (float64(ebiten.TPS()) * fadeInDuration)
		if g.alpha > 1.0 {
			g.alpha = 1.0
		}
	}

	x, y := ebiten.CursorPosition()
	hoverClose := g.closeRect.contains(x, y)
	hoverLink := g.linkRect.contains(x, y) && g.linkURL != ""

	if hoverClose || hoverLink {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if hoverClose {
			return ebiten.Termination
		}
		if hoverLink {
			openURL(g.linkURL)
		}
	}

	return nil
}

func (g *announcementGame) Draw(screen *ebiten.Image) {
	g.screenWidth, g.screenHeight = screen.Size()

	// Draw background image covering the whole window (cover scaling).
	g.drawBackground(screen)

	// Draw logo at top-left.
	g.drawLogo(screen)

	// Draw close button at top-right.
	g.drawCloseButton(screen)

	// Draw info panel at bottom-left.
	g.drawInfoPanel(screen)
}

func (g *announcementGame) drawBackground(screen *ebiten.Image) {
	g.imgMu.RLock()
	img := g.img
	g.imgMu.RUnlock()

	if img == nil {
		screen.Fill(color.Black)
		return
	}

	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	if iw == 0 || ih == 0 {
		screen.Fill(color.Black)
		return
	}

	// Cover scaling: fill entire window, cropping edges if aspect ratios differ.
	scale := max(float64(g.screenWidth)/float64(iw), float64(g.screenHeight)/float64(ih))
	dw := float64(iw) * scale
	dh := float64(ih) * scale
	dx := (float64(g.screenWidth) - dw) / 2
	dy := (float64(g.screenHeight) - dh) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(dx, dy)
	op.ColorScale.Scale(1, 1, 1, float32(g.alpha))
	screen.DrawImage(img, op)
}

func (g *announcementGame) drawLogo(screen *ebiten.Image) {
	if g.logo == nil {
		return
	}
	lw, lh := g.logo.Bounds().Dx(), g.logo.Bounds().Dy()
	if lw == 0 || lh == 0 {
		return
	}

	scale := 1.0
	if lw > logoMaxWidth {
		scale = float64(logoMaxWidth) / float64(lw)
	}
	w := int(float64(lw) * scale)
	h := int(float64(lh) * scale)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(logoMargin), float64(logoMargin))
	op.ColorScale.Scale(1, 1, 1, float32(g.alpha))
	screen.DrawImage(g.logo, op)

	_ = h
	_ = w
}

func (g *announcementGame) drawCloseButton(screen *ebiten.Image) {
	x := g.screenWidth - closeBtnSize - closeBtnMargin
	y := closeBtnMargin
	g.closeRect = rect{x: x, y: y, w: closeBtnSize, h: closeBtnSize}

	btn := ebiten.NewImage(closeBtnSize, closeBtnSize)
	// Black background circle.
	cx, cy, r := closeBtnSize/2, closeBtnSize/2, closeBtnSize/2
	for py := 0; py < closeBtnSize; py++ {
		for px := 0; px < closeBtnSize; px++ {
			dx := px - cx
			dy := py - cy
			if dx*dx+dy*dy <= r*r {
				btn.Set(px, py, color.NRGBA{R: 0, G: 0, B: 0, A: 200})
			}
		}
	}

	// White X.
	padding := closeBtnSize / 4
	c := color.White
	for i := 0; i < closeBtnSize-2*padding; i++ {
		btn.Set(padding+i, padding+i, c)
		btn.Set(closeBtnSize-padding-i-1, padding+i, c)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.Scale(1, 1, 1, float32(g.alpha))
	screen.DrawImage(btn, op)
}

func (g *announcementGame) drawInfoPanel(screen *ebiten.Image) {
	titleText := g.ann.Title
	descText := ""
	if g.ann.Description != nil {
		descText = *g.ann.Description
	}
	g.linkURL = ""
	if g.ann.LinkURL != nil {
		g.linkURL = *g.ann.LinkURL
	}

	if titleText == "" && descText == "" && g.linkURL == "" {
		return
	}

	// Panel spans the full window width respecting horizontal margins.
	panelWidth := g.screenWidth - panelMargin*2
	if panelWidth < 320 {
		panelWidth = 320
	}
	availableTextWidth := panelWidth - panelPadding*2

	titleLines := wrapText(titleText, g.titleFace, availableTextWidth)
	descLines := wrapText(descText, g.descriptionFace, availableTextWidth)

	titleLineHeight := g.titleFace.Metrics().Height.Ceil()
	descLineHeight := g.descriptionFace.Metrics().Height.Ceil()
	linkLineHeight := g.linkFace.Metrics().Height.Ceil()

	linkText := "Conocer mas"

	panelHeight := panelPadding*2 + len(titleLines)*titleLineHeight + len(descLines)*descLineHeight
	if g.linkURL != "" {
		panelHeight += 12 + linkLineHeight
	}
	if len(titleLines) > 0 && len(descLines) > 0 {
		panelHeight += 8
	}

	x := panelMargin
	y := g.screenHeight - panelHeight - panelMargin

	// Draw rounded panel.
	panel := drawRoundedRect(panelWidth, panelHeight, panelRadius, applyAlpha(color.NRGBA{R: 0, G: 0, B: 0, A: 200}, g.alpha))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(panel, op)

	// Draw title.
	textX := x + panelPadding
	textY := y + panelPadding
	drawer := &font.Drawer{
		Dst:  screen,
		Src:  image.NewUniform(applyAlpha(color.NRGBA{R: 255, G: 255, B: 255, A: 255}, g.alpha)),
		Face: g.titleFace,
	}
	for _, line := range titleLines {
		drawer.Dot = fixed.P(textX, textY+g.titleFace.Metrics().Ascent.Ceil())
		drawer.DrawString(line)
		textY += titleLineHeight
	}

	// Draw description.
	if len(descLines) > 0 {
		textY += 8
		drawer.Src = image.NewUniform(applyAlpha(color.NRGBA{R: 220, G: 220, B: 220, A: 255}, g.alpha))
		drawer.Face = g.descriptionFace
		for _, line := range descLines {
			drawer.Dot = fixed.P(textX, textY+g.descriptionFace.Metrics().Ascent.Ceil())
			drawer.DrawString(line)
			textY += descLineHeight
		}
	}

	// Draw link.
	if g.linkURL != "" {
		textY += 12
		linkY := textY
		linkW := textWidth([]string{linkText}, g.linkFace) + linkIconSize + 8
		g.linkRect = rect{x: textX, y: linkY, w: linkW, h: linkLineHeight}

		// Link icon (small square with arrow-like shape).
		g.drawLinkIcon(screen, textX, linkY+linkLineHeight/2-linkIconSize/2)

		drawer.Src = image.NewUniform(applyAlpha(color.NRGBA{R: 96, G: 165, B: 250, A: 255}, g.alpha))
		drawer.Face = g.linkFace
		drawer.Dot = fixed.P(textX+linkIconSize+8, linkY+g.linkFace.Metrics().Ascent.Ceil())
		drawer.DrawString(linkText)
	}
}

func (g *announcementGame) drawLinkIcon(screen *ebiten.Image, x, y int) {
	icon := ebiten.NewImage(linkIconSize, linkIconSize)
	c := applyAlpha(color.NRGBA{R: 96, G: 165, B: 250, A: 255}, g.alpha)
	// Draw a simple arrow-like box outline.
	for i := 2; i < linkIconSize-2; i++ {
		icon.Set(i, 2, c)
		icon.Set(2, i, c)
		icon.Set(linkIconSize-3, i, c)
		icon.Set(i, linkIconSize-3, c)
	}
	// Arrow head.
	for i := 0; i < 5; i++ {
		icon.Set(linkIconSize-4-i, i+3, c)
		icon.Set(linkIconSize-4-i, i+4, c)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(icon, op)
}

func (g *announcementGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// Show displays the announcement popup. The image is downloaded asynchronously
// so the window appears immediately; the background stays black until the image
// is ready.
func Show(ctx context.Context, apiClient *api.Client, ann models.AgentAnnouncement) error {
	if ann.ImageDownloadURL == "" {
		return fmt.Errorf("announcement has no image download URL")
	}

	fmt.Fprintf(os.Stderr, "popup: showing announcement %s\n", ann.Identifier)
	fmt.Fprintf(os.Stderr, "popup: image URL %s\n", ann.ImageDownloadURL)

	logoPath, err := getLogoPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "popup: logo not found: %v\n", err)
	}
	var logoImg *ebiten.Image
	if logoPath != "" {
		logoImg, _, err = ebitenutil.NewImageFromFile(logoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "popup: load logo failed: %v\n", err)
			logoImg = nil
		}
	}

	playNotificationSound()

	screenWidth, screenHeight := ebiten.ScreenSizeInFullscreen()
	windowWidth := int(float64(screenWidth) * 0.8)
	windowHeight := int(float64(screenHeight) * 0.8)
	posX := (screenWidth - windowWidth) / 2
	posY := (screenHeight - windowHeight) / 2

	ebiten.SetWindowTitle("EMIR Anuncio")
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowPosition(posX, posY)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetVsyncEnabled(true)

	game := &announcementGame{
		ann:             ann,
		logo:            logoImg,
		apiClient:       apiClient,
		titleFace:       loadFontFace(26, true),
		descriptionFace: loadFontFace(16, false),
		linkFace:        loadFontFace(15, false),
	}

	// Download the image asynchronously so the window opens immediately.
	go func() {
		fmt.Fprintln(os.Stderr, "popup: starting image download")
		imgPath, err := downloadImage(ctx, ann.ImageDownloadURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "popup: image download failed: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "popup: image saved to %s\n", imgPath)
		defer os.Remove(imgPath)

		img, _, err := ebitenutil.NewImageFromFile(imgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "popup: load image failed: %v\n", err)
			return
		}

		game.imgMu.Lock()
		game.img = img
		game.imgMu.Unlock()
		fmt.Fprintln(os.Stderr, "popup: image ready")
	}()

	// Report impression right before the window is shown.
	if apiClient != nil {
		if err := apiClient.ReportAnnouncementImpression(ctx, ann.Identifier); err != nil {
			fmt.Fprintf(os.Stderr, "popup: failed to report impression: %v\n", err)
		}
	}

	fmt.Fprintln(os.Stderr, "popup: running game loop")
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		return fmt.Errorf("run game: %w", err)
	}
	fmt.Fprintln(os.Stderr, "popup: game loop ended")
	return nil
}

func assetPath(name string) string {
	agentPath, err := os.Executable()
	if err != nil {
		return filepath.Join("assets", name)
	}
	return filepath.Join(filepath.Dir(agentPath), "assets", name)
}

func getLogoPath() (string, error) {
	logoPath := assetPath("logo.png")
	if _, err := os.Stat(logoPath); err != nil {
		return "", err
	}
	return logoPath, nil
}

func playNotificationSound() {
	go func() {
		const sampleRate = 48000
		audioContext := audio.NewContext(sampleRate)

		path := assetPath("notification.mp3")
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "notification sound not found: %v\n", err)
			return
		}
		defer f.Close()

		d, err := mp3.DecodeWithSampleRate(sampleRate, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decode notification sound: %v\n", err)
			return
		}

		p, err := audioContext.NewPlayer(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create audio player: %v\n", err)
			return
		}
		defer p.Close()

		p.Play()
		for p.IsPlaying() {
			time.Sleep(25 * time.Millisecond)
		}
	}()
}

func loadFontFace(size float64, bold bool) font.Face {
	var paths []string
	switch runtime.GOOS {
	case "windows":
		if bold {
			paths = []string{
				`C:\Windows\Fonts\segoeuib.ttf`,
				`C:\Windows\Fonts\arialbd.ttf`,
				`C:\Windows\Fonts\tahomabd.ttf`,
				`C:\Windows\Fonts\verdanab.ttf`,
			}
		} else {
			paths = []string{
				`C:\Windows\Fonts\segoeui.ttf`,
				`C:\Windows\Fonts\arial.ttf`,
				`C:\Windows\Fonts\tahoma.ttf`,
				`C:\Windows\Fonts\verdana.ttf`,
			}
		}
	case "darwin":
		if bold {
			paths = []string{
				"/Library/Fonts/Arial Bold.ttf",
				"/System/Library/Fonts/Helvetica.ttc",
				"/System/Library/Fonts/HelveticaNeue.ttc",
			}
		} else {
			paths = []string{
				"/Library/Fonts/Arial.ttf",
				"/System/Library/Fonts/Helvetica.ttc",
				"/System/Library/Fonts/HelveticaNeue.ttc",
			}
		}
	default:
		if bold {
			paths = []string{
				"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
				"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
				"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
				"/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf",
			}
		} else {
			paths = []string{
				"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
				"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
				"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
				"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
			}
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		ttf, err := opentype.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse font %s: %v\n", path, err)
			continue
		}
		face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
			Size:    size,
			DPI:     96,
			Hinting: font.HintingFull,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create face for %s: %v\n", path, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "loaded font: %s (size %.0f, bold=%v)\n", path, size, bold)
		return face
	}

	fmt.Fprintln(os.Stderr, "no system font found; falling back to basic font (limited character support)")
	return basicfont.Face7x13
}

func drawRoundedRect(width, height, radius int, c color.Color) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var inCorner bool
			switch {
			case x < radius && y < radius:
				inCorner = (x-radius)*(x-radius)+(y-radius)*(y-radius) > radius*radius
			case x >= width-radius && y < radius:
				cx := width - radius
				inCorner = (x-cx)*(x-cx)+(y-radius)*(y-radius) > radius*radius
			case x < radius && y >= height-radius:
				cy := height - radius
				inCorner = (x-radius)*(x-radius)+(y-cy)*(y-cy) > radius*radius
			case x >= width-radius && y >= height-radius:
				cx := width - radius
				cy := height - radius
				inCorner = (x-cx)*(x-cx)+(y-cy)*(y-cy) > radius*radius
			}
			if inCorner {
				img.Set(x, y, color.Transparent)
			}
		}
	}

	return ebiten.NewImageFromImage(img)
}

func textWidth(lines []string, face font.Face) int {
	maxW := 0
	for _, line := range lines {
		w := font.MeasureString(face, line).Ceil()
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	if text == "" {
		return nil
	}
	words := splitWords(text)
	var lines []string
	var current string
	for _, word := range words {
		test := current
		if test != "" {
			test += " "
		}
		test += word
		if font.MeasureString(face, test).Ceil() > maxWidth && current != "" {
			lines = append(lines, current)
			current = word
		} else {
			current = test
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitWords(text string) []string {
	var words []string
	var start int
	for i, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if i > start {
				words = append(words, text[start:i])
			}
			if r == '\n' {
				words = append(words, "\n")
			}
			start = i + 1
		}
	}
	if start < len(text) {
		words = append(words, text[start:])
	}
	return words
}

func downloadImage(ctx context.Context, url string) (string, error) {
	// Large announcement images can take a while on slow connections.
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "popup: GET %s\n", url)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "popup: request failed: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	fmt.Fprintf(os.Stderr, "popup: response status %d, content-length %s\n", resp.StatusCode, resp.Header.Get("Content-Length"))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	imgPath := filepath.Join(os.TempDir(), fmt.Sprintf("emir-announcement-%d.img", time.Now().UnixNano()))
	out, err := os.Create(imgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "popup: create temp file failed: %v\n", err)
		return "", err
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "popup: copy body failed after %d bytes: %v\n", written, err)
		return "", err
	}
	fmt.Fprintf(os.Stderr, "popup: downloaded %d bytes to %s\n", written, imgPath)
	return imgPath, nil
}

func openURL(link string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", link)
	case "darwin":
		cmd = exec.Command("open", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	_ = cmd.Start()
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func applyAlpha(c color.NRGBA, alpha float64) color.NRGBA {
	c.A = uint8(float64(c.A) * alpha)
	return c
}
