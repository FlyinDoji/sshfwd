package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sshfwd"
)

func main() {

	var localIp, proxyIp, remoteIp string
	var localPort, proxyPort, remotePort int

	l := flag.String("l", "localhost:33333", "Address and port of the local endpoint")
	p := flag.String("p", "", "(required) Address and port of the ssh listener daemon")
	r := flag.String("r", "", "(required) Address and port of the remote endpoint")
	user := flag.String("u", "", "(required)ssh username")
	keyfile := flag.String("k", "", "path to ssh private key")
	password := flag.String("pw", "", "password for ssh user")

	flag.Parse()

	splitAddr := func(addr string, host *string, port *int) {
		var err error
		v := strings.Split(addr, ":")
		if len(v) != 2 {
			flag.PrintDefaults()
			os.Exit(-1)
		}
		*host = v[0]
		*port, err = strconv.Atoi(v[1])
		if err != nil {
			panic(err)
		}
	}

	splitAddr(*l, &localIp, &localPort)
	local := sshfwd.Endpoint{Host: localIp, Port: localPort}
	splitAddr(*p, &proxyIp, &proxyPort)
	proxy := sshfwd.Endpoint{Host: proxyIp, Port: proxyPort}
	splitAddr(*r, &remoteIp, &remotePort)
	remote := sshfwd.Endpoint{Host: remoteIp, Port: remotePort}

	var cfg sshfwd.TunnelConfig

	if *keyfile == "" {
		cfg = sshfwd.TunnelConfig{
			User:     *user,
			Password: password,
			Timeout:  10 * time.Second,
		}
	} else {
		cfg = sshfwd.TunnelConfig{
			User:    *user,
			Keyfile: keyfile,
			Timeout: 10 * time.Second,
		}
	}

	t := sshfwd.NewTunnel(local, proxy, remote, &cfg, nil)
	go t.Start()

	t.Wait()
	fmt.Println("Tunnel established")
	fmt.Println(local.String(), "---", proxy.String(), "---", remote.String())

	bufio.NewScanner(os.Stdin).Scan()
	fmt.Println("Shutting down")
	time.Sleep(time.Second * 2)
	t.Stop()
}
