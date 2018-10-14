package resource

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/pkg/errors"
)

// AWSClient stores all the clients of AWS
// services including their sessions.
type AWSClient struct {
	EC2conn ec2iface.EC2API
	ASconn  autoscalingiface.AutoScalingAPI
	ELBconn elbiface.ELBAPI
	R53conn route53iface.Route53API
	CFconn  cloudformationiface.CloudFormationAPI
	EFSconn efsiface.EFSAPI
	IAMconn iamiface.IAMAPI
	KMSconn kmsiface.KMSAPI
	S3conn  s3iface.S3API
	STSconn stsiface.STSAPI
}

// APIDesc stores the necessary information about
// a single resource type (identified by its terraform type)
// to list and delete resources of that type via the go-aws-sdk
// and Terraform AWS provider API.
type APIDesc struct {
	TerraformType      string
	DescribeOutputName []string
	DeleteID           string
	Describe           interface{}
	DescribeInput      interface{}
	Select             func(Resources, interface{}, Filter, *AWSClient) []Resources
}

// Resources is a list of AWS resources.
type Resources []*Resource

// Resource contains information about
// a single AWS resource.
type Resource struct {
	Type  string // we use the terraform type to identify the resource type
	ID    string
	Attrs map[string]string
	Tags  map[string]string
}

