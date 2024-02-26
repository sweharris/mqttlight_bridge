package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// This handles the LIGHT status update sent from the child process
//
//	name on/off brightness

func set_light_status(status []string) {
	name := status[0]

	power := false
	pow_str := "OFF"
	if strings.ToUpper(status[1]) == "ON" {
		power = true
		pow_str = "ON"
	}

	bri, err := strconv.Atoi(status[2])
	if err != nil {
		bri = 0
	}

	light := get_or_create_light(name)

	if light.On != power {
		log.Println("Got light update for", name, "setting Power", power)
		light.On = power
	}
	publish(name, "state", pow_str)

	if light.Bri != bri {
		log.Println("Got light update for", name, "setting Brightness", bri)
		light.Bri = bri
	}
	publish(name, "bright_state", strconv.Itoa(bri))

	Lights.Store(name, light)
}

// This takes a list of lights as reported by the LIST status update
// and ensures that this list matches the Lights map.  Basically we
// create any missing entries with defaults, then delete ones that are
// no longer relevant

func update_light_list(lights []string) {
	// We'll make the list of lights into a map so we can quickly compare
	list := make(map[string]bool)

	// First, make sure every list in the LIST is in the array
	for _, name := range lights {
		list[name] = true
		Lights.Store(name, get_or_create_light(name))
	}

	// Now delete any light in the array that's not in the list
	Lights.Range(func(name interface{}, val interface{}) bool {
		_, ok := list[name.(string)]
		if !ok {
			log.Println("Deleting", name.(string))
			Lights.Delete(name)
		}
		return true
	})
}

// Input to this function will be output from the child.  The line
// either needs to be
//    LIST#name1#name2 ...
// or
//    LIGHT#name#on/off#brightness
//
// Note the name should be valid MQTT topic sequence

func on_read(s string) {
	input := strings.Split(s, "#")
	cmd := strings.ToUpper(input[0])
	if cmd == "LIGHT" {
		set_light_status(input[1:])
	} else if cmd == "LIST" {
		update_light_list(input[1:])
	} else {
		log.Println("Ignoring", s)
	}
}

func main() {
	mqtt_server := flag.String("server", "localhost", "MQTT server")
	mqtt_port := flag.Int("port", 1883, "MQTT Port")
	mqtt_user := flag.String("user", "", "MQTT Username")
	mqtt_pass := flag.String("pass", "", "MQTT Password")
	mqtt_base := flag.String("base", "mqttlight", "MQTT Base")
	app := flag.String("app", "", "Application to run")
	debug := flag.Bool("debug", false, "Allow CLI entry to child")

	flag.Parse()

	if *app == "" {
		fmt.Fprintln(os.Stderr, "No application specified; aborted")
		os.Exit(-1)
	}

	// Start the MQTT listener
	start_mqtt(*mqtt_server, *mqtt_port, *mqtt_user, *mqtt_pass, *mqtt_base)

	// Set the child process running to update the light status
	start_child(*app)
	defer child.Wait()

	if !*debug {
		read_child(on_read)
	} else {
		// Run it as a go-process if we want CLI debugging
		go read_child(on_read)

		// Now read from stdin for debugging.  This lets us send LIGHT#
		// commands direct to the child to emulate what might have been
		// received from MQTT
		stdin := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Enter text: ")
			text, _ := stdin.ReadString('\n')
			send_cmd_to_child(text)
		}
	}
}
