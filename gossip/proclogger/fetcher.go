package proclogger

import (
	"fmt"
	"os"

	"github.com/streadway/amqp"
)

var rabbitUrl = os.Getenv(
	"AMQP_URL",
)

type AMQP struct {
	con     *amqp.Connection
	channel *amqp.Channel
}

func connect(r *AMQP) {
	if rabbitUrl == "" {
		rabbitUrl = "amqp://guest:guest@localhost:5672/"
		fmt.Println("Using a default rabbit url", rabbitUrl)
	} else {
		fmt.Println("Using rabbit url", rabbitUrl)
	}

	con, err := amqp.Dial(rabbitUrl)
	if con != nil {
		fmt.Println("Successfully Connected to RabbitMQ Instance", rabbitUrl)
	} else {
		fmt.Println("RabbitMQ Instance Is Unabled", rabbitUrl, err)
	}

	channel, err := con.Channel()
	if err != nil {
		fmt.Println("RabbitMQ Instance Is Unabled", err)
	}

	r.con = con
	r.channel = channel
}

func (r AMQP) sendMessage(msg []byte, route string) {
	r.channel.Publish(
		"amq.direct",
		route,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
		},
	)
	fmt.Println("Message has beed sent to queue")
}

var rabbit AMQP

func sendReceipts(msg []byte, route string) {
	if rabbit.channel == nil {
		fmt.Println("Reconect to reddis")
		connect(&rabbit)
	}
	rabbit.sendMessage(msg, route)
}
