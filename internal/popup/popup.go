package popup

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/api"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// closeButtonSize is the side length of the square close button.
const closeButtonSize = 48

// announcementGame renders a fullscreen announcement image with optional title,
// description and link. The image is scaled with "contain" logic (letterboxed).
type announcementGame struct {
	ann       models.AgentAnnouncement
	img       *ebiten.Image
	apiClient *api.Client

	screenWidth  int
	screenHeight int

	closeRect rect
	linkRect  rect
	linkURL   string
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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.closeRect.contains(x, y) {
			return ebiten.Termination
		}
		if g.linkRect.contains(x, y) && g.linkURL != "" {
			openURL(g.linkURL)
		}
	}

	return nil
}

func (g *announcementGame) Draw(screen *ebiten.Image) {
	g.screenWidth, g.screenHeight = screen.Size()

	// Black background.
	screen.Fill(color.Black)

	if g.img == nil {
		return
	}

	// Draw image with contain scaling.
	iw, ih := g.img.Bounds().Dx(), g.img.Bounds().Dy()
	scale := 1.0
	if iw > 0 && ih > 0 {
		scale = min(float64(g.screenWidth)/float64(iw), float64(g.screenHeight)/float64(ih))
	}
	dw := float64(iw) * scale
	dh := float64(ih) * scale
	dx := (float64(g.screenWidth) - dw) / 2
	dy := (float64(g.screenHeight) - dh) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(dx, dy)
	screen.DrawImage(g.img, op)

	// Draw close button (top-right).
	g.closeRect = rect{x: g.screenWidth - closeButtonSize - 16, y: 16, w: closeButtonSize, h: closeButtonSize}
	g.drawCloseButton(screen, g.closeRect)

	// Draw title/description/link panel at the bottom.
	texts := []string{}
	if g.ann.Title != "" {
		texts = append(texts, g.ann.Title)
	}
	if g.ann.Description != nil && *g.ann.Description != "" {
		texts = append(texts, *g.ann.Description)
	}
	if g.ann.LinkURL != nil && *g.ann.LinkURL != "" {
		g.linkURL = *g.ann.LinkURL
		texts = append(texts, "Abrir enlace")
	} else {
		g.linkURL = ""
	}

	if len(texts) > 0 {
		g.drawBottomPanel(screen, texts)
	}
}

func (g *announcementGame) drawCloseButton(screen *ebiten.Image, r rect) {
	// Red background.
	btn := ebiten.NewImage(r.w, r.h)
	btn.Fill(color.RGBA{R: 220, G: 38, B: 38, A: 255})

	// White X.
	padding := r.w / 4
	colorX := color.White
	for i := 0; i < r.w-2*padding; i++ {
		btn.Set(padding+i, padding+i, colorX)
		btn.Set(r.w-padding-i-1, padding+i, colorX)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.x), float64(r.y))
	screen.DrawImage(btn, op)
}

func (g *announcementGame) drawBottomPanel(screen *ebiten.Image, texts []string) {
	lineHeight := 18
	panelPadding := 16
	panelHeight := len(texts)*lineHeight + 2*panelPadding
	if g.linkURL != "" {
		panelHeight += 4
	}

	panelImg := image.NewRGBA(image.Rect(0, 0, g.screenWidth, panelHeight))
	// Semi-transparent black background.
	draw.Draw(panelImg, panelImg.Bounds(), image.NewUniform(color.NRGBA{R: 0, G: 0, B: 0, A: 200}), image.Point{}, draw.Src)

	drawer := font.Drawer{
		Dst:  panelImg,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.I(panelPadding),
			Y: fixed.I(panelPadding + basicfont.Face7x13.Metrics().Ascent.Ceil()),
		},
	}

	for i, t := range texts {
		if g.linkURL != "" && i == len(texts)-1 {
			drawer.Src = image.NewUniform(color.RGBA{R: 96, G: 165, B: 250, A: 255})
			g.linkRect = rect{
				x: panelPadding,
				y: g.screenHeight - panelHeight + panelPadding + i*lineHeight - 2,
				w: len(t) * 7,
				h: lineHeight,
			}
		} else {
			drawer.Src = image.NewUniform(color.White)
		}
		drawer.DrawString(t)
		drawer.Dot.Y += fixed.I(lineHeight)
	}

	eimg := ebiten.NewImageFromImage(panelImg)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, float64(g.screenHeight-panelHeight))
	screen.DrawImage(eimg, op)
}

func (g *announcementGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// Show downloads the announcement image and displays it in a fullscreen popup.
// It reports the impression to the backend immediately after the window is shown.
func Show(ctx context.Context, apiClient *api.Client, ann models.AgentAnnouncement) error {
	if ann.ImageDownloadURL == "" {
		return fmt.Errorf("announcement has no image download URL")
	}

	imgPath, err := downloadImage(ctx, ann.ImageDownloadURL)
	if err != nil {
		return fmt.Errorf("download image: %w", err)
	}
	defer os.Remove(imgPath)

	// Report impression as soon as we are about to show the popup.
	if apiClient != nil {
		if err := apiClient.ReportAnnouncementImpression(ctx, ann.Identifier); err != nil {
			fmt.Fprintf(os.Stderr, "failed to report announcement impression: %v\n", err)
		}
	}

	img, _, err := ebitenutil.NewImageFromFile(imgPath)
	if err != nil {
		return fmt.Errorf("load image: %w", err)
	}

	ebiten.SetWindowTitle("EMIR Anuncio")
	ebiten.SetWindowDecorated(false)
	ebiten.SetFullscreen(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetVsyncEnabled(true)

	game := &announcementGame{
		ann:       ann,
		img:       img,
		apiClient: apiClient,
	}

	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		return fmt.Errorf("run game: %w", err)
	}
	return nil
}

func downloadImage(ctx context.Context, url string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	imgPath := filepath.Join(os.TempDir(), fmt.Sprintf("emir-announcement-%d.img", time.Now().UnixNano()))
	out, err := os.Create(imgPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
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
