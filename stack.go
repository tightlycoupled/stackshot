package stackshot

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/pkg/errors"
)

var stackDoesNotExistErrorFmt string = "%s does not exist"

// maxWaitAttempts is max number of attempts to query Cloudformation to see if
// a stack has finished updating.
const maxWaitAttempts = 720

// waitDelay is time to delay before querying Cloudformation to see if a stack
// has finished updating.
const waitDelay = 5 * time.Second

// stackDoneStatuses is a map of Cloudformation StackStatuses that represent no
// further changes are running on a stack. The keys are StackStatus and the
// values are bools denoting a successful or failed Sync() call.
var stackDoneStatuses = map[string]bool{
	"CREATE_COMPLETE":          true,
	"UPDATE_COMPLETE":          true,
	"CREATE_FAILED":            false,
	"UPDATE_FAILED":            false,
	"UPDATE_ROLLBACK_COMPLETE": false,
	"UPDATE_ROLLBACK_FAILED":   false,
	"DELETE_COMPLETE":          false,
	"DELETE_FAILED":            false,
	"ROLLBACK_FAILED":          false,
	"ROLLBACK_COMPLETE":        false,
}

// EventConsumer is an interface used by Stack.SyncAndPollEvents() to consume
// events polled from an updating Cloudformation Stack.
type EventConsumer interface {
	Consume(*cloudformation.StackEvent) error
}

type EventConsumerFunc func(*cloudformation.StackEvent) error

func (e EventConsumerFunc) Consume(event *cloudformation.StackEvent) error {
	return e(event)
}

type waiter interface {
	wait()
}

type waiterFunc func()

func (w waiterFunc) wait() {
	w()
}

func sleepWaiter() {
	time.Sleep(waitDelay)
}

// EventPrinter implements EventConsumer interface to print
// cloudformation.StackEvent to stdout.
func EventPrinter(event *cloudformation.StackEvent) error {
	fmt.Printf(
		"%s %s(%s) %s %s\n",
		event.Timestamp,
		aws.StringValue(event.LogicalResourceId),
		aws.StringValue(event.ResourceType),
		aws.StringValue(event.ResourceStatus),
		aws.StringValue(event.ResourceStatusReason),
	)
	return nil
}

// LoadStack allocates a new Stack used to synchronize a StackConfig's
// configuration with a new or existing Cloudformation Stack.
func LoadStack(api cloudformationiface.CloudFormationAPI, config *StackConfig) (*Stack, error) {
	stack := &Stack{
		api:          api,
		config:       config,
		waitAttempts: maxWaitAttempts,
		waiter:       waiterFunc(sleepWaiter),
		eventLoader: &stackEvents{
			api:       api,
			stackName: aws.String(config.Name),
		},
	}

	err := stack.load()
	if err == nil {
		err = stack.storeLastEvent()
	}

	if err != nil {
		return nil, err
	}
	return stack, nil
}

// Stack synchronizes a StackConfig with the corresponding Cloudformation
// Stack. Stack simplifies the Cloudformation's API by issuing CreateStack or
// UpdateStack depending on whether the stack name represented at
// StackConfig.Name exists in Cloudformation.
//
// To synchronize a StackConfig with the Cloudformation Stack, call Sync().
//
// If you need to wait for the Cloudformation Stack to complete creating or
// updating, call SyncAndPollEvents(). SyncAndPollEvents() takes an argument
// that implements the EventConsumer interface in which events are passed.
type Stack struct {
	eventLoader

	cloudStack *cloudformation.Stack
	api        cloudformationiface.CloudFormationAPI
	config     *StackConfig

	waiter       waiter
	waitAttempts int
}

func (s *Stack) load() error {
	input := cloudformation.DescribeStacksInput{}
	if s.cloudStack == nil {
		input.StackName = aws.String(s.config.Name)
	} else {
		input.StackName = s.cloudStack.StackId
	}

	out, err := s.api.DescribeStacks(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if stackDoesNotExist(s.config.Name, awsErr) {
				return nil
			}
			return awsErr
		}
		return err
	}

	if len(out.Stacks) != 1 {
		// This is a weird error that should have been caught above?
		return errors.New(
			fmt.Sprintf("Did not find correct number of stacks. Found: %d", len(out.Stacks)),
		)
	}

	s.cloudStack = out.Stacks[0]

	// TODO Find a better place to initialize the StackId. In load(),
	// this assignment happens repeatedly, which is wasted and could lead
	// to confusing bugs in the future.
	s.eventLoader.setStackId(s.cloudStack.StackId)

	return nil
}

func (s *Stack) Name() string {
	if s.cloudStack == nil {
		return ""
	}
	return s.config.Name
}

func (s *Stack) storeLastEvent() error {
	if s.cloudStack == nil {
		return nil
	}

	return s.eventLoader.storeLastEvent()
}

// Sync applies the stack configuration to Cloudformation Stack. If the
// Cloudformation Stack does not exist, Sync will create a new Cloudformation
// Stack. If the Cloudformation Stack does exist, then Sync will update the
// Cloudformation Stack.
func (s *Stack) Sync() error {
	if s.cloudStack == nil {
		return s.createStack()
	} else {
		return s.updateStack()
	}
}

