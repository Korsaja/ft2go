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
)

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
func Filter(records []*ftrecord, fn func(ft *ftrecord)) {
	for _, record := range records {
		fn(record)
	}
}

func main() {
	filter := flag.String("exaddr", "", "exAddr")
	root := flag.String("root", "", "roots")
	flag.Parse()
	exAddr := net.ParseIP(*filter)
	br1 := net.ParseIP("185.173.73.255")
	ftFiles, err := WalkPath(*root)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{}, len(ftFiles))

	uiprogress.Start()
	bar := uiprogress.AddBar(len(ftFiles))
	bar.PrependCompleted().PrependElapsed().AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Files::%d::rdy::%d", len(ftFiles), bar.Current())
	})
	generator := NewGenerator(runtime.NumCPU())
	go func() {
		for i := 0; i < len(ftFiles); i++ {
			<-done
			bar.Incr()
		}
	}()

	var sum uint64
	networks := []string{"10.253.0.0/16", "10.221.0.0/16"}
	cidrs := make([]*net.IPNet, len(networks))
	for i, n := range networks {
		_, ipNet, _ := net.ParseCIDR(n)
		cidrs[i] = ipNet
	}
	generator.Go(done, ftFiles)

	for entry := range generator.jobsChannel {
		Filter(entry, func(ft *ftrecord) {
			if ft.exAddr.Equal(exAddr) || ft.exAddr.Equal(br1) {
				for _, ipNet := range cidrs {
					if ipNet.Contains(ft.srcAddr) || ipNet.Contains(ft.dstAddr) {
						sum += uint64(ft.GetBytes())
					}
				}
			}
		})
	}

	fmt.Printf("Entry with filter %s SumBytes = %d\n", exAddr.String(), sum)
}
//216536942984
//216538516644