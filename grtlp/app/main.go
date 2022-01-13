package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func getDate(dateStrings []string) (date time.Time, err error) {
	// layout variable is needed for parsing of time string
	layout := "2006-01-02T15:04:05.000Z"
	stringDate := dateStrings[0] + "T" + dateStrings[1] + ".000Z"

	date, err = time.Parse(layout, stringDate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not parse date and time")
		return time.Time{}, err
	}

	return date, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// structure of data to be sent to influxdb through MQTT
	type Data struct {
		Date time.Time `json:"date"`
		// Step  int       `json:"step"`
		// Value float32   `json:"value"`
		Data map[int]float32 `json:"data"`
	}

	// Setup the MQTT client with the options specified
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("mqtt://localhost:1883"))
	opts.SetClientID("grtlp_mqtt_client")
	opts.SetUsername("balena")
	opts.SetPassword("public")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Load rtl_power variables
	lowerFreq := os.Getenv("LOWER_FREQ")
	upperFreq := os.Getenv("UPPER_FREQ")
	binSize := os.Getenv("BIN_SIZE")
	interval := os.Getenv("INTERVAL")
	tunerGain := os.Getenv("TUNER_GAIN")

	// combine frequency range and bin size for the command
	frequency := lowerFreq + ":" + upperFreq + ":" + binSize

	// execute the rtl_power comand with the environment variables
	cmd := exec.Command("rtl_power",
		"-f", frequency,
		"-g", tunerGain,
		"-i", interval,
		"-")

	// Get the command output per line
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating stdoutpipe")
		log.Fatal(err)
		return
	}

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans cmdReader in a line-by-line fashion
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		// Parse the data needed (date & time, real upperband, lowband, samples, bands, dbm)
		// time, Hz low, Hz high, Hz step, samples, dbm, dbm, ...
		for scanner.Scan() {
			line := scanner.Text()
			// TODO: debug: stdout output
			// Clean up input, and separate strings into an array
			outputLine := strings.ReplaceAll(line, " ", "")
			dataArr := strings.Split(outputLine, ",")

			// get the date info from the stdout line
			date, err := getDate(dataArr[:2])
			if err != nil {
				panic(err)
			}

			lowerBand, err := strconv.ParseFloat(dataArr[2], 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not parse lowerBand")
				log.Fatal(err)
			}

			step, err := strconv.ParseFloat(dataArr[4], 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not parse step")
				log.Fatal(err)
			}

			// make map of values, frequency_band:dBm
			dataMap := make(map[int]float32)
			for i, data := range dataArr[6:] {
				// get the data info and put it into a map of
				result := (lowerBand + (step * float64(i)))

				dataFloat, err := strconv.ParseFloat(data, 64)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Could not parse float data")
					log.Fatal(err)
				}

				// skip NaN values
				if math.IsNaN(dataFloat) {
					continue
				}

				dataMap[int(result)] = float32(dataFloat)
			}

			data := Data{Date: date, Data: dataMap}
			d, err := json.Marshal(data)
			fmt.Println(string(d))
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not Marshal data")
				log.Fatal(err)
				return
			}

			token := client.Publish("sensors", 0, false, d)
			token.Wait()
			time.Sleep(time.Second)
		}

		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// Start the command and check for errors
	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not start command")
		log.Fatal(err)
		return
	}

	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not wait for output")
		log.Fatal(err)
		return
	}

}
