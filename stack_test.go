package stackshot

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	cfn "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

// MockAPI implements the cloudformationiface.CloudFormationAPI interface along
// with helper methods to enable mocking and verifying method calls within
// tests.
//
// Each required method has a corresponding <method>Fn field that enable
// customization. The <method>Fn functions have corresponding helper functions
// for the form Gen<method>Fn and GenError<method>Fn.
type MockAPI struct {
	cloudformationiface.CloudFormationAPI

	DescribeStacksFn           func(*cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error)
	DescribeStackEventsFn      func(*cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error)
	CreateStackFn              func(*cfn.CreateStackInput) (*cfn.CreateStackOutput, error)
	UpdateStackFn              func(*cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error)
	DescribeStackEventsPagesFn func(*cfn.DescribeStackEventsInput, func(*cfn.DescribeStackEventsOutput, bool) bool) error
}

func (m *MockAPI) DescribeStacks(input *cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
	return m.DescribeStacksFn(input)
}

func (m *MockAPI) DescribeStackEvents(input *cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error) {
	return m.DescribeStackEventsFn(input)
}

func (m *MockAPI) CreateStack(input *cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
	return m.CreateStackFn(input)
}

func (m *MockAPI) UpdateStack(input *cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error) {
	return m.UpdateStackFn(input)
}

func (m *MockAPI) DescribeStackEventsPages(input *cfn.DescribeStackEventsInput, fn func(*cfn.DescribeStackEventsOutput, bool) bool) error {
	return m.DescribeStackEventsPagesFn(input, fn)
}

// Mock helpers

func NewDescribeStackPlayer(responses ...*describeStackResponse) *describeStacksResponsePlayer {
	return &describeStacksResponsePlayer{responses: responses}

}

func NewDescribeStackResponse(s *cfn.Stack) *describeStackResponse {
	return &describeStackResponse{
		stacks: []*cfn.Stack{s},
	}
}

type describeStackResponse struct {
	// out    cfn.DescribeStacksOutput
	stacks []*cfn.Stack
	err    error
}

// describeStacksResponsePlayer simulates making multiple calls to the
// DescribeStacks() api request.
//
// Build up describeStacksResponsePlayer by adding to the responses field.
// Then assign the allocated describeStacksResponsePlayer's DescribeStacksPlay
// function to the MockAPI.DescribeStacksFn
type describeStacksResponsePlayer struct {
	responses []*describeStackResponse
	call      int
}

func (d *describeStacksResponsePlayer) DescribeStacksFn(input *cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
	if d.call >= len(d.responses) {
		panic(fmt.Errorf("Call to DescribeStacksPlayer exceeded number of available responses. Call count: %d. Num responses available: %d", d.call, len(d.responses)))
	}

	resp := d.responses[d.call]
	d.call++
	return &cfn.DescribeStacksOutput{Stacks: resp.stacks}, resp.err
}

func GenDescribeStacksFn(stacks ...*cfn.Stack) func(*cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
	return func(input *cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
		out := cfn.DescribeStacksOutput{}
		out.Stacks = stacks
		return &out, nil
	}
}

func GenErrorDescribeStacksFn(err error) func(*cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
	return func(input *cfn.DescribeStacksInput) (*cfn.DescribeStacksOutput, error) {
		return nil, err
	}
}

func GenDescribeStackEventsFn(events ...*cfn.StackEvent) func(input *cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error) {
	return func(input *cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error) {
		out := cfn.DescribeStackEventsOutput{
			StackEvents: events,
		}
		return &out, nil
	}
}

func GenErrorDescribeStackEventsFn(err error) func(*cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error) {
	return func(input *cfn.DescribeStackEventsInput) (*cfn.DescribeStackEventsOutput, error) {
		return nil, err
	}
}

func GenCreateStackFn(output *cfn.CreateStackOutput) func(*cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
	return func(input *cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
		return output, nil
	}
}

func GenErrorCreateStackFn(err error) func(*cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
	return func(input *cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
		return nil, err
	}
}

