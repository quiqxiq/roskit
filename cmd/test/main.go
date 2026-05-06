package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	routeros "github.com/go-routeros/routeros/v3"
)

// Config holds MikroTik connection parameters
type Config struct {
	Address  string
	Username string
	Password string
	Timeout  time.Duration
}

// ConnectError wraps connection errors with human-readable messages
type ConnectError struct {
	Kind    string
	Detail  string
	Wrapped error
}

func (e *ConnectError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Kind, e.Wrapped.Error())
}

func (e *ConnectError) Unwrap() error {
	return e.Wrapped
}

// dial creates a RouterOS API connection with timeout and rich error classification
func dial(cfg Config) (*routeros.Client, error) {
	conn, err := net.DialTimeout("tcp", cfg.Address, cfg.Timeout)
	if err != nil {
		return nil, classifyNetError(err, cfg.Address)
	}

	// Set deadline for the login handshake itself
	_ = conn.SetDeadline(time.Now().Add(cfg.Timeout))

	client, err := routeros.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, &ConnectError{
			Kind:    "HANDSHAKE_FAILED",
			Detail:  "gagal membuat RouterOS API client: " + err.Error(),
			Wrapped: err,
		}
	}

	// Clear deadline after handshake
	_ = conn.SetDeadline(time.Time{})

	if err := client.Login(cfg.Username, cfg.Password); err != nil {
		client.Close()
		return nil, classifyLoginError(err)
	}

	return client, nil
}

// classifyNetError translates low-level net errors ke pesan yang readable
func classifyNetError(err error, address string) *ConnectError {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// Timeout
		if netErr.Timeout() {
			return &ConnectError{
				Kind:   "TIMEOUT",
				Detail: fmt.Sprintf("koneksi ke %s timeout (tidak ada respons dari host)", address),
				Wrapped: err,
			}
		}
		// Connection refused
		if strings.Contains(err.Error(), "connection refused") {
			return &ConnectError{
				Kind:   "CONNECTION_REFUSED",
				Detail: fmt.Sprintf("koneksi ke %s ditolak — pastikan port API RouterOS sudah aktif (default: 8728 / 8729)", address),
				Wrapped: err,
			}
		}
		// Host unreachable / no route
		if strings.Contains(err.Error(), "no route to host") || strings.Contains(err.Error(), "network unreachable") {
			return &ConnectError{
				Kind:   "HOST_UNREACHABLE",
				Detail: fmt.Sprintf("host %s tidak dapat dijangkau — periksa routing/firewall", address),
				Wrapped: err,
			}
		}
		// DNS failure
		if strings.Contains(err.Error(), "no such host") {
			return &ConnectError{
				Kind:   "DNS_FAILED",
				Detail: fmt.Sprintf("hostname tidak ditemukan: %s", address),
				Wrapped: err,
			}
		}
	}

	return &ConnectError{
		Kind:    "CONNECT_FAILED",
		Detail:  fmt.Sprintf("gagal terhubung ke %s: %s", address, err.Error()),
		Wrapped: err,
	}
}

// classifyLoginError translates RouterOS login errors
func classifyLoginError(err error) *ConnectError {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "cannot log in") || strings.Contains(msg, "invalid user") || strings.Contains(msg, "bad password"):
		return &ConnectError{
			Kind:    "AUTH_FAILED",
			Detail:  "autentikasi gagal — username atau password salah",
			Wrapped: err,
		}
	case strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "deadline exceeded"):
		return &ConnectError{
			Kind:    "LOGIN_TIMEOUT",
			Detail:  "timeout saat proses login — RouterOS tidak merespons",
			Wrapped: err,
		}
	case strings.Contains(msg, "EOF"):
		return &ConnectError{
			Kind:    "CONNECTION_DROPPED",
			Detail:  "koneksi terputus saat login — mungkin bukan port RouterOS API, atau TLS diperlukan",
			Wrapped: err,
		}
	default:
		return &ConnectError{
			Kind:    "LOGIN_FAILED",
			Detail:  "login gagal: " + msg,
			Wrapped: err,
		}
	}
}

