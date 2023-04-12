package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	err := serve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	return 0
}

func serve() error {
	Logger.Printf("Running")
	fs := http.FileServer(http.Dir("public"))
	err := http.ListenAndServe(":8080", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Add("Cache-Control", "no-cache")
		fs.ServeHTTP(resp, req)
	}))
	if err != nil {
		return err
	}
	return nil
}
