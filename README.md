# ECS Query CLI

The `ecsq` CLI tool provides a set of simple commands to query ECS for information. It presents the
data in compact, tabular format in most cases, with links to the AWS console where useful.

# What's so special about this compared to the `aws-cli`?

Well, if the tabular output and short, consistent commands doesn't interest you, not much. This tool
is about saving a few steps in my workflow.

The `aws-cli` tool is great for fetching the information, but it produces giant blobs of JSON and
can sometimes take a few invocations to get what I want. `ecsq` can be seen as a script that does a
few things on top of `aws-cli`. The major improvements, other than how information is presented, are

1. `eval "$(ecsq container-env <cluster> <service> --format=export)"` sources the production
   environment variables into your shell
2. direct links to the AWS console for services and tasks

## Installation

`ecsq` is distributed via Go. Make sure you have Go installed, and run

```
go get -u github.com/mightyguava/ecsq
```

## Upgrading

`ecsq` does not yet follow semantic versioning. Upgrading uses the same command as installing
```
go get -u github.com/mightyguava/ecsq
```

## Configuration and credentials

`ecsq` uses the `~/.aws/credentials` and `~/.aws/config` for credentials and configuration, respectively.

You can also use the standard AWS CLI environment variables for overriding them. The common environment variables are

- `AWS_ACCESS_KEY_ID`
- `AWS_ACCESS_SECRET_KEY`
- `AWS_DEFAULT_REGION`

