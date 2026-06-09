package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/retry"
	"ride-sharing/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange       = "trip"
	DeadLetterExchange = "dlx"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create the channel: %v", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		Channel: ch,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		// cleanup if setup fails
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

type MessageHandler func(context.Context, amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {

	err := r.Channel.Qos(1, 0, false)

	if err != nil {
		return fmt.Errorf("failed to set Qos: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {
				log.Printf("Recieved a message: %s", msg.Body)

				cfg := retry.DefaultConfig()
				err := retry.WithBackoff(ctx, cfg, func() error {
					return handler(ctx, d)
				})
				if err != nil {
					log.Printf("Message processing failed after %d retries for messagge ID: %s, err: %v", cfg.MaxRetries, d.MessageId, err)

					// Addd failurer context
					headers := amqp.Table{}
					if d.Headers != nil {
						headers = d.Headers
					}

					headers["x-death-reason"] = err.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-original-routing-key"] = d.RoutingKey
					headers["x-retry-countt"] = cfg.MaxRetries
					d.Headers = headers

					// Reject without requeue
					_ = d.Reject(false)
					return err
				}

				// Ack after msg succeedds
				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
				}

				return nil
			}); err != nil {
				log.Printf("Error processing message: %v", err)
			}

		}
	}()

	return nil
}

func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMsg,
	}

	return tracing.TracedPublisher(ctx, TripExchange, routingKey, msg, r.publish)

}

func (r *RabbitMQ) publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return r.Channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		msg,
	)
}

func (r *RabbitMQ) setupDeadLetterExchange() error {
	err := r.Channel.ExchangeDeclare(DeadLetterExchange, "topic", true, false, false, false, nil)

	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", TripExchange, err)
	}

	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind tthe queue to the dead letter excahnge
	err = r.Channel.QueueBind(
		q.Name,
		"#",
		DeadLetterExchange,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// First setup the DLQ exchange and queue
	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

	err := r.Channel.ExchangeDeclare(TripExchange, "topic", true, false, false, false, nil)

	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", TripExchange, err)
	}

	if err := r.declareAndBindQueue(FindAvailableDriversQueue, []string{contracts.TripEventCreated, contracts.TripEventDriverNotInterested}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(DriverCmdTripRequestQueue, []string{contracts.DriverCmdTripRequest}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(DriverTripResponseQueue, []string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(NotifyDriverNoDriversFoundQueue, []string{contracts.TripEventNoDriversFound}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(NotifyDriverAssignQueue, []string{contracts.TripEventDriverAssigned}, TripExchange); err != nil {
		return err
	}
	if err := r.declareAndBindQueue(PaymentTripResponseQueue, []string{contracts.PaymentCmdCreateSession}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(NotifyPaymentSessionCreatedQueue, []string{contracts.PaymentEventSessionCreated}, TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(NotifyPaymentSuccessQueue, []string{contracts.PaymentEventSuccess}, TripExchange); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, messaggeTypes []string, exchange string) error {
	// Add dead letter config
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}

	q, err := r.Channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	)

	if err != nil {
		log.Fatal(err)
	}

	for _, msg := range messaggeTypes {

		if err := r.Channel.QueueBind(
			q.Name,
			msg,
			exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}

	if r.Channel != nil {
		r.Channel.Close()
	}
}
