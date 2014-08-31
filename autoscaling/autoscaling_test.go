package autoscaling

import (
	"testing"
	"time"

	"github.com/motain/gocheck"

	"github.com/goamz/goamz/autoscaling/astest"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/testutil"
)

func Test(t *testing.T) {
	gocheck.TestingT(t)
}

var _ = gocheck.Suite(&S{})

type S struct {
	as *AutoScaling
}

var testServer = testutil.NewHTTPServer()

var mockTest bool

func (s *S) SetUpSuite(c *gocheck.C) {
	testServer.Start()
	auth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
	s.as = New(auth, aws.Region{AutoScalingEndpoint: testServer.URL})
}

func (s *S) TearDownTest(c *gocheck.C) {
	testServer.Flush()
}

func TestBasicGroupRequest(t *testing.T) {
	var as *AutoScaling
	awsAuth, err := aws.EnvAuth()
	if err != nil {
		mockTest = true
		t.Log("Running mock tests as AWS environment variables are not set")
		awsAuth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
		as = New(awsAuth, aws.Region{AutoScalingEndpoint: testServer.URL})
		testServer.Start()
		go testServer.WaitRequest()
		testServer.Response(200, nil, astest.BasicGroupResponse)
	} else {
		as = New(awsAuth, aws.USWest2)
	}

	groupResp, err := as.DescribeAutoScalingGroups(nil, 10, "")

	if err != nil {
		t.Fatal(err)
	}
	if len(groupResp.AutoScalingGroups) > 0 {
		firstGroup := groupResp.AutoScalingGroups[0]
		if len(firstGroup.AutoScalingGroupName) > 0 {
			t.Logf("Found AutoScaling group %s\n",
				firstGroup.AutoScalingGroupName)
		}
	}
	testServer.Flush()
}

