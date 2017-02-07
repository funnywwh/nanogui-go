package main

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/funnywwh/nanogui-go"
	"github.com/shibukawa/nanovgo"

	"github.com/fogleman/gg"

	"github.com/shibukawa/glfw"
)

type Svg struct {
	nanogui.WidgetImplement
	shapes       []shape
	scale        float32
	ctx          Context
	imgRes       int
	drawedShapes bool
	mux          sync.Mutex
}

func (s *Svg) PreferredSize(self nanogui.Widget, ctx *nanovgo.Context) (w int, h int) {
	w, h = s.FixedSize()
	return
}
func (s *Svg) String() string {
	return s.StringHelper("svg", "")
}

type point struct {
	x, y float32
}
type Context interface {
	rect(x, y, w, h float32, clr color.Color)
	polygon(pts []point, clr color.Color)
	image() image.Image
}

type GGContext struct {
	ctx *gg.Context
}

func (c *GGContext) rect(x, y, w, h float32, clr color.Color) {
	ctx := c.ctx
	ctx.Push()
	ctx.DrawRectangle(float64(x), float64(y), float64(w), float64(h))

	p := gg.NewSolidPattern(clr)
	ctx.SetStrokeStyle(p)
	ctx.SetFillStyle(p)

	ctx.Fill()
	ctx.Stroke()
}

func (c *GGContext) image() image.Image {
	return c.ctx.Image()
}

func (c *GGContext) polygon(pts []point, clr color.Color) {
	ctx := c.ctx
	ctx.Push()
	if len(pts) > 0 {
		ctx.MoveTo(float64(pts[0].x), float64(pts[0].y))
		for _, p := range pts[1:] {
			ctx.LineTo(float64(p.x), float64(p.y))
		}

		ctx.LineTo(float64(pts[0].x), float64(pts[0].y))
	}
	ctx.ClosePath()
	p := gg.NewSolidPattern(clr)
	ctx.SetStrokeStyle(p)
	ctx.SetFillStyle(p)
	ctx.Fill()
	ctx.Stroke()
}
func NewGGContext(w, h int) (o *GGContext) {
	o = &GGContext{
		ctx: gg.NewContext(w, h),
	}
	return o
}

func parseFloat32(s string) float32 {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0
	}
	return float32(f)
}
func parsePoints(s string) (pts []point) {
	ptsStrArr := strings.Split(s, " ")
	for _, pstr := range ptsStrArr {
		ptstr := strings.Split(pstr, ",")
		pts = append(pts, point{
			x: parseFloat32(ptstr[0]),
			y: parseFloat32(ptstr[1]),
		})
	}
	return
}

var colorName = map[string]color.Color{
	"red": color.RGBA{255, 0, 0, 0},
}

func parseColor(rgb, a string) color.Color {
	if rgb[0] == '#' {
		v, err := hex.DecodeString(rgb[1:])
		if err != nil {
			return color.RGBA{0, 0, 0, 0}
		}
		return color.NRGBA{v[0], v[1], v[2], uint8(255 * parseFloat32(a))}
	}
	if c, ok := colorName[rgb]; ok {
		return c
	}
	return color.RGBA{0, 0, 0, 0}
}

type shape interface {
	draw(ctx Context)
}

type Rect struct {
	x, y, w, h float32
	clr        color.Color
}

func (r *Rect) draw(ctx Context) {
	ctx.rect(r.x, r.y, r.w, r.h, r.clr)
}

type Polygon struct {
	pts []point
	clr color.Color
}

