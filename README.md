# stackshot
`stackshot` is a declarative approach to managing Cloudformation Stacks.
`stackshot` is a command line tool that consumes YAML files that create and
update Cloudformation Stacks.

Below is a simple workflow to create and update a Stack using [AWS's S3 bucket
sample
template](https://s3.us-west-2.amazonaws.com/cloudformation-templates-us-west-2/S3_Website_Bucket_With_Retain_On_Delete.template).

`mybucket.yaml` is a simple Stack configuration with a Stack's name and template
(the only required arguments to create a stack):

```yaml
# saved as mybucket.yaml
---
name: mybucket
template: https://s3.amazonaws.com/cloudformation-templates-us-east-1/S3_Website_Bucket_With_Retain_On_Delete.template
```

To create a stack, run `stackshot`:

```sh
$ stackshot mybucket.yaml
2020-10-04 18:28:47.277 +0000 UTC mybucket(AWS::CloudFormation::Stack) UPDATE_IN_PROGRESS User Initiated
2020-10-04 18:28:50.968 +0000 UTC S3Bucket(AWS::S3::Bucket) UPDATE_IN_PROGRESS
2020-10-04 18:29:11.769 +0000 UTC S3Bucket(AWS::S3::Bucket) UPDATE_COMPLETE
2020-10-04 18:29:13.621 +0000 UTC mybucket(AWS::CloudFormation::Stack) UPDATE_COMPLETE_CLEANUP_IN_PROGRESS
2020-10-04 18:29:14.25 +0000 UTC mybucket(AWS::CloudFormation::Stack) UPDATE_COMPLETE

```

Now let's assume that template added a Parameter to set [S3's
AccessControl](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket.html#cfn-s3-bucket-accesscontrol)
setting. You can add parameters to `mybucket.yaml` like so:

```yaml
name: mybucket
template: https://s3.amazonaws.com/cloudformation-templates-us-east-1/S3_Website_Bucket_With_Retain_On_Delete.template
parameters:
  AccessControl: Private

```

Running `stackshot` to sync the `mybucket.yaml` updates the existing stack in Cloudformation:

```sh
$ stackshot mybucket.yaml
2020-10-06 03:37:21.481 +0000 UTC mybucket(AWS::CloudFormation::Stack) UPDATE_IN_PROGRESS User Initiated
2020-10-06 03:37:24.811 +0000 UTC S3Bucket(AWS::S3::Bucket) UPDATE_IN_PROGRESS
2020-10-06 03:37:45.617 +0000 UTC S3Bucket(AWS::S3::Bucket) UPDATE_COMPLETE
2020-10-06 03:37:47.38 +0000 UTC mybucket(AWS::CloudFormation::Stack) UPDATE_COMPLETE
```

## Features
* Create/Update Cloudformation Stacks using YAML files
* Designed for use with Continuous Integration/Delivery systems like GitHub
  Actions
* Explicitly does not support dynamic YAML generation. If you'd like to add
  conditionals  and loops in your YAML, please look to templating languages that
  can handle this behavior much better. e.g., [jsonnet](#)(https://jsonnet.org).

## Installation

```yaml
git clone git@github.com:tightlycoupled/stackshot.git
cd stackshot
make build
cp stackshot /usr/local/bin
```

## Usage

```sh
stackshot path/to/stack_configuration.yaml
```

## Stack Configuration YAML

You can find all available Stack settings in the
commented [kitchen-sink.yaml](examples/kitchen-sink.yaml) example configuration file. The settings map to [`create-stack`](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/cloudformation/create-stack.html) and [`update-stack`](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/cloudformation/update-stack.html) parameters.

## Motivations
AWS Cloudformation is a service that allows you to manage your infrastructure as
code by declaring cloud resources in JSON or YAML Templates. Given that
Templates  declarative, why aren't Stacks? For a handful of stacks, the lack of
declarative Stack management isn't a problem. But when the stacks outnumber the
possible things I can keep in my head, and the number of Parameters a template
takes grows similarly, managing Stacks becomes an error-prone burden.

Therefore, I wanted two things:

1. Store the Stack configuration in a human-readable format (YAML) that I could
   store in a git repository. This allows configuration review from my peers and
   a history of what happened to our Stacks.
2. A tool that behaved like `aws cloudformation deploy` but works off of the
   YAML configuration instead of command line arguments.

Therefore, `stackshot` was built.

### The Name
`stackshot` is a play off of a [steel Misting, or better known as a Coinshot](https://coppermind.net/wiki/Steel).


## Contributing
Pull requests are welcome. For major changes, please open an issue first to
discuss what you would like to change.

Please make sure to update tests as appropriate.


## License
[MIT](https://choosealicense.com/licenses/mit/)




