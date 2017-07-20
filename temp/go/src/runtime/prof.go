// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/atomic"
	"unsafe"
)

// A profBuf is a lock-free buffer for profiling events,
// safe for concurrent use by one reader and one writer.
// If the writer gets ahead of the reader, writes are discarded
// and replaced by an overflow entry.
// If the reader catches up to the writer, reads return 0.
type profBuf struct {
	hdrsize      uintptr
	r, w         uintptr
	overflow     uintptr
	overflowTime int64
	data         []uint64
}

// newProfBuf returns a new profiling buffer with room for
// a header of hdrsize words and a buffer of at least bufwords words.
func newProfBuf(hdrsize, bufwords int) *profBuf {
	if min := 3 + hdrsize + 1; bufwords < min {
		bufwords = min
	}

	size := uintptr(bufwords)*8 + unsafe.Sizeof(profBuf{})
	size = round(size, physPageSize)
	bufwords = int((size - unsafe.Sizeof(profBuf{})) / 8)

	b := (*profBuf)(sysAlloc(size, &memstats.other_sys))
	b.hdrsize = uintptr(hdrsize)
	b.r = 0
	b.w = 0
	b.overflow = 0
	b.overflowTime = 0
	s := (*slice)(unsafe.Pointer(&b.data))
	s.array = add(unsafe.Pointer(b), unsafe.Sizeof(*b))
	s.len = bufwords
	s.cap = bufwords
	return b
}

// write writes an entry to the profiling buffer b.
// The entry begins with a fixed hdr, which must have
// length b.hdrsize, followed by a variable-sized stack.
func (b *profBuf) write(tag uint64, now int64, hdr []uint64, stk []uintptr) {
	if b == nil {
		return
	}
	if len(hdr) > int(b.hdrsize) {
		throw("misuse of profBuf.write")
	}

	available := atomic.Loaduintptr(&b.r) + uintptr(len(b.data)) - b.w
	n := 3 + b.hdrsize + uintptr(len(stk))

	if b.overflow > 0 && n+(3+b.hdrsize+1) <= available {
		// Room for both an overflow record and the one being written.
		// Write the overflow record.
		var count [1]uintptr
		count[0] = b.overflow
		b.overflow = 0
		b.write(^uint64(0), b.overflowTime, nil, count[:])
	}

	if b.overflow > 0 || n > available {
		// Extant overflow or no room for new record (likely both).
		if b.overflow == 0 {
			b.overflowTime = now
		}
		b.overflow++
		return
	}

	data := b.data
	w := int(b.w) % len(data)

	// length
	data[w] = uint64(n)
	if w++; w == len(data) {
		w = 0
	}

	// profiling tag
	data[w] = tag
	if w++; w == len(data) {
		w = 0
	}

	// time stamp
	data[w] = uint64(now)
	if w++; w == len(data) {
		w = 0
	}

	// header
	for i := uintptr(0); i < b.hdrsize; i++ {
		x := uint64(0)
		if i < uintptr(len(hdr)) {
			x = hdr[i]
		}
		data[w] = x
		if w++; w == len(data) {
			w = 0
		}
	}

	// stack frame
	for _, pc := range stk {
		data[w] = uint64(pc)
		if w++; w == len(data) {
			w = 0
		}
	}

	atomic.Xadduintptr(&b.w, n) // commit write
}

func (b *profBuf) read(dst []uint64) int {
	if b == nil {
		return 0
	}

	data := b.data
	n := atomic.Loaduintptr(&b.w) - b.r
	if n > uintptr(len(dst)) {
		n = uintptr(len(dst))
	}
	r := int(b.r) % len(data)
	i := uintptr(0)
	for i < n {
		size := uintptr(data[r])
		if size > n-i {
			break
		}
		for j := uintptr(0); j < size; j++ {
			dst[i+j] = data[r]
			if r++; r == len(data) {
				r = 0
			}
		}
		i += size
	}

	atomic.Xadduintptr(&b.r, i) // commit read
	return int(i)
}

