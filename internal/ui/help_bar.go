package ui

import "github.com/charmbracelet/lipgloss"

type HelpBar struct {
	Width int
}

func (h HelpBar) Render() string {
	style := lipgloss.NewStyle().
		Width(h.Width).
		Background(lipgloss.Color("#303446")).
		Foreground(lipgloss.Color("#c6d0f5")).
		PaddingLeft(2).
		PaddingRight(2)

	helpTxt := "↑/↓/j/k: Navigate Trace List | Enter: View details | Tab: Switch pane | q/Esc: Quit"
	return "\n" + style.Render(helpTxt)
}
