//go:build !windows

package gui

// На других системах системных диалогов нет: папку игры и архив пишем руками.
func pickZipDialog() string                { return "" }
func pickFolderDialog(title string) string { return "" }
