package main

import (
	"flag"
	"fmt"
	"github.com/gosuri/uiprogress"
	"net"
	"os"
	"path/filepath"
	"runtime"
)



func main() {
	exAddr := ExAddrFlag("gw", []net.IP{}, "gateway network address for more with sep=,")
	dirFiles := flag.String("path", "", "file directory")
	nfilters := IPNetFlag("filters",[]*net.IPNet{},"filter-primitive for more with sep=,")
	flag.Parse()

	//Get absolute path for file flow - tools
	ftFiles, err := WalkPath(*dirFiles)
	if err != nil{
		fmt.Fprintf(os.Stderr,"[!] Error :: %s \n",err.Error())
		os.Exit(1)
	}
	// For show progress bar
	counterChan := make(chan struct{},len(ftFiles))

	//Progress bar for
	//processing and converting
	// Generator
	uiprogress.Start()
	barGenerator := uiprogress.AddBar(len(ftFiles))
	barGenerator.PrependElapsed().AppendCompleted()
	barGenerator.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Processing :: [ %d / %d ] :: ",
			barGenerator.Current(),len(ftFiles))
	})
	go func() {for i:=0;i < len(ftFiles);i++{<-counterChan;barGenerator.Incr()}}()

	numCPU := runtime.NumCPU()
	generator := NewGenerator(numCPU)
	generator.Go(counterChan,ftFiles)
	go func() {
		if err := <- generator.errChannel; err != nil{
			fmt.Fprintf(os.Stderr,"[!] Error :: %s \n",err.Error())
		}
	}()

	//Progress bar for
	//filtration
	barFiltration := uiprogress.AddBar(len(ftFiles))
	barFiltration.PrependElapsed().AppendCompleted()
	barFiltration.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Filtration :: [ %d / %d ] :: ",
			barFiltration.Current(),len(ftFiles))
	})
	filter := func(rec *ftrecord) bool{
		for _, ex := range exAddr.Addr{
			if ex.Equal(rec.exAddr){
				for _, ipNet := range nfilters.IPNet{
					if ipNet.Contains(rec.srcAddr) ||
						ipNet.Contains(rec.dstAddr){
						return true
					}
				}
			}
		}
		return false
	}
	records := make([][]*ftrecord,0)
	for record := range generator.jobsChannel{
		records = append(records,SliceFilter(record,filter))
		barFiltration.Incr()
	}

	var sum uint64
	for _, rx := range records{
		for _, r := range rx{
			sum += uint64(r.GetBytes())
		}
	}



	fmt.Println("Sum = ",sum)


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