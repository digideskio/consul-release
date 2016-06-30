package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Flags struct {
	ConsulURL      string
	Port           string
	CACertFilePath string
	CertFilePath   string
	KeyFilePath    string
	RequireSSL     bool
}

func main() {
	flags := Flags{}
	flag.StringVar(&flags.ConsulURL, "consul-url", "", "fully qualified url of the consul server")
	flag.StringVar(&flags.Port, "port", "", "port of the proxy server")
	flag.StringVar(&flags.CACertFilePath, "cacert", "", "path to the consul ca certificate")
	flag.StringVar(&flags.CertFilePath, "cert", "", "path to the consul agent certificate")
	flag.StringVar(&flags.KeyFilePath, "key", "", "path to the consul agent key")
	flag.BoolVar(&flags.RequireSSL, "require-ssl", false, "require TLS communication to the remote consul server")
	flag.Parse()

	proxyUrl, err := url.Parse(flags.ConsulURL)
	if err != nil {
		fail(fmt.Sprintf("failed to parse consul-url %s", err.Error()))
	}

	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
	if flags.RequireSSL {
		proxy.Transport = &http.Transport{
			TLSAgentConfig: buildTLSConfig(flags.CACertFilePath, flags.CertFilePath, flags.KeyFilePath),
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		proxy.ServeHTTP(w, r)
	})

	if err := http.ListenAndServe(":"+flags.Port, nil); err != nil {
		fail(err)
	}
}

func fail(message interface{}) {
	fmt.Fprint(os.Stderr, message)
	os.Exit(1)
}

func buildTLSConfig(caCertFilePath, certFilePath, keyFilePath string) *tls.Config {
	tlsCert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
	if err != nil {
		fail(err)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		InsecureSkipVerify: false,
		AgentAuth:          tls.RequireAndVerifyAgentCert,
	}

	certBytes, err := ioutil.ReadFile(caCertFilePath)
	if err != nil {
		fail(err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		fail("cacert is not a PEM encoded file")
	}

	tlsConfig.RootCAs = caCertPool
	tlsConfig.AgentCAs = caCertPool

	return tlsConfig
}
