package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mmcloughlin/geohash"
	"github.com/satori/go.uuid"
	. "github.com/smartfog/fogflow/common/config"
)

func main() {
	// new random uid
	u1, err := uuid.NewV4()
	if err != nil {
		ERROR.Println(err)
		return
	}
	rid := u1.String()

	cfgFile := flag.String("f", "config.json", "A configuration file")
	id := flag.String("i", rid, "its ID in the current site")
	port := flag.String("p", "0", "the listening port")

	flag.Parse()
	config, err := LoadConfig(*cfgFile)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s\n", err.Error()))
		ERROR.Println("please specify the configuration file, for example, \r\n\t./broker -f config.json")
		os.Exit(-1)
	}

	if (*port) != "0" {
		config.Broker.Port, _ = strconv.Atoi(*port)
	}

	geohashID := geohash.EncodeWithPrecision(config.PLocation.Latitude, config.PLocation.Longitude, config.Precision)
	myID := "Broker." + geohashID + "." + (*id)

	// check if IoT Discovery is ready
	for {
		resp, err := http.Get(config.GetDiscoveryURL() + "/status")
		if err != nil {
			ERROR.Println(err)
		} else {
			INFO.Println(resp.StatusCode)
		}

		if (err == nil) && (resp.StatusCode == 200) {
			break
		} else {
			time.Sleep(2 * time.Second)
		}
	}

	// initialize broker
	broker := ThinBroker{id: myID}
	broker.Start(&config)

	// start the REST API server
	restapi := &RestApiSrv{}
	restapi.Start(&config, &broker)

	// start a timer to do something periodically
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for _ = range ticker.C {
			broker.OnTimer()
		}
	}()

	// wait for Control+C to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c

	// stop the timer
	ticker.Stop()

	// stop the REST API server
	restapi.Stop()

	// stop the broker
	broker.Stop()
}
