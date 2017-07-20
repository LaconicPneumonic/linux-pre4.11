// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Exported API for background profiling.
// This allows a background goroutine to collect both CPU and allocation profiles
// in the background and send them off to a central collection service,
// without affecting the usual on-demand profiling.

// This is an experimental feature.
// The exported API is in a separate file so it can be easily disabled by default.
// When we commit to the API, these functions should go at the top of prof.go.

package runtime

// SetProfTag sets the profiling tag associated with the current goroutine.
// Until set, the goroutine profiling tag is 0.
// TODO(matloob): Unexport this.
func SetProfTag(tag uint64) {
	setProfTag(tag)
}

// EnableProfCPU enables background CPU profiling at the given sample rate.
// Each CPU is given a buffer capable of holding n words of profiling data.
// The profiling buffers can be read using ReadProfCPU.
// Only the first call to EnableProfCPU allocates buffers.
// Future calls ignore n.
//
// Calling EnableProfCPU with hz = 0 disables background profiling
// but does not empty or free the buffers.
func EnableProfCPU(hz int, n int) {
	enableProfCPU(hz, n)
}

// ReadProfCPU reads an integral number of background CPU profiling
// events into dst. It returns the number of words written.
// If there is no data to read, ReadProfCPU returns 0. It does not block.
//
// The event format is:
//
//	n - number of words in the event, including this one
//	tag - profiling tag (set using SetProfTag)
//	time - nanoseconds since 1970
//	stack - n-3 words giving execution stack, from leaf up
//
// If the buffer fills and events must be discarded, an overflow
// event is written to the buffer. The overflow event has n=4,
// tag=-1, and stack[0] = the number of events discarded.
func ReadProfCPU(dst []uint64) int {
	return readProfCPU(dst)
}

// EnableProfMem enables recording of background memory
// profiling, which runs at MemProfileRate.
// It allocates a buffer of n words to hold data.
// The buffer can be read using ReadProfMem.
// Only the first call allocates buffers; future calls are no-ops.
func EnableProfMem(n int) {
	enableProfMem(n)
}

// ReadProfMem reads an integral number of background memory profiling
// events into dst. It returns the number of words written.
// If there is no data to read, ReadProfMem returns 0. It does not block.
//
// The event format is:
//
//	n - number of words in the event, including this one
//	tag - profiling tag (set using SetProfTag)
//	time - nanoseconds since 1970
//	ptr - block pointer
//	size - block size
//	stack - n-5 words giving execution stack, from leaf up
//
// An allocation event has size > 0. A free event has size == -1.
//
// If the buffer fills and events must be discarded, an overflow
// event is written to the buffer. The overflow event has n=6,
// tag=-1, ptr=0, size=0, and stack[0] = the number of events discarded.
func ReadProfMem(dst []uint64) int {
	return readProfMem(dst)
}
