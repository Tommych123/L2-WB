package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	var timeout time.Duration
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "connection timeout")
	flag.Parse() // парсим флаги

	log.SetOutput(os.Stderr) // настройка логера для вывода ошибок в stderr
	log.SetFlags(0)
	if len(flag.Args()) < 2 {
		log.Println("Usage: telnet [--timeout=10s] <host> <port>")
		os.Exit(1)
	}

	host := flag.Arg(0) // получаем параметр host
	port := flag.Arg(1) // получаем параметр port

	address := host + ":" + port // формируем строку для подключения

	conn, err := net.DialTimeout("tcp", address, timeout) // подключаем по tcp
	if err != nil {
		log.Println("Connection failed:", err)
		os.Exit(1)
	}
	defer conn.Close()

	log.Println("Connected to", address)

	var done = make(chan struct{}) // создаем канал для контролирования завершения горутин

	go func() { // горутина читателя(читаем данные от сервера выводим)
		_, err := io.Copy(os.Stdout, conn)
		if err != nil {
			log.Println("Server read error:", err)
		}
		done <- struct{}{} // сигнал завершения в канал
	}()

	go func() {
		_, err := io.Copy(conn, os.Stdin) // горутина писателя(читаем данные от пользователя и отправляем на сервер)
		if err != nil {
			log.Println("Client write error:", err)
		}
		done <- struct{}{} // сигнал завершения в канал
	}()

	<-done // если получен сигнал завершаем все горутины и закрываем канал conn
	log.Println("Connection closed")
}
