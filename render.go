package ansi

import (
	"fmt"
)

type Renderer struct {
	rows [][]Pixel
	screenWidth int
	row, col int
	savedCursors [][2]int
	fgBold byte
	bgBold byte
	bgColor byte
	fgColor byte
	blink Blink
}

func NewRenderer() *Renderer {
	r := &Renderer{}
	r.Reset()
	return r
}

func RenderSequence(seq []Sequence) (*Image, error) {
	return NewRenderer().RenderSequence(seq)
}

func (r *Renderer) Reset() {
	*r = Renderer{
		screenWidth: 80, // TODO: should be configurable and optional
		row: 1,
		col: 1,
		fgBold: 0,
		bgBold: 0,
		bgColor: 0,
		fgColor: 7,
		blink: BlinkNone,
	}
}

func (r *Renderer) RenderSequence(seq []Sequence) (*Image, error) {
	for _, s := range seq {
		switch s := s.(type) {
		case Character:
			switch s.C {
			// case lf:
			// 	row++
			// case cr:
			// 	col = 1
			case lf:
				r.row++
				r.col = 1
			case cr:
			default:
				y := r.row - 1
				x := r.col - 1
				// TODO: Not sure how best to handle out of bounds values
				if x < 0 {
					x = 0
				}
				if y < 0 {
					y = 0
				}
				for len(r.rows) <= y {
					r.rows = append(r.rows, nil)
				}
				row := r.rows[y]
				for len(row) <= x {
					row = append(row, Pixel{})
				}
				r.rows[y] = row
				row[x] = Pixel{
					C:               s.C,
					ForegroundColor: r.fgColor + r.fgBold,
					BackgroundColor: r.bgColor + r.bgBold,
					Blink:           r.blink,
					// TODO: attributes
				}
				r.col++
			}
		case Clear:
			switch s.Type {
			case ClearTypeScreen:
				// TODO: reset row and col?
				r.rows = r.rows[:0]
			default:
				return nil, fmt.Errorf("unhandled clear type %d", s.Type)
			}
		case CursorBackward:
			r.col -= s.N
		case CursorDown:
			r.row += s.N
		case CursorForward:
			r.col += s.N
		case CursorUp:
			r.row -= s.N
		case MoveCursorTo:
			r.row = s.Row
			r.col = s.Col
		case RestoreCursorPosition:
			x := r.savedCursors[len(r.savedCursors)-1]
			r.savedCursors = r.savedCursors[:len(r.savedCursors)-1]
			r.row = x[0]
			r.col = x[1]
		case SaveCursorPosition:
			r.savedCursors = append(r.savedCursors, [2]int{r.row, r.col})
		case SelectGraphicsRendition:
			switch {
			case s.N == GraphicsRenditionReset:
				r.bgColor = 0
				r.fgColor = 7
				r.fgBold = 0
				r.blink = BlinkNone
			case s.N == GraphicsRenditionBold:
				r.fgBold = 8
			case s.N == GraphicsrenditionDefaultTextColor:
				r.fgColor = 7
				// TODO: should this also clear bold or not?
				r.fgBold = 0
			case s.N >= GraphicsRenditionSetTextColor0 && s.N <= GraphicsRenditionSetTextColor7:
				r.fgColor = byte(s.N - GraphicsRenditionSetTextColor0)
			case s.N >= GraphicsRenditionSetBackgroundColor0 && s.N <= GraphicsRenditionSetBackgroundColor7:
				r.bgColor = byte(s.N - GraphicsRenditionSetBackgroundColor0)
			case s.N == GraphicRenditionBlinkSlow:
				r.blink = BlinkSlow
			case s.N == GraphicRenditionBlinkFast:
				r.blink = BlinkFast
			default:
				return nil, fmt.Errorf("unhandled graphics rendition %d", s.N)
			}
		default:
			return nil, fmt.Errorf("unhandled sequence %T", s)
		}
		for r.col > r.screenWidth {
			r.col -= r.screenWidth
			r.row++
		}
	}

	var width int
	for _, r := range r.rows {
		if len(r) > width {
			width = len(r)
		}
	}
	height := len(r.rows)

	pix := make([]Pixel, 0, width*height)
	for _, row := range r.rows {
		pix = append(pix, row...)
		for i := 0; i < width-len(row); i++ {
			pix = append(pix, Pixel{})
		}
	}

	img := &Image{
		Pix:    pix,
		Width:  width,
		Height: height,
	}
	return img, nil
}

