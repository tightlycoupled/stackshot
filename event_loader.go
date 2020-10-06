package stackshot

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/pkg/errors"
)

// eventLoader is an interface type that loads Stack Events from a
// Cloudformation Stack. The storeLastEvent() enables clients to store the
// latest event so that calls to latestEvents() will return all events after
// the latest event.
type eventLoader interface {
	setStackId(*string)
	storeLastEvent() error
	latestEvents(EventConsumer) error
}

type stackEvents struct {
	api               cloudformationiface.CloudFormationAPI
	stackName         *string
	stackId           *string
	lastLoadedEventId *string
}

func (s *stackEvents) setStackId(id *string) {
	s.stackId = id
}

func (s *stackEvents) storeLastEvent() error {
	output, err := s.api.DescribeStackEvents(
		&cloudformation.DescribeStackEventsInput{
			StackName: s.stackId,
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to load stack events")
	}

	s.lastLoadedEventId = output.StackEvents[0].EventId
	return nil
}

func (s *stackEvents) latestEvents(consumer EventConsumer) error {
	newEvents := make([]*cloudformation.StackEvent, 0, 5)

	err := s.api.DescribeStackEventsPages(
		&cloudformation.DescribeStackEventsInput{
			StackName: s.stackId,
		},
		func(output *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {
			events := output.StackEvents
			for _, e := range events {
				if s.lastLoadedEventId != nil && aws.StringValue(e.EventId) == aws.StringValue(s.lastLoadedEventId) {
					return false
				}

				newEvents = append(newEvents, e)
			}

			return !lastPage
		},
	)

	if err != nil {
		return err
	}

	// newEvents contains events in the same order DescribeStackEvents returns
	// them in: reverse chronological order. Therefore, we reverse newEvents to
	// send all events to the eventConsumer in chronological order.
	for i := len(newEvents) - 1; i >= 0; i-- {
		event := newEvents[i]
		err := consumer.Consume(event)
		if err != nil {
			return err
		}
	}

	if len(newEvents) > 0 {
		s.lastLoadedEventId = newEvents[0].EventId
	}

	return nil
}
