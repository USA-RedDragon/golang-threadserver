package main

import (
	"flag"
)

var verbose = flag.Bool("verbose", false, "Whether to display verbose logs")

func main() {
	var redisHost = flag.String("redis", "localhost:6379", "The hostname of redis")
	var listen = flag.String("listen", "127.0.0.1", "The IP to listen on")
	var port = flag.Int("port", 2323, "The Port to listen on")

	flag.Parse()

	start(*listen, *port, *redisHost)
}
