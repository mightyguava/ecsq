package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/olekukonko/tablewriter"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		sess       *session.Session
		svc        *ecs.ECS
		AWSProfile string
	)

	app := kingpin.New("ecsq", "A friendly ECS CLI")
	app.Flag("profile", "AWS profile to use. Overrides the ~/.aws/config and AWS_DEFAULT_PROFILE").StringVar(&AWSProfile)
	app.PreAction(func(ctx *kingpin.ParseContext) error {
		// Initialize the session and service before any commands are run
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			Profile:                 AWSProfile,
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			SharedConfigState:       session.SharedConfigEnable,
		}))
		svc = ecs.New(sess)
		return nil
	})
	app.Command("clusters", "List existing clusters").
		Action(func(ctx *kingpin.ParseContext) error {
			result, err := svc.ListClusters(&ecs.ListClustersInput{})
			app.FatalIfError(err, "Could not list clusters")
			clusters, err := svc.DescribeClusters(&ecs.DescribeClustersInput{Clusters: result.ClusterArns})
			app.FatalIfError(err, "Could not describe clusters")
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{
				"Cluster Name",
				"Container Instances",
				"Active Services",
				"Running Tasks",
				"Pending Tasks",
			})
			ClusterSlice(clusters.Clusters).Sort()
			for _, cluster := range clusters.Clusters {
				table.Append([]string{
					*cluster.ClusterName,
					strconv.FormatInt(*cluster.RegisteredContainerInstancesCount, 10),
					strconv.FormatInt(*cluster.ActiveServicesCount, 10),
					strconv.FormatInt(*cluster.RunningTasksCount, 10),
					strconv.FormatInt(*cluster.PendingTasksCount, 10),
				})
			}
			table.Render()
			return nil
		})
	var (
		argClusterName       string
		listServicesShowLink bool
		listServicesFilter   string
	)
	listServicesCommand := app.Command("services", "List services within the cluster")
	listServicesCommand.Arg("cluster", "Name of the cluster").Required().StringVar(&argClusterName)
	listServicesCommand.Flag("link", "Whether to render links to the AWS console").BoolVar(&listServicesShowLink)
	listServicesCommand.Flag("filter", "Service name to filter for, as a substring.").StringVar(&listServicesFilter)
	listServicesCommand.Action(func(ctx *kingpin.ParseContext) error {
		services := &ecs.DescribeServicesOutput{}
		fmt.Fprint(os.Stderr, "Found 0 services")
		err := svc.ListServicesPages(&ecs.ListServicesInput{Cluster: &argClusterName},
			func(page *ecs.ListServicesOutput, lastPage bool) bool {
				result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
					Cluster:  &argClusterName,
					Services: page.ServiceArns,
				})
				app.FatalIfError(err, "Could not describe services")
				services.Failures = append(services.Failures, result.Failures...)
				services.Services = append(services.Services, result.Services...)
				fmt.Fprintf(os.Stderr, "\rFound %v services", len(services.Services))
				return true
			})
		fmt.Fprint(os.Stderr, "\n")
		app.FatalIfError(err, "Could list services")
		ServiceSlice(services.Services).Sort()
		table := tablewriter.NewWriter(os.Stdout)
		header := []string{"Service Name", "Status", "Desired", "Running", "Pending"}
		if listServicesShowLink {
			header = append(header, "Link")
		}
		table.SetHeader(header)
		for _, service := range services.Services {
			if !strings.Contains(*service.ServiceName, listServicesFilter) {
				continue
			}
			row := []string{
				*service.ServiceName,
				*service.Status,
				strconv.FormatInt(*service.DesiredCount, 10),
				strconv.FormatInt(*service.RunningCount, 10),
				strconv.FormatInt(*service.PendingCount, 10),
			}
			if listServicesShowLink {
				row = append(row, ServiceLink(argClusterName, *service.ServiceName))
			}
			table.Append(row)
		}
		table.Render()
		PrintFailures(services.Failures)
		return nil
	})
	var (
		argServiceName            string
		describeServiceShowEvents bool
	)
	describeServiceCommand := app.Command("service", "Show details of a service")
	describeServiceCommand.Arg("cluster", "Name of the cluster").Required().StringVar(&argClusterName)
	describeServiceCommand.Arg("service", "Name of the service. This can be the full AWS service name, or the short one without the service- prefix and -<cluster> suffix").
		Required().StringVar(&argServiceName)
	describeServiceCommand.Flag("events", "Print service events").BoolVar(&describeServiceShowEvents)
	describeServiceCommand.Action(func(ctx *kingpin.ParseContext) error {
		result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  &argClusterName,
			Services: []*string{aws.String(FormatServiceName(argClusterName, argServiceName))},
		})
		app.FatalIfError(err, "Could not describe service")
		if len(result.Services) == 0 {
			app.Fatalf("Could not describe service")
		}
		service := result.Services[0]
		fmt.Println("Service")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		rows := [][]string{
			{"Name", *service.ServiceName},
			{"Status", *service.Status},
			{"Service ARN", *service.ServiceArn},
			{"Task Definition", *service.TaskDefinition},
			{"Desired Count", strconv.FormatInt(*service.DesiredCount, 10)},
			{"Running Count", strconv.FormatInt(*service.RunningCount, 10)},
			{"Pending Count", strconv.FormatInt(*service.PendingCount, 10)},
			{"Service Link", ServiceLink(argClusterName, *service.ServiceName)},
			{"Task Definition Link", TaskDefinitionLink(ParseARN(*service.TaskDefinition))},
		}
		table.AppendBulk(rows)
		if len(service.LoadBalancers) > 0 {
			lb := service.LoadBalancers[0]
			table.Append([]string{"LB Container Name", *lb.ContainerName})
			table.Append([]string{"LB Container Port", strconv.FormatInt(*lb.ContainerPort, 10)})
		}
		table.Render()
		tdr, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})
		app.FatalIfError(err, "Could not describe task definition")
		fmt.Println("Containers")
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Image", "CPU", "Memory", "Command"})
		for _, container := range tdr.TaskDefinition.ContainerDefinitions {
			command := []string{}
			for _, piece := range container.Command {
				command = append(command, *piece)
			}
			table.Append([]string{
				*container.Name,
				*container.Image,
				strconv.FormatInt(*container.Cpu, 10),
				strconv.FormatInt(*container.Memory, 10),
				strings.Join(command, " "),
			})
		}
		table.Render()

		if describeServiceShowEvents {
			ServiceEventSlice(service.Events).Sort()
			tmpl := `
Events:
{{- range . }}
{{formatTime .CreatedAt}}: {{.Message}}
{{- end }}
`
			t := template.New("events")
			t.Funcs(template.FuncMap{
				"formatTime": func(ts *time.Time) string {
					return ts.Format(time.RFC3339)
				},
			})
			t, err := t.Parse(tmpl)
			app.FatalIfError(err, "Failed to parse events template")
			t.Execute(os.Stdout, service.Events)
		}
		return nil
	})

	listTasksCommand := app.Command("tasks", "List tasks belonging to a service")
	listTasksCommand.Arg("cluster", "Name of the cluster").Required().StringVar(&argClusterName)
	listTasksCommand.Arg("service", "Name of the service. This can be the full AWS service name, or the short one without the service- prefix and -<cluster> suffix").
		Required().StringVar(&argServiceName)
	listTasksCommand.Action(func(ctx *kingpin.ParseContext) error {
		serviceName := FormatServiceName(argClusterName, argServiceName)
		runningTasks, err := getTasksArns(svc, argClusterName, serviceName, ecs.DesiredStatusRunning)
		app.FatalIfError(err, "Could not list tasks")
		stoppedTasks, err := getTasksArns(svc, argClusterName, serviceName, ecs.DesiredStatusStopped)
		app.FatalIfError(err, "Could not list tasks")
		if len(runningTasks) == 0 && len(stoppedTasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}
		var exampleTask *string
		if len(runningTasks) > 0 {
			exampleTask = runningTasks[0]
		} else {
			exampleTask = stoppedTasks[0]
		}
		tmpl := `
Running Tasks:
{{- range .RunningTasks }}
	{{.}}
{{- end }}

Stopped Tasks:
{{- range .StoppedTasks }}
	{{.}}
{{- end }}

Use the "task" command to get details of a task. For example:
	ecsq task {{.Cluster}} {{.ExampleTask}}
`
		t := template.New("list-tasks")
		t, err = t.Parse(tmpl)
		app.FatalIfError(err, "Could not parse task list template")
		err = t.Execute(os.Stdout, struct {
			Cluster      string
			RunningTasks []*string
			StoppedTasks []*string
			ExampleTask  *string
		}{
			Cluster:      argClusterName,
			RunningTasks: runningTasks,
			StoppedTasks: stoppedTasks,
			ExampleTask:  exampleTask,
		})
		app.FatalIfError(err, "Could not print tasks")
		return nil
	})
	var argTaskID string
	describeTaskCommand := app.Command("task", "Describe the given task. If a service name is provided instead, describes an arbitrary task for that service.")
	describeTaskCommand.Arg("cluster", "Name of the cluster").Required().StringVar(&argClusterName)
	describeTaskCommand.Arg("task or service", "ID or ARN of the task or name of service").Required().StringVar(&argTaskID)
	describeTaskCommand.Action(func(ctx *kingpin.ParseContext) error {
		if !isTaskARN(argTaskID) && !isUUID(argTaskID) {
			fmt.Println("Invalid task ID, assuming this is a service name. Looking up arbitrary task for service")
			serviceName := FormatServiceName(argClusterName, argTaskID)
			taskArns, err := getTasksArns(svc, argClusterName, serviceName, "RUNNING")
			if err != nil {
				fmt.Println("Error listing tasks", err)
				return nil
			}
			if len(taskArns) == 0 {
				fmt.Println("No tasks found for service ", serviceName)
				return nil
			}
			argTaskID = *taskArns[0]
		}
		task, err := getTaskDetail(svc, argClusterName, argTaskID)
		app.FatalIfError(err, "Could not describe task")
		containerInstanceResult, err := svc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            &argClusterName,
			ContainerInstances: []*string{task.ContainerInstanceArn},
		})
		var containerInstance *ecs.ContainerInstance
		app.FatalIfError(err, "Could not describe task container instance")
		if len(containerInstanceResult.Failures) > 0 {
			PrintFailures(containerInstanceResult.Failures)
			app.Fatalf("Could not describe task container instance")
		} else if len(containerInstanceResult.ContainerInstances) > 0 {
			containerInstance = containerInstanceResult.ContainerInstances[0]
		}
		svc := ec2.New(sess, &aws.Config{Region: aws.String("us-west-2")})
		ec2Result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{containerInstance.Ec2InstanceId},
		})
		app.FatalIfError(err, "Could not get EC2 instance")
		if len(ec2Result.Reservations) == 0 || len(ec2Result.Reservations[0].Instances) == 0 {
			app.Fatalf("Could not find EC2 instance %v", containerInstance.Ec2InstanceId)
		}
		ec2Instance := ec2Result.Reservations[0].Instances[0]

		table := tablewriter.NewWriter(os.Stdout)
		taskID := ParseARN(*task.TaskArn).Name
		taskDefinitionARN := ParseARN(*task.TaskDefinitionArn)
		taskDefinition := taskDefinitionARN.Name
		containerInstanceID := ParseARN(*task.ContainerInstanceArn).Name
		rows := [][]string{
			{"Task ID", taskID},
			{"Task ARN", *task.TaskArn},
			{"Task Definition", taskDefinition},
			{"Container Instance", containerInstanceID},
			{"EC2 Instance", *containerInstance.Ec2InstanceId},
			{"EC2 Instance Private IP", *ec2Instance.PrivateIpAddress},
			{"Task Link", TaskLink(argClusterName, taskID)},
			{"Task Definition Link", TaskDefinitionLink(taskDefinitionARN)},
			{"Container Instance Link", ContainerInstanceLink(argClusterName, containerInstanceID)},
			{"EC2 Instance Link", EC2InstanceLink(*containerInstance.Ec2InstanceId)},
		}
		table.AppendBulk(rows)
		fmt.Println("Details:")
		table.Render()

		fmt.Println("Containers:")
		table = tablewriter.NewWriter(os.Stdout)
		table.SetAutoMergeCells(true)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		for _, container := range task.Containers {
			table.Append([]string{*container.Name, "Status", *container.LastStatus})
			if *container.LastStatus == "STOPPED" || *container.LastStatus == "FAILED" {
				exitCode := ""
				if container.ExitCode != nil {
					exitCode = strconv.FormatInt(aws.Int64Value(container.ExitCode), 10)
				}
				table.Append([]string{*container.Name, "Exit Code", exitCode})
				table.Append([]string{*container.Name, "Reason", aws.StringValue(container.Reason)})
			}
			for _, network := range container.NetworkBindings {
				table.Append([]string{
					*container.Name,
					"Network - Container Port",
					strconv.FormatInt(*network.ContainerPort, 10),
				})
				link := *ec2Instance.PrivateIpAddress
				link += ":" + strconv.FormatInt(*network.HostPort, 10)
				table.Append([]string{
					*container.Name,
					"Network - External Link",
					link,
				})
			}
		}
		table.Render()
		return nil
	})
	var (
		flagContainerName string
		flagFormat        string
		flagFilter        string
	)
	containerEnvCommand := app.Command("container-env", "List environment variables for the task's container. Use --format to choose the output format")
	containerEnvCommand.Arg("cluster", "Name of the cluster").Required().StringVar(&argClusterName)
	containerEnvCommand.Arg("service", "Name of the service. This can be the full AWS service name, or the short one without the service- prefix and -<cluster> suffix").Required().StringVar(&argServiceName)
	containerEnvCommand.Flag("container", "Name of the container").StringVar(&flagContainerName)
	containerEnvCommand.Flag("format", "Format to render the environment variable in. The options are: export, shell, docker, table. Defaults to table").
		Default("table").EnumVar(&flagFormat, "export", "shell", "docker", "table")
	containerEnvCommand.Flag("filter", "Case-insensitive comma-separated list of variable names to drop").StringVar(&flagFilter)
	containerEnvCommand.Action(func(ctx *kingpin.ParseContext) error {
		task, err := getServiceDetail(svc, argClusterName, argServiceName)
		app.FatalIfError(err, "Could not describe service")
		result, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinition,
		})
		app.FatalIfError(err, "Could not describe task definition")
		taskDefinition := result.TaskDefinition
		var containerDefinition *ecs.ContainerDefinition
		if flagContainerName == "" && len(taskDefinition.ContainerDefinitions) > 1 {
			for _, c := range taskDefinition.ContainerDefinitions {
				fmt.Println("*", *c.Name)
			}
			app.Fatalf("Multiple containers found, choose one by name by setting --container")
		} else if flagContainerName == "" && len(taskDefinition.ContainerDefinitions) == 1 {
			containerDefinition = taskDefinition.ContainerDefinitions[0]
		} else {
			for _, c := range taskDefinition.ContainerDefinitions {
				if *c.Name == flagContainerName {
					containerDefinition = c
					break
				}
			}
		}
		if containerDefinition == nil {
			app.Fatalf("Container not found")
		}
		KeyValuePairSlice(containerDefinition.Environment).Sort()

		if flagFilter != "" {
			filters := map[string]bool{}
			for _, filter := range strings.Split(flagFilter, ",") {
				filters[strings.ToLower(strings.TrimSpace(filter))] = true
			}

			filtered := []*ecs.KeyValuePair{}
			for _, pair := range containerDefinition.Environment {
				if _, ok := filters[strings.ToLower(*pair.Name)]; !ok {
					filtered = append(filtered, pair)
				}
			}

			containerDefinition.Environment = filtered
		}

		if flagFormat == "table" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Value"})
			for _, env := range containerDefinition.Environment {
				table.Append([]string{*env.Name, *env.Value})
			}
			table.Render()
		} else if flagFormat == "shell" {
			envStr := ""
			for _, env := range containerDefinition.Environment {
				envStr += fmt.Sprintf("%v=\"%v\" ", *env.Name, *env.Value)
			}
			fmt.Println(envStr)
		} else if flagFormat == "export" {
			for _, env := range containerDefinition.Environment {
				fmt.Printf("export %v='%v'\n", *env.Name, *env.Value)
			}
		} else if flagFormat == "docker" {
			envStr := ""
			for _, env := range containerDefinition.Environment {
				envStr += fmt.Sprintf("-e%v=\"%v\" ", *env.Name, *env.Value)
			}
			fmt.Println(envStr)
		} else {
			app.Fatalf("Invalid format %v", flagFormat)
		}
		return nil
	})
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

