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
			quit:    "退出界面（会话继续运行）",
			stop:    "停止后台服务并退出（仅无运行中会话时）",
		}
	}
	return trayLanguageLabels{
		tooltip: "DiggerTerm · terminal workspace",
		show:    "Show DiggerTerm",
		quit:    "Quit UI (sessions keep running)",
		stop:    "Stop daemon and quit (only when idle)",
	}
}
