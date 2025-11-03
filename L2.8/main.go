package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beevik/ntp"
)

func main() {
	currentTime, err := ntp.Time("pool.ntp.org") // подключаемся к ntp серверу и получаем текущее время
	log.SetOutput(os.Stderr)                     // настройка логера для вывода ошибок в stderr
	log.SetFlags(0)
	if err != nil {
		log.Println("Error of ntp server: ", err)
		os.Exit(1) // завершаем программу с ошибкой
	}
	fmt.Println(currentTime.Format(time.RFC1123)) // вывод curtime в удобночитаемом формате
}