// ARN contains the pieces of an AWS ARN
type ARN struct {
	Type     string
	Name     string
	Instance string
}

// ParseARN breaks a raw AWS ARN string into its pieces and returns an instance of the ARN struct
func ParseARN(s string) *ARN {
	arn := &ARN{}
	pieces := strings.Split(s, ":")
	typeName := strings.SplitN(pieces[5], "/", 2)
	arn.Type = typeName[0]
	if len(typeName) >= 2 {
		arn.Name = typeName[1]
	}
	if len(pieces) >= 7 {
		arn.Instance = pieces[6]
	}
	return arn
}

const reUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

func isTaskARN(s string) bool {
	return regexp.MustCompile(`arn:aws:ecs:[a-z]+-[a-z]+-\d:\d+:task/` + reUUID).MatchString(s)
}

func isUUID(s string) bool {
	return regexp.MustCompile(reUUID).MatchString(s)
}

func getTaskDetail(svc *ecs.ECS, clusterName, taskID string) (*ecs.Task, error) {
	result, err := svc.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: &clusterName,
		Tasks:   []*string{&taskID},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Failures) > 0 {
		return nil, errors.New(*result.Failures[0].Reason)
	}
	return result.Tasks[0], nil
}

func getServiceDetail(svc *ecs.ECS, clusterName, serviceName string) (*ecs.Service, error) {
	result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &clusterName,
		Services: []*string{aws.String(FormatServiceName(clusterName, serviceName))},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Failures) > 0 {
		return nil, errors.New(*result.Failures[0].Reason)
	}
	return result.Services[0], nil
}

