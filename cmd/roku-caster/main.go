package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/TenacityMind/RokuVideoCaster/internal/api"
	"github.com/TenacityMind/RokuVideoCaster/internal/config"
	"github.com/TenacityMind/RokuVideoCaster/internal/roku"
	"github.com/TenacityMind/RokuVideoCaster/web"
)

func main() {
	portFlag := flag.Int("port", 8080, "Web dashboard port")
	ipFlag := flag.String("ip", "", "Roku TV IP address")
	pwdFlag := flag.String("password", "", "Roku developer password")
	installCmd := flag.Bool("install", false, "Install/update Roku TV channel directly from command line")
	noBrowser := flag.Bool("no-browser", false, "Don't open browser automatically")
	flag.Parse()

	// Load persistent configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("[WARN] Failed to load config.json: %v", err)
	}

	if *ipFlag != "" || *pwdFlag != "" || *portFlag != 8080 {
		_ = cfg.Update(*ipFlag, *pwdFlag, *portFlag, true)
	}

	savedIP, savedPwd, port, _ := cfg.Get()
	if *portFlag != 8080 {
		port = *portFlag
	}

	// Direct CLI installer mode
	if *installCmd {
		if savedIP == "" || savedPwd == "" {
			log.Fatalf("[ERROR] Both -ip and -password flags are required to install channel")
		}
		fmt.Printf("📦 Sideloading Roku SceneGraph channel to %s...\n", savedIP)
		res, err := roku.SideloadChannel(context.Background(), savedIP, savedPwd)
		if err != nil {
			log.Fatalf("[ERROR] Installation failed: %v\n", err)
		}
		fmt.Printf("✅ %s\n", res.Message)
		return
	}

	// Initialize API server with embedded web assets
	server := api.NewServer(cfg, http.FS(web.FS))
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: server.Handler(),
	}

	go func() {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf("  📺 Roku Video Caster V2.0 PRO\n")
		fmt.Printf("  👉 Dashboard: http://localhost:%d\n", port)
		if savedIP != "" {
			fmt.Printf("  🎯 Target TV: %s\n", savedIP)
		} else {
			fmt.Printf("  🎯 Target TV: Not configured (open dashboard to set)\n")
		}
		fmt.Printf("=======================================================\n\n")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Open browser automatically unless disabled
	if !*noBrowser {
		time.Sleep(300 * time.Millisecond)
		openBrowser(fmt.Sprintf("http://localhost:%d", port))
	}

	// Graceful shutdown on Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("\nShutting down Roku Video Caster...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	_ = exec.Command(cmd, args...).Start()
}
