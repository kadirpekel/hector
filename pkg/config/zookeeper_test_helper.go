package config

import (
	"time"

	"github.com/go-zookeeper/zk"
)

// setupZookeeperNode creates a ZooKeeper node with the given path and data
// This is a helper function for tests that need to create ZooKeeper nodes
func setupZookeeperNode(endpoints []string, path string, data []byte) error {
	conn, _, err := zk.Connect(endpoints, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Split path into parts
	parts := []string{""}
	currentPath := ""
	for _, part := range splitZookeeperPath(path) {
		if part == "" {
			continue
		}
		currentPath += "/" + part
		parts = append(parts, currentPath)
	}

	// Create intermediate paths
	for i := 1; i < len(parts)-1; i++ {
		exists, _, _ := conn.Exists(parts[i])
		if !exists {
			_, err = conn.Create(parts[i], []byte{}, 0, zk.WorldACL(zk.PermAll))
			if err != nil && err != zk.ErrNodeExists {
				return err
			}
		}
	}

	// Create or update final node
	finalPath := parts[len(parts)-1]
	exists, _, _ := conn.Exists(finalPath)
	if exists {
		_, err = conn.Set(finalPath, data, -1)
	} else {
		_, err = conn.Create(finalPath, data, 0, zk.WorldACL(zk.PermAll))
	}
	return err
}

// deleteZookeeperNode deletes a ZooKeeper node
func deleteZookeeperNode(endpoints []string, path string) error {
	conn, _, err := zk.Connect(endpoints, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	exists, _, err := conn.Exists(path)
	if err != nil {
		return err
	}
	if exists {
		return conn.Delete(path, -1)
	}
	return nil
}

// splitZookeeperPath splits a ZooKeeper path into parts
func splitZookeeperPath(path string) []string {
	parts := []string{}
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
