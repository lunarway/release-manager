package amqpextra

import (
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/log"
)

// Config configures a worker.
type Config struct {
	Connection          ConnectionConfig
	Exchange            string
	Queue               string
	RoutingKey          string
	Prefetch            int
	ReconnectionTimeout time.Duration
	Logger              *log.Logger
}

// ConnectionConfig configures an AMQP connection.
type ConnectionConfig struct {
	Host        string
	User        string
	Password    string
	VirtualHost string
	Port        int
}

// Raw returns a raw connection string used to dial. Notice that this will
// reveal the password if the result is logged.
func (c *ConnectionConfig) Raw() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.VirtualHost)
}

// String returns a masked connection string safe for logging.
func (c *ConnectionConfig) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", c.User, "***", c.Host, c.Port, c.VirtualHost)
}
