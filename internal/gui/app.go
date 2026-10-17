// Пакет gui рисует окно менеджера: список модов, переключатели,
// предупреждения о конфликтах, файлы мода, профили сборок и журнал.
package gui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/hilsetapp/sa-mod-manager/internal/config"
	"github.com/hilsetapp/sa-mod-manager/internal/conflicts"
	"github.com/hilsetapp/sa-mod-manager/internal/game"
	"github.com/hilsetapp/sa-mod-manager/internal/library"
	"github.com/hilsetapp/sa-mod-manager/internal/profiles"
)

type ui struct {
	th      *material.Theme
	version string
	w       *app.Window

	mu            sync.Mutex
	gamePath      string
	info          game.Info
	mods          []library.Mod
	found         []conflicts.Entry
	profs         []profiles.Profile
	selected      string
	journal       []string
	busy          string
	confirmDelete string
	started       bool

	pathEditor  widget.Editor
	profEditor  widget.Editor
	bPickFolder widget.Clickable
	bApplyPath  widget.Clickable
	bAddZip     widget.Clickable
	bRefresh    widget.Clickable
	bConflicts  widget.Clickable
	bToggle     widget.Clickable
	bDelete     widget.Clickable
	bSaveProf   widget.Clickable

	modList  layout.List
	sideList layout.List
	logList  layout.List

	switches  map[string]*widget.Bool
	rows      map[string]*widget.Clickable
	applyBtns map[string]*widget.Clickable
}

// gtxGap - промежуток между строками журнала в пикселях.
const gtxGap = 2

