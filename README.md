# sonar dhcp-batcher w/ integrated relay and proxy.
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
the program is a monolithic binary, with two different modes baked in. "batch" mode works like the traditional sonar batcher, where a script on a router hits a rest endpoint on the batcher, which then adds (or removes) client entries and schedules them for batching.

"proxy" (or relay but it really is proxy) mode functions by intercepting the DHCP protocol request data, siphoning the client data from it and then proxying it upstream to the clients DHCP server(s). The servers respond back to the proxy, which then forwards the DHCP requests back to the router to broadcast to the requesting client.

## features

* baked in TLS 1.2 support in batch mode, including port 80 redirect, secure right off the hop without requiring LetsEncrypt (which you can still use if you like). Generate a self signed cert and you're off to the races.
* verbose logging lets you find out why things are batching or updating, and isolate problems quickly. Logging modes include traditional text and JSON formats for programmatic parsing.
* small memory footprint (8.0 MB! for the batcher and proxy), and easy deployment
* run concurrent instances with different parameters to support multiple proxy subnets etc.
* proxy works with single interface or multi-interface NICs, improve edge security by running a proxy in front of your production dhcp servers!




