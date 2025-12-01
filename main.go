package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type toolMode int

const (
	modeDraw toolMode = iota
	modePixelErase
	modeStrokeErase
)

const (
	canvasWidth  = 4096
	canvasHeight = 4096
	uiHeight     = 110
)

type Vec2 struct {
	X float32
	Y float32
}

type stroke struct {
	Points []Vec2
	Size   float64
	Color  color.Color
	Bounds image.Rectangle
	Erased bool
}

func (s *stroke) expandBounds(p Vec2) {
	if len(s.Points) == 1 {
		s.Bounds = image.Rect(int(p.X), int(p.Y), int(p.X), int(p.Y))
	} else {
		s.Bounds = s.Bounds.Union(image.Rect(int(p.X), int(p.Y), int(p.X), int(p.Y)))
	}
}

func (s *stroke) hit(pos Vec2, tolerance float64) bool {
	if s.Erased {
		return false
	}
	inflated := s.Bounds.Inset(-int(tolerance) - int(s.Size))
	if !rectContainsPoint(inflated, image.Pt(int(pos.X), int(pos.Y))) {
		return false
	}
	for i := 0; i < len(s.Points)-1; i++ {
		a := s.Points[i]
		b := s.Points[i+1]
		if distancePointToSegment(pos, a, b) <= tolerance+(s.Size/2) {
			return true
		}
	}
	return false
}

type slider struct {
	x, y   float64
	width  float64
	min    float64
	max    float64
	value  *float64
	active bool
}

func (s *slider) handleInput(mx, my float64, pressed bool) {
	knobRadius := 10.0
	knobX := s.x + ((*s.value - s.min) / (s.max - s.min) * s.width)
	if pressed {
		if !s.active {
			if math.Hypot(mx-knobX, my-s.y) <= knobRadius*1.5 {
				s.active = true
			}
		}
		if s.active {
			t := (mx - s.x) / s.width
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			*s.value = s.min + t*(s.max-s.min)
		}
	} else {
		s.active = false
	}
}

func (s *slider) draw(dst *ebiten.Image, label string) {
	barY := s.y
	trackHeight := 6.0
	vector.DrawFilledRect(dst, float32(s.x), float32(barY-trackHeight/2), float32(s.width), float32(trackHeight), color.RGBA{60, 60, 60, 255}, false)
	knobRadius := 10.0
	knobX := s.x + ((*s.value - s.min) / (s.max - s.min) * s.width)
	vector.DrawFilledCircle(dst, float32(knobX), float32(barY), float32(knobRadius), color.RGBA{200, 200, 200, 255}, false)
	ebitenutil.DebugPrintAt(dst, fmt.Sprintf("%s: %.1f", label, *s.value), int(s.x), int(s.y)-24)
}

type button struct {
	rect    image.Rectangle
	label   string
	onClick func()
}

func rectContainsPoint(rect image.Rectangle, p image.Point) bool {
	return p.X >= rect.Min.X && p.X < rect.Max.X && p.Y >= rect.Min.Y && p.Y < rect.Max.Y
}

func (b *button) contains(x, y int) bool {
	return x >= b.rect.Min.X && x <= b.rect.Max.X && y >= b.rect.Min.Y && y <= b.rect.Max.Y
}

func (b *button) draw(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, float32(b.rect.Min.X), float32(b.rect.Min.Y), float32(b.rect.Dx()), float32(b.rect.Dy()), color.RGBA{70, 70, 70, 255}, false)
	ebitenutil.DebugPrintAt(dst, b.label, b.rect.Min.X+6, b.rect.Min.Y+8)
}

type confirmDialog struct {
	message   string
	visible   bool
	onConfirm func()
	onCancel  func()
}

