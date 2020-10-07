package main

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -L/usr/lib -lft -lz
#include "ftlib.h"
#include <stdlib.h>

typedef struct ft2go {
         unsigned long exAddr;
         unsigned long srcAddr;
         unsigned long dstAddr;
         unsigned long bytes;
}ft2go;
struct ftio ftio;
void ft2goarr(struct ftio *ftio,ft2go **in){
    struct ftprof ftp;
    struct fts3rec_offsets fo;
    struct ftver ftv;
    char *rec;
    ftprof_start(&ftp);
    ftio_get_ver (ftio, &ftv);
    fts3rec_compute_offsets (&fo, &ftv);
	int i = 0;
    while((rec = ftio_read(ftio))){
        in[i]->exAddr  = *((u_int32 *) (rec + fo.exaddr));
        in[i]->srcAddr = *((u_int32 *) (rec + fo.srcaddr));
        in[i]->dstAddr = *((u_int32 *) (rec + fo.dstaddr));
  		in[i]->bytes = *((u_int32 *) (rec + fo.dOctets));
		i++;
		// for adding ports or other fields see https://github.com/adsr/flow-tools/blob/master/lib/ftlib.h#L613
       // ft2go_rec->srcPort = *((u_int16 *) (rec + fo.srcport));
       // ft2go_rec->dstPort = *((u_int16 *) (rec + fo.dstport));
    }


}
*/
import "C"
import (
	"os"
	"sync"
	"unsafe"
)

const StructSize = 1 << 28

type Generator struct {
	jobsChannel chan []*ftrecord
	errChannel  chan error
	mu          sync.Mutex
	wg          sync.WaitGroup
	closed      bool
	threads     int
}

func NewGenerator(threads int) *Generator {
	return &Generator{
		jobsChannel: make(chan []*ftrecord, threads),
		errChannel:  make(chan error, 1),
		threads:     threads,
		wg:          sync.WaitGroup{},
	}
}
func (g *Generator) Go(done chan<- struct{},paths []string) {

	cgoLimiter := make(chan struct{}, g.threads)
	go func() {
		defer func() {
			close(g.errChannel)
			close(cgoLimiter)
		}()
		for _, path := range paths {
			cgoLimiter <- struct{}{}
			g.wg.Add(1)
			go func(filename string) {
				defer func() {
					<-cgoLimiter
					done <- struct{}{}
					g.wg.Done()
				}()
				records, err := init_entrys(filename)
				if err != nil {
					g.errChannel <- err
					return
				}
				g.jobsChannel <- records
			}(path)
		}
		g.safeClosed()
	}()
}
func (g *Generator) safeClosed() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed{
		g.wg.Wait()
		close(g.jobsChannel)
		g.closed = true
	}
}
func init_entrys(path string) ([]*ftrecord, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
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

	C.ft2goarr(ftio, &cgoRecords[0])
	ft := make([]*ftrecord, flowsCount)
	for i := 0; i < flowsCount; i++ {
		rec := &ftrecord{
			exAddr:  int2ip(uint32(cgoRecords[i].exAddr)),
			srcAddr: int2ip(uint32(cgoRecords[i].srcAddr)),
			dstAddr: int2ip(uint32(cgoRecords[i].dstAddr)),
			bytes:   uint32(cgoRecords[i].bytes),
		}
		ft[i] = rec
		C.free(unsafe.Pointer(cgoRecords[i]))
	}

	C.free(unsafe.Pointer(cgoRecords))
	return ft, nil
}