func GenUpdateStackFn(output *cfn.UpdateStackOutput) func(*cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error) {
	return func(input *cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error) {
		return output, nil
	}
}

func GenErrorUpdateStackFn(err error) func(*cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error) {
	return func(input *cfn.UpdateStackInput) (*cfn.UpdateStackOutput, error) {
		return nil, err
	}
}

func GenDescribeStackEventsPagesFn(output *cfn.DescribeStackEventsOutput, lastPage bool) func(*cfn.DescribeStackEventsInput, func(*cfn.DescribeStackEventsOutput, bool) bool) error {
	return func(input *cfn.DescribeStackEventsInput, fn func(*cfn.DescribeStackEventsOutput, bool) bool) error {
		fn(output, lastPage)
		return nil
	}
}

// impatientWaiter implements the waiter interface but hates waiting.
type impatientWaiter struct {
}

func (c *impatientWaiter) wait() {
}

// stubEventLoader implements eventLoader interface to help loosen coupling
// between synchronizing a Cloudformation Stack and polling for StackEvents of
// a Cloudformation Stack.
type stubEventLoader struct{}

func (s *stubEventLoader) storeLastEvent() error {
	return nil
}

func (s *stubEventLoader) latestEvents(consumer EventConsumer) error {
	e := &cfn.StackEvent{}
	return consumer.Consume(e)
}

func (s *stubEventLoader) setStackId(id *string) {
}

func TestLoadStack(t *testing.T) {
	config := StackConfig{
		Name:     "mystack",
		Template: "https://bucket.s3.amazonaws.com/template.yaml",
	}

	t.Run(
		"Cloudformation Stack does not exist",
		func(t *testing.T) {
			expErr := awserr.New("ValidationError", fmt.Sprintf(stackDoesNotExistErrorFmt, config.Name), errors.New("orig error"))
			api := MockAPI{}
			api.DescribeStacksFn = GenErrorDescribeStacksFn(expErr)

			stack, err := LoadStack(&api, &config)
			if err != nil {
				t.Errorf("Expected LoadStack() to succeed. Got error: %s", err)
			}

			if stack.cloudStack != nil {
				t.Errorf("Expected cloudStack to be nil. Got: %+v", stack.cloudStack)
			}
			if !cmp.Equal(*stack.config, config) {
				t.Errorf("Expected config: %+v. Got: %+v", config, stack.config)
			}
		},
	)

	t.Run(
		"Cloudformation Stack exists",
		func(t *testing.T) {
			expStack := cfn.Stack{StackName: aws.String(config.Name)}
			expEvent := cfn.StackEvent{EventId: aws.String("event-id")}
			api := MockAPI{}
			api.DescribeStacksFn = GenDescribeStacksFn(&expStack)
			api.DescribeStackEventsFn = GenDescribeStackEventsFn(&expEvent)

			stack, err := LoadStack(&api, &config)
			if err != nil {
				t.Errorf("Expected LoadStack() to succeed. Got error: %s", err)
			}

			if stack.cloudStack == nil {
				t.Errorf("Expected stack to exist. Got nil")
			}
			if aws.StringValue(stack.cloudStack.StackName) != aws.StringValue(expStack.StackName) {
				t.Errorf(
					"Expected cloudStack.StackName to be '%s'. Got: %s",
					aws.StringValue(expStack.StackName),
					aws.StringValue(stack.cloudStack.StackName),
				)
			}

			if !cmp.Equal(*stack.config, config) {
				t.Errorf("Expected config: %+v. Got: %+v", config, stack.config)
			}

		},
	)

	t.Run(
		"Failed DescribeStacks Requests",
		func(t *testing.T) {
			expErr := awserr.New("ValidationError", "bad API", errors.New("orig error"))
			api := MockAPI{}
			api.DescribeStacksFn = GenErrorDescribeStacksFn(expErr)

			stack, err := LoadStack(&api, &config)
			if err == nil {
				t.Errorf("Expected LoadStack() to fail. Got success")
			}

			if stack != nil {
				t.Errorf("Expected cloudStack to be nil. Got: %+v", stack.cloudStack)
			}

		},
	)

	t.Run(
		"Failed DescribeStackEvents Requests with awserr",
		func(t *testing.T) {
			expStack := cfn.Stack{StackName: aws.String(config.Name)}
			expErr := awserr.New(cfn.ErrCodeOperationNotFoundException, "op not found", errors.New("orig error"))
			api := MockAPI{}
			api.DescribeStacksFn = GenDescribeStacksFn(&expStack)
			api.DescribeStackEventsFn = GenErrorDescribeStackEventsFn(expErr)

			stack, err := LoadStack(&api, &config)
			if err == nil {
				t.Errorf("Expected LoadStack() to fail. Got success")
			}

			if stack != nil {
				t.Errorf("Expected cloudStack to be nil. Got: %+v", stack.cloudStack)
			}

			// TODO Rethink what type of errors to return - raw awserr or a wrapped error?
			// if err != expErr {
			// t.Errorf("Expected error:\n %s (%+v)\nGot:\n%s (%+v)", expErr, expErr, err, err)
			// }
		},
	)

	t.Run(
		"Failed DescribeStackEvents Requests with error",
		func(t *testing.T) {
			expStack := cfn.Stack{StackName: aws.String(config.Name)}
			expErr := errors.New("basic error")
			api := MockAPI{}
			api.DescribeStacksFn = GenDescribeStacksFn(&expStack)
			api.DescribeStackEventsFn = GenErrorDescribeStackEventsFn(expErr)

			stack, err := LoadStack(&api, &config)
			if err == nil {
				t.Errorf("Expected LoadStack() to fail. Got success")
			}

			if stack != nil {
				t.Errorf("Expected cloudStack to be nil. Got: %+v", stack.cloudStack)
			}

		},
	)
}

