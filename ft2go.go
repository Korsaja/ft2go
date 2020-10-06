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
	"time"
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
		generator.Off()
	}()

	var sum uint64
	networks := []string{"10.255.0.0/16", "10.254.0.0/16", "10.223.0.0/16", "10.222.0.0/16"}
	cidrs := make([]*net.IPNet, len(networks))
	for i, n := range networks {
		_, ipNet, _ := net.ParseCIDR(n)
		cidrs[i] = ipNet
	}
	go generator.Go(done, ftFiles)

	for entry := range generator.jobsChannel {
		Filter(entry, func(ft *ftrecord) {
			if ft.exAddr.Equal(exAddr) {
				for _, ipNet := range cidrs {
					if ipNet.Contains(ft.srcAddr) || ipNet.Contains(ft.dstAddr) {
						sum += uint64(ft.GetBytes())
					}
				}
			}
		})
	}
	time.Sleep(30*time.Second)
	fmt.Printf("Entry with filter %s SumBytes = %d\n", exAddr.String(), sum)
}
//27157610651
//27157602740
