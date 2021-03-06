package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"gopkg.in/mail.v2"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var usage = func() {
	fmt.Println("usage: ./ft2go -path test/ " +
		"-ex router1  -nf vlan10," +
		" -conf config.toml -w 10 -pr=true" +
		" -csv=true -emails test@email.com -subj Netflow" +
		"ex: ./bot -path var/log/netflow -ex br0  -nf satys," +
		"-w 8 -pr=true -csv=true -emails username@mails.ru -subj Netflow")
}

func WalkPath(root string) (paths []string, err error) {
	err = filepath.Walk(root, func(path string,
		info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() ||
			strings.Contains(info.Name(), "tmp") {
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
func DataInfo(conf *Config, exAdders, filters, dir string) (ex []net.IP, c *Clients, err error) {
	exAddressSplit := strings.Split(exAdders, ",")
	if len(exAddressSplit) == 0 {
		return nil, nil, ErrInvalidExAddr
	}
	exAddersIPs := make([]net.IP, 0)
	for _, device := range conf.Devices {
		for _, attr := range device.Attr {
			for _, exAddr := range exAddressSplit {
				if exAddr == attr.Name {
					ip := net.ParseIP(attr.IP)
					if ip == nil {
						return nil, nil, fmt.Errorf("[!] Error :: %v :: %v \n", ErrInvalidIP, attr.IP)
					}
					exAddersIPs = append(exAddersIPs, ip)
				}
			}
		}
	}

	nFilters := strings.Split(filters, ",")
	if len(nFilters) == 0 {
		return nil, nil, ErrNoFilters
	}

	var clients Clients
	clients = make([]*client, 0)
	splitPath := strings.Split(dir, "/")
	dateStr := ""
	if len(splitPath) == 0 {
		dateStr = time.Now().Format("2006-01-02")
	} else {
		dateStr = splitPath[len(splitPath)-2]
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			dateStr = time.Now().Format("2006-01-02")
		}
	}

	for _, filter := range conf.Nfilters {
		for _, attr := range filter.Attr {
			for _, nFilter := range nFilters {
				if attr.Name == nFilter {
					c := &client{
						Name:    attr.Name,
						ipNets:  make([]*net.IPNet, 0),
						exAddr:  make([]string, 0),
						iface:   attr.Interface,
						dateStr: dateStr,
					}
					for _, ips := range attr.Ips {
						_, ipNet, err := net.ParseCIDR(ips)
						if err != nil {
							fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
						}
						c.ipNets = append(c.ipNets, ipNet)
					}
					for _, ex := range exAddersIPs {
						for _, device := range conf.Devices {
							for _, attr := range device.Attr {
								if attr.IP == ex.String() {
									c.exAddr = append(c.exAddr, attr.Name)
								}
							}
						}
					}
					clients = append(clients, c)
				}
			}
		}
	}
	return exAddersIPs, &clients, nil
}

func main() {
	exAdders := flag.String("ex", "", "gateways name with sep=,")
	filters := flag.String("nf", "", "nfilter with sep=,")
	configPath := flag.String("conf", "config.toml", "path to configfile")
	dirFiles := flag.String("path", "", "file directory")
	emails := flag.String("emails", "", "emails with sep=,")
	subject := flag.String("subj", "Netflow", "subject for mail")
	numCPU := flag.Int("w", 8, "workers")
	print := flag.Bool("pr", false, "print table")
	csvFile := flag.Bool("csv", false, "csv file")
	help := flag.String("h", "", "help")
	flag.Parse()
	flag.Usage = usage
	if len(os.Args) < 2 ||
		len(*help) != 0 ||
		len(*filters) == 0 ||
		len(*exAdders) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Reading config...\n")
	conf, err := ReadConfig(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
		os.Exit(1)
	}

	exAddersIPs, clients, err := DataInfo(conf, *exAdders, *filters, *dirFiles)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%v", err)
	}

	//Get absolute path for file flow - tools
	ftFiles, err := WalkPath(*dirFiles)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[!] Error :: %s \n", err.Error())
		os.Exit(1)
	}

	if *numCPU == 1 {
		*numCPU = runtime.NumCPU()
	}

	generator := NewGenerator(*numCPU, ftFiles)
	generator.Go(ftFiles)
	if err := generator.Start(exAddersIPs, clients); err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "%v", err.Error())
	}

	go func() {
		for range time.Tick(time.Minute) {
			runtime.GC()
		}
	}()


	// Show results
	if *print {
		clients.Report(os.Stdout)
	}

	if *csvFile {
		err = clients.WriteCSV()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	}

	if len(*emails) != 0 {
		mails := strings.Split(*emails, ",")
		d := mail.NewDialer(conf.SMTP.Server, conf.SMTP.Port, conf.SMTP.Mail, conf.SMTP.Pass)
		for _, xMail := range mails {
			m := mail.NewMessage()
			m.SetHeader("From", conf.SMTP.Mail)
			m.SetHeader("To", xMail)
			m.SetHeader("Subject", *subject)
			m.SetBodyWriter("text/plain", func(writer io.Writer) error {
				clients.Report(writer)
				return nil
			})
			d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
			if *csvFile {
				f, err := os.Open(clients.GetNames())
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
				}
				defer f.Close()
				m.Attach(clients.GetNames())
			}
			if err := d.DialAndSend(m); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
			_, _ = fmt.Fprintf(os.Stdout, "Send mail to :: %s :: Done\n", xMail)
		}

	}
	fmt.Println("Done.")
}
