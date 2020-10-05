package main

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -L/usr/lib -lft -lz
#include "ftlib.h"
#include <stdlib.h>
#include <stdio.h>
#include <arpa/inet.h>
typedef struct ft2go {
         unsigned long exAddr;
         unsigned long srcAddr;
         unsigned long dstAddr;
         short int srcPort;
         short int dstPort;
         unsigned long bytes;
}ft2go;
void ft2goarr(struct ftio ftio,ft2go **in,char *filename){
    struct ftprof ftp;
    struct fts3rec_offsets fo;
    struct ftver ftv;
    char *rec;
    u_int32 last_time;
    u_int32 tm;
    ftprof_start(&ftp);
    ftio_get_ver (&ftio, &ftv);
    fts3rec_compute_offsets (&fo, &ftv);
    last_time = 0;
	int i = 0;
    for ( i = 0;i < ftio.fth.flows_count;i++){
        //strCnt++;
		rec = ftio_read(&ftio);
        tm = *((u_int32 *) (rec + fo.unix_secs));
        if (last_time != tm)
        {
            in[i]->exAddr = 0;
            in[i]->srcAddr = 0;
            in[i]->dstAddr = 0;
            //ft2go_rec->srcPort = htons ((u_int16) ((tm >> 16) & 0xFFFF));
           // ft2go_rec->dstPort = htons ((u_int16) (tm & 0xFFFF));
            in[i]->bytes = 0;
            last_time = tm;
			i++;
        }
        in[i]->exAddr  = *((u_int32 *) (rec + fo.exaddr));
        in[i]->srcAddr = *((u_int32 *) (rec + fo.srcaddr));
        in[i]->dstAddr = *((u_int32 *) (rec + fo.dstaddr));
       // ft2go_rec->srcPort = *((u_int16 *) (rec + fo.srcport));
       // ft2go_rec->dstPort = *((u_int16 *) (rec + fo.dstport));
        in[i]->bytes = *((u_int32 *) (rec + fo.dOctets));
		i++;
    }

    ftio_close (&ftio);
   // fclose(fp);
	//return strCnt;
}
*/
import "C"
import (
	"fmt"
	"github.com/gosuri/uiprogress"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"
)
type e struct {
	exAddr  uint32
	srcAddr uint32
	dstAddr uint32
	//srcPort uint16
	//dstPort uint16
	bytes   uint32
}

func Init_entrys(path string)([]*e){
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	ftio := C.struct_ftio{}
	C.ftio_init(&ftio, C.int(f.Fd()), C.FT_IO_FLAG_READ)
	flows_count := int(ftio.fth.flows_count)

	entries := make([]*e,flows_count)
	for i := range entries {
		entries[i] = &e{}}
	cgoEntry := (*[1 << 28]*C.ft2go)(C.malloc(C.size_t(C.sizeof_ft2go * flows_count)))
	for i,e := range entries {
		ft2go := (*C.ft2go)(C.malloc(C.size_t(C.sizeof_ft2go)))
		(*ft2go).exAddr = C.ulong(e.exAddr)
		(*ft2go).srcAddr = C.ulong(e.srcAddr)
		(*ft2go).dstAddr = C.ulong(e.dstAddr)
		//(*ft2go).srcPort = C.short(e.srcPort)
		//(*ft2go).dstPort = C.short(e.dstPort)
		(*ft2go).bytes = C.ulong(e.bytes)
		cgoEntry[i] = ft2go
	}
	C.ft2goarr(ftio,&cgoEntry[0],C.CString(path))
	for i := 0; i < flows_count;i++{
		cgoElement := cgoEntry[i]
		goElement  := entries[i]
		goElement.exAddr = uint32(cgoElement.exAddr)
		goElement.srcAddr = uint32(cgoElement.srcAddr)
		goElement.dstAddr = uint32(cgoElement.dstAddr)
		//goElement.srcPort = uint16(cgoElement.srcPort)
		//goElement.dstPort = uint16(cgoElement.dstPort)
		goElement.bytes = uint32(cgoElement.bytes)
		C.free(unsafe.Pointer(cgoEntry[i]))
	}

	C.free(unsafe.Pointer(cgoEntry))
	return entries
}

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



func main() {
	ftFiles,err  := WalkPath(os.Args[1])
	if err != nil{
		log.Fatal(err)
	}
	uiprogress.Start()
	bar := uiprogress.AddBar(len(ftFiles))
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Files::%d::compl::%d",len(ftFiles),bar.Current())
	})
	cgoLimiter := make(chan  struct{},2)
	wg := sync.WaitGroup{}
	for _, f := range ftFiles{
		cgoLimiter <- struct{}{}
		wg.Add(1)
		go func(filename string) {
			_ = Init_entrys(filename)
			bar.Incr()
			<-cgoLimiter
			wg.Done()
		}(f)
	}
	wg.Wait()
}