// ServiceLink returns the URL to the ECS service on the AWS console
func ServiceLink(cluster, service string) string {
	tmpl := "https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/%v/services/%v/tasks"
	return fmt.Sprintf(tmpl, cluster, service)
}

// TaskLink returns the URL to the ECS task on the AWS console
func TaskLink(cluster, taskID string) string {
	tmpl := "https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/%v/tasks/%v"
	return fmt.Sprintf(tmpl, cluster, taskID)
}

// TaskDefinitionLink returns the URL to the ECS task definition on the AWS console.
func TaskDefinitionLink(taskDefinition *ARN) string {
	tmpl := "https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/taskDefinitions/%v/%v"
	return fmt.Sprintf(tmpl, taskDefinition.Name, taskDefinition.Instance)
}

// ContainerInstanceLink returns the URL to the ECS container instance on the AWS console.
func ContainerInstanceLink(cluster, containerInstance string) string {
	tmpl := "https://us-west-2.console.aws.amazon.com/ecs/home?region=us-west-2#/clusters/%v/containerInstances/%v"
	return fmt.Sprintf(tmpl, cluster, containerInstance)
}

// EC2InstanceLink returns the URL to the EC2 instance on the AWS console.
func EC2InstanceLink(ec2Instance string) string {
	tmpl := "https://us-west-2.console.aws.amazon.com/ec2/v2/home?region=us-west-2#Instances:instanceId=%v"
	return fmt.Sprintf(tmpl, ec2Instance)
}