// func (r *Renderer) RenderSequence(seq []Sequence) (*Image, error) {
// 	var rows [][]Pixel
// 	screenWidth := 80 // TODO: should be configurable and optional
// 	row := 1
// 	col := 1
// 	var savedCursors [][2]int
// 	fgBold := byte(0)
// 	bgBold := byte(0)
// 	bgColor := byte(0)
// 	fgColor := byte(7)
// 	blink := BlinkNone
// 	for _, s := range seq {
// 		switch s := s.(type) {
// 		case Character:
// 			switch s.C {
// 			// case lf:
// 			// 	row++
// 			// case cr:
// 			// 	col = 1
// 			case lf:
// 				row++
// 				col = 1
// 			case cr:
// 			default:
// 				y := row - 1
// 				x := col - 1
// 				// TODO: Not sure how best to handle out of bounds values
// 				if x < 0 {
// 					x = 0
// 				}
// 				if y < 0 {
// 					y = 0
// 				}
// 				for len(rows) <= y {
// 					rows = append(rows, nil)
// 				}
// 				r := rows[y]
// 				for len(r) <= x {
// 					r = append(r, Pixel{})
// 				}
// 				rows[y] = r
// 				r[x] = Pixel{
// 					C:               s.C,
// 					ForegroundColor: fgColor + fgBold,
// 					BackgroundColor: bgColor + bgBold,
// 					Blink:           blink,
// 					// TODO: attributes
// 				}
// 				col++
// 			}
// 		case Clear:
// 			switch s.Type {
// 			case ClearTypeScreen:
// 				// TODO: reset row and col?
// 				rows = rows[:0]
// 			default:
// 				return nil, fmt.Errorf("unhandled clear type %d", s.Type)
// 			}
// 		case CursorBackward:
// 			col -= s.N
// 		case CursorDown:
// 			row += s.N
// 		case CursorForward:
// 			col += s.N
// 		case CursorUp:
// 			row -= s.N
// 		case MoveCursorTo:
// 			row = s.Row
// 			col = s.Col
// 		case RestoreCursorPosition:
// 			x := savedCursors[len(savedCursors)-1]
// 			savedCursors = savedCursors[:len(savedCursors)-1]
// 			row = x[0]
// 			col = x[1]
// 		case SaveCursorPosition:
// 			savedCursors = append(savedCursors, [2]int{row, col})
// 		case SelectGraphicsRendition:
// 			switch {
// 			case s.N == GraphicsRenditionReset:
// 				bgColor = 0
// 				fgColor = 7
// 				fgBold = 0
// 				blink = BlinkNone
// 			case s.N == GraphicsRenditionBold:
// 				fgBold = 8
// 			case s.N == GraphicsrenditionDefaultTextColor:
// 				fgColor = 7
// 				// TODO: should this also clear bold or not?
// 				fgBold = 0
// 			case s.N >= GraphicsRenditionSetTextColor0 && s.N <= GraphicsRenditionSetTextColor7:
// 				fgColor = byte(s.N - GraphicsRenditionSetTextColor0)
// 			case s.N >= GraphicsRenditionSetBackgroundColor0 && s.N <= GraphicsRenditionSetBackgroundColor7:
// 				bgColor = byte(s.N - GraphicsRenditionSetBackgroundColor0)
// 			case s.N == GraphicRenditionBlinkSlow:
// 				blink = BlinkSlow
// 			case s.N == GraphicRenditionBlinkFast:
// 				blink = BlinkFast
// 			default:
// 				return nil, fmt.Errorf("unhandled graphics rendition %d", s.N)
// 			}
// 		default:
// 			return nil, fmt.Errorf("unhandled sequence %T", s)
// 		}
// 		for col > screenWidth {
// 			col -= screenWidth
// 			row++
// 		}
// 	}
//
// 	var width int
// 	for _, r := range rows {
// 		if len(r) > width {
// 			width = len(r)
// 		}
// 	}
// 	height := len(rows)
//
// 	pix := make([]Pixel, 0, width*height)
// 	for _, row := range rows {
// 		pix = append(pix, row...)
// 		for i := 0; i < width-len(row); i++ {
// 			pix = append(pix, Pixel{})
// 		}
// 	}
//
// 	img := &Image{
// 		Pix:    pix,
// 		Width:  width,
// 		Height: height,
// 	}
// 	return img, nil
// }
