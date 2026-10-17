[Русский](README.md) | [**English**](README.en.md) | [Español](README.es.md) | [Português](README.pt.md)

![GTA San Andreas Mod Manager](assets/banner.png)

![Так выглядит окно](assets/screenshot.png)

# GTA San Andreas Mod Manager

A small Windows program that keeps a modded San Andreas install in order: one switch per mod, an early warning when two mods fight over the same file, and saved build profiles.

The classic problem: you install five mods, the game refuses to start, and the culprit is a single `infernus.dff` that two mods dropped in at once. This manager shows those clashes before you launch the game.

**Download:** [sa-mod-manager-windows.zip](https://github.com/hilsetapp/sa-mod-manager/releases/latest/download/sa-mod-manager-windows.zip)

## What it does

- **Mod list with switches.** A mod lives in `modloader` (enabled) or `modmanager/off` (disabled). Toggling just moves the folder, nothing is deleted and everything is reversible.
- **Conflict detection.** If two mods provide a file with the same name, the manager shows who is fighting over what, and separates "they break the game right now" from "keep this in mind".
- **Stock file marking.** Files that already exist inside `models/gta3.img` are marked as "replaces a stock file", so you can tell a replacement from an addition.
- **Archive warnings.** Whole-`gta3.img` replacement, an unfamiliar `d3d9.dll`, an `.exe` inside a mod archive: better to know before installing.
- **Build profiles.** Save the set of enabled mods ("SA-MP", "single player") and switch between them with one click.
- **Journal.** Everything the manager does is written to the window and to `modmanager/journal.log`.
- **Import from zip.** Pick an archive, it unpacks into the disabled folder, you enable it when you want.
- **Works with hand-made installs.** Mods you already put into `modloader` by hand show up in the list on their own.

## Quick start

1. Download `sa-mod-manager-windows.zip` and unpack it anywhere. Avoid `Program Files`, Windows restricts writing there.
2. Run `sa-mods.exe`. If SmartScreen shows a blue warning, click "More info" then "Run anyway". The program downloads nothing and has no network code.
3. Click **"Выбрать папку"** (choose folder) and point it at the folder with `gta_sa.exe`, usually `C:\Games\GTA San Andreas`.
4. Click **"Добавить мод из архива"** (add mod from archive) and pick a mod zip. It appears in the list, disabled.
5. Flip the switch and start the game.

The interface is in Russian: the tool is made for the Russian-speaking San Andreas scene first. English, Spanish and Portuguese players are welcome, and translations of the UI are on the to-do list.

Note: for the game to see the `modloader` folder you need **ModLoader** (`modloader.asi`) installed. If it is missing, the manager says so in a banner, because enabling mods without ModLoader changes nothing in game.

## Command line

`sa-mods-cli.exe` does the same without a window, handy for scripts:

```
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -check
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -list
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -conflicts
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -import C:\Downloads\silentpatch.zip
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -disable tunning
sa-mods-cli.exe -game "C:\Games\GTA San Andreas" -enable tunning
```

Flags: `-check` what was found in the game folder, `-list` mods, `-conflicts` clashes, `-import` add a zip, `-enable` / `-disable` toggle a mod by folder name, `-version`.

## How it works

- San Andreas identifies files by name. If two mods ship `infernus.dff`, the last folder wins and one mod silently stops working. That is why a conflict here means "same file name in two mods", and for data files (`.cfg`, `.dat`, `.ipl`) "same relative path".
- CLEO scripts (`.cs`), ASI plugins (`.asi`) and `.fxt` text compare by file name too, since the game looks them up by name.
- Enabling and disabling moves a folder between `modloader` and `modmanager/off`. The manager never patches the game, never writes into `.img` archives and never edits your files.
- File names inside `models/gta3.img` are read read-only, only to tell stock files from added ones.

## What it does not do

- It does not run `.exe` mod installers and does not edit other people's scripts.
- It does not compare model contents, only file names, exactly like the game does.
- It does not download mods and ships no catalogue of links.
- It is not a browser or phone app: this is a Windows desktop program.

## Build from source

Go 1.26 or newer, no C compiler needed:

```
git clone https://github.com/hilsetapp/sa-mod-manager.git
cd sa-mod-manager
go test ./...
go build -ldflags "-H=windowsgui -s -w" -o sa-mods.exe ./cmd/sa-mods
go build -s -w -o sa-mods-cli.exe ./cmd/sa-mods-cli
```

Windows builds are also produced by GitHub Actions: `build.yml` on every push, `release.yml` on a `v*` tag.

## Layout

```
cmd/sa-mods         window (entry point)
cmd/sa-mods-cli     command line
internal/gui        interface: mod list, profiles, journal
internal/game       game folder, gta3.img name reading, stock names
internal/library    mod library: zip import, enable, disable, remove
internal/conflicts  mods sharing the same file
internal/profiles   saved build profiles
internal/cli        command line
lists               hints: signs of broken mods, community favourites
scripts             banner generator, self-check
```

Manager files inside the game folder:

```
modloader/           enabled mods (this is what the game reads)
modmanager/off/      disabled mods
modmanager/meta/     mod cards: source, date, file count
modmanager/profiles/ saved profiles
modmanager/journal.log
```

## License

MIT. The Inter font is licensed under the SIL Open Font License 1.1, see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

If the manager saved your build, a star helps other modders find it.