// Run открывает окно и работает, пока его не закроют.
func Run(gamePath, version string) {
	u := &ui{
		th:        newTheme(),
		version:   version,
		gamePath:  gamePath,
		switches:  map[string]*widget.Bool{},
		rows:      map[string]*widget.Clickable{},
		applyBtns: map[string]*widget.Clickable{},
	}
	u.pathEditor.SingleLine = true
	u.pathEditor.SetText(gamePath)
	u.profEditor.SingleLine = true
	// списки строго вертикальные: у нулевого значения Axis список горизонтальный
	u.modList.Axis = layout.Vertical
	u.sideList.Axis = layout.Vertical
	u.logList.Axis = layout.Vertical
	u.logList.ScrollToEnd = true
	u.logList.Gap = gtxGap

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Менеджер модов GTA San Andreas"), app.Size(unit.Dp(1140), unit.Dp(760)))
		u.w = w
		var ops op.Ops
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				u.layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

// ---------- состояние ----------

type snapshot struct {
	gamePath string
	info     game.Info
	mods     []library.Mod
	found    []conflicts.Entry
	profs    []profiles.Profile
	selected string
	journal  []string
	busy     string
	delete   string
}

func (u *ui) snap() snapshot {
	u.mu.Lock()
	defer u.mu.Unlock()
	return snapshot{
		gamePath: u.gamePath,
		info:     u.info,
		mods:     append([]library.Mod(nil), u.mods...),
		found:    append([]conflicts.Entry(nil), u.found...),
		profs:    append([]profiles.Profile(nil), u.profs...),
		selected: u.selected,
		journal:  append([]string(nil), u.journal...),
		busy:     u.busy,
		delete:   u.confirmDelete,
	}
}

func (u *ui) say(format string, args ...any) {
	line := time.Now().Format("15:04:05") + " " + fmt.Sprintf(format, args...)
	u.mu.Lock()
	u.journal = append(u.journal, line)
	if len(u.journal) > 400 {
		u.journal = u.journal[len(u.journal)-400:]
	}
	path := u.gamePath
	u.mu.Unlock()
	if path != "" {
		dir := filepath.Join(path, "modmanager")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			f, err := os.OpenFile(filepath.Join(dir, "journal.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				fmt.Fprintln(f, line)
				f.Close()
			}
		}
	}
	u.invalidate()
}

func (u *ui) setBusy(text string) {
	u.mu.Lock()
	u.busy = text
	u.mu.Unlock()
	u.invalidate()
}

func (u *ui) invalidate() {
	if u.w != nil {
		u.w.Run(u.w.Invalidate)
	}
}

// refresh перечитывает папку игры и библиотеку модов в отдельной горутине,
// чтобы окно не замирало на большой сборке.
func (u *ui) refresh(why string) {
	u.setBusy("смотрю папку игры")
	go func() {
		u.mu.Lock()
		path := u.gamePath
		u.mu.Unlock()

		info := game.Detect(path)
		var mods []library.Mod
		var found []conflicts.Entry
		if path != "" && info.Exists {
			st := library.New(path)
			if list, err := st.List(); err == nil {
				mods = list
			} else {
				u.say("библиотеку не прочитать: %v", err)
			}
			found = conflicts.Check(mods, info.InStock)
		}
		ps, _ := profiles.New(path).List()

		u.mu.Lock()
		u.info, u.mods, u.found, u.profs = info, mods, found, ps
		enabled := 0
		for _, m := range mods {
			if m.Enabled {
				enabled++
			}
		}
		if _, ok := findMod(mods, u.selected); !ok {
			u.selected = ""
		}
		u.busy = ""
		u.mu.Unlock()

		u.say("%s: модов %d, включено %d, конфликтов %d", why, len(mods), enabled, len(found))
		u.invalidate()
	}()
}

func (u *ui) setGamePath(path string) {
	path = strings.TrimSpace(path)
	u.mu.Lock()
	u.gamePath = path
	u.mu.Unlock()
	_ = config.Save(config.Data{GamePath: path})
	u.pathEditor.SetText(path)
	u.say("папка игры: %s", path)
	u.refresh("проверил папку")
}

func (u *ui) toggleMod(slug string, on bool) {
	go func() {
		u.mu.Lock()
		path := u.gamePath
		u.mu.Unlock()
		if err := library.New(path).SetEnabled(slug, on); err != nil {
			u.say("не вышло переключить %s: %v", slug, err)
			u.refresh("после ошибки")
			return
		}
		word := "выключил"
		if on {
			word = "включил"
		}
		u.say("%s мод %s", word, slug)
		u.refresh("после переключения")
	}()
}

func (u *ui) importZip(zipPath string) {
	u.setBusy("разбираю архив")
	go func() {
		u.mu.Lock()
		path := u.gamePath
		u.mu.Unlock()
		m, err := library.New(path).ImportZip(zipPath)
		if err != nil {
			u.say("архив не добавился: %v", err)
			u.mu.Lock()
			u.busy = ""
			u.mu.Unlock()
			u.invalidate()
			return
		}
		warn := ""
		if w := m.Warnings(); len(w) > 0 {
			warn = fmt.Sprintf(" (замечание: %s)", w[0])
		}
		u.say("добавил мод «%s»: файлов %d, пока выключен%s", m.Name, len(m.Files), warn)
		u.refresh("после импорта")
	}()
}

func (u *ui) removeMod(slug string, enabled bool) {
	go func() {
		u.mu.Lock()
		path := u.gamePath
		u.mu.Unlock()
		if err := library.New(path).Remove(slug, enabled); err != nil {
			u.say("удалить не вышло: %v", err)
			u.refresh("после ошибки")
			return
		}
		u.mu.Lock()
		if u.selected == slug {
			u.selected = ""
		}
		u.confirmDelete = ""
		u.mu.Unlock()
		u.say("удалил мод %s (папка и файлы стёрты)", slug)
		u.refresh("после удаления")
	}()
}

func (u *ui) saveProfile(name string) {
	u.mu.Lock()
	path := u.gamePath
	var on []string
	for _, m := range u.mods {
		if m.Enabled {
			on = append(on, m.Slug)
		}
	}
	u.mu.Unlock()
	if name == "" {
		name = "сборка"
	}
	if err := profiles.New(path).Save(profiles.Profile{Name: name, Mods: on}); err != nil {
		u.say("профиль не сохранился: %v", err)
		return
	}
	u.say("профиль «%s» сохранён: включено модов %d", name, len(on))
	u.refresh("после сохранения профиля")
}

// applyProfile включает только те моды, что записаны в профиле.
func (u *ui) applyProfile(p profiles.Profile) {
	u.setBusy("применяю профиль")
	go func() {
		u.mu.Lock()
		path := u.gamePath
		mods := append([]library.Mod(nil), u.mods...)
		u.mu.Unlock()

		want := map[string]bool{}
		for _, s := range p.Mods {
			want[s] = true
		}
		st := library.New(path)
		changed := 0
		for _, m := range mods {
			on := want[m.Slug]
			if on == m.Enabled {
				continue
			}
			if err := st.SetEnabled(m.Slug, on); err != nil {
				u.say("мод %s не переключился: %v", m.Slug, err)
				continue
			}
			changed++
		}
		u.say("профиль «%s» применён: переключил модов %d", p.Name, changed)
		u.refresh("после профиля")
	}()
}

func findMod(mods []library.Mod, slug string) (library.Mod, bool) {
	for _, m := range mods {
		if m.Slug == slug {
			return m, true
		}
	}
	return library.Mod{}, false
}

// ---------- отрисовка ----------

func (u *ui) layout(gtx layout.Context) layout.Dimensions {
	s := u.snap()
	// первый кадр: если папку игры помним с прошлого раза, смотрим её сразу
	if !u.started {
		u.started = true
		if s.gamePath != "" {
			u.refresh("старт")
		}
	}
	paint.FillShape(gtx.Ops, colBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.header(gtx, s) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.statusStrip(gtx, s) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.modsPanel(gtx, s) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.sidePanel(gtx, s) }),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.journalBar(gtx, s) }),
	)
}