func TestAutoScalingGroup(t *testing.T) {
	var as *AutoScaling
	// Launch configuration test config
	lcReq := new(CreateLaunchConfigurationParams)
	lcReq.LaunchConfigurationName = "LConf1"
	lcReq.ImageId = "ami-03e47533" // Octave debian ami
	lcReq.KernelId = "aki-98e26fa8"
	lcReq.KeyName = "testAWS" // Replace with valid key for your account
	lcReq.InstanceType = "m1.small"

	// CreateAutoScalingGroup params test config
	asgReq := new(CreateAutoScalingGroupParams)
	asgReq.AutoScalingGroupName = "ASGTest1"
	asgReq.LaunchConfigurationName = lcReq.LaunchConfigurationName
	asgReq.DefaultCooldown = 300
	asgReq.HealthCheckGracePeriod = 300
	asgReq.DesiredCapacity = 1
	asgReq.MinSize = 1
	asgReq.MaxSize = 5
	asgReq.AvailabilityZones = []string{"us-west-2a"}

	asg := new(AutoScalingGroup)
	asg.AutoScalingGroupName = "ASGTest1"
	asg.LaunchConfigurationName = lcReq.LaunchConfigurationName
	asg.DefaultCooldown = 300
	asg.HealthCheckGracePeriod = 300
	asg.DesiredCapacity = 1
	asg.MinSize = 1
	asg.MaxSize = 5
	asg.AvailabilityZones = []string{"us-west-2a"}

	asgUpdate := new(UpdateAutoScalingGroupParams)
	asgUpdate.AutoScalingGroupName = "ASGTest1"
	asgUpdate.DesiredCapacity = 1
	asgUpdate.MinSize = 1
	asgUpdate.MaxSize = 6

	awsAuth, err := aws.EnvAuth()
	if err != nil {
		mockTest = true
		t.Log("Running mock tests as AWS environment variables are not set")
		awsAuth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
		as = New(awsAuth, aws.Region{AutoScalingEndpoint: testServer.URL})
	} else {
		as = New(awsAuth, aws.USWest2)
	}

	// Create the launch configuration
	if mockTest {
		testServer.Response(200, nil, astest.CreateLaunchConfigurationResponse)
	}
	_, err = as.CreateLaunchConfiguration(lcReq)
	if err != nil {
		t.Fatal(err)
	}

	// Check that we can get the launch configuration details
	if mockTest {
		testServer.Response(200, nil, astest.DescribeLaunchConfigurationResponse)
	}
	_, err = as.DescribeLaunchConfigurations([]string{lcReq.LaunchConfigurationName}, 10, "")
	if err != nil {
		t.Fatal(err)
	}

	// Create the AutoScalingGroup
	if mockTest {
		testServer.Response(200, nil, astest.CreateAutoScalingGroupResponse)
	}
	_, err = as.CreateAutoScalingGroup(asgReq)
	if err != nil {
		t.Fatal(err)
	}

	// Check that we can get the autoscaling group details
	if mockTest {
		testServer.Response(200, nil, astest.DescribeAutoScalingGroupResponse)
	}
	_, err = as.DescribeAutoScalingGroups(nil, 10, "")
	if err != nil {
		t.Fatal(err)
	}

	// Suspend the scaling processes for the test AutoScalingGroup
	if mockTest {
		testServer.Response(200, nil, astest.SuspendProcessesResponse)
	}
	_, err = as.SuspendProcesses(asg.AutoScalingGroupName, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Resume scaling processes for the test AutoScalingGroup
	if mockTest {
		testServer.Response(200, nil, astest.ResumeProcessesResponse)
	}
	_, err = as.ResumeProcesses(asg.AutoScalingGroupName, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Change the desired capacity from 1 to 2. This will launch a second instance
	if mockTest {
		testServer.Response(200, nil, astest.SetDesiredCapacityResponse)
	}
	_, err = as.SetDesiredCapacity(asg.AutoScalingGroupName, 2, false)
	if err != nil {
		t.Fatal(err)
	}

	// Change the desired capacity from 2 to 1. This will terminate one of the instances
	if mockTest {
		testServer.Response(200, nil, astest.SetDesiredCapacityResponse)
	}

	_, err = as.SetDesiredCapacity(asg.AutoScalingGroupName, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	// Update the max capacity for the scaling group
	if mockTest {
		testServer.Response(200, nil, astest.UpdateAutoScalingGroupResponse)
	}
	_, err = as.UpdateAutoScalingGroup(asgUpdate)
	if err != nil {
		t.Fatal(err)
	}

	// Add a scheduled action to the group
	psar := new(PutScheduledUpdateGroupActionParams)
	psar.AutoScalingGroupName = asg.AutoScalingGroupName
	psar.MaxSize = 4
	psar.ScheduledActionName = "SATest1"
	psar.Recurrence = "30 0 1 1,6,12 *"
	if mockTest {
		testServer.Response(200, nil, astest.PutScheduledUpdateGroupActionResponse)
	}
	_, err = as.PutScheduledUpdateGroupAction(psar)
	if err != nil {
		t.Fatal(err)
	}

	// List the scheduled actions for the group
	sar := new(DescribeScheduledActionsParams)
	sar.AutoScalingGroupName = asg.AutoScalingGroupName
	if mockTest {
		testServer.Response(200, nil, astest.DescribeScheduledActionsResponse)
	}
	_, err = as.DescribeScheduledActions(sar)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the test scheduled action from the group
	if mockTest {
		testServer.Response(200, nil, astest.DeleteScheduledActionResponse)
	}
	_, err = as.DeleteScheduledAction(asg.AutoScalingGroupName, psar.ScheduledActionName)
	if err != nil {
		t.Fatal(err)
	}
	testServer.Flush()
}

// --------------------------------------------------------------------------
// Detailed Unit Tests

func (s *S) TestAttachInstances(c *gocheck.C) {
	testServer.Response(200, nil, AttachInstances)
	resp, err := s.as.AttachInstances("my-test-asg", []string{"i-21321afs", "i-baaffg23"})
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "AttachInstances")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("InstanceIds.member.1"), gocheck.Equals, "i-21321afs")
	c.Assert(values.Get("InstanceIds.member.2"), gocheck.Equals, "i-baaffg23")
	c.Assert(resp.RequestId, gocheck.Equals, "8d798a29-f083-11e1-bdfb-cb223EXAMPLE")
}

func (s *S) TestCreateAutoScalingGroup(c *gocheck.C) {
	testServer.Response(200, nil, CreateAutoScalingGroup)
	testServer.Response(200, nil, DeleteAutoScalingGroup)

	createAS := &CreateAutoScalingGroupParams{
		AutoScalingGroupName:    "my-test-asg",
		AvailabilityZones:       []string{"us-east-1a", "us-east-1b"},
		MinSize:                 3,
		MaxSize:                 3,
		DefaultCooldown:         600,
		DesiredCapacity:         0,
		LaunchConfigurationName: "my-test-lc",
		LoadBalancerNames:       []string{"elb-1", "elb-2"},
		Tags: []Tag{
			{
				Key:   "foo",
				Value: "bar",
			},
			{
				Key:   "baz",
				Value: "qux",
			},
		},
		VPCZoneIdentifier: "subnet-610acd08,subnet-530fc83a",
	}
	resp, err := s.as.CreateAutoScalingGroup(createAS)
	c.Assert(err, gocheck.IsNil)
	defer s.as.DeleteAutoScalingGroup(createAS.AutoScalingGroupName, true)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "CreateAutoScalingGroup")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("AvailabilityZones.member.1"), gocheck.Equals, "us-east-1a")
	c.Assert(values.Get("AvailabilityZones.member.2"), gocheck.Equals, "us-east-1b")
	c.Assert(values.Get("MinSize"), gocheck.Equals, "3")
	c.Assert(values.Get("MaxSize"), gocheck.Equals, "3")
	c.Assert(values.Get("DefaultCooldown"), gocheck.Equals, "600")
	c.Assert(values.Get("DesiredCapacity"), gocheck.Equals, "0")
	c.Assert(values.Get("LaunchConfigurationName"), gocheck.Equals, "my-test-lc")
	c.Assert(values.Get("LoadBalancerNames.member.1"), gocheck.Equals, "elb-1")
	c.Assert(values.Get("LoadBalancerNames.member.2"), gocheck.Equals, "elb-2")
	c.Assert(values.Get("Tags.member.1.Key"), gocheck.Equals, "foo")
	c.Assert(values.Get("Tags.member.1.Value"), gocheck.Equals, "bar")
	c.Assert(values.Get("Tags.member.2.Key"), gocheck.Equals, "baz")
	c.Assert(values.Get("Tags.member.2.Value"), gocheck.Equals, "qux")
	c.Assert(values.Get("VPCZoneIdentifier"), gocheck.Equals, "subnet-610acd08,subnet-530fc83a")
	c.Assert(resp.RequestId, gocheck.Equals, "8d798a29-f083-11e1-bdfb-cb223EXAMPLE")
}

