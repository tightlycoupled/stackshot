package stackshot

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	cfn "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
)

func newTimestamp() *time.Time {
	t := time.Now()
	return &t
}

// eventCollector implements EventConsumer. Consume() saves events in
// eventCollector.events for assertion.
type eventCollector struct {
	events []*cfn.StackEvent
}

func (e *eventCollector) Consume(event *cfn.StackEvent) error {
	if e.events == nil {
		e.events = make([]*cfn.StackEvent, 0, 3)
	}

	e.events = append(e.events, event)

	return nil
}

func TestStackEvents(t *testing.T) {
	stackName := "a-test-stack"

	t.Run(
		"storeLastEvent",
		func(t *testing.T) {
			expEvent := cfn.StackEvent{EventId: aws.String("event-id")}
			api := MockAPI{}
			api.DescribeStackEventsFn = GenDescribeStackEventsFn(&expEvent)

			stackEvents := &stackEvents{
				api:       &api,
				stackName: &stackName,
			}

			err := stackEvents.storeLastEvent()
			if err != nil {
				t.Errorf("Expected storeLastEvent() to succeed. Got error: %s", err)
			}

			if aws.StringValue(stackEvents.lastLoadedEventId) != aws.StringValue(expEvent.EventId) {
				t.Errorf(
					"Expected lastLoadedEventId: %s. Got: %s",
					aws.StringValue(expEvent.EventId),
					aws.StringValue(stackEvents.lastLoadedEventId),
				)
			}

		},
	)

	t.Run(
		"storeLastEvent failure",
		func(t *testing.T) {
			expErr := errors.New("")
			api := MockAPI{}
			api.DescribeStackEventsFn = GenErrorDescribeStackEventsFn(expErr)

			stackEvents := &stackEvents{
				api:       &api,
				stackName: &stackName,
			}

			err := stackEvents.storeLastEvent()
			if err == nil {
				t.Errorf("Expected storeLastEvent() to fail. Got success.")
			}

			if stackEvents.lastLoadedEventId != nil {
				t.Errorf(
					"Expected lastLoadedEventId to be nil. Got: %s",
					aws.StringValue(stackEvents.lastLoadedEventId),
				)
			}

		},
	)

	t.Run(
		"latestEvents returns events in chronological order",
		func(t *testing.T) {
			api := MockAPI{}
			events := []*cfn.StackEvent{
				&cfn.StackEvent{
					EventId:              aws.String("3"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String(stackName),
					ResourceType:         aws.String("AWS::CloudFormation::Stack"),
					ResourceStatus:       aws.String("CREATE_COMPLETE"),
					ResourceStatusReason: aws.String("Building"),
				},
				&cfn.StackEvent{
					EventId:              aws.String("2"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String("LogicalResourceId2"),
					ResourceType:         aws.String("ResourceType2"),
					ResourceStatus:       aws.String("Build"),
					ResourceStatusReason: aws.String("Building"),
				},
				&cfn.StackEvent{
					EventId:              aws.String("1"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String("LogicalResourceId1"),
					ResourceType:         aws.String("ResourceType1"),
					ResourceStatus:       aws.String("Build"),
					ResourceStatusReason: aws.String("Building"),
				},
			}
			api.DescribeStackEventsPagesFn = GenDescribeStackEventsPagesFn(
				&cfn.DescribeStackEventsOutput{StackEvents: events},
				false,
			)
			consumer := &eventCollector{}

			stackEvents := &stackEvents{
				api:       &api,
				stackName: &stackName,
			}

			err := stackEvents.latestEvents(consumer)
			if err != nil {
				t.Errorf("Expected latestEvents to succeed. Got error: %s", err)
			}

			if aws.StringValue(consumer.events[0].EventId) != aws.StringValue(events[2].EventId) {
				t.Errorf(
					"Expected first EventId to be %s. Got: %s",
					aws.StringValue(events[2].EventId),
					aws.StringValue(consumer.events[0].EventId),
				)
			}

			if aws.StringValue(consumer.events[1].EventId) != aws.StringValue(events[1].EventId) {
				t.Errorf(
					"Expected second event EventId to be %s. Got: %s",
					aws.StringValue(events[1].EventId),
					aws.StringValue(consumer.events[1].EventId),
				)
			}

			if aws.StringValue(consumer.events[2].EventId) != aws.StringValue(events[0].EventId) {
				t.Errorf(
					"Expected second event EventId to be %s. Got: %s",
					aws.StringValue(events[0].EventId),
					aws.StringValue(consumer.events[2].EventId),
				)
			}
		},
	)

	t.Run(
		"latestEvents does not return stored event",
		func(t *testing.T) {
			api := MockAPI{}
			events := []*cfn.StackEvent{
				&cfn.StackEvent{
					EventId:              aws.String("3"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String(stackName),
					ResourceType:         aws.String("AWS::CloudFormation::Stack"),
					ResourceStatus:       aws.String("CREATE_COMPLETE"),
					ResourceStatusReason: aws.String("Building"),
				},
				&cfn.StackEvent{
					EventId:              aws.String("2"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String("LogicalResourceId2"),
					ResourceType:         aws.String("ResourceType2"),
					ResourceStatus:       aws.String("Build"),
					ResourceStatusReason: aws.String("Building"),
				},
				&cfn.StackEvent{
					EventId:              aws.String("1"),
					Timestamp:            newTimestamp(),
					LogicalResourceId:    aws.String("LogicalResourceId1"),
					ResourceType:         aws.String("ResourceType1"),
					ResourceStatus:       aws.String("Build"),
					ResourceStatusReason: aws.String("Building"),
				},
			}
			api.DescribeStackEventsPagesFn = GenDescribeStackEventsPagesFn(
				&cfn.DescribeStackEventsOutput{StackEvents: events},
				false,
			)
			consumer := &eventCollector{}

			stackEvents := &stackEvents{
				api:               &api,
				stackName:         &stackName,
				lastLoadedEventId: events[2].EventId,
			}

			err := stackEvents.latestEvents(consumer)
			if err != nil {
				t.Errorf("Expected latestEvents to succeed. Got error: %s", err)
			}

			if len(consumer.events) != 2 {
				t.Errorf("Expected 2 consumed events. Got: %d", len(consumer.events))
			}

			if aws.StringValue(consumer.events[0].EventId) != aws.StringValue(events[1].EventId) {
				t.Errorf(
					"Expected first EventId to be %s. Got: %s",
					aws.StringValue(events[1].EventId),
					aws.StringValue(consumer.events[0].EventId),
				)
			}

			if aws.StringValue(consumer.events[1].EventId) != aws.StringValue(events[0].EventId) {
				t.Errorf(
					"Expected second event EventId to be %s. Got: %s",
					aws.StringValue(events[0].EventId),
					aws.StringValue(consumer.events[1].EventId),
				)
			}
		},
	)
}
