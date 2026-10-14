package observe

// Reserve up to half the result slots for WSL, rather than letting a busy
// Windows process tree hide every Linux process. Unused slots fill from Windows.
func mergeObserved(win, wsl []Process, limit int) []Process {
	if limit <= 0 {
		return []Process{}
	}
	wslReserved := len(wsl)
	if wslReserved > limit/2 {
		wslReserved = limit / 2
	}
	winCount := len(win)
	if winCount > limit-wslReserved {
		winCount = limit - wslReserved
	}
	wslCount := len(wsl)
	if wslCount > limit-winCount {
		wslCount = limit - winCount
	}
	result := make([]Process, 0, winCount+wslCount)
	result = append(result, win[:winCount]...)
	result = append(result, wsl[:wslCount]...)
	return result
}
