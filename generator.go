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
         unsigned long bytes;
}ft2go;
struct ftio ftio;
void ft2goarr(struct ftio *ftio,ft2go **in,char *filename){
    struct ftprof ftp;
    struct fts3rec_offsets fo;
    struct ftver ftv;
    char *rec;
    ftprof_start(&ftp);
    ftio_get_ver (ftio, &ftv);
    fts3rec_compute_offsets (&fo, &ftv);
    int i = 0;
    while((rec = ftio_read(ftio))){
        //strCnt++;
        in[i]->exAddr  = *((u_int32 *) (rec + fo.exaddr));
        in[i]->srcAddr = *((u_int32 *) (rec + fo.srcaddr));
        in[i]->dstAddr = *((u_int32 *) (rec + fo.dstaddr));
  		in[i]->bytes = *((u_int32 *) (rec + fo.dOctets));
		i++;
       // ft2go_rec->srcPort = *((u_int16 *) (rec + fo.srcport));
       // ft2go_rec->dstPort = *((u_int16 *) (rec + fo.dstport));
    }

    //ftio_close (&ftio);
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
	"sync"
	"unsafe"
)

const StructSize = 1 << 28


type Generator struct {
	jobsChannel chan []*ftrecord
	wg sync.WaitGroup
	threads int
}

func NewGenerator(threads int) *Generator {
	return &Generator{
		jobsChannel: make(chan []*ftrecord,threads),
		threads: threads,
		wg: sync.WaitGroup{},
	}
}

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
func (ft *ftrecord) String() string {
	return fmt.Sprintf("ex:%s src:%s dst:%s bytes:%d",
		ft.exAddr.String(),
		ft.srcAddr.String(),
		ft.dstAddr.String(),
		ft.GetBytes())
}

func (ft *ftrecord) GetBytes() uint32 { return ft.bytes }

func (g *Generator) Go(done chan<- struct{}, paths []string){
	cgoLimiter := make(chan struct{},g.threads)
	for _, path := range paths {
		cgoLimiter <- struct{}{}
		g.wg.Add(1)
		go func(filename string) {
			g.jobsChannel <- init_entrys(filename)
			<-cgoLimiter
			done <- struct{}{}
			g.wg.Done()
		}(path)
	}
}
func(g *Generator)Off(){
	g.wg.Wait()
	close(g.jobsChannel)
}
func init_entrys(path string) []*ftrecord {

	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	ftio := (*C.struct_ftio)(C.calloc(1, C.sizeof_ftio))
	defer C.free(unsafe.Pointer(ftio))

	C.ftio_init(ftio, C.int(f.Fd()), C.FT_IO_FLAG_READ)
	flowsCount := int(ftio.fth.flows_count)
	cgoRecords := (*[StructSize]*C.ft2go)(C.calloc(C.ulong(flowsCount), C.sizeof_ft2go))
	for i := 0; i < flowsCount; i++ {
		cgoRecords[i] = (*C.struct_ft2go)(C.calloc(1, C.sizeof_ft2go))
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	C.ft2goarr(ftio, &cgoRecords[0], cPath)

	ft := make([]*ftrecord, flowsCount)
	for i := 0; i < flowsCount; i++ {
		rec := &ftrecord{
			exAddr:  int2ip(uint32(cgoRecords[i].exAddr)),
			srcAddr: int2ip(uint32(cgoRecords[i].srcAddr)),
			dstAddr: int2ip(uint32(cgoRecords[i].dstAddr)),
			bytes:   uint32(cgoRecords[i].bytes)}
		ft[i] = rec
		C.free(unsafe.Pointer(cgoRecords[i]))
	}

	C.free(unsafe.Pointer(cgoRecords))
	return ft
}

type arena []unsafe.Pointer

func (a *arena) calloc(count, size int) unsafe.Pointer {
	ptr := C.calloc(C.size_t(count), C.size_t(size))
	*a = append(*a, ptr)
	return ptr
}

func (a *arena) free() {
	for _, ptr := range *a {
		C.free(ptr)
	}
}

//11956118301 octets flowCount 509716
//11956118301  octets flowCount 509716