func (s *Stack) waitUntilDone(consumer EventConsumer) error {
	var status string
	var attempts int

	for attempts = 0; attempts < s.waitAttempts; attempts++ {
		err := s.load()
		if err != nil {
			return err
		}

		// s.eventLoader.stackId = s.cloudStack.StackId

		err = s.latestEvents(consumer)
		if err != nil {
			return err
		}

		status = aws.StringValue(s.cloudStack.StackStatus)
		if _, ok := stackDoneStatuses[status]; ok {
			break
		}

		if attempts != s.waitAttempts-1 {
			s.waiter.wait()
		}
	}

	if attempts == s.waitAttempts {
		return errors.New(
			"Stack failed to complete in time. Check your stack status in cloudformation.",
		)
	}

	isSuccess := stackDoneStatuses[status]
	if !isSuccess {
		return errors.New(fmt.Sprintf("stacked failed to complete. status: %s", status))
	}

	return nil
}

// Runs Sync() and then polls for StackEvents to pass to consumer. This call
// will block until the Cloudformation Stack has completed creating or
// updating a Cloudformation Stack.
//
// StackEvents passed to consumer appear in chronological order.
func (s *Stack) SyncAndPollEvents(consumer EventConsumer) error {
	err := s.Sync()
	if err != nil {
		return err
	}

	err = s.waitUntilDone(consumer)
	if err != nil {
		return err
	}

	return nil
}

func (s *Stack) createStack() error {
	_, err := s.api.CreateStack(s.createStackInput())

	if err != nil {
		return errors.Wrap(err, "failed to create stack: ")
	}

	return nil
}

func (s *Stack) createStackInput() *cloudformation.CreateStackInput {
	input := cloudformation.CreateStackInput{
		StackName:                   aws.String(s.config.Name),
		TemplateURL:                 aws.String(s.config.TemplateURL),
		EnableTerminationProtection: aws.Bool(s.config.EnableTerminationProtection),
	}

	// TODO: Validate this before making the API request
	// The cloudformation API only allows setting either OnFailure or
	// DisableRollback. But not together.
	//
	// Also, due to the way CreateStackInput
	// uses pointers for all fields, setting a field is a meaninful action.
	// Therefore, this block only sets CreateStackInput fields if the
	// corresponding StackConfig is NOT the zero'd default.
	if s.config.OnFailure != "" {
		input.OnFailure = aws.String(s.config.OnFailure)
	} else if s.config.DisableRollback {
		input.DisableRollback = aws.Bool(s.config.DisableRollback)
	}

	if len(s.config.Parameters) > 0 {
		input.Parameters = make([]*cloudformation.Parameter, 0, len(s.config.Parameters))
		for k, v := range s.config.Parameters {
			input.Parameters = append(
				input.Parameters,
				&cloudformation.Parameter{
					ParameterKey:   aws.String(k),
					ParameterValue: aws.String(v),
				},
			)
		}
	}

	if len(s.config.Tags) > 0 {
		input.Tags = make([]*cloudformation.Tag, 0, len(s.config.Tags))
		for k, v := range s.config.Tags {
			input.Tags = append(
				input.Tags,
				&cloudformation.Tag{Key: aws.String(k), Value: aws.String(v)},
			)
		}
	}

	if len(s.config.Capabilities) > 0 {
		input.Capabilities = aws.StringSlice(s.config.Capabilities)
	}

	return &input
}

func (s *Stack) updateStack() error {
	_, err := s.api.UpdateStack(s.updateStackInput())

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return awsErr
		}
		return errors.Wrap(err, "failed to update stack: ")
	}
	return nil
}

func (s *Stack) updateStackInput() *cloudformation.UpdateStackInput {
	input := cloudformation.UpdateStackInput{
		StackName:   aws.String(s.config.Name),
		TemplateURL: aws.String(s.config.TemplateURL),
	}

	if len(s.config.Parameters) > 0 {
		input.Parameters = make([]*cloudformation.Parameter, 0, len(s.config.Parameters))
		for k, v := range s.config.Parameters {
			input.Parameters = append(
				input.Parameters,
				&cloudformation.Parameter{
					ParameterKey:   aws.String(k),
					ParameterValue: aws.String(v),
				},
			)
		}
	}

	if len(s.config.Tags) > 0 {
		input.Tags = make([]*cloudformation.Tag, 0, len(s.config.Tags))
		for k, v := range s.config.Tags {
			input.Tags = append(
				input.Tags,
				&cloudformation.Tag{Key: aws.String(k), Value: aws.String(v)},
			)
		}
	}

	if len(s.config.Capabilities) > 0 {
		input.Capabilities = aws.StringSlice(s.config.Capabilities)
	}
	return &input
}

// NoStackUpdatesToPerform inspects awserr.Error to detect if a Cloudformation
// Stack does not rquire any updates.
//
// The aws-go-sdk doesn't return an easily detectable Code. UpdateStack returns a
// "ValidationError" error with a specific Message. This function searches for
// the message that indicates not updates are to be performed.
func NoStackUpdatesToPerform(err awserr.Error) bool {
	return err.Code() == "ValidationError" &&
		err.Message() == "No updates are to be performed."
}

// stackDoesNotExist inspects an awserr.Error to detect when a Cloudformation
// Stack does not exist.
//
// The cloudformation.DescribeStacks() method does not return a proper error
// code when a stack doesn't exist. It always returns an awserr.Error with a
// specific message. This function searches for that message.
func stackDoesNotExist(stackName string, err awserr.Error) bool {
	doesNotExist := fmt.Sprintf(stackDoesNotExistErrorFmt, stackName)
	return err.Code() == "ValidationError" && strings.Contains(err.Message(), doesNotExist)
}
