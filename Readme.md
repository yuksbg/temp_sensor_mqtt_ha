# Temperature MQTT Sensor

This  application monitors the CPU temperature of a machine and publishes the data to an MQTT broker for seamless integration with Home Assistant and MQTT Autodiscovery.

## Features

- **CPU Temperature Monitoring**: Continuously monitor the CPU temperature using the `lm-sensors` tool.
- **MQTT Integration**: Publish temperature data to an MQTT broker, with attributes such as CPU model and manufacturer.
- **Home Assistant Auto-Discovery**: Automatically discover the sensor in Home Assistant via MQTT Discovery.
- **Systemd Service Integration**: Run the application as a service on Ubuntu, ensuring it starts automatically on boot.

## Prerequisites

Before running this application, ensure you have the following installed:

- **Go**: [Install Go](https://golang.org/doc/install)
- **lm-sensors**: Install `lm-sensors` on your system to allow the application to read the CPU temperature.
- **MQTT Broker**: Set up an MQTT broker (e.g., Mosquitto) that the application can publish to.

### Install `lm-sensors`

On Ubuntu, you can install `lm-sensors` using:

```bash
sudo apt-get update
sudo apt-get install lm-sensors
sudo sensors-detect
```

### Installation
1.	Clone the Repository:
```bash
git clone https://github.com/your-username/temperature-mqtt-sensor.git
cd temperature-mqtt-sensor
```
2. Build the Application:
```bash
go build -o /usr/local/bin/temperature_mqtt main.go
```

### Running the Application
You can run the application directly from the command line:
```shell
Usage: temperature_mqtt [OPTIONS]
Monitors CPU temperature and publishes it to an MQTT broker with Home Assistant autodiscovery.

Options:
  -mqtt_broker   MQTT broker URL (default: tcp://localhost:1883)
  -client_id     MQTT client ID (default: go_temperature_sensor)
  -device_name   Name of the device (default: CPU Sensor)
  -unique_id     Unique ID for the sensor (default: cpu_temperature_sensor)
  -device_id     Device ID (default: cpu_temperature_sensor_device)
  -interval      Interval between temperature readings (default: 10s)
  -debug         Enable debug mode to see detailed logs
  -help          Show this help message

Example command:
  temperature_mqtt -mqtt_broker=tcp://192.168.1.100:1883 -device_name="My CPU Sensor" -interval=30s

In this example, the program will publish CPU temperature to the MQTT broker at tcp://192.168.1.100:1883,
with the device named 'My CPU Sensor', and it will send updates every 30 seconds.
```

## Running as a Systemd Service on Ubuntu

Create a Systemd Service File:
```shell
sudo nano /etc/systemd/system/temperature_mqtt.service
```
Add the following content:
```ini
[Unit]
Description=Temperature MQTT Service
After=network.target

[Service]
ExecStart=/usr/local/bin/temperature_mqtt -mqtt_broker=tcp://localhost:1883 -device_name="CPU Sensor" -interval=10s
Restart=always
User=nobody
Group=nogroup
WorkingDirectory=/usr/local/bin
StandardOutput=journal
StandardError=journal
SyslogIdentifier=temperature_mqtt

[Install]
WantedBy=multi-user.target
```
Reload Systemd and Start the Service:
```shell
sudo systemctl daemon-reload
sudo systemctl start temperature_mqtt
sudo systemctl enable temperature_mqtt
```
Check the Service Status:
```shell
sudo systemctl status temperature_mqtt
```
View Logs:
```shell
journalctl -u temperature_mqtt.service -f
```

### License
This project is licensed under the MIT License

