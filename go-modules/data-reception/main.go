package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf(os.Getenv("MQTT_PORT"), os.Getenv("MQTT_HOST"))
			fmt.Println("Hola mundo 222")
		}
	}
}