func TestSync(t *testing.T) {
	config := StackConfig{
		Name:     "mystack",
		Template: "https://bucket.s3.amazonaws.com/template.yaml",
		Parameters: map[string]string{
			"MyParam": "MyValue",
		},
		Tags: map[string]string{
			"environment": "production",
		},
	}

	t.Run(
		"Create new stack",
		func(t *testing.T) {
			api := MockAPI{}
			stubOutput := cfn.CreateStackOutput{}
			api.CreateStackFn = GenCreateStackFn(&stubOutput)

			stack := Stack{
				api:    &api,
				config: &config,
			}

			err := stack.Sync()
			if err != nil {
				t.Errorf("Expected Sync() to succeed. Got failure")
			}
		},
	)

	t.Run(
		"Create new stack failure",
		func(t *testing.T) {
			api := MockAPI{}
			stubErr := errors.New("stub error")
			api.CreateStackFn = GenErrorCreateStackFn(stubErr)

			stack := Stack{
				api:    &api,
				config: &config,
			}

			err := stack.Sync()
			if err == nil {
				t.Errorf("Expected Sync() to fail. Got success")
			}
		},
	)

	t.Run(
		"Update existing stack",
		func(t *testing.T) {
			api := MockAPI{}
			stubOutput := cfn.UpdateStackOutput{}
			api.UpdateStackFn = GenUpdateStackFn(&stubOutput)

			stack := Stack{
				cloudStack: &cfn.Stack{StackName: aws.String(config.Name)},
				api:        &api,
				config:     &config,
			}

			err := stack.Sync()
			if err != nil {
				t.Errorf("Expected Sync() to succeed. Got failure")
			}
		},
	)

	t.Run(
		"Update existing stack failure",
		func(t *testing.T) {
			api := MockAPI{}
			stubErr := errors.New("stub error")
			api.UpdateStackFn = GenErrorUpdateStackFn(stubErr)

			stack := Stack{
				cloudStack: &cfn.Stack{StackName: aws.String(config.Name)},
				api:        &api,
				config:     &config,
			}

			err := stack.Sync()
			if err == nil {
				t.Errorf("Expected Sync() to fail. Got success")
			}
		},
	)
}

