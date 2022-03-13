package ansi

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"strconv"
)

const (
	lf  = 10
	cr  = 13
	eof = 26 // DOS
	esc = 27
)

// VGAPalette is the palette for VGA.
var VGAPalette = []color.RGBA{
	{R: 0, G: 0, B: 0, A: 255},
	{R: 170, G: 0, B: 0, A: 255},
	{R: 0, G: 170, B: 0, A: 255},
	{R: 170, G: 85, B: 0, A: 255},
	{R: 0, G: 0, B: 170, A: 255},
	{R: 170, G: 0, B: 170, A: 255},
	{R: 0, G: 170, B: 170, A: 255},
	{R: 170, G: 170, B: 170, A: 255},
	{R: 85, G: 85, B: 85, A: 255},
	{R: 255, G: 85, B: 85, A: 255},
	{R: 85, G: 255, B: 85, A: 255},
	{R: 255, G: 255, B: 85, A: 255},
	{R: 85, G: 85, B: 255, A: 255},
	{R: 255, G: 85, B: 255, A: 255},
	{R: 85, G: 255, B: 255, A: 255},
	{R: 255, G: 255, B: 255, A: 255},
}

type Image struct {
	Pix    []Pixel
	Width  int
	Height int
}

type Pixel struct {
	C               byte
	BackgroundColor byte
	ForegroundColor byte
	Blink           Blink
	// TODO: other attributes: ?
}

type Blink byte

const (
	BlinkNone Blink = iota
	BlinkSlow
	BlinkFast
)

type Parser struct {
	r io.ByteReader
	b byte
}

type Sequence interface {
}

type Character struct {
	C byte
}

type CursorUp struct {
	N int
}

type CursorDown struct {
	N int
}

type CursorForward struct {
	N int
}

type CursorBackward struct {
	N int
}

type MoveCursorTo struct {
	Row int
	Col int
}

type Clear struct {
	Type ClearType
}

type ClearType byte

const (
	ClearTypeToEndOfScreen       ClearType = 0
	ClearTypeToBeginningOfScreen ClearType = 1
	ClearTypeScreen              ClearType = 2
	ClearTypeScreenAndScrollback ClearType = 3
)

type SaveCursorPosition struct{}

type RestoreCursorPosition struct{}

type SelectGraphicsRendition struct {
	N GraphicsRendition
}

type GraphicsRendition byte

const (
	GraphicsRenditionReset               GraphicsRendition = 0
	GraphicsRenditionBold                GraphicsRendition = 1
	GraphicRenditionBlinkSlow            GraphicsRendition = 5
	GraphicRenditionBlinkFast            GraphicsRendition = 6
	GraphicsRenditionSetTextColor0       GraphicsRendition = 30
	GraphicsRenditionSetTextColor1       GraphicsRendition = 31
	GraphicsRenditionSetTextColor2       GraphicsRendition = 32
	GraphicsRenditionSetTextColor3       GraphicsRendition = 33
	GraphicsRenditionSetTextColor4       GraphicsRendition = 34
	GraphicsRenditionSetTextColor5       GraphicsRendition = 35
	GraphicsRenditionSetTextColor6       GraphicsRendition = 36
	GraphicsRenditionSetTextColor7       GraphicsRendition = 37
	GraphicsrenditionDefaultTextColor    GraphicsRendition = 39
	GraphicsRenditionSetBackgroundColor0 GraphicsRendition = 40
	GraphicsRenditionSetBackgroundColor1 GraphicsRendition = 41
	GraphicsRenditionSetBackgroundColor2 GraphicsRendition = 42
	GraphicsRenditionSetBackgroundColor3 GraphicsRendition = 43
	GraphicsRenditionSetBackgroundColor4 GraphicsRendition = 44
	GraphicsRenditionSetBackgroundColor5 GraphicsRendition = 45
	GraphicsRenditionSetBackgroundColor6 GraphicsRendition = 46
	GraphicsRenditionSetBackgroundColor7 GraphicsRendition = 47
)

func Parse(r io.ByteReader) (*Image, error) {
	p := NewParser(r)
	seq, err := p.ParseAll()
	if err != nil {
		return nil, err
	}
	return RenderSequence(seq)
}

func NewParser(r io.ByteReader) *Parser {
	return &Parser{r: r}
}

