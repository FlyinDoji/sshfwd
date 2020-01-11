# sshfwd 

Forward traffic and connections between two remote endpoints via ssh tunnel.

Inspired by 

https://medium.com/elliotchance/how-to-create-an-ssh-tunnel-in-go-b63722d682aa

Original implementation would leave connections hanging if the remote endpoint was not reachable.
Added methods for shutting down the tunnel and awaiting for tunnel to be established.


Supported authentication methods:
1. Password
2. Keypairs generated through OpenSSH with no passphrase

# Usage 

As a standalone executable

```
sshfwd -l [local] -p [sshServer] -r [remote] -u [sshUser] -k (optional) [keyfile] -pw (optional) [ssh user password]
```


As a library

``` 
local := sshfwd.Endpoint{Host: localIp, Port: localPort}
proxy := sshfwd.Endpoint{Host: proxyIp, Port: proxyPort}
remote := sshfwd.Endpoint{Host: remoteIp, Port: remotePort}

// Authentication by private key file
cfg = sshfwd.TunnelConfig{
    User:       *user,
    Keyfile:    keyfile,
    Timeout:    10 * time.Second,
  }

t := sshfwd.NewTunnel(local, proxy, remote, &cfg, nil)
// Start the tunnel in a separate goroutine
go t.Start()

// t.Wait() will block until tunnel is ready for use
t.Wait()
time.Sleep(time.Second * 2)

// Stop the tunnel and shut down all goroutines
t.Stop()
```