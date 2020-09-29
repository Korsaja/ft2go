#include "ft2go.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <arpa/inet.h>
#include "ftlib.h"

ft2go *newItem(unsigned long exAddrr,unsigned long srcAddrr,unsigned long dstAddrr,
        short int srcPort,short int dstPort,unsigned long bytes){
        ft2go *newtGo;
               newtGo = (ft2go *)malloc(sizeof(ft2go));
               newtGo->exAddrr = exAddrr;
               newtGo->srcAddrr = srcAddrr;
               newtGo->dstAddrr = dstAddrr;
               newtGo->srcPort = srcPort;
               newtGo->dstPort = dstPort;
               newtGo->bytes = bytes;
               newtGo->next = NULL;
               return newtGo;
        }

ft2go *addfront(ft2go *listp,ft2go *newp){
     newp->next = listp;
     return newp;
 }
ft2go *addend(ft2go *listarr,ft2go *newp){
    ft2go *p;
    if (listarr == NULL)
        return newp;
    for (p = listarr; listarr != NULL;p = p->next);
    p->next = newp;
    return listarr;
}

void freeall(ft2go *listp){
        ft2go *next;
        for (; listp != NULL; listp = next){
                next = listp->next;
                free(listp);
        }
}


ft2go *listEntry(char *path){
        ft2go *listarr;
        struct ftio ftio;
        struct ftprof ftp;
        struct fts3rec_offsets fo;
        struct ftver ftv;
        char *rec;
        unsigned long exAddrr;
        unsigned long srcAddrr;
        unsigned long dstAddrr;
        short int srcPort;
        short int dstPort;
        unsigned long bytes;
        u_int32 last_time;
        u_int32 tm;
        FILE *fp;
        int fd;

        if ((fp = fopen (path, "rb")) == NULL)
        {
          perror ("Cannot open file %m");
          return NULL;
        }
        fd = fileno(fp);

        ftprof_start(&ftp);

        if (ftio_init (&ftio, fd, FT_IO_FLAG_READ) < 0){
              perror("Error in initialization ftio structure: %m");
              return NULL;
        }
        ftio_get_ver (&ftio, &ftv);
        fts3rec_compute_offsets (&fo, &ftv);

        last_time = 0;
        while ( (rec = ftio_read (&ftio)) ){
                  tm = *((u_int32 *) (rec + fo.unix_secs));
                  if (last_time != tm)
                    {
                      exAddrr = 0;
                      srcAddrr = 0;
                      dstAddrr = 0;
                      srcPort = htons ((u_int16) ((tm >> 16) & 0xFFFF));
                      dstPort = htons ((u_int16) (tm & 0xFFFF));
                      bytes = 0;
                      listarr = addfront(listarr,newItem(exAddrr,srcAddrr,dstAddrr,srcPort,dstPort,bytes));
                      last_time = tm;
                    }
                  exAddrr  = htonl (*((u_int32 *) (rec + fo.exaddr)));
                  srcAddrr = htonl (*((u_int32 *) (rec + fo.srcaddr)));
                  dstAddrr = htonl (*((u_int32 *) (rec + fo.dstaddr)));
                  srcPort = htons (*((u_int16 *) (rec + fo.srcport)));
                  dstPort = htons (*((u_int16 *) (rec + fo.dstport)));
                  bytes = htonl (*((u_int32 *) (rec + fo.dOctets)));
                  listarr = addfront(listarr,newItem(exAddrr,srcAddrr,dstAddrr,srcPort,dstPort,bytes));
                }
          ftio_close (&ftio);
          return listarr;
}