func RenderImage(ansiImage *Image) *image.Paletted {
	const fontWidth = 8
	const fontHeight = 16

	palette := make([]color.Color, len(VGAPalette))
	for i, c := range VGAPalette {
		palette[i] = c
	}
	img := image.NewPaletted(image.Rect(0, 0, ansiImage.Width*fontWidth, ansiImage.Height*fontHeight), palette)
	for y := 0; y < ansiImage.Height; y++ {
		for x := 0; x < ansiImage.Width; x++ {
			p := ansiImage.Pix[y*ansiImage.Width+x]
			o := x*fontWidth + y*fontHeight*img.Bounds().Dx()
			fc := VGAFont16[int(p.C)*16 : int(p.C)*16+16]
			for fy := 0; fy < fontHeight; fy++ {
				for fx := 0; fx < fontWidth; fx++ {
					if (fc[fy]>>uint(fontWidth-1-fx))&1 == 0 {
						img.Pix[o+fy*img.Bounds().Dx()+fx] = p.BackgroundColor
					} else {
						img.Pix[o+fy*img.Bounds().Dx()+fx] = p.ForegroundColor
					}
				}
			}
		}
	}
	return img
}

func (p *Parser) ParseAll() (seq []Sequence, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				if e != io.EOF {
					err = e
				}
			} else {
				err = fmt.Errorf("runtime error: %v", r)
			}
		}
	}()

	for {
		p.next()
		switch p.b {
		case eof: // this should be optional
			panic(io.EOF)
		case esc:
			p.next()
			if p.b != 0x5b {
				panic(fmt.Errorf("invalid escape sequence, expected 0x5b found 0x%02x", p.b))
			}
			p.next()
			var nums []int
		escLoop:
			for {
				switch {
				case p.b == ';':
					nums = append(nums, 0)
					p.next()
				case p.b >= '0' && p.b <= '9':
					nums = append(nums, p.readNum())
					if p.b == ';' {
						p.next()
					}
				default:
					break escLoop
				}
			}
			ctrl := p.b
			switch ctrl {
			case 'A', 'B', 'C', 'D':
				// Moves the cursor n (default 1) cells in the given direction. If the cursor
				// is already at the edge of the screen, this has no effect.
				if len(nums) == 0 {
					nums = append(nums, 1)
				}
				n := nums[0]
				switch ctrl {
				case 'A':
					seq = append(seq, CursorUp{N: n})
				case 'B':
					seq = append(seq, CursorDown{N: n})
				case 'C':
					seq = append(seq, CursorForward{N: n})
				case 'D':
					seq = append(seq, CursorBackward{N: n})
				}
			case 'H':
				// Moves the cursor to row n, column m. The values are 1-based, and default
				// to 1 (top left corner) if omitted. A sequence such as CSI ;5H is a synonym
				// for CSI 1;5H as well as CSI 17;H is the same as CSI 17H and CSI 17;1H
				for len(nums) < 2 {
					nums = append(nums, 1)
				}
				for i, n := range nums {
					if n == 0 {
						nums[i] = 1
					}
				}
				row := nums[0]
				col := nums[1]
				seq = append(seq, MoveCursorTo{Row: row, Col: col})
			case 'J':
				// Clears part of the screen. If n is 0 (or missing), clear from
				// cursor to end of screen. If n is 1, clear from cursor to beginning
				// of the screen. If n is 2, clear entire screen (and moves cursor
				// to upper left on DOS ANSI.SYS). If n is 3, clear entire screen and
				// delete all lines saved in the scrollback buffer (this feature was
				// added for xterm and is supported by other terminal applications).
				if len(nums) == 0 {
					nums = append(nums, 0)
				}
				seq = append(seq, Clear{Type: ClearType(nums[0])})
			case 'm':
				// Sets SGR parameters, including text color. After CSI can be zero or more parameters
				// separated with ;. With no parameters, CSI m is treated as CSI 0 m (reset / normal),
				// which is typical of most of the ANSI escape sequences.
				if len(nums) == 0 {
					nums = append(nums, 0)
				}
				for _, m := range nums {
					seq = append(seq, SelectGraphicsRendition{N: GraphicsRendition(m)})
				}
			case 's':
				// Saves the cursor position.
				seq = append(seq, SaveCursorPosition{})
			case 'u':
				// Restores the cursor position.
				seq = append(seq, RestoreCursorPosition{})
			case 'M': // TODO ?
			default:
				panic(fmt.Errorf("unknown escape sequence ESC[%+v%c", nums, ctrl))
			}
		default:
			seq = append(seq, Character{C: p.b})
		}
	}
}

func (p *Parser) next() {
	b, err := p.r.ReadByte()
	if err != nil {
		panic(err)
	}
	p.b = b
}