func (u *ui) header(gtx layout.Context, s snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(14), Left: unit.Dp(18), Right: unit.Dp(18), Bottom: unit.Dp(6)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(material.H6(u.th, "Менеджер модов GTA San Andreas").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, u.label("v"+u.version, 12, colDim, 1))
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Dimensions{Size: gtx.Constraints.Min}
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if s.busy == "" {
								return layout.Dimensions{}
							}
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min = image.Pt(gtx.Dp(18), gtx.Dp(18))
									gtx.Constraints.Max = gtx.Constraints.Min
									return material.Loader(u.th).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, u.label(s.busy, 12, colDim, 1))
								}),
							)
						}),
					)
				})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Bottom: unit.Dp(10)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(u.label("Папка игры:", 13, colDim, 1)),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return surface(gtx, colCard, gtx.Dp(8), func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx,
										material.Editor(u.th, &u.pathEditor, `C:\Games\GTA San Andreas`).Layout)
								})
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if u.bApplyPath.Clicked(gtx) {
								u.setGamePath(u.pathEditor.Text())
							}
							return material.Button(u.th, &u.bApplyPath, "Применить").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if u.bPickFolder.Clicked(gtx) {
								if p := pickFolderDialog("Выбери папку с GTA San Andreas"); p != "" {
									u.setGamePath(p)
								}
							}
							return layout.Inset{Left: unit.Dp(6)}.Layout(gtx,
								material.Button(u.th, &u.bPickFolder, "Выбрать папку").Layout)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if u.bAddZip.Clicked(gtx) {
								if p := pickZipDialog(); p != "" {
									u.importZip(p)
								}
							}
							return layout.Inset{Left: unit.Dp(6)}.Layout(gtx,
								material.Button(u.th, &u.bAddZip, "Добавить мод из архива").Layout)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if u.bRefresh.Clicked(gtx) {
								u.refresh("по кнопке")
							}
							return layout.Inset{Left: unit.Dp(6)}.Layout(gtx,
								material.Button(u.th, &u.bRefresh, "Обновить").Layout)
						}),
					)
				})
		}),
	)
}

