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
		 unsigned long sif;
		 unsigned long dif;
}ft2go;
struct ftio ftio;
struct ftprof ftp;
struct fts3rec_offsets fo;
struct ftver ftv;
void ft2goarr(struct ftio *ftio,ft2go **in){
    struct ftprof ftp;
    struct fts3rec_offsets fo;
    struct ftver ftv;
    char *rec;

    ftprof_start(&ftp);
    ftio_get_ver(ftio, &ftv);
    fts3rec_compute_offsets (&fo, &ftv);
	int i = 0;
    while((rec = ftio_read(ftio))){
        in[i]->exAddr  = *((u_int32 *) (rec + fo.exaddr));
        in[i]->srcAddr = *((u_int32 *) (rec + fo.srcaddr));
        in[i]->dstAddr = *((u_int32 *) (rec + fo.dstaddr));
  		in[i]->bytes = *((u_int32 *) (rec + fo.dOctets));
		in[i]->sif = *((u_int16 *) (rec + fo.input));
		in[i]->dif = *((u_int16 *) (rec + fo.output));
		i++;
		// for adding ports or other fields see https://github.com/adsr/flow-tools/blob/master/lib/ftlib.h#L613
       // ft2go_rec->srcPort = *((u_int16 *) (rec + fo.srcport));
       // ft2go_rec->dstPort = *((u_int16 *) (rec + fo.dstport));
    }


}
*/
import "C"
import (
	"errors"
	"fmt"
	"github.com/gosuri/uiprogress"
	"golang.org/x/sync/errgroup"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

const StructSize = 1 << 28

type Mode int

const (
	ALL Mode = iota + 1
	IFACE
	IPNET
)


type Generator struct {
	jobsChannel chan []*ftrecord
	errChannel  chan error
	done        chan struct{}
	files       []string
	mu          sync.Mutex
	wg          sync.WaitGroup
	closed      bool
	threads     int
}

func NewGenerator(threads int, ftFiles []string) *Generator {
	return &Generator{
		jobsChannel: make(chan []*ftrecord, threads*2),
		errChannel:  make(chan error, 1),
		done:        make(chan struct{},len(ftFiles)),
		files:       ftFiles,
		mu:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		closed:      false,
		threads:     threads,
	}
}
func (g *Generator) Start(exAddersIPs []net.IP, clients *Clients) error {
	lengthFiles := len(g.files)
	uiprogress.Start()
	barGenerator := uiprogress.AddBar(lengthFiles)
	barGenerator.PrependElapsed().AppendCompleted()
	barGenerator.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Processing :: [ %d / %d ] :: ",
			barGenerator.Current(), lengthFiles)
	})

	errGroup := new(errgroup.Group)
	errGroup.Go(func() error {
		for err := range g.errChannel {
			if err != nil{
				if errors.Is(err,ErrFtioInvalid){
					_, _ = fmt.Fprintf(os.Stdin, "%s\n", err.Error())
				}
			}
		}
		return nil
	})
	errGroup.Go(func() error {
		for i := 0; i < len(g.files); i++ {
			<-g.done
			barGenerator.Incr()
		}
		g.safeClosed()
		uiprogress.Stop()
		return nil
	})
	var numCPU = 8
	for i := 0; i < numCPU; i++ {
		errGroup.Go(func() error {
			for batch := range g.jobsChannel {
				sliceFilter(exAddersIPs, clients, batch, numCPU)
			}
			return nil
		})
	}

	return errGroup.Wait()
}
func (g *Generator) Go(paths []string) {
	cgoLimiter := make(chan struct{}, g.threads*2)
	go func() {
		defer func() {
			close(cgoLimiter)
		}()
		for _, path := range paths {
			cgoLimiter <- struct{}{}
			g.wg.Add(1)
			go func(filename string) {
				defer func() {
					<-cgoLimiter
					g.done <- struct{}{}
					g.wg.Done()
				}()
				rec, err := g.init_entrys(filename)
				g.jobsChannel <- rec
				g.errChannel <- err
			}(path)
		}
	}()

}
func (g *Generator) safeClosed() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.wg.Wait()
		close(g.errChannel)
		close(g.jobsChannel)
		close(g.done)
		g.closed = true
	}
}
func (g *Generator) init_entrys(path string) ([]*ftrecord, error) {
	f, err := os.OpenFile(path,os.O_RDONLY,0664)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ftio := (*C.struct_ftio)(C.calloc(1, C.sizeof_ftio))
	defer C.free(unsafe.Pointer(ftio))

	errno := C.ftio_init(ftio, C.int(f.Fd()), C.FT_IO_FLAG_READ)
	if errno < 0 {
		return nil,ErrFtioInvalid
	}
	flowsCount := int(ftio.fth.flows_count)
	cgoRecords := (*[StructSize]*C.ft2go)(C.calloc(C.ulong(flowsCount), C.sizeof_ft2go))
	defer C.free(unsafe.Pointer(cgoRecords))

	for i := 0; i < flowsCount; i++ {
		cgoRecords[i] = (*C.struct_ft2go)(C.calloc(1, C.sizeof_ft2go))
	}

	C.ft2goarr(ftio, &cgoRecords[0])

	ft := make([]*ftrecord, flowsCount)
	for i := 0; i < flowsCount; i++ {
		rec := &ftrecord{}
		rec.exAddr = uint32(cgoRecords[i].exAddr)
		rec.srcAddr = uint32(cgoRecords[i].srcAddr)
		rec.dstAddr = uint32(cgoRecords[i].dstAddr)
		rec.bytes = uint32(cgoRecords[i].bytes)
		rec.sif = uint16(cgoRecords[i].sif)
		rec.dif = uint16(cgoRecords[i].dif)
		ft[i] = rec
		C.free(unsafe.Pointer(cgoRecords[i]))
	}

	return ft, nil
}

func sliceFilter(exAddr []net.IP, clients *Clients, batch []*ftrecord, numCPU int) {

	f := func(i, j int, c chan struct{}) {
		for ; i < j; i++ {
			for _, exAdd := range exAddr {
				if exAdd.Equal((batch)[i].ExAddr()) {
					for _, c := range *clients {
						for _, ipNet := range c.ipNets {
							if ipNet.Contains((batch)[i].SrcAddr())   ||
								ipNet.Contains((batch)[i].DstAddr())  ||
								c.iface == int((batch)[i].SourceIF()) ||
								c.iface == int((batch)[i].DstIF()){
								atomic.AddUint64(&c.sum, uint64((batch)[i].bytes))
							}
						}
					}
				}
			}
		}
		c <- struct{}{}
	}

	c := make(chan struct{}, numCPU)
	length := len(batch)
	for i := 0; i < numCPU; i++ {
		go f(i*length/numCPU, (i+1)*length/numCPU, c)
	}

	for i := 0; i < numCPU; i++ {
		<-c
	}
	batch = nil
}