func (p *Parser) readNum() int {
	var buf []byte
	for p.b >= '0' && p.b <= '9' {
		buf = append(buf, p.b)
		p.next()
	}
	n, err := strconv.Atoi(string(buf))
	if err != nil {
		panic(fmt.Errorf("failed to parse number %q", string(buf)))
	}
	return n
}

var sgrDesc = [...]string{
	0: `Reset / Normal`,
	1: `Bold or increased intensity`,
	2: `Faint (decreased intensity)`,
	3: `Italic: on`,
	4: `Underline: Single`,
	5: `Blink: Slow`,
	6: `Blink: Rapid`,
	7: `Image: Negative	inverse or reverse; swap foreground and background (reverse video)`,
	8: `Conceal	Not widely supported.`,
	9: `Crossed-out	Characters legible, but marked for deletion.`,
	10: `Primary(default) font	`,
	11: `Alternate font 1`,
	12: `Alternate font 2`,
	13: `Alternate font 3`,
	14: `Alternate font 4`,
	15: `Alternate font 5`,
	16: `Alternate font 6`,
	17: `Alternate font 7`,
	18: `Alternate font 8`,
	19: `Alternate font 9`,
	20: `Fraktur`,
	21: `Bold: off or Underline: Double	Bold off not widely supported; double underline`,
	22: `Normal color or intensity`,
	23: `Not italic, not Fraktur`,
	24: `Underline: None`,
	25: `Blink: off`,
	26: `Reserved`,
	27: `Image: Positive`,
	28: `Reveal	conceal off`,
	29:  `Not crossed out`,
	30:  `Set text color 0`,
	31:  `Set text color 1`,
	32:  `Set text color 2`,
	33:  `Set text color 3`,
	34:  `Set text color 4`,
	35:  `Set text color 5`,
	36:  `Set text color 6`,
	37:  `Set text color 7`,
	38:  `Reserved for extended set foreground color`,
	39:  `Default text color (foreground)`,
	40:  "Set background color 0",
	41:  "Set background color 1",
	42:  "Set background color 2",
	43:  "Set background color 3",
	44:  "Set background color 4",
	45:  "Set background color 5",
	46:  "Set background color 6",
	47:  "Set background color 7",
	48:  `Reserved for extended set background color`,
	49:  `Default background color`,
	50:  `Reserved`,
	51:  `Framed`,
	52:  `Encircled`,
	53:  `Overlined`,
	54:  `Not framed or encircled`,
	55:  `Not overlined`,
	56:  `Reserved`,
	57:  `Reserved`,
	58:  `Reserved`,
	59:  `Reserved`,
	60:  `ideogram underline or right side line`,
	61:  `ideogram double underline or double line on the right side`,
	62:  `ideogram overline or left side line`,
	63:  `ideogram double overline or double line on the left side`,
	64:  `ideogram stress marking`,
	65:  `ideogram attributes off`,
	90:  `Set foreground text color, high intensity 0`,
	91:  `Set foreground text color, high intensity 1`,
	92:  `Set foreground text color, high intensity 2`,
	93:  `Set foreground text color, high intensity 3`,
	94:  `Set foreground text color, high intensity 4`,
	95:  `Set foreground text color, high intensity 5`,
	96:  `Set foreground text color, high intensity 6`,
	97:  `Set foreground text color, high intensity 7`,
	100: `Set background color, high intensity 0`,
	101: `Set background color, high intensity 1`,
	102: `Set background color, high intensity 2`,
	103: `Set background color, high intensity 3`,
	104: `Set background color, high intensity 4`,
	105: `Set background color, high intensity 5`,
	106: `Set background color, high intensity 6`,
	107: `Set background color, high intensity 7`,
}

