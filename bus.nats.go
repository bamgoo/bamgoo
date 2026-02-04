package bamgoo

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	. "github.com/bamgoo/bamgoo/base"
	"github.com/nats-io/nats.go"
)

var (
	errNatsInvalidConnection = errors.New("invalid nats connection")
	errNatsAlreadyRunning    = errors.New("nats bus is already running")
	errNatsNotRunning        = errors.New("nats bus is not running")
)

type (
	natsBusDriver struct{}

	natsBusConnection struct {
		mutex   sync.RWMutex
		running bool

		instance *BusInstance
		setting  natsBusSetting

		client *nats.Conn

		subjects map[string]struct{}
		subs     []*nats.Subscription
	}

	natsBusSetting struct {
		URL        string
		Username   string
		Password   string
		QueueGroup string
	}
)

func (driver *natsBusDriver) Connect(inst *BusInstance) (Connection, error) {
	setting := natsBusSetting{URL: nats.DefaultURL}

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

	return &natsBusConnection{
		instance: inst,
		setting:  setting,
		subjects: make(map[string]struct{}, 0),
		subs:     make([]*nats.Subscription, 0),
	}, nil
}

// Register registers a service subject.
func (c *natsBusConnection) Register(subject string) error {
	c.mutex.Lock()
	c.subjects[subject] = struct{}{}
	c.mutex.Unlock()
	return nil
}

func (c *natsBusConnection) Open() error {
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

func (c *natsBusConnection) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

func (c *natsBusConnection) Start() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return errNatsAlreadyRunning
	}
	if c.client == nil {
		return errNatsInvalidConnection
	}

	// Subscribe to all registered subjects
	for subject := range c.subjects {
		callSub := "call." + subject
		queueSub := "queue." + subject
		eventSub := "event." + subject

		// call - request/reply with queue group
		sub, err := c.client.QueueSubscribe(callSub, c.queueGroup(callSub), func(msg *nats.Msg) {
			resp, _ := c.handleRequest(msg.Data)
			if msg.Reply != "" {
				if resp != nil {
					_ = msg.Respond(resp)
				} else {
					_ = msg.Respond([]byte("{}"))
				}
			}
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)

		// queue - async with queue group
		sub, err = c.client.QueueSubscribe(queueSub, c.queueGroup(queueSub), func(msg *nats.Msg) {
			c.handleRequest(msg.Data)
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)

		// event - broadcast to all
		sub, err = c.client.Subscribe(eventSub, func(msg *nats.Msg) {
			c.handleRequest(msg.Data)
		})
		if err != nil {
			return err
		}
		c.subs = append(c.subs, sub)
	}

	c.running = true
	return nil
}

func (c *natsBusConnection) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return errNatsNotRunning
	}

	for _, sub := range c.subs {
		_ = sub.Unsubscribe()
	}
	c.subs = nil
	c.running = false
	return nil
}

// Request sends a synchronous request and waits for reply.
func (c *natsBusConnection) Request(_ *Meta, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if c.client == nil {
		return nil, errNatsInvalidConnection
	}

	msg, err := c.client.Request(subject, data, timeout)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

// Publish broadcasts an event to all subscribers.
func (c *natsBusConnection) Publish(_ *Meta, subject string, data []byte) error {
	if c.client == nil {
		return errNatsInvalidConnection
	}
	return c.client.Publish(subject, data)
}

// Enqueue publishes to a queue (one of the subscribers will receive).
func (c *natsBusConnection) Enqueue(_ *Meta, subject string, data []byte) error {
	if c.client == nil {
		return errNatsInvalidConnection
	}
	return c.client.Publish(subject, data)
}

func (c *natsBusConnection) queueGroup(subject string) string {
	if c.setting.QueueGroup != "" {
		return c.setting.QueueGroup
	}
	return subject
}

func (c *natsBusConnection) handleRequest(data []byte) ([]byte, error) {
	name, payload, err := c.decodeRequest(data)
	if err != nil {
		return nil, err
	}

	body, res, _ := core.invokeLocal(nil, name, payload)
	return c.encodeResponse(body, res)
}

func (c *natsBusConnection) decodeRequest(data []byte) (string, Map, error) {
	var env struct {
		Name    string `json:"name"`
		Payload Map    `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return "", nil, err
	}
	if env.Payload == nil {
		env.Payload = Map{}
	}
	return env.Name, env.Payload, nil
}

func (c *natsBusConnection) encodeResponse(data Map, res Res) ([]byte, error) {
	if res == nil {
		res = OK
	}
	env := struct {
		Code    int    `json:"code"`
		State   string `json:"state"`
		Message string `json:"message"`
		Data    Map    `json:"data"`
	}{
		Code:    res.Code(),
		State:   res.State(),
		Message: res.Error(),
		Data:    data,
	}
	return json.Marshal(env)
}
