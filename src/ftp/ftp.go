package ftp

import (
	"os"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type FTP struct {
	host    string
	port    int
	user    string
	passwd  string
	pasv    int
	cmd     string
	Code    int
	Message string
	Debug   bool
	stream  []byte
	conn    net.Conn
	Error   error
}

func (ftp *FTP) debugInfo(s string) {
	if ftp.Debug {
		fmt.Println(s)
	}
}

func (ftp *FTP) Connect(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	ftp.conn, ftp.Error = net.Dial("tcp", addr)
	if ftp.Error != nil {
		return
	}
	ftp.Response()
	ftp.host = host
	ftp.port = port
}

func (ftp *FTP) Login(user, passwd string) {
	ftp.Request("USER " + user)
	ftp.Request("PASS " + passwd)
	ftp.user = user
	ftp.passwd = passwd
}

func (ftp *FTP) Response() (code int, message string) {
	ret := make([]byte, 1024)
	n, e := ftp.conn.Read(ret)
	if e != nil {
		fmt.Printf("%v\n", e)
		return
	}
	msg := string(ret[:n])
	code, _ = strconv.Atoi(msg[:3])
	message = msg[4 : len(msg)-2]
	ftp.debugInfo("<*cmd*> " + ftp.cmd)
	ftp.debugInfo(fmt.Sprintf("<*code*> %d", code))
	ftp.debugInfo("<*message*> " + message)
	return
}

func (ftp *FTP) Request(cmd string) {
	ftp.conn.Write([]byte(cmd + "\r\n"))
	ftp.cmd = cmd
	ftp.Code, ftp.Message = ftp.Response()
	if cmd == "PASV" {
		start, end := strings.Index(ftp.Message, "("), strings.Index(ftp.Message, ")")
		s := strings.Split(ftp.Message[start:end], ",")
		fmt.Printf("s : %q, l1 : %s, l2 : %s\n", s, s[len(s)-2], s[len(s)-1])
		l1, _ := strconv.Atoi(s[len(s)-2])
		l2, _ := strconv.Atoi(s[len(s)-1])
		ftp.pasv = l1*256 + l2
	}
	if strings.HasPrefix(cmd, "PORT") {
		ftp.pasv = 202*256 + 207
		//return
	}
	if (cmd != "PASV") && (ftp.pasv > 0) {
		fmt.Printf("cmd : %s\n", cmd)
		//ftp.Message = newRequest(ftp.host, ftp.pasv, ftp.stream)
		ftp.Message = newRequest(ftp.host, 20, ftp.stream)
		ftp.debugInfo("<*response*> " + ftp.Message)
		ftp.pasv = 0
		ftp.stream = nil
		ftp.Code, _ = ftp.Response()
	}
}

func (ftp *FTP) Pasv() {
	ftp.Request("PASV")
}

func (ftp *FTP) Pwd() {
	ftp.Request("PWD")
}

func (ftp *FTP) Cwd(path string) {
	ftp.Request("CWD " + path)
}

func (ftp *FTP) Mkd(path string) {
	ftp.Request("MKD " + path)
}

func (ftp *FTP) Size(path string) (size int) {
	ftp.Request("SIZE " + path)
	size, _ = strconv.Atoi(ftp.Message)
	return
}

func (ftp *FTP) List() {
	ftp.Pasv()
	ftp.Request("LIST")
}

func (ftp *FTP) Stor(file string, data []byte) {
	//ftp.Pasv()
	//mode_start := strings.Index(ftp.Message,"(") + 1
	//mode_end := strings.Index(ftp.Message, ")")
	//fmt.Printf("******************message : %s\n", ftp.Message[mode_start:mode_end])
	////ftp.Request("PORT " + ftp.Message[mode_start:mode_end])
	ftp.Request("PORT 115,182,75,54,202,207")
	if data != nil {
		ftp.stream = data
	}
	ftp.Request("STOR " + file)
}

func (ftp *FTP) Quit() {
	ftp.Request("QUIT")
	ftp.conn.Close()
}

// new connect to FTP pasv port, return data
func newRequest(host string, port int, b []byte) string {
	conn, e := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if e != nil {
		fmt.Printf("%v\n", e)
		os.Exit(-1)
	}
	defer conn.Close()
	if b != nil {
		conn.Write(b)
		return "OK"
	}
	ret := make([]byte, 4096)
	n, _ := conn.Read(ret)
	return string(ret[:n])
}