// setProfTag sets the profiling tag of the current goroutine to tag.
// The profiling tag of a goroutine is inherited by goroutines it starts.
//
//go:linkname setProfTag google3/stats/census/go/runtime.runtime_setProfTag
func setProfTag(tag uint64) {
	getg().proftag = tag
}

// getProfTag returns the profiling tag of the current goroutine.
//
//go:linkname getProfTag google3/stats/census/go/runtime.runtime_getProfTag
func getProfTag() uint64 {
	return getg().proftag
}

var bgprof struct {
	lk  mutex
	hz  uint32
	cpu struct {
		readp uintptr
		read  mutex
		write mutex
		buf   *profBuf // for ticks without a p
	}
	mem struct {
		read  mutex
		write mutex
		buf   *profBuf
	}
}

func enableProfCPU(hz, bufsize int) {
	lock(&bgprof.lk)
	lock(&bgprof.cpu.read)
	if hz != 0 {
		for i := int32(0); i < gomaxprocs; i++ {
			p := allp[i]
			if p == nil {
				break
			}
			if p.bgcpu != nil {
				continue
			}
			p.bgcpu = newProfBuf(0, bufsize)
		}
		bgprof.cpu.buf = newProfBuf(0, bufsize)
	}
	atomic.Store(&bgprof.hz, uint32(hz))
	unlock(&bgprof.cpu.read)
	unlock(&bgprof.lk)
	enablebgcpu(int32(hz))
}

func bgcpu(gp *g, stk []uintptr) {
	if atomic.Load(&bgprof.hz) == 0 {
		return
	}

	_g_ := getg()
	pp := _g_.m.p.ptr()
	if pp != nil {
		b := pp.bgcpu
		if b == nil {
			return
		}
		b.write(gp.proftag, unixnanotime(), nil, stk)
		return
	}

	// No p. Can happen during cgo calls.
	// Write to central buffer. Will cause some serialization.
	lock(&bgprof.cpu.write)
	bgprof.cpu.buf.write(gp.proftag, unixnanotime(), nil, stk)
	unlock(&bgprof.cpu.write)
}

func readProfCPU(dst []uint64) int {
	lock(&bgprof.cpu.read)
	total := 0
	bgprof.cpu.readp++
	maxp := uintptr(gomaxprocs)
	for i := uintptr(0); i < maxp+1; i++ {
		j := (i + bgprof.cpu.readp) % (maxp + 1)
		var b *profBuf
		if j == maxp {
			b = bgprof.cpu.buf
		} else {
			p := allp[j]
			if p == nil {
				continue
			}
			b = p.bgcpu
			if b == nil {
				continue
			}
		}
		n := b.read(dst)
		total += n
		dst = dst[n:]
		if len(dst) == 0 {
			break
		}
	}
	unlock(&bgprof.cpu.read)
	return total
}

func enableProfMem(bufsize int) {
	lock(&bgprof.lk)
	lock(&bgprof.mem.read)
	if bgprof.mem.buf == nil {
		bgprof.mem.buf = newProfBuf(2, bufsize)
	}
	unlock(&bgprof.mem.read)
	unlock(&bgprof.lk)
}

func bgmalloc(ptr, size uintptr, stk []uintptr) {
	if bgprof.mem.buf == nil {
		return
	}

	var hdr [2]uint64
	hdr[0] = uint64(ptr)
	if size == ^uintptr(0) {
		hdr[1] = ^uint64(0)
	} else {
		hdr[1] = uint64(size)
	}
	lock(&bgprof.mem.write)
	if bgprof.mem.buf == nil {
		return
	}
	bgprof.mem.buf.write(getg().proftag, unixnanotime(), hdr[:], stk)
	unlock(&bgprof.mem.write)
}

func readProfMem(dst []uint64) int {
	if bgprof.mem.buf == nil {
		return 0
	}

	lock(&bgprof.mem.read)
	n := bgprof.mem.buf.read(dst)
	unlock(&bgprof.mem.read)
	return n
}
