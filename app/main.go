package main

import (
	"context"
	"flag"
	"github.com/IrvinIrvin/forum/app/forum"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	root := flag.String("root", "..", "project directory containing templates, css and database")
	database := flag.String("db", "", "SQLite database path (default: <root>/database/forum.db)")
	address := flag.String("listen", "127.0.0.1:8080", "listen address")
	insecure := flag.Bool("insecure-local", false, "allow HTTP cookies for local development only")
	migrate := flag.Bool("migrate", false, "back up and migrate the database, then exit")
	admin := flag.Int64("admin-user-id", 0, "verified existing administrator ID; requires -migrate")
	flag.Parse()
	if *insecure {
		host, _, parseErr := net.SplitHostPort(*address)
		ip := net.ParseIP(host)
		if parseErr != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			log.Fatal("-insecure-local requires a loopback listener")
		}
	}
	if *database == "" {
		*database = filepath.Join(*root, "database", "forum.db")
	}
	if *admin != 0 && !*migrate {
		log.Fatal("-admin-user-id requires -migrate")
	}
	store, err := forum.Open(*database)
	if err != nil {
		log.Fatal(err)
	}
	defer store.DB.Close()
	if *migrate {
		if err = store.Migrate(*database, *admin); err != nil {
			log.Fatal(err)
		}
		log.Print("migration completed; legacy sessions invalidated")
		return
	}
	if err = store.Initialize(); err != nil {
		log.Fatal(err)
	}
	app, err := forum.New(store, filepath.Join(*root, "templates"), filepath.Join(*root, "css"), !*insecure, log.Default())
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: *address, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	log.Print("listening on ", *address)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
