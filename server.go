package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"encoding/json"

	"github.com/armon/go-socks5"
)

type params struct {
	// User      string `json:"user"`
	// Password  string `json:"password"`
	Port      int    `json:"port"`
	LocalAddr string `json:"local_addr"`
}

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Usage: socks5-server <config.json>")
	}
	arg := os.Args[1]

	file, err := os.Open(arg)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)

	localAddrs := getLocalAddrs()
	for _, addr := range localAddrs {
		log.Printf("Local address: %s\n", addr)
	}

	var cfgs []params = make([]params, 0)
	err = decoder.Decode(&cfgs)
	if err != nil {
		log.Fatal(err)
	}

	i := 0

	for _, cfg := range cfgs {
		cfg.LocalAddr = localAddrs[i]
		i++

		log.Printf("Port: %d -> Local address: %s\n", cfg.Port, cfg.LocalAddr)
		if i >= len(localAddrs) {
			i = 0
		}
		go runServer(cfg)
	}

	for {
		time.Sleep(60 * time.Second)
	}

}

func getLocalAddrs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}

	localAddrs := make([]string, 0)
	for _, addr := range addrs {
		if !strings.HasPrefix(addr.String(), "172.") {
			continue
		}
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsGlobalUnicast() {
			localAddrs = append(localAddrs, ipnet.IP.String())
		}
	}

	return localAddrs
}

func runServer(cfg params) {

	//Initialize socks5 config
	socks5conf := &socks5.Config{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	// if cfg.User+cfg.Password != "" {
	//     creds := socks5.StaticCredentials{
	//         os.Getenv("PROXY_USER"): os.Getenv("PROXY_PASSWORD"),
	//     }
	//     cator := socks5.UserPassAuthenticator{Credentials: creds}
	//     socks5conf.AuthMethods = []socks5.Authenticator{cator}
	// }

	socks5conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout: 10 * time.Second,
			LocalAddr: &net.TCPAddr{
				IP:   net.ParseIP(cfg.LocalAddr),
				Port: 0,
			},
		}
		return dialer.Dial(network, addr)
	}

	server, err := socks5.New(socks5conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Start listening proxy service on port %d\n", cfg.Port)
	if err := server.ListenAndServe("tcp", fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatal(err)
	}
}