func TestWaitUntilDone(t *testing.T) {
	config := StackConfig{
		Name:     "mystack",
		Template: "https://bucket.s3.amazonaws.com/template.yaml",
	}

	tests := []struct {
		status      string
		shouldError bool
	}{
		{"CREATE_COMPLETE", false},
		{"UPDATE_COMPLETE", false},
		{"CREATE_FAILED", true},
		{"UPDATE_FAILED", true},
		{"UPDATE_ROLLBACK_COMPLETE", true},
		{"UPDATE_ROLLBACK_FAILED", true},
		{"DELETE_COMPLETE", true},
		{"DELETE_FAILED", true},
		{"ROLLBACK_FAILED", true},
		{"ROLLBACK_COMPLETE", true},
	}

	for _, test := range tests {
		var testName string
		if test.shouldError {
			testName = fmt.Sprintf("Wait fails when stack completes with %s", test.status)
		} else {
			testName = fmt.Sprintf("Wait succeeds when stack completes with %s", test.status)
		}

		t.Run(
			testName,
			func(t *testing.T) {
				api := MockAPI{}
				expStack := cfn.Stack{
					StackName:   aws.String(config.Name),
					StackStatus: aws.String(test.status),
				}
				api.DescribeStacksFn = GenDescribeStacksFn(&expStack)

				waiter := &impatientWaiter{}
				stack := Stack{
					api:          &api,
					config:       &config,
					waitAttempts: 10,
					waiter:       waiter,
					eventLoader:  &stubEventLoader{},
				}

				nullConsumer := func(event *cfn.StackEvent) error {
					return nil
				}

				err := stack.waitUntilDone(EventConsumerFunc(nullConsumer))
				if test.shouldError {
					if err == nil {
						t.Errorf("Expected Wait to fail when stack status: '%s'. Got success.", test.status)
					}
				} else {
					if err != nil {
						t.Errorf("Expected Wait to succeed when stack status: '%s'. Got error: %s", test.status, err)
					}
				}
			},
		)
	}

	// TODO: redo this. There are duplicate tests
	t.Run(
		"Create with OnFailure DELETE",
		func(t *testing.T) {
			api := MockAPI{}

			player := NewDescribeStackPlayer(
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("CREATE_IN_PROGRESS"),
					}),
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("DELETE_IN_PROGRESS"),
					}),
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("DELETE_COMPLETE"),
					}),
			)

			api.DescribeStacksFn = player.DescribeStacksFn

			waiter := &impatientWaiter{}
			stack := Stack{
				api:          &api,
				config:       &config,
				waitAttempts: 10,
				waiter:       waiter,
				eventLoader:  &stubEventLoader{},
			}

			nullConsumer := func(event *cfn.StackEvent) error {
				return nil
			}

			err := stack.waitUntilDone(EventConsumerFunc(nullConsumer))
			if err == nil {
				t.Errorf("Expected waitUntilDone to fail. Got success.")
			}

		},
	)

	t.Run(
		"Fails after all attempts",
		func(t *testing.T) {

			api := MockAPI{}
			expStack := cfn.Stack{
				StackName:   aws.String(config.Name),
				StackStatus: aws.String("CREATE_IN_PROGRESS"),
			}
			api.DescribeStacksFn = GenDescribeStacksFn(&expStack)

			player := NewDescribeStackPlayer(
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("CREATE_IN_PROGRESS"),
					}),
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("CREATE_IN_PROGRESS"),
					}),
				NewDescribeStackResponse(
					&cfn.Stack{
						StackId:     aws.String("stack-001"),
						StackName:   aws.String("mystackname"),
						StackStatus: aws.String("CREATE_IN_PROGRESS"),
					}),
			)

			api.DescribeStacksFn = player.DescribeStacksFn

			waiter := &impatientWaiter{}
			stack := Stack{
				api:          &api,
				config:       &config,
				waitAttempts: 3,
				waiter:       waiter,
				eventLoader:  &stubEventLoader{},
			}

			nullConsumer := func(event *cfn.StackEvent) error {
				return nil
			}
			err := stack.waitUntilDone(EventConsumerFunc(nullConsumer))
			if err == nil {
				t.Errorf("Expected Wait to fail due to max attempts. Got success instead.")
			}
		},
	)
}