// statusStrip показывает главное: что с игрой и что с конфликтами.
func (u *ui) statusStrip(gtx layout.Context, s snapshot) layout.Dimensions {
	type line struct {
		bg, fg color.NRGBA
		text   string
	}
	var lines []line
	if s.gamePath == "" {
		lines = append(lines, line{colWarnBg, colWarnText, "Папка игры не выбрана. Нажми «Выбрать папку» или впиши путь руками."})
	} else if !s.info.LooksLikeGTA {
		problems := strings.Join(s.info.Problems, "; ")
		lines = append(lines, line{colWarnBg, colWarnText, "В этой папке игра не угадывается: " + problems})
	} else {
		lines = append(lines, line{colSoft, colText, fmt.Sprintf(
			"gta_sa.exe есть · имён в gta3.img: %d · ModLoader: %s · CLEO: %s",
			s.info.IMGNameCount, yesNo(s.info.HasModLoader), yesNo(s.info.HasCLEO))})
		if !s.info.HasModLoader {
			lines = append(lines, line{colWarnBg, colWarnText,
				"Не видно modloader.asi: без него моды из этой папки игра не подхватит. Поставь ModLoader и обнови."})
		}
	}
	now := 0
	var first conflicts.Entry
	for _, e := range s.found {
		if e.Сейчас() {
			now++
			if now == 1 {
				first = e
			}
		}
	}
	if now > 0 {
		lines = append(lines, line{colBadBg, colBadText, fmt.Sprintf(
			"Конфликт прямо сейчас: %s (файл %s) делят моды %s. Отключи один из них.",
			first.Kind, first.Key, strings.Join(first.Mods, " и "))})
	} else if len(s.found) > 0 {
		lines = append(lines, line{colWarnBg, colWarnText, fmt.Sprintf(
			"Спокойные конфликты: %d файлов встречаются у нескольких модов. Пока они не мешают, но включишь оба - начнут.", len(s.found))})
	}

	if len(lines) == 0 {
		return layout.Dimensions{}
	}
	children := make([]layout.FlexChild, 0, len(lines))
	for _, l := range lines {
		l := l
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Bottom: unit.Dp(6)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return surface(gtx, l.bg, gtx.Dp(8), func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx,
							u.label(l.text, 13, l.fg, 3))
					})
				})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (u *ui) modsPanel(gtx layout.Context, s snapshot) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(9), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, gtx.Dp(10), func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Bottom: unit.Dp(8)}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								on := 0
								for _, m := range s.mods {
									if m.Enabled {
										on++
									}
								}
								txt := fmt.Sprintf("Моды: %d всего, %d включено", len(s.mods), on)
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(material.Subtitle1(u.th, txt).Layout),
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										if u.bConflicts.Clicked(gtx) {
											u.refresh("проверил конфликты")
										}
										return layout.Widget(func(gtx layout.Context) layout.Dimensions {
											return layout.E.Layout(gtx, material.Button(u.th, &u.bConflicts, "Проверить конфликты").Layout)
										})(gtx)
									}),
								)
							})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if len(s.mods) == 0 {
							return layout.Inset{Left: unit.Dp(14), Top: unit.Dp(10)}.Layout(gtx,
								u.label("Пока пусто. Нажми «Добавить мод из архива» и выбери zip.\nМоды из папки modloader тоже подхватятся сами.", 13, colDim, 4))
						}
						return u.modList.Layout(gtx, len(s.mods), func(gtx layout.Context, i int) layout.Dimensions {
							return u.modRow(gtx, s, s.mods[i])
						})
					}),
				)
			})
		})
	})
}

func (u *ui) modRow(gtx layout.Context, s snapshot, m library.Mod) layout.Dimensions {
	row := u.rows[m.Slug]
	if row == nil {
		row = &widget.Clickable{}
		u.rows[m.Slug] = row
	}
	selected := m.Slug == s.selected
	bg := color.NRGBA{}
	if selected {
		bg = colSoft
	}
	for row.Clicked(gtx) {
		u.mu.Lock()
		u.selected = m.Slug
		u.confirmDelete = ""
		u.mu.Unlock()
		// перерисовываем окно: иначе правая панель останется со старым модом
		u.invalidate()
	}
	return row.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, bg, gtx.Dp(8), func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(material.Body1(u.th, m.Name).Layout),
								layout.Rigid(u.label(u.modLine(s, m), 12, colDim, 2)),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							sw := u.switchFor(m)
							// клик по переключателю разбираем сами: иначе материал
							// перевернёт значение, а состояние мода не поменяется
							if sw.Update(gtx) {
								u.toggleMod(m.Slug, sw.Value)
							}
							return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.Switch(u.th, sw, "мод включён").Layout(gtx)
							})
						}),
					)
				})
		})
	})
}

// switchFor отдаёт переключатель мода и сразу обрабатывает клик по нему.
func (u *ui) switchFor(m library.Mod) *widget.Bool {
	sw := u.switches[m.Slug]
	if sw == nil {
		sw = &widget.Bool{}
		u.switches[m.Slug] = sw
	}
	sw.Value = m.Enabled
	return sw
}

func (u *ui) modLine(s snapshot, m library.Mod) string {
	parts := []string{m.Summary()}
	if len(m.Files) > 0 {
		parts = append(parts, fmt.Sprintf("файлов: %d", len(m.Files)))
	}
	stock := false
	for _, f := range m.Files {
		if s.info.InStock(f) {
			stock = true
			break
		}
	}
	if stock {
		parts = append(parts, "заменяет стандартные файлы")
	}
	shared := 0
	for _, e := range s.found {
		if e.Enabled > 1 {
			for _, name := range e.Mods {
				if name == m.Name {
					shared++
				}
			}
		}
	}
	if shared > 0 {
		parts = append(parts, fmt.Sprintf("делит файлы с другими: %d", shared))
	}
	if w := m.Warnings(); len(w) > 0 {
		parts = append(parts, "есть замечания")
	}
	return strings.Join(parts, " · ")
}

