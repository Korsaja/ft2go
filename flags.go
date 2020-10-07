package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
)

type ExAddr struct {
	Addr []net.IP
}

type NFilters struct {
	IPNet []*net.IPNet
}


func (e *ExAddr) String() string {
	res := ""
	for _, ip := range e.Addr {
		res += ip.String() + " "
	}
	return res
}

func (nf *NFilters) String() string {
	res := ""
	for _, ip := range nf.IPNet {
		res += ip.String() + " "
	}
	return res
}


func (e *ExAddr) Set(s string) error {
	str := strings.Split(s,",")
	if len(str) == 0{
		return fmt.Errorf("incorrect parameter input see usage")
	}
	for _, curr := range str{
		ip := net.ParseIP(curr)
		if ip == nil{
			return fmt.Errorf("incorrect ip address entered %s",curr)
		}
		e.Addr = append(e.Addr,ip)
	}
	return nil
}

func (nf *NFilters) Set(s string) error {
	str := strings.Split(s,",")
	if len(str) == 0{
		return fmt.Errorf("incorrect parameter input see usage")
	}
	for _, curr := range str{
		_, ipNet,err  := net.ParseCIDR(curr)
		if err != nil{
			return fmt.Errorf("incorrect ipNet address entered %s",curr)
		}
		nf.IPNet = append(nf.IPNet,ipNet)
	}
	return nil
}

func ExAddrFlag(name string,value []net.IP,usage string)*ExAddr{
	e := ExAddr{value}
	flag.CommandLine.Var(&e,name,usage)
	return &e
}

func IPNetFlag(name string,value []*net.IPNet,usage string)*NFilters{
	nf := NFilters{value}
	flag.CommandLine.Var(&nf,name,usage)
	return &nf
}