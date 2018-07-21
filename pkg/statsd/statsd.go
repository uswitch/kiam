package statsd

import (
	"fmt"
	"time"

	"gopkg.in/alexcesaro/statsd.v2"
)

var Client *statsd.Client

func New(address string, prefix string, interval time.Duration) error {
	var options []statsd.Option
	if address == "" {
		options = []statsd.Option{statsd.Mute(true)}
	} else {
		options = []statsd.Option{
			statsd.Address(address),
			statsd.Prefix(prefix),
			statsd.FlushPeriod(interval),
		}
	}

	sd, err := statsd.New(options...)

	if err != nil {
		return fmt.Errorf("statsd.New: %v", err)
	}

	Client = sd
	return nil
}
