package ansi

import (
	"bytes"
	"fmt"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColors(t *testing.T) {
	var sl []string
	for i := 0; i < 8; i++ {
		sl = append(sl, fmt.Sprintf("\x1b[0m\x1b[%dmdark \x1b[1;%dmlight \x1b[0;%dmdark\x1b[0m\n", i+30, i+30, i+30))
		sl = append(sl, fmt.Sprintf("\x1b[0m\x1b[%dmdark \x1b[1;%dmlight \x1b[0;%dmdark\x1b[0m\n", i+40, i+40, i+40))
	}
	s := strings.Join(sl, "")
	fmt.Println(s)

	ans, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}

	img := RenderImage(ans)
	f, err := os.Create("colors.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}

}

func TestRender(t *testing.T) {
	files, err := filepath.Glob("testfiles/*.ans")
	if err != nil {
		t.Fatal(err)
	}
	for _, fname := range files {
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			t.Fatal(err)
		}
		ans, err := Parse(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if ans.Width == 0 {
			t.Errorf("File %s has 0 width", fname)
			continue
		}
		if ans.Height == 0 {
			t.Errorf("File %s has 0 height", fname)
			continue
		}
		// fmt.Printf("%dx%d\n", ans.Width, ans.Height)
		// fmt.Print("\x1b[0m")
		// for y := 0; y < ans.Height; y++ {
		// 	for x := 0; x < ans.Width; x++ {
		// 		p := ans.Pix[y*ans.Width+x]
		// 		c := p.C
		// 		if c == 0 {
		// 			c = 32
		// 		}
		// 		if clr := p.ForegroundColor; clr > 7 {
		// 			fmt.Printf("\x1b[%d;1m", 30+clr)
		// 		} else {
		// 			fmt.Printf("\x1b[%d;20m", 30+clr)
		// 		}
		// 		if clr := p.BackgroundColor; clr > 7 {
		// 			panic("high intensity background color")
		// 		} else {
		// 			fmt.Printf("\x1b[%dm", 40+clr)
		// 		}
		// 		// if x&1 == 0 {
		// 		// 	fmt.Printf("\x1b[33m")
		// 		// } else {
		// 		// 	fmt.Printf("\x1b[33;1m")
		// 		// }
		// 		fmt.Print(string(PCAsciiToUnicode[c]))
		// 	}
		// 	fmt.Println()
		// }

		img := RenderImage(ans)
		f, err := os.Create(fmt.Sprintf("out/%s.png", filepath.Base(fname)))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}
}
