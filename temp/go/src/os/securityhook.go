package os

// This file contains the implementation of a Google-specific patch that allows
// one function to be registered to be called immediately before a file
// operation is executed. It is intended to be used by //base/go:securityhook.

type fileCheckHookFn func(perm int, path string, args []string) error

var fileCheckHook fileCheckHookFn

const (
	ReadPermission   = 1 << iota // Reading a file.
	WritePermission              // Writing to a file.
	DeletePermission             // Deleting a file or directory.
	ExecPermission               // Executing a file. Used with os.StartProcess.
)

// Name of the operation, used by os.LinkError.
const (
	symlinkOp = "symlink"
	linkOp    = "link"
	renameOp  = "rename"
)

// SetFileHook registers a function to call before a file operation. It can be
// called only once and is intended to be used by //base/go:securityhook to set
// up file operation monitoring for the Go Security Manager (go/gosm).
// Normal clients of the os package should *not* call it.
func SetFileHook(hook fileCheckHookFn) {
	if fileCheckHook != nil {
		panic("SetFileHook called more than once")
	}
	fileCheckHook = hook
}

// checkStartProcess calls the registered hook function before os.StartProcess.
func checkStartProcess(name string, argv []string, attr *ProcAttr) error {
	if fileCheckHook != nil {
		if err := fileCheckHook(ExecPermission, name, argv); err != nil {
			return &PathError{"fork/exec", name, err}
		}
	}
	return nil
}

// checkOpenFile calls the registered hook function before os.OpenFile.
func checkOpenFile(name string, flag int, perm FileMode) error {
	if fileCheckHook != nil {
		perm := ReadPermission
		if flag&O_RDWR == O_RDWR {
			perm |= WritePermission
		} else if flag&O_WRONLY == O_WRONLY {
			perm = WritePermission
		}
		if err := fileCheckHook(perm, name, nil); err != nil {
			return &PathError{"open", name, err}
		}
	}
	return nil
}

// checkRemove calls the registered hook function before os.Remove.
func checkRemove(name string) error {
	if fileCheckHook != nil {
		if err := fileCheckHook(DeletePermission, name, nil); err != nil {
			return &PathError{"remove", name, err}
		}
	}
	return nil
}

// checkLinkOp calls the registered hook function before os.Rename, os.Symlink
// and os.Link.
func checkLinkOp(op, oldname, newname string) error {
	if fileCheckHook != nil {
		if op != symlinkOp {
			if err := fileCheckHook(ReadPermission, oldname, nil); err != nil {
				return &LinkError{op, oldname, newname, err}
			}
		}
		if err := fileCheckHook(WritePermission, newname, nil); err != nil {
			return &LinkError{op, oldname, newname, err}
		}
	}
	return nil
}