func (u *ui) sidePanel(gtx layout.Context, s snapshot) layout.Dimensions {
	width := gtx.Dp(unit.Dp(430))
	if max := gtx.Constraints.Max.X; max < width {
		width = max
	}
	gtx.Constraints.Min.X = width
	gtx.Constraints.Max.X = width
	return layout.Inset{Right: unit.Dp(18), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Блок «Файлы» показываем только когда мод выбран: пустая карточка
		// посреди панели выглядит как ошибка.
		sections := []layout.Widget{
			func(gtx layout.Context) layout.Dimensions { return u.sectionMod(gtx, s) },
		}
		if _, ok := findMod(s.mods, s.selected); ok {
			sections = append(sections, func(gtx layout.Context) layout.Dimensions { return u.sectionFiles(gtx, s) })
		}
		sections = append(sections,
			func(gtx layout.Context) layout.Dimensions { return u.sectionProblems(gtx, s) },
			func(gtx layout.Context) layout.Dimensions { return u.sectionProfiles(gtx, s) },
		)
		return u.sideList.Layout(gtx, len(sections), func(gtx layout.Context, i int) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return surface(gtx, colCard, gtx.Dp(10), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(14)).Layout(gtx, sections[i])
				})
			})
		})
	})
}

func (u *ui) sectionMod(gtx layout.Context, s snapshot) layout.Dimensions {
	m, ok := findMod(s.mods, s.selected)
	if !ok {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.Subtitle1(u.th, "Мод").Layout),
			layout.Rigid(u.label("Выбери мод слева, и здесь появится всё про него.", 13, colDim, 3)),
		)
	}
	state := "выключен"
	if m.Enabled {
		state = "включён в игру"
	}
	rows := []layout.FlexChild{
		layout.Rigid(material.Subtitle1(u.th, m.Name).Layout),
		layout.Rigid(u.label("папка: "+m.Slug+" · "+state, 12, colDim, 2)),
		layout.Rigid(u.label("файлов: "+fmt.Sprint(len(m.Files))+" · добавил: "+m.Added.Format("02.01.2006 15:04"), 12, colDim, 2)),
	}
	if m.Source != "" {
		rows = append(rows, layout.Rigid(u.label("откуда: "+m.Source, 12, colDim, 2)))
	}
	rows = append(rows,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						text := "Включить"
						if m.Enabled {
							text = "Выключить"
						}
						if u.bToggle.Clicked(gtx) {
							u.toggleMod(m.Slug, !m.Enabled)
						}
						return material.Button(u.th, &u.bToggle, text).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						text := "Удалить"
						if s.delete == m.Slug {
							text = "Точно удалить?"
						}
						if u.bDelete.Clicked(gtx) {
							if s.delete == m.Slug {
								u.removeMod(m.Slug, m.Enabled)
							} else {
								u.mu.Lock()
								u.confirmDelete = m.Slug
								u.mu.Unlock()
								u.invalidate()
							}
						}
						return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Button(u.th, &u.bDelete, text).Layout)
					}),
				)
			})
		}),
		layout.Rigid(u.label("Удаление стирает папку мода с диска. Если не уверен, просто выключи его.", 11, colDim, 2)),
	)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (u *ui) sectionFiles(gtx layout.Context, s snapshot) layout.Dimensions {
	m, ok := findMod(s.mods, s.selected)
	if !ok {
		return layout.Dimensions{}
	}
	rows := []layout.FlexChild{layout.Rigid(material.Subtitle1(u.th, "Файлы мода").Layout)}
	shown := m.Files
	if len(shown) > 14 {
		shown = shown[:14]
	}
	if len(m.Files) == 0 {
		rows = append(rows, layout.Rigid(u.label("в папке мода нет файлов", 12, colDim, 2)))
	}
	for _, f := range shown {
		text := f
		if s.info.InStock(f) {
			text += "   (заменяет файл игры)"
		}
		rows = append(rows, layout.Rigid(u.label(text, 12, colText, 1)))
	}
	if len(m.Files) > len(shown) {
		rows = append(rows, layout.Rigid(u.label(fmt.Sprintf("и ещё %d файл(ов)", len(m.Files)-len(shown)), 12, colDim, 1)))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (u *ui) sectionProblems(gtx layout.Context, s snapshot) layout.Dimensions {
	rows := []layout.FlexChild{layout.Rigid(material.Subtitle1(u.th, "Конфликты и замечания").Layout)}
	if len(s.found) == 0 {
		rows = append(rows, layout.Rigid(u.label("Пока ничего не мешает: одинаковых файлов у модов не нашлось.", 12, colOkText, 3)))
	}
	for _, e := range s.found {
		e := e
		tag := "заметка"
		col := colDim
		if e.Сейчас() {
			tag = "мешают сейчас"
			col = colBadText
		}
		text := fmt.Sprintf("%s (%s): %s · %s", e.Key, e.Kind, strings.Join(e.Mods, ", "), tag)
		if e.InStock {
			text += " · заменяет файл игры"
		}
		rows = append(rows, layout.Rigid(u.label(text, 12, col, 3)))
	}
	if m, ok := findMod(s.mods, s.selected); ok {
		for _, w := range m.Warnings() {
			rows = append(rows, layout.Rigid(u.label("Замечание к «"+m.Name+"»: "+w, 12, colWarnText, 3)))
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (u *ui) sectionProfiles(gtx layout.Context, s snapshot) layout.Dimensions {
	rows := []layout.FlexChild{
		layout.Rigid(material.Subtitle1(u.th, "Профили сборок").Layout),
		layout.Rigid(u.label("Профиль - это список включённых модов. Сохранил один раз, потом переключаешься одним кликом.", 12, colDim, 3)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return surface(gtx, colBg, gtx.Dp(8), func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(5), Bottom: unit.Dp(5)}.Layout(gtx,
								material.Editor(u.th, &u.profEditor, "название, например Для САМП").Layout)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if u.bSaveProf.Clicked(gtx) {
							u.saveProfile(strings.TrimSpace(u.profEditor.Text()))
						}
						return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Button(u.th, &u.bSaveProf, "Сохранить").Layout)
					}),
				)
			})
		}),
	}
	if len(s.profs) == 0 {
		rows = append(rows, layout.Rigid(u.label("Сохранённых профилей пока нет.", 12, colDim, 2)))
	}
	for _, p := range s.profs {
		p := p
		btn := u.applyBtns[p.Name]
		if btn == nil {
			btn = &widget.Clickable{}
			u.applyBtns[p.Name] = btn
		}
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(u.label(p.Name, 13, colText, 1)),
							layout.Rigid(u.label(fmt.Sprintf("модов: %d · сохранён: %s", len(p.Mods), p.Saved.Format("02.01.2006 15:04")), 11, colDim, 1)),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if btn.Clicked(gtx) {
							u.applyProfile(p)
						}
						return material.Button(u.th, btn, "Включить этот").Layout(gtx)
					}),
				)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (u *ui) journalBar(gtx layout.Context, s snapshot) layout.Dimensions {
	lines := s.journal
	if len(lines) > 60 {
		lines = lines[len(lines)-60:]
	}
	return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return surface(gtx, colCard, gtx.Dp(10), func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(u.label("Журнал", 12, colDim, 1)),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.E.Layout(gtx, u.label(
										"то же самое пишем в modmanager/journal.log", 11, colDim, 1))
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// высоту задаём жёстко: у списка без границ высота «бесконечная»
							height := gtx.Dp(72)
							gtx.Constraints.Min.Y = height
							gtx.Constraints.Max.Y = height
							return u.logList.Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
								return u.label(lines[i], 12, colDim, 1)(gtx)
							})
						}),
					)
				})
		})
	})
}

// ---------- мелочи ----------

func (u *ui) label(txt string, size unit.Sp, col color.NRGBA, maxLines int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		l := material.Label(u.th, size, txt)
		l.Color = col
		l.MaxLines = maxLines
		return l.Layout(gtx)
	}
}

// surface кладёт под содержимое плашку с фоном.
func surface(gtx layout.Context, bg color.NRGBA, radius int, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()
	if bg.A > 0 {
		paint.FillShape(gtx.Ops, bg, clip.RRect{
			Rect: image.Rectangle{Max: dims.Size},
			NE:   radius, NW: radius, SE: radius, SW: radius,
		}.Op(gtx.Ops))
	}
	call.Add(gtx.Ops)
	return dims
}

func yesNo(v bool) string {
	if v {
		return "да"
	}
	return "нет"
}
