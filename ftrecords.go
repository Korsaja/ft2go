package main

import (
	"fmt"
	"net"
)

type ftrecord struct {
	exAddr  net.IP
	srcAddr net.IP
	dstAddr net.IP
	bytes   uint32
}

func int2ip(ip uint32) net.IP {
	result := make(net.IP, 4)
	result[0] = byte(ip >> 24)
	result[1] = byte(ip >> 16)
	result[2] = byte(ip >> 8)
	result[3] = byte(ip)
	return result
}
func (ft *ftrecord) GetBytes() uint32 { return ft.bytes }
func (ft *ftrecord) String() string {
	return fmt.Sprintf("ex:%s src:%s dst:%s bytes:%d",
		ft.exAddr.String(),
		ft.srcAddr.String(),
		ft.dstAddr.String(),
		ft.GetBytes())
}
