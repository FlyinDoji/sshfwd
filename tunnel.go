package sshfwd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type Endpoint struct {
	Host string
	Port int
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// TunnelConfig ssh authentication and settings
type TunnelConfig struct {
	User       string
	Password   *string
	Keyfile    *string
	Passphrase *string
	Timeout    time.Duration
}

func (cfg TunnelConfig) ClientConfig() *ssh.ClientConfig {

	if cfg.Password != nil {
		return &ssh.ClientConfig{User: cfg.User, Auth: []ssh.AuthMethod{ssh.Password(*cfg.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: cfg.Timeout}
	}

	if cfg.Keyfile == nil {
		log.Fatal("authentication method required: Keyfile or Password")
	}

	b, err := ioutil.ReadFile(*cfg.Keyfile)
	if err != nil {
		log.Fatal("cannot read key: ", err)
	}

	var signer ssh.Signer

	if cfg.Passphrase != nil {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(b, []byte(*cfg.Passphrase))
	}
	signer, err = ssh.ParsePrivateKey(b)
	if err != nil {
		log.Fatal("cannot parse key: ", err)
	}

	return &ssh.ClientConfig{User: cfg.User, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: cfg.Timeout}
}

type Tunnel struct {
	Local  Endpoint
	Proxy  Endpoint
	Remote Endpoint

	openSig  chan struct{}
	shutdown chan struct{}

	logger *log.Logger

	cfg *ssh.ClientConfig
}

func NewTunnel(l, p, r Endpoint, cfg *TunnelConfig, logger *log.Logger) *Tunnel {

	if logger == nil {
		logger = log.New(os.Stderr, "DefaultLogger:", log.Lshortfile)
	}
	return &Tunnel{l, p, r,
		make(chan struct{}), make(chan struct{}), logger, cfg.ClientConfig()}
}
func (t *Tunnel) Start() {

	// Establish connection to proxy
	proxyConn, err := ssh.Dial("tcp", t.Proxy.String(), t.cfg)
	if err != nil {
		t.logger.Fatal("cannot dial proxy:", err)
	}
	defer proxyConn.Close()

	// Open local endpoint
	listener, err := net.Listen("tcp", t.Local.String())
	if err != nil {
		t.logger.Fatal("cannot start listener:", err)
		return
	}
	defer listener.Close()
	defer t.logger.Println("Tunnel closed.")

	// Listen in a separate goroutine
	// Allows to select between the shutdown signal and newly received connection
	r := make(chan net.Conn)
	go func() {

		for {
			local, err := listener.Accept()
			if err != nil {
				t.logger.Println("(listener connection error):", err)
				return
			}
			r <- local
		}
	}()

	// Signal ready-for-use
	close(t.openSig)

	for {

		select {
		case <-t.shutdown:
			return
		case local := <-r:
			// Connection from local to remote
			go t.forward(local, proxyConn)
		}
	}
}

// Stop stops the channel and terminates all associated goroutines
func (t *Tunnel) Stop() {
	close(t.shutdown)
}

// Wait blocks the caller until the tunnel is established and ready for use
func (t *Tunnel) Wait() {
	<-t.openSig
}

// If the remote endpoint is not available, the local connection will be closed
// Clients connecting through the local endpoint will see the error
func (t *Tunnel) forward(local net.Conn, proxy *ssh.Client) {

	remote, err := proxy.Dial("tcp", t.Remote.String())
	if err != nil {
		t.logger.Println("(forward - cannot dial remote endpoint):", err)
		_ = local.Close()
		return
	}

	copyConn := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		_, err := io.Copy(dest, src)
		if err != nil {
			t.logger.Println("(copy):", err)
		}
	}

	go copyConn(local, remote)
	go copyConn(remote, local)
}
