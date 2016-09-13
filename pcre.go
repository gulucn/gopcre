package pcre

/*
#cgo linux LDFLAGS: -lpcre
#cgo windows CFLAGS: -I C:/pcre/inc
#cgo windows,386 LDFLAGS: -L C:/pcre/lib -lpcre3
#cgo windows,amd64 LDFLAGS: -L C:/pcre/lib/x64 -lpcre3
#include <pcre.h>
#include <string.h>
#include <stdlib.h>
typedef void  (*my_pcre_free)(void *);
void bridge_free_func(my_pcre_free f,void *p){
	f(p);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"
)

type Handle struct {
	cptr      *C.pcre
	extraptr  *C.pcre_extra
	groups    int
	enablejit bool
}

type CompileError struct {
	Pattern string
	Message string
	Offset  int
}

func (e *CompileError) String() string {
	return e.Pattern + " (" + strconv.Itoa(e.Offset) + "): " + e.Message
}

func Compile(expr string, flags int) (*Handle, *CompileError) {
	pattern := C.CString(expr)
	defer C.free(unsafe.Pointer(pattern))
	if clen := int(C.strlen(pattern)); clen != len(expr) {
		return nil, &CompileError{
			Pattern: expr,
			Message: "NUL byte in pattern",
			Offset:  clen,
		}
	}
	var errptr *C.char
	var erroffset C.int
	ptr := C.pcre_compile(pattern, C.int(flags), &errptr, &erroffset, nil)
	if ptr == nil {
		return nil, &CompileError{
			Pattern: expr,
			Message: C.GoString(errptr),
			Offset:  int(erroffset),
		}
	}
	h := &Handle{ptr, nil, 0, false}
	var count C.int = 0
	C.pcre_fullinfo(ptr, nil, C.PCRE_INFO_CAPTURECOUNT, unsafe.Pointer(&count))
	h.groups = int(count)
	extra_ptr := C.pcre_study(ptr, C.PCRE_STUDY_JIT_COMPILE, &errptr)
	if extra_ptr == nil {
		fmt.Printf("Could not study:%s,err:%s\n", expr, C.GoString(errptr))
	} else {
		count = 0
		C.pcre_fullinfo(h.cptr, extra_ptr, C.PCRE_INFO_JIT, unsafe.Pointer(&count))
		if count == 0 {
			fmt.Printf("jit not supported\n")
			C.pcre_free_study(extra_ptr)
		} else {
			h.extraptr = extra_ptr
			h.enablejit = true
		}
	}
	return h, nil
}

// Compile the pattern.  If compilation fails, panic.
func MustCompile(expr string, flags int) *Handle {
	re, err := Compile(expr, flags)
	if err != nil {
		panic(err)
	}
	return re
}

func (h *Handle) Close() {
	if h.cptr != nil {
		C.bridge_free_func(C.pcre_free, unsafe.Pointer(h.cptr))
		h.cptr = nil
	}
	if h.extraptr != nil {
		C.pcre_free_study(h.extraptr)
		h.extraptr = nil
	}

}

type Matcher struct {
	subjectb []byte
	subject  string
	groups   int
	ovector  []C.int
}

func (p *Handle) match(subject *C.char, length int, flags int) (*Matcher, error) {
	size := 3 * (p.groups + 1)
	ovector := make([]C.int, size)
	var rc C.int = 0
	rc = C.pcre_exec(p.cptr, p.extraptr, subject, C.int(length),
		0, C.int(flags), &ovector[0], C.int(size))
	match := false
	switch {
	case rc >= 0:
		match = true
	case rc == C.PCRE_ERROR_NOMATCH:
		match = false
	case rc == C.PCRE_ERROR_BADOPTION:
		panic("PCRE.Match: invalid option flag")
		return nil, errors.New("PCRE.Match: invalid option flag")
	default:
		return nil, errors.New(fmt.Sprintf("unexepected return code from pcre_exec: %v", int(rc)))
	}
	if match {
		size = int(rc)
		return &Matcher{nil, "", size, ovector[0 : rc*2]}, nil
	} else {
		return nil, nil
	}
}
func (p *Handle) Match(subject []byte, flags int) (*Matcher, error) {
	c_psub := (*C.char)(unsafe.Pointer(&subject[0]))
	match, err := p.match(c_psub, len(subject), flags)
	if match != nil {
		match.subjectb = subject
	}
	return match, err
}

func (p *Handle) MatchString(subject string, flags int) (*Matcher, error) {
	c_psub := *(**C.char)(unsafe.Pointer(&subject))
	match, err := p.match(c_psub, len(subject), flags)
	if match != nil {
		match.subject = subject
	}
	return match, err
}

func (c *Matcher) GroupsSize() int {
	if c.groups > 0 {
		return c.groups - 1
	}
	return 0
}

func (c *Matcher) Groups() [][]byte {
	size := c.GroupsSize()
	ret := make([][]byte, size)
	for i := 1; i <= size; i++ {
		begin := c.ovector[2*i]
		end := c.ovector[2*i+1]
		if c.subjectb != nil {
			ret[i-1] = c.subjectb[begin:end]
		} else {
			ret[i-1] = []byte(c.subject[begin:end])
		}
	}
	return ret
}

func (c *Matcher) GroupsString() []string {
	size := c.GroupsSize()
	ret := make([]string, size)
	for i := 1; i <= size; i++ {
		begin := c.ovector[2*i]
		end := c.ovector[2*i+1]
		if c.subjectb != nil {
			ret[i-1] = string(c.subjectb[begin:end])
		} else {
			ret[i-1] = c.subject[begin:end]
		}
	}
	return ret
}
