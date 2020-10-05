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
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

const (
	Threads    = 2 - 1
	StructSize = 1 << 28
)

type Generator struct {
	ftrecords chan []*ftrecord
	wg sync.WaitGroup
}

type ftrecord struct {
	exAddr  uint32
	srcAddr uint32
	dstAddr uint32
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
func (ft *ftrecord) String()string{
	return fmt.Sprintf("ex:%s src:%s dst:%s bytes:%d",
		ft.GetExAddr().String(),
		ft.GetSrcAddr().String(),
		ft.GetDstAddr().String(),
		ft.GetBytes())
}
func (ft *ftrecord) GetExAddr() net.IP  { return int2ip(ft.exAddr) }
func (ft *ftrecord) GetSrcAddr() net.IP { return int2ip(ft.srcAddr) }
func (ft *ftrecord) GetDstAddr() net.IP { return int2ip(ft.dstAddr) }
func (ft *ftrecord) GetBytes() uint32   { return ft.bytes }

func (g *Generator) Go(paths []string) {
	g.ftrecords = make(chan []*ftrecord, Threads)
	cgoLimiter := make(chan struct{}, Threads)
	for _, path := range paths {
		cgoLimiter <- struct{}{}
		g.wg.Add(1)
		go func(filename string) {
			records := init_entrys(filename)
			g.ftrecords <- records
			<-cgoLimiter
			g.wg.Done()
		}(path)
	}
	go func() {
		g.wg.Wait()
		close(g.ftrecords)
	}()
}
func (g *Generator)Stop(){
	g.wg.Wait()
	close(g.ftrecords)
}
func (g *Generator) GetRecordsChannel() chan []*ftrecord { return g.ftrecords }
func init_entrys(path string) []*ftrecord {
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

	ftrecords := make([]*ftrecord, flows_count)
	for i := range ftrecords {
		ftrecords[i] = &ftrecord{}
	}
	cgoEntry := (*[StructSize]*C.ft2go)(C.malloc(C.size_t(C.sizeof_ft2go * flows_count)))
	for i, record := range ftrecords {
		ft2go := (*C.ft2go)(C.malloc(C.size_t(C.sizeof_ft2go)))
		(*ft2go).exAddr = C.ulong(record.exAddr)
		(*ft2go).srcAddr = C.ulong(record.srcAddr)
		(*ft2go).dstAddr = C.ulong(record.dstAddr)
		(*ft2go).bytes = C.ulong(record.bytes)
		cgoEntry[i] = ft2go
	}
	C.ft2goarr(ftio, &cgoEntry[0], C.CString(path))
	for i := 0; i < flows_count; i++ {
		cgoElement := cgoEntry[i]
		goElement := ftrecords[i]
		goElement.exAddr = uint32(cgoElement.exAddr)
		goElement.srcAddr = uint32(cgoElement.srcAddr)
		goElement.dstAddr = uint32(cgoElement.dstAddr)
		goElement.bytes = uint32(cgoElement.bytes)
		//custom clear
		C.free(unsafe.Pointer(cgoEntry[i]))
	}
	//custom clear
	C.free(unsafe.Pointer(cgoEntry))
	return ftrecords
}