func (s *S) TestCreateLaunchConfiguration(c *gocheck.C) {
	testServer.Response(200, nil, CreateLaunchConfiguration)
	testServer.Response(200, nil, DeleteLaunchConfiguration)

	launchConfig := &CreateLaunchConfigurationParams{
		LaunchConfigurationName:  "my-test-lc",
		AssociatePublicIpAddress: true,
		EbsOptimized:             true,
		SecurityGroups:           []string{"sec-grp1", "sec-grp2"},
		UserData:                 "1234",
		KeyName:                  "secretKeyPair",
		ImageId:                  "ami-0078da69",
		InstanceType:             "m1.small",
		SpotPrice:                "0.03",
		BlockDeviceMappings: []BlockDeviceMapping{
			{
				DeviceName:  "/dev/sda1",
				VirtualName: "ephemeral0",
			},
			{
				DeviceName:  "/dev/sdb",
				VirtualName: "ephemeral1",
			},
			{
				DeviceName: "/dev/sdf",
				Ebs: EBS{
					DeleteOnTermination: true,
					SnapshotId:          "snap-2a2b3c4d",
					VolumeSize:          100,
				},
			},
		},
		InstanceMonitoring: InstanceMonitoring{
			Enabled: true,
		},
	}
	resp, err := s.as.CreateLaunchConfiguration(launchConfig)
	c.Assert(err, gocheck.IsNil)
	defer s.as.DeleteLaunchConfiguration(launchConfig.LaunchConfigurationName)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "CreateLaunchConfiguration")
	c.Assert(values.Get("LaunchConfigurationName"), gocheck.Equals, "my-test-lc")
	c.Assert(values.Get("AssociatePublicIpAddress"), gocheck.Equals, "true")
	c.Assert(values.Get("EbsOptimized"), gocheck.Equals, "true")
	c.Assert(values.Get("SecurityGroups.member.1"), gocheck.Equals, "sec-grp1")
	c.Assert(values.Get("SecurityGroups.member.2"), gocheck.Equals, "sec-grp2")
	c.Assert(values.Get("UserData"), gocheck.Equals, "MTIzNA==")
	c.Assert(values.Get("KeyName"), gocheck.Equals, "secretKeyPair")
	c.Assert(values.Get("ImageId"), gocheck.Equals, "ami-0078da69")
	c.Assert(values.Get("InstanceType"), gocheck.Equals, "m1.small")
	c.Assert(values.Get("SpotPrice"), gocheck.Equals, "0.03")
	c.Assert(values.Get("BlockDeviceMappings.member.1.DeviceName"), gocheck.Equals, "/dev/sda1")
	c.Assert(values.Get("BlockDeviceMappings.member.1.VirtualName"), gocheck.Equals, "ephemeral0")
	c.Assert(values.Get("BlockDeviceMappings.member.2.DeviceName"), gocheck.Equals, "/dev/sdb")
	c.Assert(values.Get("BlockDeviceMappings.member.2.VirtualName"), gocheck.Equals, "ephemeral1")
	c.Assert(values.Get("BlockDeviceMappings.member.3.DeviceName"), gocheck.Equals, "/dev/sdf")
	c.Assert(values.Get("BlockDeviceMappings.member.3.Ebs.DeleteOnTermination"), gocheck.Equals, "true")
	c.Assert(values.Get("BlockDeviceMappings.member.3.Ebs.SnapshotId"), gocheck.Equals, "snap-2a2b3c4d")
	c.Assert(values.Get("BlockDeviceMappings.member.3.Ebs.VolumeSize"), gocheck.Equals, "100")
	c.Assert(values.Get("InstanceMonitoring.Enabled"), gocheck.Equals, "true")
	c.Assert(resp.RequestId, gocheck.Equals, "7c6e177f-f082-11e1-ac58-3714bEXAMPLE")
}

