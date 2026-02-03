package bus_nats

import (
	"errors"
	"sync"
	"time"

	"github.com/bamgoo/bamgoo"
	"github.com/nats-io/nats.go"
)

var (
	errInvalidConnection = errors.New("invalid nats connection")
	errAlreadyRunning    = errors.New("nats bus is already running")
	errNotRunning        = errors.New("nats bus is not running")
)

type (
	natsDriver struct{}

	natsConnect struct {
		mutex   sync.RWMutex
		running bool

		instance *bamgoo.BusInstance
		setting  natsSetting
		client   *nats.Conn

		calls  map[string]bamgoo.Handler
		queues map[string]bamgoo.Handler
		events map[string]bamgoo.Handler

		subs []*nats.Subscription
	}

	natsSetting struct {
		URL        string
		Username   string
		Password   string
		QueueGroup string
	}
)

func (driver *natsDriver) Connect(inst *bamgoo.BusInstance) (bamgoo.Connect, error) {
	setting := natsSetting{URL: nats.DefaultURL}

	if v, ok := inst.Config.Setting["url"].(string); ok {
		setting.URL = v
	}
	if v, ok := inst.Config.Setting["server"].(string); ok {
		setting.URL = v
	}
	if v, ok := inst.Config.Setting["user"].(string); ok {
		setting.Username = v
	}
	if v, ok := inst.Config.Setting["username"].(string); ok {
		setting.Username = v
	}
	if v, ok := inst.Config.Setting["pass"].(string); ok {
		setting.Password = v
	}
	if v, ok := inst.Config.Setting["password"].(string); ok {
		setting.Password = v
	}
	if v, ok := inst.Config.Setting["group"].(string); ok {
		setting.QueueGroup = v
	}

	return &natsConnect{
		instance: inst,
		setting:  setting,
		calls:    make(map[string]bamgoo.Handler, 0),
		queues:   make(map[string]bamgoo.Handler, 0),
		events:   make(map[string]bamgoo.Handler, 0),
		subs:     make([]*nats.Subscription, 0),
	}, nil
}

func (c *natsConnect) Open() error {
	opts := []nats.Option{}
	if c.setting.Username != "" && c.setting.Password != "" {
		opts = append(opts, nats.UserInfo(c.setting.Username, c.setting.Password))
	}

	client, err := nats.Connect(c.setting.URL, opts...)
	if err != nil {
		return err
	}

	c.client = client
	return nil
}

func (c *natsConnect) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

func (c *natsConnect) Start() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return errAlreadyRunning
	}
	if c.client == nil {
		return errInvalidConnection
	}

	for subject, handler := range c.calls {
		sub, err := c.client.QueueSubscribe(subject, c.queueGroup(subject), func(msg *nats.Msg) {
			if handler == nil {
				return
			}
			resp, err := handler(msg.Data)
			if msg.Reply != "" {
				if err == nil && resp != nil {
					_ = msg.Respond(resp)
				} else {
					_ = msg.Respond([]byte{})
				}
			}
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)
	}

	for subject, handler := range c.queues {
		sub, err := c.client.QueueSubscribe(subject, c.queueGroup(subject), func(msg *nats.Msg) {
			if handler == nil {
				return
			}
			_, _ = handler(msg.Data)
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)
	}

	for subject, handler := range c.events {
		sub, err := c.client.Subscribe(subject, func(msg *nats.Msg) {
			if handler == nil {
				return
			}
			_, _ = handler(msg.Data)
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)
	}

	c.running = true
	return nil
}

func (c *natsConnect) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return errNotRunning
	}

	for _, sub := range c.subs {
		_ = sub.Unsubscribe()
	}
	c.subs = nil
	c.running = false
	return nil
}

func (c *natsConnect) Register(subject string, handler bamgoo.Handler) error {
	callSub := "call." + subject
	queueSub := "queue." + subject
	eventSub := "event." + subject

	c.mutex.Lock()
	c.calls[callSub] = handler
	c.queues[queueSub] = handler
	c.events[eventSub] = handler
	c.mutex.Unlock()
	return nil
}

func (c *natsConnect) Request(subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if c.client == nil {
		return nil, errInvalidConnection
	}

	msg, err := c.client.Request(subject, data, timeout)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

func (c *natsConnect) Publish(subject string, data []byte) error {
	if c.client == nil {
		return errInvalidConnection
	}
	return c.client.Publish(subject, data)
}

func (c *natsConnect) Queue(subject string, data []byte) error {
	if c.client == nil {
		return errInvalidConnection
	}
	return c.client.Publish(subject, data)
}

func (c *natsConnect) queueGroup(subject string) string {
	if c.setting.QueueGroup != "" {
		return c.setting.QueueGroup
	}
	return subject
}
