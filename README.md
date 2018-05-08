## Basic Junos OpenConfig telemetry tester

This is a Go script and demonstrates how to retrieve OpenConfig telemetry KV pair data from Junos.

This test script takes the concept of [Nilesh Simaria's JTIMon](https://github.com/nileshsimaria/jtimon) and boils it down to the raw basics. 

This script might be useful for testing basic configuration of gRPC on Junos. You can also use it to test TLS (both unidirectional and mutual auth).

Do not use this for anything other than curiosity and hacking. It's a script afterall.

## Build

Note, you still require the `.proto` files from the Juniper download website for your version of Junos. These are then converted into Go files by the `protoc` tool and the code in this repo is compiled with those files as dependencies. I have included three binary versions for simplicity.

```bash
junos-openconfig-telemetry-testclient-junos-32-0.1
junos-openconfig-telemetry-testclient-linux-64-0.1
junos-openconfig-telemetry-testclient-osx-0.1
```

If you did want to build, this repo contains Godep info for an easy restore.

```bash
# Clone in to your Go code repository
git clone https://github.com/arsonistgopher/junos-openconfig-telemetry-testclient.git
cd $GOHOME/src/github.com/arsonistgopher/junos-openconfig-telemetry-testclient
godep restore
go build
```

## Usage

The script requires some command line inputs as below.

```bash
./junos-openconfig-telemetry-testclient -h
Usage of ./junos-openconfig-telemetry-testclient:
  -certdir string
    	Directory with clientCert.crt, clientKey.crt, CA.crt
  -cid string
    	Set to Client ID (default "1")
  -host string
    	Set host to IP address or FQDN DNS record (default "127.0.0.1")
  -loops int
    	Set loops to desired iterations (default 1)
  -port string
    	Set to Server Port (default "32767")
  -resource string
    	Set resource to resource path (default "/interfaces")
  -smpfreq int
    	Set to sample frequency in milliseconds (default 1000)
  -user string
    	Set to username (default "testuser")
```

With regards to resource strings, there is a sample below and a link in this document to the Juniper gRPC guide for in depth info.

* "/interfaces"
* "/junos/system/linecard/packet/usage"
* "/bgp"
* "/components"
* "/interfaces/interface/subinterfaces"
* "/junos/npu-memory"
* "/junos/system/linecard/npu/memory"
* "/junos/task-memory-information"
* "/junos/system/linecard/firewall/"

Here is how to run the script with TLS. Ensure that you have a "certs" directory and in that director you have a `CA.crt` (for self-signed certs for mutual auth) a `client.crt` and a `client.key`.
The CA cert is the one used to create the certs (self-signed is fine and exactly what I did). Security should be a primary thought here so I've gone the opinated route and mutual-auth. Even if you do not configure it on Junos, you will still need the CA cert and client cert and key. If you do not know how to deal with certs, this document contains instructions on how to build them out.

```bash
./junos-openconfig-telemetry-testclient -host vmx01 -loops 1 -resource /interfaces -smpfreq 1000 -user jet -certdir certs
Enter Password:
-------- gRPC OC Headers from Junos --------
  init-response: [response { subscription_id: 1 } path_list { path: "/interfaces/" sample_frequency: 1000 } ]
  content-type: [application/grpc]
  grpc-accept-encoding: [identity,deflate,gzip]
-------- Transport --------
  Running with mutual TLS
-------- STATS --------
system_id: vmx01
component_id: 65535
sub_component_id: 0
path: sensor_1000_4_1:/interfaces/:/interfaces/:mib2d
sequence_number: 0
timestamp: 1525813035343
sync_response: false
  key: __timestamp__
  uint_value: 1525813035344
  key: __junos_re_stream_creation_timestamp__
  uint_value: 1525813035334
  key: __junos_re_payload_get_timestamp__
  uint_value: 1525813035334
  key: __prefix__
  str_value: /interfaces/interface[name='lsi']/
  </SNIP>
```

If you want to run clear text, simple.

```bash
./gojtemtestoc -host vmx01 -loops 1 -resource /interfaces -smpfreq 1000 -user jet
```

This is an example way to run the script and you will need t chance the input values to suit your environment. Also note, you will not get to "vmx02.corepipe.co.uk".

For the readers amongst you, note that the password field is missing. This is requested from you and the output is masked to prevent shoulder surfer dangers!

## Dealing with TLS and creating certificates.

[TODO]

## Helpful links

I read a reasonable amount of information before making the TLS side of things work properly, both with Go and Junos. Not being one to hide things, here are the links used:

Go TLS mutual auth: [https://github.com/grpc/grpc-go/issues/403](https://github.com/grpc/grpc-go/issues/403)

Go TLS mutual auth: [https://bbengfort.github.io/programmer/2017/03/03/secure-grpc.html](https://bbengfort.github.io/programmer/2017/03/03/secure-grpc.html)

Junos gRPC: [https://www.juniper.net/documentation/en_US/junos/topics/topic-map/grpc-services-telemetry.html](https://www.juniper.net/documentation/en_US/junos/topics/topic-map/grpc-services-telemetry.html)

Junos Cert Auth: [https://www.juniper.net/documentation/en_US/junos/topics/example/certificate-ca-local-manual-loading-cli.html](https://www.juniper.net/documentation/en_US/junos/topics/example/certificate-ca-local-manual-loading-cli.html)

Junos gRPC Guide: [https://www.juniper.net/documentation/en_US/junos/information-products/pathway-pages/junos-telemetry-interface/junos-telemetry-interface.pdf](https://www.juniper.net/documentation/en_US/junos/information-products/pathway-pages/junos-telemetry-interface/junos-telemetry-interface.pdf)

