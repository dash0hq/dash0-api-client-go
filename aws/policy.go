package aws

const (
	viewOnlyAccessPolicyArn = "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"
	customPolicyName        = "Dash0ReadOnly"
)

// readOnlyCustomPolicyJSON is the pre-marshaled custom inline policy for read-only resource discovery.
var readOnlyCustomPolicyJSON = mustMarshalJSON(map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []map[string]interface{}{
		{
			"Effect": "Allow",
			"Action": []string{
				"resource-explorer-2:Search",
				"resource-explorer-2:GetView",
			},
			"Resource": "*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"tag:GetResources",
				"tag:GetTagKeys",
				"tag:GetTagValues",
			},
			"Resource": "*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"lambda:GetFunction",
				"lambda:GetFunctionConfiguration",
			},
			"Resource": "*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"eks:ListClusters",
				"eks:DescribeCluster",
				"eks:ListNodegroups",
				"eks:DescribeNodegroup",
				"eks:ListFargateProfiles",
				"eks:DescribeFargateProfile",
				"eks:ListAddons",
				"eks:DescribeAddon",
			},
			"Resource": "*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"appsync:ListGraphqlApis",
				"appsync:GetGraphqlApi",
				"appsync:GetSchemaCreationStatus",
				"appsync:GetIntrospectionSchema",
				"appsync:ListDataSources",
				"appsync:ListResolvers",
				"appsync:ListFunctions",
				"appsync:ListTagsForResource",
			},
			"Resource": "*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"xray:GetTraceSegmentDestination",
				"xray:GetIndexingRules",
			},
			"Resource": "*",
		},
	},
})

// instrumentationPolicyJSON is the pre-marshaled policy for Lambda auto-instrumentation.
var instrumentationPolicyJSON = mustMarshalJSON(map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []map[string]interface{}{
		{
			"Effect": "Allow",
			"Action": []string{
				"lambda:GetFunctionConfiguration",
				"lambda:UpdateFunctionConfiguration",
			},
			"Resource": "arn:aws:lambda:*:*:function:*",
		},
		{
			"Effect": "Allow",
			"Action": []string{
				"ec2:DescribeRouteTables",
				"ec2:DescribeSecurityGroups",
				"ec2:DescribeVpcAttribute",
				"lambda:GetLayerVersion",
				"lambda:GetLayerVersionPolicy",
			},
			"Resource": "*",
		},
	},
})