More parameters can be found in [Configuring the AWS CLI](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Overview

The `ecsq` tool can query AWS ECS by cluster, service, or task. The `--help` option shows the
commands.

```
> ecsq
usage: ecsq [<flags>] <command> [<args> ...]

A friendly ECS CLI

Flags:
  --help             Show context-sensitive help (also try --help-long and --help-man).
  --profile=PROFILE  AWS profile to use. Overrides the ~/.aws/config and AWS_DEFAULT_PROFILE

Commands:
  help [<command>...]
    Show help.

  clusters
    List existing clusters

  services [<flags>] <cluster>
    List services within the cluster

  service [<flags>] <cluster> <service>
    Show details of a service

  tasks <cluster> <service>
    List tasks belonging to a service

  task <cluster> <task or service>
    Describe the given task. If a service name is provided instead, describes an arbitrary task for that service.

  container-env [<flags>] <cluster> <service>
    List environment variables for the task's container
```

## List clusters

`ecsq clusters` lists the ECS clusters in our AWS account.

```
> ecsq clusters
+------------------------+---------------------+-----------------+---------------+---------------+
|      CLUSTER NAME      | CONTAINER INSTANCES | ACTIVE SERVICES | RUNNING TASKS | PENDING TASKS |
+------------------------+---------------------+-----------------+---------------+---------------+
| default                |                   0 |               0 |             0 |             0 |
| ecs-prod               |                   3 |               3 |             3 |             0 |
| ecs-staging            |                   3 |               6 |             6 |             0 |
+------------------------+---------------------+-----------------+---------------+---------------+
```

## List services

`ecsq services` lists the services within a cluster. For large clusters. this command can take a
while. Results can be filtered using the `--filter` flag.

```
> ecsq services ecs-prod
Found 3 services
+--------------+--------+---------+--------+----------+
| SERVICE NAME | STATUS | DESIRED | RUNNING | PENDING |
+--------------+--------+---------+--------+----------+
| applepicker  | ACTIVE |       6 |       6 |       0 |
| helloworld   | ACTIVE |      89 |      89 |       0 |
| my-blog      | ACTIVE |       5 |       5 |       0 |
+--------------+--------+---------+--------+----------+
```

## Describe service

`ecsq service` shows the details of a service, and provides useful links to the dashboard.

```
> ecsq service ecs-prod applepicker
Service
+----------------------+------------------------------------------------------------------------------------------------------------------+
| Name                 | applepicker                                                                                                      |
| Status               | ACTIVE                                                                                                           |
| Service ARN          | arn:aws:ecs:us-west-2:4817267453:service/applepicker                                                             |
| Task Definition      | arn:aws:ecs:us-west-2:4817267453:task-definition/task-applepicker-ecs-prod:38                                    |
| Desired Count        | 1                                                                                                                |
| Running Count        | 1                                                                                                                |
| Pending Count        | 0                                                                                                                |
| Service Link         | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/services/applepicker/tasks |
| Task Definition Link | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/taskDefinitions/task-applepicker-ecs-prod/   |
| LB Container Name    | ngfe                                                                                                             |
| LB Container Port    | 8000                                                                                                             |
+----------------------+------------------------------------------------------------------------------------------------------------------+
Containers
+-------------+-----+--------+---------+
|    NAME     | CPU | MEMORY | COMMAND |
+-------------+-----+--------+---------+
| applepicker |   0 |    256 |         |
| ngfe        |   0 |    256 |         |
+-------------+-----+--------+---------+
```

## List service events

`ecsq service --events` lists events for that service in addition to the service details.

```
> ecsq service ecs-prod applepicker
...
2017-08-11T18:08:05Z: (service applepicker) has reached a steady state.
2017-08-15T12:12:08Z: (service applepicker) has reached a steady state.
2017-08-15T18:12:21Z: (service applepicker) has reached a steady state.
2017-08-15T19:00:53Z: (service applepicker) has stopped 2 running tasks: (task 02262781-54d0-4d1a-b76f-77693b0547f1) (task 56dce574-c297-41ef-8ec8-7b00477c5bfa).
2017-08-15T19:01:04Z: (service applepicker) has reached a steady state.
```

## List tasks

`ecsq tasks` lists the tasks belonging to the service, by ARN. It's not useful by itself, but the
ARNs can be given to the `ecsq task` command to get task details:

```
> ecsq tasks ecs-prod applepicker

Running Tasks:
	arn:aws:ecs:us-west-2:4817267453:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c

Stopped Tasks:

Use the "task" command to get details of a task. For example:
	ecsq task ecs-prod arn:aws:ecs:us-west-2:4817267453:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c
```

## Describe task

`ecsq task` shows the details of a given task, by ARN or task ID, and provides useful links to the
dashboard. If a service name is provided instead of an ARN/ID, then it will look up an arbitrary
task for the service and provide its details

```
> ecsq task ecs-prod applepicker
> ecsq task ecs-prod arn:aws:ecs:us-west-2:192431242:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c
Details:
+-------------------------+-----------------------------------------------------------------------------------------------------------------------------------------------+
| Task ID                 | bfbf861b-7f10-4dfb-b344-32169dc3e55c                                                                                                          |
| Task ARN                | arn:aws:ecs:us-west-2:192431242:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c                                                                     |
| Task Definition         | task-applepicker-ecs-prod                                                                                                                     |
| Container Instance      | 44019f70-aa88-48e3-babf-4614e10afe08                                                                                                          |
| EC2 Instance            | i-072932614cc14ccf9                                                                                                                           |
| Task Link               | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/tasks/bfbf861b-7f10-4dfb-b344-32169dc3e55c              |
| Task Definition Link    | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/taskDefinitions/task-applepicker-ecs-prod/                                |
| Container Instance Link | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/containerInstances/44019f70-aa88-48e3-babf-4614e10afe08 |
| EC2 Instance Link       | https://us-west-2.console.aws.amazon.com/ec2/v2/home?region=us-west-2#Instances:instanceId=i-072932614cc14ccf9                                |
+-------------------------+-----------------------------------------------------------------------------------------------------------------------------------------------+
Containers:
+-------------+--------------------------+--------------------+
| applepicker | Status                   | RUNNING            |
|             | Network - Container Port | 3000               |
|             | Network - External Link  | 10.10.121.212:3030 |
| ngfe        | Status                   | RUNNING            |
|             | Network - Container Port | 8000               |
|             | Network - External Link  | 10.10.121.212:8080 |
|             | Network - Container Port | 8001               |
|             | Network - External Link  | 10.10.121.212:8081 |
+-------------+--------------------------+--------------------+
```

## Show (and source) container environment variables

`ecsq container-env` fetches and dumps environment variables for a service's container definition. It
can often be useful to run a container locally with the same configuration as on ECS.

The command supports 3 formats, set with the `--format` flag

- `table` this is the default format, it renders the environment variales as a table
- `shell` renders the environment variables to prefix the command with in `bash` or `zsh`, or pass
into the `env` function
- `export` renders the environment variables as `export` statements to `bash` or `zsh`
- `docker` renders the environment variables as `-e` flags to the `docker` command

Running the command as `eval "$(ecsq container-env <my_cluster> <my_service>) --format=export"` will
automatically populate your environment with container's ECS environment variables. If you want to omit
specific variables here, you can provide a comma-separated list of names via the `--drop` flag. This list is
case-insensitive, e.g. `--drop node_env,port` is the same as `--drop NODE_ENV,PORT`.

```
> ecsq container-env ecs-prod applepicker --container applepicker
+-------------------+---------+
|       NAME        |  VALUE  |
+-------------------+---------+
| NODE_ENV          | prod    |
| PORT              | 3000    |
| ORCHARD_API_KEY   | xxxxxxx |
| ORCHARD_API_TOKEN | xxxxxxx |
+------------------+----------+

> ecsq container-env ecs-prod applepicker --format=shell --container applepicker
NODE_ENV="prod" PORT="3000" ORCHARD_API_KEY="xxxxxxxx" ORCHARD_API_TOKEN="xxxxxxxx"

> ecsq container-env ecs-prod applepicker --format=docker --container applepicker
-eNODE_ENV="prod" -ePORT="3000" -eORCHARD_API_KEY="xxxxxxxx" -eORCHARD_API_TOKEN="xxxxxxxx"

> ecsq container-env ecs-prod applepicker --format=export --container applepicker
export NODE_ENV="prod"
export PORT="3000"
export ORCHARD_API_KEY="xxxxxxx"
export ORCHARD_API_TOKEN="xxxxxxx"
```

## Environment Variables

`ECSQ_SERVICE_NAME_EXPANSION` can be used to specify a Golang template string to expand the provided
service name to. This is kind of an obscure option, to allow writing shorter service names if your
service names follow a predefined format. For example, if your services names follow the format
`service-{{.Name}}-{{.Cluster}}`, then the service `applepicker` on cluster `ecs-prod` will be
expanded to `service-applepicker-ecs-prod` when querying ECS.
