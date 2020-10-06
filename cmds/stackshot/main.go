package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"

	"github.com/tightlycoupled/stackshot"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Missing arguments!")
		fmt.Println("Usage:")
		fmt.Printf("  %s stack.yaml", os.Args[0])
		return
	}

	doc, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Could not read file: %s\n", os.Args[1])
		return
	}

	config, err := stackshot.NewStackFromYAML(doc)
	if err != nil {
		fmt.Printf("Could not load yaml stack: errors: %s\n", err)
		os.Exit(1)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := cloudformation.New(sess)

	stack, err := stackshot.LoadStack(svc, config)
	if err != nil {
		fmt.Println("Broken!", err)
		return
	}

	err = stack.SyncAndPollEvents(stackshot.EventConsumerFunc(stackshot.EventPrinter))
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case awserr.Error:
			if stackshot.NoStackUpdatesToPerform(err) {
				fmt.Println("No updates to be applied")
			} else {
				fmt.Println("AWS error")
				fmt.Println(err.Code(), err.Message(), "", err.OrigErr())
				fmt.Printf("Full error:\n%+v\n", err)

				return
			}
		default:
			fmt.Println("Failed to sync configuration:", err)
			return
		}
	}
}
