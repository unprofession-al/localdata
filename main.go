package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
)

type server struct {
	out    string
	listen string
}

func NewServer(out, listen string) *server {
	return &server{
		out:    out,
		listen: listen,
	}
}

func (s *server) OutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", s.out)
}

func (s *server) Run() {
	http.HandleFunc("/", s.OutHandler)
	go func() {
		if err := http.ListenAndServe(s.listen, nil); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()
}

func main() {
	configFile := os.Getenv("CONFIG_FILE")
	listener := os.Getenv("LISTENER")
	s := NewServer("SERVER: getting ready\n", listener)
	s.Run()

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(fmt.Sprintf("could not read file '%s', error was %s", configFile, err))
	}

	configs, err := NewVolumeManagerConfigs(yamlFile)
	if err != nil {
		panic(fmt.Sprintf("could not parse file '%s', error was %s", configFile, err))
	}

	for _, vc := range configs {
		v, err := NewVolumeManager(vc)
		if err != nil {
			s.out += fmt.Sprintf("VOLUME '%s': Could not create VolumeManager: %s\n", vc.Name, err)
		} else {
			err = v.Ensure()
			s.out += fmt.Sprintf("VOLUME '%s': Setting up...\n", vc.Name)

			if err != nil {
				s.out += fmt.Sprintf("VOLUME '%s': Could ensure filesystem: %s\n", vc.Name, err)
			} else {
				s.out += fmt.Sprintf("VOLUME '%s': ready\n", vc.Name)
			}
		}
	}

	s.out += fmt.Sprintf("SERVER: ready\n")
	waitForCtrlC()
}

func waitForCtrlC() {
	var end_waiter sync.WaitGroup
	end_waiter.Add(1)
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 1)
	signal.Notify(signal_channel, os.Interrupt)
	go func() {
		<-signal_channel
		end_waiter.Done()
	}()
	end_waiter.Wait()
}
