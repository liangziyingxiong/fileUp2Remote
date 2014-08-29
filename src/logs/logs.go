package logs

import (
	"os"
	"fmt"
	"log"
	"sync"
	"time"
	"bufio"
	"compress/gzip"
)

const (
	PDNS  = "01"
	PHTTP = "02"

	LOGTYPE_INFO  = "I"

	LOG_INFO     = 0x0001
	LOG_DEBUG    = 0x0002
)
// I [time] PHTTP CLIENTIP S_URI S_REFER R_URI

type Log struct {
	logfile    string
	logger     *log.Logger
	Logfd      *os.File
	timeSufx   string
	level      int

	flag       int
	mu         sync.Mutex
}

func (rp *Log) CreateLog(logfile string, flag int, level int) {
	rp.logfile = logfile
	rp.timeSufx = logTimeSuffix()
	logfile = logfile + rp.timeSufx

	if rp.logfile != "stdout" {
		Logfd := new(os.File)
		Logfd, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %v failed - %v\n", logfile, err)
			os.Exit(1)
		}
		rp.Logfd = Logfd
		rp.level    = level
	} else {
		rp.Logfd = os.Stdout
		rp.level     = LOG_DEBUG
	}
	rp.flag      = flag
	rp.logger = log.New(rp.Logfd, "", flag)
}

func logTimeSuffix() string {
	now := time.Now()
	year, month, day := now.Date()
	return fmt.Sprintf(".%d%02d%02d", year, month, day)
}

func itoa(buf *[]byte, i int, wid int) {
	var u uint = uint(i)
	if u == 0 && wid <= 1 {
		*buf = append(*buf, '0')
		return
	}

	// Assemble decimal in reverse order.
	var b [32]byte
	bp := len(b)
	for ; u > 0 || wid > 0; u /= 10 {
		bp--
		wid--
		b[bp] = byte(u%10) + '0'
	}
	*buf = append(*buf, b[bp:]...)
}

func (rp *Log) formatTime(t time.Time) (timestring string){
	buf := make([]byte, 1)
	buf = buf[:0]
	year, month, day := t.Date()
	itoa(&buf, year, 4)
	buf = append(buf, '/')
	itoa(&buf, int(month), 2)
	buf = append(buf, '/')
	itoa(&buf, day, 2)
	buf = append(buf, ' ')

	hour, min, sec := t.Clock()
	itoa(&buf, hour, 2)
	buf = append(buf, ':')
	itoa(&buf, min, 2)
	buf = append(buf, ':')
	itoa(&buf, sec, 2)

	buf = append(buf, '.')
	itoa(&buf, t.Nanosecond()/1e3, 6)
	//*buf = append(*buf, ' ')
	timestring = string(buf)
	return
}

func (rp *Log) APrintf(logType string, format string, v ...interface{}) (err error) {
	if rp.logfile != "stdout" {
		rp.checkRollor()
	}

	now := time.Now()
	timestring := rp.formatTime(now)
	pstring := fmt.Sprintf(format, v...)
	defer func() {
		if r := recover(); r!= nil {
			fmt.Fprintf(os.Stderr, "%v\n", r)
		}
	}()
	err = rp.logger.Output(2, fmt.Sprintf("%s %d [%s] %s", logType, now.Unix(), timestring, pstring))
	return
}

func (rp *Log) Printf(format string, v ...interface{}) (err error) {
	if rp.level != LOG_DEBUG {
		return
	}
	if rp.logfile != "stdout" {
		rp.checkRollor()
	}
	//err = rp.logger.Output(2, fmt.Sprintf(format, v...))
	s := "[DEBUG] " + fmt.Sprintf(format, v...)
	err = rp.logger.Output(2, s)
	return
}

func (rp *Log) Info(format string, v ...interface{}) (err error) {
	if rp.logfile != "stdout" {
		rp.checkRollor()
	}
	s := "[INFO] " + fmt.Sprintf(format, v...)
	//err = rp.logger.Output(2, fmt.Sprintf(format, v...))
	err = rp.logger.Output(2, s)
	return
}

func (rp *Log) Error(format string, v ...interface{}) (err error) {
	if rp.logfile != "stdout" {
		rp.checkRollor()
	}
	s := "[ERROR] " + fmt.Sprintf(format, v...)
	err = rp.logger.Output(2, s)
	return
}

func gziplogfile(filename, taget_file string) {
	rf, e := os.Open(filename)
	if e != nil {
		return
	}
	buf := make([]byte, 1024)
	r := bufio.NewReader(rf)
	wf, e := os.OpenFile(taget_file + ".gz", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if e == nil {
		defer wf.Close()
		w := gzip.NewWriter(wf)
		for {
			n, _ := r.Read(buf)
			if n == 0 { break;}
			w.Write(buf)
		}
		w.Close()
	}
	rf.Close()
	os.Remove(filename)

}

func (rp *Log) checkRollor() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	nowSufx := logTimeSuffix()
	if nowSufx != rp.timeSufx && rp.logfile != "stdout" {
		rp.FdClose()
		go gziplogfile(rp.logfile + rp.timeSufx, rp.logfile)
		logfile := rp.logfile + nowSufx

		Logfd, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %v failed - %v\n", logfile, err)
			os.Exit(1)
		}
		rp.Logfd = Logfd
		rp.logger = log.New(rp.Logfd, "", rp.flag)
		rp.timeSufx = nowSufx
	}
}

func (rp *Log) FdClose() {
	if rp.Logfd != nil {
		rp.Logfd.Close()
	}
}