// Supported returns for all supported
// resource types the API information
// to list (go-sdk API) and delete (AWS Terraform provider API)
// corresponding resources.
func Supported(c *AWSClient) []APIDesc {
	return []APIDesc{
		{
			"aws_autoscaling_group",
			[]string{"AutoScalingGroups"},
			"AutoScalingGroupName",
			c.ASconn.DescribeAutoScalingGroups,
			&autoscaling.DescribeAutoScalingGroupsInput{},
			filterGeneric,
		},
		{
			"aws_launch_configuration",
			[]string{"LaunchConfigurations"},
			"LaunchConfigurationName",
			c.ASconn.DescribeLaunchConfigurations,
			&autoscaling.DescribeLaunchConfigurationsInput{},
			filterGeneric,
		},
		{
			"aws_instance",
			[]string{"Reservations", "Instances"},
			"InstanceId",
			c.EC2conn.DescribeInstances,
			&ec2.DescribeInstancesInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("instance-state-name"),
						Values: []*string{
							aws.String("pending"), aws.String("running"),
							aws.String("stopping"), aws.String("stopped"),
						},
					},
				},
			},
			filterGeneric,
		},
		{
			"aws_key_pair",
			[]string{"KeyPairs"},
			"KeyName",
			c.EC2conn.DescribeKeyPairs,
			&ec2.DescribeKeyPairsInput{},
			filterGeneric,
		},
		{
			"aws_elb",
			[]string{"LoadBalancerDescriptions"},
			"LoadBalancerName",
			c.ELBconn.DescribeLoadBalancers,
			&elb.DescribeLoadBalancersInput{},
			filterGeneric,
		},
		{
			"aws_vpc_endpoint",
			[]string{"VpcEndpoints"},
			"VpcEndpointId",
			c.EC2conn.DescribeVpcEndpoints,
			&ec2.DescribeVpcEndpointsInput{},
			filterGeneric,
		},
		{
			// TODO support tags
			"aws_nat_gateway",
			[]string{"NatGateways"},
			"NatGatewayId",
			c.EC2conn.DescribeNatGateways,
			&ec2.DescribeNatGatewaysInput{
				Filter: []*ec2.Filter{
					{
						Name: aws.String("state"),
						Values: []*string{
							aws.String("available"),
						},
					},
				},
			},
			filterGeneric,
		},
		{
			"aws_cloudformation_stack",
			[]string{"Stacks"},
			"StackId",
			c.CFconn.DescribeStacks,
			&cloudformation.DescribeStacksInput{},
			filterGeneric,
		},
		{
			"aws_route53_zone",
			[]string{"HostedZones"},
			"Id",
			c.R53conn.ListHostedZones,
			&route53.ListHostedZonesInput{},
			filterGeneric,
		},
		{
			"aws_efs_file_system",
			[]string{"FileSystems"},
			"FileSystemId",
			c.EFSconn.DescribeFileSystems,
			&efs.DescribeFileSystemsInput{},
			filterEfsFileSystem,
		},
		// Elastic network interface (ENI) resource
		// sort by owner of the network interface?
		// support tags
		// attached to subnet
		{
			"aws_network_interface",
			[]string{"NetworkInterfaces"},
			"NetworkInterfaceId",
			c.EC2conn.DescribeNetworkInterfaces,
			&ec2.DescribeNetworkInterfacesInput{},
			filterGeneric,
		},
		{
			"aws_eip",
			[]string{"Addresses"},
			"AllocationId",
			c.EC2conn.DescribeAddresses,
			&ec2.DescribeAddressesInput{},
			filterGeneric,
		},
		{
			"aws_internet_gateway",
			[]string{"InternetGateways"},
			"InternetGatewayId",
			c.EC2conn.DescribeInternetGateways,
			&ec2.DescribeInternetGatewaysInput{},
			filterGeneric,
		},
		{
			"aws_subnet",
			[]string{"Subnets"},
			"SubnetId",
			c.EC2conn.DescribeSubnets,
			&ec2.DescribeSubnetsInput{},
			filterGeneric,
		},
		{
			"aws_route_table",
			[]string{"RouteTables"},
			"RouteTableId",
			c.EC2conn.DescribeRouteTables,
			&ec2.DescribeRouteTablesInput{},
			filterGeneric,
		},
		{
			"aws_security_group",
			[]string{"SecurityGroups"},
			"GroupId",
			c.EC2conn.DescribeSecurityGroups,
			&ec2.DescribeSecurityGroupsInput{},
			filterGeneric,
		},
		{
			"aws_network_acl",
			[]string{"NetworkAcls"},
			"NetworkAclId",
			c.EC2conn.DescribeNetworkAcls,
			&ec2.DescribeNetworkAclsInput{},
			filterGeneric,
		},
		{
			"aws_vpc",
			[]string{"Vpcs"},
			"VpcId",
			c.EC2conn.DescribeVpcs,
			&ec2.DescribeVpcsInput{},
			filterGeneric,
		},
		{
			"aws_iam_policy",
			[]string{"Policies"},
			"Arn",
			c.IAMconn.ListPolicies,
			&iam.ListPoliciesInput{},
			filterIamPolicy,
		},
		{
			"aws_iam_group",
			[]string{"Groups"},
			"GroupName",
			c.IAMconn.ListGroups,
			&iam.ListGroupsInput{},
			filterGeneric,
		},
		{
			"aws_iam_user",
			[]string{"Users"},
			"UserName",
			c.IAMconn.ListUsers,
			&iam.ListUsersInput{},
			filterIamUser,
		},
		{
			"aws_iam_role",
			[]string{"Roles"},
			"RoleName",
			c.IAMconn.ListRoles,
			&iam.ListRolesInput{},
			filterGeneric,
		},
		{
			"aws_iam_instance_profile",
			[]string{"InstanceProfiles"},
			"InstanceProfileName",
			c.IAMconn.ListInstanceProfiles,
			&iam.ListInstanceProfilesInput{},
			filterGeneric,
		},
		{
			"aws_kms_alias",
			[]string{"Aliases"},
			"AliasName",
			c.KMSconn.ListAliases,
			&kms.ListAliasesInput{},
			filterGeneric,
		},
		{
			"aws_kms_key",
			[]string{"Keys"},
			"KeyId",
			c.KMSconn.ListKeys,
			&kms.ListKeysInput{},
			filterKmsKeys,
		},
		{
			"aws_s3_bucket",
			[]string{"Buckets"},
			"Name",
			c.S3conn.ListBuckets,
			&s3.ListBucketsInput{},
			filterGeneric,
		},
		{
			"aws_ebs_snapshot",
			[]string{"Snapshots"},
			"SnapshotId",
			c.EC2conn.DescribeSnapshots,
			&ec2.DescribeSnapshotsInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("owner-id"),
						Values: []*string{
							accountID(c),
						},
					},
				},
			},
			filterGeneric,
		},
		{
			"aws_ebs_volume",
			[]string{"Volumes"},
			"VolumeId",
			c.EC2conn.DescribeVolumes,
			&ec2.DescribeVolumesInput{},
			filterGeneric,
		},
		{
			"aws_ami",
			[]string{"Images"},
			"ImageId",
			c.EC2conn.DescribeImages,
			&ec2.DescribeImagesInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("owner-id"),
						Values: []*string{
							accountID(c),
						},
					},
				},
			},
			filterGeneric,
		},
	}
}

// getSupported returns the apiDesc by the name of
// a given resource type
func getSupported(resType string, c *AWSClient) (APIDesc, error) {
	for _, apiDesc := range Supported(c) {
		if apiDesc.TerraformType == resType {
			return apiDesc, nil
		}
	}
	return APIDesc{}, errors.Errorf("no APIDesc found for resource type %s", resType)
}

// accountID returns the account ID of the AWS account
// for the currently used credentials or AWS profile, resp.
func accountID(c *AWSClient) *string {
	res, err := c.STSconn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal(err)
	}
	return res.Account
}
