package xwindow

import (
	"fmt"
	"image"
	"image/color"

	"github.com/gastfreund/screenshot/internal/util"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
)

func Capture(x, y, width, height int) (img *image.RGBA, e error) {
	defer func() {
		err := recover()
		if err != nil {
			img = nil
			e = fmt.Errorf("%v", err)
		}
	}()
	c, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return nil, err
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return nil, err
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)

	screen := xproto.Setup(c).DefaultScreen(c)
	wholeScreenBounds := image.Rect(0, 0, int(screen.WidthInPixels), int(screen.HeightInPixels))
	targetBounds := image.Rect(x+x0, y+y0, x+x0+width, y+y0+height)
	intersect := wholeScreenBounds.Intersect(targetBounds)

	rect := image.Rect(0, 0, width, height)
	img, err = util.CreateImage(rect)
	if err != nil {
		return nil, err
	}

	// Paint with opaque black
	index := 0
	for iy := 0; iy < height; iy++ {
		j := index
		for ix := 0; ix < width; ix++ {
			img.Pix[j+3] = 255
			j += 4
		}
		index += img.Stride
	}

	if !intersect.Empty() {
		var data []byte

		xImg, err2 := xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
			int16(intersect.Min.X), int16(intersect.Min.Y),
			uint16(intersect.Dx()), uint16(intersect.Dy()), 0xffffffff).Reply()
		if err2 != nil {
			return nil, err2
		}
		data = xImg.Data

		// BitBlt by hand
		offset := 0
		for iy := intersect.Min.Y; iy < intersect.Max.Y; iy++ {
			for ix := intersect.Min.X; ix < intersect.Max.X; ix++ {
				r := data[offset+2]
				g := data[offset+1]
				b := data[offset]
				img.SetRGBA(ix-(x+x0), iy-(y+y0), color.RGBA{r, g, b, 255})
				offset += 4
			}
		}
	}

	return img, e
}

func NumActiveDisplays() (num int) {
	defer func() {
		e := recover()
		if e != nil {
			num = 0
		}
	}()

	c, err := xgb.NewConn()
	if err != nil {
		return 0
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return 0
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return 0
	}

	num = int(reply.Number)
	return num
}

func GetDisplayBounds(displayIndex int) (rect image.Rectangle) {
	defer func() {
		e := recover()
		if e != nil {
			rect = image.Rectangle{}
		}
	}()

	c, err := xgb.NewConn()
	if err != nil {
		return image.Rectangle{}
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return image.Rectangle{}
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return image.Rectangle{}
	}

	if displayIndex >= int(reply.Number) {
		return image.Rectangle{}
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)

	screen := reply.ScreenInfo[displayIndex]
	x := int(screen.XOrg) - x0
	y := int(screen.YOrg) - y0
	w := int(screen.Width)
	h := int(screen.Height)
	rect = image.Rect(x, y, x+w, y+h)
	return rect
}