func (s *S) TestCreateOrUpdateTags(c *gocheck.C) {
	testServer.Response(200, nil, CreateOrUpdateTags)
	tags := []Tag{
		{
			Key:        "foo",
			Value:      "bar",
			ResourceId: "my-test-asg",
		},
		{
			Key:               "baz",
			Value:             "qux",
			ResourceId:        "my-test-asg",
			PropagateAtLaunch: true,
		},
	}
	resp, err := s.as.CreateOrUpdateTags(tags)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "CreateOrUpdateTags")
	c.Assert(values.Get("Tags.member.1.Key"), gocheck.Equals, "foo")
	c.Assert(values.Get("Tags.member.1.Value"), gocheck.Equals, "bar")
	c.Assert(values.Get("Tags.member.1.ResourceId"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("Tags.member.2.Key"), gocheck.Equals, "baz")
	c.Assert(values.Get("Tags.member.2.Value"), gocheck.Equals, "qux")
	c.Assert(values.Get("Tags.member.2.ResourceId"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("Tags.member.2.PropagateAtLaunch"), gocheck.Equals, "true")
	c.Assert(resp.RequestId, gocheck.Equals, "b0203919-bf1b-11e2-8a01-13263EXAMPLE")
}

func (s *S) TestDeleteAutoScalingGroup(c *gocheck.C) {
	testServer.Response(200, nil, DeleteAutoScalingGroup)
	resp, err := s.as.DeleteAutoScalingGroup("my-test-asg", true)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DeleteAutoScalingGroup")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(resp.RequestId, gocheck.Equals, "70a76d42-9665-11e2-9fdf-211deEXAMPLE")
}

func (s *S) TestDeleteAutoScalingGroupWithExistingInstances(c *gocheck.C) {
	testServer.Response(400, nil, DeleteAutoScalingGroupError)
	resp, err := s.as.DeleteAutoScalingGroup("my-test-asg", false)
	testServer.WaitRequest()
	c.Assert(resp, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	e, ok := err.(*Error)
	if !ok {
		c.Errorf("Unable to unmarshal error into AWS Autoscaling Error")
	}
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(e.Message, gocheck.Equals, "You cannot delete an AutoScalingGroup while there are instances or pending Spot instance request(s) still in the group.")
	c.Assert(e.Code, gocheck.Equals, "ResourceInUse")
	c.Assert(e.StatusCode, gocheck.Equals, 400)
	c.Assert(e.RequestId, gocheck.Equals, "70a76d42-9665-11e2-9fdf-211deEXAMPLE")
}

func (s *S) TestDeleteLaunchConfiguration(c *gocheck.C) {
	testServer.Response(200, nil, DeleteLaunchConfiguration)
	resp, err := s.as.DeleteLaunchConfiguration("my-test-lc")
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DeleteLaunchConfiguration")
	c.Assert(values.Get("LaunchConfigurationName"), gocheck.Equals, "my-test-lc")
	c.Assert(resp.RequestId, gocheck.Equals, "7347261f-97df-11e2-8756-35eEXAMPLE")
}

func (s *S) TestDeleteLaunchConfigurationInUse(c *gocheck.C) {
	testServer.Response(400, nil, DeleteLaunchConfigurationInUse)
	resp, err := s.as.DeleteLaunchConfiguration("my-test-lc")
	testServer.WaitRequest()
	c.Assert(resp, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	e, ok := err.(*Error)
	if !ok {
		c.Errorf("Unable to unmarshal error into AWS Autoscaling Error")
	}
	c.Logf("%v %v %v", e.Code, e.Message, e.RequestId)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(e.Message, gocheck.Equals, "Cannot delete launch configuration my-test-lc because it is attached to AutoScalingGroup test")
	c.Assert(e.Code, gocheck.Equals, "ResourceInUse")
	c.Assert(e.StatusCode, gocheck.Equals, 400)
	c.Assert(e.RequestId, gocheck.Equals, "7347261f-97df-11e2-8756-35eEXAMPLE")
}

func (s *S) TestDeleteTags(c *gocheck.C) {
	testServer.Response(200, nil, DeleteTags)
	tags := []Tag{
		{
			Key:        "foo",
			Value:      "bar",
			ResourceId: "my-test-asg",
		},
		{
			Key:               "baz",
			Value:             "qux",
			ResourceId:        "my-test-asg",
			PropagateAtLaunch: true,
		},
	}
	resp, err := s.as.DeleteTags(tags)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DeleteTags")
	c.Assert(values.Get("Tags.member.1.Key"), gocheck.Equals, "foo")
	c.Assert(values.Get("Tags.member.1.Value"), gocheck.Equals, "bar")
	c.Assert(values.Get("Tags.member.1.ResourceId"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("Tags.member.2.Key"), gocheck.Equals, "baz")
	c.Assert(values.Get("Tags.member.2.Value"), gocheck.Equals, "qux")
	c.Assert(values.Get("Tags.member.2.ResourceId"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("Tags.member.2.PropagateAtLaunch"), gocheck.Equals, "true")
	c.Assert(resp.RequestId, gocheck.Equals, "b0203919-bf1b-11e2-8a01-13263EXAMPLE")
}

func (s *S) TestDescribeAccountLimits(c *gocheck.C) {
	testServer.Response(200, nil, DescribeAccountLimits)

	resp, err := s.as.DescribeAccountLimits()
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeAccountLimits")
	c.Assert(resp.RequestId, gocheck.Equals, "a32bd184-519d-11e3-a8a4-c1c467cbcc3b")
	c.Assert(resp.MaxNumberOfAutoScalingGroups, gocheck.Equals, int64(20))
	c.Assert(resp.MaxNumberOfLaunchConfigurations, gocheck.Equals, int64(100))

}

func (s *S) TestDescribeAdjustmentTypes(c *gocheck.C) {
	testServer.Response(200, nil, DescribeAdjustmentTypes)
	resp, err := s.as.DescribeAdjustmentTypes()
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeAdjustmentTypes")
	c.Assert(resp.RequestId, gocheck.Equals, "cc5f0337-b694-11e2-afc0-6544dEXAMPLE")
	c.Assert(resp.AdjustmentTypes, gocheck.DeepEquals, []AdjustmentType{{"ChangeInCapacity"}, {"ExactCapacity"}, {"PercentChangeInCapacity"}})
}

func (s *S) TestDescribeAutoScalingGroups(c *gocheck.C) {
	testServer.Response(200, nil, DescribeAutoScalingGroups)
	resp, err := s.as.DescribeAutoScalingGroups([]string{"my-test-asg-lbs"}, 0, "")
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	t, _ := time.Parse(time.RFC3339, "2013-05-06T17:47:15.107Z")
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeAutoScalingGroups")
	c.Assert(values.Get("AutoScalingGroupNames.member.1"), gocheck.Equals, "my-test-asg-lbs")

	expected := &DescribeAutoScalingGroupsResp{
		AutoScalingGroups: []AutoScalingGroup{
			{
				AutoScalingGroupName: "my-test-asg-lbs",
				Tags: []Tag{
					{
						Key:               "foo",
						Value:             "bar",
						ResourceId:        "my-test-asg-lbs",
						PropagateAtLaunch: true,
						ResourceType:      "auto-scaling-group",
					},
					{
						Key:               "baz",
						Value:             "qux",
						ResourceId:        "my-test-asg-lbs",
						PropagateAtLaunch: true,
						ResourceType:      "auto-scaling-group",
					},
				},
				Instances: []Instance{
					{
						AvailabilityZone:        "us-east-1b",
						HealthStatus:            "Healthy",
						InstanceId:              "i-zb1f313",
						LaunchConfigurationName: "my-test-lc",
						LifecycleState:          "InService",
					},
					{
						AvailabilityZone:        "us-east-1a",
						HealthStatus:            "Healthy",
						InstanceId:              "i-90123adv",
						LaunchConfigurationName: "my-test-lc",
						LifecycleState:          "InService",
					},
				},
				HealthCheckType:         "ELB",
				CreatedTime:             t,
				LaunchConfigurationName: "my-test-lc",
				DesiredCapacity:         2,
				AvailabilityZones:       []string{"us-east-1b", "us-east-1a"},
				LoadBalancerNames:       []string{"my-test-asg-loadbalancer"},
				MinSize:                 2,
				MaxSize:                 10,
				VPCZoneIdentifier:       "subnet-32131da1,subnet-1312dad2",
				HealthCheckGracePeriod:  120,
				DefaultCooldown:         300,
				AutoScalingGroupARN:     "arn:aws:autoscaling:us-east-1:803981987763:autoScalingGroup:ca861182-c8f9-4ca7-b1eb-cd35505f5ebb:autoScalingGroupName/my-test-asg-lbs",
				TerminationPolicies:     []string{"Default"},
			},
		},
		RequestId: "0f02a07d-b677-11e2-9eb0-dd50EXAMPLE",
	}
	c.Assert(resp, gocheck.DeepEquals, expected)
}

func (s *S) TestDescribeAutoScalingInstances(c *gocheck.C) {
	testServer.Response(200, nil, DescribeAutoScalingInstances)
	resp, err := s.as.DescribeAutoScalingInstances([]string{"i-78e0d40b"}, 0, "")
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeAutoScalingInstances")
	c.Assert(resp.RequestId, gocheck.Equals, "df992dc3-b72f-11e2-81e1-750aa6EXAMPLE")
	c.Assert(resp.AutoScalingInstances, gocheck.DeepEquals, []Instance{
		{
			AutoScalingGroupName:    "my-test-asg",
			AvailabilityZone:        "us-east-1a",
			HealthStatus:            "Healthy",
			InstanceId:              "i-78e0d40b",
			LaunchConfigurationName: "my-test-lc",
			LifecycleState:          "InService",
		},
	})
}

func (s *S) TestDescribeLaunchConfigurations(c *gocheck.C) {
	testServer.Response(200, nil, DescribeLaunchConfigurations)
	resp, err := s.as.DescribeLaunchConfigurations([]string{"my-test-lc"}, 0, "")
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	t, _ := time.Parse(time.RFC3339, "2013-01-21T23:04:42.200Z")
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeLaunchConfigurations")
	c.Assert(values.Get("LaunchConfigurationNames.member.1"), gocheck.Equals, "my-test-lc")
	expected := &DescribeLaunchConfigurationsResp{
		LaunchConfigurations: []LaunchConfiguration{
			{
				AssociatePublicIpAddress: true,
				BlockDeviceMappings: []BlockDeviceMapping{
					{
						DeviceName:  "/dev/sdb",
						VirtualName: "ephemeral0",
					},
					{
						DeviceName: "/dev/sdf",
						Ebs: EBS{
							SnapshotId:          "snap-XXXXYYY",
							VolumeSize:          100,
							Iops:                50,
							VolumeType:          "io1",
							DeleteOnTermination: true,
						},
					},
				},
				EbsOptimized:            false,
				CreatedTime:             t,
				LaunchConfigurationName: "my-test-lc",
				InstanceType:            "m1.small",
				ImageId:                 "ami-514ac838",
				InstanceMonitoring:      InstanceMonitoring{Enabled: true},
				LaunchConfigurationARN:  "arn:aws:autoscaling:us-east-1:803981987763:launchConfiguration:9dbbbf87-6141-428a-a409-0752edbe6cad:launchConfigurationName/my-test-lc",
			},
		},
		RequestId: "d05a22f8-b690-11e2-bf8e-2113fEXAMPLE",
	}
	c.Assert(resp, gocheck.DeepEquals, expected)
}

func (s *S) DescribeScheduledActions(c *gocheck.C) {
	testServer.Response(200, nil, DescribeScheduledActionsResponse)
	st, _ := time.Parse(time.RFC3339, "2014-06-01T00:30:00Z")
	request := &DescribeScheduledActionsParams{
		AutoScalingGroupName: "ASGTest1",
		MaxRecords:           1,
		StartTime:            st,
	}
	resp, err := s.as.DescribeScheduledActions(request)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "DescribeScheduledActions")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "ASGTest1")
	c.Assert(values.Get("ScheduledActionArn"), gocheck.Equals, "arn:aws:autoscaling:us-west-2:193024542802:scheduledUpdateGroupAction:61f68b2c-bde3-4316-9a81-eb95dc246509:autoScalingGroupName/ASGTest1:scheduledActionName/SATest1")
	c.Assert(values.Get("ScheduledActionName"), gocheck.Equals, "SATest1")
	c.Assert(values.Get("Recurrence"), gocheck.Equals, "30 0 1 1,6,12 *")
	c.Assert(values.Get("MaxSize"), gocheck.Equals, "4")
	c.Assert(values.Get("StartTime"), gocheck.Equals, "2014-06-01T00:30:00Z")
	c.Assert(values.Get("Time"), gocheck.Equals, "2014-06-01T00:30:00Z")
	c.Assert(resp.RequestId, gocheck.Equals, "0eb4217f-8421-11e3-9233-7100ef811766")
}

func (s *S) TestUpdateAutoScalingGroup(c *gocheck.C) {
	testServer.Response(200, nil, UpdateAutoScalingGroup)

	asg := &UpdateAutoScalingGroupParams{
		AutoScalingGroupName:    "my-test-asg",
		AvailabilityZones:       []string{"us-east-1a", "us-east-1b"},
		MinSize:                 3,
		MaxSize:                 3,
		DefaultCooldown:         600,
		DesiredCapacity:         3,
		LaunchConfigurationName: "my-test-lc",
		VPCZoneIdentifier:       "subnet-610acd08,subnet-530fc83a",
	}
	resp, err := s.as.UpdateAutoScalingGroup(asg)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "UpdateAutoScalingGroup")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("AvailabilityZones.member.1"), gocheck.Equals, "us-east-1a")
	c.Assert(values.Get("AvailabilityZones.member.2"), gocheck.Equals, "us-east-1b")
	c.Assert(values.Get("MinSize"), gocheck.Equals, "3")
	c.Assert(values.Get("MaxSize"), gocheck.Equals, "3")
	c.Assert(values.Get("DefaultCooldown"), gocheck.Equals, "600")
	c.Assert(values.Get("DesiredCapacity"), gocheck.Equals, "3")
	c.Assert(values.Get("LaunchConfigurationName"), gocheck.Equals, "my-test-lc")
	c.Assert(values.Get("VPCZoneIdentifier"), gocheck.Equals, "subnet-610acd08,subnet-530fc83a")
	c.Assert(resp.RequestId, gocheck.Equals, "8d798a29-f083-11e1-bdfb-cb223EXAMPLE")
}

func (s *S) TestResumeProcesses(c *gocheck.C) {
	testServer.Response(200, nil, ResumeProcesses)
	resp, err := s.as.ResumeProcesses("my-test-asg", []string{"Launch", "Terminate"})
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "ResumeProcesses")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("ScalingProcesses.member.1"), gocheck.Equals, "Launch")
	c.Assert(values.Get("ScalingProcesses.member.2"), gocheck.Equals, "Terminate")
	c.Assert(resp.RequestId, gocheck.Equals, "8d798a29-f083-11e1-bdfb-cb223EXAMPLE")

}

