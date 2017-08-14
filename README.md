# ECS CLI

The `ecs` CLI tool provides a set of simple commands to query ECS for information. It presents the
data in compact, tabular format in most cases, with links to the AWS console where useful.

Other than querying, it has a command to dump environment variables in formats useful for local
development.

## Overview

The `ecs` tool can query AWS ECS by cluster, service, or task. The `--help` option shows the
commands.

```
> ecs
usage: ecs [<flags>] <command> [<args> ...]

A friendly ECS CLI

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).

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
    List tasks belong to a service

  task <cluster> <task>
    Describe the given task

  container-env [<flags>] <cluster> <service>
    List environment variables for the task's container
```

## List clusters

`ecs clusters` lists the ECS clusters in our AWS account.

```
> ecs clusters
+------------------------+---------------------+-----------------+---------------+---------------+
|      CLUSTER NAME      | CONTAINER INSTANCES | ACTIVE SERVICES | RUNNING TASKS | PENDING TASKS |
+------------------------+---------------------+-----------------+---------------+---------------+
| default                |                   0 |               0 |             0 |             0 |
| ecs-prod               |                   3 |               3 |             3 |             0 |
| ecs-staging            |                   3 |               6 |             6 |             0 |
+------------------------+---------------------+-----------------+---------------+---------------+
```

## List services

`ecs services` lists the services within a cluster. For large clusters. this command can take a
while.

```
> ecs services ecs-prod
Found 6 services
+------------------------------+--------+---------+---------+---------+
|             SERVICE NAME     | STATUS | DESIRED | RUNNING | PENDING |
+------------------------------+--------+---------+---------+---------+
| service-applepicker-ecs-prod | ACTIVE |       6 |       6 |       0 |
| service-helloworld-ecs-prod  | ACTIVE |      89 |      89 |       0 |
| service-my-blog-ecs-prod     | ACTIVE |       5 |       5 |       0 |
+------------------------------+--------+---------+---------+---------+
```

## Describe service

`ecs service` shows the details of a service, and provides useful links to the dashboard.

```
> ecs service ecs-prod service-my-blog-ecs-prod
Service
+----------------------+-----------------------------------------------------------------------------------------------------------------------------------+
| Name                 | service-applepicker-ecs-prod                                                                                                      |
| Status               | ACTIVE                                                                                                                            |
| Service ARN          | arn:aws:ecs:us-west-2:4817267453:service/service-applepicker-ecs-prod                                                             |
| Task Definition      | arn:aws:ecs:us-west-2:4817267453:task-definition/task-applepicker-ecs-prod:38                                                     |
| Desired Count        | 1                                                                                                                                 |
| Running Count        | 1                                                                                                                                 |
| Pending Count        | 0                                                                                                                                 |
| Service Link         | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/services/service-applepicker-ecs-prod/tasks |
| Task Definition Link | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/taskDefinitions/task-applepicker-ecs-prod/                    |
| LB Container Name    | ngfe                                                                                                                              |
| LB Container Port    | 8000                                                                                                                              |
+----------------------+-----------------------------------------------------------------------------------------------------------------------------------+
Containers
+-------------+-----+--------+---------+
|    NAME     | CPU | MEMORY | COMMAND |
+-------------+-----+--------+---------+
| applepicker |   0 |    256 |         |
| ngfe        |   0 |    256 |         |
+-------------+-----+--------+---------+
```

## List tasks

`ecs tasks` lists the tasks belonging to the service, by ARN. It's not useful by itself, but the
ARNs can be given to the `ecs task` command to get task details:

```
> ecs tasks ecs-prod service-applepicker-ecs-prod

Running Tasks:
	arn:aws:ecs:us-west-2:4817267453:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c

Stopped Tasks:

Use the "task" command to get details of a task. For example:
	ecs task ecs-prod arn:aws:ecs:us-west-2:4817267453:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c
```

## Describe task

`ecs task` shows the details of a given task, by ARN, and provides useful links to the dashboard.

```
> ecs task ecs-prod arn:aws:ecs:us-west-2:192431242:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c
Details:
+-------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------+
| Task ID                 | bfbf861b-7f10-4dfb-b344-32169dc3e55c                                                                                                                  |
| Task ARN                | arn:aws:ecs:us-west-2:192431242:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c                                                                             |
| Task Definition         | task-applepicker-ecs-prod                                                                                                                             |
| Container Instance      | 44019f70-aa88-48e3-babf-4614e10afe08                                                                                                                  |
| EC2 Instance            | i-072932614cc14ccf9                                                                                                                                   |
| Task Link               | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/tasks/bfbf861b-7f10-4dfb-b344-32169dc3e55c                      |
| Task Definition Link    | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/taskDefinitions/task-applepicker-ecs-prod/                                        |
| Container Instance Link | https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/ecs-prod/containerInstances/44019f70-aa88-48e3-babf-4614e10afe08         |
| EC2 Instance Link       | https://us-west-2.console.aws.amazon.com/ec2/v2/home?region=us-west-2#Instances:instanceId=i-072932614cc14ccf9                                        |
+-------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------+
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

## Show container environment variables

`ecs container-env` fetches and dumps environment variables for a service's container definition. It
can often be useful to run a container locally with the same configuration as on ECS.

The command supports 3 formats, set with the `--format` flag

- `table` this is the default format, it renders the environment variales as a table
- `shell` renders the environment variables to prefix the command with in `bash` or `zsh`, or pass
into the `env` function
- `export` renders the environment variables as `export` statements to `bash` or `zsh`
- `docker` renders the environment variables as `-e` flags to the `docker` command

```
> ecs container-env ecs-prod service-applepicker-ecs-prod --container applepicker
+-------------------+---------+
|       NAME        |  VALUE  |
+-------------------+---------+
| NODE_ENV          | prod    |
| PORT              | 3000    |
| ORCHARD_API_KEY   | xxxxxxx |
| ORCHARD_API_TOKEN | xxxxxxx |
+------------------+----------+

> ecs container-env ecs-prod service-applepicker-ecs-prod --format=shell --container applepicker
NODE_ENV="prod" PORT="3000" ORCHARD_API_KEY="xxxxxxxx" ORCHARD_API_TOKEN="xxxxxxxx"

> ecs container-env ecs-prod service-applepicker-ecs-prod --format=docker --container applepicker
-eNODE_ENV="prod" -ePORT="3000" -eORCHARD_API_KEY="xxxxxxxx" -eORCHARD_API_TOKEN="xxxxxxxx"

> ecs container-env ecs-prod service-applepicker-ecs-prod --format=export --container applepicker
export NODE_ENV="prod"
export PORT="3000"
export ORCHARD_API_KEY="xxxxxxx"
export ORCHARD_API_TOKEN="xxxxxxx"
```
