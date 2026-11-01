//go:build !windows

package desktop

type guiLaunchBroker struct{ launches chan []string }

func acquireGUIInstance(_ []string) (*guiLaunchBroker, bool, error) { return nil, true, nil }
func (b *guiLaunchBroker) close()                                   {}
