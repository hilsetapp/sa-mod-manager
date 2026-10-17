package gui

import (
	_ "embed"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// Шрифт Inter лежит внутри программы: русский текст выглядит нормально
// на любой Windows, ничего доустанавливать не нужно.
//
//go:embed InterVariable.ttf
var interTTF []byte

func newTheme() *material.Theme {
	th := material.NewTheme()
	faces := []font.FontFace{}
	if f, err := opentype.Parse(interTTF); err == nil {
		faces = append(faces, font.FontFace{Font: font.Font{Typeface: "Inter"}, Face: f})
	}
	faces = append(faces, gofont.Collection()...)
	th.Shaper = text.NewShaper(text.WithCollection(faces))
	th.Face = "Inter"
	th.TextSize = 14
	th.Palette = material.Palette{
		Bg:         colBg,
		Fg:         colText,
		ContrastBg: colAccent,
		ContrastFg: colOnAccent,
	}
	return th
}
