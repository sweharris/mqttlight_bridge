package main

// This file handles the state configuration for each bulb.  The main
// routines just track Light_State in the Lights variable.  State changes
// are sent/read from MQTT
//
// func NewLight() -- returns a new Light_State structure that can be
//                    used to be added to the Lights map
//
// func get_or_create_light() -- if a light exists in the map, return
//                               that, otherwise create a new one

import (
	"log"
	"sync"
)

// Data structures for light modelling
//
// We have a power state (ON/OFF) and a brightness (0-255).  That's it!
//
// We don't actually have to save this information; really we only need
// to keep a list of lights the child knows about so can we limit what
// we send to the child (why tell it about light Y if it only handles light
// X), but since we have this information we might as well save it.  Maybe
// useful in the future (eg only send MQTT updates if data changes?)

type Light_State struct {
	On   bool
	Bri  int
	Name string
}

// We will index lights based on their name.  
var Lights sync.Map

func NewLight(name string) Light_State {
	return Light_State{
		On:   false,
		Bri:  0,
		Name: name,
	}
}

func get_or_create_light(name string) Light_State {
	// Find the current light, create a new one if needed
	light, ok := Lights.Load(name)
	if !ok {
		log.Println("Creating new light:", name)
		light = NewLight(name)
		Lights.Store(name,light)
	}
	return light.(Light_State)
}