func (c *confirmDialog) draw(dst *ebiten.Image) {
	if !c.visible {
		return
	}
	w, h := dst.Size()
	vector.DrawFilledRect(dst, 0, 0, float32(w), float32(h), color.RGBA{0, 0, 0, 120}, false)
	dialogW, dialogH := 400, 160
	x := (w - dialogW) / 2
	y := (h - dialogH) / 2
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(dialogW), float32(dialogH), color.RGBA{30, 30, 30, 255}, false)
	ebitenutil.DebugPrintAt(dst, c.message, x+20, y+30)
	yesRect := image.Rect(x+40, y+90, x+140, y+130)
	noRect := image.Rect(x+dialogW-140, y+90, x+dialogW-40, y+130)
	vector.DrawFilledRect(dst, float32(yesRect.Min.X), float32(yesRect.Min.Y), float32(yesRect.Dx()), float32(yesRect.Dy()), color.RGBA{70, 120, 70, 255}, false)
	vector.DrawFilledRect(dst, float32(noRect.Min.X), float32(noRect.Min.Y), float32(noRect.Dx()), float32(noRect.Dy()), color.RGBA{120, 70, 70, 255}, false)
	ebitenutil.DebugPrintAt(dst, "确认", yesRect.Min.X+30, yesRect.Min.Y+10)
	ebitenutil.DebugPrintAt(dst, "取消", noRect.Min.X+30, noRect.Min.Y+10)
}

func (c *confirmDialog) handleInput(mx, my, viewW, viewH int, pressed bool) {
	if !c.visible || !pressed {
		return
	}
	dialogW, dialogH := 400, 160
	x := (viewW - dialogW) / 2
	y := (viewH - dialogH) / 2
	yesRect := image.Rect(x+40, y+90, x+140, y+130)
	noRect := image.Rect(x+dialogW-140, y+90, x+dialogW-40, y+130)
	if rectContainsPoint(yesRect, image.Pt(mx, my)) {
		c.visible = false
		if c.onConfirm != nil {
			c.onConfirm()
		}
	} else if rectContainsPoint(noRect, image.Pt(mx, my)) {
		c.visible = false
		if c.onCancel != nil {
			c.onCancel()
		}
	}
}

type Game struct {
	canvas       *ebiten.Image
	strokes      []*stroke
	current      *stroke
	mode         toolMode
	brushSize    float64
	eraserSize   float64
	buttons      []*button
	sliders      []*slider
	confirm      confirmDialog
	lastMouseBtn bool
}

func NewGame() *Game {
	g := &Game{
		canvas:     ebiten.NewImage(canvasWidth, canvasHeight),
		strokes:    []*stroke{},
		mode:       modeDraw,
		brushSize:  10,
		eraserSize: 20,
	}
	g.canvas.Fill(color.Black)
	g.setupUI()
	return g
}

