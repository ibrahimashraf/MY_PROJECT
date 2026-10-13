// Command pilot-tsa is the pilot-only RFC 3161 timestamp responder.
//
// It serves TimeStampResp tokens signed by the pilot TSA key over HTTP POST,
// backed by the production timestamp response builder (unique 128-bit serial
// per token). Trust posture: loopback-only pilot anchor, same as the isolated
// Casdoor IdP — the signer certificate must be provisioned as the TSA root
// (INTEGIN_TSA_ROOTS_FILE) consumed by integin-server. Never expose this outside
// the pilot loopback network: its signatures are only as trustworthy as the
// pilot key ceremony.
package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"integin/internal/timestamp"
)

const maxRequestBytes = 64 << 10

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func loadSignerKey(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found in signer key file")
	}
	switch {
	case strings.Contains(block.Type, "PRIVATE KEY"):
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		switch key.(type) {
		case *ecdsa.PrivateKey, *rsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("signer key must be ECDSA or RSA (production timestamp signer constraint)")
		}
	case block.Type == "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case block.Type == "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
	}
}

func loadSignerCert(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, errors.New("no CERTIFICATE block found in roots file")
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
}

func main() {
	addr := strings.TrimSpace(os.Getenv("INTEGIN_TSA_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:18280"
	}
	keyPath := strings.TrimSpace(os.Getenv("INTEGIN_TSA_SIGNER_KEY_FILE"))
	rootsPath := strings.TrimSpace(os.Getenv("INTEGIN_TSA_ROOTS_FILE"))
	if keyPath == "" || rootsPath == "" {
		log.Fatal("INTEGIN_TSA_SIGNER_KEY_FILE and INTEGIN_TSA_ROOTS_FILE are both required")
	}
	signerKey, err := loadSignerKey(keyPath)
	if err != nil {
		log.Fatalf("signer key: %v", err)
	}
	signerCert, err := loadSignerCert(rootsPath)
	if err != nil {
		log.Fatalf("signer certificate: %v", err)
	}
	if time.Now().Before(signerCert.NotBefore) || time.Now().After(signerCert.NotAfter) {
		log.Fatalf("signer certificate not currently valid (%s to %s)", signerCert.NotBefore.Format(time.RFC3339), signerCert.NotAfter.Format(time.RFC3339))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, maxRequestBytes))
		if err != nil {
			http.Error(writer, "unreadable request", http.StatusBadRequest)
			return
		}
		parsed, err := timestamp.ParseRequest(body)
		if err != nil {
			log.Printf("rejected timestamp request: %v", err)
			http.Error(writer, "invalid TimeStampReq", http.StatusBadRequest)
			return
		}
		serial, err := timestamp.RandomSerial()
		if err != nil {
			log.Printf("serial generation failed: %v", err)
			http.Error(writer, "responder unavailable", http.StatusInternalServerError)
			return
		}
		respDER, err := timestamp.BuildResponse(signerCert, signerKey, parsed.MessageImprint.HashedMessage, time.Now().UTC(), serial)
		if err != nil {
			log.Printf("token minting failed: %v", err)
			http.Error(writer, "responder unavailable", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/timestamp-reply")
		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write(respDER); err != nil {
			log.Printf("response write failed: %v", err)
		}
	})

	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("pilot TSA listening on %s (signer %s)", addr, signerCert.Subject.CommonName)
	log.Fatal(server.ListenAndServe())
}
