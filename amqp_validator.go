package validation

import amqp "github.com/rabbitmq/amqp091-go"

type AMQPValidator = Validator[*AMQPMessageParser, Validatable]

type AMQPMessageParser struct {
	delivery *amqp.Delivery
}

func (ap *AMQPMessageParser) Parse(v Validatable) error {
	return nil
}
x