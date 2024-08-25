package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

// HomeAssistantConfig represents the JSON payload for the Home Assistant configuration
type HomeAssistantConfig struct {
	Name                string              `json:"name"`
	StateTopic          string              `json:"state_topic"`
	UniqueID            string              `json:"unique_id"`
	DeviceClass         string              `json:"device_class,omitempty"`
	UnitOfMeasurement   string              `json:"unit_of_measurement,omitempty"`
	ValueTemplate       string              `json:"value_template,omitempty"`
	Device              HomeAssistantDevice `json:"device"`
	JsonAttributesTopic string              `json:"json_attributes_topic,omitempty"`
}

// HomeAssistantDevice represents the device information in the Home Assistant configuration
type HomeAssistantDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Model        string   `json:"model"`
	Manufacturer string   `json:"manufacturer"`
}

// SensorState represents the JSON payload for the sensor state
type SensorState struct {
	Temperature     float64 `json:"temperature"`
	CPUModel        string  `json:"cpu_model,omitempty"`
	CPUManufacturer string  `json:"cpu_manufacturer,omitempty"`
}

// Function to check if the 'sensors' command is available
func checkSensorsCommand() error {
	_, err := exec.LookPath("sensors")
	if err != nil {
		return fmt.Errorf("'sensors' command not found. Please install 'lm-sensors' package.")
	}
	return nil
}

// Function to get the CPU model and manufacturer from /proc/cpuinfo
func getCPUInfo() (model string, manufacturer string, err error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", "", fmt.Errorf("error reading /proc/cpuinfo: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				model = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "vendor_id") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				manufacturer = strings.TrimSpace(parts[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error scanning /proc/cpuinfo: %v", err)
	}

	if model == "" || manufacturer == "" {
		return "", "", fmt.Errorf("could not find CPU information in /proc/cpuinfo")
	}

	return model, manufacturer, nil
}

// Function to get the temperature from the sensors command
func getTemperature() (float64, error) {
	cmd := exec.Command("sensors")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	outputStr := string(output)
	re := regexp.MustCompile(`(?m)Package id 0:\s+\+([0-9.]+)°C`)
	match := re.FindStringSubmatch(outputStr)
	if len(match) < 2 {
		return 0, fmt.Errorf("temperature not found in sensors output")
	}

	temp, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, err
	}

	return temp, nil
}

// Function to publish the temperature and CPU attributes to the MQTT broker
func publishTemperature(client MQTT.Client, temp float64, stateTopic, cpuModel, cpuManufacturer string) {
	state := SensorState{
		Temperature:     temp,
		CPUModel:        cpuModel,
		CPUManufacturer: cpuManufacturer,
	}
	payload, err := json.Marshal(state)
	if err != nil {
		log.WithError(err).Error("Error encoding JSON")
		return
	}

	if *debug {
		log.WithField("payload", string(payload)).Debug("Publishing temperature payload with attributes")
	}

	token := client.Publish(stateTopic, 0, false, payload)
	token.Wait()
	log.Info("Temperature and attributes sent to MQTT")
}

// Function to publish the Home Assistant discovery configuration
func publishConfig(client MQTT.Client, model, manufacturer, stateTopic, configTopic string) {
	config := HomeAssistantConfig{
		Name:                sensorName,
		StateTopic:          stateTopic,
		UniqueID:            *uniqueID,
		DeviceClass:         "temperature",
		UnitOfMeasurement:   "°C",
		ValueTemplate:       "{{ value_json.temperature }}",
		JsonAttributesTopic: stateTopic,
		Device: HomeAssistantDevice{
			Identifiers:  []string{*deviceID},
			Name:         *deviceName,
			Model:        model,
			Manufacturer: manufacturer,
		},
	}

	payload, err := json.Marshal(config)
	if err != nil {
		log.WithError(err).Error("Error encoding JSON")
		return
	}

	if *debug {
		log.WithField("payload", string(payload)).Debug("Publishing config payload")
	}

	token := client.Publish(configTopic, 0, false, payload)
	token.Wait()
	log.Info("Home Assistant autodiscovery config sent")
}

