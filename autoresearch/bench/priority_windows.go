package main

import "syscall"

func raisePriority() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setPriority := kernel32.NewProc("SetPriorityClass")
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return
	}
	const highPriorityClass = 0x00000080
	_, _, _ = setPriority.Call(uintptr(handle), highPriorityClass)
}
