package net

// This file contains the implementation of a Google-specific patch that allows
// one function to be registered to be called immediately before a network
// connection is dialed. It is intended to be used by //base/go:securityhook.

type dialHookFn func(network, address string) error

var dialHook dialHookFn

// SetDialHook registers a function to call before a network connection is
// established. It can be called only once and is intended to be used
// by //base/go:securityhook to set up network connection monitoring for
// the Go Security Manager (go/gosm).
// Normal clients of the net package should *not* call it.
func SetDialHook(hook dialHookFn) {
	if dialHook != nil {
		panic("SetDialHook called more than once")
	}
	dialHook = hook
}

// checkDial calls the registered hook function before net.Dial.
func checkDial(network, address string) error {
	if dialHook != nil {
		if err := dialHook(network, address); err != nil {
			return &OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: err}
		}
	}
	return nil
}
