package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/skylinnan/ghca/ghca-logs"
	"net"
	"os"
	"strconv"
)
//var testmerge
var serverip string
var exit chan bool
var l4port, aaaport int
var l4recvchan, l4sendchan, aaarecvchan, aaasendbackchan chan []byte
var l4conn, aaaconn *net.UDPConn
var logfile *logs.BeeLogger
var prenodeconfig config.Configer

type RadiusData struct {
	data []byte
	addr *net.UDPAddr
}

func init() {
	var err error
	prenodeconfig, err = config.NewConfig("ini", "../etc/prenode.ini")
	panic_err(err)
	serverip = prenodeconfig.String("server::ip")
	if serverip == "" {
		serverip = "0.0.0.0"
	}
	l4port, _ = prenodeconfig.Int("server::l4port")
	if l4port == 0 {
		l4port = 1812
	}
	aaaport, _ = prenodeconfig.Int("server::aaaport")
	if aaaport == 0 {
		aaaport = 2812
	}
	addr, _ := net.ResolveUDPAddr("udp", serverip+":"+strconv.Itoa(l4port))
	l4conn, err = net.ListenUDP("udp", addr)
	panic_err(err)
	addr, _ = net.ResolveUDPAddr("udp", serverip+":"+strconv.Itoa(aaaport))
	aaaconn, err = net.ListenUDP("udp", addr)
	panic_err(err)
	logfile = init_logptr("prenode.log")

}
func init_logptr(logname string) *logs.BeeLogger {
	logdir, err := os.Stat("log")
	if err != nil {
		err = os.Mkdir("log", 0777)
		panic_err(err)
	} else {
		if logdir.IsDir() == false {
			err = os.Mkdir("log", 0777)
			panic_err(err)
		}
	}
	logptr := logs.NewLogger(10000)
	loglevel, _ := prenodeconfig.Int("log::debug")
	if loglevel == 0 || loglevel > 4 {
		loglevel = 1
	}
	logptr.SetLevel(loglevel)
	logsize, _ := prenodeconfig.Int("log::logsize")
	if logsize == 0 {
		logsize = 500
	}
	logsaveday, _ := prenodeconfig.Int("log::logsavedays")
	if logsaveday == 0 {
		logsaveday = 3
	}
	logname = "../log/" + logname
	logstr := fmt.Sprintf(`{"filename":"%s","maxsize":%d,"maxdays":%d}`, logname, logsize, logsaveday)
	logptr.SetLogger("file", logstr)
	return logptr
}
func panic_err(err error) {
	if err != nil {
		panic(err.Error())
	}
}
func recvfroml4data() {
	l4recvchan = make(chan []byte, 1000)
	for {
		var b [65535]byte
		n, addr, _ := l4conn.ReadFromUDP(b[:])
		m := b[:n]
		m = append(m, addr.IP...)
		port := int32(addr.Port)
		fmt.Println(port, addr.Port)
		b_buf := bytes.NewBuffer([]byte{})
		binary.Write(b_buf, binary.BigEndian, port)
		m = append(m, b_buf.Bytes()...)
		fmt.Println(m)
		l4recvchan <- m
		fmt.Println("stop to send")
	}
}
func sendbackl4data() {
	for {
		b := <-l4sendchan
		_, err := l4conn.Write(b)
		if err != nil {
			logfile.Info("send back l4 %s.", err.Error())
		}
	}
}
func sendtoaaa() {
	for {
		b := <-aaarecvchan
		_, err := aaaconn.Write(b)
		if err != nil {
			logfile.Info("send to aaa %s.", err.Error())
		}
	}
}
func main() {
	fmt.Println("started.")
	go recvfroml4data()
	go sendbackl4data()
	go sendtoaaa()
	logfile.Info("started.")
	<-exit
}
