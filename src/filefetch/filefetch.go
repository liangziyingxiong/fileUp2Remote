package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"bytes"

	"go.crypto/ssh"
	"go.crypto/ssh/agent"
	"logs"
	"sftp"
)

var (
	USER       = flag.String("user", "srslog", "ssh username")
	HOST       = flag.String("host", "115.182.75.5", "ssh server hostname")
	PORT       = flag.Int("port", 22, "ssh server port")
	PASS       = flag.String("pass", "", "ssh passwd")
	REMOTE_DIR = flag.String("remotedir", "/home/srslog/logs/", "remote dir")

	log_prefix = "srs.log-"
	logger logs.Log
)

func init() {
	flag.Parse()
	logger.CreateLog("stdout", log.Ldate|log.Ltime|log.Lshortfile, logs.LOG_DEBUG)
	if len(flag.Args()) < 1 {
		logger.Printf("subcommit required")
		os.Exit(-1)
	}
	if *PASS == "" {
		*PASS = "123456"
	}
}

func basename(name string) string {
	i := len(name) - 1
	// Remove trailing slashes
	for ; i > 0 && name[i] == '/'; i-- {
		name = name[:i]
	}
	// Remove leading directory name
	for i--; i >= 0; i-- {
		if name[i] == '/' {
			name = name[i+1:]
			break
		}
	}

	return name
}

func get_dev_ip() string {
	cmd := exec.Command("/bin/sh", "-c", "ifconfig | grep 'inet addr' | head -1 | awk '{print $2}' | cut -d: -f2")
	cmd_ret, err := cmd.CombinedOutput()
	if err != nil || len(cmd_ret) == 0 {
		logger.Error("get first net card ip failed, %v\n", err)
		os.Exit(-1)
	}
	cmd_ret = bytes.TrimSuffix(cmd_ret, []byte("\n"))
	cmd_ret = bytes.TrimSuffix(cmd_ret, []byte("\r"))
	return string(cmd_ret)
}

func sftp_pro() {
	var auths []ssh.AuthMethod
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}
	if *PASS != "" {
		auths = append(auths, ssh.Password(*PASS))
	}
	config := ssh.ClientConfig{
		User: *USER,
		Auth: auths,
	}
	addr := fmt.Sprintf("%s:%d", *HOST, *PORT)
	conn, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		logger.Error("unable to connect to [%s]: %v", addr, err)
		os.Exit(-1)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		logger.Error("unable to start sftp subsytem: %v", err)
		os.Exit(-1)
	}
	defer client.Close()

	switch cmd := flag.Args()[0]; cmd {
	case "ls":
		if len(flag.Args()) < 2 {
			logger.Error("%s %s: remote path required", cmd, os.Args[0])
			os.Exit(-1)
		}
		walker := client.Walk(flag.Args()[1])
		for walker.Step() {
			if err := walker.Err(); err != nil {
				logger.Error("%v\n", err)
				continue
			}
			logger.Printf("%s\n", walker.Path())
		}
	case "put":
		if len(flag.Args()) < 2 {
			logger.Error("%s %s: local file required", cmd, os.Args[0])
			os.Exit(-1)
		}
		l_files, err := walk_local_dir(flag.Args()[1])
		if err != nil {
			logger.Error("%v\n", err)
			os.Exit(-1)
		}
		logger.Printf("%q\n", l_files)
		ip := get_dev_ip()
		for _, l_file := range l_files {
			remote_file := *REMOTE_DIR + ip + "-" + basename(l_file)
			logger.Info("%s to %s\n", l_file, remote_file)
			f, err := client.Create(remote_file)
			if err != nil {
				logger.Error("%v", err)
				continue
			}
			fr, err := os.Open(l_file)
			if err != nil {
				logger.Printf("%v\n", err)
				continue
			}
			if _, err := io.Copy(f, fr); err != nil {
				logger.Error("%v\n", err)
				f.Close()
				fr.Close()
				continue
			}
			os.Remove(l_file)
			f.Close()
			fr.Close()
		}
	case "stat":
		if len(flag.Args()) < 2 {
			logger.Error("%s %s: remote path required", cmd, os.Args[0])
			os.Exit(-1)
		}
		f, err := client.Open(flag.Args()[1])
		if err != nil {
			logger.Error("%v\n", err)
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			logger.Error("unable to stat file: %v", err)
			os.Exit(-1)
		}
		logger.Printf("%s %d %v\n", fi.Name(), fi.Size(), fi.Mode())
	case "rm":
		if len(flag.Args()) < 2 {
			logger.Error("%s %s: remote path required", cmd, os.Args[0])
			os.Exit(-1)
		}
		if err := client.Remove(flag.Args()[1]); err != nil {
			logger.Error("unable to remove file: %v", err)
			os.Exit(-1)
		}
	case "mv":
		if len(flag.Args()) < 3 {
			logger.Error("%s %s: old and new name required", cmd, os.Args[0])
			os.Exit(-1)
		}
		if err := client.Rename(flag.Args()[1], flag.Args()[2]); err != nil {
			logger.Error("unable to rename file: %v", err)
			os.Exit(-1)
		}
	default:
		logger.Error("unknown subcommand: %v", cmd)
		os.Exit(-1)
	}
}

func walk_local_dir(ldir string) ([]string, error) {
	files := make([]string, 0)
	dir, err := os.Open(ldir)
	if err != nil {
		logger.Printf("%v\n", err)
		return files, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		logger.Printf("%v\n", err)
		return files, err
	}
	sort.Strings(names)
	for _, file := range names {
		if strings.HasPrefix(file, log_prefix) {
			files = append(files, ldir + file)
		}
	}
	return files, nil
}

func main() {
	logger.Printf("[%s]\n", get_dev_ip())
	sftp_pro()
}
