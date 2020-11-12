package main

import (
	"encoding/csv"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

type Clients []*client

var header = []string{"NAME", "DATE", "EXADDR", "IFACE", "TOTAL_BYTES", "TOTAL\n"}

func (clients Clients) Report(w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetHeader(header)
	for _, c := range clients {
		table.Append(c.arrString())
	}
	table.Render()
}

func (clients Clients) WriteCSV() error {
	f, err := os.Create(clients.GetNames())
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	err = w.Write(header)
	if err != nil {
		return err
	}
	for _, c := range clients {
		w.Write(c.arrString())
	}

	w.Flush()
	return w.Error()
}

type client struct {
	Name    string
	exAddr  []string
	ipNets  []*net.IPNet
	iface   int
	dateStr string
	sum     uint64
}

func (clients Clients) GetNames() string {
	var str string
	for _, c := range clients {
		str += c.Name
	}
	return clients[0].dateStr + "_" + str + ".csv"
}
func (c *client) arrString() []string {
	var arr = make([]string, 0)
	arr = append(arr, c.Name, c.dateStr)
	arr = append(arr, strings.Join(c.exAddr, ","))
	arr = append(arr,strconv.Itoa(c.iface))
	arr = append(arr, strconv.FormatUint(c.sum, 10))
	arr = append(arr, byteCountIEC(int64(c.sum)))
	return arr
}

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
