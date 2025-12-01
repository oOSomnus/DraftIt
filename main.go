package main

import (
	"embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sqweek/dialog"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type toolMode int

const (
	modeDraw toolMode = iota
	modePixelErase
	modeStrokeErase
)

const (
	initialCanvasSize = 2048
	uiHeight          = 110
)

//go:embed assets/NotoSansSC-Regular.otf
var fontBytes []byte

var uiFont font.Face

type Vec2 struct {
	X float32
	Y float32
}

type vec2d struct {
	X float64
	Y float64
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

func initFont() {
	parsed, err := opentype.Parse(fontBytes)
	if err != nil {
		panic(fmt.Errorf("加载字体失败: %w", err))
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Errorf("构建字体失败: %w", err))
	}
	uiFont = face
}

func drawText(dst *ebiten.Image, str string, x, y int, clr color.Color) {
	if uiFont == nil {
		return
	}
	text.Draw(dst, str, uiFont, x, y, clr)
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
	drawText(dst, fmt.Sprintf("%s: %.1f", label, *s.value), int(s.x), int(s.y)-8, color.White)
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
	textY := b.rect.Min.Y + b.rect.Dy()/2 + 6
	drawText(dst, b.label, b.rect.Min.X+12, textY, color.White)
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
	drawText(dst, c.message, x+20, y+40, color.White)
	yesRect := image.Rect(x+40, y+90, x+140, y+130)
	noRect := image.Rect(x+dialogW-140, y+90, x+dialogW-40, y+130)
	vector.DrawFilledRect(dst, float32(yesRect.Min.X), float32(yesRect.Min.Y), float32(yesRect.Dx()), float32(yesRect.Dy()), color.RGBA{70, 120, 70, 255}, false)
	vector.DrawFilledRect(dst, float32(noRect.Min.X), float32(noRect.Min.Y), float32(noRect.Dx()), float32(noRect.Dy()), color.RGBA{120, 70, 70, 255}, false)
	drawText(dst, "确认", yesRect.Min.X+36, yesRect.Min.Y+24, color.White)
	drawText(dst, "取消", noRect.Min.X+36, noRect.Min.Y+24, color.White)
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
	canvasOrigin vec2d
	strokes      []*stroke
	current      *stroke
	currentMode  toolMode
	mode         toolMode
	brushSize    float64
	eraserSize   float64
	buttons      []*button
	sliders      []*slider
	confirm      confirmDialog
	lastMouseBtn bool
	camera       vec2d
	panning      bool
	panLast      Vec2
}

func NewGame() *Game {
	initFont()
	g := &Game{
		canvas:       ebiten.NewImage(initialCanvasSize, initialCanvasSize),
		canvasOrigin: vec2d{X: -initialCanvasSize / 2, Y: -initialCanvasSize / 2},
		strokes:      []*stroke{},
		mode:         modeDraw,
		currentMode:  modeDraw,
		brushSize:    10,
		eraserSize:   20,
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

func (g *Game) canvasRect() image.Rectangle {
	originX := int(math.Floor(g.canvasOrigin.X))
	originY := int(math.Floor(g.canvasOrigin.Y))
	return image.Rect(originX, originY, originX+g.canvas.Bounds().Dx(), originY+g.canvas.Bounds().Dy())
}

func (g *Game) ensurePointVisible(p Vec2, radius float64) {
	margin := int(math.Ceil(radius)) + 8
	neededMinX := int(math.Floor(float64(p.X))) - margin
	neededMaxX := int(math.Ceil(float64(p.X))) + margin
	neededMinY := int(math.Floor(float64(p.Y))) - margin
	neededMaxY := int(math.Ceil(float64(p.Y))) + margin

	rect := g.canvasRect()
	newOriginX := rect.Min.X
	newOriginY := rect.Min.Y
	newW := rect.Dx()
	newH := rect.Dy()
	expanded := false

	if neededMinX < rect.Min.X {
		extra := rect.Min.X - neededMinX + 128
		newOriginX -= extra
		newW += extra
		expanded = true
	}
	if neededMaxX > rect.Max.X {
		extra := neededMaxX - rect.Max.X + 128
		newW += extra
		expanded = true
	}
	if neededMinY < rect.Min.Y {
		extra := rect.Min.Y - neededMinY + 128
		newOriginY -= extra
		newH += extra
		expanded = true
	}
	if neededMaxY > rect.Max.Y {
		extra := neededMaxY - rect.Max.Y + 128
		newH += extra
		expanded = true
	}

	if expanded {
		g.canvasOrigin = vec2d{X: float64(newOriginX), Y: float64(newOriginY)}
		g.canvas = ebiten.NewImage(newW, newH)
		g.rebuildCanvas()
	}
}

func (g *Game) worldFromScreen(mx, my int) Vec2 {
	return Vec2{X: float32(float64(mx) + g.camera.X), Y: float32(float64(my) + g.camera.Y)}
}

func (g *Game) worldToCanvas(p Vec2) Vec2 {
	return Vec2{X: p.X - float32(g.canvasOrigin.X), Y: p.Y - float32(g.canvasOrigin.Y)}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	leftPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	rightPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	justClicked := leftPressed && !g.lastMouseBtn
	viewW, viewH := ebiten.WindowSize()

	if g.confirm.visible {
		g.confirm.handleInput(mx, my, viewW, viewH, justClicked)
		g.lastMouseBtn = leftPressed
		return nil
	}

	if my > uiHeight && rightPressed {
		if !g.panning {
			g.panning = true
			g.panLast = Vec2{X: float32(mx), Y: float32(my)}
		} else {
			dx := float32(mx) - g.panLast.X
			dy := float32(my) - g.panLast.Y
			g.camera.X -= float64(dx)
			g.camera.Y -= float64(dy)
			g.panLast = Vec2{X: float32(mx), Y: float32(my)}
		}
	} else {
		g.panning = false
	}

	for _, s := range g.sliders {
		s.handleInput(float64(mx), float64(my), leftPressed)
	}

	if justClicked && !g.panning {
		for _, b := range g.buttons {
			if b.contains(mx, my) {
				b.onClick()
				g.lastMouseBtn = leftPressed
				return nil
			}
		}
	}

	if my <= uiHeight {
		g.lastMouseBtn = leftPressed
		return nil
	}

	if g.panning {
		g.lastMouseBtn = leftPressed
		return nil
	}

	switch g.mode {
	case modeDraw:
		g.handleStrokeDrawing(mx, my, leftPressed, g.brushSize, color.White)
	case modePixelErase:
		g.handleStrokeDrawing(mx, my, leftPressed, g.eraserSize, color.Black)
	case modeStrokeErase:
		g.handleStrokeErase(mx, my, justClicked)
	}

	g.lastMouseBtn = leftPressed
	return nil
}

func (g *Game) handleStrokeDrawing(mx, my int, pressed bool, size float64, clr color.Color) {
	if pressed {
		p := g.worldFromScreen(mx, my)
		g.ensurePointVisible(p, size)
		canvasPoint := g.worldToCanvas(p)
		if g.current == nil || g.currentMode != g.mode {
			g.current = &stroke{Points: []Vec2{p}, Size: size, Color: clr}
			g.currentMode = g.mode
			g.current.expandBounds(p)
		} else {
			g.current.Points = append(g.current.Points, p)
			g.current.expandBounds(p)
		}
		if len(g.current.Points) >= 2 {
			a := g.current.Points[len(g.current.Points)-2]
			b := g.current.Points[len(g.current.Points)-1]
			g.drawSegment(a, b, size, clr)
		} else {
			vector.DrawFilledCircle(g.canvas, canvasPoint.X, canvasPoint.Y, float32(size/2), clr, true)
		}
	} else if g.current != nil && g.currentMode == g.mode {
		g.strokes = append(g.strokes, g.current)
		g.current = nil
	}
}

func (g *Game) handleStrokeErase(mx, my int, clicked bool) {
	if !clicked {
		return
	}
	pos := g.worldFromScreen(mx, my)
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
	render := func(s *stroke) {
		for i := 0; i < len(s.Points)-1; i++ {
			g.drawSegment(s.Points[i], s.Points[i+1], s.Size, s.Color)
		}
	}

	for _, s := range g.strokes {
		if s.Erased {
			continue
		}
		render(s)
	}

	if g.current != nil && g.currentMode == g.mode {
		render(g.current)
	}
}

func (g *Game) drawSegment(a, b Vec2, size float64, clr color.Color) {
	ca := g.worldToCanvas(a)
	cb := g.worldToCanvas(b)
	vector.StrokeLine(g.canvas, ca.X, ca.Y, cb.X, cb.Y, float32(size), clr, true)
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
	suggested := fmt.Sprintf("drawing_%s.png", now)
	path, err := dialog.File().Title("保存图片").Filter("PNG 图片", "png").SetStartFile(suggested).Save()
	if err != nil {
		if errors.Is(err, dialog.ErrCancelled) {
			return
		}
		fmt.Println("保存失败:", err)
		return
	}

	if filepath.Ext(path) == "" {
		path += ".png"
	}

	bounds, ok := g.drawingBounds()
	if !ok {
		fmt.Println("没有内容可保存")
		return
	}

	canvasRect := g.canvasRect()
	subRect := image.Rect(bounds.Min.X-canvasRect.Min.X, bounds.Min.Y-canvasRect.Min.Y, bounds.Max.X-canvasRect.Min.X, bounds.Max.Y-canvasRect.Min.Y)
	subImage := g.canvas.SubImage(subRect).(*ebiten.Image)
	pixels := subImage.ReadPixels()
	img := image.NewRGBA(image.Rect(0, 0, subRect.Dx(), subRect.Dy()))
	copy(img.Pix, pixels)

	f, err := os.Create(path)
	if err != nil {
		fmt.Println("保存失败:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Println("保存失败:", err)
		return
	}
	fmt.Println("已保存到", path)
}

func (g *Game) drawingBounds() (image.Rectangle, bool) {
	minX, minY := math.MaxInt32, math.MaxInt32
	maxX, maxY := math.MinInt32, math.MinInt32

	considerStroke := func(s *stroke) {
		if len(s.Points) == 0 {
			return
		}
		padding := int(math.Ceil(s.Size / 2))
		b := s.Bounds.Inset(-padding)
		if b.Min.X < minX {
			minX = b.Min.X
		}
		if b.Min.Y < minY {
			minY = b.Min.Y
		}
		if b.Max.X > maxX {
			maxX = b.Max.X
		}
		if b.Max.Y > maxY {
			maxY = b.Max.Y
		}
	}

	for _, s := range g.strokes {
		if s.Erased {
			continue
		}
		considerStroke(s)
	}

	if g.current != nil && g.currentMode == g.mode {
		considerStroke(g.current)
	}

	if minX == math.MaxInt32 {
		return image.Rectangle{}, false
	}

	padding := 8
	return image.Rect(minX-padding, minY-padding, maxX+padding, maxY+padding), true
}

func (g *Game) Draw(screen *ebiten.Image) {
	w, _ := screen.Size()
	screen.Fill(color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.camera.X+g.canvasOrigin.X, -g.camera.Y+g.canvasOrigin.Y)
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
	drawText(screen, status, 20, uiHeight-20, color.White)

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
