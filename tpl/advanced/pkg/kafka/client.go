/**
* @Author: spruce
 * @Date: 2024-03-28 14:24
 * @Desc: kafka
*/

package kafka

import (
	"advanced/pkg/xconfig"
	"errors"
	"fmt"

	kafkaGo "github.com/segmentio/kafka-go"
)

type (
	Client struct {
		brokers []string
	}
)

func NewClient(conf *xconfig.Conf) *Client {
	return &Client{
		brokers: conf.Kafka.Brokers,
	}
}

// dialBroker 尝试连接 brokers 列表中的可用节点进行运维操作
func (c *Client) dialBroker() (*kafkaGo.Conn, error) {
	if len(c.brokers) == 0 {
		return nil, errors.New("kafka brokers not configured")
	}
	var lastErr error
	for _, b := range c.brokers {
		conn, err := kafkaGo.Dial("tcp", b)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("failed to dial any kafka broker in %v: %w", c.brokers, lastErr)
}

// TopicList 全部topic
func (c *Client) TopicList() ([]string, error) {
	conn, err := c.dialBroker()
	if err != nil {
		return nil, err
	}
	defer func(conn *kafkaGo.Conn) {
		_ = conn.Close()
	}(conn)

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]struct{})
	for _, p := range partitions {
		topicMap[p.Topic] = struct{}{}
	}

	topics := make([]string, 0, len(topicMap))
	for k := range topicMap {
		topics = append(topics, k)
	}
	return topics, nil
}

// CreateTopic 创建topic
func (c *Client) CreateTopic(topic string, partition int) error {
	conn, err := c.dialBroker()
	if err != nil {
		return err
	}
	defer func(conn *kafkaGo.Conn) {
		_ = conn.Close()
	}(conn)

	topicConfigs := []kafkaGo.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partition,
			ReplicationFactor: 1,
		},
	}

	return conn.CreateTopics(topicConfigs...)
}

// DelTopic 删除 topic
func (c *Client) DelTopic(topic string) error {
	conn, err := c.dialBroker()
	if err != nil {
		return err
	}
	defer func(conn *kafkaGo.Conn) {
		_ = conn.Close()
	}(conn)

	return conn.DeleteTopics(topic)
}