func (g *Game) setupUI() {
	btns := []*button{
		{rect: image.Rect(20, 20, 120, 60), label: "画笔", onClick: func() { g.mode = modeDraw }},
		{rect: image.Rect(140, 20, 260, 60), label: "像素橡皮", onClick: func() { g.mode = modePixelErase }},
		{rect: image.Rect(260, 20, 380, 60), label: "笔画橡皮", onClick: func() { g.mode = modeStrokeErase }},
		{rect: image.Rect(400, 20, 520, 60), label: "保存", onClick: func() { g.saveImage() }},
		{rect: image.Rect(540, 20, 660, 60), label: "清空", onClick: func() { g.confirmClear() }},
	}
	g.buttons = btns
	g.sliders = []*slider{
		{x: 700, y: 40, width: 200, min: 2, max: 60, value: &g.brushSize},
		{x: 950, y: 40, width: 200, min: 4, max: 80, value: &g.eraserSize},
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	viewW, viewH := ebiten.WindowSize()

	if g.confirm.visible {
		g.confirm.handleInput(mx, my, viewW, viewH, pressed && !g.lastMouseBtn)
		g.lastMouseBtn = pressed
		return nil
	}

	for _, s := range g.sliders {
		s.handleInput(float64(mx), float64(my), pressed)
	}

	if pressed && !g.lastMouseBtn {
		for _, b := range g.buttons {
			if b.contains(mx, my) {
				b.onClick()
				g.lastMouseBtn = pressed
				return nil
			}
		}
	}

	if my <= uiHeight {
		g.lastMouseBtn = pressed
		return nil
	}

	switch g.mode {
	case modeDraw:
		g.handleDrawing(mx, my, pressed)
	case modePixelErase:
		g.handlePixelErase(mx, my, pressed)
	case modeStrokeErase:
		g.handleStrokeErase(mx, my, pressed && !g.lastMouseBtn)
	}

	g.lastMouseBtn = pressed
	return nil
}

func (g *Game) handleDrawing(mx, my int, pressed bool) {
	if pressed {
		p := Vec2{X: float32(mx), Y: float32(my)}
		if g.current == nil {
			g.current = &stroke{Points: []Vec2{p}, Size: g.brushSize, Color: color.White}
			g.current.expandBounds(p)
		} else {
			g.current.Points = append(g.current.Points, p)
			g.current.expandBounds(p)
		}
		if len(g.current.Points) >= 2 {
			a := g.current.Points[len(g.current.Points)-2]
			b := g.current.Points[len(g.current.Points)-1]
			g.drawSegment(a, b, g.brushSize, color.White)
		} else {
			vector.DrawFilledCircle(g.canvas, p.X, p.Y, float32(g.brushSize/2), color.White, true)
		}
	} else if g.current != nil {
		g.strokes = append(g.strokes, g.current)
		g.current = nil
	}
}

func (g *Game) handlePixelErase(mx, my int, pressed bool) {
	if pressed {
		vector.DrawFilledCircle(g.canvas, float32(mx), float32(my), float32(g.eraserSize/2), color.Black, true)
	}
}

func (g *Game) handleStrokeErase(mx, my int, clicked bool) {
	if !clicked {
		return
	}
	pos := Vec2{X: float32(mx), Y: float32(my)}
	tolerance := g.eraserSize / 2
	removed := false
	for _, s := range g.strokes {
		if s.hit(pos, tolerance) {
			s.Erased = true
			removed = true
		}
	}
	if removed {
		g.rebuildCanvas()
	}
}

func (g *Game) rebuildCanvas() {
	g.canvas.Fill(color.Black)
	for _, s := range g.strokes {
		if s.Erased {
			continue
		}
		for i := 0; i < len(s.Points)-1; i++ {
			g.drawSegment(s.Points[i], s.Points[i+1], s.Size, s.Color)
		}
	}
}

func (g *Game) drawSegment(a, b Vec2, size float64, clr color.Color) {
	vector.StrokeLine(g.canvas, a.X, a.Y, b.X, b.Y, float32(size), clr, true)
}

func (g *Game) confirmClear() {
	g.confirm = confirmDialog{
		message: "确认清空画布吗？",
		visible: true,
		onConfirm: func() {
			g.canvas.Fill(color.Black)
			g.strokes = []*stroke{}
			g.current = nil
		},
		onCancel: func() {},
	}
}

func (g *Game) saveImage() {
	now := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("drawing_%s.png", now)
	filePath := filepath.Join(".", filename)
	img := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	draw.Draw(img, img.Bounds(), g.canvas, image.Point{}, draw.Src)
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("保存失败:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Println("保存失败:", err)
		return
	}
	fmt.Println("已保存到", filePath)
}

func (g *Game) Draw(screen *ebiten.Image) {
	w, _ := screen.Size()
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.canvas, op)

	vector.DrawFilledRect(screen, 0, 0, float32(w), uiHeight, color.RGBA{20, 20, 20, 255}, false)
	for _, b := range g.buttons {
		b.draw(screen)
	}
	g.sliders[0].draw(screen, "笔粗细")
	g.sliders[1].draw(screen, "橡皮大小")

	status := "当前模式: "
	switch g.mode {
	case modeDraw:
		status += "画笔"
	case modePixelErase:
		status += "像素橡皮"
	case modeStrokeErase:
		status += "笔画橡皮"
	}
	ebitenutil.DebugPrintAt(screen, status, 20, uiHeight-24)

	if g.confirm.visible {
		g.confirm.draw(screen)
	}
}

func distancePointToSegment(p, a, b Vec2) float64 {
	apx := float64(p.X - a.X)
	apy := float64(p.Y - a.Y)
	abx := float64(b.X - a.X)
	aby := float64(b.Y - a.Y)
	abLen2 := abx*abx + aby*aby
	if abLen2 == 0 {
		return math.Hypot(apx, apy)
	}
	t := (apx*abx + apy*aby) / abLen2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx := float64(a.X) + t*abx
	cy := float64(a.Y) + t*aby
	return math.Hypot(float64(p.X)-cx, float64(p.Y)-cy)
}

func main() {
	game := NewGame()
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("DraftIt - 无限画布")
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
