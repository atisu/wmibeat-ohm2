package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/atisu/wmibeat-ohm2/config"
)

// wmibeat-ohm2 configuration.
type wmibeatohm2 struct {
	done   chan struct{}
	config config.Config
	client beat.Client
}

// New creates an instance of wmibeat-ohm2.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &wmibeatohm2{
		done:   make(chan struct{}),
		config: c,
	}
	return bt, nil
}

// Run starts wmibeat-ohm2.
func (bt *wmibeatohm2) Run(b *beat.Beat) error {
	logp.Info("wmibeat-ohm2 is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	counter := 1
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    b.Info.Name,
				"counter": counter,
			},
		}
		bt.client.Publish(event)
		logp.Info("Event sent")
		counter++
	}
}

// Stop stops wmibeat-ohm2.
func (bt *wmibeatohm2) Stop() {
	bt.client.Close()
	close(bt.done)
}
