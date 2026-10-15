package main

// trayLanguageLabels are native menu strings, separate from daemon/session
// state. The GUI tells the host its effective UI locale after WebView loads.
type trayLanguageLabels struct {
	tooltip string
	show    string
	quit    string
	stop    string
}

func trayLabelsFor(locale string) trayLanguageLabels {
	if locale == "zh" {
		return trayLanguageLabels{
			tooltip: "DiggerTerm · 终端工作区",
			show:    "显示 DiggerTerm",
			quit:    "仅退出 DiggerTerm",
			stop:    "停止所有会话并退出",
		}
	}
	return trayLanguageLabels{
		tooltip: "DiggerTerm · terminal workspace",
		show:    "Show DiggerTerm",
		quit:    "Quit DiggerTerm",
		stop:    "Stop all sessions and quit",
	}
}
