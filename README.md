# Transparent http proxy with Golang 1.11+ and tproxy
This is a basic implementation of fully transparent reverse or forward proxy in Go.  

It requires Go 1.11+ becouse of ablility to create custom socket with IP_TRANSPARENT param.
From Go 1.11 there is possible to pass socket option before start listening or dialing. ListenConfig provide this. (https://golang.org/pkg/net/#ListenConfig)  
It working together with tproxy, which allows to redirect traffic designated to remote location to the local process.

The key in implementation is to create custom listener for http.Serve and use LocalAddrContextKey to get destinetion address to which client want to connect. In fact address:port values from http.LocalAddrContextKey, are the values from local socket dynamicly created by tproxy. 

## Example usage scenario - as transparent http proxy on router/gateway for local network
```
+------------------------------------------+   +--------------------------------------------+
|                                          |   |                                            |
|                        +-----------------+---+----------------+                           |
|   Local network        |       Router, gateway on Linux       |     Wan Network           |
|   192.168.1.1/24 -->>  |           http proxy in Go           |     37.247.61.X -->>      |
|   eth1                 |                                      |     eth0                  |
|                        +-----------------+---+----------------+                           |
|                                          |   |                                            |
+------------------------------------------+   +--------------------------------------------+
```
In this example http traffic outgoing from local network 192.168.1.0/24 is passing transparently through http proxy, located on default gateway for this network.

## Tproxy configuration (From Linux 4.18 tproxy is included in nf_tables):
```
# create new routing table and tell that 0.0.0.0/0 addresses range is a local.
ip route add local 0.0.0.0/0 dev lo table 100
 
# redirect marked packets to table created above
ip rule add fwmark 1 lookup 100
 
# mark packets with dst port = 80 (and use route table 100) nad redirect to Go http proxy listening on 127.0.0.1:8888
iptables -t mangle -A PREROUTING -s 192.168.1.0/24 -p tcp --dport 80 -j TPROXY --tproxy-mark 0x1/0x1 --on-port 8888 --on-ip 127.0.0.1
```

How tproxy works in details is described here:  
https://www.kernel.org/doc/Documentation/networking/tproxy.txt  
https://powerdns.org/tproxydoc/tproxy.md.html  
https://people.netfilter.org/hidden/nfws/nfws-2008-tproxy_slides.pdf  

## Start proxy:  
```
go run proxy.go
```
## Checking it
Client from local network (192.168.1.6) is connecting to remote site on port 217.73.181.X:80. This connection is handled through the proxy.
Nestat is showing one very interesting thing:

```
router:~ # netstat -tpna | grep proxy | grep ESTABLISHED
tcp        0      0 37.247.61.X:57554       217.73.181.X:80       ESTABLISHED 12777/proxy
tcp        0      0 217.73.181.X:80       192.168.1.6:59752       ESTABLISHED 12777/proxy
```
Tproxy created tcp socket with remote site address - 217.73.181.X:80 on my local machine. My router has only 192.168.1.1 and 37.247.61.X addresses, routing table 100 does the job.
Go http proxy after receive request from client (192.168.1.6), made a connection to exactly the same address:port as it received. MAGIC! :)

More deails about this: https://taczanowski.net/transparent-http-proxy-with-golang-and-tproxy/
