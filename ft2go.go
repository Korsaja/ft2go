package main


import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)



func WalkPath(root string)(paths []string,err error){
	err = filepath.Walk(root, func(path string,
		info os.FileInfo, err error) error {
		if err != nil{
			return err
		}
		if !info.Mode().IsRegular(){
			return nil
		}
		paths = append(paths,path)
		return nil
	})
	if err != nil{
		return nil,err
	}
	return
}
func Filter(records []*ftrecord,fn func(ft *ftrecord)){
	for _, record := range records{
		fn(record)
	}
}

func main() {
	filter := flag.String("exaddr","","exAddr")
	root := flag.String("root","","roots")
	flag.Parse()
	exAddr := net.ParseIP(*filter)
	ftFiles,err  := WalkPath(*root)
	if err != nil{
		log.Fatal(err)
	}
	generator := Generator{}
	generator.Go(ftFiles)
	ch := generator.GetRecordsChannel()
	e := make([]*ftrecord,0)
	for c := range ch {
		Filter(c, func(ft *ftrecord) {
			if ft.GetExAddr().Equal(exAddr) {
				e = append(e, ft)
			}
		})
	}
	fmt.Printf("Entry with filter %s = %d",exAddr.String(),len(e))
}
