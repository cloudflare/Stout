package fs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Create new IAM user upon using the 'create' command, '--no-user' flag disables this
func CreateS3User(iamSession *iam.IAM, domain string) (key iam.AccessKey, err error) {
	name := domain + "_deploy"

	_, err = iamSession.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(name),
		Path:     aws.String("/"),
	})
	if err != nil && err.Error() == iam.ErrCodeEntityAlreadyExistsException {
		return
	}

	// user policy that only allows access to the specified bucket
	_, err = iamSession.PutUserPolicy(&iam.PutUserPolicyInput{
		PolicyDocument: aws.String(`{
					"Version": "2012-10-17",
					"Statement": [
						{
							"Effect": "Allow",
							"Action": [
								"s3:DeleteObject",
								"s3:ListBucket",
								"s3:PutObject",
								"s3:PutObjectAcl",
								"s3:GetObject"
							],
							"Resource": [
								"arn:aws:s3:::` + domain + `", "arn:aws:s3:::` + domain + `/*"
							]
						}
					]
				}`),
		PolicyName: aws.String(name),
		UserName:   aws.String(name),
	})
	if err != nil {
		return
	}

	keyResp, err := iamSession.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(name),
	})
	if err != nil {
		return
	}

	return *keyResp.AccessKey, nil
}

func CreateS3Bucket(s3Session *s3.S3, domain string, region string) error {
	var bucketConfig *s3.CreateBucketConfiguration
	if region != "us-east-1" {
		bucketConfig = &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		}
	}

	bucket, err := s3Session.CreateBucket(&s3.CreateBucketInput{
		ACL:    aws.String(s3.BucketCannedACLPublicRead),
		Bucket: aws.String(domain),
		CreateBucketConfiguration: bucketConfig,
	})
	if err != nil && err.Error() != s3.ErrCodeBucketAlreadyExists {
		return err
	}

	_, err = s3Session.PutBucketWebsite(&s3.PutBucketWebsiteInput{
		Bucket: bucket.Location,
		WebsiteConfiguration: &s3.WebsiteConfiguration{
			IndexDocument: &s3.IndexDocument{
				Suffix: aws.String("index.html"),
			},
			ErrorDocument: &s3.ErrorDocument{
				Key: aws.String("index.html"),
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = s3Session.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: bucket.Location,
		Policy: aws.String(`{
				"Version": "2008-10-17",
				"Statement": [
					{
						"Sid": "PublicReadForGetBucketObjects",
						"Effect": "Allow",
						"Principal": {
							"AWS": "*"
						},
						"Action": "s3:GetObject",
						"Resource": "arn:aws:s3:::` + domain + `/*"
					}
				]
			}`,
		),
	})
	if err != nil {
		return err
	}

	return nil
}
