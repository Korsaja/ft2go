package main

import (
	"fmt"
	"net"
)


type ftrecord struct {
	exAddr  uint32
	srcAddr uint32
	dstAddr uint32
	bytes   uint32
}

func (ft *ftrecord) ExAddr() net.IP  { return uint2ip(ft.exAddr) }
func (ft *ftrecord) SrcAddr() net.IP { return uint2ip(ft.srcAddr) }
func (ft *ftrecord) DstAddr() net.IP { return uint2ip(ft.dstAddr) }

func ip2uint(ip net.IP) (uint32, error) {
	ip = ip.To4()
	if ip == nil {
		return 0, ErrInvalidIP
	}
	return uint32(ip[3]) | uint32(ip[2])<<8 | uint32(ip[1])<<16 | uint32(ip[0])<<24, nil
}

func uint2ip(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16&0xFF),
		byte(ip>>8)&0xFF, byte(ip&0xFF))
}
func (ft *ftrecord) GetBytes() uint32 { return ft.bytes }
func (ft *ftrecord) String() string {
	return fmt.Sprintf("ex:%s src:%s dst:%s bytes:%d",
		uint2ip(ft.exAddr).String(),
		uint2ip(ft.srcAddr).String(),
		uint2ip(ft.dstAddr).String(),
		ft.GetBytes())
}