// FormatServiceName parses a potentially short service name and returns the full service name
func FormatServiceName(cluster, service string) string {
	var (
		serviceNameExpansion = os.Getenv("ECSQ_SERVICE_NAME_EXPANSION")
		serviceNameTemplate  *template.Template
	)
	if serviceNameExpansion != "" {
		// First detect if service name has already been expanded.
		interpolateRegex := regexp.MustCompile("{{.*}}")
		alreadyExpandedRegex, err := regexp.Compile(interpolateRegex.ReplaceAllString(serviceNameExpansion, ".*"))
		if err == nil && alreadyExpandedRegex.MatchString(service) {
			return service
		}
		serviceNameTemplate, err = template.New("serviceName").Parse(serviceNameExpansion)
		if err != nil {
			panic(fmt.Errorf("Invalid ECSQ_SERVICE_NAME_EXPANSION template %v", err))
		}
		buffer := bytes.NewBuffer(nil)
		err = serviceNameTemplate.Execute(buffer, struct {
			Name    string
			Cluster string
		}{
			Name:    service,
			Cluster: cluster,
		})
		if err != nil {
			panic(fmt.Errorf("Invalid ECSQ_SERVICE_NAME_EXPANSION template %v", err))
		}
		return buffer.String()
	}
	return service
}

func getTasksArns(svc *ecs.ECS, clusterName, serviceName, status string) ([]*string, error) {
	tasks := []*string{}
	err := svc.ListTasksPages(&ecs.ListTasksInput{
		Cluster:       &clusterName,
		ServiceName:   &serviceName,
		DesiredStatus: aws.String(status),
	}, func(page *ecs.ListTasksOutput, lastPage bool) bool {
		tasks = append(tasks, page.TaskArns...)
		return true
	})
	return tasks, err
}

// PrintFailures prints failures from bulk commands
func PrintFailures(failures []*ecs.Failure) {
	if len(failures) == 0 {
		return
	}
	for _, failure := range failures {
		fmt.Printf("Failure for resource %v, reason: %v\n", *failure.Arn, *failure.Reason)
	}
}