var (
	mqttBroker  = flag.String("mqtt_broker", "tcp://localhost:1883", "MQTT broker URL")
	clientID    = flag.String("client_id", "go_temperature_sensor", "MQTT client ID")
	deviceName  = flag.String("device_name", "CPU Sensor", "Name of the device")
	uniqueID    = flag.String("unique_id", "cpu_temperature_sensor", "Unique ID for the sensor")
	deviceID    = flag.String("device_id", "cpu_temperature_sensor_device", "Device ID")
	interval    = flag.Duration("interval", 10*time.Second, "Interval between temperature readings")
	debug       = flag.Bool("debug", false, "Enable debug mode to see detailed logs")
	showHelp    = flag.Bool("help", false, "Show help information")
	sensorName  string // Generated based on device_name
	stateTopic  string // Generated based on device_name
	configTopic string // Generated based on device_name
)

func main() {
	flag.Parse()

	// Initialize logrus logger
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Generate sensorName, stateTopic, and configTopic based on deviceName
	sensorName = fmt.Sprintf("%s Temperature", *deviceName)
	stateTopic = fmt.Sprintf("homeassistant/sensor/%s/state", strings.ReplaceAll(strings.ToLower(*deviceName), " ", "_"))
	configTopic = fmt.Sprintf("homeassistant/sensor/%s/config", strings.ReplaceAll(strings.ToLower(*deviceName), " ", "_"))

	// Show custom help if the help flag is set
	if *showHelp {
		fmt.Println("Usage: temperature_mqtt [OPTIONS]")
		fmt.Println("Monitors CPU temperature and publishes it to an MQTT broker with Home Assistant autodiscovery.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -mqtt_broker   MQTT broker URL (default: tcp://localhost:1883)")
		fmt.Println("  -client_id     MQTT client ID (default: go_temperature_sensor)")
		fmt.Println("  -device_name   Name of the device (default: CPU Sensor)")
		fmt.Println("  -unique_id     Unique ID for the sensor (default: cpu_temperature_sensor)")
		fmt.Println("  -device_id     Device ID (default: cpu_temperature_sensor_device)")
		fmt.Println("  -interval      Interval between temperature readings (default: 10s)")
		fmt.Println("  -debug         Enable debug mode to see detailed logs")
		fmt.Println("  -help          Show this help message")
		fmt.Println()
		fmt.Println("Example command:")
		fmt.Println("  temperature_mqtt -mqtt_broker=tcp://192.168.1.100:1883 -device_name=\"My CPU Sensor\" -interval=30s")
		fmt.Println()
		fmt.Println("In this example, the program will publish CPU temperature to the MQTT broker at tcp://192.168.1.100:1883,")
		fmt.Println("with the device named 'My CPU Sensor', and it will send updates every 30 seconds.")
		return
	}

	// Check if the 'sensors' command is available
	if err := checkSensorsCommand(); err != nil {
		log.WithError(err).Fatal("Sensors command not found")
	}

	// Get CPU model and manufacturer
	model, manufacturer, err := getCPUInfo()
	if err != nil {
		log.WithError(err).Fatal("Error getting CPU information")
	}

	// MQTT client options
	opts := MQTT.NewClientOptions().
		AddBroker(*mqttBroker).
		SetClientID(*clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(10 * time.Second).
		SetOnConnectHandler(func(c MQTT.Client) {
			log.Info("Connected to MQTT broker")
			// Publish the config message after reconnecting
			publishConfig(c, model, manufacturer, stateTopic, configTopic)
		}).
		SetConnectionLostHandler(func(c MQTT.Client, err error) {
			log.WithError(err).Warn("Connection to MQTT broker lost. Attempting to reconnect...")
		})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.WithError(token.Error()).Fatal("Failed to connect to MQTT broker")
	}

	// Publish the config message on the initial connection
	publishConfig(client, model, manufacturer, stateTopic, configTopic)

	for {
		temp, err := getTemperature()
		if err != nil {
			log.WithError(err).Error("Error getting temperature")
			continue
		}
		publishTemperature(client, temp, stateTopic, model, manufacturer)
		time.Sleep(*interval)
	}
}