func (s *S) TestPutScheduledUpdateGroupAction(c *gocheck.C) {
	testServer.Response(200, nil, PutScheduledUpdateGroupAction)
	st, _ := time.Parse(time.RFC3339, "2013-05-25T08:00:00Z")
	request := &PutScheduledUpdateGroupActionParams{
		AutoScalingGroupName: "my-test-asg",
		DesiredCapacity:      3,
		ScheduledActionName:  "ScaleUp",
		StartTime:            st,
	}
	resp, err := s.as.PutScheduledUpdateGroupAction(request)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "PutScheduledUpdateGroupAction")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("ScheduledActionName"), gocheck.Equals, "ScaleUp")
	c.Assert(values.Get("DesiredCapacity"), gocheck.Equals, "3")
	c.Assert(values.Get("StartTime"), gocheck.Equals, "2013-05-25T08:00:00Z")
	c.Assert(resp.RequestId, gocheck.Equals, "3bc8c9bc-6a62-11e2-8a51-4b8a1EXAMPLE")
}

func (s *S) TestSetDesiredCapacity(c *gocheck.C) {
	testServer.Response(200, nil, SetDesiredCapacity)
	resp, err := s.as.SetDesiredCapacity("my-test-asg", 3, true)
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "SetDesiredCapacity")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("HonorCooldown"), gocheck.Equals, "true")
	c.Assert(values.Get("DesiredCapacity"), gocheck.Equals, "3")
	c.Assert(resp.RequestId, gocheck.Equals, "9fb7e2db-6998-11e2-a985-57c82EXAMPLE")
}

func (s *S) TestSuspendProcesses(c *gocheck.C) {
	testServer.Response(200, nil, SuspendProcesses)
	resp, err := s.as.SuspendProcesses("my-test-asg", []string{"Launch", "Terminate"})
	c.Assert(err, gocheck.IsNil)
	values := testServer.WaitRequest().PostForm
	c.Assert(values.Get("Version"), gocheck.Equals, "2011-01-01")
	c.Assert(values.Get("Action"), gocheck.Equals, "SuspendProcesses")
	c.Assert(values.Get("AutoScalingGroupName"), gocheck.Equals, "my-test-asg")
	c.Assert(values.Get("ScalingProcesses.member.1"), gocheck.Equals, "Launch")
	c.Assert(values.Get("ScalingProcesses.member.2"), gocheck.Equals, "Terminate")
	c.Assert(resp.RequestId, gocheck.Equals, "8d798a29-f083-11e1-bdfb-cb223EXAMPLE")
}
