# MQTT Light_bridge

A number of years ago I wrote a [Hue Bridge
Emulator](https://github.com/sweharris/huebridge-shell) that would let
you emulate light bulbs in shell script in such a way that these devices
could be controlled by Alexa (and so used in routines and the like).  It
worked well.

But recently Amazon appear to be changing how hue bridges are detected.
The big challenge appears to be it wants the server to listen on port 80.
This is annoying for a number of reasons, including that it needs privilges
to bind to that port, _and_ it may conflict with a web server already on
that port.

Now I also run [Home Assistant](https://www.home-assistant.io/) which has
an inbuilt Hue emulator.  So I figured we can use that functionality, _and_
let you control more things through HA which don't normally have integrations.
A simple example might be [turning your screen on/off](screen_on_off);
have a "bed time" routine that turns your screen off!

We do this by configuring a light in MQTT configuration; this code will
accept commands to turn/on and to (optionally) change brightness.  It will
pass this data down to your program (which could be a shell script) and
update MQTT topics with any feedback from the program.  This means your
program only needs to deal with stdin/stdout and doesn't need to care
about MQTT at all.

## Configuring Home Assistant for MQTT

You need to set up (if you haven't already) the MQTT integration.  Running
an MQTT broker and configuring the integration is beyond the scope of this
README.  I, personally, use `mosquitto` as the broker; it comes with many
Linux distributions and is easy to install.

Now we need to define a light.  This is done in the `mqtt:` section of
`configuration.yaml`.

After manually configuring MQTT instances the way I'm describing here you
will need to tell HA to reload the configuration (e.g from Developer tools),
or else restart HA totally.

There are two kinds

### On/Off only

```
mqtt:
  light:
  - name: "Test Light"
    command_topic: "mqttlight/test/command"
    state_topic: "mqttlight/test/state"
```

With this light it will create a new `light.` entity in HA (probably `light.test_light`).  If you click the ON button it will send an "ON" message to
`mqttlight/test/command`.  The state of the button will be reported back to
HA on `mqttlight/test/state`.

### On/Off with brightness

This emulates a dimmer switch; it has on/off and brightness controls.

```
mqtt:
  light:
  - name: "Test Light"
    command_topic: "mqttlight/test/command"
    state_topic: "mqttlight/test/state"
    brightness_command_topic: "mqttlight/test/bright_command"
    brightness_state_topic: "mqttlight/test/bright_state"
    brightness_scale: 100
```

### Exposing these lights to Alexa (maybe even Google?)

Once the MQTT entries have been configured you will be able to see the
entity IDs in Settings / Devices / Entities.  If you search for the name
you set you'll see it show in the entity list.  We'll need this.

Now in `configuration.yaml` we need to tell HA to expose these lights to
Alexa, via the `emulated_hue` configuration.   I normally don't expose
_every_ device, just the ones I select:

```
emulated_hue:
  listen_port: 80
  expose_by_default: false
  entities: !include emulated_hue.yaml
```

This lets me use a new file `emulated_hue.yaml` to list the devices I want.

```
light.test_light:
  name: "Test light with brightness"
  hidden: false
```

The entity ID is the one we found earlier; the name is the how we want this
light to show in Alexa.

Changing this file requires a full HA restart.
More complex setups are described in the [documentation](https://www.home-assistant.io/integrations/emulated_hue/).

Once you've started it you should be able to confirm the bulb shows up by
talking to the emulated hue endpoint.

e.g.

```
% curl http://your_ha_ip_address:80/api/v2/lights | jq .
{
  "1": {
    "state": {
      "on": true,
      "reachable": true,
      "mode": "homeautomation",
      "bri": 1
    },
    "name": "Test light with brightness",
    "uniqueid": "00:42:af:28:ad:f5:e0:ea-de",
    "manufacturername": "Home Assistant",
    "swversion": "123",
    "type": "Dimmable light",
    "modelid": "HASS123"
  }
}
```

(`jq` is a nice command that make the JSON look pretty).

You can now tell Alexa to discover new devices... and it should(!) work.

More complex `emulated_hue` setups are described in the [HA documentation](https://www.home-assistant.io/integrations/emulated_hue/).

## Communication between bridge and program

The communication patterns are asynchronous.  The bridge can send
commands to the program and the program can send status updates back
to the bridge at any time.  It's a very simple protocol.

### Commands:

There's only one command:

`LIGHT#name#on/off#brightness`  -- Set the light named "name" to the
on/off status and the brightness (0-100).  The name must not contain a #
since that is used as the separator.  If either the on/off or brightness
values are - or missing then that value should be left unchanged.

examples:
```
LIGHT#test##70
LIGHT#test#on#
LIGHT#test#on#70
```

### Status Updates:

The child can send two messages to the bridge.  These are best sent
periodically (e.g. every 5 seconds).

`LIST#name1#name2#name3 ...`  -- This can be used to tell the
bridge of the complete set of lights being controlled.  For example, if
you have an environment where devices may join and leave (a mobile
phone, perhaps?) then this can be used as a way of refreshing the bridge's
knowledge and to stop telling clients about lights that no longer exist.

The bridge uses this list to filter out what messages seen on MQTT topics
should be sent to the child program or not.

`LIGHT#name#on/off#brightness` -- This is the current state of
the light.  When this is sent it will trigger an update of the relevant
MQTT topics.  The brightness may be left blank if it's not relevant.  If
this is a new light then it will be added to the list seen from the `LIST`
command.

e.g.
```
LIST#monitor
LIGHT#monitor#ON#
```

Importantly, the name of the light sent in the MONITOR command will be used
to create the topic name.  e.g.  `LIGHT#monitor#ON#` will cause an ON message
to be sent on topic `mqttlight/monitor/status`.   This _is_ case sensitive;
"monitor" is different to "Monitor"

## Running the program

```
Usage of ./mqttlight_bridge:
  -app string
        Application to run
  -base string
        MQTT Base (default "mqttlight")
  -debug
        Allow CLI entry to child
  -pass string
        MQTT Password
  -port int
        MQTT Port (default 1883)
  -server string
        MQTT server (default "localhost")
  -user string
        MQTT Username
```

The `-app` value is mandatory, the rest are optional.  If your MQTT broker
requires a username and password then these should be specified.

## Examples

Two simple example scripts are provided:
* `small-example` - acts as a simple light called "test" which can be turned
on or off and have the brightness changed

* `screen_on_off` - an On/Off light that uses `xset q` to determine if the
monitor is on or off, and `xset dpms` to change the state.

## Possible gotchas

I've sometimes seen that changing the brightness from the Alexa app may
cause two updates to be sent; the brightness value and _then_ an ON message.
This would cause two commands to be sent to the child
```
  LIGHT#name##brightness
  LIGHT#name#ON#
```

you should be careful that actions taken in response to "ON" without a
brightness value should not change the brightness by mistake.

