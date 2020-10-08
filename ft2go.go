package main

import (
	"flag"
	"fmt"
	"github.com/gosuri/uiprogress"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	Name   string
	ipNets []*net.IPNet
	sum    uint64
}

func Filter(exAddr []net.IP, clients []*Client) filterFunc {
	return func(f *ftrecord) {
		for _, exAdd := range exAddr{
			if exAdd.Equal(f.ExAddr()){
				for _, c := range clients{
					for _,ipNet := range c.ipNets{
						if ipNet.Contains(f.SrcAddr()) || ipNet.Contains(f.DstAddr()){
							atomic.AddUint64(&c.sum,uint64(f.bytes))
						}
					}
				}
			}
		}
	}
}
func WalkPath(root string) (paths []string, err error) {
	err = filepath.Walk(root, func(path string,
		info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return
}

func main() {
	exAdders := flag.String("ex", "", "gateways name with sep=,")
	filters := flag.String("nf", "", "nfilter with sep=,")
	configPath := flag.String("conf", "config.toml", "path to configfile")
	dirFiles := flag.String("path", "", "file directory")
	numCPU := flag.Int("w", 1, "workers")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}


	fmt.Fprintf(os.Stdout, "Reading config...\n")
	conf, err := ReadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
		os.Exit(1)
	}

	exAddressSplit := strings.Split(*exAdders, ",")
	if len(exAddressSplit) == 0 {
		fmt.Fprintf(os.Stderr, "[!] Error :: Invalid exAddrsor not entered\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Initialization and parameter setting...\n")
	exAddersIPs := make([]net.IP, 0)
	for _, device := range conf.Devices {
		for _, attr := range device.Attr {
			for _, exAddr := range exAddressSplit {
				if exAddr == attr.Name {
					ip := net.ParseIP(attr.IP)
					if ip == nil {
						fmt.Fprintf(os.Stderr, "[!] Error :: %v :: %v \n", ErrInvalidIP, attr.IP)
					}
					exAddersIPs = append(exAddersIPs, ip)
				}
			}
		}
	}
	clients := make([]*Client, 0)
	nFilters := strings.Split(*filters, ",")
	if len(nFilters) == 0 {
		fmt.Fprintf(os.Stderr, "[!] Error :: not entered nfilter\n")
		os.Exit(1)
	}
	for _, filter := range conf.Nfilters {
		for _, attr := range filter.Attr {
			for _, nFilter := range nFilters {
				if attr.Name == nFilter {
					c := &Client{Name: attr.Name, ipNets: make([]*net.IPNet,0),}
					for _, ips := range attr.Ips {
						_, ipNet, err := net.ParseCIDR(ips)
						if err != nil {
							fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
						}
						c.ipNets = append(c.ipNets,ipNet)
					}
					clients = append(clients,c)
				}
			}
		}
	}

	//Get absolute path for file flow - tools
	ftFiles, err := WalkPath(*dirFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
		os.Exit(1)
	}
	// For show progress bar
	counterChan := make(chan struct{}, len(ftFiles))

	//Progress bar for
	//processing and converting
	uiprogress.Start()
	barGenerator := uiprogress.AddBar(len(ftFiles))
	barGenerator.PrependElapsed().AppendCompleted()
	barGenerator.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Processing :: [ %d / %d ] :: ",
			barGenerator.Current(), len(ftFiles))
	})

	if *numCPU == 1 {
		*numCPU = runtime.NumCPU()
	}
	generator := NewGenerator(*numCPU, Filter(exAddersIPs, clients))
	generator.Go(counterChan, ftFiles)
	go func() {
		for i := 0; i < len(ftFiles); i++ {
			<-counterChan
			barGenerator.Incr()
		}
	}()
	if err := <-generator.errChannel; err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
	}
	time.Sleep(10*time.Second)
	for _, c := range clients{
		fmt.Printf("Cl: %s Sum: %d",c.Name,c.sum)
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}

