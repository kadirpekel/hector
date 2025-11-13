package config

import (
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

type ZookeeperProvider struct {
	conn      *zk.Conn
	path      string
	endpoints []string
}

func NewZookeeperProvider(endpoints []string, path string) (*ZookeeperProvider, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("zookeeper endpoints are required")
	}

	if path == "" {
		return nil, fmt.Errorf("zookeeper path is required")
	}

	conn, _, err := zk.Connect(endpoints, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	return &ZookeeperProvider{
		conn:      conn,
		path:      path,
		endpoints: endpoints,
	}, nil
}

func (p *ZookeeperProvider) ReadBytes() ([]byte, error) {

	data, _, err := p.conn.Get(p.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read from zookeeper path %s: %w", p.path, err)
	}

	return data, nil
}

func (p *ZookeeperProvider) Watch(callback func(event interface{}, err error)) error {
	for {

		data, _, eventCh, err := p.conn.GetW(p.path)
		if err != nil {
			callback(nil, fmt.Errorf("failed to watch zookeeper path %s: %w", p.path, err))
			continue
		}

		event := <-eventCh

		switch event.Type {
		case zk.EventNodeDataChanged:

			callback(data, nil)
		case zk.EventNodeDeleted:

			callback(nil, fmt.Errorf("zookeeper node %s was deleted", p.path))
			return nil
		case zk.EventNotWatching:

			callback(nil, fmt.Errorf("zookeeper watch lost for path %s", p.path))
			return nil
		}
	}
}

func (p *ZookeeperProvider) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}
