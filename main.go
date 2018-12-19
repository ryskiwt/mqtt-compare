package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type dataset struct {
	tx time.Time
	rx time.Time
}

func main() {
	if err := exec(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func exec() error {

	var (
		url      = "tcp://localhost:1883"
		topic    = "topic"
		qos      = byte(0)
		duration = 20 * time.Second
		interval = 100 * time.Millisecond
		total    = int(duration / interval)
	)

	eg, ctx := errgroup.WithContext(context.Background())

	//
	// client
	//

	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
		ClientAuth:         tls.NoClientCert,
	})

	client := mqtt.NewClient(opts)
	defer client.Disconnect(250)

	if err := wrap(client.Connect()); err != nil {
		return err
	}

	//
	// publish
	//

	buf := bytes.NewBuffer(make([]byte, 8+8))
	eg.Go(func() error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for i := 0; i < total; i++ {
			select {
			case <-ctx.Done():
				return nil

			case <-ticker.C:
				tx := time.Now()
				binary.Write(buf, binary.BigEndian, uint64(i))
				binary.Write(buf, binary.BigEndian, uint64(tx.UnixNano()))
				if err := wrap(client.Publish(topic, qos, false, buf.Bytes())); err != nil {
					return err
				}
				buf.Reset()
			}
		}

		return nil
	})

	//
	// subscribe
	//

	writeC := make(chan dataset, 256)
	callback := func(client mqtt.Client, msg mqtt.Message) {
		rd := bytes.NewReader(msg.Payload())
		var i, t uint64
		binary.Read(rd, binary.BigEndian, &i)
		binary.Read(rd, binary.BigEndian, &t)
		tx := time.Unix(0, int64(t))
		writeC <- dataset{tx: tx, rx: time.Now()}

		if int(i) == total-1 {
			close(writeC)
		}
	}
	if err := wrap(client.Subscribe(topic, qos, callback)); err != nil {
		return err
	}
	defer client.Unsubscribe(topic)

	//
	// influx
	//

	eg.Go(func() error {
		client, err := influx.NewHTTPClient(influx.HTTPConfig{
			Addr:     "http://localhost:8086",
			Username: "username",
			Password: "password",
		})
		if err != nil {
			return err
		}
		defer client.Close()

		bpConfig := influx.BatchPointsConfig{
			Database:  "dbname",
			Precision: "ns",
		}
		bps, _ := influx.NewBatchPoints(bpConfig)
		defer client.Write(bps)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil

			case ds, ok := <-writeC:
				if !ok {
					return nil
				}

				fields := map[string]interface{}{
					"tx": ds.tx.UnixNano(),
					"rx": ds.rx.UnixNano(),
				}
				point, _ := influx.NewPoint("measurement", nil, fields, ds.tx)
				bps.AddPoint(point)

			case <-ticker.C:
				if err := client.Write(bps); err != nil {
					return err
				}
				bps, _ = influx.NewBatchPoints(bpConfig)
			}
		}
	})

	//
	// wait
	//

	return eg.Wait()
}

func wrap(token mqtt.Token) error {
	if ok := token.Wait(); !ok {
		return errors.New("not ok")
	}
	return token.Error()
}
