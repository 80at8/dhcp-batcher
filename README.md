# sonar dhcp-batcher w/ integrated proxy.
![written in Go!](https://github.com/80at8/dhcp-batcher/blob/master/assets/netgopher.svg)

(not for production use, poc only!).

todo
- add router checks for batcher
- write tests for proxy code (thx 42wim! https://github.com/42wim/dhcprelay/)

break down a couple of large functions ~100 lines to smaller testable units.
convert structs to interface for better testing

## what is it?
this program batches DHCP client requests to Sonar V1 instances, and, after some testing it should also work with DHCP configurations that make extended use of Option 82.

## why
why not?

## how does it work
the program is a monolithic binary, with two different modes baked in.

"batch" mode works like the traditional sonar batcher, where a script on a router hits a rest endpoint on the batcher, which then adds (or removes) client entries and schedules them for batching.

![batching topology](https://github.com/80at8/dhcp-batcher/blob/master/assets/Screenshot%20from%202020-05-15%2014-31-31.png)

"proxy" (or relay but it really is proxy) mode functions by intercepting the DHCP protocol request data, siphoning the client data from it and then proxying it upstream to the clients DHCP server(s). The servers respond back to the proxy, which then forwards the DHCP requests back to the router to broadcast to the requesting client.

![proxying topology](https://github.com/80at8/dhcp-batcher/blob/master/assets/Screenshot%20from%202020-05-15%2014-25-23.png)

each mode runs a concurrent scheduler which will batch all discovered clients to sonar, the timer for the scheduler is adjustable using the --batch_cycle_time switch.

## features

* baked in TLS 1.2 support in batch mode, including port 80 redirect, secure right off the hop without requiring LetsEncrypt (which you can still use if you like). Generate a self signed cert and you're off to the races.
* verbose logging lets you find out why things are batching or updating, and isolate problems quickly. Logging modes include traditional text and JSON formats for programmatic parsing.
* small memory footprint (8.0 MB! for the batcher and proxy), and easy deployment
* run concurrent instances with different parameters to support multiple proxy subnets etc.
* proxy works with single interface or multi-interface NICs, improve edge security by running a proxy in front of your production dhcp servers!
* no conf files to mess with, use command line switches and a shell script, or dockerize it if you like.

## system requirements (batcher)

Requests/Second: NA
    OS: Ubuntu 18.04
    Memory: 2 GB RAM
    CPU: 1 core

## system requirements (proxy)

Requests/Second: 10-20 - 30-40% CPU load
    OS: Ubuntu 18.04
    Memory: 2 GB RAM
    CPU: 1 core
    DHCP: 1 upstream DHCP server (2 is always better though)

Requests/Second: 20-50 - 40-60% CPU load
    OS: Ubuntu 18.04
    Memory: 2 GB RAM
    CPU: 2 core
    DHCP: 2 upstream DHCP servers (3 is always better though)


## installation

#### Linux
from a fresh linux install (whatever version you like, but we'll use Ubuntu 18.04 in this example)

    sudo apt get install go
    sudo mkdir /opt/sonar/
    cd /opt/sonar/
    sudo git clone https://github.com/80at8/dhcp-batcher
    cd dhcp-batcher
    sudo mkdir logs
    sudo mkdir tls
    sudo chown -R <yourusername>:<yourusername> /opt/sonar/
    go get .
    go build

#### Windows (coming soon!)

## testing

the batcher and proxy haven't been throuroughly tested, so obviously don't use them on a production system -- I still have unit tests to write for the proxy code. I've tested the DHCP DORA proxying over a meraki relay, to the batcher-proxy to the client (Fluke LinkSprinter 200).

update: throughput testing has been completed using perfdhcp, it should be quite performant for most networks where subnets have aggregate request traffic of 10 to 50 requests per second with one or two upstream dhcp servers, for more requests per second more dhcp servers and proxies should be used.

Here are the numbers from perfdhcp:

100/SEC

    ISC Rate: 47.7622 4-way exchanges/second, expected rate: 100
    PROXY Rate: 47.4405 4-way exchanges/second, expected rate: 100
    
1000/SEC

    ISC Rate 388.77 4-way exchanges/second, expected rate: 1000
    PROXY Rate: 290.86 4-way exchanges/second, expected rate: 1000

Would be nice to test the API endpoints (thx Chris!) for V1 more thorougly, and convert some of the functions to function receivers and interfaces for better unit tests and code coverage.



## usage flags

run

    ./dhcp-batcher --help
to access the help w/ flag usage, the flags are covered in more detail below.

    -app_mode string
    	DHCP operation mode [ batch | proxy ] (default "batch")
sets the programs operation mode, either batch mode or proxy mode -- each mode uploads it's batched or discovered items to sonar.
 
    -batch_cycle_time int
    	Batch scheduler cycle time (in minutes), set to 0 to enable near-realtime batching (15 seconds) (default 5)
sets the batch cycle time in minutes, this is the interval that batched and proxy-discovered items are sent to sonar.

    -batch_http_port string
    	HTTP port to listen for dhcp batcher requests on, or redirect to TLS (default "80")
this is the port where the batch endpoint resides, when using TLS this option is overriden and port 80 is used as a redirect to TLS / port 443       

    -batch_ip string
    	Local IP to bind DHCP batching requests to (default "127.0.0.1")
the ip address that the batch endpoint will listen on.

    -batch_logging_format string
    	Batch endpoint logging format [ text | json ] (default "text")
this sets the format for the logging output, text is human redable, or JSON for something that can be parsed programmatically
  
    -batch_logging_mode string
    	Batch endpoint logging Level [ none | info | warn | debug ] (default "info")
the level of logging detail to record in the log

    -batch_logging_path string
    	Batch endpoint logging output [ path | "console"] (default "/opt/sonar/dhcp-batcher/logs/dhcpbatcher.log")
where to send the logging output, use a path to write to a file or use console to run batcher in interactive mode.

    -batch_password string      Password for batch endpoint authentication (minimum 16 characters)
    -batch_username string   	Username for batch endpoint authentication (minimum 5 characters)
the username and password for the batch endpoint (not the sonar instance!) -- this allows you to secure the endpoint with basic auth so that only authorized routers can create batch entries.

     -batch_tls_cert string
path to TLS public certificate (default "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.crt")
        
    -batch_tls_key string
path to TLS private key (default "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.key")
    
    -batch_tls_port string
TLS port to listen for dhcp batcher requests on (default "443")

    -batch_use_tls string
enable TLS, set to [true || 1] || [false || 0]
        
    -proxy_downstream_if string
downstream interface to listen for DHCP client requests on (default "eth1")

    -proxy_upstream_if string
upstream interface to pass requests to DHCP server(s) (default "eth0")

    -proxy_single_if string
downstream and upstream interface to listen to requests on, if specified disables --proxy_upstream_if and   

    -proxy_server_ip string
proxy server IP address that routers will point to as relay ip (must be bound to downstream interface)

    -proxy_upstream_dhcp_ips string
IP addresses of the DHCP servers ["a.b.c.d" || "a.b.c.d, ..., w.x.y.z"]

    -sonar_api_key string
v1 sonar password or v2 sonar bearer token

    -sonar_api_username string
v1 sonar username

    -sonar_instance string
v1 or v2 sonar instance name (use FQDN e.g: example.sonar.software)
  
    -sonar_version int
sonar version batcher will report to, [ 1 | 2 ] (default 2)

## basic command example

```sudo ./dhcp-batcher -app_mode proxy --proxy_upstream_dhcp_ips DHCPSERVER_IP1,DHCPSERVER_IP2 --proxy_server_ip VMSERVER_IP -proxy_single_if INTERFACENAME --batch_logging_path console --batch_logging_mode debug --sonar_api_username SONAR_USERNAME --sonar_instance SONAR_INSTANCE --sonar_api_key SONAR_API```
