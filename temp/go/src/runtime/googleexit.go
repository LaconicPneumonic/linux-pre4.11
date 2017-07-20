// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/atomic"
	"unsafe"
)

// This file contains the implementation of a Google-specific patch that allows
// one function to be registered to be called immediately before exit, and
// another to be called with stderr output when the program is crashing.

var (
	googleExitFunction  func()
	googleCrashFunction func([]byte)
)

// GoogleRegisterExitFunction is a Google-specific function that registers
// a function to be called when the program exits. It can be called only once
// and is intended to be used by //base/go:log to set up log flush on exit.
func GoogleRegisterExitFunction(fn func()) {
	if googleExitFunction != nil {
		panic("GoogleRegisterExitFunction called more than once")
	}
	googleExitFunction = fn
}

// callGoogleExitFunction calls the exit function. It's done as a plain func
// in Go code so the C code in googlesupport.c can invoke it directly
// rather than worry about the complexities of a possible closure.
func callGoogleExitFunction(chatty bool) {
	if googleExitFunction != nil {
		// Warn that we're about to run the function, in case we hang.
		// Print a stack trace of the current goroutine for context.
		// After the flush the full panic crash will happen.
		// If !chatty, we're being called from the implicit
		// exit(0) at the end of main, or from an explicit syscall.Exit
		// (aka os.Exit), in which case we can't afford to print.
		if chatty {
			print("runtime: running GoogleExitFunction\n\n")
			gp := getg()
			goroutineheader(gp)
			pc := getcallerpc(unsafe.Pointer(&chatty))
			sp := getcallersp(unsafe.Pointer(&chatty))
			systemstack(func() { traceback(pc, sp, 0, gp) })
			print("\n")
		}
		googleExitFunction()
	}
}

var google_flushed uint32

func google_flush_once(chatty bool) {
	gp := getg()
	// Only run once, only on non-g0 goroutine, only when not holding locks.
	if gp != gp.m.curg || gp == gp.m.g0 {
		if googleExitFunction != nil {
			print("runtime: not running GoogleExitFunction - on system goroutine\n")
		}
		return
	}
	if gp.m.locks > 0 {
		if googleExitFunction != nil {
			print("runtime: not running GoogleExitFunction - holding locks\n")
		}
		return
	}
	if atomic.Xadd(&google_flushed, 1) != 1 {
		if googleExitFunction != nil {
			print("runtime: not running GoogleExitFunction - already ran once\n")
		}
		return
	}

	callGoogleExitFunction(chatty)
}

func exit(exitcode int32) {
	if getg().m.dying == 1 {
		callGoogleCrashFunction(nil)
	}

	sysexit(exitcode)
}

// We also deleted the "//sys Exit" line from ../syscall/syscall_{darwin,linux}.go
// and "func Exit" from ../syscall/syscall_windows.go, so the os.Exit call will
// come here.

//go:linkname googleExit syscall.Exit
func googleExit(exitcode int) {
	callGoogleExitFunction(false)
	sysexit(int32(exitcode))
}

// GoogleRegisterCrashFunction registers a function to call during a program
// crash. All crash output is both printed to standard error and passed to the
// registered function. The end of crash output is marked by calling fn(nil).
//
// fn runs with the world stopped and is extremely limited in what it can do.
// In particular it cannot allocate memory, use arbitrary amounts of stack. fn
// must not retain the buffer it is given. Syscalls should be made via
// syscall.RawSyscall.
//
// GoogleRegisterCrashFunction can be called only once and is intended to be
// used by //production/crash_analysis/reporting/go:handler to set up crash reporting.
func GoogleRegisterCrashFunction(fn func([]byte)) {
	if googleCrashFunction != nil {
		panic("GoogleRegisterCrashFunction called more than once")
	}
	googleCrashFunction = fn
}

// callGoogleCrashFunction calls the crash function with the given buffer.
func callGoogleCrashFunction(b []byte) {
	if googleCrashFunction != nil {
		traceback_cache |= 2 << 2                      // add pc= to traceback
		bb := *(*[]byte)(noescape(unsafe.Pointer(&b))) // bb := non-escaping b
		googleCrashFunction(bb)
	}
}