// runCommand menjalankan satu RouterOS API command dan mengembalikan hasilnya
func runCommand(client *routeros.Client, cmd string, args ...string) (*routeros.Reply, error) {
	sentence := append([]string{cmd}, args...)
	reply, err := client.RunArgs(sentence)
	if err != nil {
		return nil, fmt.Errorf("perintah %q gagal: %w", cmd, err)
	}
	return reply, nil
}

// printReply menampilkan hasil reply RouterOS secara terformat
func printReply(title string, reply *routeros.Reply) {
	fmt.Printf("\n┌─ %s\n", title)
	if len(reply.Re) == 0 {
		fmt.Println("│  (tidak ada data)")
	}
	for i, sentence := range reply.Re {
		if len(reply.Re) > 1 {
			fmt.Printf("│  [%d]\n", i+1)
		}
		for key, val := range sentence.Map {
			fmt.Printf("│  %-30s = %s\n", key, val)
		}
	}
	fmt.Println("└─────────────────────────────────────────")
}

func main() {
	cfg := Config{
		Address:  "103.147.134.201:200",
		Username: "mimon",
		Password: "11",
		Timeout:  10 * time.Second,
	}

	fmt.Printf("Menghubungkan ke MikroTik %s sebagai '%s'...\n", cfg.Address, cfg.Username)

	client, err := dial(cfg)
	if err != nil {
		var connErr *ConnectError
		if errors.As(err, &connErr) {
			fmt.Fprintf(os.Stderr, "\n✗ Koneksi gagal!\n")
			fmt.Fprintf(os.Stderr, "  Jenis   : %s\n", connErr.Kind)
			fmt.Fprintf(os.Stderr, "  Pesan   : %s\n", connErr.Detail)
			if connErr.Wrapped != nil {
				fmt.Fprintf(os.Stderr, "  Raw err : %s\n", connErr.Wrapped.Error())
			}
		} else {
			fmt.Fprintf(os.Stderr, "\n✗ Error tidak diketahui: %v\n", err)
		}
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✓ Terhubung!")
	fmt.Println()

	// ── Test 1: System Identity ─────────────────────────────────────────────
	identReply, err := runCommand(client, "/system/identity/print")
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Gagal mengambil identity: %v\n", err)
	} else {
		printReply("System Identity", identReply)
	}

	// ── Test 2: System Resource ─────────────────────────────────────────────
	resReply, err := runCommand(client, "/system/resource/print")
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Gagal mengambil resource: %v\n", err)
	} else {
		printReply("System Resource", resReply)
	}

	// ── Test 3: RouterOS Version (dari resource) ────────────────────────────
	if resReply != nil && len(resReply.Re) > 0 {
		m := resReply.Re[0].Map
		fmt.Printf("\n📊 Ringkasan:\n")
		fmt.Printf("   Router OS   : %s (%s)\n", m["version"], m["board-name"])
		fmt.Printf("   Platform    : %s\n", m["platform"])
		fmt.Printf("   CPU         : %s @ %s MHz (load: %s%%)\n", m["cpu"], m["cpu-frequency"], m["cpu-load"])

		// Memory dalam bytes → MB
		fmt.Printf("   Memory      : free %s / total %s bytes\n", m["free-memory"], m["total-memory"])
		fmt.Printf("   HDD         : free %s / total %s bytes\n", m["free-hdd-space"], m["total-hdd-space"])
		fmt.Printf("   Uptime      : %s\n", m["uptime"])
	}

	// ── Test 4: IP Address List ─────────────────────────────────────────────
	ipReply, err := runCommand(client, "/ip/address/print")
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Gagal mengambil IP address: %v\n", err)
	} else {
		printReply("IP Address List", ipReply)
	}
}