var PCASCIIToUnicode = [256]rune{
	0:   0,
	1:   1,
	2:   2,
	3:   3,
	4:   4,
	5:   5,
	6:   6,
	7:   7,
	8:   8,
	9:   9,
	10:  10,
	11:  11,
	12:  12,
	13:  13,
	14:  14,
	15:  15,
	16:  16,
	17:  17,
	18:  18,
	19:  19,
	20:  20,
	21:  21,
	22:  22,
	23:  23,
	24:  24,
	25:  25,
	26:  26,
	27:  27,
	28:  28,
	29:  29,
	30:  30,
	31:  31,
	32:  32,
	33:  33,
	34:  34,
	35:  35,
	36:  36,
	37:  37,
	38:  38,
	39:  39,
	40:  40,
	41:  41,
	42:  42,
	43:  43,
	44:  44,
	45:  45,
	46:  46,
	47:  47,
	48:  48,
	49:  49,
	50:  50,
	51:  51,
	52:  52,
	53:  53,
	54:  54,
	55:  55,
	56:  56,
	57:  57,
	58:  58,
	59:  59,
	60:  60,
	61:  61,
	62:  62,
	63:  63,
	64:  64,
	65:  65,
	66:  66,
	67:  67,
	68:  68,
	69:  69,
	70:  70,
	71:  71,
	72:  72,
	73:  73,
	74:  74,
	75:  75,
	76:  76,
	77:  77,
	78:  78,
	79:  79,
	80:  80,
	81:  81,
	82:  82,
	83:  83,
	84:  84,
	85:  85,
	86:  86,
	87:  87,
	88:  88,
	89:  89,
	90:  90,
	91:  91,
	92:  92,
	93:  93,
	94:  94,
	95:  95,
	96:  96,
	97:  97,
	98:  98,
	99:  99,
	100: 100,
	101: 101,
	102: 102,
	103: 103,
	104: 104,
	105: 105,
	106: 106,
	107: 107,
	108: 108,
	109: 109,
	110: 110,
	111: 111,
	112: 112,
	113: 113,
	114: 114,
	115: 115,
	116: 116,
	117: 117,
	118: 118,
	119: 119,
	120: 120,
	121: 121,
	122: 122,
	123: 123,
	124: 124,
	125: 125,
	126: 126,
	127: 127,
	128: 0x00C7, // Ç : latin capital letter c with cedilla
	129: 0x00FC, // ü : latin small letter u with diaeresis
	130: 0x00E9, // é : latin small letter e with acute
	131: 0x00E2, // â : latin small letter a with circumflex
	132: 0x00E4, // ä : latin small letter a with diaeresis
	133: 0x00E0, // à : latin small letter a with grave
	134: 0x00E5, // å : latin small letter a with ring above
	135: 0x00E7, // ç : latin small letter c with cedilla
	136: 0x00EA, // ê : latin small letter e with circumflex
	137: 0x00EB, // ë : latin small letter e with diaeresis
	138: 0x00E8, // è : latin small letter e with grave
	139: 0x00EF, // ï : latin small letter i with diaeresis
	140: 0x00EE, // î : latin small letter i with circumflex
	141: 0x00EC, // ì : latin small letter i with grave
	142: 0x00C4, // Ä : latin capital letter a with diaeresis
	143: 0x00C5, // Å : latin capital letter a with ring above
	144: 0x00C9, // É : latin capital letter e with acute
	145: 0x00E6, // æ : latin small ligature ae
	146: 0x00C6, // Æ : latin capital ligature ae
	147: 0x00F4, // ô : latin small letter o with circumflex
	148: 0x00F6, // ö : latin small letter o with diaeresis
	149: 0x00F2, // ò : latin small letter o with grave
	150: 0x00FB, // û : latin small letter u with circumflex
	151: 0x00F9, // ù : latin small letter u with grave
	152: 0x00FF, // ÿ : latin small letter y with diaeresis
	153: 0x00D6, // Ö : latin capital letter o with diaeresis
	154: 0x00DC, // Ü : latin capital letter u with diaeresis
	155: 0x00A2, // ¢ : cent sign
	156: 0x00A3, // £ : pound sign
	157: 0x00A5, // ¥ : yen sign
	158: 0x20A7, // ₧ : peseta sign
	159: 0x0192, // ƒ : latin small letter f with hook
	160: 0x00E1, // á : latin small letter a with acute
	161: 0x00ED, // í : latin small letter i with acute
	162: 0x00F3, // ó : latin small letter o with acute
	163: 0x00FA, // ú : latin small letter u with acute
	164: 0x00F1, // ñ : latin small letter n with tilde
	165: 0x00D1, // Ñ : latin capital letter n with tilde
	166: 0x00AA, // ª : feminine ordinal indicator
	167: 0x00BA, // º : masculine ordinal indicator
	168: 0x00BF, // ¿ : inverted question mark
	169: 0x2310, // ⌐ : reversed not sign
	170: 0x00AC, // ¬ : not sign
	171: 0x00BD, // ½ : vulgar fraction one half
	172: 0x00BC, // ¼ : vulgar fraction one quarter
	173: 0x00A1, // ¡ : inverted exclamation mark
	174: 0x00AB, // « : left-pointing double angle quotation mark
	175: 0x00BB, // » : right-pointing double angle quotation mark
	176: 0x2591, // ░ : light shade
	177: 0x2592, // ▒ : medium shade
	178: 0x2593, // ▓ : dark shade
	179: 0x2502, // │ : box drawings light vertical
	180: 0x2524, // ┤ : box drawings light vertical and left
	181: 0x2561, // ╡ : box drawings vertical single and left double
	182: 0x2562, // ╢ : box drawings vertical double and left single
	183: 0x2556, // ╖ : box drawings down double and left single
	184: 0x2555, // ╕ : box drawings down single and left double
	185: 0x2563, // ╣ : box drawings double vertical and left
	186: 0x2551, // ║ : box drawings double vertical
	187: 0x2557, // ╗ : box drawings double down and left
	188: 0x255D, // ╝ : box drawings double up and left
	189: 0x255C, // ╜ : box drawings up double and left single
	190: 0x255B, // ╛ : box drawings up single and left double
	191: 0x2510, // ┐ : box drawings light down and left
	192: 0x2514, // └ : box drawings light up and right
	193: 0x2534, // ┴ : box drawings light up and horizontal
	194: 0x252C, // ┬ : box drawings light down and horizontal
	195: 0x251C, // ├ : box drawings light vertical and right
	196: 0x2500, // ─ : box drawings light horizontal
	197: 0x253C, // ┼ : box drawings light vertical and horizontal
	198: 0x255E, // ╞ : box drawings vertical single and right double
	199: 0x255F, // ╟ : box drawings vertical double and right single
	200: 0x255A, // ╚ : box drawings double up and right
	201: 0x2554, // ╔ : box drawings double down and right
	202: 0x2569, // ╩ : box drawings double up and horizontal
	203: 0x2566, // ╦ : box drawings double down and horizontal
	204: 0x2560, // ╠ : box drawings double vertical and right
	205: 0x2550, // ═ : box drawings double horizontal
	206: 0x256C, // ╬ : box drawings double vertical and horizontal
	207: 0x2567, // ╧ : box drawings up single and horizontal double
	208: 0x2568, // ╨ : box drawings up double and horizontal single
	209: 0x2564, // ╤ : box drawings down single and horizontal double
	210: 0x2565, // ╥ : box drawings down double and horizontal single
	211: 0x2559, // ╙ : box drawings up double and right single
	212: 0x2558, // ╘ : box drawings up single and right double
	213: 0x2552, // ╒ : box drawings down single and right double
	214: 0x2553, // ╓ : box drawings down double and right single
	215: 0x256B, // ╫ : box drawings vertical double and horizontal single
	216: 0x256A, // ╪ : box drawings vertical single and horizontal double
	217: 0x2518, // ┘ : box drawings light up and left
	218: 0x250C, // ┌ : box drawings light down and right
	219: 0x2588, // █ : full block
	220: 0x2584, // ▄ : lower half block
	221: 0x258C, // ▌ : left half block
	222: 0x2590, // ▐ : right half block
	223: 0x2580, // ▀ : upper half block
	224: 0x03B1, // α : greek small letter alpha
	225: 0x00DF, // ß : latin small letter sharp s
	226: 0x0393, // Γ : greek capital letter gamma
	227: 0x03C0, // π : greek small letter pi
	228: 0x03A3, // Σ : greek capital letter sigma
	229: 0x03C3, // σ : greek small letter sigma
	230: 0x00B5, // µ : micro sign
	231: 0x03C4, // τ : greek small letter tau
	232: 0x03A6, // Φ : greek capital letter phi
	233: 0x0398, // Θ : greek capital letter theta
	234: 0x03A9, // Ω : greek capital letter omega
	235: 0x03B4, // δ : greek small letter delta
	236: 0x221E, // ∞ : infinity
	237: 0x03C6, // φ : greek small letter phi
	238: 0x03B5, // ε : greek small letter epsilon
	239: 0x2229, // ∩ : intersection
	240: 0x2261, // ≡ : identical to
	241: 0x00B1, // ± : plus-minus sign
	242: 0x2265, // ≥ : greater-than or equal to
	243: 0x2264, // ≤ : less-than or equal to
	244: 0x2320, // ⌠ : top half integral
	245: 0x2321, // ⌡ : bottom half integral
	246: 0x00F7, // ÷ : division sign
	247: 0x2248, // ≈ : almost equal to
	248: 0x00B0, // ° : degree sign
	249: 0x2219, // ∙ : bullet operator
	250: 0x00B7, // · : middle dot
	251: 0x221A, // √ : square root
	252: 0x207F, // ⁿ : superscript latin small letter n
	253: 0x00B2, // ² : superscript two
	254: 0x25A0, // ■ : black square
	255: 0x00A0, //   : no-break space
}
