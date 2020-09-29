#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <arpa/inet.h>
#include "ftlib.h"

typedef struct ft2go ft2go;
struct ft2go {
         unsigned long exAddrr;
         unsigned long srcAddrr;
         unsigned long dstAddrr;
         short int srcPort;
         short int dstPort;
         unsigned long bytes;
         ft2go *next;
};

ft2go *newItem(unsigned long exAddrr,unsigned long srcAddrr,unsigned long dstAddrr,
        short int srcPort,short int dstPort,unsigned long bytes);

ft2go *addfront(ft2go *listp,ft2go *newp);
ft2go *addend(ft2go *listarr,ft2go *newp);
void freeall(ft2go *listp);
ft2go *listEntry(char *path);