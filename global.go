package nanogui

import (
	"sync"
	"time"

	"github.com/goxjs/gl"
	"github.com/shibukawa/glfw"
)

var mainloopActive bool = false
var startTime time.Time
var debugFlag bool

func Init() {
	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	startTime = time.Now()
}

func GetTime() float32 {
	return float32(time.Now().Sub(startTime)/time.Millisecond) * 0.001
}

func MainLoop() {
	mainloopActive = true

	var wg sync.WaitGroup

	//win delete for save power

	for mainloopActive {
		haveActiveScreen := false
		for _, screen := range nanoguiScreens {
			if !screen.Visible() {
				continue
			} else if screen.GLFWWindow().ShouldClose() {
				screen.SetVisible(false)
				continue
			}
			//screen.DebugPrint()
			screen.DrawAll()
			haveActiveScreen = true
		}
		if !haveActiveScreen {
			mainloopActive = false
			break
		}
		glfw.WaitEvents()
	}

	wg.Wait()
}

func SetDebug(d bool) {
	debugFlag = d
}

func InitWidget(child, parent Widget) {
	//w.cursor = Arrow
	if parent != nil {
		parent.AddChild(parent, child)
		child.SetTheme(parent.Theme())
	}
	child.SetVisible(true)
	child.SetEnabled(true)
	child.SetFontSize(-1)
}
