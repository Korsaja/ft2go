package main

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -L/usr/lib -lft -lz

#include "ft2go.h"
*/
import "C"

import (
	"fmt"
	"net"
	"unsafe"
	"encoding/binary"
)

type entry struct{
	ex    uint32
	src   uint32
	dst   uint32
	bytes uint32
}

func (e *entry)String()string{
	return fmt.Sprintf("ex: %s src: %s dst: %s bytes: %d",
		Long2ip(e.ex),
		Long2ip(e.src),
		Long2ip(e.dst),
		e.bytes)
}

func Long2ip(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.LittleEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}



func main() {
	path := C.CString("ft-v05.2019-01-30.005000+0400")
	ftgo := new(C.struct_ft2go)
	ftgo = C.listEntry(path)
	p := new(C.struct_ft2go)
	for p = ftgo; p != nil; p = p.next{
		e := &entry{ex:uint32(p.exAddrr),src:uint32(p.srcAddrr),dst:uint32(p.dstAddrr),bytes: uint32(p.bytes)}
		fmt.Println(e)
	}
	C.free(unsafe.Pointer(ftgo))
}
