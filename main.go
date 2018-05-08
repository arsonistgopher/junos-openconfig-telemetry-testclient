/*
MIT License

Copyright (c) 2018 David Gee

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	auth_pb "github.com/arsonistgopher/junos-openconfig-telemetry-testclient/proto/auth"
	oct_pb "github.com/arsonistgopher/junos-openconfig-telemetry-testclient/telemetry"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func handleOneTelemetryPkt(ocData *oct_pb.OpenConfigData) *bytes.Buffer {

	var buffer bytes.Buffer

	buffer.Write([]byte(fmt.Sprintln("-------- STATS --------")))
	buffer.Write([]byte(fmt.Sprintf("system_id: %s\n", ocData.SystemId)))
	buffer.Write([]byte(fmt.Sprintf("component_id: %d\n", ocData.ComponentId)))
	buffer.Write([]byte(fmt.Sprintf("sub_component_id: %d\n", ocData.SubComponentId)))
	buffer.Write([]byte(fmt.Sprintf("path: %s\n", ocData.Path)))
	buffer.Write([]byte(fmt.Sprintf("sequence_number: %d\n", ocData.SequenceNumber)))
	buffer.Write([]byte(fmt.Sprintf("timestamp: %d\n", ocData.Timestamp)))
	buffer.Write([]byte(fmt.Sprintf("sync_response: %t\n", ocData.SyncResponse)))
	if ocData.SyncResponse {
		buffer.Write([]byte(fmt.Sprintf("Received sync_response\n")))
	}

	del := ocData.GetDelete()
	for _, d := range del {
		buffer.Write([]byte(fmt.Sprintf("Delete: %s\n", d.GetPath())))
	}

	prefixSeen := false
	for _, kv := range ocData.Kv {

		buffer.Write([]byte(fmt.Sprintf("  key: %s\n", kv.Key)))
		switch value := kv.Value.(type) {
		case *oct_pb.KeyValue_DoubleValue:
			buffer.Write([]byte(fmt.Sprintf("  double_value: %f\n", value.DoubleValue)))
		case *oct_pb.KeyValue_IntValue:
			buffer.Write([]byte(fmt.Sprintf("  int_value: %d\n", value.IntValue)))
		case *oct_pb.KeyValue_UintValue:
			buffer.Write([]byte(fmt.Sprintf("  uint_value: %d\n", value.UintValue)))
		case *oct_pb.KeyValue_SintValue:
			buffer.Write([]byte(fmt.Sprintf("  sint_value: %d\n", value.SintValue)))
		case *oct_pb.KeyValue_BoolValue:
			buffer.Write([]byte(fmt.Sprintf("  bool_value: %t\n", value.BoolValue)))
		case *oct_pb.KeyValue_StrValue:
			buffer.Write([]byte(fmt.Sprintf("  str_value: %s\n", value.StrValue)))
		case *oct_pb.KeyValue_BytesValue:
			buffer.Write([]byte(fmt.Sprintf("  bytes_value: %s\n", value.BytesValue)))
		default:
			buffer.Write([]byte(fmt.Sprintf("  default: %v\n", value)))
		}

		if kv.Key == "__prefix__" {
			prefixSeen = true
		} else if !strings.HasPrefix(kv.Key, "__") {
			if !prefixSeen && !strings.HasPrefix(kv.Key, "/") {
				buffer.Write([]byte(fmt.Sprintf("Missing prefix for sensor: %s\n", ocData.Path)))
			}
		}
	}
	return &buffer
}

func main() {

	// gRPC options
	var opts []grpc.DialOption

	// Create instance of subscription request proto
	var subReqM oct_pb.SubscriptionRequest

	// Create instance of Path from proto
	var pathM oct_pb.Path

	// Parse flags
	var loops = flag.Int("loops", 1, "Set loops to desired iterations")
	var host = flag.String("host", "127.0.0.1", "Set host to IP address or FQDN DNS record")
	var resource = flag.String("resource", "/interfaces", "Set resource to resource path")
	var user = flag.String("user", "testuser", "Set to username")
	var cid = flag.String("cid", "1", "Set to Client ID")
	var port = flag.String("port", "32767", "Set to Server Port")
	var sampleFreq = flag.Int("smpfreq", 1000, "Set to sample frequency in milliseconds")
	var certDir = flag.String("certdir", "", "Directory with clientCert.crt, clientKey.crt, CA.crt")
	flag.Parse()

	// Are we going to run with TLS?
	runningWithTLS := false
	if *certDir != "" {
		runningWithTLS = true
	}

	// Grab password
	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("Err: %v", err)
	}
	password := string(bytePassword)

	// gRPC dial needs a connection string
	hostandport := fmt.Sprintf("%s:%s", *host, *port)

	// Get context
	ctx := context.Background()

	// If we're running with TLS
	if runningWithTLS {

		// Grab x509 cert/key for client
		cert, err := tls.LoadX509KeyPair(fmt.Sprintf("%s/client.crt", *certDir), fmt.Sprintf("%s/client.key", *certDir))

		if err != nil {
			log.Fatalf("Could not load certFile: %v", err)
		}
		// Create certPool for CA
		certPool := x509.NewCertPool()

		// Get CA
		ca, err := ioutil.ReadFile(fmt.Sprintf("%s/CA.crt", *certDir))
		if err != nil {
			log.Fatalf("Could not read ca certificate: %s", err)
		}

		// Append CA cert to pool
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			log.Fatalf("Failed to append client certs")
		}

		// build creds
		creds := credentials.NewTLS(&tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{cert},
			ServerName:   *host,
		})

		if err != nil {
			log.Fatalf("Could not load clientCert: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else { // Else we're not running with TLS
		opts = append(opts, grpc.WithInsecure())
	}

	//Set grpc initial window size. This value seems to work from another project
	opts = append(opts, grpc.WithInitialWindowSize(524288))
	conn, err := grpc.Dial(hostandport, opts...)

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	// lazy close
	defer conn.Close()

	// Get client
	c := oct_pb.NewOpenConfigTelemetryClient(conn)

	pathM.Path = *resource
	pathM.SampleFrequency = uint32(*sampleFreq)

	subReqM.PathList = append(subReqM.PathList, &pathM)

	l := auth_pb.NewLoginClient(conn)
	dat, err := l.LoginCheck(context.Background(), &auth_pb.LoginRequest{UserName: *user, Password: password, ClientId: *cid})

	if err != nil {
		log.Fatalf("Could not login: %v", err)
	}
	if dat.Result == false {
		log.Fatalf("LoginCheck failed\n")
	}

	stream, err := c.TelemetrySubscribe(ctx, &subReqM)

	if err != nil {
		log.Fatalf("Could not send RPC: %v\n", err)
	}

	hdr, errh := stream.Header()
	if errh != nil {
		log.Fatalf("Failed to get header for stream: %v", errh)
	}

	// We make the switch here from logging to fmt'ing due to application logging vs data output

	fmt.Println("\n-------- gRPC OC Headers from Junos --------")
	for k, v := range hdr {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Println("-------- Transport --------")
	if runningWithTLS {
		fmt.Println("  Running with mutual TLS")
	} else {
		fmt.Println("  Running clear text")
	}

	// Run for MAX_LOOPS
	for i := 0; i < *loops; i++ {
		ocData, err := stream.Recv()

		// EOF detected
		if err == io.EOF {
			fmt.Println("End (EOF) detected")
			break
		}

		// Err
		if err != nil {
			logrus.Fatalf("%v.TelemetrySubscribe(_) = _, %v", conn, err)
		}

		// Go and process a lovely ocData pkt
		str := handleOneTelemetryPkt(ocData)
		// Dump it out to stdout
		fmt.Print(str)
	}
}
