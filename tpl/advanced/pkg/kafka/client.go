/**
* @Author: spruce
 * @Date: 2024-03-28 14:24
 * @Desc: kafka
*/

package kafka

import (
	"advanced/pkg/xconfig"
	"errors"

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

// TopicList 全部topic
func (c *Client) TopicList() ([]string, error) {
	if len(c.brokers) == 0 {
		return nil, errors.New("kafka brokers not configured")
	}
	conn, err := kafkaGo.Dial("tcp", c.brokers[0])
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
	if len(c.brokers) == 0 {
		return errors.New("kafka brokers not configured")
	}
	conn, err := kafkaGo.Dial("tcp", c.brokers[0])
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
	if len(c.brokers) == 0 {
		return errors.New("kafka brokers not configured")
	}
	conn, err := kafkaGo.Dial("tcp", c.brokers[0])
	if err != nil {
		return err
	}
	defer func(conn *kafkaGo.Conn) {
		_ = conn.Close()
	}(conn)

	return conn.DeleteTopics(topic)
}
