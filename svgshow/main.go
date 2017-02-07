// +build !js

package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/fogleman/primitive/primitive"
	"github.com/funnywwh/nanogui-go"
	"github.com/nfnt/resize"
	"github.com/shibukawa/glfw"
	"github.com/shibukawa/nanovgo"
)

type Application struct {
	screen   *nanogui.Screen
	progress *nanogui.ProgressBar
	shader   *nanogui.GLShader
}

func showSvg(screen *nanogui.Screen) {
	window := nanogui.NewWindow(screen, "showSvg")
	window.SetPosition(0, 0)
	window.SetFixedSize(512, 512)
	window.SetLayout(nanogui.NewGroupLayout())

	nanogui.NewLabel(window, "开始").SetFont("sans-bold")

	NewSvg(window, "./1.svg")
	imgView := nanogui.NewImageView(window)

	imgRes := screen.NVGContext().CreateImage("./gopher.png", nanovgo.ImageGenerateMipmaps)
	imgView.SetImage(imgRes)

}

func (a *Application) makePrimitive(screen *nanogui.Screen) {

	window := nanogui.NewWindow(screen, "向量化相片")
	window.SetPosition(512, 0)
	window.SetFixedSize(512, 800)
	window.SetLayout(nanogui.NewGroupLayout())

	nanogui.NewButton(window, "开始").SetCallback(func() {

	})
	nanogui.NewLabel(window, "进度")

	var imgView *nanogui.ImageView

	progressBar := nanogui.NewProgressBar(window)
	{
		imgBox := nanogui.NewWidget(window)
		imgBox.SetFixedSize(512, 256)
		gridLayout := nanogui.NewGridLayout(nanogui.Horizontal, 2, nanogui.Middle)

		imgBox.SetLayout(gridLayout)

		imgOrgRes := screen.NVGContext().CreateImage("./me.jpg", nanovgo.ImageGenerateMipmaps)
		imgOrg := nanogui.NewImageView(imgBox)
		imgOrg.SetImage(imgOrgRes)
		imgOrg.SetFixedSize(256, 256)

		imgView = nanogui.NewImageView(imgBox)
		imgView.SetFixedSize(256, 256)
	}
	nanogui.NewLabel(window, "状态")
	label := nanogui.NewLabel(window, "")

	a.procPrimitive(progressBar, imgView, label)
}

func check(err error) {
	if err != nil {
		if reflect.ValueOf(err).IsNil() {
			return
		}
		panic(err)
	}
}
func (a *Application) procPrimitive(progressBar *nanogui.ProgressBar, imgView *nanogui.ImageView, label *nanogui.Label) {

	size := uint(256)
	OutputSize := 1024
	Workers := 8
	Mode := 0
	Alpha := 128
	Repeat := 100
	Count := 300

	input, err := primitive.LoadImage("./me.jpg")
	check(err)
	input = resize.Thumbnail(size, size, input, resize.Bilinear)
	var bg primitive.Color
	bg = primitive.MakeColor(primitive.AverageImageColor(input))

	model := primitive.NewModel(input, bg, OutputSize, Workers)
	start := time.Now()
	frame := 0
	drawframe := false
	imgRes := 0
	n := 0
	nps := ""

	status := fmt.Sprintf("processing")
	label.SetCaption(status)
	var draw = func() {
		if !drawframe {
			return
		}
		ctx := a.screen.NVGContext()
		_image := model.Context.Image()

		if imgRes != 0 {
			ctx.DeleteImage(imgRes)
		}
		imgRes = ctx.CreateImageFromGoImage(nanovgo.ImageGenerateMipmaps, _image)

		imgView.SetImage(imgRes)
		p := float32(frame) / float32(Count)
		progressBar.SetValue(p)

		elapsed := time.Since(start).Seconds()

		status := fmt.Sprintf("%d: t=%.3f, score=%.6f, n=%d, n/s=%s p=%v", frame, elapsed, model.Score, n, nps, p)
		label.SetCaption(status)

		drawframe = false

	}
	a.screen.SetDrawContentsCallback(draw)
	go func() {
		for frame = 0; frame < Count; frame++ {
			t := time.Now()
			n = model.Step(primitive.ShapeType(Mode), Alpha, Repeat)
			nps = primitive.NumberString(float64(n) / time.Since(t).Seconds())

			drawframe = true
			glfw.PostEmptyEvent()
		}
	}()

}
func (a *Application) init() {
	glfw.WindowHint(glfw.Samples, 4)
	a.screen = nanogui.NewScreen(1024, 768, "NanoGUI.Go Test", true, true)

	a.screen.NVGContext().CreateFont("chinese", "/Library/Fonts/Microsoft/Fangsong.ttf")
	theme := a.screen.Theme()
	theme.FontBold = "chinese"
	theme.FontNormal = "chinese"

	showSvg(a.screen)
	a.makePrimitive(a.screen)

	a.screen.PerformLayout()
	a.screen.DebugPrint()

	/* All NanoGUI widgets are initialized at this point. Now
	create an OpenGL shader to draw the main window contents.

	NanoGUI comes with a simple Eigen-based wrapper around OpenGL 3,
	which eliminates most of the tedious and error-prone shader and
	buffer object management.
	*/
}

func main() {
	nanogui.Init()
	//nanogui.SetDebug(true)
	app := Application{}
	app.init()
	app.screen.DrawAll()
	app.screen.SetVisible(true)
	nanogui.MainLoop()
}
