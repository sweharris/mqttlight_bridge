package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The MQTT client handle and base
var client mqtt.Client
var mqttbase string

// When we receive a message we want to check the topic it came in on
//   It must be for our base (shouldn't ever fail, but)
//   It should be for a light the child process created
//   It should be for .../command or .../bright_command
//   And the payload should look sane.
//
// In theory we could also update the internal state of the light and
// publish the response, but we don't do this; we expect the child to send
// a new status update pretty quickly and that will handle the updates.
// Otherwise we could get a weird loop like "remote sends ON, we reply ON"
// and then five seconds later the child says "we're OFF" so we send an OFF
// and a program like Home Assistant will see the power toggle on/off when,
// in reality, the power hasn't changed.

var MQTTrx mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	topic := strings.Split(msg.Topic(), "/")

	// Is the topic well formed and for our base?
	if len(topic) != 3 {
		return
	}

	if topic[0] != mqttbase {
		return
	}

	// is this one of our lights?
	name := topic[1]

	if _, ok := Lights.Load(name); !ok {
		return
	}

	channel := topic[2]
	payload := string(msg.Payload())

	// Is the channel and payload sane?  If so send a LIGHT command
	// to the child to let it know

	if channel == "command" && (payload == "ON" || payload == "OFF") {
		log.Println("Received MQTT power command for", name, payload)
		send_cmd_to_child("LIGHT#" + name + "#" + payload)
	}

	if channel == "bright_command" {
		bri, _ := strconv.Atoi(payload)
		log.Println("Received MQTT brightness command for", name, bri)
		send_cmd_to_child("LIGHT#" + name + "##" + strconv.Itoa(bri))
	}
}

// Once we've connected, ensure we subscribe to the topics we want
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT server")
	sub(client)
}

// We're set autoreconnect, so this is just for logging purposes
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connect lost: %v\n", err)
}

func publish(name string, channel string, value string) {
	topic := mqttbase + "/" + name + "/" + channel
	// log.Println("Publishing",value,"to",topic)

	// QoS = 0, retain=true
	client.Publish(topic, 0, true, value)
}

func sub(client mqtt.Client) {
	topic := mqttbase + "/#"
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	log.Printf("Subscribed to topic %s\n", topic)
}

func start_mqtt(broker string, port int, mqtt_user string, mqtt_pass string, base string) {
	mqttbase = base

	hostname, _ := os.Hostname()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("mqttlight" + "_" + hostname + "_" + strconv.Itoa(time.Now().Second()))
	opts.SetDefaultPublishHandler(MQTTrx)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetAutoReconnect(true)

	if mqtt_user != "" {
		opts.SetUsername(mqtt_user)
		if mqtt_pass != "" {
			opts.SetPassword(mqtt_pass)
		}
	}

	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
