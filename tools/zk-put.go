package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-zookeeper/zk"
)

func main() {
	var (
		path    = flag.String("path", "", "ZooKeeper path")
		servers = flag.String("servers", "127.0.0.1:2181", "ZooKeeper servers (comma-separated)")
	)
	flag.Parse()

	if *path == "" {
		fmt.Fprintf(os.Stderr, "Error: -path is required\n")
		os.Exit(1)
	}

	// Read config from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Connect to ZooKeeper
	conn, _, err := zk.Connect([]string{*servers}, 10*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to ZooKeeper: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Ensure parent path exists
	parent := ""
	parts := splitPath(*path)
	for i := 0; i < len(parts)-1; i++ {
		if parent == "" {
			parent = "/" + parts[i]
		} else {
			parent = parent + "/" + parts[i]
		}
		if parent != "" {
			exists, _, err := conn.Exists(parent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking path %s: %v\n", parent, err)
				os.Exit(1)
			}
			if !exists {
				_, err := conn.Create(parent, []byte{}, 0, zk.WorldACL(zk.PermAll))
				if err != nil && err != zk.ErrNodeExists {
					fmt.Fprintf(os.Stderr, "Error creating parent path %s: %v\n", parent, err)
					os.Exit(1)
				}
			}
		}
	}

	// Check if node exists
	exists, _, err := conn.Exists(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking path: %v\n", err)
		os.Exit(1)
	}

	if exists {
		// Update existing node
		_, err = conn.Set(*path, data, -1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting node: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Create new node
		_, err = conn.Create(*path, data, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating node: %v\n", err)
			os.Exit(1)
		}
	}
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}
	// Remove leading slash
	if path[0] == '/' {
		path = path[1:]
	}
	parts := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