func (p *Polygon) draw(ctx Context) {
	ctx.polygon(p.pts, p.clr)
}
func (s *Svg) parse(r io.Reader) {
	dec := xml.NewDecoder(r)
	for tok, err := dec.Token(); err == nil; tok, err = dec.Token() {
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "rect":

				var x, y, w, h float32
				var clr color.Color
				var color, opacity string
				for _, a := range e.Attr {
					switch a.Name.Local {
					case "x":
						x = parseFloat32(a.Value)
					case "y":
						y = parseFloat32(a.Value)
					case "width":
						w = parseFloat32(a.Value)
					case "height":
						h = parseFloat32(a.Value)
					case "fill-opacity":
						opacity = a.Value
					case "fill":
						color = a.Value
					}

				}

				clr = parseColor(color, opacity)

				rect := Rect{
					x: x, y: y,
					w: w, h: h,
					clr: clr,
				}
				s.shapes = append(s.shapes, &rect)
			case "polygon":

				var clr color.Color
				var color, opacity string
				var pts []point
				for _, a := range e.Attr {
					switch a.Name.Local {
					case "points":
						pts = parsePoints(a.Value)
					case "fill-opacity":
						opacity = a.Value
					case "fill":
						color = a.Value
					}

				}

				clr = parseColor(color, opacity)

				s.shapes = append(s.shapes, &Polygon{
					pts: pts,
					clr: clr,
				})
			case "g":
				for _, a := range e.Attr {
					switch a.Name.Local {
					case "transform":
						transform := a.Value
						m := regexp.MustCompile(`scale\(([\d\.]+)\)`).FindStringSubmatch(transform)
						fmt.Printf("m:%#v\n", m)
						if m != nil {
							s.scale = parseFloat32(m[1])
						}
					}

				}
			case "svg":
				for _, a := range e.Attr {
					switch a.Name.Local {
					case "width":
						w, h := parseFloat32(a.Value), parseFloat32(a.Value)
						if w > 0 {
							s.SetFixedWidth(int(w))
						}
						if h > 0 {
							s.SetFixedHeight(int(h))
						}
					}

				}
			}
		}
	}

}
func (s *Svg) Draw(self nanogui.Widget, ctx *nanovgo.Context) {
	s.WidgetImplement.Draw(self, ctx)

	ctx.Save()

	//draw begin
	x, y := s.Position()
	ctx.Translate(float32(x), float32(y))
	ctx.Scale(s.scale, s.scale)

	s.mux.Lock()
	drawed := s.drawedShapes
	s.mux.Unlock()
	if drawed {
		s.imgRes = ctx.CreateImageFromGoImage(nanovgo.ImageGenerateMipmaps, s.ctx.image())

		w, h := float32(s.Width()), float32(s.Height())
		imgPaint := nanovgo.ImagePattern(0, 0, w, h, 0, s.imgRes, 1)
		ctx.BeginPath()
		ctx.Rect(0, 0, w, h)
		ctx.SetFillPaint(imgPaint)
		ctx.Fill()
	}
	//draw end
	ctx.Restore()
}

type Period struct {
	start time.Time
}

func NewPeriod() *Period {
	return &Period{
		start: time.Now(),
	}
}
func (p *Period) Start() {
	p.start = time.Now()
}
func (p *Period) PassMs() (ms int64) {
	return time.Since(p.start).Nanoseconds() / int64(time.Millisecond)
}
func NewSvg(parent nanogui.Widget, svgPath string) *Svg {
	s := &Svg{
		scale: 1,
	}
	nanogui.InitWidget(s, parent)

	go func() {
		f, err := os.Open(svgPath)
		if err != nil {
			return
		}
		svgr := f
		defer f.Close()

		start := NewPeriod()

		start.Start()
		s.parse(svgr)
		fmt.Printf("svg parse shape time:%v\n", start.PassMs())

		fmt.Printf("shape len:%d\n", len(s.shapes))

		s.ctx = NewGGContext(s.FixedSize())

		start.Start()
		for _, p := range s.shapes {
			p.draw(s.ctx)
		}
		fmt.Printf("draw shape time:%v\n", start.PassMs())
		s.mux.Lock()
		s.drawedShapes = true
		s.mux.Unlock()

		glfw.PostEmptyEvent()

	}()

	return s
}
