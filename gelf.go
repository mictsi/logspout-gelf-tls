package gelf

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gliderlabs/logspout/router"
	"github.com/Graylog2/go-gelf/gelf"
)

const defaultRetryCount = 10

var (
	hostname string
	retryCount       uint
	econnResetErrStr string
)

func init() {
	hostname, _ = os.Hostname()
	econnResetErrStr = fmt.Sprintf("write: %s", syscall.ECONNRESET.Error())
	router.AdapterFactories.Register(NewGelfAdapter, "gelf")
	setRetryCount()
}

func setRetryCount() {
	if count, err := strconv.Atoi(getopt("RETRY_COUNT", strconv.Itoa(defaultRetryCount))); err != nil {
		retryCount = uint(defaultRetryCount)
	} else {
		retryCount = uint(count)
	}
	debug("Graylog: setting retryCount to:", retryCount)
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func getHostname() string {
	hostname, _ = os.Hostname()
	return hostname

}

// GelfAdapter is an adapter that streams UDP JSON to Graylog
type GelfAdapter struct {
	conn  net.Conn
	route *router.Route
	transport router.AdapterTransport
}

// NewGelfAdapter creates a GelfAdapter with UDP as the default transport.
func NewGelfAdapter(route *router.Route) (router.LogAdapter, error) {
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}

	conn, err := transport.Dial(route.Address, route.Options)
	if err != nil {
		return nil, err
	}

	hostname = getHostname()

	return &GelfAdapter{
		route:     route,
		conn:      conn,
		transport: transport,
	}, nil
}

// Stream implements the router.LogAdapter interface.
func (a *GelfAdapter) Stream(logstream chan *router.Message) {
	for m := range logstream {
		extra, err := extraFields(m)
		if err != nil {
			log.Println("Graylog:", err)
			continue
		}

		messageHostname := hostname
		if messageHostname == "" {
			// Same default as https://github.com/gliderlabs/logspout/blob/master/adapters/syslog/syslog.go
			messageHostname = m.Container.Config.Hostname
		}

		shortMessage := m.Data
		if shortMessage == "" {
			continue
		}
		
		fullMessage := ""
		shortMessageNewLine := strings.Index(shortMessage, "\n")
		if shortMessageNewLine != -1 {
			fullMessage = shortMessage
			shortMessage = shortMessage[:shortMessageNewLine]
		}

		msg := GelfMessage{
			Version:        "1.1",
			Host:           messageHostname,
			ShortMessage:   shortMessage,
			FullMessage:    fullMessage,
			Timestamp:      float64(m.Time.UnixNano()/int64(time.Millisecond)) / 1000.0,
			Level:          gelf.LOG_INFO,
			ContainerId:    m.Container.ID,
			ContainerName:  m.Container.Name[1:], // might be better to use strings.TrimLeft() to remove the first /
			ContainerCmd:   strings.Join(m.Container.Config.Cmd," "),
			ImageId:        m.Container.Image,
			ImageName:      m.Container.Config.Image,
			Created:        m.Container.Created.Format(time.RFC3339Nano),
		}

		if m.Source == "stderr" {
			msg.Level = gelf.LOG_ERR
		}

		js, err := json.Marshal(msg)
		if err != nil {
			log.Println("Graylog:", err)
			continue
		}
		
		if len(extra) > 2 {
			js = append(js[:len(js)-1], ',')
			js = append(js, extra[1:]...)
		}

		js = append(js, 0)

		_, err = a.conn.Write(js)
		if err != nil {
			log.Println("Graylog:", err)
			switch a.conn.(type) {
			case *net.UDPConn:
				continue
			default:
				if err = a.retry(js, err); err != nil {
					log.Panicf("Graylog retry err: %+v", err)
					return
				}
			}
		}
	}
}

func (a *GelfAdapter) retry(buf []byte, err error) error {
	if opError, ok := err.(*net.OpError); ok {
		if (opError.Temporary() && opError.Err.Error() != econnResetErrStr) || opError.Timeout() {
			retryErr := a.retryTemporary(buf)
			if retryErr == nil {
				return nil
			}
		}
	}
	if reconnErr := a.reconnect(); reconnErr != nil {
		return reconnErr
	}
	if _, err = a.conn.Write(buf); err != nil {
		log.Println("Graylog: reconnect failed")
		return err
	}
	log.Println("Graylog: reconnect successful")
	return nil
}

func (a *GelfAdapter) retryTemporary(buf []byte) error {
	log.Printf("Graylog: retrying tcp up to %v times\n", retryCount)
	err := retryExp(func() error {
		_, err := a.conn.Write(buf)
		if err == nil {
			log.Println("Graylog: retry successful")
			return nil
		}

		return err
	}, retryCount)

	if err != nil {
		log.Println("Graylog: retry failed")
		return err
	}

	return nil
}

func (a *GelfAdapter) reconnect() error {
	log.Printf("Graylog: reconnecting up to %v times\n", retryCount)
	err := retryExp(func() error {
		conn, err := a.transport.Dial(a.route.Address, a.route.Options)
		if err != nil {
			return err
		}
		a.conn = conn
		return nil
	}, retryCount)

	if err != nil {
		return err
	}
	return nil
}

func retryExp(fun func() error, tries uint) error {
	try := uint(0)
	for {
		err := fun()
		if err == nil {
			return nil
		}

		try++
		if try > tries {
			return err
		}

		time.Sleep((1 << try) * 10 * time.Millisecond)
	}
}

type GelfMessage struct {
	Version      string  `json:"version"`
	Host         string  `json:"host,omitempty"`
	ShortMessage string  `json:"short_message"`
	FullMessage  string  `json:"full_message,omitempty"`
	Timestamp    float64 `json:"timestamp,omitempty"`
	Level        int32   `json:"level,omitempty"`

	ImageId        string `json:"_image_id,omitempty"`
	ImageName      string `json:"_image_name,omitempty"`
	ContainerId    string `json:"_container_id,omitempty"`
	ContainerName  string `json:"_container_name,omitempty"`
	ContainerCmd   string `json:"_command,omitempty"`
	Created        string `json:"_created,omitempty"`
}

func extraFields(m *router.Message) (json.RawMessage, error) {

	extra := map[string]interface{}{
	}
	for name, label := range m.Container.Config.Labels {
		if len(name) > 5 && strings.ToLower(name[0:5]) == "gelf_" {
			extra[name[4:]] = label
		}
	}
	swarmnode := m.Container.Node
	if swarmnode != nil {
		extra["_swarm_node"] = swarmnode.Name
	}

	rawExtra, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}
	return rawExtra, nil
}
